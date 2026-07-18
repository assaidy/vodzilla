-- +goose Up
create schema reaction_service;

create table reaction_service.views (
  target_id  uuid        not null,
  user_id    uuid        not null,
  kind       varchar     not null,
  created_at timestamptz not null default now(),

  primary key (target_id, user_id, kind)
);

create table reaction_service.feelings (
  target_id   uuid        not null,
  user_id     uuid        not null,
  target_kind varchar     not null,
  kind        varchar     not null,
  added_at    timestamptz not null default now(),

  primary key(target_id, user_id, target_kind)
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
