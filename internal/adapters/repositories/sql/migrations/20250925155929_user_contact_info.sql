-- +goose Up

CREATE TABLE IF NOT EXISTS shld_user_contacts (
    id INT AUTO_INCREMENT PRIMARY KEY,
    external_user_id VARCHAR(255) NOT NULL,
    email CHAR(128),
    phone CHAR(128)
);

ALTER TABLE shld_user_contacts ADD CONSTRAINT uk_external_user_id UNIQUE (external_user_id);

-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down

DROP TABLE IF EXISTS shld_user_contacts;

-- +goose StatementBegin
-- +goose StatementEnd
