-- +goose Up
-- +goose StatementBegin
CREATE TABLE sessions (
    accessToken TEXT NOT NULL,
    refreshToken TEXT NOT NULL,
    valid INTEGER DEFAULT 1,
    UNIQUE(accessToken, refreshToken),
    PRIMARY KEY(accessToken, refreshToken)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE sessions;
-- +goose StatementEnd
