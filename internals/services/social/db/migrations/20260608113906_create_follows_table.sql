-- +goose Up
create table social_service.follows (
  follower_id uuid        not null,
  followed_id uuid        not null,
  created_at  timestamptz not null default now(),

  primary key (follower_id, followed_id)
);

-- +goose Down
drop table social_service.follows;
