-- name: IncrementGamesCreatedCount :exec
INSERT INTO game_server_analytics (server_ip, games_created, last_updated)
VALUES ($1, 1, CURRENT_TIMESTAMP) ON CONFLICT (server_ip) DO
UPDATE
SET games_created = game_server_analytics.games_created + 1,
    last_updated = CURRENT_TIMESTAMP;

-- name: IncrementRematchCalledCount :exec
INSERT INTO game_server_analytics (server_ip, rematch_called, last_updated)
VALUES ($1, 1, CURRENT_TIMESTAMP) ON CONFLICT (server_ip) DO
UPDATE
SET rematch_called = game_server_analytics.rematch_called + 1,
    last_updated = CURRENT_TIMESTAMP;

-- name: GetGamesCreatedCount :one
SELECT games_created FROM game_server_analytics WHERE server_ip = $1;

-- name: GetRematchCalledCount :one
SELECT rematch_called FROM game_server_analytics WHERE server_ip = $1;