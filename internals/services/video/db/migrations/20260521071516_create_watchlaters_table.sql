-- +goose Up
create table video_service.watchlaters (
  video_id uuid not null references video_service.videos (id) on delete cascade,
  user_id  uuid not null,
  added_at timestamptz not null default now(),

  primary key (video_id, user_id)
);

-- +goose Down
drop table video_service.watchlaters;
