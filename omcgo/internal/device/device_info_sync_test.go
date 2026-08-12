package device

import (
	"context"
	"strconv"
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
	stringLimits     map[string]int
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

func (s stubDeviceInfoRepo) GetStringFieldLimits(context.Context) (map[string]int, error) {
	if s.stringLimits != nil {
		return s.stringLimits, nil
	}
	return defaultDeviceInfoVarcharLimits, nil
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

func (s stubDeviceInfoRepo) UpdateNameSyncFields(context.Context, uuid.UUID, bool, string) error {
	return nil
}

func (s stubDeviceInfoRepo) UpdateDeviceName(context.Context, uuid.UUID, string) error {
	return nil
}

type stubDeviceParamRepo struct {
	params []model.DeviceParameter
}

type stubDeviceCoordinateWriter struct {
	updateCoordinates func(ctx context.Context, deviceID uuid.UUID, latitude, longitude float64) error
	coordinates       *Location
}

func (s stubDeviceCoordinateWriter) UpdateCoordinates(ctx context.Context, deviceID uuid.UUID, latitude, longitude float64) error {
	if s.updateCoordinates != nil {
		return s.updateCoordinates(ctx, deviceID, latitude, longitude)
	}
	return nil
}

func (s stubDeviceCoordinateWriter) GetCoordinates(context.Context, uuid.UUID) (*Location, error) {
	return s.coordinates, nil
}

type stubLocationObservationRepo struct {
	upsert func(deviceID uuid.UUID, observation ReportedLocation) error
}

func (s stubLocationObservationRepo) SaveLatestWithOutbox(
	_ context.Context,
	deviceID uuid.UUID,
	observation ReportedLocation,
) (LocationObservationWriteResult, error) {
	if s.upsert != nil {
		if err := s.upsert(deviceID, observation); err != nil {
			return LocationObservationWriteResult{}, err
		}
	}
	return LocationObservationWriteResult{Current: observation}, nil
}

func (s stubLocationObservationRepo) GetLatest(context.Context, uuid.UUID) (*ReportedLocation, error) {
	return nil, nil
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
			name:  "fallback to BaiBNQ locked altitude",
			paths: map[string]string{"Device.FAP.GPS.LockedAltitude": "168"},
			want:  "168", ok: true,
		},
		{
			name: "BaiBNQ locked altitude wins over synchronization altitude",
			paths: map[string]string{
				"Device.FAP.GPS.LockedAltitude":       "168",
				"Device.FAP.Synchronization.Altitude": "514.49",
			},
			want: "168", ok: true,
		},
		{
			name:  "fallback to Height when GPS altitude paths missing",
			paths: map[string]string{"Device.FAP.GPS.Height": "150"},
			want:  "150", ok: true,
		},
		{
			name:  "fallback to BaiBNQ synchronization altitude",
			paths: map[string]string{"Device.FAP.Synchronization.Altitude": "514.49"},
			want:  "514.49", ok: true,
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
			name: "literal instance template falls back to its mac",
			paths: map[string]string{
				"Device.Ethernet.Interface.{i}.MACAddress": "48:bf:74:37:84:e2",
			},
			want: "48:bf:74:37:84:e2",
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

func TestLookupDeviceInfoMAC(t *testing.T) {
	cases := []struct {
		name  string
		paths map[string]string
		want  string
		ok    bool
	}{
		{
			name: "uses x_com device info mac",
			paths: map[string]string{
				"Device.DeviceInfo.X_COM_MACAddress": "48:BF:74:2B:C5:D4",
			},
			want: "48:BF:74:2B:C5:D4",
			ok:   true,
		},
		{
			name: "ignores eu instance mac",
			paths: map[string]string{
				"Device.DeviceInfo.EU.1.Mac": "00:11:22:33:44:55",
			},
			want: "",
			ok:   false,
		},
		{
			name: "ignores ru instance mac",
			paths: map[string]string{
				"Device.DeviceInfo.EU.1.RU.1.Mac": "66:77:88:99:AA:BB",
			},
			want: "",
			ok:   false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := lookupDeviceInfoMAC(tt.paths)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestInfoSyncer_SyncFromParameters_BackfillsMACFromWANTraversalWhenDirectPathMissing(t *testing.T) {
	deviceID := uuid.New()
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Ethernet.Interface.1.MACAddress", ParameterValue: "00:11:22:33:44:55"},
		{ParameterPath: "Device.Ethernet.Interface.1.Name", ParameterValue: "LAN1"},
		{ParameterPath: "Device.Ethernet.Interface.2.MACAddress", ParameterValue: "66:77:88:99:AA:BB"},
		{ParameterPath: "Device.Ethernet.Interface.2.IPv4Address.1.PortType", ParameterValue: "WAN"},
	}

	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})

	t.Run("nr uses WAN traversal", func(t *testing.T) {
		infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, _ uuid.UUID, fields map[string]interface{}) error {
			assert.Equal(t, "66:77:88:99:AA:BB", fields["mac"])
			return nil
		}}
		paramRepo := stubDeviceParamRepo{params: params}
		syncer := NewInfoSyncer(infoRepo, paramRepo, nil, registry, zap.NewNop())

		_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechNR, "")
		assert.NoError(t, err)
	})

	t.Run("lte also backfills from WAN traversal", func(t *testing.T) {
		infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, _ uuid.UUID, fields map[string]interface{}) error {
			assert.Equal(t, "66:77:88:99:AA:BB", fields["mac"])
			return nil
		}}
		paramRepo := stubDeviceParamRepo{params: params}
		syncer := NewInfoSyncer(infoRepo, paramRepo, nil, registry, zap.NewNop())

		_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechLTE, "")
		assert.NoError(t, err)
	})
}

