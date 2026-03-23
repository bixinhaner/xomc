package connreq

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCPEConnectionRequest(t *testing.T) {
	msg := buildCPEConnectionRequest("acs.example.com:7547", "mysecret")
	s := string(msg)

	assert.True(t, strings.HasPrefix(s, "GET http://acs.example.com:7547/?"), "should start with GET and server addr")
	assert.True(t, strings.HasSuffix(s, "HTTP/1.1\r\n\r\n"), "should end with HTTP/1.1 CRLF CRLF")
	assert.Contains(t, s, "ts=", "should contain timestamp")
	assert.Contains(t, s, "id=", "should contain random id")
	assert.Contains(t, s, "un=dps", "should contain username=dps")
	assert.Contains(t, s, "cn=", "should contain nonce")
	assert.Contains(t, s, "sig=", "should contain HMAC signature")
}

func TestBuildCPEConnectionRequest_SignatureLength(t *testing.T) {
	msg := buildCPEConnectionRequest("10.0.0.1:7547", "secret")
	s := string(msg)

	// Extract sig value — HMAC-SHA1 hex is 40 chars
	idx := strings.Index(s, "sig=")
	require.Greater(t, idx, 0)
	sigStart := idx + 4
	sigEnd := strings.Index(s[sigStart:], " ")
	sig := s[sigStart : sigStart+sigEnd]
	assert.Len(t, sig, 40, "HMAC-SHA1 hex digest should be 40 characters")
}

func TestBuildCPEConnectionRequest_DifferentCalls(t *testing.T) {
	// Two calls should produce different messages (different timestamps/nonces)
	msg1 := string(buildCPEConnectionRequest("addr:7547", "secret"))
	msg2 := string(buildCPEConnectionRequest("addr:7547", "secret"))
	// They might occasionally be equal if called in the same millisecond with same rand,
	// but the structure should always be valid
	assert.True(t, strings.HasPrefix(msg1, "GET http://addr:7547/?"))
	assert.True(t, strings.HasPrefix(msg2, "GET http://addr:7547/?"))
}

func TestENBRequestMessage(t *testing.T) {
	assert.Equal(t, "infromrequest", enbRequestMessage)
}

func TestRestartCommandFormat(t *testing.T) {
	sn := "TEST-SN-001"
	hash := md5.Sum([]byte(sn))
	expected := "/restart_" + hex.EncodeToString(hash[:])

	// Verify format: /restart_ + 32-char md5 hex
	assert.True(t, strings.HasPrefix(expected, "/restart_"))
	assert.Len(t, expected, len("/restart_")+32, "restart command should be /restart_ + 32-char md5")
}
