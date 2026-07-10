-- +goose Up
create schema notification_service;

create table notification_service.notifications (
  id         uuid        primary key,
  user_id    uuid        not null,
  kind       varchar     not null,
  payload    jsonb       not null,
  created_at timestamptz not null default now(),
  is_read    bool        not null default false
);

-- +goose Down
drop table notification_service.notifications;
drop schema notification_service cascade;
