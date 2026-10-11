package helper

import (
	"net/mail"
	"regexp"
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func ValidUsername(username string) bool {
	return MaxLength(username, 100) && usernamePattern.MatchString(username)
}
func ValidEmail(email *string) bool {
	if email == nil || *email == "" {
		return true
	}
	address, err := mail.ParseAddress(*email)
	return err == nil && address.Address == *email && MaxLength(*email, 150)
}
