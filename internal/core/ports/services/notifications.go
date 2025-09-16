package services

import (
	"context"
)

type NotificationsService interface {
	SendEmail(ctx context.Context, to string, subject string, body string, userId string) error
	SendSMS(ctx context.Context, to string, message string) error
}
