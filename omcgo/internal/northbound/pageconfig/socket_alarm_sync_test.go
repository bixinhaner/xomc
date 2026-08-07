package pageconfig

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

func TestSocketAlarmPayloadMatchesNorthboundFields(t *testing.T) {
	deviceName := "Site-A"
	technology := "ENB"
	source := "CELL_1"
	eventType := "communicationsAlarm"
	location := "Cell-1"
	raisedAt := time.Date(2026, 7, 28, 10, 20, 30, 0, time.Local)
	createdAt := raisedAt.Add(2 * time.Second)
	alarm := model.Alarm{
		ID:              uuid.MustParse("7d3b9b19-4831-4240-a3ab-9e0ef4c7af42"),
		DeviceSN:        "ENB_SN001",
		Carrier:         model.CarrierCTCC,
		Severity:        model.AlarmMajor,
		AlarmType:       "communicationsAlarm",
		AlarmIdentifier: "40123",
		Description:     "Backhaul Link Down",
		Status:          model.AlarmActive,
		RaisedAt:        raisedAt,
		DeviceName:      &deviceName,
		Technology:      &technology,
		AlarmSource:     &source,
		EventType:       &eventType,
		NetworkLocation: &location,
		AdditionalInfo: map[string]string{
			"alarmSeq": "123456",
			"omcUID":   "OMC_GD_01",
			"pci":      "123",
		},
		CreatedAt: createdAt,
	}

	ctcc := socketAlarmCTCCPayload(alarm, event.SubjectAlarmRaised)
	require.Equal(t, ctccMsgRealtimeAlarm, ctcc["msgType"])
	require.Len(t, ctcc, 18)
	require.Equal(t, int64(123456), ctcc["alarmSequenceId"])
	require.Equal(t, "1", ctcc["alarmStatus"])
	require.Equal(t, "communicationsAlarm", ctcc["alarmType"])
	require.Equal(t, "major", ctcc["origSeverity"])
	require.Equal(t, "2026-07-28 10:20:30", ctcc["eventTime"])
	require.Equal(t, "40123", ctcc["alarmId"])
	require.Equal(t, "40123", ctcc["specificProblemID"])
	require.Equal(t, "Backhaul Link Down", ctcc["specificProblem"])
	require.Equal(t, "ENB_SN001", ctcc["neDn"])
	require.Equal(t, "Site-A", ctcc["neName"])
	require.Equal(t, "ENB", ctcc["neType"])
	require.Equal(t, "CELL_1", ctcc["objectDn"])
	require.Equal(t, "Cell-1", ctcc["objectName"])
	require.Equal(t, "communicationsAlarm", ctcc["objectType"])
	require.Equal(t, "alarmSeq=123456;omcUID=OMC_GD_01;pci=123", ctcc["addInfo"])
	require.Equal(t, "20260728102032", ctcc["omcReceivedTime"])
	require.Equal(t, "OMC_GD_01", ctcc["omcUID"])
	require.NotContains(t, ctcc, "objectUID")

	clearedAt := raisedAt.Add(time.Hour)
	alarm.Status = model.AlarmCleared
	alarm.ClearedAt = &clearedAt
	cucc := cuccCommand("realTimeAlarm", socketAlarmCUCCFields(alarm, event.SubjectAlarmCleared))
	command, cuccFields := parseCUCCCommand(cucc)
	require.Equal(t, "realTimeAlarm", command)
	require.Len(t, cuccFields, 34)
	require.Equal(t, "123456", cuccFields["alarmSeq"])
	require.Equal(t, "0", cuccFields["alarmStatus"])
	require.Equal(t, "communicationsAlarm", cuccFields["alarmType"])
	require.Equal(t, "major", cuccFields["origSeverity"])
	require.Equal(t, "20260728112030", cuccFields["eventTime"])
	require.Equal(t, "40123", cuccFields["alarmId"])
	require.Equal(t, "40123", cuccFields["specificProblemID"])
	require.Equal(t, "Backhaul Link Down", cuccFields["specificProblem"])
	require.Equal(t, "ENB_SN001", cuccFields["neUID"])
	require.Equal(t, "Site-A", cuccFields["neName"])
	require.Equal(t, "ENB", cuccFields["neType"])
	require.Equal(t, "CELL_1", cuccFields["objectUID"])
	require.Equal(t, "Cell-1", cuccFields["objectName"])
	require.Equal(t, "communicationsAlarm", cuccFields["objectType"])
	require.Equal(t, "alarmSeq=123456,omcUID=OMC_GD_01,pci=123", cuccFields["addInfo"])
	require.Equal(t, "20260728102032", cuccFields["omcReceivedTime"])
	require.Equal(t, "OMC_GD_01", cuccFields["omcUID"])
}

