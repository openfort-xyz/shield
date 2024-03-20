package sharerepo

import (
	"gorm.io/gorm"
	"time"
)

type Share struct {
	ID        string         `gorm:"column:id;primary_key"`
	Data      string         `gorm:"column:data; not null"`
	UserID    string         `gorm:"column:user_id;not null"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (Share) TableName() string {
	return "shld_shares"
}
