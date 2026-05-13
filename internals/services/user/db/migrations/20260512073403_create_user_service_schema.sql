-- +goose Up
create schema user_service;

-- +goose Down
drop schema user_service cascade;
