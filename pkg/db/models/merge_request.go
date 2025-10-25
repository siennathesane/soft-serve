package models

import (
	"database/sql"
	"time"
)

// MergeRequestState represents the state of a merge request.
type MergeRequestState int

const (
	// MergeRequestStateOpen is an open merge request.
	MergeRequestStateOpen MergeRequestState = iota
	// MergeRequestStateMerged is a merged merge request.
	MergeRequestStateMerged
	// MergeRequestStateClosed is a closed merge request.
	MergeRequestStateClosed
)

// String returns the string representation of the merge request state.
func (s MergeRequestState) String() string {
	switch s {
	case MergeRequestStateOpen:
		return "open"
	case MergeRequestStateMerged:
		return "merged"
	case MergeRequestStateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// MergeRequest represents a merge request.
type MergeRequest struct {
	ID           int64              `db:"id"`
	RepoID       int64              `db:"repo_id"`
	Title        string             `db:"title"`
	Description  string             `db:"description"`
	SourceBranch string             `db:"source_branch"`
	TargetBranch string             `db:"target_branch"`
	State        MergeRequestState  `db:"state"`
	AuthorID     int64              `db:"author_id"`
	MergedBy     sql.NullInt64      `db:"merged_by"`
	MergedAt     sql.NullTime       `db:"merged_at"`
	ClosedBy     sql.NullInt64      `db:"closed_by"`
	ClosedAt     sql.NullTime       `db:"closed_at"`
	CreatedAt    time.Time          `db:"created_at"`
	UpdatedAt    time.Time          `db:"updated_at"`
}
