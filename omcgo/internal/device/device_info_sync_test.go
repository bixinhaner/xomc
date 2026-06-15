package device

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type testCarrier struct{}

func (testCarrier) Code() model.CarrierCode { return model.CarrierCMCC }
func (testCarrier) Name() string            { return "test" }
func (testCarrier) SupportedTechnologies() []model.Technology {
	return []model.Technology{model.TechLTE, model.TechNR}
}
func (testCarrier) DefaultDataModelVersions(model.Technology) []string                    { return nil }
func (testCarrier) KnownOUIProductClasses(model.Technology) []carrier.OUIProductClassInfo { return nil }
func (testCarrier) MapParameterToUnified(carrierPath string) string                       { return carrierPath }
func (testCarrier) MapUnifiedToParameter(unifiedName string) string                       { return unifiedName }
func (testCarrier) ProvisioningTemplates(model.Technology) []*carrier.ProvisionTemplate   { return nil }
func (testCarrier) AlarmSeverityMapping(string) model.AlarmSeverity                       { return 0 }
func (testCarrier) ValidateParameter(string, string) error                                { return nil }
func (testCarrier) GetInfoParamMapping(model.Technology) map[string]string                { return nil }
func (testCarrier) RFControlPath(model.Technology) string                                 { return "" }
func (testCarrier) SupportsMRType(model.MRType) bool                                      { return true }

type stubDeviceInfoRepo struct {
	updateSyncFields func(ctx context.Context, deviceID uuid.UUID, fields map[string]interface{}) error
}

func (s stubDeviceInfoRepo) GetByDeviceID(context.Context, uuid.UUID) (*DeviceInfo, error) {
	return nil, nil
}

func (s stubDeviceInfoRepo) Create(context.Context, *DeviceInfo) error { return nil }

func (s stubDeviceInfoRepo) UpdateManualFields(context.Context, uuid.UUID, UpdateDeviceInfoRequest, string) error {
	return nil
}

func (s stubDeviceInfoRepo) UpdateSyncFields(ctx context.Context, deviceID uuid.UUID, fields map[string]interface{}) error {
	if s.updateSyncFields != nil {
		return s.updateSyncFields(ctx, deviceID, fields)
	}
	return nil
}

func (s stubDeviceInfoRepo) GetTopologyAttributes(context.Context, uuid.UUID) (map[string]string, error) {
	return nil, nil
}

func (s stubDeviceInfoRepo) ListDevicesWithInfo(context.Context, DeviceFilter) (*model.ListResponse[DeviceWithInfo], error) {
	return nil, nil
}

func (s stubDeviceInfoRepo) GetByIDWithInfo(context.Context, uuid.UUID) (*DeviceWithInfo, error) {
	return nil, nil
}

func (s stubDeviceInfoRepo) ComputeListStats(context.Context, DeviceFilter) (*DeviceListStats, error) {
	return nil, nil
}

type stubDeviceParamRepo struct {
	params []model.DeviceParameter
}

type stubDeviceCoordinateWriter struct {
	updateCoordinates func(ctx context.Context, deviceID uuid.UUID, latitude, longitude float64) error
}

func (s stubDeviceCoordinateWriter) UpdateCoordinates(ctx context.Context, deviceID uuid.UUID, latitude, longitude float64) error {
	if s.updateCoordinates != nil {
		return s.updateCoordinates(ctx, deviceID, latitude, longitude)
	}
	return nil
}

