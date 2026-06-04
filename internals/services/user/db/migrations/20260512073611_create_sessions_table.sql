-- +goose Up
create table user_service.sessions (
  id            uuid         primary key,
  owner_id      uuid         not null references user_service.users (id) on delete cascade,
  session_token varchar(256) not null unique, 
  csrf_token    varchar(256) not null,
  created_at    timestamptz  not null default now(),
  expires_at    timestamptz  not null
);

-- +goose Down
drop table user_service.sessions;
