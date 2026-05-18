-- +goose Up
create schema reaction_service;

-- +goose Down
drop schema reaction_service;
