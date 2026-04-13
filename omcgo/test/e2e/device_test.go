package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceList_RequiresAuth(t *testing.T) {
	resp, err := newClient().Get(baseURL() + "/api/v1/devices")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestDeviceList_WithAuth(t *testing.T) {
	token := getAdminToken(t)
	if token == "" {
		t.Skip("could not get admin token")
	}

	req, _ := http.NewRequest("GET", baseURL()+"/api/v1/devices?page=1&page_size=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := newClient().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Contains(t, result, "items")
	assert.Contains(t, result, "total")
}

// getAdminToken logs in as admin and returns the access token, or empty string on failure.
func getAdminToken(t *testing.T) string {
	t.Helper()

	body := map[string]string{
		"username": "admin",
		"password": "admin123",
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := newClient().Post(baseURL()+"/api/v1/auth/login", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}

	token, ok := result["access_token"].(string)
	if !ok {
		return ""
	}
	return token
}
