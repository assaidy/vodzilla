-- +goose Up
create table video_service.playlists (
  id         varchar     primary key,
  name       varchar(50) not null,
  owner_id   varchar     not null,
  created_at timestamptz not null default now(),

  unique (name, owner_id)
);

-- +goose Down
drop table video_service.playlists;
