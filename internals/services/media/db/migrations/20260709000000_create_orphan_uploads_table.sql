-- +goose Up
create table media_service.orphan_uploads (
  object_key varchar    primary key,
  user_id    uuid       not null,
  created_at timestamptz not null default now()
);

-- +goose Down
drop table media_service.orphan_uploads;