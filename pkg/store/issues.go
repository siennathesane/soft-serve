package store

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
)

// IssueStore is an interface for managing issues.
type IssueStore interface {
	// GetIssueByID returns an issue by its ID.
	GetIssueByID(ctx context.Context, h db.Handler, repoID int64, id int64) (models.Issue, error)
	// GetIssuesByRepoID returns all issues for a repository.
	GetIssuesByRepoID(ctx context.Context, h db.Handler, repoID int64) ([]models.Issue, error)
	// GetIssuesByRepoIDAndState returns all issues for a repository with a specific state.
	GetIssuesByRepoIDAndState(ctx context.Context, h db.Handler, repoID int64, state models.IssueState) ([]models.Issue, error)
	// CreateIssue creates an issue.
	CreateIssue(ctx context.Context, h db.Handler, repoID int64, authorID int64, title string, description string) (int64, error)
	// UpdateIssue updates an issue.
	UpdateIssue(ctx context.Context, h db.Handler, repoID int64, id int64, title string, description string) error
	// CloseIssue marks an issue as closed.
	CloseIssue(ctx context.Context, h db.Handler, repoID int64, id int64, closedBy int64) error
	// ReopenIssue reopens a closed issue.
	ReopenIssue(ctx context.Context, h db.Handler, repoID int64, id int64) error
	// DeleteIssue deletes an issue by its ID.
	DeleteIssue(ctx context.Context, h db.Handler, repoID int64, id int64) error
}
