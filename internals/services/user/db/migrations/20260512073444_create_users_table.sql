-- +goose Up
create table user_service.users (
  id            uuid          primary key,
  email         varchar(256)  not null unique,
  password_hash varchar(256)  not null,
  name          varchar(256)  not null,
  username      varchar(32)   not null unique,
  bio           varchar(500), 
  created_at    timestamptz   not null default now(),
  is_verified   bool          not null default false
);

create table user_service.retired_usernames (
  username   varchar(32)  primary key,
  user_id    uuid         not null,
  retired_at timestamptz  not null default now()
);

-- +goose Down
drop table user_service.retired_usernames;
drop table user_service.users;
