package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateUserPhone(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		valid bool
	}{
		{name: "accepts blank phone", valid: true},
		{name: "accepts international phone", phone: " +1 (212) 555-0100 x12 ", valid: true},
		{name: "accepts landline", phone: "010-12345678", valid: true},
		{name: "rejects phone with unsupported characters", phone: "123#456", valid: false},
		{name: "rejects phone with fewer than three digits", phone: "+-12", valid: false},
		{name: "rejects phone longer than 32 characters", phone: "+1 (212) 555-0100 ext. 1234567890", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUserPhone(tt.phone)
			if tt.valid {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)
		})
	}
}

func TestNormalizeAndValidateUserPhone(t *testing.T) {
	phone, err := normalizeAndValidateUserPhone(" 010-12345678 ")
	assert.NoError(t, err)
	assert.Equal(t, "010-12345678", phone)
}
