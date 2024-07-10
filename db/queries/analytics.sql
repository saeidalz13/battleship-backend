-- name: UpdateServerHit :exec
INSERT INTO server_hit_counts (server_ip, hit_count, last_updated)
VALUES ($1, 1, CURRENT_TIMESTAMP) ON CONFLICT (server_ip) DO
UPDATE
SET server_hit_counts.hit_count = server_hit_counts.hit_count + 1,
    last_updated = CURRENT_TIMESTAMP;