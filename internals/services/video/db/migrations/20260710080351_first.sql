-- +goose Up
create schema video_service;

create table video_service.videos (
  id           uuid         primary key,
  user_id     uuid         not null,
  title        varchar(256) not null,
  description  varchar(500),
  created_at   timestamptz  not null default now()
);

create table video_service.watchlaters (
  id       bigserial   primary key,
  video_id uuid        not null references video_service.videos (id) on delete cascade,
  user_id  uuid        not null,
  added_at timestamptz not null default now(),

  unique (video_id, user_id)
);

create table video_service.playlists (
  id          uuid        primary key,
  name        varchar(50) not null,
  description varchar(500),
  user_id     uuid        not null,
  created_at  timestamptz not null default now(),
  is_public   boolean     not null
);

create table video_service.playlist_videos (
  id          bigserial   primary key,
  playlist_id uuid        not null references video_service.playlists (id) on delete cascade,
  video_id    uuid        not null references video_service.videos (id) on delete cascade,
  added_at    timestamptz not null default now(),

  unique (playlist_id, video_id)
);

create table video_service.pending_videos (
  id           uuid         primary key,
  user_id     uuid         not null,
  title        varchar(256) not null,
  description  varchar(500),
  created_at   timestamptz  not null default now()
);

-- +goose Down
drop schema video_service cascade;
