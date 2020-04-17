create table server_context
(
    id         BIGSERIAL PRIMARY KEY,
    server_id  text not null,
    username   text NOT NULL,
    discord_id text NOT NULL references users (discord_id)
);

create index on server_context (server_id);
create index on server_context (discord_id);
create unique index on server_context (server_id, discord_id);

insert into server_context (server_id, username, discord_id)
SELECT '693251733593391116', username, discord_id
from users;
grant all on all tables in schema public to turnips;

alter table users
    drop column if exists username;