-- +goose Up
create table video_service.playlists (
  id         uuid        primary key,
  name       varchar(50) not null,
  owner_id   uuid        not null,
  created_at timestamptz not null default now(),

  unique (name, owner_id)
);

-- +goose Down
drop table video_service.playlists;
