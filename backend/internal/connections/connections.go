package connections

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
)

// Client holds the database connection pool.
type Client struct {
	Pool *pgxpool.Pool
}

// ConnectDB establishes a connection to the PostgreSQL database.
func ConnectDB(databaseURL string, logger *slog.Logger) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database URL: %w", err)
	}

	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxvec.RegisterTypes(ctx, conn)
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool with custom config: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	logger.Info("Database connection established and pgvector type registered")
	return &Client{Pool: pool}, nil
}

// Close gracefully closes the database connection pool.
func (c *Client) Close() {
	c.Pool.Close()
}

// Ping verifies the connection to the database is still alive.
func (c *Client) Ping() error {
	return c.Pool.Ping(context.Background())
}
