package pageconfig

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	g "github.com/gosnmp/gosnmp"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/core/model"
	nbsnmp "github.com/omcgo/omcgo/internal/northbound/snmp"
)

func TestSNMPMIBAgentRespondsToV2GetAndWalk(t *testing.T) {
	deviceName := "Site-A"
	technology := "ENB"
	source := "CELL_1"
	eventType := "communicationsAlarm"
	location := "Cell-1"
	probableCause := "Link Failure"
	raisedAt := time.Date(2026, 7, 28, 10, 20, 30, 0, time.Local)
	store := &snmpAgentAlarmStore{alarms: []model.Alarm{{
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
		ProbableCause:   &probableCause,
		AdditionalInfo: map[string]string{
			"notificationID": "123456",
			"pci":            "123",
		},
	}}}
	svc := NewService(NewDefaultCatalog())
	svc.SetAlarmStore(store)
	server := newSNMPMIBServer(svc, snmpMIBServerConfig{
		ListenIP:    "127.0.0.1",
		ListenPort:  0,
		Communities: []string{"baicells"},
	}, nil)
	require.NoError(t, server.Start(context.Background()))
	defer server.Stop()

	host, portText, err := net.SplitHostPort(server.Address())
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)
	client := &g.GoSNMP{
		Target:    host,
		Port:      uint16(port),
		Version:   g.Version2c,
		Community: "baicells",
		Timeout:   time.Second,
		Retries:   0,
	}
	require.NoError(t, client.Connect())
	defer client.Conn.Close()

	getResp, err := client.Get([]string{OIDSpecificProblemValue + ".123456"})
	require.NoError(t, err)
	require.Len(t, getResp.Variables, 1)
	require.Equal(t, OIDSpecificProblemValue+".123456", trimOID(getResp.Variables[0].Name))
	require.Equal(t, "Backhaul Link Down", snmpStringValue(getResp.Variables[0].Value))

	walkRows, err := client.WalkAll(snmpAlarmEntryRootOID())
	require.NoError(t, err)
	require.Len(t, walkRows, 18)
	require.Equal(t, OIDNotificationIDValue+".123456", trimOID(walkRows[0].Name))
	require.Equal(t, 123456, walkRows[0].Value)
	require.Equal(t, OIDPerceivedSeverityValue+".123456", trimOID(walkRows[15].Name))
	require.Equal(t, "major", snmpStringValue(walkRows[15].Value))
}

func TestSNMPMIBAgentRespondsToV3GetAndWalk(t *testing.T) {
	deviceName := "Site-A"
	technology := "ENB"
	source := "CELL_1"
	eventType := "communicationsAlarm"
	location := "Cell-1"
	probableCause := "Link Failure"
	raisedAt := time.Date(2026, 7, 28, 10, 20, 30, 0, time.Local)
	store := &snmpAgentAlarmStore{alarms: []model.Alarm{{
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
		ProbableCause:   &probableCause,
		AdditionalInfo: map[string]string{
			"notificationID": "123456",
			"pci":            "123",
		},
	}}}
	svc := NewService(NewDefaultCatalog())
	svc.SetAlarmStore(store)
	server := newSNMPMIBServer(svc, snmpMIBServerConfig{
		ListenIP:   "127.0.0.1",
		ListenPort: 0,
		Users: []snmpMIBUser{{
			Username:       "secBaiV3",
			AuthProtocol:   "SHA",
			AuthCredential: "AuthPassword",
			PrivProtocol:   "DES",
			PrivCredential: "PrivPassword",
		}},
	}, zap.NewExample())
	require.NoError(t, server.Start(context.Background()))
	defer server.Stop()

	host, portText, err := net.SplitHostPort(server.Address())
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)
	client := &g.GoSNMP{
		Target:        host,
		Port:          uint16(port),
		Version:       g.Version3,
		SecurityModel: g.UserSecurityModel,
		MsgFlags:      g.AuthPriv,
		SecurityParameters: &g.UsmSecurityParameters{
			UserName:                 "secBaiV3",
			AuthenticationProtocol:   g.SHA,
			AuthenticationPassphrase: "AuthPassword",
			PrivacyProtocol:          g.DES,
			PrivacyPassphrase:        "PrivPassword",
		},
		Timeout: time.Second,
		Retries: 0,
	}
	require.NoError(t, client.Connect())
	defer client.Conn.Close()

	getResp, err := client.Get([]string{OIDSpecificProblemValue + ".123456"})
	require.NoError(t, err)
	require.Len(t, getResp.Variables, 1)
	require.Equal(t, OIDSpecificProblemValue+".123456", trimOID(getResp.Variables[0].Name))
	require.Equal(t, "Backhaul Link Down", snmpStringValue(getResp.Variables[0].Value))

	walkRows, err := client.WalkAll(snmpAlarmEntryRootOID())
	require.NoError(t, err)
	require.Len(t, walkRows, 18)
	require.Equal(t, OIDNotificationIDValue+".123456", trimOID(walkRows[0].Name))
	require.Equal(t, 123456, walkRows[0].Value)
	require.Equal(t, OIDPerceivedSeverityValue+".123456", trimOID(walkRows[15].Name))
	require.Equal(t, "major", snmpStringValue(walkRows[15].Value))
}

