package auth

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoopAuthenticator_Authenticate(t *testing.T) {
	a := &NoopAuthenticator{}
	r := httptest.NewRequest(http.MethodPost, "/acs", nil)

	identity, err := a.Authenticate(r)

	require.NoError(t, err)
	assert.NotNil(t, identity)
}

func TestBasicAuthenticator_Success(t *testing.T) {
	a := &BasicAuthenticator{Username: "cpe-user", Password: "cpe-pass"}
	r := httptest.NewRequest(http.MethodPost, "/acs", nil)
	r.SetBasicAuth("cpe-user", "cpe-pass")

	identity, err := a.Authenticate(r)

	require.NoError(t, err)
	assert.NotNil(t, identity)
	assert.Equal(t, "cpe-user", identity.SerialNumber)
}

func TestBasicAuthenticator_WrongPassword(t *testing.T) {
	a := &BasicAuthenticator{Username: "cpe-user", Password: "cpe-pass"}
	r := httptest.NewRequest(http.MethodPost, "/acs", nil)
	r.SetBasicAuth("cpe-user", "wrong-pass")

	identity, err := a.Authenticate(r)

	assert.Error(t, err)
	assert.Nil(t, identity)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestBasicAuthenticator_MissingAuth(t *testing.T) {
	a := &BasicAuthenticator{Username: "cpe-user", Password: "cpe-pass"}
	r := httptest.NewRequest(http.MethodPost, "/acs", nil)

	identity, err := a.Authenticate(r)

	assert.Error(t, err)
	assert.Nil(t, identity)
	assert.Contains(t, err.Error(), "missing basic auth")
}

func TestBasicAuthenticator_Challenge(t *testing.T) {
	a := &BasicAuthenticator{Username: "cpe-user", Password: "cpe-pass"}
	w := httptest.NewRecorder()

	a.Challenge(w)

	assert.Equal(t, `Basic realm="ACS"`, w.Header().Get("WWW-Authenticate"))
}

func TestDigestAuthenticator_Success(t *testing.T) {
	a := NewDigestAuthenticator("cpe-user", "cpe-pass")

	// Step 1: Call Challenge to populate nonce
	w := httptest.NewRecorder()
	a.Challenge(w)
	wwwAuth := w.Header().Get("WWW-Authenticate")
	require.Contains(t, wwwAuth, "nonce=")

	// Extract nonce from header
	nonce := extractDigestField(wwwAuth, "nonce")
	require.NotEmpty(t, nonce)

	// Step 2: Build valid digest response
	uri := "/acs"
	method := http.MethodPost
	realm := "ACS"
	ha1 := testMD5(fmt.Sprintf("%s:%s:%s", "cpe-user", realm, "cpe-pass"))
	ha2 := testMD5(fmt.Sprintf("%s:%s", method, uri))
	response := testMD5(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2))

	authHeader := fmt.Sprintf(
		`Digest username="cpe-user", realm="%s", nonce="%s", uri="%s", response="%s"`,
		realm, nonce, uri, response,
	)

	r := httptest.NewRequest(method, uri, nil)
	r.Header.Set("Authorization", authHeader)

	identity, err := a.Authenticate(r)

	require.NoError(t, err)
	assert.NotNil(t, identity)
	assert.Equal(t, "cpe-user", identity.SerialNumber)
}

func TestDigestAuthenticator_InvalidNonce(t *testing.T) {
	a := NewDigestAuthenticator("cpe-user", "cpe-pass")

	// Use a nonce that was never issued via Challenge
	authHeader := `Digest username="cpe-user", realm="ACS", nonce="bad-nonce", uri="/acs", response="abcdef1234567890abcdef1234567890"`
	r := httptest.NewRequest(http.MethodPost, "/acs", nil)
	r.Header.Set("Authorization", authHeader)

	identity, err := a.Authenticate(r)

	assert.Error(t, err)
	assert.Nil(t, identity)
	assert.Contains(t, err.Error(), "invalid nonce")
}

func TestDigestAuthenticator_MissingAuth(t *testing.T) {
	a := NewDigestAuthenticator("cpe-user", "cpe-pass")
	r := httptest.NewRequest(http.MethodPost, "/acs", nil)

	identity, err := a.Authenticate(r)

	assert.Error(t, err)
	assert.Nil(t, identity)
	assert.Contains(t, err.Error(), "missing digest auth")
}

func TestNewAuthenticator_Basic(t *testing.T) {
	a := NewAuthenticator("basic", "user", "pass")

	_, ok := a.(*BasicAuthenticator)
	assert.True(t, ok, "expected *BasicAuthenticator")
}

func TestNewAuthenticator_Digest(t *testing.T) {
	a := NewAuthenticator("digest", "user", "pass")

	_, ok := a.(*DigestAuthenticator)
	assert.True(t, ok, "expected *DigestAuthenticator")
}

func TestNewAuthenticator_Default(t *testing.T) {
	a := NewAuthenticator("", "user", "pass")

	_, ok := a.(*NoopAuthenticator)
	assert.True(t, ok, "expected *NoopAuthenticator")
}

func TestParseDigestAuth(t *testing.T) {
	input := `username="admin", realm="ACS", nonce="abc123", uri="/acs", response="deadbeef"`
	result := parseDigestAuth(input)

	assert.Equal(t, "admin", result["username"])
	assert.Equal(t, "ACS", result["realm"])
	assert.Equal(t, "abc123", result["nonce"])
	assert.Equal(t, "/acs", result["uri"])
	assert.Equal(t, "deadbeef", result["response"])
}

// --- helpers ---

// testMD5 duplicates the production helper for test digest computation.
func testMD5(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

// extractDigestField extracts a named field from a WWW-Authenticate Digest header value.
func extractDigestField(header, field string) string {
	prefix := field + `="`
	idx := strings.Index(header, prefix)
	if idx < 0 {
		return ""
	}
	start := idx + len(prefix)
	end := strings.Index(header[start:], `"`)
	if end < 0 {
		return ""
	}
	return header[start : start+end]
}

