package snmp

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	g "github.com/gosnmp/gosnmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSender is a Sender double for tests. It records every call and lets
// individual tests stub the return value (per-target if desired).
type mockSender struct {
	mu        sync.Mutex
	callCount int
	lastVars  []Variable
	lastTgt   *TrapTarget
	// returnFn lets a test inject error per call site.
	returnFn func(ctx context.Context, target *TrapTarget, vars []Variable) error
}

func (m *mockSender) Send(ctx context.Context, target *TrapTarget, vars []Variable) error {
	m.mu.Lock()
	m.callCount++
	m.lastTgt = target
	m.lastVars = vars
	m.mu.Unlock()
	if m.returnFn != nil {
		return m.returnFn(ctx, target, vars)
	}
	return nil
}

func (m *mockSender) calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

func TestMockSender_Send_Success(t *testing.T) {
	m := &mockSender{}
	tgt := newTestTarget("osr", true)
	vars := []Variable{{OID: OIDAlarmIdentifier, Type: VarTypeOctetString, Value: "A1"}}

	err := m.Send(context.Background(), tgt, vars)
	require.NoError(t, err)
	assert.Equal(t, 1, m.calls())
	assert.Equal(t, "osr", m.lastTgt.OSSName)
	assert.Len(t, m.lastVars, 1)
}

func TestMockSender_Send_ErrorPropagates(t *testing.T) {
	wantErr := errors.New("boom")
	m := &mockSender{
		returnFn: func(ctx context.Context, target *TrapTarget, vars []Variable) error {
			return wantErr
		},
	}
	err := m.Send(context.Background(), newTestTarget("osr", true), []Variable{{OID: "1.2.3"}})
	require.Error(t, err)
	assert.True(t, errors.Is(err, wantErr))
}

func TestGoSNMPSender_Send_Validates(t *testing.T) {
	s := NewGoSNMPSender(nil)

	cases := []struct {
		name string
		tgt  *TrapTarget
		vars []Variable
		want error
	}{
		{
			name: "nil target",
			tgt:  nil,
			vars: []Variable{{OID: "1.2.3"}},
			want: ErrNilTarget,
		},
		{
			name: "empty vars",
			tgt:  newTestTarget("osr", true),
			vars: nil,
			want: ErrNoVariables,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := s.Send(context.Background(), tc.tgt, tc.vars)
			require.Error(t, err)
			assert.True(t, errors.Is(err, tc.want), "got %v, want %v", err, tc.want)
		})
	}
}

