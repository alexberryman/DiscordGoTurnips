-- name: GetServerContext :one
SELECT *
FROM server_context
WHERE discord_id = $1
LIMIT 1;

-- name: ListServerContext :many
SELECT *
FROM server_context
ORDER BY discord_id;

-- name: CountServerContextByDiscordId :one
SELECT count(*)
FROM server_context
where discord_id = $1
  and server_id = $2;

-- name: CreateServerContext :one
INSERT INTO server_context (discord_id, server_id, username)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateUsername :one
UPDATE server_context
set username = $2
where discord_id = $1
  and server_id = $3
RETURNING *;

-- name: DeleteServerContext :exec
DELETE
FROM server_context
WHERE discord_id = $1;