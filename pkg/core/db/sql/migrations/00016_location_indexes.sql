-- +migrate Up

-- Drop existing indexes first to avoid duplicates
DROP INDEX IF EXISTS core_tx_decoded_plays_city_idx;
DROP INDEX IF EXISTS core_tx_decoded_plays_region_idx;
DROP INDEX IF EXISTS core_tx_decoded_plays_country_idx;

-- Add optimized composite index for location queries
CREATE INDEX IF NOT EXISTS idx_decoded_plays_location 
ON core_tx_decoded_plays (country, region, city);

-- Add individual filtered indexes for each location column
CREATE INDEX IF NOT EXISTS idx_decoded_plays_country 
ON core_tx_decoded_plays (country) 
WHERE country IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_decoded_plays_region 
ON core_tx_decoded_plays (region) 
WHERE region IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_decoded_plays_city 
ON core_tx_decoded_plays (city) 
WHERE city IS NOT NULL;

-- Add index for play count queries with nulls last to optimize DESC ordering
CREATE INDEX IF NOT EXISTS idx_decoded_plays_location_counts 
ON core_tx_decoded_plays (country NULLS LAST, region NULLS LAST, city NULLS LAST, played_at DESC);

-- For GetDecodedPlaysByTimeRange queries
CREATE INDEX idx_decoded_plays_played_at 
ON core_tx_decoded_plays (played_at DESC);

-- For GetDecodedPlaysByUser queries
CREATE INDEX idx_decoded_plays_user_played_at 
ON core_tx_decoded_plays (user_id, played_at DESC);

-- For GetDecodedPlaysByTrack queries
CREATE INDEX idx_decoded_plays_track_played_at 
ON core_tx_decoded_plays (track_id, played_at DESC);

-- For GetDecodedTransactionsByType and time-based queries
CREATE INDEX idx_decoded_tx_type_height 
ON core_tx_decoded (tx_type, block_height DESC);

-- For SLA rollup queries
CREATE INDEX idx_sla_node_reports_address_rollup 
ON sla_node_reports (address, sla_rollup_id);

-- +migrate Down
DROP INDEX IF EXISTS idx_decoded_plays_location_counts;
DROP INDEX IF EXISTS idx_decoded_plays_city;
DROP INDEX IF EXISTS idx_decoded_plays_region;
DROP INDEX IF EXISTS idx_decoded_plays_country;
DROP INDEX IF EXISTS idx_decoded_plays_location;
DROP INDEX IF EXISTS idx_decoded_plays_played_at;
DROP INDEX IF EXISTS idx_decoded_plays_user_played_at;
DROP INDEX IF EXISTS idx_decoded_plays_track_played_at;
DROP INDEX IF EXISTS idx_decoded_tx_type_height;
DROP INDEX IF EXISTS idx_sla_node_reports_address_rollup;

-- Restore original indexes
CREATE INDEX IF NOT EXISTS core_tx_decoded_plays_city_idx ON core_tx_decoded_plays(city);
CREATE INDEX IF NOT EXISTS core_tx_decoded_plays_region_idx ON core_tx_decoded_plays(region);
CREATE INDEX IF NOT EXISTS core_tx_decoded_plays_country_idx ON core_tx_decoded_plays(country); 