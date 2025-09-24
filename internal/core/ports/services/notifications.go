package services

import (
	"context"
)

type NotificationsService interface {
	SendEmail(ctx context.Context, to string, subject string, body string, userId string) (price float32, err error)
	SendSMS(ctx context.Context, to string, message string) (price float32, err error)
}
