-- +goose Up
alter table user_service.users add column is_deleted bool not null default false;

-- +goose Down
alter table user_service.users drop column is_deleted;
