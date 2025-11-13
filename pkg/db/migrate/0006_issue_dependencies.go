package migrate

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
)

const (
	issueDependenciesName    = "issue_dependencies"
	issueDependenciesVersion = 6
)

var issueDependencies = Migration{
	Name:    issueDependenciesName,
	Version: issueDependenciesVersion,
	Migrate: func(ctx context.Context, tx *db.Tx) error {
		return migrateUp(ctx, tx, issueDependenciesVersion, issueDependenciesName)
	},
	Rollback: func(ctx context.Context, tx *db.Tx) error {
		return migrateDown(ctx, tx, issueDependenciesVersion, issueDependenciesName)
	},
}
