package connreq

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	dedupTTL   = 30 * time.Second
	maxRetries = 3
)

// DigestCredentials holds optional HTTP Digest credentials for Connection Request.
type DigestCredentials struct {
	Username string
	Password string
}

// Client sends TR069 Connection Request messages to CPE devices.
type Client struct {
	httpClient *http.Client
	redis      redis.UniversalClient
	logger     *zap.Logger
	digest     *DigestCredentials
}

// NewClient creates a new Connection Request client.
// Supports TLS (https:// URLs) and optionally skips certificate verification
// for development environments.
func NewClient(redisClient redis.UniversalClient, logger *zap.Logger) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	return &Client{
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
		redis:  redisClient,
		logger: logger,
	}
}

// SetDigestCredentials configures HTTP Digest credentials for Connection Request.
func (c *Client) SetDigestCredentials(username, password string) {
	c.digest = &DigestCredentials{
		Username: username,
		Password: password,
	}
}

// SetInsecureTLS disables TLS certificate verification (for development only).
func (c *Client) SetInsecureTLS(insecure bool) {
	if t, ok := c.httpClient.Transport.(*http.Transport); ok {
		t.TLSClientConfig.InsecureSkipVerify = insecure
	}
}

// Send sends a Connection Request to the device's Connection Request URL.
// Uses Redis for deduplication (30s TTL).
// Supports HTTP Digest authentication and exponential backoff retry (up to 3 attempts).
func (c *Client) Send(ctx context.Context, deviceSN, url string) error {
	if url == "" {
		return fmt.Errorf("empty connection request URL for device %s", deviceSN)
	}

	// Dedup check
	dedupKey := redisx.Keys.ACSConnReqPending(deviceSN)
	set, err := c.redis.SetNX(ctx, dedupKey, "1", dedupTTL).Result()
	if err != nil {
		return fmt.Errorf("dedup check: %w", err)
	}
	if !set {
		c.logger.Debug("connection request deduplicated",
			zap.String("device_sn", deviceSN))
		return nil
	}

	// Exponential backoff retry: 1s → 2s → 4s
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			c.logger.Debug("connection request retry",
				zap.String("device_sn", deviceSN),
				zap.Int("attempt", attempt+1),
				zap.Duration("backoff", backoff))

			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		lastErr = c.doRequest(ctx, deviceSN, url)
		if lastErr == nil {
			return nil
		}
		c.logger.Warn("connection request failed",
			zap.String("device_sn", deviceSN),
			zap.Int("attempt", attempt+1),
			zap.Error(lastErr))
	}

	return fmt.Errorf("connection request failed after %d attempts: %w", maxRetries, lastErr)
}

// doRequest performs a single Connection Request HTTP GET, handling Digest auth if configured.
func (c *Client) doRequest(ctx context.Context, deviceSN, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send connection request: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	// Handle HTTP 401 with Digest authentication
	if resp.StatusCode == http.StatusUnauthorized && c.digest != nil {
		wwwAuth := resp.Header.Get("WWW-Authenticate")
		if strings.HasPrefix(wwwAuth, "Digest ") {
			return c.doDigestRequest(ctx, url, wwwAuth)
		}
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("connection request returned status %d", resp.StatusCode)
	}

	c.logger.Info("connection request sent",
		zap.String("device_sn", deviceSN),
		zap.String("url", url))

	return nil
}

// doDigestRequest performs a Digest-authenticated Connection Request.
func (c *Client) doDigestRequest(ctx context.Context, url, wwwAuth string) error {
	// Parse WWW-Authenticate header fields
	params := parseDigestChallenge(wwwAuth)
	realm := params["realm"]
	nonce := params["nonce"]
	qop := params["qop"]

	// Generate client nonce
	cnonce := generateCNonce()
	nc := "00000001"

	// Compute Digest response
	ha1 := md5Hex(c.digest.Username + ":" + realm + ":" + c.digest.Password)
	ha2 := md5Hex("GET:" + url)

	var response string
	if qop == "auth" || qop == "auth-int" {
		response = md5Hex(ha1 + ":" + nonce + ":" + nc + ":" + cnonce + ":" + qop + ":" + ha2)
	} else {
		response = md5Hex(ha1 + ":" + nonce + ":" + ha2)
	}

	// Build Authorization header
	authHeader := fmt.Sprintf(
		`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s"`,
		c.digest.Username, realm, nonce, url, response,
	)
	if qop != "" {
		authHeader += fmt.Sprintf(`, qop=%s, nc=%s, cnonce="%s"`, qop, nc, cnonce)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create digest request: %w", err)
	}
	req.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send digest connection request: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("digest connection request returned status %d", resp.StatusCode)
	}

	return nil
}

// parseDigestChallenge parses a WWW-Authenticate: Digest ... header value into key-value pairs.
func parseDigestChallenge(header string) map[string]string {
	params := make(map[string]string)
	// Strip "Digest " prefix
	header = strings.TrimPrefix(header, "Digest ")

	// Split by comma and parse key="value" pairs
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		idx := strings.IndexByte(part, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(part[:idx])
		val := strings.TrimSpace(part[idx+1:])
		// Remove surrounding quotes
		val = strings.Trim(val, `"`)
		params[key] = val
	}

	return params
}

func md5Hex(data string) string {
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func generateCNonce() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