func TestInfoSyncer_SyncFromParameters_PrefersDirectMACPathOverTraversal(t *testing.T) {
	deviceID := uuid.New()
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Ethernet.Interface.MACAddress", ParameterValue: "48:BF:74:2B:C5:D4"},
		{ParameterPath: "Device.Ethernet.Interface.1.MACAddress", ParameterValue: "00:11:22:33:44:55"},
		{ParameterPath: "Device.Ethernet.Interface.2.MACAddress", ParameterValue: "66:77:88:99:AA:BB"},
		{ParameterPath: "Device.Ethernet.Interface.2.IPv4Address.1.PortType", ParameterValue: "WAN"},
	}

	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})

	infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, _ uuid.UUID, fields map[string]interface{}) error {
		assert.Equal(t, "48:BF:74:2B:C5:D4", fields["mac"])
		return nil
	}}
	paramRepo := stubDeviceParamRepo{params: params}
	syncer := NewInfoSyncer(infoRepo, paramRepo, nil, registry, zap.NewNop())

	_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechLTE, "")
	assert.NoError(t, err)
}

func TestInfoSyncer_SyncFromParameters_BackfillsMACFromXCOMWhenStandardPathMissing(t *testing.T) {
	deviceID := uuid.New()
	params := []model.DeviceParameter{
		{ParameterPath: "Device.DeviceInfo.X_COM_MACAddress", ParameterValue: "48:BF:74:2B:C5:D4"},
	}

	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})

	infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, _ uuid.UUID, fields map[string]interface{}) error {
		assert.Equal(t, "48:BF:74:2B:C5:D4", fields["mac"])
		return nil
	}}
	paramRepo := stubDeviceParamRepo{params: params}
	syncer := NewInfoSyncer(infoRepo, paramRepo, nil, registry, zap.NewNop())

	_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechLTE, "")
	assert.NoError(t, err)
}

func TestInfoSyncer_SyncFromParameters_DoesNotUseEUInstanceMAC(t *testing.T) {
	deviceID := uuid.New()
	params := []model.DeviceParameter{
		{ParameterPath: "Device.DeviceInfo.EU.1.Mac", ParameterValue: "00:11:22:33:44:55"},
	}

	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})

	infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, _ uuid.UUID, fields map[string]interface{}) error {
		_, exists := fields["mac"]
		assert.False(t, exists)
		return nil
	}}
	paramRepo := stubDeviceParamRepo{params: params}
	syncer := NewInfoSyncer(infoRepo, paramRepo, nil, registry, zap.NewNop())

	_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechLTE, "")
	assert.NoError(t, err)
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

	_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechNR, "")
	assert.NoError(t, err)
}

func TestInfoSyncer_SyncFromParameters_StoresStandardGPSObservationWithoutOverwritingAccepted(t *testing.T) {
	deviceID := uuid.New()
	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})

	paramRepo := stubDeviceParamRepo{params: []model.DeviceParameter{
		{ParameterPath: "Device.DeviceInfo.SAS.FAP.GPS.LockedLatitude", ParameterValue: "39904200"},
		{ParameterPath: "Device.DeviceInfo.SAS.FAP.GPS.LockedLongitude", ParameterValue: "116407400"},
		{ParameterPath: "Device.FAP.GPS.Height", ParameterValue: "174"},
		{ParameterPath: "Device.DeviceInfo.X_COM_GPS_Status", ParameterValue: "1"},
		{ParameterPath: "Device.FAP.GPS.NumberOfSatellites", ParameterValue: "8"},
	}}

	accepted := &Location{Latitude: 31.2, Longitude: 121.5}
	coordinateWriter := stubDeviceCoordinateWriter{coordinates: accepted, updateCoordinates: func(context.Context, uuid.UUID, float64, float64) error {
		t.Fatal("existing accepted coordinates must not be overwritten during parameter sync")
		return nil
	}}
	observationRepo := stubLocationObservationRepo{upsert: func(gotDeviceID uuid.UUID, observation ReportedLocation) error {
		assert.Equal(t, deviceID, gotDeviceID)
		assert.Equal(t, 39.9042, observation.Latitude)
		assert.Equal(t, 116.4074, observation.Longitude)
		if assert.NotNil(t, observation.GPSHeight) {
			assert.Equal(t, 174.0, *observation.GPSHeight)
		}
		if assert.NotNil(t, observation.GPSLockStatus) {
			assert.Equal(t, "normal", *observation.GPSLockStatus)
		}
		if assert.NotNil(t, observation.SatelliteCount) {
			assert.Equal(t, 8, *observation.SatelliteCount)
		}
		assert.False(t, observation.ReceivedAt.IsZero())
		assert.Equal(t, observation.ReceivedAt, observation.ObservedAt)
		assert.Nil(t, observation.DeviceReportedAt)
		assert.Nil(t, observation.AccuracyMeters)
		assert.Equal(t, "Device.DeviceInfo.SAS.FAP.GPS", observation.SourcePath)
		return nil
	}}

	syncer := NewInfoSyncer(infoRepoNoop{}, paramRepo, coordinateWriter, registry, zap.NewNop(), observationRepo)
	_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechLTE, "")
	assert.NoError(t, err)
}

