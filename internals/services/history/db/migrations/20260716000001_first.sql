-- +goose Up
create schema history_service;

create table history_service.watch_history (
    id         bigserial   primary key,
    user_id    uuid        not null,
    video_id   uuid        not null,
    watched_at timestamptz not null default now()
);

-- +goose Down
drop schema history_service cascade;
