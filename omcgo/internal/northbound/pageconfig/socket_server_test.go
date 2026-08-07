package pageconfig

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

func TestSocketAlarmServerAcceptsCUCCLoginHeartbeatAndRealtimePush(t *testing.T) {
	repo := newFakeRepository()
	repo.socketConfigs = []SocketAlarmConfig{
		{
			Key:                 "socket-cucc-server",
			Name:                "CUCC Socket 告警服务端",
			Enabled:             true,
			Profile:             "CUCC",
			Mode:                "server",
			ListenIP:            "127.0.0.1",
			ListenPort:          freeTCPPort(t),
			RealtimePushEnabled: true,
			ClientSyncEnabled:   true,
			HeartbeatSeconds:    5,
			IdleTimeoutSeconds:  30,
			Accounts: []SocketAccount{
				{Key: "cucc-msg", Enabled: true, Username: "oss", Type: "msg", Credential: "secret"},
			},
		},
	}
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)
	bus := event.NewChannelEventBus(16, zap.NewNop())
	manager := NewSocketAlarmServerManager(svc, bus, zap.NewNop())
	manager.SetReloadInterval(time.Hour)
	require.NoError(t, manager.Start(context.Background()))
	defer func() { require.NoError(t, manager.Stop()) }()

	addr, ok := manager.ListenerAddr("socket-cucc-server")
	require.True(t, ok)
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.Write(mustSocketFrame(t, "CUCC", cuccMsgLogin, socketMessageFormatString, []byte("reqLoginAlarm;user=oss;key=secret;type=msg")))
	require.NoError(t, err)
	loginAck, err := readSocketFrame(conn, "CUCC")
	require.NoError(t, err)
	require.Equal(t, cuccMsgLoginAck, loginAck.MessageType)
	require.Contains(t, string(loginAck.Body), "result=succ")

	_, err = conn.Write(mustSocketFrame(t, "CUCC", cuccMsgHeartbeat, socketMessageFormatString, []byte("reqHeartBeat;reqId=hb-1")))
	require.NoError(t, err)
	heartbeatAck, err := readSocketFrame(conn, "CUCC")
	require.NoError(t, err)
	require.Equal(t, cuccMsgHeartbeatAck, heartbeatAck.MessageType)
	require.Contains(t, string(heartbeatAck.Body), "reqId=hb-1")

	alarm := model.Alarm{
		ID:              uuid.New(),
		DeviceSN:        "867294050000001",
		Carrier:         model.CarrierCUCC,
		Severity:        model.AlarmMajor,
		AlarmType:       "COMM",
		AlarmIdentifier: "A0188",
		Description:     "Cell Unavailable",
		Status:          model.AlarmActive,
		RaisedAt:        time.Now(),
		AdditionalInfo:  map[string]string{"alarmSeq": "920188"},
	}
	evt, err := event.NewEvent(event.SubjectAlarmRaised, alarm)
	require.NoError(t, err)
	require.NoError(t, bus.Publish(context.Background(), event.SubjectAlarmRaised, evt))

	pushFrame, err := readSocketFrame(conn, "CUCC")
	require.NoError(t, err)
	require.Equal(t, cuccMsgRealtimeAlarm, pushFrame.MessageType)
	pushBody := string(pushFrame.Body)
	require.Contains(t, pushBody, "realTimeAlarm")
	require.Contains(t, pushBody, "alarmSeq=920188")
	require.Contains(t, pushBody, "specificProblem=Cell Unavailable")

	require.Eventually(t, func() bool {
		for _, item := range repo.events {
			if item.Capability == "socket" && item.EventType == "alarm_push" && item.Status == RunStatusSuccess &&
				strings.Contains(item.Payload, "alarmSeq=920188") {
				return true
			}
		}
		return false
	}, time.Second, 10*time.Millisecond)

	_, err = conn.Write(mustSocketFrame(t, "CUCC", cuccMsgSyncAlarm, socketMessageFormatString, []byte("syncAlarmReq;reqId=1;alarmSeq=0")))
	require.NoError(t, err)
	syncAck, err := readSocketFrame(conn, "CUCC")
	require.NoError(t, err)
	require.Equal(t, cuccMsgSyncAlarmAck, syncAck.MessageType)
	require.Contains(t, string(syncAck.Body), "reqId=1")
	require.Contains(t, string(syncAck.Body), "count=1")

	replayFrame, err := readSocketFrame(conn, "CUCC")
	require.NoError(t, err)
	require.Equal(t, cuccMsgRealtimeAlarm, replayFrame.MessageType)
	require.Contains(t, string(replayFrame.Body), "alarmSeq=920188")
}

func TestSocketAlarmServerReloadsImmediatelyWhenConfigChanges(t *testing.T) {
	port := freeTCPPort(t)
	config := SocketAlarmConfig{
		Key:                 "socket-cucc-server",
		Name:                "CUCC Socket 告警服务端",
		Enabled:             false,
		Profile:             "CUCC",
		Mode:                "server",
		ListenIP:            "127.0.0.1",
		ListenPort:          port,
		MaxClients:          5,
		RealtimePushEnabled: true,
		ClientSyncEnabled:   true,
		HeartbeatSeconds:    5,
		HeartbeatTimes:      1,
		IdleTimeoutSeconds:  10,
		Accounts: []SocketAccount{
			{Key: "cucc-msg", Enabled: true, Username: "oss", Type: "msg", Credential: "secret"},
		},
	}
	repo := newFakeRepository()
	repo.socketConfigs = []SocketAlarmConfig{config}
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)
	manager := NewSocketAlarmServerManager(svc, nil, zap.NewNop())
	manager.SetReloadInterval(time.Hour)
	svc.RegisterSocketConfigChangeListener(manager.RequestReload)
	require.NoError(t, manager.Start(context.Background()))
	defer func() { require.NoError(t, manager.Stop()) }()

	_, ok := manager.ListenerAddr("socket-cucc-server")
	require.False(t, ok)

	config.Enabled = true
	_, err := svc.UpdateSocketAlarmConfig(context.Background(), config.Key, config)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		addr, ok := manager.ListenerAddr("socket-cucc-server")
		if !ok {
			return false
		}
		_, gotPort, err := net.SplitHostPort(addr)
		return err == nil && gotPort == strconv.Itoa(port)
	}, time.Second, 10*time.Millisecond)

	conn, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	require.NoError(t, err)
	defer conn.Close()
	require.Eventually(t, func() bool {
		return manager.sessionCountForConfig(config.Key) == 1
	}, time.Second, 10*time.Millisecond)

	config.RealtimePushEnabled = false
	_, err = svc.UpdateSocketAlarmConfig(context.Background(), config.Key, config)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return manager.sessionCountForConfig(config.Key) == 0
	}, time.Second, 10*time.Millisecond)
	_, ok = manager.ListenerAddr("socket-cucc-server")
	require.True(t, ok)

	config.Enabled = false
	_, err = svc.UpdateSocketAlarmConfig(context.Background(), config.Key, config)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		_, ok := manager.ListenerAddr("socket-cucc-server")
		return !ok
	}, time.Second, 10*time.Millisecond)
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func mustSocketFrame(t *testing.T, profile string, messageType int, messageFormat int, body []byte) []byte {
	t.Helper()
	frame, err := encodeSocketFrame(profile, messageType, messageFormat, body)
	require.NoError(t, err)
	return frame
}
