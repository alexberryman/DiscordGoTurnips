CREATE TABLE users
(
    id          BIGSERIAL PRIMARY KEY,
    username    text        NOT NULL,
    discord_id  text unique not null,
    friend_code text
);

create index on users (discord_id);

