-- +goose Up
create table user_service.users (
  id            varchar       primary key,
  email         varchar(256)  not null unique,
  password_hash varchar(256)  not null,
  name          varchar(256)  not null,
  username      varchar(32)   not null unique,
  bio           varchar(500),
  created_at    timestamptz   not null default now(),
  is_verified   boolean       not null default false 
);

-- +goose Down
drop table user_service.users;
