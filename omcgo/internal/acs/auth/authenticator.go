package auth

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"hash"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// digestAlgorithm represents a supported Digest Auth hash algorithm.
type digestAlgorithm string

const (
	algorithmMD5    digestAlgorithm = "MD5"
	algorithmSHA256 digestAlgorithm = "SHA-256"
)

// DeviceIdentity holds identifying information extracted during authentication.
type DeviceIdentity struct {
	// CredentialID is the identity presented by the HTTP authentication
	// mechanism. It is deliberately not treated as the TR-069 serial number:
	// many carrier deployments use one shared ACS credential for a device
	// population.
	CredentialID  string
	Authenticated bool
	Method        string
}

// DeviceAuthenticator defines the interface for CPE authentication.
type DeviceAuthenticator interface {
	Authenticate(r *http.Request) (*DeviceIdentity, error)
	Challenge(w http.ResponseWriter)
}

// NoopAuthenticator allows the protocol request to continue for compatibility
// while explicitly reporting that no device authentication was performed.
type NoopAuthenticator struct{}

func (a *NoopAuthenticator) Authenticate(r *http.Request) (*DeviceIdentity, error) {
	return &DeviceIdentity{Method: "none"}, nil
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
	return &DeviceIdentity{CredentialID: username, Authenticated: true, Method: "basic"}, nil
}

func (a *BasicAuthenticator) Challenge(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="ACS"`)
}

const nonceTTL = 5 * time.Minute

// DigestAuthenticator implements HTTP Digest authentication.
//
// nonce 的存取委托给 NonceStore（issue #65 Option B）：单实例 / 测试用进程内实现，
// 多实例横扩用 Redis 实现，使 Challenge 与 Authenticate 可落在不同实例。
type DigestAuthenticator struct {
	Username string
	Password string
	realm    string
	nonces   NonceStore
}

// NewDigestAuthenticator creates a Digest auth handler with an in-process nonce store.
// 用于单实例部署 / 测试；多实例横扩请用 NewDigestAuthenticatorWithStore 注入 Redis store。
func NewDigestAuthenticator(username, password string) *DigestAuthenticator {
	return NewDigestAuthenticatorWithStore(username, password, NewMemoryNonceStore(nonceTTL))
}

// NewDigestAuthenticatorWithStore creates a Digest auth handler with a custom NonceStore.
func NewDigestAuthenticatorWithStore(username, password string, store NonceStore) *DigestAuthenticator {
	if store == nil {
		store = NewMemoryNonceStore(nonceTTL)
	}
	return &DigestAuthenticator{
		Username: username,
		Password: password,
		realm:    "ACS",
		nonces:   store,
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
	if subtle.ConstantTimeCompare([]byte(username), []byte(a.Username)) != 1 {
		return nil, fmt.Errorf("invalid digest credentials")
	}

	// 原子消费 nonce（一次性）：命中即有效，未命中表示不存在 / 已过期 / 已被用过。
	if !a.nonces.Consume(r.Context(), nonce) {
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

	return &DeviceIdentity{CredentialID: username, Authenticated: true, Method: "digest"}, nil
}

func (a *DigestAuthenticator) Challenge(w http.ResponseWriter) {
	nonce := uuid.New().String()
	// Challenge 不在 http.Request 上下文中（仅 ResponseWriter），用 Background。
	// nonce 写入是亚毫秒级 Redis SET，不阻塞响应。
	a.nonces.Store(context.Background(), nonce)
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
// 使用进程内 nonce store（单实例 / 测试）。多实例横扩用 NewAuthenticatorWithRedis。
func NewAuthenticator(mode, username, password string) DeviceAuthenticator {
	return NewAuthenticatorWithRedis(mode, username, password, nil)
}

// NewAuthenticatorWithRedis creates an authenticator, using a Redis-backed nonce
// store for the digest mode when rdb != nil（issue #65 Option B：多实例横扩共享 nonce）。
// rdb == nil 时退化为进程内 nonce store，单实例行为不变。
func NewAuthenticatorWithRedis(mode, username, password string, rdb redis.UniversalClient) DeviceAuthenticator {
	switch mode {
	case "basic":
		return &BasicAuthenticator{Username: username, Password: password}
	case "digest":
		if rdb != nil {
			return NewDigestAuthenticatorWithStore(username, password, NewRedisNonceStore(rdb, nonceTTL))
		}
		return NewDigestAuthenticator(username, password)
	default:
		return &NoopAuthenticator{}
	}
}
