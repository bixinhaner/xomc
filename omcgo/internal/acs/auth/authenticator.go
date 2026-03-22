package auth

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"hash"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// digestAlgorithm represents a supported Digest Auth hash algorithm.
type digestAlgorithm string

const (
	algorithmMD5    digestAlgorithm = "MD5"
	algorithmSHA256 digestAlgorithm = "SHA-256"
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
	nc := params["nc"]
	cnonce := params["cnonce"]
	qop := params["qop"]

	a.mu.Lock()
	created, exists := a.nonces[nonce]
	if exists {
		delete(a.nonces, nonce)
	}
	a.mu.Unlock()

	if !exists || time.Since(created) > nonceTTL {
		return nil, fmt.Errorf("invalid or expired nonce")
	}

	// Determine algorithm from client response; default to MD5 for backward compatibility
	algo := parseAlgorithm(params["algorithm"])
	hashFn := hashFuncFor(algo)

	// Compute expected digest response (RFC 7616)
	ha1 := digestHash(hashFn, fmt.Sprintf("%s:%s:%s", username, a.realm, a.Password))
	ha2 := digestHash(hashFn, fmt.Sprintf("%s:%s", r.Method, uri))

	var expected string
	if qop == "auth" && nc != "" && cnonce != "" {
		expected = digestHash(hashFn, fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2))
	} else {
		expected = digestHash(hashFn, fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2))
	}

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
	// Offer SHA-256 (preferred) and MD5 (fallback) per RFC 7616.
	// Each algorithm gets its own WWW-Authenticate header so the CPE can pick one.
	w.Header().Add("WWW-Authenticate",
		fmt.Sprintf(`Digest realm="%s", nonce="%s", qop="auth", algorithm=SHA-256`, a.realm, nonce))
	w.Header().Add("WWW-Authenticate",
		fmt.Sprintf(`Digest realm="%s", nonce="%s", qop="auth", algorithm=MD5`, a.realm, nonce))
}

// digestHash computes a hex-encoded hash using the given hash constructor.
func digestHash(newHash func() hash.Hash, s string) string {
	h := newHash()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// hashFuncFor returns the hash.Hash constructor for a given digest algorithm.
func hashFuncFor(algo digestAlgorithm) func() hash.Hash {
	switch algo {
	case algorithmSHA256:
		return sha256.New
	default:
		return md5.New
	}
}

// parseAlgorithm normalizes the algorithm parameter from client response.
func parseAlgorithm(s string) digestAlgorithm {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "SHA-256":
		return algorithmSHA256
	case "MD5", "":
		return algorithmMD5
	default:
		return algorithmMD5
	}
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
