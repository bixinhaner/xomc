package admin

import (
	"errors"
	"regexp"
)

const (
	usernameMinLength = 3
	usernameMaxLength = 32
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func validateUsername(username string) error {
	if len(username) < usernameMinLength || len(username) > usernameMaxLength {
		return errors.New("username must be 3-32 characters")
	}
	if !usernameRegex.MatchString(username) {
		return errors.New("username can only contain letters, numbers, underscores and hyphens")
	}
	return nil
}
