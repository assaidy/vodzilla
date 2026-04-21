-- +goose Up
create schema auth;

create table auth.users (
  id            uuid                      primary key,
  email         varchar(256)              not null unique,
  password_hash varchar(256)              not null,
  created_at    timestamptz default now() not null,
  is_verified   boolean     default false not null
);

create table auth.sessions (
  id            uuid                      primary key,
  owner_id      uuid                      not null references auth.users (id) on delete cascade,
  session_token varchar(256)              not null unique, 
  csrf_token    varchar(256)              not null,
  created_at    timestamptz default now() not null,
  expires_at    timestamptz               not null
);

create index session_owner_id_index on auth.sessions (owner_id);


create table auth.email_verification_tokens (
  id         uuid                      primary key,
  owner_id   uuid                      not null references auth.users (id) on delete cascade,
  token      varchar(256)              not null unique,
  created_at timestamptz default now() not null,
  expires_at timestamptz               not null
);

-- +goose Down
drop schema auth cascade;
