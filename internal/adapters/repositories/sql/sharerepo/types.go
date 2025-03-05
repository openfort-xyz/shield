package sharerepo

import (
	"time"

	"gorm.io/gorm"
)

type Share struct {
	ID         string         `gorm:"column:id;primary_key"`
	Data       string         `gorm:"column:data; not null"`
	UserID     *string        `gorm:"column:user_id;default:null"`
	Entropy    Entropy        `gorm:"column:entropy;default:none"`
	Salt       string         `gorm:"column:salt;default:null"`
	Iterations int            `gorm:"column:iterations;default:null"`
	Length     int            `gorm:"column:length;default:null"`
	Digest     string         `gorm:"column:digest;default:null"`
	KeyChainID *string        `gorm:"column:keychain_id;default:null"`
	Reference  *string        `gorm:"column:reference;default:null"`
	CreatedAt  time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (Share) TableName() string {
	return "shld_shares"
}

type Entropy string

const (
	EntropyNone    Entropy = "none"
	EntropyUser    Entropy = "user"
	EntropyProject Entropy = "project"
)
