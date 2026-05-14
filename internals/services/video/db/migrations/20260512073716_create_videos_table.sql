-- +goose Up
create table video_service.videos (
  id          varchar      primary key,
  object_key  varchar      not null unique,
  owner_id    varchar      not null,
  title       varchar(256) not null,
  description varchar(500),
  created_at  timestamptz  not null default now(),
  status      varchar(10)  not null
);

-- +goose Down
drop table video_service.videos;
