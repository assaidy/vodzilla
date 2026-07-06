-- +goose Up
create table reaction_service.comments (
  id         uuid         primary key,
  for_id     uuid         not null,
  user_id    uuid         not null,
  content    varchar(500) not null,
  created_at timestamptz  not null default now()
);

-- +goose Down
drop table reaction_service.comments;
