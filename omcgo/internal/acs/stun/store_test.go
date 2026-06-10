package stun

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestStore(t *testing.T) (*Store, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })
	return NewStore(rdb, zap.NewNop()), mr
}

func TestStore_SetAndGet(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	info := &StunInfo{
		IP:        "1.2.3.4",
		Port:      5000,
		UpdatedAt: time.Now(),
	}

	err := store.Set(ctx, "DEV001", info)
	require.NoError(t, err)

	got, err := store.Get(ctx, "DEV001")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "1.2.3.4", got.IP)
	assert.Equal(t, 5000, got.Port)
}

func TestStore_Get_NotFound(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	got, err := store.Get(ctx, "NONEXISTENT")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestStore_Get_L1Hit(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	info := &StunInfo{
		IP:        "10.0.0.1",
		Port:      8080,
		UpdatedAt: time.Now(),
	}
	store.Set(ctx, "DEV002", info)

	// Should hit L1 cache
	got, err := store.Get(ctx, "DEV002")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "10.0.0.1", got.IP)
}

func TestStore_Get_L2Backfill(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	info := &StunInfo{
		IP:        "172.16.0.1",
		Port:      9090,
		UpdatedAt: time.Now(),
	}
	store.Set(ctx, "DEV003", info)

	// Clear L1 to force L2 lookup
	store.mu.Lock()
	delete(store.local, "DEV003")
	store.mu.Unlock()

	got, err := store.Get(ctx, "DEV003")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "172.16.0.1", got.IP)
	assert.Equal(t, 9090, got.Port)

	// Verify L1 backfill
	store.mu.RLock()
	_, inL1 := store.local["DEV003"]
	store.mu.RUnlock()
	assert.True(t, inL1)
}

func TestStore_SetFromInform(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	err := store.SetFromInform(ctx, "CPE001", "203.0.113.50:5060")
	require.NoError(t, err)

	got, err := store.Get(ctx, "CPE001")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "203.0.113.50", got.IP)
	assert.Equal(t, 5060, got.Port)
}

func TestStore_SetFromInform_InvalidAddr(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	err := store.SetFromInform(ctx, "CPE001", "invalid-addr")
	assert.Error(t, err)
}

func TestStore_SetFromUDPAddr(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	addr := &net.UDPAddr{IP: net.IPv4(192, 168, 1, 100), Port: 12345}
	err := store.SetFromUDPAddr(ctx, "ENB001", addr)
	require.NoError(t, err)

	got, err := store.Get(ctx, "ENB001")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "192.168.1.100", got.IP)
	assert.Equal(t, 12345, got.Port)
}

