drop index if exists etl_latest_indexed_block_block_height_idx;
drop table if exists etl_latest_indexed_block;

drop index if exists etl_plays_block_height_idx;
drop index if exists etl_plays_played_at_idx;
drop index if exists etl_plays_track_id_idx;
drop index if exists etl_plays_address_idx;
drop table if exists etl_plays;
