-- +goose Up
create table media_service.uploads (
  id         varchar     not null,
  object_key varchar     not null references media_service.object_keys (object_key) on delete cascade,
  expires_at timestamptz not null,

  primary key (id, object_key)
);

-- +goose Down
drop table media_service.uploads;
