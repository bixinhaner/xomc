package acs

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newAccessSnapshotStoreTest(t *testing.T, ttl time.Duration) (*RedisAccessSnapshotStore, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewRedisAccessSnapshotStore(rdb, ttl), mr
}

func TestRedisAccessSnapshotStoreRoundTripAndIdentityScope(t *testing.T) {
	store, _ := newAccessSnapshotStoreTest(t, time.Minute)
	expiresAt := time.Now().UTC().Add(time.Minute)
	want := AccessSnapshot{
		SerialNumber: "SN-1", Carrier: "cmcc", OUI: "001122", ProductClass: "FAP/BM",
		State: "accepted", DecisionVersion: 3, EvidenceVersion: 4,
		PublishedAt: time.Now().UTC(), ExpiresAt: expiresAt,
	}
	require.NoError(t, store.Set(context.Background(), want))

	got, found, err := store.Get(context.Background(), "SN-1", "001122", "FAP/BM")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, want.SerialNumber, got.SerialNumber)
	require.Equal(t, want.State, got.State)
	require.Equal(t, want.DecisionVersion, got.DecisionVersion)

	_, found, err = store.Get(context.Background(), "SN-1", "ffffff", "FAP/BM")
	require.NoError(t, err)
	require.False(t, found)
}

func TestRedisAccessSnapshotStoreDoesNotOverwriteNewerDecision(t *testing.T) {
	store, _ := newAccessSnapshotStoreTest(t, time.Minute)
	expiresAt := time.Now().UTC().Add(time.Minute)
	require.NoError(t, store.Set(context.Background(), AccessSnapshot{
		SerialNumber: "SN-ORDER", Carrier: "cmcc", State: "accepted",
		DecisionVersion: 5, EvidenceVersion: 6, ExpiresAt: expiresAt,
	}))
	require.NoError(t, store.Set(context.Background(), AccessSnapshot{
		SerialNumber: "SN-ORDER", Carrier: "cmcc", State: "rejected",
		DecisionVersion: 4, EvidenceVersion: 5, ExpiresAt: expiresAt,
	}))

	got, found, err := store.Get(context.Background(), "SN-ORDER", "", "")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "accepted", got.State)
	require.Equal(t, int64(5), got.DecisionVersion)
}

func TestRedisAccessSnapshotStoreExpiresAndRejectsInvalidSnapshot(t *testing.T) {
	store, mr := newAccessSnapshotStoreTest(t, time.Minute)
	store.now = func() time.Time { return time.Unix(100, 0).UTC() }

	_, found, err := store.Get(context.Background(), "SN-unknown", "", "")
	require.NoError(t, err)
	require.False(t, found)

	require.Error(t, store.Set(context.Background(), AccessSnapshot{SerialNumber: "SN-1", Carrier: "cmcc", State: "accepted", ExpiresAt: time.Unix(99, 0).UTC()}))
	require.NoError(t, store.Set(context.Background(), AccessSnapshot{
		SerialNumber: "SN-1", Carrier: "cmcc", State: "accepted", ExpiresAt: time.Unix(101, 0).UTC(),
	}))
	mr.FastForward(2 * time.Second)
	_, found, err = store.Get(context.Background(), "SN-1", "", "")
	require.NoError(t, err)
	require.False(t, found)
}

func TestAccessSnapshotProjectorPersistsDecisionEvent(t *testing.T) {
	store, _ := newAccessSnapshotStoreTest(t, time.Minute)
	projector := NewAccessSnapshotProjector(nil, store, nil)
	evt, err := event.NewEvent("device.access.accepted", AccessSnapshot{
		SerialNumber: "SN-2", Carrier: "ctcc", OUI: "AABBCC", ProductClass: "FAP/MLN",
		State: "accepted", NormalTasksFrozen: false, DecisionVersion: 8,
		EvidenceVersion: 9, PolicyVersionID: "policy-9", ExpiresAt: time.Now().UTC().Add(time.Minute),
	})
	require.NoError(t, err)
	require.NoError(t, projector.project(context.Background(), evt))

	snapshot, found, err := store.Get(context.Background(), "SN-2", "aabbcc", "FAP/MLN")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "policy-9", snapshot.PolicyVersionID)
	require.Equal(t, int64(8), snapshot.DecisionVersion)
}
