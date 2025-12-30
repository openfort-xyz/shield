package notificationsapp

import (
	"context"

	"go.openfort.xyz/shield/pkg/resend"
	"go.openfort.xyz/shield/pkg/smsapi"
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

	smsApiConfig, err := smsapi.GetConfigFromEnv()
	if err != nil || smsApiConfig.SMSAPIKey == "" {
		return nil, err
	}

	resendClient, err := resend.NewClient(*resendConfig)
	if err != nil {
		return nil, err
	}

	smsApiClient, err := smsapi.NewClient(*smsApiConfig)
	if err != nil {
		return nil, err
	}

	return &NotificationApplication{
		emailProvider: *resendClient,
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
