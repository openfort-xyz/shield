package brevo

import (
	"context"
	"fmt"

	env "github.com/caarlos0/env/v10"
	brevo "github.com/getbrevo/brevo-go/lib"
)

const (
	FROM_EMAIL        = "shield@openfort.xyz"
	EMAIL_SENDER_NAME = "Shield"
)

type Config struct {
	BrevoAPIKey string `env:"BREVO_API_KEY" envDefault:""`
}

type Client struct {
	config      Config
	brevoClient *brevo.APIClient
}

func GetConfigFromEnv() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func NewClient(config Config) (*Client, error) {
	if config.BrevoAPIKey == "" {
		return nil, fmt.Errorf("BREVO_API_KEY is required")
	}

	cfg := brevo.NewConfiguration()
	cfg.AddDefaultHeader("api-key", config.BrevoAPIKey)
	cfg.AddDefaultHeader("partner-key", config.BrevoAPIKey)

	br := brevo.NewAPIClient(cfg)

	return &Client{
		config:      config,
		brevoClient: br,
	}, nil
}

func (c *Client) SendEmail(ctx context.Context, toEmail string, subject string, body string, userId string) error {
	sender := brevo.SendSmtpEmailSender{
		Name:  EMAIL_SENDER_NAME,
		Email: FROM_EMAIL,
	}
	to := brevo.SendSmtpEmailTo{
		Email: toEmail,
		Name:  userId,
	}
	email := brevo.SendSmtpEmail{
		Sender:      &sender,
		To:          []brevo.SendSmtpEmailTo{to},
		Subject:     subject,
		TextContent: body,
	}

	_, _, err := c.brevoClient.TransactionalEmailsApi.SendTransacEmail(ctx, email)
	if err != nil {
		return err
	}

	return nil
}
