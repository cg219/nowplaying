-- +goose Up
-- +goose StatementBegin
CREATE TABLE apikeys (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    uid INTEGER
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE apikeys;
-- +goose StatementEnd
