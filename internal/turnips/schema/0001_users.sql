drop table if exists users cascade;
CREATE TABLE users
(
    id          BIGSERIAL PRIMARY KEY,
    discord_id  text unique not null,
    friend_code text,
    time_zone   text        not null default 'America/Chicago'
);

create index on users (discord_id);