func TestSNMPMIBAgentRespondsToV3NoAuthNoPrivGet(t *testing.T) {
	deviceName := "Site-A"
	technology := "ENB"
	source := "CELL_1"
	eventType := "communicationsAlarm"
	location := "Cell-1"
	probableCause := "Link Failure"
	raisedAt := time.Date(2026, 7, 28, 10, 20, 30, 0, time.Local)
	store := &snmpAgentAlarmStore{alarms: []model.Alarm{{
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
		ProbableCause:   &probableCause,
		AdditionalInfo: map[string]string{
			"notificationID": "123456",
		},
	}}}
	svc := NewService(NewDefaultCatalog())
	svc.SetAlarmStore(store)
	server := newSNMPMIBServer(svc, snmpMIBServerConfig{
		ListenIP:   "127.0.0.1",
		ListenPort: 0,
		Users: []snmpMIBUser{{
			Username: "noAuthUser",
		}},
	}, zap.NewExample())
	require.NoError(t, server.Start(context.Background()))
	defer server.Stop()

	host, portText, err := net.SplitHostPort(server.Address())
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)
	client := &g.GoSNMP{
		Target:        host,
		Port:          uint16(port),
		Version:       g.Version3,
		SecurityModel: g.UserSecurityModel,
		MsgFlags:      g.NoAuthNoPriv,
		SecurityParameters: &g.UsmSecurityParameters{
			UserName: "noAuthUser",
		},
		Timeout: time.Second,
		Retries: 0,
	}
	require.NoError(t, client.Connect())
	defer client.Conn.Close()

	getResp, err := client.Get([]string{OIDSpecificProblemValue + ".123456"})
	require.NoError(t, err)
	require.Len(t, getResp.Variables, 1)
	require.Equal(t, OIDSpecificProblemValue+".123456", trimOID(getResp.Variables[0].Name))
	require.Equal(t, "Backhaul Link Down", snmpStringValue(getResp.Variables[0].Value))
}

func TestSNMPClearSeverityPolicyAppliesPerTarget(t *testing.T) {
	alarmEvent := &nbsnmp.AlarmEvent{
		AlarmID:      "40123",
		DeviceSerial: "ENB_SN001",
		Severity:     "major",
		AlarmType:    "communicationsAlarm",
		OccurTime:    time.Date(2026, 7, 28, 10, 20, 30, 0, time.Local),
		Extra: map[string]string{
			"notificationID":   "123456",
			"notificationType": "0",
		},
	}

	retainVars, err := nbsnmp.DefaultAlarmMapper().MapAlarmToTrapPDU(snmpAlarmForTarget(SNMPAlarmTarget{ClearSeverityPolicy: "保留原级别"}, alarmEvent))
	require.NoError(t, err)
	require.Equal(t, "major", retainVars[15].Value)

	zeroVars, err := nbsnmp.DefaultAlarmMapper().MapAlarmToTrapPDU(snmpAlarmForTarget(SNMPAlarmTarget{ClearSeverityPolicy: "清除置 0"}, alarmEvent))
	require.NoError(t, err)
	require.Equal(t, "0", zeroVars[15].Value)
}

