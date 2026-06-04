-- +goose Up
create table reaction_service.views (
  video_id   uuid        not null,
  user_id    uuid        not null,
  created_at timestamptz not null default now(),

  primary key (video_id, user_id)
);

-- +goose Down
drop table reaction_service.views;
