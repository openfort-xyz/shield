-- +goose Up
CREATE TABLE IF NOT EXISTS shld_notifications (
    id INT AUTO_INCREMENT PRIMARY KEY,
    project_id VARCHAR(36) NOT NULL,
    external_user_id VARCHAR(255) NOT NULL,
    notif_type ENUM('SMS', 'Email') NOT NULL,
    price FLOAT NOT NULL,
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

ALTER TABLE shld_notifications ADD CONSTRAINT fk_notifications_project FOREIGN KEY (project_id) REFERENCES shld_projects(id) ON DELETE CASCADE;

-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS shld_notifications;

-- +goose StatementBegin
-- +goose StatementEnd
