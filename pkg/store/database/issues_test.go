package database_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/migrate"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store/database"
	"github.com/matryer/is"
)

// openTestDB opens a temporary SQLite database for testing.
func openTestDB(ctx context.Context, t *testing.T) (*db.DB, error) {
	dbpath := filepath.Join(t.TempDir(), "test.db")
	dbx, err := db.Open(ctx, "sqlite", dbpath)
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() {
		if err := dbx.Close(); err != nil {
			t.Error(err)
		}
	})
	return dbx, nil
}

func TestIssueStore(t *testing.T) {
	is := is.New(t)

	// Setup database
	ctx := config.WithContext(context.TODO(), config.DefaultConfig())
	dbx, err := openTestDB(ctx, t)
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
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
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
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			issueID, err = store.CreateIssue(ctx, tx, repoID, userID, "Get Test Issue", "Description")
			return err
		})
		is.NoErr(err)

		// Get issue
		var issue models.Issue
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
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
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
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
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
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
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
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
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
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
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
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
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			issueID, err = store.CreateIssue(ctx, tx, repoID, userID, "Original Title", "Original Description")
			return err
		})
		is.NoErr(err)

		// Update issue
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			return store.UpdateIssue(ctx, tx, repoID, issueID, "Updated Title", "Updated Description")
		})
		is.NoErr(err)

		// Verify update
		var issue models.Issue
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
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
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			issueID, err = store.CreateIssue(ctx, tx, repoID, userID, "To Close", "Description")
			return err
		})
		is.NoErr(err)

		// Close issue
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			return store.CloseIssue(ctx, tx, repoID, issueID, userID)
		})
		is.NoErr(err)

		// Verify state
		var issue models.Issue
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
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
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			issueID, err = store.CreateIssue(ctx, tx, repoID, userID, "To Reopen", "Description")
			if err != nil {
				return err
			}
			return store.CloseIssue(ctx, tx, repoID, issueID, userID)
		})
		is.NoErr(err)

		// Reopen issue
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			return store.ReopenIssue(ctx, tx, repoID, issueID)
		})
		is.NoErr(err)

		// Verify state
		var issue models.Issue
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
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
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			issueID, err = store.CreateIssue(ctx, tx, repoID, userID, "To Delete", "Description")
			return err
		})
		is.NoErr(err)

		// Delete issue
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			return store.DeleteIssue(ctx, tx, repoID, issueID)
		})
		is.NoErr(err)

		// Verify deletion (should return error)
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			_, err := store.GetIssueByID(ctx, tx, repoID, issueID)
			return err
		})
		is.True(err != nil) // Should error (not found)
	})

	// Test AddIssueDependency
	t.Run("AddIssueDependency", func(t *testing.T) {
		is := is.New(t)

		// Create two issues
		var issue1ID, issue2ID int64
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			issue1ID, err = store.CreateIssue(ctx, tx, repoID, userID, "Issue 1", "Description 1")
			if err != nil {
				return err
			}
			issue2ID, err = store.CreateIssue(ctx, tx, repoID, userID, "Issue 2", "Description 2")
			return err
		})
		is.NoErr(err)

		// Add dependency: issue1 depends on issue2
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			return store.AddIssueDependency(ctx, tx, repoID, issue1ID, issue2ID)
		})
		is.NoErr(err)

		// Verify dependency exists
		var hasDep bool
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			hasDep, err = store.HasIssueDependency(ctx, tx, repoID, issue1ID, issue2ID)
			return err
		})
		is.NoErr(err)
		is.True(hasDep)
	})

	// Test GetIssueDependencies
	t.Run("GetIssueDependencies", func(t *testing.T) {
		is := is.New(t)

		// Create three issues
		var mainIssueID, dep1ID, dep2ID int64
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			mainIssueID, err = store.CreateIssue(ctx, tx, repoID, userID, "Main Issue", "Main")
			if err != nil {
				return err
			}
			dep1ID, err = store.CreateIssue(ctx, tx, repoID, userID, "Dependency 1", "Dep 1")
			if err != nil {
				return err
			}
			dep2ID, err = store.CreateIssue(ctx, tx, repoID, userID, "Dependency 2", "Dep 2")
			return err
		})
		is.NoErr(err)

		// Add dependencies: mainIssue depends on dep1 and dep2
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			if err := store.AddIssueDependency(ctx, tx, repoID, mainIssueID, dep1ID); err != nil {
				return err
			}
			return store.AddIssueDependency(ctx, tx, repoID, mainIssueID, dep2ID)
		})
		is.NoErr(err)

		// Get dependencies
		var dependencies []models.Issue
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			dependencies, err = store.GetIssueDependencies(ctx, tx, repoID, mainIssueID)
			return err
		})
		is.NoErr(err)
		is.Equal(len(dependencies), 2) // Should have 2 dependencies
	})

	// Test GetIssueDependents
	t.Run("GetIssueDependents", func(t *testing.T) {
		is := is.New(t)

		// Create three issues
		var blockerID, blocked1ID, blocked2ID int64
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			blockerID, err = store.CreateIssue(ctx, tx, repoID, userID, "Blocker Issue", "Blocker")
			if err != nil {
				return err
			}
			blocked1ID, err = store.CreateIssue(ctx, tx, repoID, userID, "Blocked 1", "Blocked 1")
			if err != nil {
				return err
			}
			blocked2ID, err = store.CreateIssue(ctx, tx, repoID, userID, "Blocked 2", "Blocked 2")
			return err
		})
		is.NoErr(err)

		// Add dependencies: blocked1 and blocked2 depend on blocker
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			if err := store.AddIssueDependency(ctx, tx, repoID, blocked1ID, blockerID); err != nil {
				return err
			}
			return store.AddIssueDependency(ctx, tx, repoID, blocked2ID, blockerID)
		})
		is.NoErr(err)

		// Get dependents
		var dependents []models.Issue
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			dependents, err = store.GetIssueDependents(ctx, tx, repoID, blockerID)
			return err
		})
		is.NoErr(err)
		is.Equal(len(dependents), 2) // Should have 2 dependents
	})

	// Test RemoveIssueDependency
	t.Run("RemoveIssueDependency", func(t *testing.T) {
		is := is.New(t)

		// Create two issues
		var issue1ID, issue2ID int64
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			issue1ID, err = store.CreateIssue(ctx, tx, repoID, userID, "Issue A", "Description A")
			if err != nil {
				return err
			}
			issue2ID, err = store.CreateIssue(ctx, tx, repoID, userID, "Issue B", "Description B")
			return err
		})
		is.NoErr(err)

		// Add dependency
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			return store.AddIssueDependency(ctx, tx, repoID, issue1ID, issue2ID)
		})
		is.NoErr(err)

		// Remove dependency
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			return store.RemoveIssueDependency(ctx, tx, repoID, issue1ID, issue2ID)
		})
		is.NoErr(err)

		// Verify dependency removed
		var hasDep bool
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			hasDep, err = store.HasIssueDependency(ctx, tx, repoID, issue1ID, issue2ID)
			return err
		})
		is.NoErr(err)
		is.True(!hasDep) // Should not have dependency
	})

	// Test HasIssueDependency
	t.Run("HasIssueDependency", func(t *testing.T) {
		is := is.New(t)

		// Create two issues
		var issue1ID, issue2ID int64
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			issue1ID, err = store.CreateIssue(ctx, tx, repoID, userID, "Issue X", "Description X")
			if err != nil {
				return err
			}
			issue2ID, err = store.CreateIssue(ctx, tx, repoID, userID, "Issue Y", "Description Y")
			return err
		})
		is.NoErr(err)

		// Check dependency doesn't exist initially
		var hasDep bool
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			hasDep, err = store.HasIssueDependency(ctx, tx, repoID, issue1ID, issue2ID)
			return err
		})
		is.NoErr(err)
		is.True(!hasDep) // Should not exist

		// Add dependency
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			return store.AddIssueDependency(ctx, tx, repoID, issue1ID, issue2ID)
		})
		is.NoErr(err)

		// Check dependency exists now
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			hasDep, err = store.HasIssueDependency(ctx, tx, repoID, issue1ID, issue2ID)
			return err
		})
		is.NoErr(err)
		is.True(hasDep) // Should exist now
	})
}
