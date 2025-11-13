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
