package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/spf13/cobra"
)

func issueCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "issue",
		Aliases: []string{"issues"},
		Short:   "Manage issues",
	}

	cmd.AddCommand(
		issueCreateCommand(),
		issueListCommand(),
		issueShowCommand(),
		issueUpdateCommand(),
		issueCloseCommand(),
		issueReopenCommand(),
		issueAddDependencyCommand(),
		issueRemoveDependencyCommand(),
	)

	return cmd
}

func issueCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "create REPOSITORY TITLE [DESCRIPTION]",
		Short:             "Create an issue",
		Args:              cobra.RangeArgs(2, 3),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := args[0]
			title := args[1]
			description := ""
			if len(args) > 2 {
				description = args[2]
			}

			issueID, err := be.CreateIssue(ctx, repo, title, description)
			if err != nil {
				return err
			}

			cmd.Printf("Created issue #%d\n", issueID)
			return nil
		},
	}

	return cmd
}

func issueListCommand() *cobra.Command {
	var stateFilter string

	cmd := &cobra.Command{
		Use:               "list REPOSITORY",
		Short:             "List issues",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := args[0]

			var state *models.IssueState
			if stateFilter != "" {
				s := parseIssueState(stateFilter)
				if s < 0 {
					return fmt.Errorf("invalid state: %s (must be one of: open, closed)", stateFilter)
				}
				state = &s
			}

			issues, err := be.ListIssues(ctx, repo, state)
			if err != nil {
				return err
			}

			if len(issues) == 0 {
				cmd.Println("No issues found")
				return nil
			}

			for _, issue := range issues {
				cmd.Printf("#%d: %s [%s]\n",
					issue.ID,
					issue.Title,
					issue.State.String(),
				)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&stateFilter, "state", "", "Filter by state (open, closed)")

	return cmd
}

func issueShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "show REPOSITORY ISSUE_ID",
		Short:             "Show issue details",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := args[0]

			issueID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue ID: %w", err)
			}

			issue, err := be.GetIssue(ctx, repo, issueID)
			if err != nil {
				return err
			}

			cmd.Printf("Issue #%d\n", issue.ID)
			cmd.Printf("Title: %s\n", issue.Title)
			cmd.Printf("Description: %s\n", issue.Description)
			cmd.Printf("State: %s\n", issue.State.String())
			cmd.Printf("Created At: %s\n", issue.CreatedAt.Format("2006-01-02 15:04:05"))
			cmd.Printf("Updated At: %s\n", issue.UpdatedAt.Format("2006-01-02 15:04:05"))

			if issue.ClosedAt.Valid {
				cmd.Printf("Closed At: %s\n", issue.ClosedAt.Time.Format("2006-01-02 15:04:05"))
			}

			// Display dependencies
			dependencies, err := be.GetIssueDependencies(ctx, repo, issueID)
			if err == nil && len(dependencies) > 0 {
				cmd.Printf("\nDepends on:\n")
				for _, dep := range dependencies {
					cmd.Printf("  #%d - %s\n", dep.ID, dep.Title)
				}
			}

			// Display dependents
			dependents, err := be.GetIssueDependents(ctx, repo, issueID)
			if err == nil && len(dependents) > 0 {
				cmd.Printf("\nBlocked by:\n")
				for _, dep := range dependents {
					cmd.Printf("  #%d - %s\n", dep.ID, dep.Title)
				}
			}

			return nil
		},
	}

	return cmd
}

func issueUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "update REPOSITORY ISSUE_ID TITLE [DESCRIPTION]",
		Short:             "Update an issue",
		Args:              cobra.RangeArgs(3, 4),
		PersistentPreRunE: checkIfReadableAndCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := args[0]

			issueID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue ID: %w", err)
			}

			title := args[2]
			description := ""
			if len(args) > 3 {
				description = args[3]
			}

			if err := be.UpdateIssue(ctx, repo, issueID, title, description); err != nil {
				return err
			}

			cmd.Printf("Updated issue #%d\n", issueID)
			return nil
		},
	}

	return cmd
}

func issueCloseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "close REPOSITORY ISSUE_ID",
		Short:             "Close an issue",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfReadableAndCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := args[0]

			issueID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue ID: %w", err)
			}

			if err := be.CloseIssue(ctx, repo, issueID); err != nil {
				return err
			}

			cmd.Printf("Closed issue #%d\n", issueID)
			return nil
		},
	}

	return cmd
}

func issueReopenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "reopen REPOSITORY ISSUE_ID",
		Short:             "Reopen a closed issue",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfReadableAndCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := args[0]

			issueID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue ID: %w", err)
			}

			if err := be.ReopenIssue(ctx, repo, issueID); err != nil {
				return err
			}

			cmd.Printf("Reopened issue #%d\n", issueID)
			return nil
		},
	}

	return cmd
}

func issueAddDependencyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "add-dependency REPOSITORY ISSUE_ID DEPENDS_ON_ID",
		Aliases:           []string{"add-dep"},
		Short:             "Add a dependency to an issue",
		Args:              cobra.ExactArgs(3),
		PersistentPreRunE: checkIfReadableAndCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := args[0]

			issueID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue ID: %w", err)
			}

			dependsOnID, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid depends on ID: %w", err)
			}

			if err := be.AddIssueDependency(ctx, repo, issueID, dependsOnID); err != nil {
				return err
			}

			cmd.Printf("Added dependency: issue #%d now depends on issue #%d\n", issueID, dependsOnID)
			return nil
		},
	}

	return cmd
}

func issueRemoveDependencyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "remove-dependency REPOSITORY ISSUE_ID DEPENDS_ON_ID",
		Aliases:           []string{"remove-dep", "rm-dep"},
		Short:             "Remove a dependency from an issue",
		Args:              cobra.ExactArgs(3),
		PersistentPreRunE: checkIfReadableAndCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := args[0]

			issueID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue ID: %w", err)
			}

			dependsOnID, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid depends on ID: %w", err)
			}

			if err := be.RemoveIssueDependency(ctx, repo, issueID, dependsOnID); err != nil {
				return err
			}

			cmd.Printf("Removed dependency: issue #%d no longer depends on issue #%d\n", issueID, dependsOnID)
			return nil
		},
	}

	return cmd
}

// parseIssueState parses a state string into an IssueState.
func parseIssueState(s string) models.IssueState {
	switch strings.ToLower(s) {
	case "open":
		return models.IssueStateOpen
	case "closed":
		return models.IssueStateClosed
	default:
		return -1
	}
}
