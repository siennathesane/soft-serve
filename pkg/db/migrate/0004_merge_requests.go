package migrate

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
)

const (
	mergeRequestsName    = "merge_requests"
	mergeRequestsVersion = 4
)

var mergeRequests = Migration{
	Name:    mergeRequestsName,
	Version: mergeRequestsVersion,
	Migrate: func(ctx context.Context, tx *db.Tx) error {
		return migrateUp(ctx, tx, mergeRequestsVersion, mergeRequestsName)
	},
	Rollback: func(ctx context.Context, tx *db.Tx) error {
		return migrateDown(ctx, tx, mergeRequestsVersion, mergeRequestsName)
	},
}