func (s stubDeviceParamRepo) BatchUpsert(context.Context, uuid.UUID, []model.DeviceParameter) error {
	return nil
}
func (s stubDeviceParamRepo) GetByDevice(context.Context, uuid.UUID) ([]model.DeviceParameter, error) {
	return s.params, nil
}
func (s stubDeviceParamRepo) GetByPath(context.Context, uuid.UUID, string) (*model.DeviceParameter, error) {
	return nil, nil
}
func (s stubDeviceParamRepo) DeleteByDevice(context.Context, uuid.UUID) error { return nil }
func (s stubDeviceParamRepo) DeleteByPathPrefix(context.Context, uuid.UUID, string) (int64, error) {
	return 0, nil
}
func (s stubDeviceParamRepo) GetByPathPrefix(context.Context, uuid.UUID, string) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (s stubDeviceParamRepo) CountByPathPrefix(context.Context, uuid.UUID, string) (int, error) {
	return 0, nil
}
func (s stubDeviceParamRepo) SearchByKeyword(context.Context, uuid.UUID, string, int) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (s stubDeviceParamRepo) GetDirectChildLeaves(context.Context, uuid.UUID, string, int, int) ([]model.DeviceParameter, int, error) {
	return nil, 0, nil
}
func (s stubDeviceParamRepo) GetByGroup(context.Context, uuid.UUID, string) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (s stubDeviceParamRepo) GetByFAPInstance(context.Context, uuid.UUID, int) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (s stubDeviceParamRepo) GetByFAPInstanceAndGroup(context.Context, uuid.UUID, int, string) ([]model.DeviceParameter, error) {
	return nil, nil
}

