-- +goose Up
CREATE TABLE shld_share_storage_methods (
    id   INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

INSERT INTO shld_share_storage_methods (id, name) VALUES
    (0, 'Shield'),
    (1, 'Google Drive'),
    (2, 'iCloud');



-- +goose Down
DROP TABLE IF EXISTS shld_share_storage_methods;