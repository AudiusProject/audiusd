# Integration Example

Here's how to integrate the materialized view manager into your ETL service:

## 1. Update ETL Service Initialization

```go
// In pkg/etl/service.go or similar
func NewETLService(db *db.Queries, dbPool *pgxpool.Pool, logger *slog.Logger) *ETLService {
    // Create materialized view manager
    viewManager := db.NewMaterializedViewManager(dbPool, logger)
    
    service := &ETLService{
        db:          db,
        logger:      logger,
        viewManager: viewManager,
    }
    
    // Start periodic refresh every 5 minutes
    ctx := context.Background()
    service.viewManager.StartPeriodicRefresh(ctx, 5*time.Minute)
    
    return service
}
```

## 2. Add Health Check Endpoint

```go
// Add to your ETL service struct
type ETLService struct {
    db          *db.Queries
    logger      *slog.Logger
    viewManager *db.MaterializedViewManager
}

// Add health check method
func (e *ETLService) GetHealth(ctx context.Context, req *connect.Request[v1.GetHealthRequest]) (*connect.Response[v1.GetHealthResponse], error) {
    // Check materialized view health
    if err := e.viewManager.HealthCheck(ctx); err != nil {
        return connect.NewResponse(&v1.GetHealthResponse{
            Status: "unhealthy",
            Details: fmt.Sprintf("Materialized view error: %v", err),
        }), nil
    }
    
    return connect.NewResponse(&v1.GetHealthResponse{
        Status: "healthy",
        Details: "All systems operational",
    }), nil
}
```

## 3. Manual Refresh Endpoint (Optional)

```go
// Add admin endpoint to manually refresh views
func (e *ETLService) RefreshViews(ctx context.Context, req *connect.Request[v1.RefreshViewsRequest]) (*connect.Response[v1.RefreshViewsResponse], error) {
    start := time.Now()
    
    if err := e.viewManager.RefreshSlaRollupViews(ctx); err != nil {
        return nil, fmt.Errorf("failed to refresh views: %w", err)
    }
    
    return connect.NewResponse(&v1.RefreshViewsResponse{
        Success: true,
        Duration: time.Since(start).String(),
    }), nil
}
```

## 4. Monitoring Integration

```go
// Add metrics collection
func (e *ETLService) collectViewMetrics(ctx context.Context) {
    status, err := e.viewManager.GetViewRefreshStatus(ctx)
    if err != nil {
        e.logger.Error("Failed to get view status", "error", err)
        return
    }
    
    for viewName, lastRefresh := range status {
        age := time.Since(lastRefresh)
        // Report to your metrics system
        // e.g., prometheus.ViewAgeGauge.WithLabelValues(viewName).Set(age.Seconds())
        e.logger.Debug("View status", "view", viewName, "age", age)
    }
}
```

## 5. Graceful Shutdown

```go
func (e *ETLService) Shutdown(ctx context.Context) error {
    // The periodic refresh will stop automatically when ctx is cancelled
    e.logger.Info("ETL service shutting down")
    return nil
}
```

## 6. Environment Configuration

Add these environment variables for configuration:

```bash
# Materialized view refresh interval (default: 5m)
MATERIALIZED_VIEW_REFRESH_INTERVAL=5m

# Max age before forcing refresh (default: 15m)
MATERIALIZED_VIEW_MAX_AGE=15m
```

## 7. Usage in Handlers

```go
// Use optimized queries for performance-critical paths
func (e *ETLService) GetStats(ctx context.Context, req *connect.Request[v1.GetStatsRequest]) (*connect.Response[v1.GetStatsResponse], error) {
    // Force refresh if views are stale (fallback safety)
    if err := e.viewManager.ForceRefreshIfStale(ctx, 15*time.Minute); err != nil {
        e.logger.Warn("Failed to refresh stale views", "error", err)
        // Continue anyway - optimized query will still be faster than original
    }
    
    // Use the optimized dashboard query
    latestSlaRollup, err := e.db.GetLatestSlaRollupForDashboardOptimized(ctx)
    if err != nil {
        // Handle error...
    }
    
    // Rest of implementation...
}
```

This integration provides:
- Automatic periodic refresh of materialized views
- Health checking for monitoring
- Manual refresh capability for admin operations
- Graceful handling of stale data
- Performance monitoring and logging 
