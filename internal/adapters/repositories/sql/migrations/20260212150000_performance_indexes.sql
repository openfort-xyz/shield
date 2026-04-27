-- +goose Up
CREATE UNIQUE INDEX idx_projects_api_key ON shld_projects(api_key);
CREATE INDEX idx_providers_project_type ON shld_providers(project_id, type);
-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
DROP INDEX IF EXISTS idx_projects_api_key;
DROP INDEX IF EXISTS idx_providers_project_type;
-- +goose StatementBegin
-- +goose StatementEnd
