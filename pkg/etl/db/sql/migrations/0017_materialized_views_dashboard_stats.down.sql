-- Migration 0017 Down: Drop Dashboard Stats Materialized Views

-- Drop the refresh function
DROP FUNCTION IF EXISTS refresh_dashboard_materialized_views();

-- Drop materialized views
DROP MATERIALIZED VIEW IF EXISTS mv_dashboard_network_rates;
DROP MATERIALIZED VIEW IF EXISTS mv_dashboard_validator_stats;
DROP MATERIALIZED VIEW IF EXISTS mv_dashboard_transaction_breakdown;
DROP MATERIALIZED VIEW IF EXISTS mv_dashboard_transaction_stats; 
