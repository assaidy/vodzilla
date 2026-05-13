-- +goose Up
create schema video_service;

-- +goose Down
drop schema video_service cascade;