func TestInfoSyncer_SyncFromParameters_IgnoresTR069GPSInExternalMode(t *testing.T) {
	deviceID := uuid.New()
	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})
	paramRepo := stubDeviceParamRepo{params: []model.DeviceParameter{
		{ParameterPath: "Device.DeviceInfo.SAS.FAP.GPS.LockedLatitude", ParameterValue: "39904200"},
		{ParameterPath: "Device.DeviceInfo.SAS.FAP.GPS.LockedLongitude", ParameterValue: "116407400"},
	}}
	coordinateWriter := stubDeviceCoordinateWriter{updateCoordinates: func(
		context.Context,
		uuid.UUID,
		float64,
		float64,
	) error {
		t.Fatal("external positioning mode must not accept TR069 coordinates")
		return nil
	}}
	observationRepo := stubLocationObservationRepo{upsert: func(
		uuid.UUID,
		ReportedLocation,
	) error {
		return ErrLocationSourceNotAllowed
	}}
	syncer := NewInfoSyncer(
		infoRepoNoop{},
		paramRepo,
		coordinateWriter,
		registry,
		zap.NewNop(),
		observationRepo,
	)

	_, err := syncer.SyncFromParameters(
		context.Background(),
		deviceID,
		model.CarrierCMCC,
		model.TechLTE,
		"",
	)

	assert.NoError(t, err)
}

type infoRepoNoop struct{}

func (infoRepoNoop) GetByDeviceID(context.Context, uuid.UUID) (*DeviceInfo, error) { return nil, nil }
func (infoRepoNoop) Create(context.Context, *DeviceInfo) error                     { return nil }
func (infoRepoNoop) UpdateManualFields(context.Context, uuid.UUID, UpdateDeviceInfoRequest, string) error {
	return nil
}
func (infoRepoNoop) UpdateSyncFields(context.Context, uuid.UUID, map[string]interface{}) error {
	return nil
}
func (infoRepoNoop) UpdateNameSyncFields(context.Context, uuid.UUID, bool, string) error { return nil }
func (infoRepoNoop) UpdateDeviceName(context.Context, uuid.UUID, string) error           { return nil }
func (infoRepoNoop) GetTopologyAttributes(context.Context, uuid.UUID) (map[string]string, error) {
	return nil, nil
}
func (infoRepoNoop) ListDevicesWithInfo(context.Context, DeviceFilter) (*model.ListResponse[DeviceWithInfo], error) {
	return nil, nil
}
func (infoRepoNoop) GetByIDWithInfo(context.Context, uuid.UUID) (*DeviceWithInfo, error) {
	return nil, nil
}
func (infoRepoNoop) ComputeListStats(context.Context, DeviceFilter) (*DeviceListStats, error) {
	return nil, nil
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

	_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechNR, "")
	assert.NoError(t, err)
}

func TestInfoSyncer_SyncFromParameters_AggregatesUECountAcrossPhysicalCells(t *testing.T) {
	deviceID := uuid.New()
	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})
	paramRepo := stubDeviceParamRepo{params: []model.DeviceParameter{
		{ParameterPath: "Device.DeviceInfo.UE_Count", ParameterValue: "2"},
		{ParameterPath: "Device.DeviceInfo.2.UE_Count", ParameterValue: "3"},
	}}
	infoRepo := stubDeviceInfoRepo{updateSyncFields: func(
		_ context.Context,
		gotDeviceID uuid.UUID,
		fields map[string]interface{},
	) error {
		assert.Equal(t, deviceID, gotDeviceID)
		assert.Equal(t, 5, fields["ue_count"])
		return nil
	}}
	syncer := NewInfoSyncer(infoRepo, paramRepo, nil, registry, zap.NewNop())

	_, err := syncer.SyncFromParameters(
		context.Background(),
		deviceID,
		model.CarrierCMCC,
		model.TechLTE,
		"",
	)
	assert.NoError(t, err)
}

