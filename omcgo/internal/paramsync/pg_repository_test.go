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

func TestAutomaticAdmissionBucketIsStableAndBounded(t *testing.T) {
	deviceID := uuid.MustParse("00112233-4455-6677-8899-aabbccddeeff")

	first := automaticAdmissionBucket(deviceID)
	second := automaticAdmissionBucket(deviceID)

	assert.Equal(t, first, second)
	assert.GreaterOrEqual(t, first, int16(0))
	assert.Less(t, first, int16(automaticAdmissionBucketCount))
}

func TestListReleaseCandidatesQueryUsesPersistedCampaignBoundary(t *testing.T) {
	campaignID := uuid.New()
	query, args, err := listReleaseCandidatesQuery(campaignID, 200)

	require.NoError(t, err)
	assert.Contains(t, query, "superseded_requests AS")
	assert.Contains(t, query, "req.campaign_id <> $5")
	assert.Contains(t, query, "req.status IN ($7, $8)")
	assert.Contains(t, query, "result_code = $3")
	assert.Contains(t, query, "NOT EXISTS")
	assert.Contains(t, query, "other_campaign.updated_at >= current_campaign.updated_at")
	assert.Contains(t, query, "other_campaign.category = $6 || req.campaign_id::text")
	assert.Contains(t, query, "config_apply_versions")
	assert.Contains(t, query, "VALUES ($1, 0, now())")
	assert.Contains(t, query, "ON CONFLICT (category) DO NOTHING")
	assert.Contains(t, query, "d.created_at <")
	assert.NotContains(t, query, "parameter_sync_runs")
	assert.Contains(t, args, releaseCampaignCategory(campaignID))
	assert.Equal(t, releaseCampaignCategory(campaignID), args[0])
	assert.Equal(t, releaseCampaignCategory(campaignID), args[1])
	assert.Equal(t, ResultCodeSupersededRelease, args[2])
	assert.Equal(t, TriggerOMCUpgrade, args[3])
	assert.Equal(t, campaignID, args[4])
	assert.Equal(t, releaseCampaignCategoryPrefix, args[5])
	assert.Equal(t, RequestStatusAccepted, args[6])
	assert.Equal(t, RequestStatusQueued, args[7])
}

func TestAdmissionReleaseCandidateQueryCoversTerminalAndExpiredLeases(t *testing.T) {
	query, _, err := admissionReleaseCandidatesQuery(time.Now().UTC(), 500)

	require.NoError(t, err)
	assert.Contains(t, query, "parameter_sync_admission_reservations")
	assert.Contains(t, query, "lease_until <")
	assert.Contains(t, query, "req.status")
	assert.Contains(t, query, "FOR UPDATE")
	assert.Contains(t, query, "SKIP LOCKED")
}
