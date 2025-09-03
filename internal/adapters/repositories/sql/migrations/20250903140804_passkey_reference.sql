-- +goose Up
CREATE TABLE shld_passkey_references (
    passkey_id      VARCHAR(255) NOT NULL,
    passkey_env     VARCHAR(255) NOT NULL,
    share_reference VARCHAR(255) NOT NULL,
    PRIMARY KEY (share_reference),
    CONSTRAINT fk_share
        FOREIGN KEY (share_reference)
        REFERENCES shld_shares(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS shld_passkey_references;
