package validation

import (
	"regexp"
	"strings"
)

func IsValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if len(email) == 0 {
		return false
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}
