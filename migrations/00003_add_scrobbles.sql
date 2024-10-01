-- +goose Up
-- +goose StatementBegin
CREATE TABLE scrobbles (
    id INTEGER PRIMARY KEY,
    artist_name TEXT NOT NULL,
    track_name TEXT NOT NULL,
    album_name TEXT,
    album_artist TEXT,
    track_number TEXT,
    duration INTEGER NOT NULL,
    timestamp INTEGER NOT NULL,
    source TEXT,
    mbid TEXT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE scrobbles;
-- +goose StatementEnd
