package auth

import (
	"crypto/md5"
	"crypto/sha256"
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
	assert.False(t, identity.Authenticated)
	assert.Equal(t, "none", identity.Method)
}

func TestBasicAuthenticator_Success(t *testing.T) {
	a := &BasicAuthenticator{Username: "cpe-user", Password: "cpe-pass"}
	r := httptest.NewRequest(http.MethodPost, "/acs", nil)
	r.SetBasicAuth("cpe-user", "cpe-pass")

	identity, err := a.Authenticate(r)

	require.NoError(t, err)
	assert.NotNil(t, identity)
	assert.Equal(t, "cpe-user", identity.CredentialID)
	assert.True(t, identity.Authenticated)
	assert.Equal(t, "basic", identity.Method)
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

func TestDigestAuthenticator_Challenge_OffersMultipleAlgorithms(t *testing.T) {
	a := NewDigestAuthenticator("cpe-user", "cpe-pass")
	w := httptest.NewRecorder()

	a.Challenge(w)

	values := w.Header().Values("WWW-Authenticate")
	require.Len(t, values, 2, "expected two WWW-Authenticate headers (SHA-256 + MD5)")

	assert.Contains(t, values[0], "algorithm=SHA-256", "first header should offer SHA-256")
	assert.Contains(t, values[1], "algorithm=MD5", "second header should offer MD5")

	// Both should share the same nonce
	nonce0 := extractDigestField(values[0], "nonce")
	nonce1 := extractDigestField(values[1], "nonce")
	assert.Equal(t, nonce0, nonce1, "both challenges must use the same nonce")
}

func TestDigestAuthenticator_Success_MD5(t *testing.T) {
	a := NewDigestAuthenticator("cpe-user", "cpe-pass")

	w := httptest.NewRecorder()
	a.Challenge(w)
	nonce := extractDigestField(w.Header().Values("WWW-Authenticate")[1], "nonce")
	require.NotEmpty(t, nonce)

	uri := "/acs"
	method := http.MethodPost
	realm := "ACS"
	ha1 := testMD5(fmt.Sprintf("%s:%s:%s", "cpe-user", realm, "cpe-pass"))
	ha2 := testMD5(fmt.Sprintf("%s:%s", method, uri))
	response := testMD5(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2))

	authHeader := fmt.Sprintf(
		`Digest username="cpe-user", realm="%s", nonce="%s", uri="%s", response="%s", algorithm=MD5`,
		realm, nonce, uri, response,
	)

	r := httptest.NewRequest(method, uri, nil)
	r.Header.Set("Authorization", authHeader)

	identity, err := a.Authenticate(r)

	require.NoError(t, err)
	assert.NotNil(t, identity)
	assert.Equal(t, "cpe-user", identity.CredentialID)
	assert.True(t, identity.Authenticated)
	assert.Equal(t, "digest", identity.Method)
}

func TestDigestAuthenticator_Success_MD5_NoAlgorithmParam(t *testing.T) {
	// Legacy CPE that does not send algorithm parameter — should default to MD5.
	a := NewDigestAuthenticator("cpe-user", "cpe-pass")

	w := httptest.NewRecorder()
	a.Challenge(w)
	nonce := extractDigestField(w.Header().Values("WWW-Authenticate")[0], "nonce")
	require.NotEmpty(t, nonce)

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
	assert.Equal(t, "cpe-user", identity.CredentialID)
}

func TestDigestAuthenticator_Success_SHA256(t *testing.T) {
	a := NewDigestAuthenticator("cpe-user", "cpe-pass")

	w := httptest.NewRecorder()
	a.Challenge(w)
	nonce := extractDigestField(w.Header().Values("WWW-Authenticate")[0], "nonce")
	require.NotEmpty(t, nonce)

	uri := "/acs"
	method := http.MethodPost
	realm := "ACS"
	ha1 := testSHA256(fmt.Sprintf("%s:%s:%s", "cpe-user", realm, "cpe-pass"))
	ha2 := testSHA256(fmt.Sprintf("%s:%s", method, uri))
	response := testSHA256(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2))

	authHeader := fmt.Sprintf(
		`Digest username="cpe-user", realm="%s", nonce="%s", uri="%s", response="%s", algorithm=SHA-256`,
		realm, nonce, uri, response,
	)

	r := httptest.NewRequest(method, uri, nil)
	r.Header.Set("Authorization", authHeader)

	identity, err := a.Authenticate(r)

	require.NoError(t, err)
	assert.NotNil(t, identity)
	assert.Equal(t, "cpe-user", identity.CredentialID)
}

func TestDigestAuthenticator_Success_SHA256_WithQopAuth(t *testing.T) {
	a := NewDigestAuthenticator("cpe-user", "cpe-pass")

	w := httptest.NewRecorder()
	a.Challenge(w)
	nonce := extractDigestField(w.Header().Values("WWW-Authenticate")[0], "nonce")
	require.NotEmpty(t, nonce)

	uri := "/acs"
	method := http.MethodPost
	realm := "ACS"
	nc := "00000001"
	cnonce := "test-cnonce"
	qop := "auth"

	ha1 := testSHA256(fmt.Sprintf("%s:%s:%s", "cpe-user", realm, "cpe-pass"))
	ha2 := testSHA256(fmt.Sprintf("%s:%s", method, uri))
	response := testSHA256(fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2))

	authHeader := fmt.Sprintf(
		`Digest username="cpe-user", realm="%s", nonce="%s", uri="%s", response="%s", algorithm=SHA-256, qop=%s, nc=%s, cnonce="%s"`,
		realm, nonce, uri, response, qop, nc, cnonce,
	)

	r := httptest.NewRequest(method, uri, nil)
	r.Header.Set("Authorization", authHeader)

	identity, err := a.Authenticate(r)

	require.NoError(t, err)
	assert.NotNil(t, identity)
	assert.Equal(t, "cpe-user", identity.CredentialID)
}

