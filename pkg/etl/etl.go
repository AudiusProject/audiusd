package etl

import (
	"context"
	"fmt"
	"os"

	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Run(ctx context.Context, logger *common.Logger) error {
	logger.Info("Starting ETL service...")

	// Get database connection string from environment
	dbUrl := os.Getenv("dbUrl")
	if dbUrl == "" {
		return fmt.Errorf("dbUrl environment variable not set")
	}

	// Create connection pool
	pgConfig, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return fmt.Errorf("error parsing database config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgConfig)
	if err != nil {
		return fmt.Errorf("error creating database pool: %v", err)
	}
	defer pool.Close()

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("error connecting to database: %v", err)
	}

	logger.Info("Successfully connected to database")

	// Block until context is cancelled
	<-ctx.Done()
	logger.Info("ETL service shutting down...")
	return nil
}
