-- +goose Up
CREATE TABLE IF NOT EXISTS shld_encryption_parts (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) NOT NULL,
    part VARCHAR(255) NOT NULL
);
ALTER TABLE shld_encryption_parts ADD CONSTRAINT fk_part_project FOREIGN KEY (project_id) REFERENCES shld_projects(id) ON DELETE CASCADE;
-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS shld_encryption_parts;
-- +goose StatementBegin
-- +goose StatementEnd
