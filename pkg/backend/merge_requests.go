package backend

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/utils"
)

// CreateMergeRequest creates a new merge request for a repository.
func (d *Backend) CreateMergeRequest(ctx context.Context, repoName string, title string, description string, sourceBranch string, targetBranch string) (int64, error) {
	repoName = utils.SanitizeRepo(repoName)

	// Get repository
	r, err := d.Repository(ctx, repoName)
	if err != nil {
		return 0, err
	}

	// Get current user
	user := proto.UserFromContext(ctx)
	if user == nil {
		return 0, proto.ErrUserNotFound
	}

	// Validate that branches exist
	gr, err := r.Open()
	if err != nil {
		return 0, fmt.Errorf("failed to open repository: %w", err)
	}

	// Check if source branch exists
	if _, err := gr.ShowRefVerify(fmt.Sprintf("refs/heads/%s", sourceBranch)); err != nil {
		return 0, fmt.Errorf("source branch %q does not exist", sourceBranch)
	}

	// Check if target branch exists
	if _, err := gr.ShowRefVerify(fmt.Sprintf("refs/heads/%s", targetBranch)); err != nil {
		return 0, fmt.Errorf("target branch %q does not exist", targetBranch)
	}

	// Create merge request in database
	var mrID int64
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		mrID, err = d.store.CreateMergeRequest(ctx, tx, r.ID(), user.ID(), title, description, sourceBranch, targetBranch)
		return err
	}); err != nil {
		return 0, db.WrapError(err)
	}

	return mrID, nil
}

// GetMergeRequest returns a merge request by its ID.
func (d *Backend) GetMergeRequest(ctx context.Context, repoName string, mrID int64) (models.MergeRequest, error) {
	repoName = utils.SanitizeRepo(repoName)

	r, err := d.Repository(ctx, repoName)
	if err != nil {
		return models.MergeRequest{}, err
	}

	var mr models.MergeRequest
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		mr, err = d.store.GetMergeRequestByID(ctx, tx, r.ID(), mrID)
		return err
	}); err != nil {
		return models.MergeRequest{}, db.WrapError(err)
	}

	return mr, nil
}

// ListMergeRequests returns all merge requests for a repository.
func (d *Backend) ListMergeRequests(ctx context.Context, repoName string, state *models.MergeRequestState) ([]models.MergeRequest, error) {
	repoName = utils.SanitizeRepo(repoName)

	r, err := d.Repository(ctx, repoName)
	if err != nil {
		return nil, err
	}

	var mrs []models.MergeRequest
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		if state == nil {
			mrs, err = d.store.GetMergeRequestsByRepoID(ctx, tx, r.ID())
		} else {
			mrs, err = d.store.GetMergeRequestsByRepoIDAndState(ctx, tx, r.ID(), *state)
		}
		return err
	}); err != nil {
		return nil, db.WrapError(err)
	}

	return mrs, nil
}

// UpdateMergeRequest updates a merge request.
func (d *Backend) UpdateMergeRequest(ctx context.Context, repoName string, mrID int64, title string, description string) error {
	repoName = utils.SanitizeRepo(repoName)

	r, err := d.Repository(ctx, repoName)
	if err != nil {
		return err
	}

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return d.store.UpdateMergeRequest(ctx, tx, r.ID(), mrID, title, description)
	}); err != nil {
		return db.WrapError(err)
	}

	return nil
}

// MergeMergeRequest merges a merge request.
func (d *Backend) MergeMergeRequest(ctx context.Context, repoName string, mrID int64) error {
	repoName = utils.SanitizeRepo(repoName)

	r, err := d.Repository(ctx, repoName)
	if err != nil {
		return err
	}

	// Get current user
	user := proto.UserFromContext(ctx)
	if user == nil {
		return proto.ErrUserNotFound
	}

	// Get merge request details
	mr, err := d.GetMergeRequest(ctx, repoName, mrID)
	if err != nil {
		return err
	}

	if mr.State != models.MergeRequestStateOpen {
		return errors.New("merge request is not open")
	}

	// Open git repository
	gr, err := r.Open()
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Perform the merge
	if err := performMerge(gr, mr.SourceBranch, mr.TargetBranch, user.Username()); err != nil {
		return fmt.Errorf("failed to merge: %w", err)
	}

	// Update merge request state
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return d.store.MergeMergeRequest(ctx, tx, r.ID(), mrID, user.ID())
	}); err != nil {
		return db.WrapError(err)
	}

	return nil
}

// CloseMergeRequest closes a merge request.
func (d *Backend) CloseMergeRequest(ctx context.Context, repoName string, mrID int64) error {
	repoName = utils.SanitizeRepo(repoName)

	r, err := d.Repository(ctx, repoName)
	if err != nil {
		return err
	}

	// Get current user
	user := proto.UserFromContext(ctx)
	if user == nil {
		return proto.ErrUserNotFound
	}

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return d.store.CloseMergeRequest(ctx, tx, r.ID(), mrID, user.ID())
	}); err != nil {
		return db.WrapError(err)
	}

	return nil
}

// ReopenMergeRequest reopens a closed merge request.
func (d *Backend) ReopenMergeRequest(ctx context.Context, repoName string, mrID int64) error {
	repoName = utils.SanitizeRepo(repoName)

	r, err := d.Repository(ctx, repoName)
	if err != nil {
		return err
	}

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return d.store.ReopenMergeRequest(ctx, tx, r.ID(), mrID)
	}); err != nil {
		return db.WrapError(err)
	}

	return nil
}

// performMerge performs a git merge operation.
func performMerge(repo *git.Repository, sourceBranch, targetBranch, author string) error {
	// Checkout target branch
	_, err := git.NewCommand("checkout", targetBranch).RunInDir(repo.Path)
	if err != nil {
		return fmt.Errorf("failed to checkout target branch: %w", err)
	}

	// Merge source branch
	commitMsg := fmt.Sprintf("Merge branch '%s' into '%s'", sourceBranch, targetBranch)
	_, err = git.NewCommand("merge", "--no-ff", "-m", commitMsg, sourceBranch).RunInDir(repo.Path)
	if err != nil {
		return fmt.Errorf("failed to merge branches: %w", err)
	}

	return nil
}
