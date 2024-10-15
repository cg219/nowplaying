-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
DROP COLUMN lastfm_token;

ALTER TABLE users
ADD COLUMN lastfm_session_name TEXT;

ALTER TABLE users
ADD COLUMN lastfm_session_key TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
ADD COLUMN lastfm_token;

ALTER TABLE users
DROP COLUMN lastfm_session_name;

ALTER TABLE users
DROP COLUMN lastfm_session_key;
-- +goose StatementEnd
