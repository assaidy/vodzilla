-- +goose Up
create table video_service.playlist_videos (
  playlist_id varchar     not null references video_service.playlists (id) on delete cascade,
  video_id    varchar     not null references video_service.videos (id)    on delete cascade,
  added_at    timestamptz not null default now(),

  primary key (playlist_id, video_id)
);

-- +goose Down
drop table video_service.playlist_videos;