func TestInfoSyncer_SyncFromParameters_RFProjection(t *testing.T) {
	tests := []struct {
		name         string
		productClass string
		tech         model.Technology
		params       []model.DeviceParameter
		want         string
	}{
		{
			name:         "DC persists only physical carrier statuses",
			productClass: "FAP/MLN/DC",
			tech:         model.TechLTE,
			params: []model.DeviceParameter{
				{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells", ParameterValue: "2"},
				{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", ParameterValue: "true"},
				{ParameterPath: "Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus", ParameterValue: "false"},
				{ParameterPath: "Device.Services.FAPService.3.FAPControl.LTE.RFTxStatus", ParameterValue: "true"},
			},
			want: "on,off",
		},
		{
			name:         "BSC clears child RF projection",
			productClass: "FAP/PGSM",
			tech:         model.TechGSM,
			params: []model.DeviceParameter{
				{ParameterPath: "Device.Services.GsmBTSCellDT.1.RfState", ParameterValue: "1"},
			},
			want: "",
		},
		{
			name:         "incomplete DC clears stale RF projection",
			productClass: "FAP/MLN/DC",
			tech:         model.TechLTE,
			params: []model.DeviceParameter{
				{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells", ParameterValue: "2"},
				{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", ParameterValue: "true"},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviceID := uuid.New()
			registry := carrier.NewRegistry()
			registry.Register(testCarrier{})
			infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, gotDeviceID uuid.UUID, fields map[string]interface{}) error {
				assert.Equal(t, deviceID, gotDeviceID)
				assert.Equal(t, tt.want, fields["rf_status"])
				return nil
			}}
			syncer := NewInfoSyncer(infoRepo, stubDeviceParamRepo{params: tt.params}, nil, registry, zap.NewNop())

			_, err := syncer.SyncFromParameters(
				context.Background(), deviceID, model.CarrierCMCC, tt.tech, tt.productClass,
			)
			assert.NoError(t, err)
		})
	}
}

func TestInfoSyncer_SyncFromParameters_RadioFrequencyProjection(t *testing.T) {
	deviceID := uuid.New()
	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})

	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells", ParameterValue: "2"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNDL", ParameterValue: "39751"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.EARFCNDL", ParameterValue: "42599"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNUL", ParameterValue: "39751"},
		{ParameterPath: "Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.EARFCNDL", ParameterValue: "39952"},
		{ParameterPath: "Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.EARFCNUL", ParameterValue: "39751"},
		{ParameterPath: "Device.Services.FAPService.3.CellConfig.LTE.RAN.RF.EARFCNDL", ParameterValue: "42599"},
		{ParameterPath: "Device.Services.FAPService.3.CellConfig.LTE.RAN.RF.EARFCNUL", ParameterValue: "42599"},
	}

	infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, gotDeviceID uuid.UUID, fields map[string]interface{}) error {
		assert.Equal(t, deviceID, gotDeviceID)
		assert.Equal(t, "39751,39952", fields["freq_point"])
		assert.Equal(t, "39751,39751", fields["ul_earfcn"])
		return nil
	}}
	syncer := NewInfoSyncer(infoRepo, stubDeviceParamRepo{params: params}, nil, registry, zap.NewNop())

	_, err := syncer.SyncFromParameters(
		context.Background(), deviceID, model.CarrierCMCC, model.TechLTE, "FAP/MLN/DC",
	)
	assert.NoError(t, err)
}

func TestInfoSyncer_SyncFromParameters_ClearsIncompleteRadioFrequencyProjection(t *testing.T) {
	deviceID := uuid.New()
	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})

	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells", ParameterValue: "2"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNDL", ParameterValue: "39751"},
	}
	infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, _ uuid.UUID, fields map[string]interface{}) error {
		assert.Equal(t, "", fields["freq_point"], "不完整投影必须覆盖并清除通用聚合生成的单值")
		return nil
	}}
	syncer := NewInfoSyncer(infoRepo, stubDeviceParamRepo{params: params}, nil, registry, zap.NewNop())

	_, err := syncer.SyncFromParameters(
		context.Background(), deviceID, model.CarrierCMCC, model.TechLTE, "FAP/MLN/DC",
	)
	assert.NoError(t, err)
}

func TestInfoSyncer_SyncFromParameters_ClearsStaleCoreNetworkStatus(t *testing.T) {
	deviceID := uuid.New()
	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})

	infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, gotDeviceID uuid.UUID, fields map[string]interface{}) error {
		assert.Equal(t, deviceID, gotDeviceID)
		mmeStatus, exists := fields["mme_status"]
		assert.True(t, exists, "mme_status must be explicitly cleared when NR AMF status is absent")
		assert.Equal(t, "", mmeStatus)
		return nil
	}}
	syncer := NewInfoSyncer(infoRepo, stubDeviceParamRepo{}, nil, registry, zap.NewNop())

	_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechNR, "")
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

// TestInfoSyncer_SyncFromParameters_TransmitPowerSource 锁定快速设置/列表统一口径：
// transmit_power 直接取快速设置同源参数的已落库值，不再做范围或占位值判断。
func TestInfoSyncer_SyncFromParameters_TransmitPowerSource(t *testing.T) {
	deviceID := uuid.New()
	registry := carrier.NewRegistry()
	registry.Register(txPowerCarrier{})

	const refSignalPath = "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.ReferenceSignalPower"
	const xcomMaxTxPath = "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_MaxTxPowerExpanded"
	const nrPowerModifyPath = "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PowerModify"
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
			name: "X_COM 配置功率优先于 ReferenceSignalPower",
			params: []model.DeviceParameter{
				{ParameterPath: refSignalPath, ParameterValue: "18.2"},
				{ParameterPath: xcomMaxTxPath, ParameterValue: "24"},
			},
			want: "24",
		},
		{
			name: "NR PowerModify 应直接落到 transmit_power",
			params: []model.DeviceParameter{
				{ParameterPath: nrPowerModifyPath, ParameterValue: "30"},
			},
			want: "30",
		},
		{
			name: "only MaxTxPower → 不再投影到 transmit_power（#362 已删 universal 映射）",
			params: []model.DeviceParameter{
				{ParameterPath: maxTxPath, ParameterValue: "46"},
			},
			want: nil,
		},
		{
			name: "ReferenceSignalPower 原始值直接落库",
			params: []model.DeviceParameter{
				{ParameterPath: refSignalPath, ParameterValue: "-21"},
			},
			want: "-21",
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

			_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechLTE, "")
			assert.NoError(t, err)
		})
	}
}

