-- +goose Up
create table user_service.email_verification_tokens (
  id         varchar      primary key,
  owner_id   varchar      not null references user_service.users (id) on delete cascade,
  token      varchar(256) not null unique,
  created_at timestamptz  not null default now(),
  expires_at timestamptz  not null
);

-- +goose Down
drop table user_service.email_verification_tokens;
