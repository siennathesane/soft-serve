package database

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

type issueStore struct{}

var _ store.IssueStore = (*issueStore)(nil)

// GetIssueByID implements store.IssueStore.
func (*issueStore) GetIssueByID(ctx context.Context, h db.Handler, repoID int64, id int64) (models.Issue, error) {
	var issue models.Issue
	query := h.Rebind(`
		SELECT * FROM issues
		WHERE repo_id = ? AND id = ?
	`)
	err := h.GetContext(ctx, &issue, query, repoID, id)
	return issue, err
}

// GetIssuesByRepoID implements store.IssueStore.
func (*issueStore) GetIssuesByRepoID(ctx context.Context, h db.Handler, repoID int64) ([]models.Issue, error) {
	var issues []models.Issue
	query := h.Rebind(`
		SELECT * FROM issues
		WHERE repo_id = ?
		ORDER BY created_at DESC
	`)
	err := h.SelectContext(ctx, &issues, query, repoID)
	return issues, err
}

// GetIssuesByRepoIDAndState implements store.IssueStore.
func (*issueStore) GetIssuesByRepoIDAndState(ctx context.Context, h db.Handler, repoID int64, state models.IssueState) ([]models.Issue, error) {
	var issues []models.Issue
	query := h.Rebind(`
		SELECT * FROM issues
		WHERE repo_id = ? AND state = ?
		ORDER BY created_at DESC
	`)
	err := h.SelectContext(ctx, &issues, query, repoID, state)
	return issues, err
}

// CreateIssue implements store.IssueStore.
func (*issueStore) CreateIssue(ctx context.Context, h db.Handler, repoID int64, authorID int64, title string, description string) (int64, error) {
	query := h.Rebind(`
		INSERT INTO issues (repo_id, author_id, title, description, state, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`)
	res, err := h.ExecContext(ctx, query, repoID, authorID, title, description, models.IssueStateOpen)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateIssue implements store.IssueStore.
func (*issueStore) UpdateIssue(ctx context.Context, h db.Handler, repoID int64, id int64, title string, description string) error {
	query := h.Rebind(`
		UPDATE issues
		SET title = ?, description = ?, updated_at = CURRENT_TIMESTAMP
		WHERE repo_id = ? AND id = ?
	`)
	_, err := h.ExecContext(ctx, query, title, description, repoID, id)
	return err
}

// CloseIssue implements store.IssueStore.
func (*issueStore) CloseIssue(ctx context.Context, h db.Handler, repoID int64, id int64, closedBy int64) error {
	query := h.Rebind(`
		UPDATE issues
		SET state = ?, closed_by = ?, closed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE repo_id = ? AND id = ? AND state = ?
	`)
	_, err := h.ExecContext(ctx, query, models.IssueStateClosed, closedBy, repoID, id, models.IssueStateOpen)
	return err
}

// ReopenIssue implements store.IssueStore.
func (*issueStore) ReopenIssue(ctx context.Context, h db.Handler, repoID int64, id int64) error {
	query := h.Rebind(`
		UPDATE issues
		SET state = ?, closed_by = NULL, closed_at = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE repo_id = ? AND id = ? AND state = ?
	`)
	_, err := h.ExecContext(ctx, query, models.IssueStateOpen, repoID, id, models.IssueStateClosed)
	return err
}

// DeleteIssue implements store.IssueStore.
func (*issueStore) DeleteIssue(ctx context.Context, h db.Handler, repoID int64, id int64) error {
	query := h.Rebind(`
		DELETE FROM issues
		WHERE repo_id = ? AND id = ?
	`)
	_, err := h.ExecContext(ctx, query, repoID, id)
	return err
}
