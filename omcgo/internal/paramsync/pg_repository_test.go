package paramsync

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCreateRequestSilentlyIgnoresIdempotencyConflict(t *testing.T) {
	now := time.Now().UTC()
	key := "device_registered:" + uuid.NewString()
	req := &SyncRequest{
		ID: uuid.New(), DeviceID: uuid.New(), DeviceSN: "TEST-IDEMPOTENCY",
		CallerType: "system", TriggerReason: TriggerDeviceRegistered, SyncScope: SyncScopeFull,
		Status: RequestStatusAccepted, Priority: 10, NextAttemptAt: now,
		IdempotencyKey: &key, CreatedAt: now, UpdatedAt: now,
	}

	query, _, err := buildCreateRequest(req)

	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(query, "ON CONFLICT"))
	assert.Contains(t, query, "ON CONFLICT (caller_type, idempotency_key)")
	assert.Contains(t, query, "WHERE idempotency_key IS NOT NULL DO NOTHING")
}
