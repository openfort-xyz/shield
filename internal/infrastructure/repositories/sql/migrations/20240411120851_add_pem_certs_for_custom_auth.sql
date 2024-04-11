-- +goose Up
ALTER TABLE shld_custom_providers ADD COLUMN pem_cert text DEFAULT NULL;
ALTER TABLE shld_custom_providers ADD COLUMN key_type ENUM('RSA', 'ECDSA', 'ED25519') DEFAULT NULL;
ALTER TABLE shld_custom_providers MODIFY COLUMN jwk_url VARCHAR(255);

-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
ALTER TABLE shld_custom_providers DROP COLUMN pem_cert;
ALTER TABLE shld_custom_providers DROP COLUMN key_type;
ALTER TABLE shld_custom_providers MODIFY COLUMN jwk_url VARCHAR(255) NOT NULL;
-- +goose StatementBegin
-- +goose StatementEnd
