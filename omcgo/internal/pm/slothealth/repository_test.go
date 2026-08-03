package slothealth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpectedGroupsQuery_UsesVersionedTaskMembershipAtSlotStart(t *testing.T) {
	slotStart := time.Date(2026, 8, 3, 14, 45, 0, 0, time.UTC)

	query, args, err := expectedGroupsQuery(slotStart)

	require.NoError(t, err)
	assert.Contains(t, query, "pm_aggregation_task_versions")
	assert.Contains(t, query, "pm_aggregation_version_members")
	assert.Contains(t, query, "COUNT(DISTINCT m.device_id)")
	assert.Contains(t, query, "v.effective_from <=")
	assert.Contains(t, query, "v.effective_to >")
	assert.NotContains(t, query, "t.enabled")
	assert.Contains(t, query, "d.created_at <=")
	assert.Contains(t, query, "d.deleted_at >")
	assert.NotContains(t, query, "pm_metric_values")
	assert.NotContains(t, query, "pm_measurement_anchors")
	assert.Equal(t, []any{true, slotStart, slotStart, slotStart, slotStart}, args)
}

func TestReceivedGroupsQuery_IsBoundedToOneMeasurementSlot(t *testing.T) {
	slotEnd := time.Date(2026, 8, 3, 15, 0, 0, 0, time.UTC)

	query, args, err := receivedGroupsQuery(slotEnd)

	require.NoError(t, err)
	assert.Contains(t, query, "FROM pm_files")
	assert.Contains(t, query, "measurement_end = $1")
	assert.Contains(t, query, "parsed = $2")
	assert.Contains(t, query, "COUNT(DISTINCT device_id)")
	assert.NotContains(t, query, "pm_metric_values")
	assert.NotContains(t, query, "pm_measurement_anchors")
	assert.Equal(t, []any{slotEnd, true}, args)
}

func TestUpsertSnapshotsQuery_PersistsOneSummaryPerSlotAndGroup(t *testing.T) {
	slotEnd := time.Date(2026, 8, 3, 15, 0, 0, 0, time.UTC)
	snapshots := []Snapshot{{
		SlotStart: slotEnd.Add(-15 * time.Minute), SlotEnd: slotEnd,
		Technology: "lte", Carrier: "cmcc", ExpectedDevices: 20000,
		ReceivedDevices: 19600, CoverageRatio: 0.98,
		ExpectedSnapshotVersion: "version-lte", EvaluatedAt: slotEnd.Add(12 * time.Minute),
		Status: StatusComplete,
	}}

	query, args, err := upsertSnapshotsQuery(snapshots)

	require.NoError(t, err)
	assert.Contains(t, query, "INSERT INTO pm_slot_health")
	assert.Contains(t, query, "ON CONFLICT (slot_end, technology, carrier) DO UPDATE")
	assert.Contains(t, query, "pm_slot_health.status = 'bootstrap_ignored' AND EXCLUDED.status <> 'bootstrap_ignored'")
	assert.Contains(t, query, "EXCLUDED.evaluated_at >= pm_slot_health.evaluated_at")
	assert.Contains(t, query, "RETURNING slot_start, slot_end, technology, carrier")
	assert.Len(t, args, 10)
}
