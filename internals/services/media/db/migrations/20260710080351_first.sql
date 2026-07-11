-- +goose Up
create schema media_service;

create table media_service.videos (
  id         uuid        primary key,
  object_key varchar     not null unique,
  created_at timestamptz not null default now()
);

create table media_service.avatars (
    user_id    uuid        primary key,
    object_key varchar     not null,
    created_at timestamptz not null default now()
);

create table media_service.thumbnails (
    video_id   uuid        primary key,
    object_key varchar     not null,
    created_at timestamptz not null default now()
);

create table media_service.orphan_uploads (
  object_key varchar    primary key,
  user_id    uuid       not null,
  created_at timestamptz not null default now()
);

-- +goose Down
drop schema media_service cascade;
