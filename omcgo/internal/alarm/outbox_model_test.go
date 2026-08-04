package alarm

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
)

func TestOutboxStatus_ValidValues(t *testing.T) {
	valid := []OutboxStatus{
		OutboxStatusPending,
		OutboxStatusPublishing,
		OutboxStatusPublished,
		OutboxStatusFailed,
		OutboxStatusDead,
	}
	for _, status := range valid {
		require.True(t, status.IsValid(), string(status))
	}
	require.False(t, OutboxStatus("").IsValid())
	require.False(t, OutboxStatus("delivered").IsValid())
}

func TestOutboxRecord_UsesOccurrenceVersionIdentity(t *testing.T) {
	eventID := uuid.New()
	occurrenceID := uuid.New()
	payload := json.RawMessage(`{"schema_version":1}`)
	record := OutboxRecord{
		ID:               uuid.New(),
		EventID:          eventID,
		AggregateType:    OutboxAggregateTypeAlarmOccurrence,
		AggregateID:      occurrenceID,
		AggregateVersion: 3,
		Subject:          "domain.alarm.lifecycle.updated",
		Payload:          payload,
		Status:           OutboxStatusPending,
		NextAttemptAt:    time.Now(),
	}

	require.Equal(t, eventID, record.EventID)
	require.Equal(t, occurrenceID, record.AggregateID)
	require.Equal(t, int64(3), record.AggregateVersion)
	require.JSONEq(t, string(payload), string(record.Payload))
}

func TestAlarmRowVersionColumnAligned(t *testing.T) {
	versionIndex := slices.Index(activeColumns, "alarms_active.alarm_version")
	require.NotEqual(t, -1, versionIndex)

	alarm, err := scanAlarmRow(alarmVersionScanRow{
		t:            t,
		versionIndex: versionIndex,
		version:      7,
	})
	require.NoError(t, err)
	require.Equal(t, int64(7), alarm.Version)
}

func TestEnsureAlarmVersion_InitializesOnlyUnsetVersion(t *testing.T) {
	unset := &model.Alarm{}
	ensureAlarmVersion(unset)
	require.Equal(t, int64(1), unset.Version)

	existing := &model.Alarm{Version: 4}
	ensureAlarmVersion(existing)
	require.Equal(t, int64(4), existing.Version)
}

func TestAlarmHistoryQueriesSelectVersion(t *testing.T) {
	rawSQL, _, err := historyAlarmRawSelect().ToSql()
	require.NoError(t, err)
	require.Contains(t, rawSQL, "alarms_history.alarm_version")
	require.Contains(t, rawSQL, "alarms_history.probable_cause, alarms_history.alarm_version FROM")

	localizedSQL, _, err := historyAlarmSelect("").ToSql()
	require.NoError(t, err)
	require.Contains(t, localizedSQL, "alarms_history.alarm_version FROM")
}

func TestAlarmHistoryRowVersionAligned(t *testing.T) {
	alarm, err := scanHistoryAlarmRow(historyVersionScanRow{t: t, version: 9})
	require.NoError(t, err)
	require.Equal(t, int64(9), alarm.Version)
}

type alarmVersionScanRow struct {
	t            *testing.T
	versionIndex int
	version      int64
}

func (r alarmVersionScanRow) Scan(dest ...interface{}) error {
	r.t.Helper()
	require.Len(r.t, dest, len(activeColumns))
	version, ok := dest[r.versionIndex].(*int64)
	require.True(r.t, ok, "alarm_version destination must be *int64")
	*version = r.version
	return nil
}

type historyVersionScanRow struct {
	t       *testing.T
	version int64
}

func (r historyVersionScanRow) Scan(dest ...interface{}) error {
	r.t.Helper()
	require.NotEmpty(r.t, dest)
	version, ok := dest[len(dest)-1].(*int64)
	require.True(r.t, ok, "last alarms_history destination must be *int64 alarm_version")
	*version = r.version
	return nil
}
