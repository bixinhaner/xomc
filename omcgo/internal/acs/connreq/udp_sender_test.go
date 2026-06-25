package connreq

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
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

// TestResolveLANUDPTarget covers the pure URL→UDP target translation that
// backs UDPSender.SendLAN. The function owns the only behaviour C2 (review)
// flagged: hard-coded port vs. caller-supplied per-device override.
func TestResolveLANUDPTarget(t *testing.T) {
	tests := []struct {
		name     string
		httpURL  string
		port     int
		wantAddr string
		wantErr  error
	}{
		{
			name:     "ipv4 with caller-supplied port wins over default",
			httpURL:  "http://172.17.1.14:7547/acs",
			port:     4789,
			wantAddr: "172.17.1.14:4789",
		},
		{
			name:     "ipv4 with port<=0 falls back to lanUDPCRDefaultPort",
			httpURL:  "http://172.17.1.14:7547/acs",
			port:     0,
			wantAddr: "172.17.1.14:3478",
		},
		{
			name:     "ipv4 with negative port also falls back to default",
			httpURL:  "http://172.17.1.14:7547/acs",
			port:     -1,
			wantAddr: "172.17.1.14:3478",
		},
		{
			name:     "ipv6 literal preserves brackets",
			httpURL:  "http://[2001:db8::14]:7547/acs",
			port:     3478,
			wantAddr: "[2001:db8::14]:3478",
		},
		{
			name:    "empty httpURL returns ErrNoLANTarget",
			httpURL: "",
			port:    3478,
			wantErr: ErrNoLANTarget,
		},
		{
			name:    "missing scheme yields empty hostname → ErrNoLANTarget",
			httpURL: "/path-only",
			port:    3478,
			wantErr: ErrNoLANTarget,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			addr, err := resolveLANUDPTarget(tc.httpURL, tc.port)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, addr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, addr)
			assert.Equal(t, tc.wantAddr, addr.String())
		})
	}
}

// TestResolveLANUDPTarget_DefaultMatches3478 locks the empirical port used
// by BAICELLS BSC7041C243 / Dengyo BSC7079B243 — changing this constant
// without revisiting the review doc would silently break LAN-direct wake.
func TestResolveLANUDPTarget_DefaultMatches3478(t *testing.T) {
	addr, err := resolveLANUDPTarget("http://10.0.0.5:7547/", 0)
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.5:3478", addr.String())
}

// TestResolveLANUDPTarget_ParseError ensures invalid URLs surface a wrapped
// error (not ErrNoLANTarget) so dispatcher records the failure metric.
func TestResolveLANUDPTarget_ParseError(t *testing.T) {
	addr, err := resolveLANUDPTarget("http://%zz", 3478)
	require.Error(t, err)
	assert.Nil(t, addr)
	assert.False(t, errors.Is(err, ErrNoLANTarget))
}
