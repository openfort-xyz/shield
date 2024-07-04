package userrepo

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        string         `gorm:"column:id;primaryKey"`
	ProjectID string         `gorm:"column:project_id;not null"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (User) TableName() string {
	return "shld_users"
}

type ExternalUser struct {
	ID             string         `gorm:"column:id;primaryKey"`
	UserID         string         `gorm:"column:user_id;not null"`
	ExternalUserID string         `gorm:"column:external_user_id;not null"`
	ProviderID     string         `gorm:"column:provider_id;not null"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (ExternalUser) TableName() string {
	return "shld_external_users"
}
