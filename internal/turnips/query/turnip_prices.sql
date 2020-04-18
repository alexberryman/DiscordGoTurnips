-- name: GetWeeksPriceHistoryByAccount :many
SELECT *
FROM turnip_prices
WHERE discord_id = $1
  and day_of_year > extract(DOY FROM now()) - 7
  and year = extract(year from now())
order by day_of_year, am_pm;

-- name: GetWeeksPriceHistoryByServer :many
SELECT *
FROM turnip_prices
         inner join nicknames nick on turnip_prices.discord_id = nick.discord_id
WHERE nick.server_id = $1
  and day_of_year > extract(DOY FROM now()) - 7
  and year = extract(year from now())
order by day_of_year, am_pm;

-- name: ListPrices :many
SELECT *
FROM turnip_prices
ORDER BY created_at;

-- name: CountPricesByDiscordId :one
SELECT count(*)
FROM turnip_prices
where discord_id = $1;

-- name: CreatePrice :one
INSERT INTO turnip_prices (discord_id, price, am_pm, day_of_week, day_of_year, year)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdatePrice :one
update turnip_prices
set price = $2
where discord_id = $1
  and am_pm = $3
  and day_of_week = $4
  and day_of_year = $5
  and year = $6
returning *;

-- name: DeletePricesForUser :exec
DELETE
FROM turnip_prices
WHERE discord_id = $1;