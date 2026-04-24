package notificationsapp

import (
	"context"

	"github.com/openfort-xyz/shield/pkg/resend"
	"github.com/openfort-xyz/shield/pkg/smsapi"
)

type NotificationApplication struct {
	emailProvider resend.Client
	smsProvider   smsapi.Client
}

func NewNotificationApp() (*NotificationApplication, error) {
	resendConfig, err := resend.GetConfigFromEnv()
	if err != nil || resendConfig.ResendAPIKey == "" {
		return nil, err
	}

	smsAPIConfig, err := smsapi.GetConfigFromEnv()
	if err != nil || smsAPIConfig.SMSAPIKey == "" {
		return nil, err
	}

	resendClient, err := resend.NewClient(*resendConfig)
	if err != nil {
		return nil, err
	}

	smsAPIClient, err := smsapi.NewClient(*smsAPIConfig)
	if err != nil {
		return nil, err
	}

	return &NotificationApplication{
		emailProvider: *resendClient,
		smsProvider:   *smsAPIClient,
	}, nil
}

func (c *NotificationApplication) SendEmail(ctx context.Context, toEmail string, subject string, body string, userID string) (price float32, err error) {
	// do not track prices per email yet because there is subscription based payments
	err = c.emailProvider.SendEmail(ctx, toEmail, subject, body, userID)

	return price, err
}

func (c *NotificationApplication) SendSMS(ctx context.Context, to string, message string) (price float32, err error) {
	price, err = c.smsProvider.SendSMS(ctx, to, message)

	return price, err
}
