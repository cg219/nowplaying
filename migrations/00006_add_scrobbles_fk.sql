-- +goose Up
-- +goose StatementBegin
CREATE TABLE scrobbles_new (
    id INTEGER PRIMARY KEY,
    artist_name TEXT NOT NULL,
    track_name TEXT NOT NULL,
    album_name TEXT,
    album_artist TEXT,
    track_number TEXT,
    duration INTEGER NOT NULL,
    timestamp INTEGER NOT NULL,
    source TEXT,
    mbid TEXT,
    uid INTEGER NOT NULL DEFAULT 1,
    CONSTRAINT fk_users
    FOREIGN KEY(uid)
    REFERENCES users(id)
);

INSERT INTO scrobbles_new(id, artist_name, track_name, album_name, album_artist, track_number, duration, timestamp, source, mbid, uid)
SELECT id, artist_name, track_name, album_name, album_artist, track_number, duration, timestamp, source, mbid
FROM scrobbles;

DROP TABLE scrobbles;

ALTER TABLE scrobbles_new
RENAME TO scrobbles;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE scrobbles_new (
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

INSERT INTO scrobbles_new(id, artist_name, track_name, album_name, album_artist, track_number, duration, timestamp, source, mbid)
SELECT id, artist_name, track_name, album_name, album_artist, track_number, duration, timestamp, source, mbid
FROM scrobbles;

DROP TABLE scrobbles

ALTER TABLE scrobbles_new
RENAME scrobbles;
-- +goose StatementEnd
