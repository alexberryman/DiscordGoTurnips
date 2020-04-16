-- name: GetUsers :one
SELECT *
FROM users
WHERE discord_id = $1
LIMIT 1;

-- name: ListUsers :many
SELECT *
FROM users
ORDER BY username;

-- name: CountUsersByDiscordId :one
SELECT count(*)
FROM users
where discord_id = $1;

-- name: CreateUser :one
INSERT INTO users (username, discord_id)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateFriendCode :one
UPDATE users
set friend_code = $1
where discord_id = $2
RETURNING *;

-- name: DeleteUser :exec
DELETE
FROM users
WHERE discord_id = $1;