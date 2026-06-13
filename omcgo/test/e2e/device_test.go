package e2e

import (
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

	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Unified response envelope: { data: { items, total }, msg, ret }.
	var env struct {
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&env))
	require.NotNil(t, env.Data, "devices response missing data envelope")
	assert.Contains(t, env.Data, "items")
	assert.Contains(t, env.Data, "total")
}

// getAdminToken logs in as admin via the encrypted flow and returns the access token,
// or empty string on failure.
func getAdminToken(t *testing.T) string {
	t.Helper()
	res := encryptedLogin(t, "admin", "admin123")
	if res.status != http.StatusOK || res.data == nil {
		return ""
	}
	token, _ := res.data["access_token"].(string)
	return token
}
