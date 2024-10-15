-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
ADD COLUMN twitter_request_token TEXT;

ALTER TABLE users
ADD COLUMN twitter_request_secret TEXT;

ALTER TABLE users
ADD COLUMN twitter_oauth_token TEXT;

ALTER TABLE users
ADD COLUMN twitter_oauth_secret TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
DROP COLUMN twitter_request_token;

ALTER TABLE users
DROP COLUMN twitter_request_secret;

ALTER TABLE users
DROP COLUMN twitter_oauth_token;

ALTER TABLE users
DROP COLUMN twitter_oauth_secret;
-- +goose StatementEnd