func TestInfoSyncer_SyncFromParameters_BLQPreferredBandwidthAndAdminState(t *testing.T) {
	deviceID := uuid.New()
	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})

	paramRepo := stubDeviceParamRepo{params: []model.DeviceParameter{
		{ParameterPath: "Device.DeviceInfo.SAS.PreferredBandwidth", ParameterValue: "n100"},
		{ParameterPath: "Device.DeviceInfo.FAP_adminstate", ParameterValue: "true"},
	}}
	infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, gotDeviceID uuid.UUID, fields map[string]interface{}) error {
		assert.Equal(t, deviceID, gotDeviceID)
		assert.Equal(t, float64(20), fields["bandwidth"])
		assert.Equal(t, "true", fields["admin_state"])
		return nil
	}}
	syncer := NewInfoSyncer(infoRepo, paramRepo, nil, registry, zap.NewNop())

	_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechLTE, "")
	assert.NoError(t, err)
}

func TestInfoSyncer_SyncFromParameters_GSMBTSRadioFields(t *testing.T) {
	deviceID := uuid.New()
	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})

	paramRepo := stubDeviceParamRepo{params: []model.DeviceParameter{
		{ParameterPath: "Device.Services.GsmBTSCellDT.1.InUse", ParameterValue: "true"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.1.GsmCellID", ParameterValue: "1001"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.1.CurrLocAreaCode", ParameterValue: "2001"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.1.CurrentArfcn", ParameterValue: "45"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.1.GsmBtsBand", ParameterValue: "GSM900"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.1.GsmBtsRFPower", ParameterValue: "33"},
	}}
	infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, _ uuid.UUID, fields map[string]interface{}) error {
		assert.Equal(t, "45", fields["freq_point"])
		assert.Equal(t, "33", fields["transmit_power"])
		assert.Equal(t, "GSM900", fields["band"])
		assert.Equal(t, "1001", fields["cell_id"])
		assert.Equal(t, "2001", fields["lac"])
		return nil
	}}
	syncer := NewInfoSyncer(infoRepo, paramRepo, nil, registry, zap.NewNop())

	_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechGSM, "")
	assert.NoError(t, err)
}

func TestInfoSyncer_SyncFromParameters_LegacyBTSRadioFields(t *testing.T) {
	deviceID := uuid.New()
	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})

	paramRepo := stubDeviceParamRepo{params: []model.DeviceParameter{
		{ParameterPath: "Device.DeviceInfo.BTS.CurrentArfcn", ParameterValue: "1010"},
		{ParameterPath: "Device.DeviceInfo.BTS.CurrentLac", ParameterValue: "227"},
		{ParameterPath: "Device.DeviceInfo.GSM.BtsRfPower", ParameterValue: "43"},
	}}
	infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, _ uuid.UUID, fields map[string]interface{}) error {
		assert.Equal(t, "1010", fields["freq_point"])
		assert.Equal(t, "43", fields["transmit_power"])
		assert.Equal(t, "227", fields["lac"])
		return nil
	}}
	syncer := NewInfoSyncer(infoRepo, paramRepo, nil, registry, zap.NewNop())

	_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechGSM, "")
	assert.NoError(t, err)
}

func TestInfoSyncer_SyncFromParameters_BSCTrxARFCNAggregatesToFreqPoint(t *testing.T) {
	deviceID := uuid.New()
	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})

	paramRepo := stubDeviceParamRepo{params: []model.DeviceParameter{
		{ParameterPath: "DeviceGSM.Bts.0.Trx.1.Arfcn", ParameterValue: "1010"},
		{ParameterPath: "DeviceGSM.Bts.0.Trx.2.Arfcn", ParameterValue: "1013"},
	}}
	infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, _ uuid.UUID, fields map[string]interface{}) error {
		assert.Equal(t, "1010,1013", fields["freq_point"])
		return nil
	}}
	syncer := NewInfoSyncer(infoRepo, paramRepo, nil, registry, zap.NewNop())

	_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechGSM, "")
	assert.NoError(t, err)
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

