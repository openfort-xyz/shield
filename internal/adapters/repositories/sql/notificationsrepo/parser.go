package notificationsrepo

import (
	"go.openfort.xyz/shield/internal/core/domain/notifications"
)

type parser struct {
}

func newParser() *parser {
	return &parser{}
}

func (p *parser) toDomain(notif *Notification) *notifications.Notification {
	return &notifications.Notification{
		ID:        notif.ID,
		ProjectID: notif.ProjectID,
		NotifType: notif.NotifType,
		Price:     notif.Price,
		SentAt:    notif.SentAt,
	}
}

func (p *parser) toDatabase(notif *notifications.Notification) *Notification {
	return &Notification{
		ID:        notif.ID,
		ProjectID: notif.ProjectID,
		NotifType: notif.NotifType,
		Price:     notif.Price,
		SentAt:    notif.SentAt,
	}
}
