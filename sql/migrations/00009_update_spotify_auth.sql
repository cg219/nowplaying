-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
ADD COLUMN spotify_auth_state TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
DROP COLUMN spotify_auth_state;
-- +goose StatementEnd
