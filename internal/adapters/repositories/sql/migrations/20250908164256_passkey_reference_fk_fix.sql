-- +goose Up
ALTER TABLE shld_passkey_references
    DROP CONSTRAINT fk_share;

ALTER TABLE shld_passkey_references
    ADD CONSTRAINT fk_share
        FOREIGN KEY (share_reference)
        REFERENCES shld_shares(id);

-- +goose Down
ALTER TABLE shld_passkey_references
    DROP CONSTRAINT fk_share;

ALTER TABLE shld_passkey_references
    ADD CONSTRAINT fk_share
        FOREIGN KEY (share_reference)
        REFERENCES shld_shares(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE;
