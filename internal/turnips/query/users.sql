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
INSERT INTO users (discord_id, username)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateFriendCode :one
UPDATE users
set friend_code = $2
where discord_id = $1
RETURNING *;

-- name: UpdateTimeZone :one
UPDATE users
set time_zone = $2
where discord_id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE
FROM users
WHERE discord_id = $1;