// TestDeriveEnbID covers Phase 3 (设计文档 §4.2 Layer C) — eNodeB ID 派生。
// LTE 28-bit ECI = 20-bit eNB-ID + 8-bit Cell-ID，即 enb_id = eci >> 8。
func TestDeriveEnbID(t *testing.T) {
	tests := []struct {
		name string
		eci  string
		want string
		ok   bool
	}{
		{"valid ECI 654321", "654321", "2555", true}, // 654321 >> 8 = 2555 (0x9FBF1 >> 8 = 0x9FB)
		{"single cell ECI 256", "256", "1", true},
		{"empty input", "", "", false},
		{"non-numeric", "abc", "", false},
		{"zero", "0", "", false},
		{"negative literal", "-1", "", false},
		{"max LTE ECI", "268435455", "1048575", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := deriveEnbID(tt.eci)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

// TestDeriveNetworkModel 验证 TDD / FDD 判定逻辑。
func TestDeriveNetworkModel(t *testing.T) {
	cases := []struct {
		name  string
		paths map[string]string
		want  string
		ok    bool
	}{
		{
			name: "TDD with SubFrameAssignment",
			paths: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.RAN.PHY.TDDFrame.SubFrameAssignment": "2",
			},
			want: "TDD", ok: true,
		},
		{
			name: "FDD subtree present",
			paths: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.RAN.PHY.FDDFrame.SubFrameAssignment": "0",
			},
			want: "FDD", ok: true,
		},
		{
			name:  "no PHY frame path",
			paths: map[string]string{"Device.DeviceInfo.SoftwareVersion": "1.0"},
			want:  "", ok: false,
		},
		{
			name:  "empty paramValues",
			paths: map[string]string{},
			want:  "", ok: false,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := deriveNetworkModel(tt.paths)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

// TestLookupGPSHeight 验证 GPS 高度多 path 优先级（Baicells 拼写优先）。
func TestLookupGPSHeight(t *testing.T) {
	cases := []struct {
		name  string
		paths map[string]string
		want  string
		ok    bool
	}{
		{
			name:  "altidute (Baicells spelling) wins",
			paths: map[string]string{"Device.FAP.GPS.altidute": "170", "Device.FAP.GPS.Altitude": "999"},
			want:  "170", ok: true,
		},
		{
			name:  "fallback to Altitude when altidute missing",
			paths: map[string]string{"Device.FAP.GPS.Altitude": "200"},
			want:  "200", ok: true,
		},
		{
			name:  "fallback to Height when both missing",
			paths: map[string]string{"Device.FAP.GPS.Height": "150"},
			want:  "150", ok: true,
		},
		{
			name:  "empty value is skipped",
			paths: map[string]string{"Device.FAP.GPS.altidute": "", "Device.FAP.GPS.Altitude": "100"},
			want:  "100", ok: true,
		},
		{
			name:  "all missing",
			paths: map[string]string{},
			want:  "", ok: false,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := lookupGPSHeight(tt.paths)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestLookupGPSCoordinates(t *testing.T) {
	cases := []struct {
		name    string
		paths   map[string]string
		wantLat float64
		wantLng float64
		ok      bool
	}{
		{
			name: "decimal degrees",
			paths: map[string]string{
				"Device.FAP.GPS.LockedLatitude":  "39.9042",
				"Device.FAP.GPS.LockedLongitude": "116.4074",
			},
			wantLat: 39.9042,
			wantLng: 116.4074,
			ok:      true,
		},
		{
			name: "microdegrees are normalized",
			paths: map[string]string{
				"Device.FAP.GPS.LockedLatitude":  "39904200",
				"Device.FAP.GPS.LockedLongitude": "116407400",
			},
			wantLat: 39.9042,
			wantLng: 116.4074,
			ok:      true,
		},
		{
			name: "invalid latitude rejects pair",
			paths: map[string]string{
				"Device.FAP.GPS.LockedLatitude":  "abc",
				"Device.FAP.GPS.LockedLongitude": "116.4074",
			},
			ok: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			lat, lng, ok := lookupGPSCoordinates(tt.paths)
			assert.Equal(t, tt.wantLat, lat)
			assert.Equal(t, tt.wantLng, lng)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestLookupWANMAC(t *testing.T) {
	cases := []struct {
		name  string
		paths map[string]string
		want  string
		ok    bool
	}{
		{
			name: "prefer interface marked as WAN by port type",
			paths: map[string]string{
				"Device.Ethernet.Interface.1.MACAddress":                                   "00:11:22:33:44:55",
				"Device.Ethernet.Interface.1.Name":                                         "LAN1",
				"Device.Ethernet.Interface.2.MACAddress":                                   "66:77:88:99:AA:BB",
				"Device.Ethernet.Interface.2.Name":                                         "GE2",
				"Device.Ethernet.Interface.2.IPv4Address.1.PortType":                       "WAN",
				"Device.Ethernet.Interface.2.IPv4Address.1.IPAddress":                      "10.0.0.2",
				"Device.Ethernet.Interface.2.IPv4Address.1.DefaultGateway":                 "10.0.0.1",
				"Device.Ethernet.Interface.2.VlanInterface.1.IPv4Address.1.PortType":       "WAN_VLAN",
				"Device.Ethernet.Interface.2.VlanInterface.1.IPv4Address.1.IPAddress":      "192.168.1.2",
				"Device.Ethernet.Interface.2.VlanInterface.1.IPv4Address.1.DefaultGateway": "192.168.1.1",
			},
			want: "66:77:88:99:AA:BB",
			ok:   true,
		},
		{
			name: "single ethernet interface falls back to its mac",
			paths: map[string]string{
				"Device.Ethernet.Interface.1.MACAddress": "AA:BB:CC:DD:EE:FF",
			},
			want: "AA:BB:CC:DD:EE:FF",
			ok:   true,
		},
		{
			name: "name based WAN hint wins when port type missing",
			paths: map[string]string{
				"Device.Ethernet.Interface.1.MACAddress": "00:00:00:00:00:01",
				"Device.Ethernet.Interface.1.Name":       "LAN1",
				"Device.Ethernet.Interface.2.MACAddress": "00:00:00:00:00:02",
				"Device.Ethernet.Interface.2.UserLabel":  "WAN uplink",
			},
			want: "00:00:00:00:00:02",
			ok:   true,
		},
		{
			name:  "no ethernet mac path",
			paths: map[string]string{"Device.DeviceInfo.SoftwareVersion": "1.0"},
			want:  "",
			ok:    false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := lookupWANMAC(tt.paths)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestInfoSyncer_SyncFromParameters_OnlyNRUsesWANMACTraversal(t *testing.T) {
	deviceID := uuid.New()
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Ethernet.Interface.1.MACAddress", ParameterValue: "00:11:22:33:44:55"},
		{ParameterPath: "Device.Ethernet.Interface.1.Name", ParameterValue: "LAN1"},
		{ParameterPath: "Device.Ethernet.Interface.2.MACAddress", ParameterValue: "66:77:88:99:AA:BB"},
		{ParameterPath: "Device.Ethernet.Interface.2.IPv4Address.1.PortType", ParameterValue: "WAN"},
	}

	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})

	t.Run("nr overrides mac with WAN traversal", func(t *testing.T) {
		infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, _ uuid.UUID, fields map[string]interface{}) error {
			assert.Equal(t, "66:77:88:99:AA:BB", fields["mac"])
			return nil
		}}
		paramRepo := stubDeviceParamRepo{params: params}
		syncer := NewInfoSyncer(infoRepo, paramRepo, nil, registry, zap.NewNop())

		_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechNR)
		assert.NoError(t, err)
	})

	t.Run("lte keeps legacy fixed-path behavior", func(t *testing.T) {
		infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, _ uuid.UUID, fields map[string]interface{}) error {
			_, exists := fields["mac"]
			assert.False(t, exists)
			return nil
		}}
		paramRepo := stubDeviceParamRepo{params: params}
		syncer := NewInfoSyncer(infoRepo, paramRepo, nil, registry, zap.NewNop())

		_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechLTE)
		assert.NoError(t, err)
	})
}

func TestInfoSyncer_SyncFromParameters_BackfillsCoordinates(t *testing.T) {
	deviceID := uuid.New()
	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})

	paramRepo := stubDeviceParamRepo{params: []model.DeviceParameter{
		{ParameterPath: "Device.FAP.GPS.LockedLatitude", ParameterValue: "39904200"},
		{ParameterPath: "Device.FAP.GPS.LockedLongitude", ParameterValue: "116407400"},
	}}

	coordinateWriter := stubDeviceCoordinateWriter{updateCoordinates: func(_ context.Context, gotDeviceID uuid.UUID, latitude, longitude float64) error {
		assert.Equal(t, deviceID, gotDeviceID)
		assert.Equal(t, 39.9042, latitude)
		assert.Equal(t, 116.4074, longitude)
		return nil
	}}

	infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, _ uuid.UUID, fields map[string]interface{}) error {
		assert.NotEmpty(t, fields)
		return nil
	}}

	syncer := NewInfoSyncer(infoRepo, paramRepo, coordinateWriter, registry, zap.NewNop())

	_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechNR)
	assert.NoError(t, err)
}

