-- +goose Up
CREATE TABLE IF NOT EXISTS shld_shamir_migrations (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    success BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX idx_project_id_success ON shld_shamir_migrations(project_id, success);
DROP TABLE IF EXISTS shld_allowed_origins;
-- +goose StatementBegin
-- +d

-- +goose Down
DROP INDEX idx_project_id_success ON shld_shamir_migrations;
DROP TABLE IF EXISTS shld_shamir_migrations;
CREATE TABLE IF NOT EXISTS shld_allowed_origins (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) NOT NULL,
    origin VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP DEFAULT NULL
);
ALTER TABLE shld_allowed_origins ADD CONSTRAINT fk_origin_project FOREIGN KEY (project_id) REFERENCES shld_projects(id) ON DELETE CASCADE;
-- +goose StatementBegin
-- +goose StatementEnd
