package connreq

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDigestChallenge(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   map[string]string
	}{
		{
			name:   "standard digest challenge",
			header: `Digest realm="testrealm@host.com", nonce="abc123", qop="auth"`,
			want: map[string]string{
				"realm": "testrealm@host.com",
				"nonce": "abc123",
				"qop":   "auth",
			},
		},
		{
			name:   "digest with opaque and algorithm",
			header: `Digest realm="CPE", nonce="xyz789", qop="auth", opaque="opq456", algorithm="MD5"`,
			want: map[string]string{
				"realm":     "CPE",
				"nonce":     "xyz789",
				"qop":       "auth",
				"opaque":    "opq456",
				"algorithm": "MD5",
			},
		},
		{
			name:   "without Digest prefix",
			header: `realm="test", nonce="n1"`,
			want: map[string]string{
				"realm": "test",
				"nonce": "n1",
			},
		},
		{
			name:   "empty header",
			header: "",
			want:   map[string]string{},
		},
		{
			name:   "values without quotes",
			header: `Digest realm=test, qop=auth`,
			want: map[string]string{
				"realm": "test",
				"qop":   "auth",
			},
		},
		{
			name:   "malformed entry without equals",
			header: `Digest realm="test", badentry, nonce="n1"`,
			want: map[string]string{
				"realm": "test",
				"nonce": "n1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDigestChallenge(tt.header)
			for k, v := range tt.want {
				assert.Equal(t, v, got[k], "key %q mismatch", k)
			}
		})
	}
}

func TestMd5Hex(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "d41d8cd98f00b204e9800998ecf8427e",
		},
		{
			name:  "known value",
			input: "hello",
			want:  "5d41402abc4b2a76b9719d911017c592",
		},
		{
			name:  "digest HA1 style",
			input: "admin:testrealm:password",
			want:  "2223c68c2e2d73e8a4e64ff749300c62",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, md5Hex(tt.input))
		})
	}
}

func TestGenerateCNonce(t *testing.T) {
	// Should produce a 16-char hex string (8 random bytes)
	cnonce := generateCNonce()
	assert.Len(t, cnonce, 16)

	// Two calls should produce different values (probabilistically)
	cnonce2 := generateCNonce()
	assert.NotEqual(t, cnonce, cnonce2)
}

func TestSetDigestCredentials(t *testing.T) {
	c := &Client{}
	assert.Nil(t, c.digest)

	c.SetDigestCredentials("user", "pass")
	assert.NotNil(t, c.digest)
	assert.Equal(t, "user", c.digest.Username)
	assert.Equal(t, "pass", c.digest.Password)
}

func TestDigestCredentials_Struct(t *testing.T) {
	creds := DigestCredentials{
		Username: "admin",
		Password: "secret",
	}
	assert.Equal(t, "admin", creds.Username)
	assert.Equal(t, "secret", creds.Password)
}
