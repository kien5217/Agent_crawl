package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func Open(ctx context.Context, databaseURL string) (*pgx.Conn, error) {
	return pgx.Connect(ctx, databaseURL)
}
