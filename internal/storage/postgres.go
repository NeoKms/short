package storage

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vladislav/short/internal/link"
)

//go:embed migrations/001_init.sql
var initialMigration string

type Postgres struct{ pool *pgxpool.Pool }

func Open(ctx context.Context, databaseURL string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Postgres{pool: pool}, nil
}

func (p *Postgres) Close()                         { p.pool.Close() }
func (p *Postgres) Ping(ctx context.Context) error { return p.pool.Ping(ctx) }

func (p *Postgres) Migrate(ctx context.Context) error {
	if _, err := p.pool.Exec(ctx, initialMigration); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

func (p *Postgres) Create(ctx context.Context, value link.Link) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO links (code, original_url, created_at, expires_at) VALUES ($1, $2, $3, $4)`,
		value.Code, value.OriginalURL, value.CreatedAt, value.ExpiresAt,
	)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return link.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("insert link: %w", err)
	}
	return nil
}

func (p *Postgres) Get(ctx context.Context, code string) (link.Link, error) {
	var value link.Link
	err := p.pool.QueryRow(ctx,
		`SELECT code, original_url, created_at, expires_at FROM links WHERE code = $1`, code,
	).Scan(&value.Code, &value.OriginalURL, &value.CreatedAt, &value.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return link.Link{}, link.ErrNotFound
	}
	if err != nil {
		return link.Link{}, fmt.Errorf("select link: %w", err)
	}
	return value, nil
}