// TestUniversalInformMapping_NRBandAndULEARFCN 防回归守卫：NR 制式下设备列表
// Band / UL EARFCN 列必须由 universalInformInstanceMappings 用 NR.RAN.RF.* 模板
// 覆盖；删除任一条会令 BaiBNQ 等 NR 设备列表对应列长期空白。
func TestUniversalInformMapping_NRBandAndULEARFCN(t *testing.T) {
	assert.Contains(t, instanceTemplatesFor(t, "band"),
		"Device.Services.FAPService.{f}.CellConfig.{c}.NR.RAN.PHY.FrequencyInfoDLSIB.MultiFrequencyBandListNRSIB.{b}.FreqBandIndicatorNR",
		"BaiBNQ FreqBandIndicatorNR 必须在 band 列的实例聚合模板中")
	assert.Contains(t, instanceTemplatesFor(t, "freq_point"),
		"Device.Services.FAPService.{f}.CellConfig.LTE.RAN.RF.EARFCNDL",
		"LTE RAN.RF.EARFCNDL 必须在 freq_point 列的实例聚合模板中，避免列表与详情页频点路径分叉")
	assert.Contains(t, instanceTemplatesFor(t, "ul_earfcn"),
		"Device.Services.FAPService.{f}.CellConfig.{c}.NR.RAN.RF.NRARFCNUL",
		"NR NRARFCNUL 必须在 ul_earfcn 列的实例聚合模板中")
}

// TestUniversalInformMapping_NRTACAndCellID 防回归守卫：NR 制式下设备列表
// TAC / Cell ID 列必须由 universalInformInstanceMappings 用 NR.CN.TA.{t}.* 模板
// 覆盖；删除任一条会令 BaiBNQ 等 NR 设备列表对应列长期空白。
// 与 LTE 的 EPC.TAC → tac 映射形成跨制式对称。
func TestUniversalInformMapping_NRTACAndCellID(t *testing.T) {
	assert.Contains(t, instanceTemplatesFor(t, "tac"),
		"Device.Services.FAPService.{f}.CellConfig.{c}.NR.CN.TA.{t}.TAC",
		"NR CN.TA.{t}.TAC 必须在 tac 列的实例聚合模板中")
	assert.Contains(t, instanceTemplatesFor(t, "cell_id"),
		"Device.Services.FAPService.{f}.CellConfig.{c}.NR.CN.TA.{t}.NrcellIdentity",
		"NR CN.TA.{t}.NrcellIdentity 必须在 cell_id 列的实例聚合模板中")
}

// TestUniversalInformMapping_AdminStateAndIpsecAddr 防回归守卫：设备列表
// "Admin State" 列必须与详情页「小区信息」表同源（detail_assembler 用
// CellConfig.{c}.NR.RAN.CellEnable.AdminState），"IPSec 地址" 列必须与详情页
// 「IPSec 参数」表"网关地址"列同源（Device.FAP.Ipsec.{i}.TUNNEL_GATEWAY）。
func TestUniversalInformMapping_AdminStateAndIpsecAddr(t *testing.T) {
	assert.Contains(t, instanceTemplatesFor(t, "admin_state"),
		"Device.Services.FAPService.{f}.CellConfig.{c}.NR.RAN.CellEnable.AdminState",
		"CellEnable.AdminState 必须在 admin_state 列的实例聚合模板中（与详情页小区表同源）")
	assert.Contains(t, instanceTemplatesFor(t, "ipsec_addr"),
		"Device.FAP.Ipsec.{i}.TUNNEL_GATEWAY",
		"FAP.Ipsec.{i}.TUNNEL_GATEWAY 必须在 ipsec_addr 列的实例聚合模板中（与详情页 IPSec 网关地址列同源）")

	// 反向断言：之前一版选错的路径不应再被任何 mapping 路径写入 admin_state/ipsec_addr，
	// 否则 list 会与详情页给出相反结论。
	assert.NotEqual(t, "admin_state",
		universalInformMapping["Device.Services.FAPService.1.FAPControl.NR.RAN.Common.AdminState"],
		"FAPControl.NR.RAN.Common.AdminState（设备级三态）不得复用 admin_state 列")
	assert.NotEqual(t, "ipsec_addr",
		universalInformMapping["Device.DeviceInfo.SERVING_UNIT1_IPSEC_Address"],
		"SERVING_UNIT1_IPSEC_Address（本端 source IP, 未建立隧道时 0.0.0.0）不得复用 ipsec_addr 列")
}

// instanceTemplatesFor 返回 universalInformInstanceMappings 中目标列的所有模板。
func instanceTemplatesFor(t *testing.T, column string) []string {
	t.Helper()
	for _, m := range universalInformInstanceMappings {
		if m.column == column {
			return m.templates
		}
	}
	return nil
}

// TestAggregateInstanceFields_SingleCell 单 cell 设备聚合结果应等价单值,不破坏
// 历史行为(BaiBNQ 等 1 个 cell 设备 sync 后 list 列仍是单值字符串)。
func TestAggregateInstanceFields_SingleCell(t *testing.T) {
	params := map[string]string{
		"Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.PhyCellID":          "21",
		"Device.Services.FAPService.1.CellConfig.1.NR.CN.TA.1.TAC":               "81",
		"Device.Services.FAPService.1.CellConfig.1.NR.CN.TA.1.NrcellIdentity":    "1153",
		"Device.Services.FAPService.1.CellConfig.1.NR.RAN.CellEnable.AdminState": "1",
		"Device.FAP.Ipsec.1.TUNNEL_GATEWAY":                                      "192.168.13.180",
	}
	fields := map[string]interface{}{}
	aggregateInstanceFields(params, fields)

	assert.Equal(t, "21", fields["pci"])
	assert.Equal(t, "81", fields["tac"])
	assert.Equal(t, "1153", fields["cell_id"])
	assert.Equal(t, "1", fields["admin_state"])
	assert.Equal(t, "192.168.13.180", fields["ipsec_addr"])
}

