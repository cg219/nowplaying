-- +goose Up
-- +goose StatementBegin
CREATE TABLE music_sessions (
    id INTEGER PRIMARY KEY,
    data TEXT NOT NULL,
    active INTEGER NOT NULL,
    type TEXT NOT NULL,
    uid INTEGER NOT NULL,
    CONSTRAINT fk_music_sessios_uid
    FOREIGN KEY (uid)
    REFERENCES users(id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE music_sessions;
-- +goose StatementEnd
