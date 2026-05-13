-- +goose Up
create schema media_service;

-- +goose Down
drop schema media_service;
