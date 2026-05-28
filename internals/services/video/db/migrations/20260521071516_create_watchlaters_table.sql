-- +goose Up
create table video_service.watchlaters (
  video_id varchar not null references video_service.videos (id) on delete cascade,
  user_id  varchar not null,
  added_at timestamptz not null default now(),

  primary key (video_id, user_id)
);

-- +goose Down
drop table video_service.watch_later;
