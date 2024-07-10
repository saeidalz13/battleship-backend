CREATE TABLE IF NOT EXISTS server_hit_counts (
    server_ip inet PRIMARY KEY,
    hit_count bigint NOT NULL,
    last_updated timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
)