-- +goose Up
-- +goose StatementBegin
CREATE TABLE history_spotify (
    id INTEGER PRIMARY KEY,
    artist_name TEXT NOT NULL,
    track_name TEXT NOT NULL,
    album_name TEXT,
    timestamp INTEGER NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE history_spotify;
-- +goose StatementEnd
