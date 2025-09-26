package notificationsapp

import (
	"context"

	"go.openfort.xyz/shield/pkg/brevo"
	"go.openfort.xyz/shield/pkg/smsapi"
)

type NotificationApplication struct {
	emailProvider brevo.Client
	smsProvider   smsapi.Client
}

func NewNotificationApp() (*NotificationApplication, error) {
	brevoConfig, err := brevo.GetConfigFromEnv()
	if err != nil || brevoConfig.BrevoAPIKey == "" {
		return nil, err
	}

	smsApiConfig, err := smsapi.GetConfigFromEnv()
	if err != nil || smsApiConfig.SMSAPIKey == "" {
		return nil, err
	}

	brevoClient, err := brevo.NewClient(*brevoConfig)
	if err != nil {
		return nil, err
	}

	smsApiClient, err := smsapi.NewClient(*smsApiConfig)
	if err != nil {
		return nil, err
	}

	return &NotificationApplication{
		emailProvider: *brevoClient,
		smsProvider:   *smsApiClient,
	}, nil
}

func (c *NotificationApplication) SendEmail(ctx context.Context, toEmail string, subject string, body string, userId string) (price float32, err error) {
	// do not track prices per email yet because there is subscription based payments
	err = c.emailProvider.SendEmail(ctx, toEmail, subject, body, userId)

	return price, err
}

func (c *NotificationApplication) SendSMS(ctx context.Context, to string, message string) (price float32, err error) {
	price, err = c.smsProvider.SendSMS(ctx, to, message)

	return price, err
}
