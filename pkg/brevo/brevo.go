package brevo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	env "github.com/caarlos0/env/v10"
)

const (
	BREVO_API_URL     = "https://api.brevo.com/v3/smtp/email"
	FROM_EMAIL        = "shield@openfort.xyz"
	EMAIL_SENDER_NAME = "Shield"
)

type Config struct {
	BrevoAPIKey string `env:"BREVO_API_KEY"`
}

type Client struct {
	config Config
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

	return &Client{
		config: config,
	}, nil
}

type Sender struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Recipient struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type EmailRequest struct {
	Sender      Sender      `json:"sender"`
	To          []Recipient `json:"to"`
	Subject     string      `json:"subject"`
	TextContent string      `json:"textContent"`
	// For future reference
	// HTMLContent string      `json:"htmlContent"`
}

func (c *Client) SendEmail(ctx context.Context, toEmail string, subject string, body string, userId string) error {
	const ()

	emailReq := EmailRequest{
		Sender: Sender{
			Name:  EMAIL_SENDER_NAME,
			Email: FROM_EMAIL,
		},
		To: []Recipient{
			{
				Email: toEmail,
				Name:  userId,
			},
		},
		Subject:     subject,
		TextContent: body,
	}

	jsonData, err := json.Marshal(emailReq)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	req, err := http.NewRequest("POST", BREVO_API_URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("api-key", c.config.BrevoAPIKey)
	req.Header.Set("content-type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
