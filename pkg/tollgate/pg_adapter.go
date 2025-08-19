// Package tollgate provides a tollgate middleware for HTTP requests.
package tollgate

import (
	"context"
	"database/sql"
)

type Adapter interface {
	Consume(ctx context.Context, ticket string) (int, error)
	Balance(ctx context.Context, ticket string) (int, error)
	// Topup(ctx context.Context, ticket string, amount int) error
}

type PostgresAdapter struct {
	db *sql.DB
}

func NewPostgresAdapter(db *sql.DB) Adapter {
	return &PostgresAdapter{db: db}
}

func (a *PostgresAdapter) Consume(ctx context.Context, ticket string) (int, error) {
	return 0, nil
}

func (a *PostgresAdapter) Balance(ctx context.Context, ticket string) (int, error) {
	return 0, nil
}

// func (a *PostgresAdapter) Topup(ctx context.Context, ticket string, amount int) error {
// 	return nil
// }
