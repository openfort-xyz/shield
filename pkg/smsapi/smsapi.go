package smsapi

import (
	"context"
	"fmt"

	"github.com/caarlos0/env/v10"
	"github.com/smsapi/smsapi-go/smsapi"
)

type Config struct {
	SMSAPIKey string `env:"SMS_API_KEY" envDefault:""`
}

type Client struct {
	config    Config
	apiClient *smsapi.Client
}

func GetConfigFromEnv() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func NewClient(config Config) (*Client, error) {
	if config.SMSAPIKey == "" {
		return nil, fmt.Errorf("SMSAPIKey is required")
	}

	client := smsapi.NewInternationalClient(config.SMSAPIKey, nil)

	return &Client{
		config:    config,
		apiClient: client,
	}, nil
}

func (c *Client) SendSMS(ctx context.Context, to string, message string) error {
	_, err := c.apiClient.Sms.Send(context.Background(), to, message, "")

	return err
}
