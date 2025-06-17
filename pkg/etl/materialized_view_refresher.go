package etl

import (
	"context"
	"time"

	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/etl/db"
	"golang.org/x/sync/errgroup"
)

// MaterializedViewRefresher refreshes dashboard materialized views periodically
type MaterializedViewRefresher struct {
	db     db.DBTX
	logger *common.Logger
	ticker *time.Ticker
	done   chan bool
}

// NewMaterializedViewRefresher creates a new refresher service
func NewMaterializedViewRefresher(database db.DBTX, logger *common.Logger) *MaterializedViewRefresher {
	return &MaterializedViewRefresher{
		db:     database,
		logger: logger.Child("mv_refresher"),
		done:   make(chan bool),
	}
}

// Start begins the periodic refresh cycle (every 2 minutes)
// This method blocks and should be run in a goroutine (e.g., via errgroup)
func (r *MaterializedViewRefresher) Start(ctx context.Context) error {
	r.ticker = time.NewTicker(2 * time.Minute)
	defer r.ticker.Stop()

	r.logger.Info("Starting materialized view refresher", "interval", "2m")

	// Initial refresh on startup
	r.refreshViews(ctx)

	for {
		select {
		case <-r.ticker.C:
			r.refreshViews(ctx)
		case <-r.done:
			r.logger.Info("Materialized view refresher stopped via done channel")
			return nil
		case <-ctx.Done():
			r.logger.Info("Materialized view refresher stopped via context cancellation")
			return ctx.Err()
		}
	}
}

// Stop stops the refresher
func (r *MaterializedViewRefresher) Stop() {
	if r.ticker != nil {
		r.ticker.Stop()
	}
	close(r.done)
	r.logger.Info("Stopped materialized view refresher")
}

// refreshViews calls the database function to refresh all dashboard materialized views
// It attempts to refresh independent views in parallel for better performance
func (r *MaterializedViewRefresher) refreshViews(ctx context.Context) {
	start := time.Now()

	// Try parallel refresh first (faster)
	err := r.refreshViewsParallel(ctx)

	// Fallback to sequential refresh if parallel fails
	if err != nil {
		r.logger.Warn("Parallel refresh failed, falling back to sequential", "error", err)
		_, err = r.db.Exec(ctx, "SELECT refresh_dashboard_materialized_views()")
	}

	duration := time.Since(start)

	if err != nil {
		r.logger.Error("Failed to refresh materialized views", "error", err, "duration", duration)
	} else {
		r.logger.Info("Successfully refreshed materialized views", "duration", duration)
	}
}

// refreshViewsParallel refreshes materialized views in parallel where possible
func (r *MaterializedViewRefresher) refreshViewsParallel(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	// Phase 1: Refresh independent views with UNIQUE indexes (can use CONCURRENTLY)
	concurrentViews := []string{
		"mv_dashboard_transaction_stats", // Has unique index on calculated_at
		"mv_dashboard_validator_stats",   // Has unique index on calculated_at
		"mv_sla_rollup",                  // Has unique index on id
		"mv_dashboard_network_rates",     // Has unique index on calculated_at
	}

	for _, viewName := range concurrentViews {
		viewName := viewName // Capture for closure
		g.Go(func() error {
			refreshStart := time.Now()
			_, err := r.db.Exec(gCtx, "REFRESH MATERIALIZED VIEW CONCURRENTLY "+viewName)
			if err != nil {
				r.logger.Error("Failed to refresh concurrent view", "view", viewName, "error", err, "duration", time.Since(refreshStart))
				return err
			}
			r.logger.Debug("Refreshed concurrent view", "view", viewName, "duration", time.Since(refreshStart))
			return nil
		})
	}

	// Wait for phase 1 to complete
	if err := g.Wait(); err != nil {
		return err
	}

	// Phase 2: Refresh views WITHOUT unique indexes (cannot use CONCURRENTLY)
	// These must be refreshed with exclusive locks, but we can still do them in parallel
	g, gCtx = errgroup.WithContext(ctx)
	nonConcurrentViews := []string{
		"mv_sla_rollup_score",                // No unique index - depends on mv_sla_rollup
		"mv_dashboard_transaction_breakdown", // No unique index
	}

	for _, viewName := range nonConcurrentViews {
		viewName := viewName // Capture for closure
		g.Go(func() error {
			refreshStart := time.Now()
			_, err := r.db.Exec(gCtx, "REFRESH MATERIALIZED VIEW "+viewName) // No CONCURRENTLY
			if err != nil {
				r.logger.Error("Failed to refresh non-concurrent view", "view", viewName, "error", err, "duration", time.Since(refreshStart))
				return err
			}
			r.logger.Debug("Refreshed non-concurrent view", "view", viewName, "duration", time.Since(refreshStart))
			return nil
		})
	}

	// Wait for phase 2 to complete
	if err := g.Wait(); err != nil {
		return err
	}

	// Phase 3: Refresh final dependent views
	g, gCtx = errgroup.WithContext(ctx)
	finalViews := []string{
		"mv_sla_rollup_dashboard_stats", // Has unique index, depends on mv_sla_rollup
	}

	for _, viewName := range finalViews {
		viewName := viewName // Capture for closure
		g.Go(func() error {
			refreshStart := time.Now()
			_, err := r.db.Exec(gCtx, "REFRESH MATERIALIZED VIEW CONCURRENTLY "+viewName)
			if err != nil {
				r.logger.Error("Failed to refresh final view", "view", viewName, "error", err, "duration", time.Since(refreshStart))
				return err
			}
			r.logger.Debug("Refreshed final view", "view", viewName, "duration", time.Since(refreshStart))
			return nil
		})
	}

	return g.Wait()
}
