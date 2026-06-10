package stun

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Default TTL for cached STUN address entries.
const defaultTTL = 30 * time.Minute

// defaultMaxEntries caps the in-process L1 STUN address map so a UDP flood with
// spoofed device serials cannot grow it without bound (DoS / memory exhaustion,
// issue #7). 100K matches the project's baseline device scale (root CLAUDE.md
// §1); operators expecting more set a higher cap via Server config.
const defaultMaxEntries = 100_000

// ErrStoreFull is returned by Set when the L1 cache has reached its entry cap
// and the entry is for a new device serial. Existing serials are always allowed
// to refresh, so steady-state operation is never blocked — only unbounded
// growth from unknown sources is rejected (backpressure).
var ErrStoreFull = errors.New("stun store at capacity")

// StunInfo holds a device's public (NAT-mapped) address discovered via STUN.
type StunInfo struct {
	IP        string
	Port      int
	UpdatedAt time.Time
}

// UDPAddr converts StunInfo to a net.UDPAddr.
func (s *StunInfo) UDPAddr() *net.UDPAddr {
	return &net.UDPAddr{
		IP:   net.ParseIP(s.IP),
		Port: s.Port,
	}
}

// Store manages device STUN address information with a two-level cache:
//   - L1: in-process memory (sync.RWMutex + map) for hot-path lookups
//   - L2: Redis for cross-instance sharing
type Store struct {
	mu         sync.RWMutex
	local      map[string]*StunInfo // key = device serial number
	rdb        redis.UniversalClient
	ttl        time.Duration
	maxEntries int // L1 cap; 0 means unlimited (not recommended in production)
	logger     *zap.Logger
}

// NewStore creates a new STUN address store.
func NewStore(rdb redis.UniversalClient, logger *zap.Logger) *Store {
	return &Store{
		local:      make(map[string]*StunInfo),
		rdb:        rdb,
		ttl:        defaultTTL,
		maxEntries: defaultMaxEntries,
		logger:     logger,
	}
}

// SetTTL overrides the default TTL for cached entries.
func (s *Store) SetTTL(ttl time.Duration) {
	s.ttl = ttl
}

// SetMaxEntries overrides the L1 cache entry cap. A value <= 0 disables the cap
// (unbounded growth — not recommended; provided only for tests / special
// deployments). Larger fleets should raise this above the device count.
func (s *Store) SetMaxEntries(n int) {
	s.mu.Lock()
	s.maxEntries = n
	s.mu.Unlock()
}

// Set stores a device's STUN address in both L1 and L2 cache.
//
// To bound L1 memory under a UDP flood (issue #7) the map size is capped at
// maxEntries. An update to an already-known serial always succeeds (steady
// state is never blocked); a *new* serial is rejected with ErrStoreFull once
// the cap is hit, after a best-effort sweep of TTL-expired entries. On rejection
// neither L1 nor L2 is written.
func (s *Store) Set(ctx context.Context, deviceSN string, info *StunInfo) error {
	// L1: memory
	s.mu.Lock()
	if _, known := s.local[deviceSN]; !known && s.maxEntries > 0 && len(s.local) >= s.maxEntries {
		// At capacity for a new serial: try to make room by dropping expired
		// entries inline, then re-check.
		s.evictExpiredLocked()
		if len(s.local) >= s.maxEntries {
			s.mu.Unlock()
			if s.logger != nil {
				s.logger.Warn("stun store at capacity, rejecting new device address",
					zap.String("device_sn", deviceSN),
					zap.Int("max_entries", s.maxEntries))
			}
			return ErrStoreFull
		}
	}
	s.local[deviceSN] = info
	s.mu.Unlock()

	// L2: Redis
	if s.rdb != nil {
		key := redisx.Keys.ACSSTUN(deviceSN)
		err := s.rdb.HSet(ctx, key, map[string]interface{}{
			"ip":         info.IP,
			"port":       info.Port,
			"updated_at": info.UpdatedAt.Unix(),
		}).Err()
		if err != nil {
			return fmt.Errorf("redis hset stun info: %w", err)
		}
		s.rdb.Expire(ctx, key, s.ttl)
	}

	return nil
}

