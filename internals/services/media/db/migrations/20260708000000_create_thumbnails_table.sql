-- +goose Up
create table media_service.thumbnails (
    video_id   uuid        primary key,
    object_key varchar     not null,
    created_at timestamptz not null default now()
);

-- +goose Down
drop table media_service.thumbnails;
