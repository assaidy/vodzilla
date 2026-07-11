-- +goose Up
create schema reaction_service;

create table reaction_service.views (
  video_id   uuid        not null,
  user_id    uuid        not null,
  created_at timestamptz not null default now(),

  primary key (video_id, user_id)
);

create table reaction_service.feelings (
  for_id   uuid        not null,
  user_id  uuid        not null,
  kind     varchar     not null,
  added_at timestamptz not null default now(),

  primary key(for_id, user_id)
);

create table reaction_service.comments (
  id         uuid         primary key,
  for_id     uuid         not null,
  user_id    uuid         not null,
  content    varchar(500) not null,
  created_at timestamptz  not null default now()
);

-- +goose Down
drop schema reaction_service cascade;
