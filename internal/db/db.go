package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Open establishes a new database connection using the provided database URL. It returns the connection object or an error if the connection fails.
func Open(ctx context.Context, databaseURL string) (*pgx.Conn, error) {
	return pgx.Connect(ctx, databaseURL)
}
