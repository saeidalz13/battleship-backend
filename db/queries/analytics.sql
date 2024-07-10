-- name: UpdateGameCreated :exec
INSERT INTO game_server_analytics (server_ip, games_created, last_updated)
VALUES ($1, 1, CURRENT_TIMESTAMP) ON CONFLICT (server_ip) DO
UPDATE
SET game_server_analytics.games_created = game_server_analytics.games_created + 1,
    last_updated = CURRENT_TIMESTAMP;
-- name: UpdateRematchCalled :exec
INSERT INTO game_server_analytics (server_ip, rematch_called, last_updated)
VALUES ($1, 1, CURRENT_TIMESTAMP) ON CONFLICT (server_ip) DO
UPDATE
SET game_server_analytics.rematch_called = game_server_analytics.rematch_called + 1,
    last_updated = CURRENT_TIMESTAMP;