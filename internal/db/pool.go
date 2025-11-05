package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool wraps pgxpool.Pool for future extensions.
type Pool struct {
	*pgxpool.Pool
}

// Connect establishes a pgx connection pool with sane defaults.
func Connect(ctx context.Context, cfg Config) (*Pool, error) {
	conf, err := pgxpool.ParseConfig(cfg.ConnString())
	if err != nil {
		return nil, err
	}
	// Reasonable defaults for server-side prepared statements and timeouts
	conf.MaxConns = 8
	conf.MinConns = 0
	conf.MaxConnLifetime = 55 * time.Minute
	conf.MaxConnIdleTime = 10 * time.Minute
	conf.HealthCheckPeriod = 30 * time.Second

	p, err := pgxpool.NewWithConfig(ctx, conf)
	if err != nil {
		return nil, err
	}
	// Verify connectivity
	if err := p.Ping(ctx); err != nil {
		p.Close()
		return nil, err
	}
	return &Pool{Pool: p}, nil
}

// Close closes the underlying pool.
func (p *Pool) Close() {
	if p != nil && p.Pool != nil {
		p.Pool.Close()
	}
}