// TestAggregateInstanceFields_MultiCell 多 cell / 多 IPSec 隧道场景：按实例索引
// 升序、值去重后用 "," 拼接，覆盖此前可能写入的单值。这是 #364-followup 的
// 核心契约：前端 device list 小区级/多实例字段统一显示 csv。
func TestAggregateInstanceFields_MultiCell(t *testing.T) {
	params := map[string]string{
		// cell 1 / cell 2：两个 PCI、两个 TAC、两个 NR Cell Identity、两个 admin_state
		"Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.PhyCellID":          "21",
		"Device.Services.FAPService.1.CellConfig.2.NR.RAN.RF.PhyCellID":          "22",
		"Device.Services.FAPService.1.CellConfig.1.NR.CN.TA.1.TAC":               "81",
		"Device.Services.FAPService.1.CellConfig.2.NR.CN.TA.1.TAC":               "82",
		"Device.Services.FAPService.1.CellConfig.1.NR.CN.TA.1.NrcellIdentity":    "1153",
		"Device.Services.FAPService.1.CellConfig.2.NR.CN.TA.1.NrcellIdentity":    "1154",
		"Device.Services.FAPService.1.CellConfig.1.NR.RAN.CellEnable.AdminState": "1",
		"Device.Services.FAPService.1.CellConfig.2.NR.RAN.CellEnable.AdminState": "2",
		// IPSec 多隧道
		"Device.FAP.Ipsec.1.TUNNEL_GATEWAY": "10.0.0.1",
		"Device.FAP.Ipsec.2.TUNNEL_GATEWAY": "10.0.0.2",
		"Device.FAP.Ipsec.3.TUNNEL_GATEWAY": "10.0.0.3",
	}
	// 单值预填（模拟上游 carrier mapping 已写 cell-1 单值），聚合后应覆盖。
	fields := map[string]interface{}{
		"pci":         "21",
		"admin_state": "1",
		"ipsec_addr":  "10.0.0.1",
	}
	aggregateInstanceFields(params, fields)

	assert.Equal(t, "21,22", fields["pci"], "PCI 多 cell 升序 csv")
	assert.Equal(t, "81,82", fields["tac"], "TAC 多 cell 升序 csv")
	assert.Equal(t, "1153,1154", fields["cell_id"], "Cell ID 多 cell 升序 csv")
	assert.Equal(t, "1,2", fields["admin_state"], "Admin State 多 cell 升序 csv（覆盖单值 \"1\"）")
	assert.Equal(t, "10.0.0.1,10.0.0.2,10.0.0.3", fields["ipsec_addr"], "IPSec 多隧道升序 csv")
}

// TestAggregateInstanceFields_DedupSameValue 多 cell 上报相同值（如双 cell 同
// PCI 不太合理但 admin_state 双 cell 都 "1" 常见）时，应去重而非显示 "1,1"。
func TestAggregateInstanceFields_DedupSameValue(t *testing.T) {
	params := map[string]string{
		"Device.Services.FAPService.1.CellConfig.1.NR.RAN.CellEnable.AdminState": "1",
		"Device.Services.FAPService.1.CellConfig.2.NR.RAN.CellEnable.AdminState": "1",
		"Device.Services.FAPService.1.CellConfig.3.NR.RAN.CellEnable.AdminState": "2",
	}
	fields := map[string]interface{}{}
	aggregateInstanceFields(params, fields)
	assert.Equal(t, "1,2", fields["admin_state"], "相同值去重，仅保留首次出现")
}

// TestAggregateInstanceFields_NoMatch 模板都不命中时 fields 不动，保留上游
// carrier/universal mapping 的单值。
func TestAggregateInstanceFields_NoMatch(t *testing.T) {
	params := map[string]string{
		"Device.DeviceInfo.UpTime": "12345",
	}
	fields := map[string]interface{}{
		"pci": "21",
	}
	aggregateInstanceFields(params, fields)
	assert.Equal(t, "21", fields["pci"], "无命中时单值字段不变")
	_, hasTac := fields["tac"]
	assert.False(t, hasTac, "无命中时不引入新字段")
}

func TestEnforceDeviceInfoFieldSizeLimits_TrimsCSVInsteadOfFailingWholeSync(t *testing.T) {
	fields := map[string]interface{}{
		"ipsec_addr": "baicells-epc.cloudapp.net,baicells-east-epc.eastus.cloudapp.azure.com",
		"pci":        "241,503",
		"plmn":       "46068,46088,46066,46077,46055,46044,46011",
	}

	enforceDeviceInfoFieldSizeLimits(fields)

	assert.Equal(t, "baicells-epc.cloudapp.net", fields["ipsec_addr"], "超长 CSV 应保留能装进 device_info 列宽的前缀值")
	assert.Equal(t, "241,503", fields["pci"])
	assert.Equal(t, "46068,46088,46066,46077,46055,46044", fields["plmn"], "PLMN CSV 应按 varchar(40) 边界保留完整条目")
}

