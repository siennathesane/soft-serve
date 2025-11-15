package database

import (
	"context"
	"database/sql"

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

// AddIssueDependency implements store.IssueStore.
func (*issueStore) AddIssueDependency(ctx context.Context, h db.Handler, repoID int64, issueID int64, dependsOnID int64) error {
	// Verify both issues exist and belong to the same repository
	query := h.Rebind(`
		SELECT COUNT(*) FROM issues
		WHERE repo_id = ? AND (id = ? OR id = ?)
	`)
	var count int
	if err := h.GetContext(ctx, &count, query, repoID, issueID, dependsOnID); err != nil {
		return err
	}
	if count != 2 {
		return sql.ErrNoRows
	}

	// Insert the dependency
	query = h.Rebind(`
		INSERT INTO issue_dependencies (issue_id, depends_on_id)
		VALUES (?, ?)
	`)
	_, err := h.ExecContext(ctx, query, issueID, dependsOnID)
	return err
}

// RemoveIssueDependency implements store.IssueStore.
func (*issueStore) RemoveIssueDependency(ctx context.Context, h db.Handler, repoID int64, issueID int64, dependsOnID int64) error {
	// Verify the issue belongs to the repository
	query := h.Rebind(`
		SELECT COUNT(*) FROM issues
		WHERE repo_id = ? AND id = ?
	`)
	var count int
	if err := h.GetContext(ctx, &count, query, repoID, issueID); err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}

	// Delete the dependency
	query = h.Rebind(`
		DELETE FROM issue_dependencies
		WHERE issue_id = ? AND depends_on_id = ?
	`)
	_, err := h.ExecContext(ctx, query, issueID, dependsOnID)
	return err
}

// GetIssueDependencies implements store.IssueStore.
func (*issueStore) GetIssueDependencies(ctx context.Context, h db.Handler, repoID int64, issueID int64) ([]models.Issue, error) {
	var issues []models.Issue
	query := h.Rebind(`
		SELECT i.* FROM issues i
		INNER JOIN issue_dependencies d ON i.id = d.depends_on_id
		WHERE d.issue_id = ? AND i.repo_id = ?
		ORDER BY i.created_at DESC
	`)
	err := h.SelectContext(ctx, &issues, query, issueID, repoID)
	return issues, err
}

// GetIssueDependents implements store.IssueStore.
func (*issueStore) GetIssueDependents(ctx context.Context, h db.Handler, repoID int64, issueID int64) ([]models.Issue, error) {
	var issues []models.Issue
	query := h.Rebind(`
		SELECT i.* FROM issues i
		INNER JOIN issue_dependencies d ON i.id = d.issue_id
		WHERE d.depends_on_id = ? AND i.repo_id = ?
		ORDER BY i.created_at DESC
	`)
	err := h.SelectContext(ctx, &issues, query, issueID, repoID)
	return issues, err
}

// HasIssueDependency implements store.IssueStore.
func (*issueStore) HasIssueDependency(ctx context.Context, h db.Handler, repoID int64, issueID int64, dependsOnID int64) (bool, error) {
	// Verify the issue belongs to the repository
	query := h.Rebind(`
		SELECT COUNT(*) FROM issues
		WHERE repo_id = ? AND id = ?
	`)
	var count int
	if err := h.GetContext(ctx, &count, query, repoID, issueID); err != nil {
		return false, err
	}
	if count == 0 {
		return false, sql.ErrNoRows
	}

	// Check if the dependency exists
	query = h.Rebind(`
		SELECT COUNT(*) FROM issue_dependencies
		WHERE issue_id = ? AND depends_on_id = ?
	`)
	if err := h.GetContext(ctx, &count, query, issueID, dependsOnID); err != nil {
		return false, err
	}
	return count > 0, nil
}