func TestSNMPRuntimeProtocolMappingCoversFieldDocumentAlgorithms(t *testing.T) {
	require.Equal(t, nbsnmp.AuthSHA224, runtimeSNMPAuthProtocol("SHA224"))
	require.Equal(t, nbsnmp.AuthSHA256, runtimeSNMPAuthProtocol("SHA256"))
	require.Equal(t, nbsnmp.AuthSHA384, runtimeSNMPAuthProtocol("SHA384"))
	require.Equal(t, nbsnmp.AuthSHA512, runtimeSNMPAuthProtocol("SHA512"))
	require.Equal(t, nbsnmp.PrivAES, runtimeSNMPPrivProtocol("AES128"))
	require.Equal(t, nbsnmp.PrivAES192, runtimeSNMPPrivProtocol("AES192"))
	require.Equal(t, nbsnmp.PrivAES256, runtimeSNMPPrivProtocol("AES256"))
}

func trimOID(oid string) string {
	if len(oid) > 0 && oid[0] == '.' {
		return oid[1:]
	}
	return oid
}

func snmpStringValue(value any) string {
	if raw, ok := value.([]byte); ok {
		return string(raw)
	}
	return fmt.Sprint(value)
}

type snmpAgentAlarmStore struct {
	alarms []model.Alarm
}

func (s *snmpAgentAlarmStore) SaveActive(context.Context, *model.Alarm) error { return nil }
func (s *snmpAgentAlarmStore) GetActiveByID(context.Context, uuid.UUID) (*model.Alarm, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *snmpAgentAlarmStore) GetHistoryByID(context.Context, uuid.UUID) (*model.Alarm, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *snmpAgentAlarmStore) GetActiveByDeviceAndIdentifier(context.Context, string, string) (*model.Alarm, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *snmpAgentAlarmStore) GetActiveByDeviceSN(context.Context, string) ([]*model.Alarm, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *snmpAgentAlarmStore) UpdateActive(context.Context, *model.Alarm) error { return nil }
func (s *snmpAgentAlarmStore) RemoveActive(context.Context, uuid.UUID) error    { return nil }
func (s *snmpAgentAlarmStore) ListActive(_ context.Context, filter alarm.AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = len(s.alarms)
	}
	start := (page - 1) * pageSize
	if start >= len(s.alarms) {
		return model.NewListResponse([]model.Alarm{}, int64(len(s.alarms)), page, pageSize), nil
	}
	end := start + pageSize
	if end > len(s.alarms) {
		end = len(s.alarms)
	}
	return model.NewListResponse(s.alarms[start:end], int64(len(s.alarms)), page, pageSize), nil
}
func (s *snmpAgentAlarmStore) Archive(context.Context, *model.Alarm) error { return nil }
func (s *snmpAgentAlarmStore) ListHistory(context.Context, alarm.AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	return model.NewListResponse([]model.Alarm{}, 0, 1, 100), nil
}
func (s *snmpAgentAlarmStore) Statistics(context.Context, alarm.AlarmFilter) (*alarm.AlarmStatistics, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *snmpAgentAlarmStore) HistoryStatistics(context.Context, alarm.AlarmFilter) (*alarm.AlarmStatistics, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *snmpAgentAlarmStore) BatchAcknowledge(context.Context, []uuid.UUID, string, string) error {
	return nil
}
func (s *snmpAgentAlarmStore) BatchUnacknowledge(context.Context, []uuid.UUID) error { return nil }
func (s *snmpAgentAlarmStore) BatchClear(context.Context, []uuid.UUID, string, string) error {
	return nil
}
func (s *snmpAgentAlarmStore) BatchHistoryAcknowledge(context.Context, []uuid.UUID, string, string) error {
	return nil
}
func (s *snmpAgentAlarmStore) BatchHistoryUnacknowledge(context.Context, []uuid.UUID) error {
	return nil
}
func (s *snmpAgentAlarmStore) BatchHistoryDelete(context.Context, []uuid.UUID) error { return nil }
func (s *snmpAgentAlarmStore) MarkRead(context.Context, uuid.UUID) error             { return nil }
