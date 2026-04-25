-- +goose Up
create schema user_service;

create table user_service.users (
  id            varchar                   primary key,
  email         varchar(256)              not null unique,
  password_hash varchar(256)              not null,
  name          varchar(256)              not null,
  username      varchar(32)               not null unique,
  created_at    timestamptz default now() not null,
  is_verified   boolean     default false not null
);

create table user_service.sessions (
  id            varchar                   primary key,
  owner_id      varchar                   not null references user_service.users (id) on delete cascade,
  session_token varchar(256)              not null unique, 
  csrf_token    varchar(256)              not null,
  created_at    timestamptz default now() not null,
  expires_at    timestamptz               not null
);

create table user_service.email_verification_tokens (
  id         varchar                   primary key,
  owner_id   varchar                   not null references user_service.users (id) on delete cascade,
  token      varchar(256)              not null unique,
  created_at timestamptz default now() not null,
  expires_at timestamptz               not null
);

-- +goose Down
drop schema user_service cascade;
