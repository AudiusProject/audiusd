-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS core_decoded_tx (
    id BIGSERIAL PRIMARY KEY,
    block_height BIGINT NOT NULL,
    tx_index INTEGER NOT NULL,
    tx_hash TEXT NOT NULL,
    tx_type TEXT NOT NULL,
    tx_data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(block_height, tx_index),
    UNIQUE(tx_hash)
);

CREATE INDEX IF NOT EXISTS core_decoded_tx_block_height_idx ON core_decoded_tx(block_height);
CREATE INDEX IF NOT EXISTS core_decoded_tx_tx_hash_idx ON core_decoded_tx(tx_hash);
CREATE INDEX IF NOT EXISTS core_decoded_tx_tx_type_idx ON core_decoded_tx(tx_type);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS core_decoded_tx;
-- +goose StatementEnd 