-- +goose Up
-- +goose StatementBegin
alter table user_service.users add column bio varchar(500);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table user_service.users drop column bio;
-- +goose StatementEnd