// Get retrieves a device's STUN address. Checks L1 first, then L2.
// Returns nil, nil if not found.
func (s *Store) Get(ctx context.Context, deviceSN string) (*StunInfo, error) {
	// L1: memory
	s.mu.RLock()
	info, ok := s.local[deviceSN]
	s.mu.RUnlock()

	if ok && time.Since(info.UpdatedAt) < s.ttl {
		return info, nil
	}

	// L2: Redis
	if s.rdb == nil {
		return nil, nil
	}

	key := redisx.Keys.ACSSTUN(deviceSN)
	vals, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis hgetall stun info: %w", err)
	}
	if len(vals) == 0 {
		return nil, nil
	}

	port, _ := strconv.Atoi(vals["port"])
	ts, _ := strconv.ParseInt(vals["updated_at"], 10, 64)
	info = &StunInfo{
		IP:        vals["ip"],
		Port:      port,
		UpdatedAt: time.Unix(ts, 0),
	}

	// Backfill L1
	s.mu.Lock()
	s.local[deviceSN] = info
	s.mu.Unlock()

	return info, nil
}

// SetFromInform stores a STUN address from a TR-069 Inform's UDPConnectionRequestAddress
// parameter. The address format is "IP:Port".
func (s *Store) SetFromInform(ctx context.Context, deviceSN, udpAddr string) error {
	ip, portStr, err := net.SplitHostPort(udpAddr)
	if err != nil {
		return fmt.Errorf("parse udp address %q: %w", udpAddr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("parse port %q: %w", portStr, err)
	}

	info := &StunInfo{
		IP:        ip,
		Port:      port,
		UpdatedAt: time.Now(),
	}
	return s.Set(ctx, deviceSN, info)
}

// SetFromUDPAddr stores a STUN address directly from a net.UDPAddr (used by the STUN processor).
func (s *Store) SetFromUDPAddr(ctx context.Context, deviceSN string, addr *net.UDPAddr) error {
	info := &StunInfo{
		IP:        addr.IP.String(),
		Port:      addr.Port,
		UpdatedAt: time.Now(),
	}
	return s.Set(ctx, deviceSN, info)
}

// Delete removes a device's STUN address from both caches.
func (s *Store) Delete(ctx context.Context, deviceSN string) {
	s.mu.Lock()
	delete(s.local, deviceSN)
	s.mu.Unlock()

	if s.rdb != nil {
		s.rdb.Del(ctx, redisx.Keys.ACSSTUN(deviceSN))
	}
}

// Size returns the number of entries in the L1 cache.
func (s *Store) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.local)
}

// CleanExpired removes entries from L1 cache that have exceeded the TTL.
func (s *Store) CleanExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.evictExpiredLocked()
}

// evictExpiredLocked removes TTL-expired entries from the L1 map and returns the
// count removed. Caller must hold s.mu (write lock).
func (s *Store) evictExpiredLocked() int {
	now := time.Now()
	cleaned := 0
	for sn, info := range s.local {
		if now.Sub(info.UpdatedAt) >= s.ttl {
			delete(s.local, sn)
			cleaned++
		}
	}
	return cleaned
}

// ParseDeviceCode parses the non-standard format "deviceCode_IP_Port"
// used by the legacy system's Redis queue for CPE STUN address transfer.
func ParseDeviceCode(raw string) (deviceCode, ip string, port int, err error) {
	parts := strings.SplitN(raw, "_", 3)
	if len(parts) != 3 {
		return "", "", 0, fmt.Errorf("invalid device code format: %q", raw)
	}
	port, err = strconv.Atoi(parts[2])
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid port in device code %q: %w", raw, err)
	}
	return parts[0], parts[1], port, nil
}
