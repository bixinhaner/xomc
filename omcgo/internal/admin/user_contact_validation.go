package admin

import (
	"fmt"
	"regexp"
	"strings"
)

var userPhonePattern = regexp.MustCompile(`^[0-9 +()xX.\-]+$`)

func normalizeAndValidateUserPhone(phone string) (string, error) {
	phone = strings.TrimSpace(phone)
	if err := validateUserPhone(phone); err != nil {
		return "", err
	}
	return phone, nil
}

func validateUserPhone(phone string) error {
	if phone == "" {
		return nil
	}
	if len(phone) < 3 || len(phone) > 32 || !userPhonePattern.MatchString(phone) {
		return fmt.Errorf("phone is invalid")
	}
	digits := 0
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	if digits < 3 {
		return fmt.Errorf("phone is invalid")
	}
	return nil
}