func TestEnforceDeviceInfoFieldSizeLimits_UsesSchemaAndDropsUnsafeScalar(t *testing.T) {
	fields := map[string]interface{}{
		"rf_status":        "off,off,off,off,off,off,on,on,on",
		"hardware_version": "BM_2.1.3_1.8G_freq_20260604_103928",
		"mac":              "52:af:00:7c:92:6d",
	}

	enforceDeviceInfoFieldSizeLimitsWithSchema(fields, map[string]int{
		"rf_status":        64,
		"hardware_version": 16,
		"mac":              64,
	}, zap.NewNop(), uuid.New())

	assert.Equal(t, "off,off,off,off,off,off,on,on,on", fields["rf_status"], "9 个小区 RF 状态应能完整保存")
	assert.Equal(t, "52:af:00:7c:92:6d", fields["mac"], "合法字段不能被超长字段拖累")
	assert.NotContains(t, fields, "hardware_version", "普通字符串超长时跳过更新，避免截断成坏数据")
}

func TestInfoSyncer_SyncFromParameters_OverlongIpsecCSVDoesNotAbortOtherFields(t *testing.T) {
	deviceID := uuid.New()
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.EPC.TAC", ParameterValue: "1"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID", ParameterValue: "241"},
		{ParameterPath: "Device.FAP.Ipsec.1.TUNNEL_GATEWAY", ParameterValue: "baicells-epc.cloudapp.net"},
		{ParameterPath: "Device.FAP.Ipsec.2.TUNNEL_GATEWAY", ParameterValue: "baicells-east-epc.eastus.cloudapp.azure.com"},
	}

	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})

	infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, _ uuid.UUID, fields map[string]interface{}) error {
		assert.Equal(t, "1", fields["tac"])
		assert.Equal(t, "241", fields["pci"])
		assert.Equal(t, "baicells-epc.cloudapp.net", fields["ipsec_addr"])
		return nil
	}}
	paramRepo := stubDeviceParamRepo{params: params}
	syncer := NewInfoSyncer(infoRepo, paramRepo, nil, registry, zap.NewNop())

	_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechLTE, "")
	assert.NoError(t, err)
}

func TestInfoSyncer_SyncFromParameters_NineCellRFStatusDoesNotAbortMAC(t *testing.T) {
	deviceID := uuid.New()
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Ethernet.Interface.MACAddress", ParameterValue: "52:af:00:7c:92:6d"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells", ParameterValue: "9"},
	}
	for i := 1; i <= 9; i++ {
		params = append(params, model.DeviceParameter{
			ParameterPath:  "Device.Services.FAPService." + strconv.Itoa(i) + ".FAPControl.LTE.RFTxStatus",
			ParameterValue: "0",
		})
	}

	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})

	infoRepo := stubDeviceInfoRepo{
		stringLimits: map[string]int{
			"rf_status": 64,
			"mac":       64,
		},
		updateSyncFields: func(_ context.Context, _ uuid.UUID, fields map[string]interface{}) error {
			assert.Equal(t, "52:af:00:7c:92:6d", fields["mac"])
			assert.Equal(t, "off,off,off,off,off,off,off,off,off", fields["rf_status"])
			return nil
		},
	}
	paramRepo := stubDeviceParamRepo{params: params}
	syncer := NewInfoSyncer(infoRepo, paramRepo, nil, registry, zap.NewNop())

	_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechLTE, "FAP/BU1810")
	assert.NoError(t, err)
}

func TestInfoSyncer_SyncFromParameters_VendorRunTimeFallback(t *testing.T) {
	deviceID := uuid.New()
	params := []model.DeviceParameter{
		{ParameterPath: ParamStationRunTime, ParameterValue: "2 days 5 hours 4 minutes"},
	}

	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})

	infoRepo := stubDeviceInfoRepo{updateSyncFields: func(_ context.Context, _ uuid.UUID, fields map[string]interface{}) error {
		assert.Equal(t, int64(2*86400+5*3600+4*60), fields["run_time"])
		return nil
	}}
	paramRepo := stubDeviceParamRepo{params: params}
	syncer := NewInfoSyncer(infoRepo, paramRepo, nil, registry, zap.NewNop())

	_, err := syncer.SyncFromParameters(context.Background(), deviceID, model.CarrierCMCC, model.TechLTE, "")
	assert.NoError(t, err)
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
			name:  "english long units",
			input: "2 days 5 hours 4 minutes",
			want:  2*86400 + 5*3600 + 4*60,
		},
		{
			name:  "chinese units",
			input: "2天5小时4分钟3秒",
			want:  2*86400 + 5*3600 + 4*60 + 3,
		},
		{
			name:  "plain numeric seconds",
			input: "190800",
			want:  190800,
		},
		{
			name:  "colon hours minutes seconds",
			input: "53:04:03",
			want:  53*3600 + 4*60 + 3,
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
