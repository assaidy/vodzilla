-- +goose Up
create table video_service.pending_videos (
  id           uuid         primary key,
  owner_id     uuid         not null,
  title        varchar(256) not null,
  description  varchar(500),
  created_at   timestamptz  not null default now()
);

-- +goose Down
drop table video_service.pending_videos;