func TestStore_Delete(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	store.Set(ctx, "DEV004", &StunInfo{IP: "1.1.1.1", Port: 80, UpdatedAt: time.Now()})
	store.Delete(ctx, "DEV004")

	got, err := store.Get(ctx, "DEV004")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestStore_Size(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	assert.Equal(t, 0, store.Size())

	store.Set(ctx, "DEV1", &StunInfo{IP: "1.1.1.1", Port: 1, UpdatedAt: time.Now()})
	store.Set(ctx, "DEV2", &StunInfo{IP: "2.2.2.2", Port: 2, UpdatedAt: time.Now()})
	assert.Equal(t, 2, store.Size())
}

func TestStore_CleanExpired(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	store.SetTTL(1 * time.Second)

	store.Set(ctx, "OLD", &StunInfo{
		IP:        "1.1.1.1",
		Port:      1,
		UpdatedAt: time.Now().Add(-2 * time.Second),
	})
	store.Set(ctx, "NEW", &StunInfo{
		IP:        "2.2.2.2",
		Port:      2,
		UpdatedAt: time.Now(),
	})

	cleaned := store.CleanExpired()
	assert.Equal(t, 1, cleaned)
	assert.Equal(t, 1, store.Size())

	// NEW should still be there
	got, _ := store.Get(ctx, "NEW")
	assert.NotNil(t, got)
}

func TestStore_NilRedis(t *testing.T) {
	store := NewStore(nil, zap.NewNop())
	ctx := context.Background()

	// Should work with nil Redis (L1 only)
	err := store.Set(ctx, "DEV1", &StunInfo{IP: "1.1.1.1", Port: 1, UpdatedAt: time.Now()})
	require.NoError(t, err)

	got, err := store.Get(ctx, "DEV1")
	require.NoError(t, err)
	assert.NotNil(t, got)
}

func TestStore_MaxEntries_RejectsNewSerialWhenFull(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	store.SetMaxEntries(2)

	require.NoError(t, store.Set(ctx, "A", &StunInfo{IP: "1.1.1.1", Port: 1, UpdatedAt: time.Now()}))
	require.NoError(t, store.Set(ctx, "B", &StunInfo{IP: "2.2.2.2", Port: 2, UpdatedAt: time.Now()}))

	// A third distinct serial must be rejected (backpressure, not unbounded growth).
	err := store.Set(ctx, "C", &StunInfo{IP: "3.3.3.3", Port: 3, UpdatedAt: time.Now()})
	require.ErrorIs(t, err, ErrStoreFull)
	assert.Equal(t, 2, store.Size())

	got, _ := store.Get(ctx, "C")
	assert.Nil(t, got, "rejected serial must not be stored in L1")
}

func TestStore_MaxEntries_KnownSerialAlwaysRefreshes(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	store.SetMaxEntries(1)

	require.NoError(t, store.Set(ctx, "A", &StunInfo{IP: "1.1.1.1", Port: 1, UpdatedAt: time.Now()}))
	// Updating the same serial at capacity must succeed (steady state not blocked).
	require.NoError(t, store.Set(ctx, "A", &StunInfo{IP: "9.9.9.9", Port: 9, UpdatedAt: time.Now()}))

	got, _ := store.Get(ctx, "A")
	require.NotNil(t, got)
	assert.Equal(t, "9.9.9.9", got.IP)
}

func TestStore_MaxEntries_EvictsExpiredToMakeRoom(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	store.SetTTL(1 * time.Second)
	store.SetMaxEntries(2)

	// Fill with one expired + one fresh entry.
	require.NoError(t, store.Set(ctx, "OLD", &StunInfo{IP: "1.1.1.1", Port: 1, UpdatedAt: time.Now().Add(-2 * time.Second)}))
	require.NoError(t, store.Set(ctx, "FRESH", &StunInfo{IP: "2.2.2.2", Port: 2, UpdatedAt: time.Now()}))

	// New serial at capacity: inline eviction of the expired OLD entry makes
	// room → succeeds. Size (L1) stays at the cap.
	require.NoError(t, store.Set(ctx, "NEW", &StunInfo{IP: "3.3.3.3", Port: 3, UpdatedAt: time.Now()}))
	assert.Equal(t, 2, store.Size())

	// OLD must be gone from L1 (eviction is L1-only; L2/Redis entries expire via
	// their own TTL, so we assert on the in-process map directly, not Get()).
	store.mu.RLock()
	_, oldInL1 := store.local["OLD"]
	_, newInL1 := store.local["NEW"]
	store.mu.RUnlock()
	assert.False(t, oldInL1, "expired entry should have been evicted from L1")
	assert.True(t, newInL1, "new entry should be present in L1")
}

func TestStore_MaxEntries_ZeroDisablesCap(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	store.SetMaxEntries(0) // unlimited

	for i := 0; i < 5; i++ {
		require.NoError(t, store.Set(ctx, string(rune('A'+i)),
			&StunInfo{IP: "1.1.1.1", Port: i, UpdatedAt: time.Now()}))
	}
	assert.Equal(t, 5, store.Size())
}

func TestStunInfo_UDPAddr(t *testing.T) {
	info := &StunInfo{IP: "10.0.0.1", Port: 3478}
	addr := info.UDPAddr()
	assert.Equal(t, net.IPv4(10, 0, 0, 1).To4(), addr.IP.To4())
	assert.Equal(t, 3478, addr.Port)
}

func TestParseDeviceCode(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantCode   string
		wantIP     string
		wantPort   int
		wantErr    bool
	}{
		{
			name:     "valid",
			raw:      "CPE001_1.2.3.4_5000",
			wantCode: "CPE001",
			wantIP:   "1.2.3.4",
			wantPort: 5000,
		},
		{
			name:    "too few parts",
			raw:     "CPE001_1.2.3.4",
			wantErr: true,
		},
		{
			name:    "invalid port",
			raw:     "CPE001_1.2.3.4_abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, ip, port, err := ParseDeviceCode(tt.raw)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantCode, code)
			assert.Equal(t, tt.wantIP, ip)
			assert.Equal(t, tt.wantPort, port)
		})
	}
}
