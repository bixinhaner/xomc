package auth

import (
	"crypto/md5"
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DeviceIdentity holds identifying information extracted during authentication.
type DeviceIdentity struct {
	SerialNumber string
	OUI          string
}

// DeviceAuthenticator defines the interface for CPE authentication.
type DeviceAuthenticator interface {
	Authenticate(r *http.Request) (*DeviceIdentity, error)
	Challenge(w http.ResponseWriter)
}

// NoopAuthenticator always allows access (for development).
type NoopAuthenticator struct{}

func (a *NoopAuthenticator) Authenticate(r *http.Request) (*DeviceIdentity, error) {
	return &DeviceIdentity{}, nil
}

func (a *NoopAuthenticator) Challenge(w http.ResponseWriter) {}

// BasicAuthenticator implements HTTP Basic authentication.
type BasicAuthenticator struct {
	Username string
	Password string
}

func (a *BasicAuthenticator) Authenticate(r *http.Request) (*DeviceIdentity, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return nil, fmt.Errorf("missing basic auth credentials")
	}
	if subtle.ConstantTimeCompare([]byte(username), []byte(a.Username)) != 1 ||
		subtle.ConstantTimeCompare([]byte(password), []byte(a.Password)) != 1 {
		return nil, fmt.Errorf("invalid credentials")
	}
	return &DeviceIdentity{SerialNumber: username}, nil
}

func (a *BasicAuthenticator) Challenge(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="ACS"`)
}

const nonceTTL = 5 * time.Minute

// DigestAuthenticator implements HTTP Digest authentication.
type DigestAuthenticator struct {
	Username string
	Password string
	realm    string
	mu       sync.Mutex
	nonces   map[string]time.Time
}

// NewDigestAuthenticator creates a Digest auth handler.
func NewDigestAuthenticator(username, password string) *DigestAuthenticator {
	da := &DigestAuthenticator{
		Username: username,
		Password: password,
		realm:    "ACS",
		nonces:   make(map[string]time.Time),
	}
	go da.cleanupLoop()
	return da
}

// cleanupLoop periodically removes expired nonces.
func (a *DigestAuthenticator) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		a.mu.Lock()
		now := time.Now()
		for nonce, created := range a.nonces {
			if now.Sub(created) > nonceTTL {
				delete(a.nonces, nonce)
			}
		}
		a.mu.Unlock()
	}
}

func (a *DigestAuthenticator) Authenticate(r *http.Request) (*DeviceIdentity, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Digest ") {
		return nil, fmt.Errorf("missing digest auth")
	}

	params := parseDigestAuth(authHeader[7:])
	username := params["username"]
	nonce := params["nonce"]
	uri := params["uri"]
	response := params["response"]

	a.mu.Lock()
	created, exists := a.nonces[nonce]
	if exists {
		delete(a.nonces, nonce)
	}
	a.mu.Unlock()

	if !exists || time.Since(created) > nonceTTL {
		return nil, fmt.Errorf("invalid or expired nonce")
	}

	// Compute expected digest response
	ha1 := md5Hash(fmt.Sprintf("%s:%s:%s", username, a.realm, a.Password))
	ha2 := md5Hash(fmt.Sprintf("%s:%s", r.Method, uri))
	expected := md5Hash(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2))

	if subtle.ConstantTimeCompare([]byte(response), []byte(expected)) != 1 {
		return nil, fmt.Errorf("invalid digest response")
	}

	return &DeviceIdentity{SerialNumber: username}, nil
}

func (a *DigestAuthenticator) Challenge(w http.ResponseWriter) {
	nonce := uuid.New().String()
	a.mu.Lock()
	a.nonces[nonce] = time.Now()
	a.mu.Unlock()
	w.Header().Set("WWW-Authenticate",
		fmt.Sprintf(`Digest realm="%s", nonce="%s", qop="auth"`, a.realm, nonce))
}

func md5Hash(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

func parseDigestAuth(s string) map[string]string {
	result := make(map[string]string)
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.Trim(strings.TrimSpace(kv[1]), `"`)
			result[key] = value
		}
	}
	return result
}

// NewAuthenticator creates an authenticator based on the configured mode.
func NewAuthenticator(mode, username, password string) DeviceAuthenticator {
	switch mode {
	case "basic":
		return &BasicAuthenticator{Username: username, Password: password}
	case "digest":
		return NewDigestAuthenticator(username, password)
	default:
		return &NoopAuthenticator{}
	}
}
