package database_test

import (
	"context"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/internal/test"
	"github.com/charmbracelet/soft-serve/pkg/db/migrate"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store/database"
	"github.com/matryer/is"
)

func TestIssueStore(t *testing.T) {
	is := is.New(t)

	// Setup database
	ctx := config.WithContext(context.TODO(), config.DefaultConfig())
	dbx, err := test.OpenSqlite(ctx, t)
	is.NoErr(err)

	// Run migrations
	is.NoErr(migrate.Migrate(ctx, dbx))

	// Create store
	store := database.New(ctx, dbx)

	// Create test data: user and repo
	var userID, repoID int64
	err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
		// Create user
		result, err := tx.ExecContext(ctx, "INSERT INTO users (username, admin, created_at, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)", "testuser", false)
		if err != nil {
			return err
		}
		userID, err = result.LastInsertId()
		if err != nil {
			return err
		}

		// Create repo
		result, err = tx.ExecContext(ctx, "INSERT INTO repos (name, project_name, description, private, mirror, hidden, user_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
			"testrepo", "Test Repo", "Test Description", false, false, false, userID)
		if err != nil {
			return err
		}
		repoID, err = result.LastInsertId()
		return err
	})
	is.NoErr(err)

	// Test CreateIssue
	t.Run("CreateIssue", func(t *testing.T) {
		is := is.New(t)

		var issueID int64
		err := dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			var err error
			issueID, err = store.CreateIssue(ctx, tx, repoID, userID, "Test Issue", "Test Description")
			return err
		})
		is.NoErr(err)
		is.True(issueID > 0) // Issue ID should be positive
	})

	// Test GetIssueByID
	t.Run("GetIssueByID", func(t *testing.T) {
		is := is.New(t)

		// Create issue first
		var issueID int64
		err := dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			var err error
			issueID, err = store.CreateIssue(ctx, tx, repoID, userID, "Get Test Issue", "Description")
			return err
		})
		is.NoErr(err)

		// Get issue
		var issue models.Issue
		err = dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			var err error
			issue, err = store.GetIssueByID(ctx, tx, repoID, issueID)
			return err
		})
		is.NoErr(err)
		is.Equal(issue.ID, issueID)
		is.Equal(issue.Title, "Get Test Issue")
		is.Equal(issue.State, models.IssueStateOpen)
	})

	// Test GetIssuesByRepoID
	t.Run("GetIssuesByRepoID", func(t *testing.T) {
		is := is.New(t)

		// Create multiple issues
		err := dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			_, err := store.CreateIssue(ctx, tx, repoID, userID, "Issue 1", "Desc 1")
			if err != nil {
				return err
			}
			_, err = store.CreateIssue(ctx, tx, repoID, userID, "Issue 2", "Desc 2")
			return err
		})
		is.NoErr(err)

		// Get all issues
		var issues []models.Issue
		err = dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			var err error
			issues, err = store.GetIssuesByRepoID(ctx, tx, repoID)
			return err
		})
		is.NoErr(err)
		is.True(len(issues) >= 2) // At least 2 issues
	})

	// Test GetIssuesByRepoIDAndState
	t.Run("GetIssuesByRepoIDAndState", func(t *testing.T) {
		is := is.New(t)

		// Create and close one issue
		var issueID int64
		err := dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			var err error
			issueID, err = store.CreateIssue(ctx, tx, repoID, userID, "Closed Issue", "Description")
			if err != nil {
				return err
			}
			return store.CloseIssue(ctx, tx, repoID, issueID, userID)
		})
		is.NoErr(err)

		// Get only open issues
		var openIssues []models.Issue
		err = dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			var err error
			openIssues, err = store.GetIssuesByRepoIDAndState(ctx, tx, repoID, models.IssueStateOpen)
			return err
		})
		is.NoErr(err)

		// Verify none are closed
		for _, issue := range openIssues {
			is.Equal(issue.State, models.IssueStateOpen)
		}

		// Get only closed issues
		var closedIssues []models.Issue
		err = dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			var err error
			closedIssues, err = store.GetIssuesByRepoIDAndState(ctx, tx, repoID, models.IssueStateClosed)
			return err
		})
		is.NoErr(err)
		is.True(len(closedIssues) >= 1) // At least one closed issue
	})

	// Test UpdateIssue
	t.Run("UpdateIssue", func(t *testing.T) {
		is := is.New(t)

		// Create issue
		var issueID int64
		err := dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			var err error
			issueID, err = store.CreateIssue(ctx, tx, repoID, userID, "Original Title", "Original Description")
			return err
		})
		is.NoErr(err)

		// Update issue
		err = dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			return store.UpdateIssue(ctx, tx, repoID, issueID, "Updated Title", "Updated Description")
		})
		is.NoErr(err)

		// Verify update
		var issue models.Issue
		err = dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			var err error
			issue, err = store.GetIssueByID(ctx, tx, repoID, issueID)
			return err
		})
		is.NoErr(err)
		is.Equal(issue.Title, "Updated Title")
		is.Equal(issue.Description, "Updated Description")
	})

	// Test CloseIssue
	t.Run("CloseIssue", func(t *testing.T) {
		is := is.New(t)

		// Create issue
		var issueID int64
		err := dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			var err error
			issueID, err = store.CreateIssue(ctx, tx, repoID, userID, "To Close", "Description")
			return err
		})
		is.NoErr(err)

		// Close issue
		err = dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			return store.CloseIssue(ctx, tx, repoID, issueID, userID)
		})
		is.NoErr(err)

		// Verify state
		var issue models.Issue
		err = dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			var err error
			issue, err = store.GetIssueByID(ctx, tx, repoID, issueID)
			return err
		})
		is.NoErr(err)
		is.Equal(issue.State, models.IssueStateClosed)
		is.True(issue.ClosedBy.Valid)
		is.Equal(issue.ClosedBy.Int64, userID)
		is.True(issue.ClosedAt.Valid)
	})

	// Test ReopenIssue
	t.Run("ReopenIssue", func(t *testing.T) {
		is := is.New(t)

		// Create and close issue
		var issueID int64
		err := dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			var err error
			issueID, err = store.CreateIssue(ctx, tx, repoID, userID, "To Reopen", "Description")
			if err != nil {
				return err
			}
			return store.CloseIssue(ctx, tx, repoID, issueID, userID)
		})
		is.NoErr(err)

		// Reopen issue
		err = dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			return store.ReopenIssue(ctx, tx, repoID, issueID)
		})
		is.NoErr(err)

		// Verify state
		var issue models.Issue
		err = dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			var err error
			issue, err = store.GetIssueByID(ctx, tx, repoID, issueID)
			return err
		})
		is.NoErr(err)
		is.Equal(issue.State, models.IssueStateOpen)
		is.True(!issue.ClosedBy.Valid)  // Should be NULL
		is.True(!issue.ClosedAt.Valid)  // Should be NULL
	})

	// Test DeleteIssue
	t.Run("DeleteIssue", func(t *testing.T) {
		is := is.New(t)

		// Create issue
		var issueID int64
		err := dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			var err error
			issueID, err = store.CreateIssue(ctx, tx, repoID, userID, "To Delete", "Description")
			return err
		})
		is.NoErr(err)

		// Delete issue
		err = dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			return store.DeleteIssue(ctx, tx, repoID, issueID)
		})
		is.NoErr(err)

		// Verify deletion (should return error)
		err = dbx.TransactionContext(ctx, func(tx *database.Tx) error {
			_, err := store.GetIssueByID(ctx, tx, repoID, issueID)
			return err
		})
		is.True(err != nil) // Should error (not found)
	})
}
