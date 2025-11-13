package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

type mergeRequestStore struct{}

var _ store.MergeRequestStore = (*mergeRequestStore)(nil)

// GetMergeRequestByID implements store.MergeRequestStore.
func (*mergeRequestStore) GetMergeRequestByID(ctx context.Context, h db.Handler, repoID int64, id int64) (models.MergeRequest, error) {
	var mr models.MergeRequest
	query := h.Rebind(`
		SELECT * FROM merge_requests
		WHERE repo_id = ? AND id = ?
	`)
	err := h.GetContext(ctx, &mr, query, repoID, id)
	return mr, err
}

// GetMergeRequestsByRepoID implements store.MergeRequestStore.
func (*mergeRequestStore) GetMergeRequestsByRepoID(ctx context.Context, h db.Handler, repoID int64) ([]models.MergeRequest, error) {
	var mrs []models.MergeRequest
	query := h.Rebind(`
		SELECT * FROM merge_requests
		WHERE repo_id = ?
		ORDER BY created_at DESC
	`)
	err := h.SelectContext(ctx, &mrs, query, repoID)
	return mrs, err
}

// GetMergeRequestsByRepoIDAndState implements store.MergeRequestStore.
func (*mergeRequestStore) GetMergeRequestsByRepoIDAndState(ctx context.Context, h db.Handler, repoID int64, state models.MergeRequestState) ([]models.MergeRequest, error) {
	var mrs []models.MergeRequest
	query := h.Rebind(`
		SELECT * FROM merge_requests
		WHERE repo_id = ? AND state = ?
		ORDER BY created_at DESC
	`)
	err := h.SelectContext(ctx, &mrs, query, repoID, state)
	return mrs, err
}

// CreateMergeRequest implements store.MergeRequestStore.
func (*mergeRequestStore) CreateMergeRequest(ctx context.Context, h db.Handler, repoID int64, authorID int64, title string, description string, sourceBranch string, targetBranch string) (int64, error) {
	query := h.Rebind(`
		INSERT INTO merge_requests (repo_id, author_id, title, description, source_branch, target_branch, state, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`)
	res, err := h.ExecContext(ctx, query, repoID, authorID, title, description, sourceBranch, targetBranch, models.MergeRequestStateOpen)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateMergeRequest implements store.MergeRequestStore.
func (*mergeRequestStore) UpdateMergeRequest(ctx context.Context, h db.Handler, repoID int64, id int64, title string, description string) error {
	query := h.Rebind(`
		UPDATE merge_requests
		SET title = ?, description = ?, updated_at = CURRENT_TIMESTAMP
		WHERE repo_id = ? AND id = ?
	`)
	_, err := h.ExecContext(ctx, query, title, description, repoID, id)
	return err
}

// MergeMergeRequest implements store.MergeRequestStore.
func (*mergeRequestStore) MergeMergeRequest(ctx context.Context, h db.Handler, repoID int64, id int64, mergedBy int64) error {
	query := h.Rebind(`
		UPDATE merge_requests
		SET state = ?, merged_by = ?, merged_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE repo_id = ? AND id = ? AND state = ?
	`)
	_, err := h.ExecContext(ctx, query, models.MergeRequestStateMerged, mergedBy, repoID, id, models.MergeRequestStateOpen)
	return err
}

// CloseMergeRequest implements store.MergeRequestStore.
func (*mergeRequestStore) CloseMergeRequest(ctx context.Context, h db.Handler, repoID int64, id int64, closedBy int64) error {
	query := h.Rebind(`
		UPDATE merge_requests
		SET state = ?, closed_by = ?, closed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE repo_id = ? AND id = ? AND state = ?
	`)
	_, err := h.ExecContext(ctx, query, models.MergeRequestStateClosed, closedBy, repoID, id, models.MergeRequestStateOpen)
	return err
}

// ReopenMergeRequest implements store.MergeRequestStore.
func (*mergeRequestStore) ReopenMergeRequest(ctx context.Context, h db.Handler, repoID int64, id int64) error {
	query := h.Rebind(`
		UPDATE merge_requests
		SET state = ?, closed_by = NULL, closed_at = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE repo_id = ? AND id = ? AND state = ?
	`)
	_, err := h.ExecContext(ctx, query, models.MergeRequestStateOpen, repoID, id, models.MergeRequestStateClosed)
	return err
}

// DeleteMergeRequest implements store.MergeRequestStore.
func (*mergeRequestStore) DeleteMergeRequest(ctx context.Context, h db.Handler, repoID int64, id int64) error {
	query := h.Rebind(`
		DELETE FROM merge_requests
		WHERE repo_id = ? AND id = ?
	`)
	_, err := h.ExecContext(ctx, query, repoID, id)
	return err
}
