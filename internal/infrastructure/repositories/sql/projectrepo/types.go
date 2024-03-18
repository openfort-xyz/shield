package projectrepo

import (
	"gorm.io/gorm"
	"time"
)

type Project struct {
	ID        string         `gorm:"column:id;primaryKey"`
	Name      string         `gorm:"column:name"`
	APIKey    string         `gorm:"column:api_key"`
	APISecret string         `gorm:"column:api_secret"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (Project) TableName() string {
	return "shld_projects"
}