func TestInfoSyncer_SyncFromParameters_ComputesQuickFieldsWithoutCarrierMapping(t *testing.T) {
	deviceID := uuid.New()
	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})

	paramRepo := stubDeviceParamRepo{params: []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.1.NR.RAN.OpState", ParameterValue: "1"},
	}}

	infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, gotDeviceID uuid.UUID, fields map[string]interface{}) error {
		assert.Equal(t, deviceID, gotDeviceID)
		assert.Equal(t, "normal", fields["cell_status"])
		assert.Equal(t, 1, fields["num_of_cells"])
		return nil
	}}

	syncer := NewInfoSyncer(infoRepo, paramRepo, nil, registry, zap.NewNop())

	_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechNR)
	assert.NoError(t, err)
}

// txPowerCarrier 是带 LTE ReferenceSignalPower→transmit_power 映射的运营商桩，
// 模拟 cmcc/ctcc adapter 的 GetInfoParamMapping 行为，用于 #362 取值校验。
type txPowerCarrier struct{ testCarrier }

func (txPowerCarrier) GetInfoParamMapping(tech model.Technology) map[string]string {
	if tech == model.TechLTE {
		return map[string]string{
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.ReferenceSignalPower": "transmit_power",
		}
	}
	return nil
}

// TestInfoSyncer_SyncFromParameters_TransmitPowerSource 锁定 #362：
// 4G(LTE) 设备的 transmit_power 必须取 carrier adapter 的 ReferenceSignalPower
// （与 LMT 口径一致），且 universalInformMapping 不再用 Capabilities.MaxTxPower
// 覆盖它（删除 MaxTxPower→transmit_power 后，单一权威来源生效）。
func TestInfoSyncer_SyncFromParameters_TransmitPowerSource(t *testing.T) {
	deviceID := uuid.New()
	registry := carrier.NewRegistry()
	registry.Register(txPowerCarrier{})

	const refSignalPath = "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.ReferenceSignalPower"
	const maxTxPath = "Device.Services.FAPService.1.Capabilities.MaxTxPower"

	cases := []struct {
		name   string
		params []model.DeviceParameter
		want   interface{} // nil 表示不应写 transmit_power
	}{
		{
			name: "only ReferenceSignalPower → 取 RS 功率",
			params: []model.DeviceParameter{
				{ParameterPath: refSignalPath, ParameterValue: "18.2"},
			},
			want: "18.2",
		},
		{
			name: "only MaxTxPower → 不再投影到 transmit_power（#362 已删 universal 映射）",
			params: []model.DeviceParameter{
				{ParameterPath: maxTxPath, ParameterValue: "46"},
			},
			want: nil,
		},
		{
			name: "both present → 仍取 ReferenceSignalPower，不被 MaxTxPower 覆盖",
			params: []model.DeviceParameter{
				{ParameterPath: refSignalPath, ParameterValue: "18.2"},
				{ParameterPath: maxTxPath, ParameterValue: "46"},
			},
			want: "18.2",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, _ uuid.UUID, fields map[string]interface{}) error {
				got, exists := fields["transmit_power"]
				if tt.want == nil {
					assert.False(t, exists, "transmit_power 不应被写入（MaxTxPower 不再映射）")
					return nil
				}
				assert.True(t, exists, "transmit_power 应被写入")
				assert.Equal(t, tt.want, got)
				return nil
			}}
			paramRepo := stubDeviceParamRepo{params: tt.params}
			syncer := NewInfoSyncer(infoRepo, paramRepo, nil, registry, zap.NewNop())

			_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechLTE)
			assert.NoError(t, err)
		})
	}
}

