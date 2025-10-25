package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/spf13/cobra"
)

func mergeRequestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "merge-request",
		Aliases: []string{"mr", "mrs", "merge-requests"},
		Short:   "Manage merge requests",
	}

	cmd.AddCommand(
		mergeRequestCreateCommand(),
		mergeRequestListCommand(),
		mergeRequestShowCommand(),
		mergeRequestMergeCommand(),
		mergeRequestCloseCommand(),
		mergeRequestReopenCommand(),
	)

	return cmd
}

func mergeRequestCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "create REPOSITORY SOURCE_BRANCH TARGET_BRANCH TITLE [DESCRIPTION]",
		Short:             "Create a merge request",
		Args:              cobra.RangeArgs(4, 5),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := args[0]
			sourceBranch := args[1]
			targetBranch := args[2]
			title := args[3]
			description := ""
			if len(args) > 4 {
				description = args[4]
			}

			mrID, err := be.CreateMergeRequest(ctx, repo, title, description, sourceBranch, targetBranch)
			if err != nil {
				return err
			}

			cmd.Printf("Created merge request #%d\n", mrID)
			return nil
		},
	}

	return cmd
}

func mergeRequestListCommand() *cobra.Command {
	var stateFilter string

	cmd := &cobra.Command{
		Use:               "list REPOSITORY",
		Short:             "List merge requests",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := args[0]

			var state *models.MergeRequestState
			if stateFilter != "" {
				s := parseState(stateFilter)
				if s < 0 {
					return fmt.Errorf("invalid state: %s (must be one of: open, merged, closed)", stateFilter)
				}
				state = &s
			}

			mrs, err := be.ListMergeRequests(ctx, repo, state)
			if err != nil {
				return err
			}

			if len(mrs) == 0 {
				cmd.Println("No merge requests found")
				return nil
			}

			for _, mr := range mrs {
				cmd.Printf("#%d: %s (%s -> %s) [%s]\n",
					mr.ID,
					mr.Title,
					mr.SourceBranch,
					mr.TargetBranch,
					mr.State.String(),
				)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&stateFilter, "state", "", "Filter by state (open, merged, closed)")

	return cmd
}

func mergeRequestShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "show REPOSITORY MR_ID",
		Short:             "Show merge request details",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := args[0]

			mrID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid merge request ID: %w", err)
			}

			mr, err := be.GetMergeRequest(ctx, repo, mrID)
			if err != nil {
				return err
			}

			cmd.Printf("Merge Request #%d\n", mr.ID)
			cmd.Printf("Title: %s\n", mr.Title)
			cmd.Printf("Description: %s\n", mr.Description)
			cmd.Printf("Source Branch: %s\n", mr.SourceBranch)
			cmd.Printf("Target Branch: %s\n", mr.TargetBranch)
			cmd.Printf("State: %s\n", mr.State.String())
			cmd.Printf("Created At: %s\n", mr.CreatedAt.Format("2006-01-02 15:04:05"))
			cmd.Printf("Updated At: %s\n", mr.UpdatedAt.Format("2006-01-02 15:04:05"))

			if mr.MergedAt.Valid {
				cmd.Printf("Merged At: %s\n", mr.MergedAt.Time.Format("2006-01-02 15:04:05"))
			}
			if mr.ClosedAt.Valid {
				cmd.Printf("Closed At: %s\n", mr.ClosedAt.Time.Format("2006-01-02 15:04:05"))
			}

			return nil
		},
	}

	return cmd
}

func mergeRequestMergeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "merge REPOSITORY MR_ID",
		Short:             "Merge a merge request",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfReadableAndCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := args[0]

			mrID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid merge request ID: %w", err)
			}

			if err := be.MergeMergeRequest(ctx, repo, mrID); err != nil {
				return err
			}

			cmd.Printf("Merged merge request #%d\n", mrID)
			return nil
		},
	}

	return cmd
}

func mergeRequestCloseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "close REPOSITORY MR_ID",
		Short:             "Close a merge request",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfReadableAndCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := args[0]

			mrID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid merge request ID: %w", err)
			}

			if err := be.CloseMergeRequest(ctx, repo, mrID); err != nil {
				return err
			}

			cmd.Printf("Closed merge request #%d\n", mrID)
			return nil
		},
	}

	return cmd
}

func mergeRequestReopenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "reopen REPOSITORY MR_ID",
		Short:             "Reopen a closed merge request",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfReadableAndCollab,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo := args[0]

			mrID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid merge request ID: %w", err)
			}

			if err := be.ReopenMergeRequest(ctx, repo, mrID); err != nil {
				return err
			}

			cmd.Printf("Reopened merge request #%d\n", mrID)
			return nil
		},
	}

	return cmd
}

// parseState parses a state string into a MergeRequestState.
func parseState(s string) models.MergeRequestState {
	switch strings.ToLower(s) {
	case "open":
		return models.MergeRequestStateOpen
	case "merged":
		return models.MergeRequestStateMerged
	case "closed":
		return models.MergeRequestStateClosed
	default:
		return -1
	}
}
