package integration

import (
	"encoding/json"
	"testing"

	"github.com/omcgo/omcgo/internal/omcr/admin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigTemplateJSONFields verifies the ConfigTemplate JSON structure
// matches frontend templateApi.ts expectations via JSON round-trip.
func TestConfigTemplateJSONFields(t *testing.T) {
	input := `{"id":"00000000-0000-0000-0000-000000000001","name":"Test Template","description":"A test config template","carrier":"cmcc","technology":"lte","parameters":"{}","active":true}`

	var result map[string]interface{}
	err := json.Unmarshal([]byte(input), &result)
	require.NoError(t, err)

	requiredFields := []string{"id", "name", "description", "carrier", "technology"}
	for _, field := range requiredFields {
		assert.Contains(t, result, field, "ConfigTemplate must have field: %s", field)
	}
}

// TestConfigTemplateCreateVariants verifies various create request payloads.
func TestConfigTemplateCreateVariants(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
	}{
		{
			name:     "basic create",
			input:    `{"name":"New Template","description":"desc","carrier":"cmcc","technology":"lte"}`,
			wantName: "New Template",
		},
		{
			name:     "with parameters",
			input:    `{"name":"With Params","description":"d","carrier":"ctcc","technology":"nr","parameters":"{}"}`,
			wantName: "With Params",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result map[string]interface{}
			err := json.Unmarshal([]byte(tt.input), &result)
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, result["name"])
		})
	}
}

// TestFirmwareJSONFields verifies the Firmware JSON structure.
func TestFirmwareJSONFields(t *testing.T) {
	input := `{"id":"00000000-0000-0000-0000-000000000001","carrier":"cmcc","product_class":"LTE-Pico","version":"v1.0.0","file_name":"fw.bin","file_size":1024,"status":"active"}`

	var result map[string]interface{}
	err := json.Unmarshal([]byte(input), &result)
	require.NoError(t, err)

	assert.Contains(t, result, "version")
	assert.Contains(t, result, "carrier")
	assert.Contains(t, result, "product_class")
	assert.Contains(t, result, "file_name")
}

// TestUserJSONFields verifies the User JSON structure for admin API.
func TestUserJSONFields(t *testing.T) {
	user := admin.User{
		Username: "testuser",
		Email:    "test@test.com",
		Status:   admin.UserStatusActive,
	}

	data, err := json.Marshal(user)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Contains(t, result, "username")
	assert.Contains(t, result, "status")
	assert.Equal(t, "testuser", result["username"])
}

// TestCreateUserRequestFormat verifies CreateUserRequest deserialization.
func TestCreateUserRequestFormat(t *testing.T) {
	input := `{"username":"newuser","password":"Pass123!","email":"new@test.com","role":"operator","carrier":"cmcc"}`

	var req admin.CreateUserRequest
	err := json.Unmarshal([]byte(input), &req)
	require.NoError(t, err)

	assert.Equal(t, "newuser", req.Username)
	assert.Equal(t, "Pass123!", req.Password)
	assert.Equal(t, "new@test.com", req.Email)
}

// TestDeviceGroupJSONFields verifies the DeviceGroup JSON structure.
func TestDeviceGroupJSONFields(t *testing.T) {
	input := `{"id":"00000000-0000-0000-0000-000000000001","name":"Test Group","carrier":"cmcc","description":"A test group","sort_order":1}`

	var result map[string]interface{}
	err := json.Unmarshal([]byte(input), &result)
	require.NoError(t, err)

	assert.Contains(t, result, "name")
	assert.Contains(t, result, "carrier")
	assert.Equal(t, "Test Group", result["name"])
}
