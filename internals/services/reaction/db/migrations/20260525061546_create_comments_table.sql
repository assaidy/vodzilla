-- +goose Up
create table reaction_service.comments (
  id         varchar      primary key,
  video_id   varchar      not null,
  owner_id   varchar      not null,
  content    varchar(500) not null,
  created_at timestamptz  not null default now(),
  parent_id  varchar      references reaction_service.comments (id) on delete cascade
);

create index on reaction_service.comments (video_id);
create index on reaction_service.comments (parent_id) where parent_id is not null;
create index on reaction_service.comments (id, owner_id);

-- +goose Down
drop table reaction_service.comments;
