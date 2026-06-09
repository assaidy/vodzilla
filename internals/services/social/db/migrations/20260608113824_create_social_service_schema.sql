-- +goose Up
create schema social_service;

-- +goose Down
drop schema social_service;
