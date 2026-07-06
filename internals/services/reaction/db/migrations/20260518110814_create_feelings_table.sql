-- +goose Up
create table reaction_service.feelings (
  for_id   uuid        not null,
  user_id  uuid        not null,
  kind     varchar     not null,
  added_at timestamptz not null default now(),

  primary key(for_id, user_id)
);

-- +goose Down
drop table reaction_service.feelings;
