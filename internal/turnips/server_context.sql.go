// Code generated by sqlc. DO NOT EDIT.
// source: server_context.sql

package turnips

import (
	"context"
)

const countServerContextByDiscordId = `-- name: CountServerContextByDiscordId :one
SELECT count(*)
FROM server_context
where discord_id = $1
  and server_id = $2
`

type CountServerContextByDiscordIdParams struct {
	DiscordID string `json:"discord_id"`
	ServerID  string `json:"server_id"`
}

func (q *Queries) CountServerContextByDiscordId(ctx context.Context, arg CountServerContextByDiscordIdParams) (int64, error) {
	row := q.queryRow(ctx, q.countServerContextByDiscordIdStmt, countServerContextByDiscordId, arg.DiscordID, arg.ServerID)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const createServerContext = `-- name: CreateServerContext :one
INSERT INTO server_context (discord_id, server_id, username)
VALUES ($1, $2, $3)
RETURNING id, server_id, username, discord_id
`

type CreateServerContextParams struct {
	DiscordID string `json:"discord_id"`
	ServerID  string `json:"server_id"`
	Username  string `json:"username"`
}

func (q *Queries) CreateServerContext(ctx context.Context, arg CreateServerContextParams) (ServerContext, error) {
	row := q.queryRow(ctx, q.createServerContextStmt, createServerContext, arg.DiscordID, arg.ServerID, arg.Username)
	var i ServerContext
	err := row.Scan(
		&i.ID,
		&i.ServerID,
		&i.Username,
		&i.DiscordID,
	)
	return i, err
}

const deleteServerContext = `-- name: DeleteServerContext :exec
DELETE
FROM server_context
WHERE discord_id = $1
`

func (q *Queries) DeleteServerContext(ctx context.Context, discordID string) error {
	_, err := q.exec(ctx, q.deleteServerContextStmt, deleteServerContext, discordID)
	return err
}

const getServerContext = `-- name: GetServerContext :one
SELECT id, server_id, username, discord_id
FROM server_context
WHERE discord_id = $1
LIMIT 1
`

func (q *Queries) GetServerContext(ctx context.Context, discordID string) (ServerContext, error) {
	row := q.queryRow(ctx, q.getServerContextStmt, getServerContext, discordID)
	var i ServerContext
	err := row.Scan(
		&i.ID,
		&i.ServerID,
		&i.Username,
		&i.DiscordID,
	)
	return i, err
}

const listServerContext = `-- name: ListServerContext :many
SELECT id, server_id, username, discord_id
FROM server_context
ORDER BY discord_id
`

func (q *Queries) ListServerContext(ctx context.Context) ([]ServerContext, error) {
	rows, err := q.query(ctx, q.listServerContextStmt, listServerContext)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ServerContext
	for rows.Next() {
		var i ServerContext
		if err := rows.Scan(
			&i.ID,
			&i.ServerID,
			&i.Username,
			&i.DiscordID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateUsername = `-- name: UpdateUsername :one
UPDATE server_context
set username = $2
where discord_id = $1
  and server_id = $3
RETURNING id, server_id, username, discord_id
`

type UpdateUsernameParams struct {
	DiscordID string `json:"discord_id"`
	Username  string `json:"username"`
	ServerID  string `json:"server_id"`
}

func (q *Queries) UpdateUsername(ctx context.Context, arg UpdateUsernameParams) (ServerContext, error) {
	row := q.queryRow(ctx, q.updateUsernameStmt, updateUsername, arg.DiscordID, arg.Username, arg.ServerID)
	var i ServerContext
	err := row.Scan(
		&i.ID,
		&i.ServerID,
		&i.Username,
		&i.DiscordID,
	)
	return i, err
}
