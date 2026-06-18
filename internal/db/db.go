package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool wraps a pgx connection pool used by the rule loader and audit writer.
type Pool struct {
	*pgxpool.Pool
}

func Connect(ctx context.Context, dsn string) (*Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Pool{pool}, nil
}

// Ping reports the database status as a short string, suitable for the
// /health endpoint.
func (p *Pool) Ping(ctx context.Context) string {
	if err := p.Pool.Ping(ctx); err != nil {
		return err.Error()
	}
	return "ok"
}
