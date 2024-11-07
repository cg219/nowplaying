-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
ADD COLUMN reset TEXT;

ALTER TABLE users
ADD COLUMN reset_time INTEGER;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
DROP COLUMN reset;

ALTER TABLE users
DROP COLUMN reset_time;
-- +goose StatementEnd
