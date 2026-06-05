-- +goose Up
create table media_service.uploads (
  id           varchar     not null,
  video_id     uuid        not null references media_service.videos (id) on delete cascade,
  expires_at   timestamptz not null,
  completed_at timestamptz,

  primary key (video_id)
);

-- +goose Down
drop table media_service.uploads;
