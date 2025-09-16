package validation

import (
	"regexp"
	"strings"
)

func IsValidPhoneNumber(phone string) bool {
	phone = strings.TrimSpace(phone)
	if len(phone) == 0 {
		return false
	}

	// Remove common formatting characters
	cleanPhone := regexp.MustCompile(`[^\d+]`).ReplaceAllString(phone, "")

	// Check for valid international format (+1234567890) or domestic (10+ digits)
	phoneRegex := regexp.MustCompile(`^(\+\d{1,3})?\d{10,15}$`)
	return phoneRegex.MatchString(cleanPhone)
}