func TestValidateTarget(t *testing.T) {
	cases := []struct {
		name    string
		target  TrapTarget
		wantErr bool
	}{
		{
			name: "valid v2c",
			target: TrapTarget{
				Host: "h", Port: 162, Version: VersionV2c, Community: "public",
			},
		},
		{
			name: "valid v3",
			target: TrapTarget{
				Host: "h", Port: 162, Version: VersionV3, Username: "u",
			},
		},
		{name: "missing host", target: TrapTarget{Port: 162, Version: VersionV2c, Community: "x"}, wantErr: true},
		{name: "missing port", target: TrapTarget{Host: "h", Version: VersionV2c, Community: "x"}, wantErr: true},
		{name: "invalid version", target: TrapTarget{Host: "h", Port: 162, Version: "v1"}, wantErr: true},
		{name: "v2c missing community", target: TrapTarget{Host: "h", Port: 162, Version: VersionV2c}, wantErr: true},
		{name: "v3 missing username", target: TrapTarget{Host: "h", Port: 162, Version: VersionV3}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTarget(&tc.target)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBuildPDUs_PrependsTrapEnvelope(t *testing.T) {
	vars := []Variable{
		{OID: OIDAlarmIdentifier, Type: VarTypeOctetString, Value: "A1"},
		{OID: OIDAlarmSeverity, Type: VarTypeInteger, Value: 1},
	}
	pdus := buildPDUs(vars)
	require.Len(t, pdus, 4, "expect 2 envelope PDUs + 2 user PDUs")
	assert.Equal(t, OIDSysUpTime, pdus[0].Name)
	assert.Equal(t, OIDSnmpTrapID, pdus[1].Name)
	assert.Equal(t, OIDAlarmIdentifier, pdus[2].Name)
	assert.Equal(t, OIDAlarmSeverity, pdus[3].Name)
}

func TestBuildPDUs_TypeMapping(t *testing.T) {
	vars := []Variable{
		{OID: "1.2.3", Type: VarTypeOctetString, Value: "x"},
		{OID: "1.2.4", Type: VarTypeInteger, Value: 1},
		{OID: "1.2.5", Type: VarTypeCounter32, Value: uint32(2)},
		{OID: "1.2.6", Type: VarTypeCounter64, Value: uint64(3)},
		{OID: "1.2.7", Type: VarTypeTimeTicks, Value: uint32(4)},
		{OID: "1.2.8", Type: VarTypeIPAddress, Value: "127.0.0.1"},
		{OID: "1.2.9", Type: VarTypeObjectID, Value: "1.3.6.1"},
		{OID: "1.2.10", Type: VarType("unknown"), Value: "fallback"}, // unknown → OctetString
	}
	pdus := buildPDUs(vars)
	require.Len(t, pdus, 2+len(vars))
	// last entry must have downgraded to OctetString rather than been dropped
	last := pdus[len(pdus)-1]
	assert.Equal(t, "1.2.10", last.Name)
	assert.Equal(t, "fallback", last.Value)
}

func TestGoSNMPSender_Send_DeliversV2TrapToReceiver(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)
	conn, err := net.ListenUDP("udp", addr)
	require.NoError(t, err)
	defer conn.Close()
	_, portText, err := net.SplitHostPort(conn.LocalAddr().String())
	require.NoError(t, err)
	port, err := net.LookupPort("udp", portText)
	require.NoError(t, err)

	received := make(chan *g.SnmpPacket, 1)
	go func() {
		buf := make([]byte, 4096)
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		packet, err := g.Default.UnmarshalTrap(buf[:n], false)
		if err == nil {
			received <- packet
		}
	}()

	sender := NewGoSNMPSender(nil)
	target := &TrapTarget{
		ID:        "receiver-test",
		OSSName:   "receiver-test",
		Host:      "127.0.0.1",
		Port:      uint16(port),
		Version:   VersionV2c,
		Community: "public",
		Timeout:   time.Second,
	}
	vars := []Variable{
		{OID: OIDNotificationID, Type: VarTypeInteger, Value: 920188},
		{OID: OIDAlarmUniqueID, Type: VarTypeOctetString, Value: "A0188"},
	}
	require.NoError(t, sender.Send(context.Background(), target, vars))

	select {
	case packet := <-received:
		require.NotNil(t, packet)
		require.GreaterOrEqual(t, len(packet.Variables), 4)
		assert.Equal(t, OIDNotificationID, strings.TrimPrefix(packet.Variables[2].Name, "."))
		assert.Equal(t, OIDAlarmUniqueID, strings.TrimPrefix(packet.Variables[3].Name, "."))
	case <-time.After(3 * time.Second):
		t.Fatal("did not receive SNMP trap")
	}
}

func TestDefaultMapper_FieldOrderAndContent(t *testing.T) {
	mapper := DefaultAlarmMapper()
	now := time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)

	cases := []struct {
		name      string
		alarm     *AlarmEvent
		wantErr   bool
		assertVar func(t *testing.T, vars []Variable)
	}{
		{
			name:    "nil alarm rejected",
			alarm:   nil,
			wantErr: true,
		},
		{
			name:    "missing alarm id",
			alarm:   &AlarmEvent{DeviceSerial: "D1", OccurTime: now},
			wantErr: true,
		},
		{
			name:    "missing device serial",
			alarm:   &AlarmEvent{AlarmID: "A1", OccurTime: now},
			wantErr: true,
		},
		{
			name: "minimal alarm produces 18 vars in MIB order",
			alarm: &AlarmEvent{
				AlarmID: "A1", DeviceSerial: "D1", Severity: "critical", OccurTime: now,
			},
			assertVar: func(t *testing.T, vars []Variable) {
				require.Len(t, vars, 18)
				assert.Equal(t, OIDNotificationID, vars[0].OID)
				assert.Equal(t, 1, vars[0].Value)
				assert.Equal(t, OIDAlarmUniqueID, vars[1].OID)
				assert.Equal(t, "A1", vars[1].Value)
				assert.Equal(t, OIDNotificationType, vars[2].OID)
				assert.Equal(t, "1", vars[2].Value)
				assert.Equal(t, OIDEventTime, vars[3].OID)
				assert.Equal(t, OIDEquipmentSDN, vars[4].OID)
				assert.Equal(t, "D1", vars[4].Value)
				assert.Equal(t, OIDPerceivedSeverity, vars[15].OID)
				assert.Equal(t, "critical", vars[15].Value)
			},
		},
		{
			name: "full alarm maps type and carrier into MIB fields",
			alarm: &AlarmEvent{
				AlarmID: "A1", DeviceSerial: "D1", Severity: "major",
				AlarmType: "POWER_FAIL", Carrier: "cmcc", OccurTime: now,
			},
			assertVar: func(t *testing.T, vars []Variable) {
				require.Len(t, vars, 18)
				assert.Equal(t, OIDAlarmType, vars[14].OID)
				assert.Equal(t, "POWER_FAIL", vars[14].Value)
				assert.Equal(t, OIDAdditionalInformation, vars[17].OID)
				assert.Equal(t, "cmcc", vars[17].Value)
				assert.Equal(t, "major", vars[15].Value)
			},
		},
		{
			name: "unknown severity maps to unknown string",
			alarm: &AlarmEvent{
				AlarmID: "A1", DeviceSerial: "D1", Severity: "FOO",
				OccurTime: now,
			},
			assertVar: func(t *testing.T, vars []Variable) {
				assert.Equal(t, "unknown", vars[15].Value)
			},
		},
		{
			name: "negative time clamps to 0",
			alarm: &AlarmEvent{
				AlarmID: "A1", DeviceSerial: "D1",
				OccurTime: time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			assertVar: func(t *testing.T, vars []Variable) {
				assert.Equal(t, uint64(0), vars[3].Value)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vars, err := mapper.MapAlarmToTrapPDU(tc.alarm)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tc.assertVar(t, vars)
		})
	}
}

func TestSeverityToInt(t *testing.T) {
	cases := map[string]int{
		"critical":      1,
		"CRITICAL":      1,
		" major ":       2,
		"minor":         3,
		"warning":       4,
		"cleared":       5,
		"indeterminate": 5,
		"":              5,
		"unknown":       5,
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			assert.Equal(t, want, severityToInt(in))
		})
	}
}

func TestGosnmpVersion(t *testing.T) {
	assert.NotPanics(t, func() {
		_ = gosnmpVersion(VersionV2c)
		_ = gosnmpVersion(VersionV3)
		_ = gosnmpVersion(SNMPVersion("bogus")) // falls back to v2c
	})
}
