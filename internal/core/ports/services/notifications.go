package services

import (
	"context"
)

type NotificationsService interface {
	SendEmail(ctx context.Context, to string, subject string, body string) error
	SendSMS(ctx context.Context, to string, message string) error
}
