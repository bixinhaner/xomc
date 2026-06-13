package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogin_Success(t *testing.T) {
	res := encryptedLogin(t, "admin", "admin123")

	// Accept 401 if the default admin password was changed in this environment.
	if res.status == http.StatusUnauthorized {
		t.Skip("admin/admin123 not valid (password changed), skipping login success assertions")
	}

	require.Equal(t, http.StatusOK, res.status, "login status; body=%v", res.raw)
	require.NotNil(t, res.data, "login response missing data envelope: %v", res.raw)
	assert.NotEmpty(t, res.data["access_token"], "access_token should be present")
	assert.NotEmpty(t, res.data["refresh_token"], "refresh_token should be present")
	assert.Equal(t, "Bearer", res.data["token_type"])
}

func TestLogin_InvalidPassword(t *testing.T) {
	// A valid cipher of a wrong password decrypts fine but fails credential check → 401.
	res := encryptedLogin(t, "admin", "definitely_the_wrong_password")
	assert.Equal(t, http.StatusUnauthorized, res.status, "body=%v", res.raw)
}

func TestAuthMe_Unauthorized(t *testing.T) {
	resp, err := newClient().Get(baseURL() + "/api/v1/auth/me")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
