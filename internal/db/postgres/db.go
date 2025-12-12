package postgres

import (
	"context"

	"github.com/travisbale/knowhere/db/postgres"
	"github.com/travisbale/mailman/internal/db/postgres/internal/sqlc"
)

// DB is a type alias for the generic knowhere DB with sqlc.Queries
type DB = postgres.DB[*sqlc.Queries]

// NewDB creates a new database connection pool
func NewDB(ctx context.Context, databaseURL string) (*DB, error) {
	// Wrap sqlc.New to satisfy the generic constructor signature
	queries := func(d any) *sqlc.Queries {
		return sqlc.New(d.(sqlc.DBTX))
	}

	return postgres.NewDB(ctx, databaseURL, queries, nil)
}
