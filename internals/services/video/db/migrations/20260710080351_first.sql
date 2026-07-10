-- +goose Up
create schema video_service;

create table video_service.videos (
  id           uuid         primary key,
  owner_id     uuid         not null,
  title        varchar(256) not null,
  description  varchar(500),
  created_at   timestamptz  not null default now()
);

create table video_service.watchlaters (
  id       bigserial primary key,
  video_id uuid not null references video_service.videos (id) on delete cascade,
  user_id  uuid not null,
  added_at timestamptz not null default now(),

  unique (video_id, user_id)
);

create table video_service.playlists (
  id         uuid        primary key,
  name       varchar(50) not null,
  owner_id   uuid        not null,
  created_at timestamptz not null default now(),

  unique (name, owner_id)
);

create table video_service.playlist_videos (
  id          bigserial primary key,
  playlist_id uuid        not null references video_service.playlists (id) on delete cascade,
  video_id    uuid        not null references video_service.videos (id)    on delete cascade,
  added_at    timestamptz not null default now(),

  unique (playlist_id, video_id)
);

create table video_service.pending_videos (
  id           uuid         primary key,
  owner_id     uuid         not null,
  title        varchar(256) not null,
  description  varchar(500),
  created_at   timestamptz  not null default now()
);

-- +goose Down
drop table video_service.pending_videos;
drop table video_service.playlist_videos;
drop table video_service.playlists;
drop table video_service.watchlaters;
drop table video_service.videos;
drop schema video_service cascade;
