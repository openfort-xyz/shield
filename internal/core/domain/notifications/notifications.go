package notifications

import "time"

const (
	EmailNotificationType = "Email"
	SMSNotificationType   = "SMS"
)

type Notification struct {
	ID             int
	ProjectID      string
	ExternalUserID string
	NotifType      string
	Price          float32
	SentAt         time.Time
}
