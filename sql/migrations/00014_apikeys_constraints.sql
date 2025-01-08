-- +goose Up
-- +goose StatementBegin
CREATE TABLE apikeys_new (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    uid INTEGER,
    CONSTRAINT fk_users
    FOREIGN KEY(uid)
    REFERENCES users(uid)
);

INSERT INTO apikeys_new(key, name, uid)
SELECT key, name, uid
FROM apikeys;

DROP TABLE apikeys;

ALTER TABLE apikeys_new
RENAME TO apikeys;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE apikeys_new (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    uid INTEGER
);

INSERT INTO apikeys_new(key, name, uid)
SELECT key, name, uid
FROM apikeys;

DROP TABLE apikeys;

ALTER TABLE apikeys_new
RENAME TO apikeys;
-- +goose StatementEnd
