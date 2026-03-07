package connreq

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	connReqKeyPrefix = "acs:connreq:pending:"
	dedupTTL         = 30 * time.Second
)

// Client sends TR069 Connection Request messages to CPE devices.
type Client struct {
	httpClient *http.Client
	redis      redis.UniversalClient
	logger     *zap.Logger
}

// NewClient creates a new Connection Request client.
func NewClient(redisClient redis.UniversalClient, logger *zap.Logger) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		redis:      redisClient,
		logger:     logger,
	}
}

// Send sends a Connection Request to the device's Connection Request URL.
// Uses Redis for deduplication (30s TTL).
func (c *Client) Send(ctx context.Context, deviceSN, url string) error {
	if url == "" {
		return fmt.Errorf("empty connection request URL for device %s", deviceSN)
	}

	// Dedup check
	dedupKey := connReqKeyPrefix + deviceSN
	set, err := c.redis.SetNX(ctx, dedupKey, "1", dedupTTL).Result()
	if err != nil {
		return fmt.Errorf("dedup check: %w", err)
	}
	if !set {
		c.logger.Debug("connection request deduplicated",
			zap.String("device_sn", deviceSN))
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send connection request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("connection request returned status %d", resp.StatusCode)
	}

	c.logger.Info("connection request sent",
		zap.String("device_sn", deviceSN),
		zap.String("url", url))

	return nil
}
