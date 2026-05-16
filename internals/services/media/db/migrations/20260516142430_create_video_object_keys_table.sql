-- +goose Up
create table media_service.object_keys (
  video_id   varchar primary key,      
  object_key varchar not null unique
);

-- +goose Down
drop table media_service.object_keys;