// TestUniversalInformMapping_NoTransmitPowerOverride 防回归守卫：#362 删除
// universalInformMapping 里 MaxTxPower→transmit_power 后，该表不应再含任何
// transmit_power 映射目标（避免后人误加回 universal 覆盖）。
func TestUniversalInformMapping_NoTransmitPowerOverride(t *testing.T) {
	for path, col := range universalInformMapping {
		assert.NotEqualf(t, "transmit_power", col,
			"universalInformMapping 不应映射到 transmit_power（#362）；命中 path=%s", path)
	}
}

func TestParseRunTimeToSeconds(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int64
	}{
		{
			name:  "full format with days hours minutes",
			input: "40d 4h 58m",
			want:  40*86400 + 4*3600 + 58*60, // 3482308
		},
		{
			name:  "hours and minutes only",
			input: "4h 58m",
			want:  4*3600 + 58*60, // 17880
		},
		{
			name:  "minutes only",
			input: "58m",
			want:  58 * 60, // 3480
		},
		{
			name:  "full format with seconds",
			input: "1d 2h 3m 4s",
			want:  86400 + 2*3600 + 3*60 + 4, // 93784
		},
		{
			name:  "days only",
			input: "10d",
			want:  10 * 86400, // 864000
		},
		{
			name:  "hours only",
			input: "24h",
			want:  24 * 3600, // 86400
		},
		{
			name:  "seconds only",
			input: "30s",
			want:  30,
		},
		{
			name:  "zero value",
			input: "0d 0h 0m",
			want:  0,
		},
		{
			name:  "empty string",
			input: "",
			want:  0,
		},
		{
			name:  "invalid format",
			input: "invalid",
			want:  0,
		},
		{
			name:  "large values",
			input: "365d 23h 59m 59s",
			want:  365*86400 + 23*3600 + 59*60 + 59, // 31622399
		},
		{
			name:  "with extra spaces",
			input: "  10d   5h  ",
			want:  10*86400 + 5*3600, // 903600
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRunTimeToSeconds(tt.input)
			if got != tt.want {
				t.Errorf("parseRunTimeToSeconds(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
