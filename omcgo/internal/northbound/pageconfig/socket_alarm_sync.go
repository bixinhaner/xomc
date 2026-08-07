package pageconfig

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

const (
	socketAlarmQueryLimit     = 1000
	socketAlarmMessageLimit   = 1000
	socketAlarmFileMaxRows    = 50000
	socketDefaultOMCUID       = "OMC"
	socketFileSyncSourceState = "0"
	socketFileSyncSourceFlow  = "1"
)

type socketAlarmSyncRequest struct {
	ReqID      string
	AlarmSeq   string
	SyncSource string
	StartTime  *time.Time
	EndTime    *time.Time
}

type socketAlarmSyncItem struct {
	Alarm     model.Alarm
	Subject   string
	Sequence  string
	EventTime time.Time
}

func (s *socketAlarmSession) loadSocketSyncItems(ctx context.Context, req socketAlarmSyncRequest, limit int) ([]socketAlarmSyncItem, error) {
	if s.manager == nil || s.manager.svc == nil || s.manager.svc.alarmStore == nil {
		return nil, fmt.Errorf("northbound socket alarm store is not configured")
	}
	if limit <= 0 {
		limit = socketAlarmMessageLimit
	}
	items, err := querySocketAlarmItems(ctx, s.manager.svc.alarmStore, req, limit)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func querySocketAlarmItems(ctx context.Context, store alarm.AlarmStore, req socketAlarmSyncRequest, limit int) ([]socketAlarmSyncItem, error) {
	if store == nil {
		return nil, fmt.Errorf("northbound socket alarm store is not configured")
	}
	if limit <= 0 {
		limit = socketAlarmMessageLimit
	}
	if limit > socketAlarmFileMaxRows {
		limit = socketAlarmFileMaxRows
	}

	var out []socketAlarmSyncItem
	includeActive := req.SyncSource == "" || req.SyncSource == socketFileSyncSourceState || req.SyncSource == socketFileSyncSourceFlow
	includeHistory := req.SyncSource == "" || req.SyncSource == socketFileSyncSourceFlow
	if includeActive {
		items, err := querySocketActiveAlarms(ctx, store, req, limit)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
	}
	if includeHistory && len(out) < limit {
		items, err := querySocketHistoryAlarms(ctx, store, req, limit-len(out))
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
	}

	out = filterSocketAlarmItemsBySeq(out, req.AlarmSeq)
	sort.SliceStable(out, func(i, j int) bool {
		leftSeq, leftOK := parseSocketAlarmSeq(out[i].Sequence)
		rightSeq, rightOK := parseSocketAlarmSeq(out[j].Sequence)
		if leftOK && rightOK && leftSeq != rightSeq {
			return leftSeq < rightSeq
		}
		if !out[i].EventTime.Equal(out[j].EventTime) {
			return out[i].EventTime.Before(out[j].EventTime)
		}
		return out[i].Alarm.ID.String() < out[j].Alarm.ID.String()
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func querySocketActiveAlarms(ctx context.Context, store alarm.AlarmStore, req socketAlarmSyncRequest, limit int) ([]socketAlarmSyncItem, error) {
	out := make([]socketAlarmSyncItem, 0, minPositive(limit, socketAlarmQueryLimit))
	pageSize := minPositive(limit, socketAlarmQueryLimit)
	for page := 1; len(out) < limit; page++ {
		filter := alarm.AlarmFilter{
			StartTime: req.StartTime,
			EndTime:   req.EndTime,
			ListRequest: model.ListRequest{
				Page:     page,
				PageSize: pageSize,
			},
		}
		resp, err := store.ListActive(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("query active socket alarms: %w", err)
		}
		if len(resp.Items) == 0 {
			break
		}
		for _, item := range resp.Items {
			eventTime := socketAlarmEventTime(item, eventSubjectForAlarm(item))
			out = append(out, socketAlarmSyncItem{
				Alarm:     item,
				Subject:   event.SubjectAlarmRaised,
				Sequence:  socketAlarmSequence(item),
				EventTime: eventTime,
			})
			if len(out) >= limit {
				break
			}
		}
		if len(out) >= int(resp.Total) || len(resp.Items) < pageSize {
			break
		}
	}
	return out, nil
}

func querySocketHistoryAlarms(ctx context.Context, store alarm.AlarmStore, req socketAlarmSyncRequest, limit int) ([]socketAlarmSyncItem, error) {
	out := make([]socketAlarmSyncItem, 0, minPositive(limit, socketAlarmQueryLimit))
	pageSize := minPositive(limit, socketAlarmQueryLimit)
	for page := 1; len(out) < limit; page++ {
		filter := alarm.AlarmFilter{
			StartTime: req.StartTime,
			EndTime:   req.EndTime,
			ListRequest: model.ListRequest{
				Page:     page,
				PageSize: pageSize,
			},
		}
		resp, err := store.ListHistory(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("query history socket alarms: %w", err)
		}
		if len(resp.Items) == 0 {
			break
		}
		for _, item := range resp.Items {
			subject := eventSubjectForAlarm(item)
			out = append(out, socketAlarmSyncItem{
				Alarm:     item,
				Subject:   subject,
				Sequence:  socketAlarmSequence(item),
				EventTime: socketAlarmEventTime(item, subject),
			})
			if len(out) >= limit {
				break
			}
		}
		if len(out) >= int(resp.Total) || len(resp.Items) < pageSize {
			break
		}
	}
	return out, nil
}

func eventSubjectForAlarm(alarm model.Alarm) string {
	if alarm.Status == model.AlarmCleared || alarm.ClearedAt != nil {
		return event.SubjectAlarmCleared
	}
	return event.SubjectAlarmRaised
}

func filterSocketAlarmItemsBySeq(items []socketAlarmSyncItem, alarmSeq string) []socketAlarmSyncItem {
	base, ok := parseSocketAlarmSeq(alarmSeq)
	if !ok {
		return items
	}
	out := items[:0]
	for _, item := range items {
		seq, seqOK := parseSocketAlarmSeq(item.Sequence)
		if !seqOK || seq > base {
			out = append(out, item)
		}
	}
	return out
}

func parseSocketAlarmSeq(value string) (int64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	seq, err := strconv.ParseInt(value, 10, 64)
	return seq, err == nil
}

func validateSocketAlarmSeq(value string) bool {
	if strings.TrimSpace(value) == "" {
		return true
	}
	_, ok := parseSocketAlarmSeq(value)
	return ok
}

func parseSocketTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("20060102150405", value, time.Local)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func validateSocketFileSyncRequest(fields map[string]string) (socketAlarmSyncRequest, string) {
	req := socketAlarmSyncRequest{
		ReqID:      strings.TrimSpace(fields["reqid"]),
		AlarmSeq:   strings.TrimSpace(fields["alarmseq"]),
		SyncSource: strings.TrimSpace(fields["syncsource"]),
	}
	if req.ReqID == "" {
		req.ReqID = strconv.FormatInt(time.Now().UnixNano(), 10)
	} else if _, err := strconv.Atoi(req.ReqID); err != nil {
		return req, "reqId is not an integer."
	}
	if req.SyncSource == "" {
		req.SyncSource = socketFileSyncSourceState
	}
	if req.SyncSource != socketFileSyncSourceState && req.SyncSource != socketFileSyncSourceFlow {
		return req, "syncSource value is incorrect"
	}
	if req.AlarmSeq != "" && !validateSocketAlarmSeq(req.AlarmSeq) {
		return req, "alarmSeq is not an integer"
	}
	if req.AlarmSeq != "" && req.SyncSource != socketFileSyncSourceFlow {
		return req, "Exist alarmSeq, syncSource must be 1"
	}
	start, err := parseSocketTime(fields["starttime"])
	if err != nil {
		return req, "startTime format error"
	}
	end, err := parseSocketTime(fields["endtime"])
	if err != nil {
		return req, "endTime format error"
	}
	if req.AlarmSeq != "" && (start != nil || end != nil) {
		return req, "there are both alarmSeq and startTime or endTime parameters"
	}
	req.StartTime = start
	req.EndTime = end
	return req, ""
}

func validateSocketMessageSyncRequest(req socketAlarmSyncRequest, cucc bool) string {
	if strings.TrimSpace(req.AlarmSeq) != "" && !validateSocketAlarmSeq(req.AlarmSeq) {
		return "alarmSeq is not an integer"
	}
	if cucc && strings.TrimSpace(req.ReqID) != "" {
		if _, err := strconv.Atoi(req.ReqID); err != nil {
			return "reqId is not an integer."
		}
	}
	return ""
}

func socketRealtimeFrameFromItem(config SocketAlarmConfig, item socketAlarmSyncItem) (socketProtocolFrame, string, string) {
	return socketRealtimeFrameFromModel(config, item.Alarm, item.Subject)
}

func socketCTCCSyncFrameFromItem(item socketAlarmSyncItem) socketProtocolFrame {
	payload := socketAlarmCTCCPayload(item.Alarm, item.Subject)
	payload["msgType"] = ctccMsgSyncAlarmResult
	body, _ := json.Marshal(payload)
	return socketProtocolFrame{MessageType: ctccMsgSyncAlarmResult, MessageFormat: socketMessageFormatJSON, Body: body}
}

func socketAlarmCTCCPayload(alarm model.Alarm, subject string) map[string]any {
	eventTime := socketAlarmEventTime(alarm, subject)
	objectDn := firstNonEmpty(stringPtrValue(alarm.AlarmSource), stringPtrValue(alarm.NetworkLocation), alarm.DeviceSN)
	neName := firstNonEmpty(stringPtrValue(alarm.DeviceName), alarm.DeviceSN)
	neType := firstNonEmpty(stringPtrValue(alarm.Technology), string(alarm.Carrier))
	objectName := firstNonEmpty(stringPtrValue(alarm.NetworkLocation), objectDn)
	objectType := firstNonEmpty(stringPtrValue(alarm.EventType), alarm.AlarmType)
	return map[string]any{
		"msgType":           ctccMsgRealtimeAlarm,
		"alarmSequenceId":   socketAlarmSequenceJSONValue(alarm),
		"alarmStatus":       socketAlarmStatus(alarm, subject),
		"alarmType":         firstNonEmpty(alarm.AlarmType, stringPtrValue(alarm.EventType)),
		"origSeverity":      alarmSeverityText(alarm.Severity),
		"eventTime":         eventTime.Local().Format("2006-01-02 15:04:05"),
		"alarmId":           socketAlarmID(alarm),
		"specificProblemID": firstNonEmpty(alarm.AlarmIdentifier, alarm.AlarmType),
		"specificProblem":   firstNonEmpty(alarm.Description, alarm.AlarmType),
		"neDn":              alarm.DeviceSN,
		"neName":            neName,
		"neType":            neType,
		"objectDn":          objectDn,
		"objectName":        objectName,
		"objectType":        objectType,
		"addInfo":           compactAdditionalInfo(alarm.AdditionalInfo),
		"omcReceivedTime":   socketOMCReceivedTime(alarm),
		"omcUID":            socketOMCUID(alarm),
	}
}

func socketAlarmCUCCFields(alarm model.Alarm, subject string) [][2]string {
	eventTime := socketAlarmEventTime(alarm, subject)
	objectUID := firstNonEmpty(stringPtrValue(alarm.AlarmSource), stringPtrValue(alarm.NetworkLocation), alarm.DeviceSN)
	neName := firstNonEmpty(stringPtrValue(alarm.DeviceName), alarm.DeviceSN)
	neType := firstNonEmpty(stringPtrValue(alarm.Technology), string(alarm.Carrier))
	objectName := firstNonEmpty(stringPtrValue(alarm.NetworkLocation), objectUID)
	objectType := firstNonEmpty(stringPtrValue(alarm.EventType), alarm.AlarmType)
	return [][2]string{
		{"alarmSeq", socketAlarmSequence(alarm)},
		{"alarmStatus", socketAlarmStatus(alarm, subject)},
		{"alarmType", firstNonEmpty(alarm.AlarmType, stringPtrValue(alarm.EventType))},
		{"origSeverity", alarmSeverityText(alarm.Severity)},
		{"eventTime", eventTime.Local().Format("20060102150405")},
		{"alarmId", socketAlarmID(alarm)},
		{"specificProblemID", firstNonEmpty(alarm.AlarmIdentifier, alarm.AlarmType)},
		{"specificProblem", firstNonEmpty(alarm.Description, alarm.AlarmType)},
		{"neUID", alarm.DeviceSN},
		{"neName", neName},
		{"neType", neType},
		{"objectUID", objectUID},
		{"objectName", objectName},
		{"objectType", objectType},
		{"addInfo", compactAdditionalInfo(alarm.AdditionalInfo)},
		{"omcReceivedTime", socketOMCReceivedTime(alarm)},
		{"omcUID", socketOMCUID(alarm)},
	}
}

func socketAlarmSequenceJSONValue(alarm model.Alarm) any {
	seq := socketAlarmSequence(alarm)
	if numeric, ok := parseSocketAlarmSeq(seq); ok {
		return numeric
	}
	return seq
}

func socketAlarmStatus(alarm model.Alarm, subject string) string {
	if subject == event.SubjectAlarmCleared || alarm.Status == model.AlarmCleared {
		return "0"
	}
	return "1"
}

func socketAlarmEventTime(alarm model.Alarm, subject string) time.Time {
	eventTime := alarm.RaisedAt
	if subject == event.SubjectAlarmCleared || alarm.Status == model.AlarmCleared {
		if alarm.ClearedAt != nil && !alarm.ClearedAt.IsZero() {
			eventTime = *alarm.ClearedAt
		} else if !alarm.UpdatedAt.IsZero() {
			eventTime = alarm.UpdatedAt
		}
	}
	if eventTime.IsZero() && !alarm.LastUpdatedAt.IsZero() {
		eventTime = alarm.LastUpdatedAt
	}
	if eventTime.IsZero() && !alarm.CreatedAt.IsZero() {
		eventTime = alarm.CreatedAt
	}
	if eventTime.IsZero() {
		eventTime = time.Now()
	}
	return eventTime
}

func socketOMCReceivedTime(alarm model.Alarm) string {
	t := alarm.CreatedAt
	if t.IsZero() {
		t = alarm.UpdatedAt
	}
	if t.IsZero() {
		t = socketAlarmEventTime(alarm, eventSubjectForAlarm(alarm))
	}
	return t.Local().Format("20060102150405")
}

func socketOMCUID(alarm model.Alarm) string {
	if alarm.AdditionalInfo != nil {
		for _, key := range []string{"omcUID", "omc_uid", "omc_id"} {
			if value := strings.TrimSpace(alarm.AdditionalInfo[key]); value != "" {
				return value
			}
		}
	}
	return socketDefaultOMCUID
}

func socketAlarmFileContent(config SocketAlarmConfig, req socketAlarmSyncRequest, items []socketAlarmSyncItem) (string, []byte, string, error) {
	generatedAt := time.Now()
	token := safeSocketToken(firstNonEmpty(req.ReqID, "alarm-sync"))
	name := fmt.Sprintf("%s-ALARM-%s-%s-%s.TXT.gz", strings.ToUpper(config.Profile), config.Key, generatedAt.Format("20060102150405"), token)
	var raw bytes.Buffer
	for _, item := range items {
		payload := socketAlarmCTCCPayload(item.Alarm, item.Subject)
		if strings.EqualFold(config.Profile, "CUCC") {
			payload = socketCUCCFilePayload(item.Alarm, item.Subject)
		}
		line, err := json.Marshal(payload)
		if err != nil {
			return "", nil, "", fmt.Errorf("marshal socket alarm file row: %w", err)
		}
		raw.Write(line)
		raw.WriteByte('\n')
	}
	var compressed bytes.Buffer
	gw := gzip.NewWriter(&compressed)
	if _, err := gw.Write(raw.Bytes()); err != nil {
		_ = gw.Close()
		return "", nil, "", fmt.Errorf("gzip socket alarm sync file: %w", err)
	}
	if err := gw.Close(); err != nil {
		return "", nil, "", fmt.Errorf("close socket alarm sync gzip: %w", err)
	}
	return name, compressed.Bytes(), raw.String(), nil
}

func socketCUCCFilePayload(alarm model.Alarm, subject string) map[string]any {
	payload := map[string]any{}
	for _, field := range socketAlarmCUCCFields(alarm, subject) {
		if field[0] == "alarmSeq" {
			if numeric, ok := parseSocketAlarmSeq(field[1]); ok {
				payload[field[0]] = numeric
				continue
			}
		}
		payload[field[0]] = field[1]
	}
	return payload
}

func minPositive(a, b int) int {
	if a <= 0 {
		return b
	}
	if b <= 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
}
