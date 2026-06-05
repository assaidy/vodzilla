-- +goose Up
create table media_service.videos (
  id         uuid primary key,      
  object_key varchar not null unique,
  status     varchar not null
);

-- +goose Down
drop table media_service.videos;
