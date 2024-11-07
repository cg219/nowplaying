-- +goose Up
-- +goose StatementBegin
CREATE TABLE scrobbles_temp (
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
    uid INTEGER NOT NULL DEFAULT 1
);

INSERT INTO scrobbles_temp(id, artist_name, track_name, album_name, album_artist, track_number, duration, timestamp, source, mbid, uid)
SELECT id, artist_name, track_name, album_name, album_artist, track_number, duration, timestamp, source, mbid, uid
FROM scrobbles;

DROP TABLE scrobbles;

CREATE TABLE music_sessions_temp (
    id INTEGER PRIMARY KEY,
    data TEXT NOT NULL,
    active INTEGER NOT NULL,
    type TEXT NOT NULL,
    uid INTEGER NOT NULL
);

INSERT INTO music_sessions_temp(id, data, active, type, uid)
SELECT id, data, active, type, uid
FROM music_sessions;

DROP TABLE music_sessions;

CREATE TABLE users_new (
    id INTEGER PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    spotify_access_token TEXT,
    spotify_refresh_token TEXT,
    spotify_id TEXT UNIQUE,
    spotify_auth_state TEXT,
    lastfm_session_name TEXT,
    lastfm_session_key TEXT,
    password TEXT NOT NULL DEFAULT "___",
    twitter_request_token TEXT,
    twitter_request_secret TEXT,
    twitter_oauth_token TEXT,
    twitter_oauth_secret TEXT
);

INSERT INTO users_new(id, username, spotify_access_token, spotify_refresh_token, spotify_auth_state, lastfm_session_name, lastfm_session_key, password, twitter_request_token, twitter_request_secret, twitter_oauth_token, twitter_oauth_secret)
SELECT id, username, spotify_access_token, spotify_refresh_token, spotify_auth_state, lastfm_session_name, lastfm_session_key, password, twitter_request_token, twitter_request_secret, twitter_oauth_token, twitter_oauth_secret
FROM users;

DROP TABLE users;

ALTER TABLE users_new
RENAME TO users;

CREATE TABLE scrobbles_new(
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
SELECT id, artist_name, track_name, album_name, album_artist, track_number, duration, timestamp, source, mbid, uid
FROM scrobbles_temp;

ALTER TABLE scrobbles_new
RENAME TO scrobbles;

DROP TABLE scrobbles_temp;

CREATE TABLE music_sessions_new (
    id INTEGER PRIMARY KEY,
    data TEXT NOT NULL,
    active INTEGER NOT NULL,
    type TEXT NOT NULL,
    uid INTEGER NOT NULL,
    CONSTRAINT fk_music_sessios_uid
    FOREIGN KEY (uid)
    REFERENCES users(id)
);

INSERT INTO music_sessions_new(id, data, active, type, uid)
SELECT id, data, active, type, uid
FROM music_sessions_temp;

ALTER TABLE music_sessions_new
RENAME TO music_sessions;

DROP TABLE music_sessions_temp;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
DROP COLUMN spotify_id;
-- +goose StatementEnd
