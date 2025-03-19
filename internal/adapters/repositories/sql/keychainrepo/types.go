package keychainrepo

import (
	"gorm.io/gorm"
)

type Keychain struct {
	ID        string         `gorm:"column:id;primary_key"`
	UserID    string         `gorm:"column:user_id;not null"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (Keychain) TableName() string {
	return "shld_keychains"
}
