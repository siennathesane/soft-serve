package database_test

import (
	"context"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/migrate"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store/database"
	"github.com/matryer/is"
)

func TestMergeRequestStore(t *testing.T) {
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

	// Test CreateMergeRequest
	t.Run("CreateMergeRequest", func(t *testing.T) {
		is := is.New(t)

		var mrID int64
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			mrID, err = store.CreateMergeRequest(ctx, tx, repoID, userID, "Test MR", "Test Description", "feature", "main")
			return err
		})
		is.NoErr(err)
		is.True(mrID > 0) // MR ID should be positive
	})

	// Test GetMergeRequestByID
	t.Run("GetMergeRequestByID", func(t *testing.T) {
		is := is.New(t)

		// Create MR first
		var mrID int64
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			mrID, err = store.CreateMergeRequest(ctx, tx, repoID, userID, "Get Test MR", "Description", "feature", "main")
			return err
		})
		is.NoErr(err)

		// Get MR
		var mr models.MergeRequest
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			mr, err = store.GetMergeRequestByID(ctx, tx, repoID, mrID)
			return err
		})
		is.NoErr(err)
		is.Equal(mr.ID, mrID)
		is.Equal(mr.Title, "Get Test MR")
		is.Equal(mr.SourceBranch, "feature")
		is.Equal(mr.TargetBranch, "main")
		is.Equal(mr.State, models.MergeRequestStateOpen)
	})

	// Test GetMergeRequestsByRepoID
	t.Run("GetMergeRequestsByRepoID", func(t *testing.T) {
		is := is.New(t)

		// Create multiple MRs
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			_, err := store.CreateMergeRequest(ctx, tx, repoID, userID, "MR 1", "Desc 1", "f1", "main")
			if err != nil {
				return err
			}
			_, err = store.CreateMergeRequest(ctx, tx, repoID, userID, "MR 2", "Desc 2", "f2", "main")
			return err
		})
		is.NoErr(err)

		// Get all MRs
		var mrs []models.MergeRequest
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			mrs, err = store.GetMergeRequestsByRepoID(ctx, tx, repoID)
			return err
		})
		is.NoErr(err)
		is.True(len(mrs) >= 2) // At least 2 MRs
	})

	// Test GetMergeRequestsByRepoIDAndState
	t.Run("GetMergeRequestsByRepoIDAndState", func(t *testing.T) {
		is := is.New(t)

		// Create and close one MR
		var mrID int64
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			mrID, err = store.CreateMergeRequest(ctx, tx, repoID, userID, "Closed MR", "Description", "feature", "main")
			if err != nil {
				return err
			}
			return store.CloseMergeRequest(ctx, tx, repoID, mrID, userID)
		})
		is.NoErr(err)

		// Get only open MRs
		var openMRs []models.MergeRequest
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			openMRs, err = store.GetMergeRequestsByRepoIDAndState(ctx, tx, repoID, models.MergeRequestStateOpen)
			return err
		})
		is.NoErr(err)

		// Verify none are closed
		for _, mr := range openMRs {
			is.Equal(mr.State, models.MergeRequestStateOpen)
		}

		// Get only closed MRs
		var closedMRs []models.MergeRequest
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			closedMRs, err = store.GetMergeRequestsByRepoIDAndState(ctx, tx, repoID, models.MergeRequestStateClosed)
			return err
		})
		is.NoErr(err)
		is.True(len(closedMRs) >= 1) // At least one closed MR
	})

	// Test UpdateMergeRequest
	t.Run("UpdateMergeRequest", func(t *testing.T) {
		is := is.New(t)

		// Create MR
		var mrID int64
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			mrID, err = store.CreateMergeRequest(ctx, tx, repoID, userID, "Original Title", "Original Description", "feature", "main")
			return err
		})
		is.NoErr(err)

		// Update MR
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			return store.UpdateMergeRequest(ctx, tx, repoID, mrID, "Updated Title", "Updated Description")
		})
		is.NoErr(err)

		// Verify update
		var mr models.MergeRequest
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			mr, err = store.GetMergeRequestByID(ctx, tx, repoID, mrID)
			return err
		})
		is.NoErr(err)
		is.Equal(mr.Title, "Updated Title")
		is.Equal(mr.Description, "Updated Description")
	})

	// Test MergeMergeRequest
	t.Run("MergeMergeRequest", func(t *testing.T) {
		is := is.New(t)

		// Create MR
		var mrID int64
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			mrID, err = store.CreateMergeRequest(ctx, tx, repoID, userID, "To Merge", "Description", "feature", "main")
			return err
		})
		is.NoErr(err)

		// Merge MR
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			return store.MergeMergeRequest(ctx, tx, repoID, mrID, userID)
		})
		is.NoErr(err)

		// Verify state
		var mr models.MergeRequest
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			mr, err = store.GetMergeRequestByID(ctx, tx, repoID, mrID)
			return err
		})
		is.NoErr(err)
		is.Equal(mr.State, models.MergeRequestStateMerged)
		is.True(mr.MergedBy.Valid)
		is.Equal(mr.MergedBy.Int64, userID)
		is.True(mr.MergedAt.Valid)
	})

	// Test CloseMergeRequest
	t.Run("CloseMergeRequest", func(t *testing.T) {
		is := is.New(t)

		// Create MR
		var mrID int64
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			mrID, err = store.CreateMergeRequest(ctx, tx, repoID, userID, "To Close", "Description", "feature", "main")
			return err
		})
		is.NoErr(err)

		// Close MR
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			return store.CloseMergeRequest(ctx, tx, repoID, mrID, userID)
		})
		is.NoErr(err)

		// Verify state
		var mr models.MergeRequest
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			mr, err = store.GetMergeRequestByID(ctx, tx, repoID, mrID)
			return err
		})
		is.NoErr(err)
		is.Equal(mr.State, models.MergeRequestStateClosed)
		is.True(mr.ClosedBy.Valid)
		is.Equal(mr.ClosedBy.Int64, userID)
		is.True(mr.ClosedAt.Valid)
	})

	// Test ReopenMergeRequest
	t.Run("ReopenMergeRequest", func(t *testing.T) {
		is := is.New(t)

		// Create and close MR
		var mrID int64
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			mrID, err = store.CreateMergeRequest(ctx, tx, repoID, userID, "To Reopen", "Description", "feature", "main")
			if err != nil {
				return err
			}
			return store.CloseMergeRequest(ctx, tx, repoID, mrID, userID)
		})
		is.NoErr(err)

		// Reopen MR
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			return store.ReopenMergeRequest(ctx, tx, repoID, mrID)
		})
		is.NoErr(err)

		// Verify state
		var mr models.MergeRequest
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			mr, err = store.GetMergeRequestByID(ctx, tx, repoID, mrID)
			return err
		})
		is.NoErr(err)
		is.Equal(mr.State, models.MergeRequestStateOpen)
		is.True(!mr.ClosedBy.Valid)  // Should be NULL
		is.True(!mr.ClosedAt.Valid)  // Should be NULL
	})

	// Test DeleteMergeRequest
	t.Run("DeleteMergeRequest", func(t *testing.T) {
		is := is.New(t)

		// Create MR
		var mrID int64
		err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			var err error
			mrID, err = store.CreateMergeRequest(ctx, tx, repoID, userID, "To Delete", "Description", "feature", "main")
			return err
		})
		is.NoErr(err)

		// Delete MR
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			return store.DeleteMergeRequest(ctx, tx, repoID, mrID)
		})
		is.NoErr(err)

		// Verify deletion (should return error)
		err = dbx.TransactionContext(ctx, func(tx *db.Tx) error {
			_, err := store.GetMergeRequestByID(ctx, tx, repoID, mrID)
			return err
		})
		is.True(err != nil) // Should error (not found)
	})
}
