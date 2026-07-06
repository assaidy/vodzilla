-- +goose Up
create table media_service.videos (
  id         uuid        primary key,
  object_key varchar     not null unique,
  created_at timestamptz not null default now()
);

-- +goose Down
drop table media_service.videos;
