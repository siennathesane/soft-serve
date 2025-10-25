package store

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
)

// MergeRequestStore is an interface for managing merge requests.
type MergeRequestStore interface {
	// GetMergeRequestByID returns a merge request by its ID.
	GetMergeRequestByID(ctx context.Context, h db.Handler, repoID int64, id int64) (models.MergeRequest, error)
	// GetMergeRequestsByRepoID returns all merge requests for a repository.
	GetMergeRequestsByRepoID(ctx context.Context, h db.Handler, repoID int64) ([]models.MergeRequest, error)
	// GetMergeRequestsByRepoIDAndState returns all merge requests for a repository with a specific state.
	GetMergeRequestsByRepoIDAndState(ctx context.Context, h db.Handler, repoID int64, state models.MergeRequestState) ([]models.MergeRequest, error)
	// CreateMergeRequest creates a merge request.
	CreateMergeRequest(ctx context.Context, h db.Handler, repoID int64, authorID int64, title string, description string, sourceBranch string, targetBranch string) (int64, error)
	// UpdateMergeRequest updates a merge request.
	UpdateMergeRequest(ctx context.Context, h db.Handler, repoID int64, id int64, title string, description string) error
	// MergeMergeRequest marks a merge request as merged.
	MergeMergeRequest(ctx context.Context, h db.Handler, repoID int64, id int64, mergedBy int64) error
	// CloseMergeRequest marks a merge request as closed.
	CloseMergeRequest(ctx context.Context, h db.Handler, repoID int64, id int64, closedBy int64) error
	// ReopenMergeRequest reopens a closed merge request.
	ReopenMergeRequest(ctx context.Context, h db.Handler, repoID int64, id int64) error
	// DeleteMergeRequest deletes a merge request by its ID.
	DeleteMergeRequest(ctx context.Context, h db.Handler, repoID int64, id int64) error
}