func TestDigestAuthenticator_Success_MD5_WithQopAuth(t *testing.T) {
	a := NewDigestAuthenticator("cpe-user", "cpe-pass")

	w := httptest.NewRecorder()
	a.Challenge(w)
	nonce := extractDigestField(w.Header().Values("WWW-Authenticate")[0], "nonce")
	require.NotEmpty(t, nonce)

	uri := "/acs"
	method := http.MethodPost
	realm := "ACS"
	nc := "00000001"
	cnonce := "test-cnonce"
	qop := "auth"

	ha1 := testMD5(fmt.Sprintf("%s:%s:%s", "cpe-user", realm, "cpe-pass"))
	ha2 := testMD5(fmt.Sprintf("%s:%s", method, uri))
	response := testMD5(fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2))

	authHeader := fmt.Sprintf(
		`Digest username="cpe-user", realm="%s", nonce="%s", uri="%s", response="%s", algorithm=MD5, qop=%s, nc=%s, cnonce="%s"`,
		realm, nonce, uri, response, qop, nc, cnonce,
	)

	r := httptest.NewRequest(method, uri, nil)
	r.Header.Set("Authorization", authHeader)

	identity, err := a.Authenticate(r)

	require.NoError(t, err)
	assert.NotNil(t, identity)
	assert.Equal(t, "cpe-user", identity.CredentialID)
}

func TestDigestAuthenticator_InvalidNonce(t *testing.T) {
	a := NewDigestAuthenticator("cpe-user", "cpe-pass")

	authHeader := `Digest username="cpe-user", realm="ACS", nonce="bad-nonce", uri="/acs", response="abcdef1234567890abcdef1234567890"`
	r := httptest.NewRequest(http.MethodPost, "/acs", nil)
	r.Header.Set("Authorization", authHeader)

	identity, err := a.Authenticate(r)

	assert.Error(t, err)
	assert.Nil(t, identity)
	assert.Contains(t, err.Error(), "invalid or expired nonce")
}

func TestDigestAuthenticator_WrongResponse_SHA256(t *testing.T) {
	a := NewDigestAuthenticator("cpe-user", "cpe-pass")

	w := httptest.NewRecorder()
	a.Challenge(w)
	nonce := extractDigestField(w.Header().Values("WWW-Authenticate")[0], "nonce")
	require.NotEmpty(t, nonce)

	authHeader := fmt.Sprintf(
		`Digest username="cpe-user", realm="ACS", nonce="%s", uri="/acs", response="0000000000000000000000000000000000000000000000000000000000000000", algorithm=SHA-256`,
		nonce,
	)

	r := httptest.NewRequest(http.MethodPost, "/acs", nil)
	r.Header.Set("Authorization", authHeader)

	identity, err := a.Authenticate(r)

	assert.Error(t, err)
	assert.Nil(t, identity)
	assert.Contains(t, err.Error(), "invalid digest response")
}

func TestDigestAuthenticator_MissingAuth(t *testing.T) {
	a := NewDigestAuthenticator("cpe-user", "cpe-pass")
	r := httptest.NewRequest(http.MethodPost, "/acs", nil)

	identity, err := a.Authenticate(r)

	assert.Error(t, err)
	assert.Nil(t, identity)
	assert.Contains(t, err.Error(), "missing digest auth")
}

func TestDigestAuthenticator_RejectsUnexpectedUsername(t *testing.T) {
	a := NewDigestAuthenticator("cpe-user", "cpe-pass")
	w := httptest.NewRecorder()
	a.Challenge(w)
	nonce := extractDigestField(w.Header().Values("WWW-Authenticate")[1], "nonce")
	uri := "/acs"
	ha1 := testMD5(fmt.Sprintf("%s:%s:%s", "other-user", "ACS", "cpe-pass"))
	ha2 := testMD5(fmt.Sprintf("%s:%s", http.MethodPost, uri))
	response := testMD5(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2))
	req := httptest.NewRequest(http.MethodPost, uri, nil)
	req.Header.Set("Authorization", fmt.Sprintf(
		`Digest username="other-user", realm="ACS", nonce="%s", uri="%s", response="%s", algorithm=MD5`,
		nonce, uri, response,
	))

	identity, err := a.Authenticate(req)

	assert.Error(t, err)
	assert.Nil(t, identity)
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
	input := `username="admin", realm="ACS", nonce="abc123", uri="/acs", response="deadbeef", algorithm=SHA-256`
	result := parseDigestAuth(input)

	assert.Equal(t, "admin", result["username"])
	assert.Equal(t, "ACS", result["realm"])
	assert.Equal(t, "abc123", result["nonce"])
	assert.Equal(t, "/acs", result["uri"])
	assert.Equal(t, "deadbeef", result["response"])
	assert.Equal(t, "SHA-256", result["algorithm"])
}

func TestParseAlgorithm(t *testing.T) {
	tests := []struct {
		input    string
		expected digestAlgorithm
	}{
		{"SHA-256", algorithmSHA256},
		{"sha-256", algorithmSHA256},
		{"MD5", algorithmMD5},
		{"md5", algorithmMD5},
		{"", algorithmMD5},
		{"unknown", algorithmMD5},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("input=%q", tt.input), func(t *testing.T) {
			assert.Equal(t, tt.expected, parseAlgorithm(tt.input))
		})
	}
}

// --- helpers ---

func testMD5(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

func testSHA256(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

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
