package notificationsapp

import (
	"context"

	"go.openfort.xyz/shield/pkg/brevo"
)

type NotificationApplication struct {
	emailProvider brevo.Client
	smsProvider   brevo.Client
}

func NewNotificationApp() (*NotificationApplication, error) {
	brevoConfig, err := brevo.GetConfigFromEnv()
	if err != nil {
		return nil, err
	}

	brevoClient, err := brevo.NewClient(*brevoConfig)
	if err != nil {
		return nil, err
	}

	return &NotificationApplication{
		emailProvider: *brevoClient,
		smsProvider:   *brevoClient,
	}, nil
}

func (c *NotificationApplication) SendEmail(ctx context.Context, toEmail string, subject string, body string) error {
	err := c.emailProvider.SendEmail(ctx, toEmail, subject, body)

	return err
}

func (c *NotificationApplication) SendSMS(ctx context.Context, to string, message string) error {
	err := c.smsProvider.SendSMS(ctx, to, message)

	return err
}
