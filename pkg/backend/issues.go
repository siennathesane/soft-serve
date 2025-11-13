package backend

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/utils"
)

// CreateIssue creates a new issue for a repository.
func (d *Backend) CreateIssue(ctx context.Context, repoName string, title string, description string) (int64, error) {
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

	// Create issue in database
	var issueID int64
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		issueID, err = d.store.CreateIssue(ctx, tx, r.ID(), user.ID(), title, description)
		return err
	}); err != nil {
		return 0, db.WrapError(err)
	}

	return issueID, nil
}

// GetIssue returns an issue by its ID.
func (d *Backend) GetIssue(ctx context.Context, repoName string, issueID int64) (models.Issue, error) {
	repoName = utils.SanitizeRepo(repoName)

	r, err := d.Repository(ctx, repoName)
	if err != nil {
		return models.Issue{}, err
	}

	var issue models.Issue
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		issue, err = d.store.GetIssueByID(ctx, tx, r.ID(), issueID)
		return err
	}); err != nil {
		return models.Issue{}, db.WrapError(err)
	}

	return issue, nil
}

// ListIssues returns all issues for a repository.
func (d *Backend) ListIssues(ctx context.Context, repoName string, state *models.IssueState) ([]models.Issue, error) {
	repoName = utils.SanitizeRepo(repoName)

	r, err := d.Repository(ctx, repoName)
	if err != nil {
		return nil, err
	}

	var issues []models.Issue
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		if state == nil {
			issues, err = d.store.GetIssuesByRepoID(ctx, tx, r.ID())
		} else {
			issues, err = d.store.GetIssuesByRepoIDAndState(ctx, tx, r.ID(), *state)
		}
		return err
	}); err != nil {
		return nil, db.WrapError(err)
	}

	return issues, nil
}

// UpdateIssue updates an issue.
func (d *Backend) UpdateIssue(ctx context.Context, repoName string, issueID int64, title string, description string) error {
	repoName = utils.SanitizeRepo(repoName)

	r, err := d.Repository(ctx, repoName)
	if err != nil {
		return err
	}

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return d.store.UpdateIssue(ctx, tx, r.ID(), issueID, title, description)
	}); err != nil {
		return db.WrapError(err)
	}

	return nil
}

// CloseIssue closes an issue.
func (d *Backend) CloseIssue(ctx context.Context, repoName string, issueID int64) error {
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
		return d.store.CloseIssue(ctx, tx, r.ID(), issueID, user.ID())
	}); err != nil {
		return db.WrapError(err)
	}

	return nil
}

// ReopenIssue reopens a closed issue.
func (d *Backend) ReopenIssue(ctx context.Context, repoName string, issueID int64) error {
	repoName = utils.SanitizeRepo(repoName)

	r, err := d.Repository(ctx, repoName)
	if err != nil {
		return err
	}

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return d.store.ReopenIssue(ctx, tx, r.ID(), issueID)
	}); err != nil {
		return db.WrapError(err)
	}

	return nil
}

// AddIssueDependency creates a dependency relationship where issueID depends on dependsOnID.
func (d *Backend) AddIssueDependency(ctx context.Context, repoName string, issueID int64, dependsOnID int64) error {
	repoName = utils.SanitizeRepo(repoName)

	r, err := d.Repository(ctx, repoName)
	if err != nil {
		return err
	}

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return d.store.AddIssueDependency(ctx, tx, r.ID(), issueID, dependsOnID)
	}); err != nil {
		return db.WrapError(err)
	}

	return nil
}

// RemoveIssueDependency removes a dependency relationship.
func (d *Backend) RemoveIssueDependency(ctx context.Context, repoName string, issueID int64, dependsOnID int64) error {
	repoName = utils.SanitizeRepo(repoName)

	r, err := d.Repository(ctx, repoName)
	if err != nil {
		return err
	}

	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return d.store.RemoveIssueDependency(ctx, tx, r.ID(), issueID, dependsOnID)
	}); err != nil {
		return db.WrapError(err)
	}

	return nil
}

// GetIssueDependencies returns all issues that the given issue depends on.
func (d *Backend) GetIssueDependencies(ctx context.Context, repoName string, issueID int64) ([]models.Issue, error) {
	repoName = utils.SanitizeRepo(repoName)

	r, err := d.Repository(ctx, repoName)
	if err != nil {
		return nil, err
	}

	var dependencies []models.Issue
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		dependencies, err = d.store.GetIssueDependencies(ctx, tx, r.ID(), issueID)
		return err
	}); err != nil {
		return nil, db.WrapError(err)
	}

	return dependencies, nil
}

// GetIssueDependents returns all issues that depend on the given issue.
func (d *Backend) GetIssueDependents(ctx context.Context, repoName string, issueID int64) ([]models.Issue, error) {
	repoName = utils.SanitizeRepo(repoName)

	r, err := d.Repository(ctx, repoName)
	if err != nil {
		return nil, err
	}

	var dependents []models.Issue
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		dependents, err = d.store.GetIssueDependents(ctx, tx, r.ID(), issueID)
		return err
	}); err != nil {
		return nil, db.WrapError(err)
	}

	return dependents, nil
}
