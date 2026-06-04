-- +goose Up
create table reaction_service.comments (
  id         uuid         primary key,
  video_id   uuid         not null,
  owner_id   uuid         not null,
  content    varchar(500) not null,
  created_at timestamptz  not null default now(),
  parent_id  uuid         references reaction_service.comments (id) on delete cascade
);

create index on reaction_service.comments (video_id);
create index on reaction_service.comments (parent_id) where parent_id is not null;
create index on reaction_service.comments (id, owner_id);

-- +goose Down
drop table reaction_service.comments;
