package notificationsrepo

import (
	"time"
)

type Notification struct {
	ID        int       `gorm:"column:id;primaryKey"`
	ProjectID string    `gorm:"column:project_id"`
	NotifType string    `gorm:"column:notif_type"`
	Price     float32   `gorm:"column:price"`
	SentAt    time.Time `gorm:"column:sent_at;autoCreateTime"`
}

func (Notification) TableName() string {
	return "shld_notifications"
}
