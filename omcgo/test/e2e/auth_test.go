package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogin_Success(t *testing.T) {
	body := map[string]string{
		"username": "admin",
		"password": "admin123",
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := newClient().Post(baseURL()+"/api/v1/auth/login", "application/json", bytes.NewReader(jsonBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Accept both 200 (success) and 401 (if default password was changed)
	if resp.StatusCode == http.StatusUnauthorized {
		t.Skip("default admin password not set, skipping login test")
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Contains(t, result, "access_token")
	assert.Contains(t, result, "refresh_token")
	assert.Equal(t, "Bearer", result["token_type"])
}

func TestLogin_InvalidPassword(t *testing.T) {
	body := map[string]string{
		"username": "admin",
		"password": "wrong_password",
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := newClient().Post(baseURL()+"/api/v1/auth/login", "application/json", bytes.NewReader(jsonBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAuthMe_Unauthorized(t *testing.T) {
	resp, err := newClient().Get(baseURL() + "/api/v1/auth/me")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
