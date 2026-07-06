-- +goose Up
create table user_service.users (
  id            uuid          primary key,
  email         varchar(256)  not null unique,
  password_hash varchar(256)  not null,
  name          varchar(256)  not null,
  username      varchar(32)   not null unique,
  bio           varchar(500), 
  created_at    timestamptz   not null default now(),
  is_verified   bool          not null default false,
  is_deleted    bool          not null default false
);

-- +goose Down
drop table user_service.users;

-- FIX: remove is_deleted column and move the dirty usernames to a sparate table.
-- so we simplify user queries and still keep usernames consistent (used once).

