package migrate

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
)

const (
	issuesName    = "issues"
	issuesVersion = 5
)

var issues = Migration{
	Name:    issuesName,
	Version: issuesVersion,
	Migrate: func(ctx context.Context, tx *db.Tx) error {
		return migrateUp(ctx, tx, issuesVersion, issuesName)
	},
	Rollback: func(ctx context.Context, tx *db.Tx) error {
		return migrateDown(ctx, tx, issuesVersion, issuesName)
	},
}
