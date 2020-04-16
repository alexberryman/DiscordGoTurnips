drop table if exists prices;
drop type if exists meridiem;
create type meridiem as ENUM ('am', 'pm');
CREATE TABLE prices
(
    id          BIGSERIAL PRIMARY KEY,
    discord_id  text      NOT NULL references users (discord_id),
    price       int       not null,
    meridiem    meridiem  not null,
    day_of_week int       not null,
    day_of_year int       not null,
    year        int       not null,
    created_at  timestamp not null default now()
);

create index on prices (discord_id);
create index on prices (created_at);
create index on prices (year, day_of_year, day_of_week);


create unique index
    on prices
        (
         meridiem,
         day_of_week,
         day_of_year,
         year
            );