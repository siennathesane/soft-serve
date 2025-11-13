package models

import (
	"database/sql"
	"time"
)

// IssueState represents the state of an issue.
type IssueState int

const (
	// IssueStateOpen is an open issue.
	IssueStateOpen IssueState = iota
	// IssueStateClosed is a closed issue.
	IssueStateClosed
)

// String returns the string representation of the issue state.
func (s IssueState) String() string {
	switch s {
	case IssueStateOpen:
		return "open"
	case IssueStateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// Issue represents an issue.
type Issue struct {
	ID          int64         `db:"id"`
	RepoID      int64         `db:"repo_id"`
	Title       string        `db:"title"`
	Description string        `db:"description"`
	State       IssueState    `db:"state"`
	AuthorID    int64         `db:"author_id"`
	ClosedBy    sql.NullInt64 `db:"closed_by"`
	ClosedAt    sql.NullTime  `db:"closed_at"`
	CreatedAt   time.Time     `db:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at"`
}
