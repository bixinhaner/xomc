package software

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var releaseOwnedDeviceLockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)

func upgradeDeviceLockKey(deviceSN string) string {
	return fmt.Sprintf("software:upgrade:active:%s", deviceSN)
}

func releaseOwnedDeviceLock(ctx context.Context, client redis.UniversalClient, deviceSN string, subTaskID uuid.UUID) error {
	if client == nil || deviceSN == "" || subTaskID == uuid.Nil {
		return nil
	}
	return releaseOwnedDeviceLockScript.Run(ctx, client, []string{upgradeDeviceLockKey(deviceSN)}, subTaskID.String()).Err()
}