func TestSocketCTCCSyncFrameUsesSyncMessageType(t *testing.T) {
	alarm := model.Alarm{
		ID:              uuid.New(),
		DeviceSN:        "ENB_SN002",
		Severity:        model.AlarmMinor,
		AlarmType:       "processingErrorAlarm",
		AlarmIdentifier: "50012",
		Description:     "Sync Loss",
		Status:          model.AlarmActive,
		RaisedAt:        time.Date(2026, 7, 28, 10, 20, 30, 0, time.Local),
		AdditionalInfo:  map[string]string{"alarmSeq": "100"},
	}
	frame := socketCTCCSyncFrameFromItem(socketAlarmSyncItem{
		Alarm:   alarm,
		Subject: event.SubjectAlarmRaised,
	})

	require.Equal(t, ctccMsgSyncAlarmResult, frame.MessageType)
	require.Equal(t, socketMessageFormatJSON, frame.MessageFormat)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(frame.Body, &payload))
	require.Equal(t, float64(ctccMsgSyncAlarmResult), payload["msgType"])
	require.Equal(t, float64(100), payload["alarmSequenceId"])
	require.Equal(t, "1", payload["alarmStatus"])
}

func TestValidateSocketFileSyncRequest(t *testing.T) {
	req, msg := validateSocketFileSyncRequest(map[string]string{"reqid": "7"})
	require.Empty(t, msg)
	require.Equal(t, "7", req.ReqID)
	require.Equal(t, socketFileSyncSourceState, req.SyncSource)

	cases := []struct {
		name   string
		fields map[string]string
		want   string
	}{
		{
			name:   "non numeric request id",
			fields: map[string]string{"reqid": "sync-1"},
			want:   "reqId is not an integer.",
		},
		{
			name:   "invalid sync source",
			fields: map[string]string{"reqid": "1", "syncsource": "2"},
			want:   "syncSource value is incorrect",
		},
		{
			name:   "non numeric alarm sequence",
			fields: map[string]string{"reqid": "1", "syncsource": "1", "alarmseq": "abc"},
			want:   "alarmSeq is not an integer",
		},
		{
			name:   "alarm sequence requires flow sync source",
			fields: map[string]string{"reqid": "1", "syncsource": "0", "alarmseq": "100"},
			want:   "Exist alarmSeq, syncSource must be 1",
		},
		{
			name:   "bad start time",
			fields: map[string]string{"reqid": "1", "syncsource": "1", "starttime": "2026-07-28"},
			want:   "startTime format error",
		},
		{
			name:   "alarm sequence conflicts with time window",
			fields: map[string]string{"reqid": "1", "syncsource": "1", "alarmseq": "100", "starttime": "20260728102030"},
			want:   "there are both alarmSeq and startTime or endTime parameters",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, msg := validateSocketFileSyncRequest(tc.fields)
			require.Equal(t, tc.want, msg)
		})
	}
}

func TestSocketAlarmFileContentWritesGzipJSONLines(t *testing.T) {
	alarm := model.Alarm{
		ID:              uuid.New(),
		DeviceSN:        "ENB_SN003",
		Severity:        model.AlarmWarning,
		AlarmType:       "equipmentAlarm",
		AlarmIdentifier: "70001",
		Description:     "Fan Fault",
		Status:          model.AlarmActive,
		RaisedAt:        time.Date(2026, 7, 28, 10, 20, 30, 0, time.Local),
		AdditionalInfo: map[string]string{
			"alarmSeq": "987654321",
			"omcUID":   "OMC_GD_01",
		},
		CreatedAt: time.Date(2026, 7, 28, 10, 20, 32, 0, time.Local),
	}

	name, compressed, raw, err := socketAlarmFileContent(SocketAlarmConfig{Key: "socket-cucc-server", Profile: "CUCC"}, socketAlarmSyncRequest{ReqID: "42"}, []socketAlarmSyncItem{{
		Alarm:   alarm,
		Subject: event.SubjectAlarmRaised,
	}})
	require.NoError(t, err)
	require.Contains(t, name, "CUCC-ALARM-socket-cucc-server-")
	require.True(t, strings.HasSuffix(name, "-42.TXT.gz"))
	require.NotEmpty(t, compressed)

	gz, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	defer gz.Close()
	decompressed, err := io.ReadAll(gz)
	require.NoError(t, err)
	require.Equal(t, raw, string(decompressed))

	lines := strings.Split(strings.TrimSpace(string(decompressed)), "\n")
	require.Len(t, lines, 1)
	var row map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &row))
	require.Equal(t, float64(987654321), row["alarmSeq"])
	require.Len(t, row, 17)
	require.Equal(t, "1", row["alarmStatus"])
	require.Equal(t, "equipmentAlarm", row["alarmType"])
	require.Equal(t, "warning", row["origSeverity"])
	require.Equal(t, "20260728102030", row["eventTime"])
	require.Equal(t, "70001", row["alarmId"])
	require.Equal(t, "70001", row["specificProblemID"])
	require.Equal(t, "Fan Fault", row["specificProblem"])
	require.Equal(t, "ENB_SN003", row["neUID"])
	require.Equal(t, "ENB_SN003", row["neName"])
	require.NotEmpty(t, row["objectUID"])
	require.NotEmpty(t, row["objectName"])
	require.Equal(t, "alarmSeq=987654321;omcUID=OMC_GD_01", row["addInfo"])
	require.Equal(t, "20260728102032", row["omcReceivedTime"])
	require.Equal(t, "OMC_GD_01", row["omcUID"])
}
