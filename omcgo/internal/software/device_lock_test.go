package software

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReleaseOwnedDeviceLockKeepsDifferentOwner(t *testing.T) {
	ctx := context.Background()
	deviceSN := "SN-LOCK-OWNER"
	ownerID := uuid.New()
	otherID := uuid.New()

	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	require.NoError(t, redisClient.Set(ctx, upgradeDeviceLockKey(deviceSN), ownerID.String(), time.Hour).Err())

	require.NoError(t, releaseOwnedDeviceLock(ctx, redisClient, deviceSN, otherID))

	got, err := redisClient.Get(ctx, upgradeDeviceLockKey(deviceSN)).Result()
	require.NoError(t, err)
	assert.Equal(t, ownerID.String(), got)
}
