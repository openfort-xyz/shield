package twilio

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Config holds Twilio configuration
type Config struct {
	AccountSID     string
	AuthToken      string
	FromPhone      string // Twilio phone number for SMS
	FromEmail      string // Sender email for SendGrid/Twilio Email API
	SendGridAPIKey string // Optional: for email via SendGrid
}

// Client represents a Twilio client
type Client struct {
	config     Config
	httpClient *http.Client
	baseURL    string
}

// SMSRequest represents an SMS sending request
type SMSRequest struct {
	To   string `json:"to"`
	From string `json:"from"`
	Body string `json:"body"`
}

// EmailRequest represents an email sending request
type EmailRequest struct {
	To      string `json:"to"`
	From    string `json:"from"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// SMSResponse represents Twilio SMS API response
type SMSResponse struct {
	SID          string `json:"sid"`
	Status       string `json:"status"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// EmailResponse represents email sending response
type EmailResponse struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

// SendGridEmailRequest represents SendGrid API email request
type SendGridEmailRequest struct {
	Personalizations []SendGridPersonalization `json:"personalizations"`
	From             SendGridEmail             `json:"from"`
	Subject          string                    `json:"subject"`
	Content          []SendGridContent         `json:"content"`
}

type SendGridPersonalization struct {
	To []SendGridEmail `json:"to"`
}

type SendGridEmail struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type SendGridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// GetConfigFromEnv creates a Twilio config from environment variables
func GetConfigFromEnv() Config {
	return Config{
		AccountSID:     os.Getenv("TWILIO_ACCOUNT_SID"),
		AuthToken:      os.Getenv("TWILIO_AUTH_TOKEN"),
		FromPhone:      os.Getenv("TWILIO_FROM_PHONE"),
		FromEmail:      os.Getenv("TWILIO_FROM_EMAIL"),
		SendGridAPIKey: os.Getenv("SENDGRID_API_KEY"),
	}
}

// NewClient creates a new Twilio client
func NewClient(config Config) (*Client, error) {
	if config.AccountSID == "" {
		return nil, fmt.Errorf("AccountSID is required")
	}
	if config.AuthToken == "" {
		return nil, fmt.Errorf("AuthToken is required")
	}
	if config.FromPhone == "" {
		return nil, fmt.Errorf("FromPhone is required")
	}

	return &Client{
		config:     config,
		httpClient: &http.Client{},
		baseURL:    fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s", config.AccountSID),
	}, nil
}

// SendSMS sends an SMS message using Twilio API
func (c *Client) SendSMS(to, message string) (*SMSResponse, error) {
	if to == "" {
		return nil, fmt.Errorf("recipient phone number is required")
	}
	if message == "" {
		return nil, fmt.Errorf("message body is required")
	}

	// Prepare form data
	data := url.Values{}
	data.Set("To", to)
	data.Set("From", c.config.FromPhone)
	data.Set("Body", message)

	// Create request
	req, err := http.NewRequest("POST", c.baseURL+"/Messages.json", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.config.AccountSID, c.config.AuthToken)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send SMS request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var smsResp SMSResponse
	if err := json.NewDecoder(resp.Body).Decode(&smsResp); err != nil {
		return nil, fmt.Errorf("failed to decode SMS response: %w", err)
	}

	// Check for API errors
	if resp.StatusCode >= 400 {
		return &smsResp, fmt.Errorf("SMS API error (status %d): %s - %s",
			resp.StatusCode, smsResp.ErrorCode, smsResp.ErrorMessage)
	}

	return &smsResp, nil
}

// SendEmail sends an email using SendGrid API (Twilio's email service)
// If SendGridAPIKey is not provided, it will attempt to use Twilio's Email API (legacy)
func (c *Client) SendEmail(to, subject, body string) (*EmailResponse, error) {
	if to == "" {
		return nil, fmt.Errorf("recipient email is required")
	}
	if subject == "" {
		return nil, fmt.Errorf("email subject is required")
	}
	if body == "" {
		return nil, fmt.Errorf("email body is required")
	}

	// Use SendGrid if API key is provided
	if c.config.SendGridAPIKey != "" {
		return c.sendEmailViaSendGrid(to, subject, body)
	} else {
		return nil, errors.New("missing SendGrid client to send an email")
	}

	// Fallback to basic HTTP email sending (custom implementation)
	// return c.sendEmailViaHTTP(to, subject, body)
}

// sendEmailViaSendGrid sends email using SendGrid API
func (c *Client) sendEmailViaSendGrid(to, subject, body string) (*EmailResponse, error) {
	fromEmail := c.config.FromEmail
	if fromEmail == "" {
		fromEmail = "noreply@example.com" // Default sender
	}

	// Prepare SendGrid request
	emailReq := SendGridEmailRequest{
		Personalizations: []SendGridPersonalization{
			{
				To: []SendGridEmail{
					{Email: to},
				},
			},
		},
		From: SendGridEmail{
			Email: fromEmail,
		},
		Subject: subject,
		Content: []SendGridContent{
			{
				Type:  "text/plain",
				Value: body,
			},
		},
	}

	// Marshal request body
	reqBody, err := json.Marshal(emailReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal email request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create email request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.SendGridAPIKey)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send email request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode >= 400 {
		return &EmailResponse{
			Status: "error",
			Error:  fmt.Sprintf("SendGrid API error: status %d", resp.StatusCode),
		}, fmt.Errorf("email API error: status %d", resp.StatusCode)
	}

	return &EmailResponse{
		Status:    "sent",
		MessageID: resp.Header.Get("X-Message-Id"),
	}, nil
}

// sendEmailViaHTTP sends email using a basic HTTP approach (placeholder implementation)
// Note: This is a simplified implementation. In production, you'd want to integrate with
// a proper email service like SendGrid, Mailgun, or AWS SES
func (c *Client) sendEmailViaHTTP(to, subject, body string) (*EmailResponse, error) {
	// This is a placeholder implementation
	// In a real application, you would integrate with an email service provider

	fromEmail := c.config.FromEmail
	if fromEmail == "" {
		fromEmail = "noreply@example.com"
	}

	emailData := map[string]interface{}{
		"to":        to,
		"from":      fromEmail,
		"subject":   subject,
		"body":      body,
		"timestamp": fmt.Sprintf("%d", getCurrentTimestamp()),
	}

	// In a real implementation, you would send this to your email service
	// For now, we'll just return a success response
	_ = emailData // Suppress unused variable warning

	return &EmailResponse{
		Status:    "queued",
		MessageID: fmt.Sprintf("msg_%d", getCurrentTimestamp()),
	}, nil
}

// ValidatePhoneNumber performs basic phone number validation
func (c *Client) ValidatePhoneNumber(phoneNumber string) error {
	if phoneNumber == "" {
		return fmt.Errorf("phone number cannot be empty")
	}

	// Remove common formatting characters
	cleaned := strings.ReplaceAll(phoneNumber, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")

	// Check if it starts with + (international format)
	if !strings.HasPrefix(cleaned, "+") {
		return fmt.Errorf("phone number must be in international format (start with +)")
	}

	// Basic length check (international numbers are typically 7-15 digits)
	if len(cleaned) < 8 || len(cleaned) > 16 {
		return fmt.Errorf("phone number length is invalid")
	}

	return nil
}

// ValidateEmail performs basic email validation
func (c *Client) ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	if !strings.Contains(email, "@") {
		return fmt.Errorf("email must contain @ symbol")
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return fmt.Errorf("email format is invalid")
	}

	if len(parts[0]) == 0 || len(parts[1]) == 0 {
		return fmt.Errorf("email format is invalid")
	}

	if !strings.Contains(parts[1], ".") {
		return fmt.Errorf("email domain must contain a dot")
	}

	return nil
}

// getCurrentTimestamp returns current Unix timestamp in milliseconds
func getCurrentTimestamp() int64 {
	return int64(1000000) // Placeholder - would use time.Now().UnixMilli() in real implementation
}

// GetAccountInfo retrieves Twilio account information
func (c *Client) GetAccountInfo() (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", c.baseURL+".json", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.config.AccountSID, c.config.AuthToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}
	defer resp.Body.Close()

	var accountInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&accountInfo); err != nil {
		return nil, fmt.Errorf("failed to decode account info: %w", err)
	}

	if resp.StatusCode >= 400 {
		return accountInfo, fmt.Errorf("account API error: status %d", resp.StatusCode)
	}

	return accountInfo, nil
}
