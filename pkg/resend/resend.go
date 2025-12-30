package resend

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"time"

	env "github.com/caarlos0/env/v10"
	resendlib "github.com/resend/resend-go/v3"
)

//go:embed otp_template.html
var otpTemplateFS embed.FS

type Config struct {
	FromEmail    string `env:"RESEND_FROM_EMAIL"`
	SenderName   string `env:"RESEND_EMAIL_SENDER_NAME"`
	ResendAPIKey string `env:"RESEND_API_KEY"`
}

type Client struct {
	config       Config
	apiClient    *resendlib.Client
	htmlTemplate *template.Template
}

func GetConfigFromEnv() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func NewClient(config Config) (*Client, error) {
	if config.ResendAPIKey == "" {
		return nil, fmt.Errorf("RESEND_API_KEY is required")
	}

	if config.FromEmail == "" {
		return nil, fmt.Errorf("RESEND_FROM_EMAIL is required")
	}

	if config.SenderName == "" {
		return nil, fmt.Errorf("RESEND_EMAIL_SENDER_NAME is required")
	}

	apiClient := resendlib.NewClient(config.ResendAPIKey)

	// Parse template once at initialization
	tmplContent, err := otpTemplateFS.ReadFile("otp_template.html")
	if err != nil {
		return nil, fmt.Errorf("failed to read OTP template: %v", err)
	}

	htmlTemplate, err := template.New("otp").Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse OTP template: %v", err)
	}

	return &Client{
		config:       config,
		apiClient:    apiClient,
		htmlTemplate: htmlTemplate,
	}, nil
}

type OTPData struct {
	OTP string
}

// SendEmail sends an OTP email to the specified recipient.
// Note: ctx is accepted for interface compatibility but not used by the Resend SDK.
func (c *Client) SendEmail(ctx context.Context, toEmail string, subject string, otp string, userId string) error {
	// Execute the pre-parsed template with OTP data
	var htmlBuffer bytes.Buffer
	if err := c.htmlTemplate.Execute(&htmlBuffer, OTPData{OTP: otp}); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	htmlContent := htmlBuffer.String()
	if htmlContent == "" {
		return fmt.Errorf("HTML template generated empty content")
	}

	// Create fallback text content
	textContent := fmt.Sprintf("Your verification code is: %s\n\nThis code is valid for 5 minutes only. Do not share this code with anyone.", otp)

	// Combine sender name and email in Resend's expected format: "Name <email>"
	from := fmt.Sprintf("%s <%s>", c.config.SenderName, c.config.FromEmail)

	// Format recipient with userId as display name if provided
	var to string
	if userId != "" {
		to = fmt.Sprintf("%s <%s>", userId, toEmail)
	} else {
		to = toEmail
	}

	params := &resendlib.SendEmailRequest{
		From:    from,
		To:      []string{to},
		Subject: fmt.Sprintf("%v - %s", subject, time.Now().Format("Jan 02, 15:04")),
		Html:    htmlContent,
		Text:    textContent,
	}

	_, err := c.apiClient.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}
