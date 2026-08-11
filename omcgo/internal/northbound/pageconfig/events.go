package pageconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/ssh"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

func (r *PgRepository) CreateEvent(ctx context.Context, event PageConfigEvent) (*PageConfigEvent, error) {
	event = normalizeEvent(event)
	summary, err := json.Marshal(event.Summary)
	if err != nil {
		return nil, fmt.Errorf("marshal northbound page-config event summary: %w", err)
	}
	row := r.pool.QueryRow(ctx, `
INSERT INTO northbound_page_config_events (
  capability, owner_code, target_key, event_type, status, artifact_type,
  artifact_name, artifact_path, payload, payload_content_type, error_message, summary
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
RETURNING id::text, capability, owner_code, target_key, event_type, status,
          artifact_type, artifact_name, artifact_path, payload,
          payload_content_type, error_message, summary, created_at, updated_at`,
		event.Capability, event.OwnerCode, event.TargetKey, event.EventType, event.Status,
		event.ArtifactType, event.ArtifactName, event.ArtifactPath, event.Payload,
		event.PayloadContentType, event.ErrorMessage, summary)
	return scanPageConfigEvent(row)
}

func (r *PgRepository) ListEvents(ctx context.Context, filter EventFilter) (EventListResult, error) {
	args := make([]any, 0, 6)
	where := []string{"true"}
	if strings.TrimSpace(filter.Capability) != "" {
		args = append(args, filter.Capability)
		where = append(where, fmt.Sprintf("capability = $%d", len(args)))
	}
	if strings.TrimSpace(filter.OwnerCode) != "" {
		args = append(args, filter.OwnerCode)
		where = append(where, fmt.Sprintf("owner_code = $%d", len(args)))
	}
	if strings.TrimSpace(filter.TargetKey) != "" {
		args = append(args, filter.TargetKey)
		where = append(where, fmt.Sprintf("target_key = $%d", len(args)))
	}
	if strings.TrimSpace(filter.EventType) != "" {
		args = append(args, filter.EventType)
		where = append(where, fmt.Sprintf("event_type = $%d", len(args)))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	whereClause := strings.Join(where, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, fmt.Sprintf(`
SELECT COUNT(*)
  FROM northbound_page_config_events
 WHERE %s`, whereClause), args...).Scan(&total); err != nil {
		return EventListResult{}, fmt.Errorf("count northbound_page_config_events: %w", err)
	}
	limit := normalizeLimit(filter.Limit)
	offset := normalizeOffset(filter.Offset)
	args = append(args, limit, offset)
	query := fmt.Sprintf(`
SELECT id::text, capability, owner_code, target_key, event_type, status,
       artifact_type, artifact_name, artifact_path, payload,
       payload_content_type, error_message, summary, created_at, updated_at
  FROM northbound_page_config_events
 WHERE %s
 ORDER BY created_at DESC
 LIMIT $%d OFFSET $%d`, whereClause, len(args)-1, len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return EventListResult{}, fmt.Errorf("query northbound_page_config_events: %w", err)
	}
	defer rows.Close()

	events := make([]PageConfigEvent, 0)
	for rows.Next() {
		event, err := scanPageConfigEvent(rows)
		if err != nil {
			return EventListResult{}, err
		}
		events = append(events, *event)
	}
	if err := rows.Err(); err != nil {
		return EventListResult{}, err
	}
	return EventListResult{Items: events, Total: total, Limit: limit, Offset: offset}, nil
}

func (r *PgRepository) GetEvent(ctx context.Context, id string) (*PageConfigEvent, error) {
	row := r.pool.QueryRow(ctx, `
SELECT id::text, capability, owner_code, target_key, event_type, status,
       artifact_type, artifact_name, artifact_path, payload,
       payload_content_type, error_message, summary, created_at, updated_at
  FROM northbound_page_config_events
 WHERE id::text = $1`, id)
	return scanPageConfigEvent(row)
}

func (r *PgRepository) PruneEvents(ctx context.Context, filter EventFilter, keep int) (int64, error) {
	if keep <= 0 {
		return 0, nil
	}
	args := make([]any, 0, 4)
	where := []string{"true"}
	if strings.TrimSpace(filter.Capability) != "" {
		args = append(args, filter.Capability)
		where = append(where, fmt.Sprintf("capability = $%d", len(args)))
	}
	if strings.TrimSpace(filter.OwnerCode) != "" {
		args = append(args, filter.OwnerCode)
		where = append(where, fmt.Sprintf("owner_code = $%d", len(args)))
	}
	if strings.TrimSpace(filter.TargetKey) != "" {
		args = append(args, filter.TargetKey)
		where = append(where, fmt.Sprintf("target_key = $%d", len(args)))
	}
	if strings.TrimSpace(filter.EventType) != "" {
		args = append(args, filter.EventType)
		where = append(where, fmt.Sprintf("event_type = $%d", len(args)))
	}
	args = append(args, keep)
	whereClause := strings.Join(where, " AND ")
	tag, err := r.pool.Exec(ctx, fmt.Sprintf(`
WITH stale AS (
  SELECT id
    FROM northbound_page_config_events
   WHERE %s
   ORDER BY created_at DESC, id DESC
  OFFSET $%d
)
DELETE FROM northbound_page_config_events
 WHERE id IN (SELECT id FROM stale)`, whereClause, len(args)), args...)
	if err != nil {
		return 0, fmt.Errorf("prune northbound_page_config_events: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *PgRepository) CleanupExpiredResults(ctx context.Context, runBefore time.Time, eventBefore time.Time) (ResultCleanupSummary, error) {
	var summary ResultCleanupSummary
	if !runBefore.IsZero() {
		tag, err := r.pool.Exec(ctx, `DELETE FROM northbound_file_runs WHERE created_at < $1`, runBefore)
		if err != nil {
			return summary, fmt.Errorf("cleanup expired northbound_file_runs: %w", err)
		}
		summary.RunsDeleted = tag.RowsAffected()
	}
	if !eventBefore.IsZero() {
		tag, err := r.pool.Exec(ctx, `DELETE FROM northbound_page_config_events WHERE created_at < $1`, eventBefore)
		if err != nil {
			return summary, fmt.Errorf("cleanup expired northbound_page_config_events: %w", err)
		}
		summary.EventsDeleted = tag.RowsAffected()
	}
	summary.RunBefore = runBefore
	summary.EventBefore = eventBefore
	return summary, nil
}

func scanPageConfigEvent(row scanner) (*PageConfigEvent, error) {
	var event PageConfigEvent
	var status, artifactType string
	var summaryRaw []byte
	if err := row.Scan(
		&event.ID, &event.Capability, &event.OwnerCode, &event.TargetKey, &event.EventType,
		&status, &artifactType, &event.ArtifactName, &event.ArtifactPath, &event.Payload,
		&event.PayloadContentType, &event.ErrorMessage, &summaryRaw, &event.CreatedAt, &event.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan northbound_page_config_events row: %w", err)
	}
	event.Status = RunStatus(status)
	event.ArtifactType = EventArtifactType(artifactType)
	if len(summaryRaw) > 0 {
		if err := json.Unmarshal(summaryRaw, &event.Summary); err != nil {
			return nil, fmt.Errorf("unmarshal northbound page-config event summary: %w", err)
		}
	}
	return &event, nil
}

func normalizeEvent(event PageConfigEvent) PageConfigEvent {
	event.Capability = strings.TrimSpace(event.Capability)
	event.OwnerCode = strings.TrimSpace(event.OwnerCode)
	event.TargetKey = strings.TrimSpace(event.TargetKey)
	event.EventType = strings.TrimSpace(event.EventType)
	if event.EventType == "" {
		event.EventType = "status"
	}
	if event.Status == "" {
		event.Status = RunStatusSuccess
	}
	if event.ArtifactType == "" {
		event.ArtifactType = EventArtifactMessage
	}
	if event.PayloadContentType == "" {
		event.PayloadContentType = "text/plain; charset=utf-8"
	}
	if event.Summary == nil {
		event.Summary = map[string]any{}
	}
	return event
}

func eventFromFileRun(run FileRun) PageConfigEvent {
	capability := string(run.ProfileKind)
	if capability == "" {
		capability = "file"
	}
	payload := run.ErrorMessage
	if run.Status == RunStatusSuccess {
		payload = ""
	}
	return PageConfigEvent{
		Capability:         capability,
		OwnerCode:          run.ProfileCode,
		TargetKey:          firstNonEmpty(run.GroupID, run.ObjectCode),
		EventType:          "run",
		Status:             run.Status,
		ArtifactType:       EventArtifactFile,
		ArtifactName:       run.ArtifactName,
		ArtifactPath:       run.ArtifactPath,
		Payload:            payload,
		PayloadContentType: "text/plain; charset=utf-8",
		ErrorMessage:       run.ErrorMessage,
		Summary: map[string]any{
			"run_id":              run.ID,
			"domain":              run.Domain,
			"object_code":         run.ObjectCode,
			"row_count":           run.RowCount,
			"artifact_size":       run.ArtifactSize,
			"compression_enabled": run.CompressionEnabled,
			"compression_format":  run.CompressionFormat,
		},
	}
}

func eventFromDeliveryTest(target DeliveryTarget, result DeliveryConnectionResult) PageConfigEvent {
	status := RunStatusFailed
	if result.Success {
		status = RunStatusSuccess
	}
	payload := mustMarshalEventPayload(result)
	return PageConfigEvent{
		Capability:         "delivery",
		OwnerCode:          target.OwnerCode,
		TargetKey:          target.Key,
		EventType:          "connection_test",
		Status:             status,
		ArtifactType:       EventArtifactJSON,
		ArtifactName:       fmt.Sprintf("%s connection test", firstNonEmpty(target.Name, target.Key)),
		ArtifactPath:       deliveryArtifactPath(target),
		Payload:            payload,
		PayloadContentType: "application/json; charset=utf-8",
		ErrorMessage:       errorMessageWhenFailed(result.Success, result.Message),
		Summary: map[string]any{
			"protocol":             result.Protocol,
			"host":                 result.Host,
			"port":                 result.Port,
			"tcp_reachable":        result.TCPReachable,
			"auth_probe_supported": result.AuthProbeSupported,
			"auth_probe_passed":    result.AuthProbePassed,
			"latency_ms":           result.LatencyMs,
		},
	}
}

func eventFromDeliveryRun(target DeliveryTarget, result DeliveryUploadResult) PageConfigEvent {
	status := RunStatusFailed
	if result.Success {
		status = RunStatusSuccess
	}
	payload := mustMarshalEventPayload(result)
	return PageConfigEvent{
		Capability:         "delivery",
		OwnerCode:          result.ProfileCode,
		TargetKey:          target.Key,
		EventType:          "delivery",
		Status:             status,
		ArtifactType:       EventArtifactJSON,
		ArtifactName:       fmt.Sprintf("%s delivery %s", firstNonEmpty(target.Name, target.Key), result.ProfileCode),
		ArtifactPath:       result.RemotePath,
		Payload:            payload,
		PayloadContentType: "application/json; charset=utf-8",
		ErrorMessage:       result.Error,
		Summary: map[string]any{
			"run_id":       result.RunID,
			"target_owner": target.OwnerCode,
			"profile_kind": result.ProfileKind,
			"profile_code": result.ProfileCode,
			"protocol":     result.Protocol,
			"host":         result.Host,
			"port":         result.Port,
			"remote_path":  result.RemotePath,
			"attempts":     result.Attempts,
			"bytes":        result.Bytes,
			"latency_ms":   result.LatencyMs,
			"message":      result.Message,
		},
	}
}

func eventFromSNMPTest(target SNMPAlarmTarget) PageConfigEvent {
	status := RunStatusSuccess
	message := "SNMP alarm varbind sample generated from omcAlarmMIB"
	if strings.TrimSpace(target.TargetHost) == "" {
		status = RunStatusFailed
		message = "SNMP target_host is empty; sample varbind generated but no target is configured"
	}
	return PageConfigEvent{
		Capability:         "snmp",
		OwnerCode:          target.Key,
		TargetKey:          target.Key,
		EventType:          "message_test",
		Status:             status,
		ArtifactType:       EventArtifactMessage,
		ArtifactName:       fmt.Sprintf("omcAlarmNotification %s", target.NotificationType),
		ArtifactPath:       fmt.Sprintf("snmp://%s:%d", firstNonEmpty(target.TargetHost, "-"), target.TargetPort),
		Payload:            sampleSNMPPayload(),
		PayloadContentType: "text/plain; charset=utf-8",
		ErrorMessage:       errorMessageWhenFailed(status == RunStatusSuccess, message),
		Summary: map[string]any{
			"message":           message,
			"version":           target.Version,
			"notification_type": target.NotificationType,
			"mib_oid_root":      "1.3.6.1.4.1.53058",
			"field_count":       len(defaultSNMPAlarmFields()),
		},
	}
}

func eventFromSNMPSend(target SNMPAlarmTarget, result SNMPSendResult, payload string) PageConfigEvent {
	status := RunStatusFailed
	if result.Success {
		status = RunStatusSuccess
	}
	if payload == "" {
		payload = mustMarshalEventPayload(result)
	}
	return PageConfigEvent{
		Capability:         "snmp",
		OwnerCode:          target.Key,
		TargetKey:          target.Key,
		EventType:          "message_test",
		Status:             status,
		ArtifactType:       EventArtifactMessage,
		ArtifactName:       fmt.Sprintf("omcAlarmNotification %s", target.NotificationType),
		ArtifactPath:       fmt.Sprintf("snmp://%s:%d", firstNonEmpty(target.TargetHost, "-"), target.TargetPort),
		Payload:            payload,
		PayloadContentType: "text/plain; charset=utf-8",
		ErrorMessage:       result.Error,
		Summary: map[string]any{
			"message":           result.Message,
			"alarm_id":          result.AlarmID,
			"version":           result.Version,
			"notification_type": result.NotificationType,
			"mib_oid_root":      "1.3.6.1.4.1.53058",
			"field_count":       result.VarbindCount,
			"latency_ms":        result.LatencyMs,
			"host":              result.Host,
			"port":              result.Port,
		},
	}
}

func eventFromSocketTest(config SocketAlarmConfig) PageConfigEvent {
	status := RunStatusSuccess
	if !config.Enabled {
		status = RunStatusTerminated
	}
	return PageConfigEvent{
		Capability:         "socket",
		OwnerCode:          config.Key,
		TargetKey:          config.Key,
		EventType:          "message_test",
		Status:             status,
		ArtifactType:       EventArtifactMessage,
		ArtifactName:       fmt.Sprintf("%s alarm message sample", config.Profile),
		ArtifactPath:       fmt.Sprintf("socket://%s:%d", firstNonEmpty(config.ListenIP, "0.0.0.0"), config.ListenPort),
		Payload:            sampleSocketPayload(config),
		PayloadContentType: "text/plain; charset=utf-8",
		Summary: map[string]any{
			"profile":                config.Profile,
			"mode":                   config.Mode,
			"realtime_push_enabled":  config.RealtimePushEnabled,
			"client_sync_enabled":    config.ClientSyncEnabled,
			"enabled_account_count":  enabledSocketAccountCount(config.Accounts),
			"heartbeat_seconds":      config.HeartbeatSeconds,
			"idle_timeout_seconds":   config.IdleTimeoutSeconds,
			"data_plane_integration": "socket server manager",
		},
	}
}

func eventFromSocketRuntime(config SocketAlarmConfig, eventType string, status RunStatus, payload string, errMessage string, summary map[string]any) PageConfigEvent {
	if summary == nil {
		summary = map[string]any{}
	}
	summary["profile"] = config.Profile
	summary["mode"] = config.Mode
	summary["listen_ip"] = firstNonEmpty(config.ListenIP, "0.0.0.0")
	summary["listen_port"] = config.ListenPort
	summary["realtime_push_enabled"] = config.RealtimePushEnabled
	summary["client_sync_enabled"] = config.ClientSyncEnabled
	return PageConfigEvent{
		Capability:         "socket",
		OwnerCode:          config.Key,
		TargetKey:          config.Key,
		EventType:          eventType,
		Status:             status,
		ArtifactType:       EventArtifactMessage,
		ArtifactName:       fmt.Sprintf("%s %s", config.Profile, eventType),
		ArtifactPath:       socketListenArtifactPath(config),
		Payload:            payload,
		PayloadContentType: "text/plain; charset=utf-8",
		ErrorMessage:       errMessage,
		Summary:            summary,
	}
}

func eventFromSocketFileSync(config SocketAlarmConfig, reqID string, artifactName string, remotePaths []string, failures []string, payload string) PageConfigEvent {
	status := RunStatusSuccess
	errMessage := ""
	if len(failures) > 0 {
		status = RunStatusFailed
		errMessage = strings.Join(failures, "; ")
	}
	return PageConfigEvent{
		Capability:         "socket",
		OwnerCode:          config.Key,
		TargetKey:          config.Key,
		EventType:          "file_sync",
		Status:             status,
		ArtifactType:       EventArtifactMessage,
		ArtifactName:       artifactName,
		ArtifactPath:       socketListenArtifactPath(config),
		Payload:            payload,
		PayloadContentType: "text/plain; charset=utf-8",
		ErrorMessage:       errMessage,
		Summary: map[string]any{
			"profile":      config.Profile,
			"request_id":   reqID,
			"artifact":     artifactName,
			"remote_paths": remotePaths,
			"failures":     failures,
		},
	}
}

func eventFromAPITest(config APIConfig) PageConfigEvent {
	status := RunStatusSuccess
	if !config.Enabled {
		status = RunStatusTerminated
	}
	payload := mustMarshalEventPayload(map[string]any{
		"method":            config.Method,
		"path":              config.Path,
		"old_system":        config.OldSystemSupported,
		"current_supported": config.CurrentSupported,
		"response_contract": config.ResponseContract,
	})
	return PageConfigEvent{
		Capability:         "api",
		OwnerCode:          config.Key,
		TargetKey:          config.Key,
		EventType:          "contract_check",
		Status:             status,
		ArtifactType:       EventArtifactJSON,
		ArtifactName:       fmt.Sprintf("%s 接口检查", config.Name),
		ArtifactPath:       config.Path,
		Payload:            payload,
		PayloadContentType: "application/json; charset=utf-8",
		Summary: map[string]any{
			"method":               config.Method,
			"kind":                 config.Kind,
			"data_type":            config.DataType,
			"old_system_supported": config.OldSystemSupported,
			"current_supported":    config.CurrentSupported,
			"source":               config.Source,
		},
	}
}

func probeDeliveryTarget(ctx context.Context, target DeliveryTarget) DeliveryConnectionResult {
	start := time.Now()
	result := DeliveryConnectionResult{
		Protocol: target.Protocol,
		Host:     strings.TrimSpace(target.Host),
		Port:     target.Port,
	}
	if result.Host == "" {
		result.Message = "delivery target host is empty"
		return result
	}

	timeout := time.Duration(target.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if timeout > 30*time.Second {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addr := net.JoinHostPort(result.Host, strconv.Itoa(result.Port))
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		result.LatencyMs = time.Since(start).Milliseconds()
		result.Message = scrubSecret(err.Error(), target.Credential)
		return result
	}
	result.TCPReachable = true

	switch target.Protocol {
	case DeliveryProtocolFTP:
		result.AuthProbeSupported = target.AuthMode == "" || target.AuthMode == DeliveryAuthPassword
		if !result.AuthProbeSupported {
			result.Success = true
			result.Message = "FTP TCP reachable; private-key auth probe is not applicable"
			_ = conn.Close()
			break
		}
		passed, msg := probeFTPPasswordAuth(conn, target, timeout)
		result.AuthProbePassed = &passed
		result.Success = passed
		result.Message = scrubSecret(msg, target.Credential)
	case DeliveryProtocolSFTP:
		_ = conn.Close()
		if target.AuthMode == DeliveryAuthPrivateKey {
			result.AuthProbeSupported = false
			result.Success = true
			result.Message = "SFTP TCP reachable; private-key auth validation is not supported by this probe"
			break
		}
		result.AuthProbeSupported = true
		passed, msg := probeSFTPPasswordAuth(ctx, addr, target, timeout)
		result.AuthProbePassed = &passed
		result.Success = passed
		result.Message = scrubSecret(msg, target.Credential)
	default:
		_ = conn.Close()
		result.Success = true
		result.Message = fmt.Sprintf("protocol %q has no auth probe; TCP reachability verified", target.Protocol)
	}
	result.LatencyMs = time.Since(start).Milliseconds()
	return result
}

func probeFTPPasswordAuth(conn net.Conn, target DeliveryTarget, timeout time.Duration) (bool, string) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	tc := textproto.NewConn(conn)
	defer tc.Close()
	if _, _, err := tc.ReadResponse(2); err != nil {
		return false, fmt.Sprintf("FTP banner read failed: %s", err.Error())
	}
	if _, err := tc.Cmd("USER %s", target.Username); err != nil {
		return false, fmt.Sprintf("FTP USER write failed: %s", err.Error())
	}
	userCode, _, err := tc.ReadResponse(0)
	if err != nil {
		return false, fmt.Sprintf("FTP USER response read failed: %s", err.Error())
	}
	if userCode == 230 {
		return true, "FTP USER accepted without password"
	}
	if userCode != 331 {
		return false, fmt.Sprintf("FTP USER rejected: code %d", userCode)
	}
	if _, err := tc.Cmd("PASS %s", target.Credential); err != nil {
		return false, fmt.Sprintf("FTP PASS write failed: %s", err.Error())
	}
	passCode, passMsg, err := tc.ReadResponse(0)
	if err != nil {
		return false, fmt.Sprintf("FTP PASS response read failed: %s", err.Error())
	}
	if passCode >= 200 && passCode < 300 {
		return true, fmt.Sprintf("FTP authenticated successfully (code %d)", passCode)
	}
	return false, fmt.Sprintf("FTP auth rejected: code %d %s", passCode, passMsg)
}

func probeSFTPPasswordAuth(ctx context.Context, addr string, target DeliveryTarget, timeout time.Duration) (bool, string) {
	cfg := &ssh.ClientConfig{
		User:            target.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(target.Credential)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false, fmt.Sprintf("SFTP dial failed: %s", err.Error())
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		_ = conn.Close()
		return false, fmt.Sprintf("SFTP auth failed: %s", err.Error())
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	if err := client.Close(); err != nil {
		return false, fmt.Sprintf("SFTP close failed: %s", err.Error())
	}
	return true, "SFTP authenticated successfully"
}

func sampleSNMPPayload() string {
	values := map[string]string{
		"notificationID":        "920188",
		"alarmUniqueId":         "A0188",
		"notificationType":      "1",
		"eventTime":             "1785830400000",
		"equipmentSDN":          "867294050000001",
		"equipmentName":         "Site-A",
		"equipmentClass":        "LTE",
		"objectSDN":             "SubNetwork=OMC,ManagedElement=867294050000001",
		"objectInstanceName":    "Cell-1",
		"objectClass":           "Cell",
		"additionalText":        "Cell Unavailable",
		"deviceVendorOUI":       "Baicells",
		"specificProblemID":     "10001",
		"specificProblem":       "Cell Unavailable",
		"alarmType":             "COMM",
		"perceivedSeverity":     "major",
		"probableCause":         "Radio link unavailable",
		"additionalInformation": "carrier=LTE",
	}
	var b strings.Builder
	for _, field := range defaultSNMPAlarmFields() {
		b.WriteString(field.OID)
		b.WriteString(" = ")
		b.WriteString(values[field.Field])
		b.WriteByte('\n')
	}
	return b.String()
}

func sampleSocketPayload(config SocketAlarmConfig) string {
	if config.Profile == "CUCC" {
		return "syncAlarmReq;reqId=20260804170000;alarmSeq=920176\nackSyncAlarmMsg;reqId=20260804170000;result=0;alarmSeq=920176;count=12\nrealTimeAlarm;alarmSeq=920188;alarmStatus=1;alarmType=communicationsAlarm;origSeverity=major;eventTime=20260804170000;alarmId=10001;specificProblemID=10001;specificProblem=Cell Unavailable;neUID=867294050000001;neName=Site-A;neType=LTE;objectUID=Cell-1;objectName=Cell-1;objectType=Cell;addInfo=carrier=LTE;omcReceivedTime=20260804170002;omcUID=OMC_GD_01"
	}
	return "{\n  \"msgType\": 10,\n  \"alarmSequenceId\": 920188,\n  \"alarmStatus\": \"1\",\n  \"alarmType\": \"communicationsAlarm\",\n  \"origSeverity\": \"major\",\n  \"eventTime\": \"2026-08-04 17:00:00\",\n  \"alarmId\": \"10001\",\n  \"specificProblemID\": \"10001\",\n  \"specificProblem\": \"Cell Unavailable\",\n  \"neDn\": \"867294050000001\",\n  \"neName\": \"Site-A\",\n  \"neType\": \"LTE\",\n  \"objectDn\": \"Cell-1\",\n  \"objectName\": \"Cell-1\",\n  \"objectType\": \"Cell\",\n  \"addInfo\": \"carrier=LTE\",\n  \"omcReceivedTime\": \"20260804170002\",\n  \"omcUID\": \"OMC_GD_01\"\n}"
}

func deliveryArtifactPath(target DeliveryTarget) string {
	return fmt.Sprintf("%s://%s:%d%s",
		strings.ToLower(string(target.Protocol)),
		firstNonEmpty(target.Host, "-"),
		target.Port,
		firstNonEmpty(target.RemoteRoot, "/"),
	)
}

func socketListenArtifactPath(config SocketAlarmConfig) string {
	return fmt.Sprintf("socket://%s:%d", firstNonEmpty(config.ListenIP, "0.0.0.0"), config.ListenPort)
}

func enabledSocketAccountCount(accounts []SocketAccount) int {
	total := 0
	for _, account := range accounts {
		if account.Enabled {
			total++
		}
	}
	return total
}

func errorMessageWhenFailed(success bool, message string) string {
	if success {
		return ""
	}
	return message
}

func mustMarshalEventPayload(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func scrubSecret(value string, secrets ...string) string {
	out := value
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		if secret == "" || len(secret) < 4 {
			continue
		}
		out = strings.ReplaceAll(out, secret, "***")
	}
	return out
}
