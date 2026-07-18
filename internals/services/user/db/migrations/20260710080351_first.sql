-- +goose Up
create schema user_service;

create table user_service.users (
  id            uuid          primary key,
  email         varchar(256)  not null unique,
  password_hash varchar(256)  not null,
  name          varchar(256)  not null,
  username      varchar(32)   not null unique,
  bio           varchar(500), 
  created_at    timestamptz   not null default now(),
  is_email_verified   bool          not null default false
);

create table user_service.retired_usernames (
  username   varchar(32)  primary key,
  user_id    uuid         not null,
  retired_at timestamptz  not null default now()
);

create table user_service.email_verification_tokens (
  id         uuid         primary key,
  user_id   uuid         not null references user_service.users (id) on delete cascade,
  token      varchar(256) not null unique,
  created_at timestamptz  not null default now(),
  expires_at timestamptz  not null
);

create table user_service.sessions (
  id            uuid         primary key,
  user_id      uuid         not null references user_service.users (id) on delete cascade,
  session_token varchar(256) not null unique, 
  csrf_token    varchar(256) not null,
  created_at    timestamptz  not null default now(),
  expires_at    timestamptz  not null
);

-- +goose Down
drop schema user_service cascade;
