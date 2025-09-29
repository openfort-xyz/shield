-- +goose Up
CREATE TABLE IF NOT EXISTS shld_rate_limit (
    id INT AUTO_INCREMENT PRIMARY KEY,
    project_id VARCHAR(36) NOT NULL,
    requests_per_minute INT NOT NULL
);

ALTER TABLE shld_rate_limit ADD CONSTRAINT fk_shdl_rate_limit_project FOREIGN KEY (project_id) REFERENCES shld_projects(id) ON DELETE CASCADE;

INSERT INTO shld_rate_limit (project_id, requests_per_minute)
SELECT id, 50 FROM shld_projects WHERE deleted_at IS NULL;

-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS shld_rate_limit;

-- +goose StatementBegin
-- +goose StatementEnd
