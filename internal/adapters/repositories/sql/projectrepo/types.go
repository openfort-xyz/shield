package projectrepo

import (
	"time"

	"gorm.io/gorm"
)

type Project struct {
	ID        string         `gorm:"column:id;primaryKey"`
	Name      string         `gorm:"column:name"`
	APIKey    string         `gorm:"column:api_key"`
	APISecret string         `gorm:"column:api_secret"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at"`
	Enable2FA bool           `gorm:"column:enable_2fa"`
}

func (Project) TableName() string {
	return "shld_projects"
}

type EncryptionPart struct {
	ID        string `gorm:"column:id;primaryKey"`
	ProjectID string `gorm:"column:project_id"`
	Part      string `gorm:"column:part"`
}

func (EncryptionPart) TableName() string {
	return "shld_encryption_parts"
}

type Migration struct {
	ID        string    `gorm:"column:id;primaryKey"`
	ProjectID string    `gorm:"column:project_id"`
	Timestamp time.Time `gorm:"column:timestamp;autoCreateTime"`
	Success   bool      `gorm:"column:success"`
}

func (Migration) TableName() string {
	return "shld_shamir_migrations"
}
