-- +goose Up
create table video_service.playlist_videos (
  id          bigserial primary key,
  playlist_id uuid        not null references video_service.playlists (id) on delete cascade,
  video_id    uuid        not null references video_service.videos (id)    on delete cascade,
  added_at    timestamptz not null default now(),

  unique (playlist_id, video_id)
);

-- +goose Down
drop table video_service.playlist_videos;
