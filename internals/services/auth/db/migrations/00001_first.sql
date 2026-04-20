-- +goose Up
create schema auth;

-- +goose Down
drop schema auth cascade;
