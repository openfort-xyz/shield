package brevo

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	env "github.com/caarlos0/env/v10"
)

const (
	BREVO_API_URL = "https://api.brevo.com/v3/smtp/email"
)

//go:embed otp_template.html
var otpTemplateFS embed.FS

type Config struct {
	FromEmail   string `env:"BREVO_FROM_EMAIL"`
	SenderName  string `env:"BREVO_EMAIL_SENDER_NAME"`
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

	if config.FromEmail == "" {
		return nil, fmt.Errorf("BREVO_FROM_EMAIL is required")
	}

	if config.SenderName == "" {
		return nil, fmt.Errorf("BREVO_EMAIL_SENDER_NAME is required")
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
	TextContent string      `json:"textContent,omitempty"`
	HTMLContent string      `json:"htmlContent,omitempty"`
}

type OTPData struct {
	OTP string
}

func (c *Client) SendEmail(ctx context.Context, toEmail string, subject string, otp string, userId string) error {
	// Load and parse the HTML template
	tmplContent, err := otpTemplateFS.ReadFile("otp_template.html")
	if err != nil {
		return fmt.Errorf("failed to read OTP template: %v", err)
	}

	tmpl, err := template.New("otp").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse OTP template: %v", err)
	}

	// Execute the template with OTP data
	var htmlBuffer bytes.Buffer
	if err := tmpl.Execute(&htmlBuffer, OTPData{OTP: otp}); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	// Debug: check if HTML content was generated
	htmlContent := htmlBuffer.String()
	if htmlContent == "" {
		return fmt.Errorf("HTML template generated empty content")
	}

	// Create fallback text content
	textContent := fmt.Sprintf("Your verification code is: %s\n\nThis code is valid for 5 minutes only. Do not share this code with anyone.", otp)

	emailReq := EmailRequest{
		Sender: Sender{
			Name:  c.config.SenderName,
			Email: c.config.FromEmail,
		},
		To: []Recipient{
			{
				Email: toEmail,
				Name:  userId,
			},
		},
		Subject:     subject,
		TextContent: textContent,
		HTMLContent: htmlContent,
	}

	return c.sendEmailRequest(emailReq)
}

func (c *Client) sendEmailRequest(emailReq EmailRequest) error {
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
