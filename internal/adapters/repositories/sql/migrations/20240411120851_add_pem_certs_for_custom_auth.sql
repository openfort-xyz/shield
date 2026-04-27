-- +goose Up
ALTER TABLE shld_custom_providers ADD COLUMN pem_cert text DEFAULT NULL;
ALTER TABLE shld_custom_providers ADD COLUMN key_type VARCHAR(16) DEFAULT NULL CHECK (key_type IN ('RSA', 'ECDSA', 'ED25519'));
ALTER TABLE shld_custom_providers ALTER COLUMN jwk_url DROP NOT NULL;

-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
ALTER TABLE shld_custom_providers DROP COLUMN pem_cert;
ALTER TABLE shld_custom_providers DROP COLUMN key_type;
ALTER TABLE shld_custom_providers ALTER COLUMN jwk_url SET NOT NULL;
-- +goose StatementBegin
-- +goose StatementEnd
