CREATE TABLE IF NOT EXISTS game_server_analytics (
    server_ip inet PRIMARY KEY,
    games_created bigint NOT NULL,
    rematch_called bigint NOT NULL,
    last_updated timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
)