package acs

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	coremodel "github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/task"
)

// ─────────────────────────────────────────────────────────────────────
// Stubs for unit-only PathTranslationService tests.
// ─────────────────────────────────────────────────────────────────────

type stubDeviceLookup struct {
	dev *coremodel.Device
	err error
}

func (s *stubDeviceLookup) GetBySerialNumber(_ context.Context, _ string) (*coremodel.Device, error) {
	return s.dev, s.err
}

type stubProductMatcher struct {
	res *product.MatchResult
	err error
}

func (s *stubProductMatcher) MatchProductClass(_ context.Context, _ string) (*product.MatchResult, error) {
	return s.res, s.err
}

type stubTranslatorFactory struct {
	tr  *parammodel.Translator
	err error
}

func (s *stubTranslatorFactory) Translator(_ context.Context, _ uuid.UUID, _ string) (*parammodel.Translator, error) {
	return s.tr, s.err
}

// buildTranslator constructs a real parammodel.Translator from a tiny mapping table.
// Easier than mocking the *Translator type, which is concrete.
func buildTranslator(t *testing.T, pairs map[string]string) *parammodel.Translator {
	t.Helper()
	mappings := make([]parammodel.ParamMapping, 0, len(pairs))
	for std, priv := range pairs {
		mappings = append(mappings, parammodel.ParamMapping{
			StandardPath: std,
			PrivatePath:  priv,
		})
	}
	set := &parammodel.MappingSet{
		Source:   parammodel.MappingSourceDefault,
		Mappings: mappings,
	}
	tr := parammodel.NewTranslator(set, nil, zap.NewNop())
	require.NotNil(t, tr)
	return tr
}

func TestPathTranslationService_Enabled(t *testing.T) {
	cases := []struct {
		name    string
		dev     pathTranslatorDeviceLookup
		prod    pathTranslatorProductMatcher
		trans   pathTranslatorFactory
		enabled bool
	}{
		{"all-wired", &stubDeviceLookup{}, &stubProductMatcher{}, &stubTranslatorFactory{}, true},
		{"no-device", nil, &stubProductMatcher{}, &stubTranslatorFactory{}, false},
		{"no-product", &stubDeviceLookup{}, nil, &stubTranslatorFactory{}, false},
		{"no-translator", &stubDeviceLookup{}, &stubProductMatcher{}, nil, false},
		{"all-nil", nil, nil, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := NewPathTranslationService(c.dev, c.prod, c.trans, zap.NewNop())
			assert.Equal(t, c.enabled, s.Enabled())
		})
	}
}

func TestPathTranslationService_ResolveUECountPathsUsesCurrentProductMappings(t *testing.T) {
	dev := &coremodel.Device{ProductClass: "X-MLN", FirmwareVersion: "1.0.0"}
	modelID := uuid.New()
	prod := &product.Product{ID: uuid.New(), ParamModelID: &modelID}
	tr := parammodel.NewTranslator(&parammodel.MappingSet{
		Mappings: []parammodel.ParamMapping{
			{StandardPath: "Device.DeviceInfo.2.UE_Count", PrivatePath: "Device.FAP.2.UE", EntryType: "parameter", IsActive: true, IsSupported: true},
			{StandardPath: "Device.DeviceInfo.UE_Count", PrivatePath: "Device.FAP.1.UE", EntryType: "parameter", IsActive: true, IsSupported: true},
			{StandardPath: "Device.DeviceInfo.1.UE_Count", PrivatePath: "Device.FAP.Alias.UE", EntryType: "parameter", IsActive: true, IsSupported: true},
			{StandardPath: "Device.DeviceInfo.3.UE_Count", PrivatePath: "Device.FAP.3.UE", EntryType: "parameter", IsActive: false, IsSupported: true},
			{StandardPath: "Device.DeviceInfo.4.UE_Count", PrivatePath: "Device.FAP.4.UE", EntryType: "parameter", IsActive: true, IsSupported: false},
			{StandardPath: "Device.DeviceInfo.UpTime", PrivatePath: "Device.Info.UpTime", EntryType: "parameter", IsActive: true, IsSupported: true},
		},
	}, nil, zap.NewNop())
	service := NewPathTranslationService(
		&stubDeviceLookup{dev: dev},
		&stubProductMatcher{res: &product.MatchResult{Product: prod}},
		&stubTranslatorFactory{tr: tr},
		zap.NewNop(),
	)

	paths, err := service.ResolveUECountPaths(context.Background(), "SN-220")

	require.NoError(t, err)
	assert.Equal(t, []string{
		"Device.DeviceInfo.UE_Count",
		"Device.DeviceInfo.2.UE_Count",
	}, paths)
}

func TestTranslateTaskParams_Passthrough_WhenDisabled(t *testing.T) {
	// no dependencies → Enabled=false → original Params returned
	s := NewPathTranslationService(nil, nil, nil, zap.NewNop())
	in := json.RawMessage(`{"names":["Device.X."]}`)
	tk := &task.Task{ID: "t1", DeviceSN: "SN1", Method: "GetParameterValues", Params: in}
	out, changed := s.TranslateTaskParams(context.Background(), tk)
	assert.False(t, changed)
	assert.JSONEq(t, string(in), string(out))
}

func TestTranslateTaskParams_MapsGeofenceStandardRFPathToQRTBPrivatePath(t *testing.T) {
	dev := &coremodel.Device{
		ProductClass:    "FAP/mBS31001/DC",
		FirmwareVersion: "BLQ_5.1.11.9",
	}
	prod := &product.Product{ID: uuid.New(), ParamModelID: func() *uuid.UUID { id := uuid.New(); return &id }()}
	tr := buildTranslator(t, map[string]string{
		"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus": "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable",
	})
	service := NewPathTranslationService(
		&stubDeviceLookup{dev: dev},
		&stubProductMatcher{res: &product.MatchResult{Product: prod}},
		&stubTranslatorFactory{tr: tr},
		zap.NewNop(),
	)

	input := json.RawMessage(`{"values":[{"name":"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus","value":"0","type":"xsd:boolean"}]}`)
	taskRecord := &task.Task{
		ID:       "geofence-rf",
		DeviceSN: "SN-MBS31001-1",
		Method:   "SetParameterValues",
		Params:   input,
	}

	output, changed := service.TranslateTaskParams(context.Background(), taskRecord)

	assert.True(t, changed)
	assert.JSONEq(t, `{"values":[{"name":"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable","value":"0","type":"xsd:boolean"}]}`, string(output))
}

func TestTranslateTaskParams_TranslatesGPVNames(t *testing.T) {
	dev := &coremodel.Device{ProductClass: "X-BLQ", FirmwareVersion: "1.0.0"}
	prodID := uuid.New()
	modelID := uuid.New()
	prod := &product.Product{ID: prodID, ParamModelID: &modelID}
	matcher := &stubProductMatcher{res: &product.MatchResult{Product: prod}}
	tr := buildTranslator(t, map[string]string{
		"Device.WiFi.SSID.": "X_COM_VENDOR.WiFi.SSID.",
		"Device.LAN.IP.":    "X_COM_VENDOR.LAN.IP.",
	})

	s := NewPathTranslationService(
		&stubDeviceLookup{dev: dev},
		matcher,
		&stubTranslatorFactory{tr: tr},
		zap.NewNop(),
	)

	in := json.RawMessage(`{"names":["Device.WiFi.SSID.","Device.LAN.IP.","Device.Unknown."]}`)
	tk := &task.Task{ID: "t1", DeviceSN: "SN1", Method: "GetParameterValues", Params: in}
	out, changed := s.TranslateTaskParams(context.Background(), tk)

	require.True(t, changed)
	var got struct {
		Names []string `json:"names"`
	}
	require.NoError(t, json.Unmarshal(out, &got))
	assert.Equal(t, []string{
		"X_COM_VENDOR.WiFi.SSID.",
		"X_COM_VENDOR.LAN.IP.",
		"Device.Unknown.", // miss → original kept
	}, got.Names)
}

func TestTranslateTaskParams_ExpandsQueryPrefixCandidates(t *testing.T) {
	dev := &coremodel.Device{ProductClass: "X-BLQ", FirmwareVersion: "1.0.0"}
	modelID := uuid.New()
	prod := &product.Product{ID: uuid.New(), ParamModelID: &modelID}
	tr := buildTranslator(t, map[string]string{
		"Device.Services.FAPService.{i}.CellConfig.LTE.ParamA": "Device.VendorA.FAPService.{i}.CellConfig.LTE.ParamA",
		"Device.Services.FAPService.{i}.CellConfig.NR.ParamB":  "InternetGatewayDevice.Services.FAPService.{i}.CellConfig.NR.ParamB",
	})
	s := NewPathTranslationService(
		&stubDeviceLookup{dev: dev},
		&stubProductMatcher{res: &product.MatchResult{Product: prod}},
		&stubTranslatorFactory{tr: tr},
		zap.NewNop(),
	)

	for _, method := range []string{"GetParameterValues", "GetParameterAttributes"} {
		t.Run(method, func(t *testing.T) {
			in := json.RawMessage(`{"names":["Device.Services.FAPService."]}`)
			tk := &task.Task{ID: "t1", DeviceSN: "SN1", Method: method, Params: in}

			out, changed := s.TranslateTaskParams(context.Background(), tk)

			require.True(t, changed)
			var got struct {
				Names []string `json:"names"`
			}
			require.NoError(t, json.Unmarshal(out, &got))
			require.Equal(t, []string{
				"Device.VendorA.FAPService.",
				"InternetGatewayDevice.Services.FAPService.",
			}, got.Names)
		})
	}
}

func TestTranslateTaskParams_DeduplicatesExpandedNames(t *testing.T) {
	dev := &coremodel.Device{ProductClass: "X-BLQ", FirmwareVersion: "1.0.0"}
	modelID := uuid.New()
	prod := &product.Product{ID: uuid.New(), ParamModelID: &modelID}
	tr := buildTranslator(t, map[string]string{
		"Device.Services.FAPService.{i}.CellConfig.LTE.ParamA": "Device.VendorA.FAPService.{i}.CellConfig.LTE.ParamA",
		"Device.Services.FAPService.{i}.CellConfig.NR.ParamB":  "InternetGatewayDevice.Services.FAPService.{i}.CellConfig.NR.ParamB",
		"Device.Alias.": "Device.VendorA.FAPService.",
	})
	s := NewPathTranslationService(
		&stubDeviceLookup{dev: dev},
		&stubProductMatcher{res: &product.MatchResult{Product: prod}},
		&stubTranslatorFactory{tr: tr},
		zap.NewNop(),
	)
	in := json.RawMessage(`{"names":["Device.Services.FAPService.","Device.Alias.","Device.Unknown.","Device.Services.FAPService."]}`)
	tk := &task.Task{ID: "t1", DeviceSN: "SN1", Method: "GetParameterValues", Params: in}

	out, changed := s.TranslateTaskParams(context.Background(), tk)

	require.True(t, changed)
	var got struct {
		Names []string `json:"names"`
	}
	require.NoError(t, json.Unmarshal(out, &got))
	require.Equal(t, []string{
		"Device.VendorA.FAPService.",
		"InternetGatewayDevice.Services.FAPService.",
		"Device.Unknown.",
	}, got.Names)
}

func TestTranslateTaskParams_SkipsPrivatePathMode(t *testing.T) {
	called := false
	dev := &stubDeviceLookup{dev: &coremodel.Device{ProductClass: "X-BLQ", FirmwareVersion: "1.0.0"}}
	wrapped := devLookupCounter{inner: dev, called: &called}
	s := NewPathTranslationService(wrapped, &stubProductMatcher{}, &stubTranslatorFactory{}, zap.NewNop())
	in := json.RawMessage(`{"path_mode":"private","names":["InternetGatewayDevice.DeviceInfo.X_VENDOR_NotRegistered"]}`)
	tk := &task.Task{ID: "t1", DeviceSN: "SN1", Method: "GetParameterValues", Params: in}

	out, changed := s.TranslateTaskParams(context.Background(), tk)

	require.False(t, changed)
	assert.JSONEq(t, string(in), string(out))
	assert.False(t, called, "private path mode should bypass translator lookup")
}

// issue #424：响应方向（私有→标准）回译，与出站 TranslateTaskParams 对称。
func TestTranslateResponseNames_PrivateToStandard(t *testing.T) {
	dev := &coremodel.Device{ProductClass: "X-BLQ", FirmwareVersion: "1.0.0"}
	prodID := uuid.New()
	modelID := uuid.New()
	prod := &product.Product{ID: prodID, ParamModelID: &modelID}
	matcher := &stubProductMatcher{res: &product.MatchResult{Product: prod}}
	tr := buildTranslator(t, map[string]string{
		"Device.WiFi.SSID.": "X_COM_VENDOR.WiFi.SSID.",
		"Device.LAN.IP.":    "X_COM_VENDOR.LAN.IP.",
	})

	s := NewPathTranslationService(
		&stubDeviceLookup{dev: dev},
		matcher,
		&stubTranslatorFactory{tr: tr},
		zap.NewNop(),
	)

	// 基站响应里的私有 path → 回译为标准 path；未命中的私有 path 原样保留（不丢值）。
	in := []string{"X_COM_VENDOR.WiFi.SSID.", "X_COM_VENDOR.LAN.IP.", "X_COM_VENDOR.Unknown."}
	out, changed := s.TranslateResponseNames(context.Background(), "SN1", in)

	require.True(t, changed)
	assert.Equal(t, []string{
		"Device.WiFi.SSID.",
		"Device.LAN.IP.",
		"X_COM_VENDOR.Unknown.", // miss → 私有 path 原样保留
	}, out)
	// 不可变：返回的是新切片，入参不被改写。
	assert.Equal(t, "X_COM_VENDOR.WiFi.SSID.", in[0])
}

func TestTranslateResponseNamesForRequest_PreservesConcreteRequestedStandardPath(t *testing.T) {
	dev := &coremodel.Device{ProductClass: "X-BNQ", FirmwareVersion: "1.0.0"}
	modelID := uuid.New()
	prod := &product.Product{ID: uuid.New(), ParamModelID: &modelID}
	tr := buildTranslator(t, map[string]string{
		"Device.Services.FAPService.{i}.FAPControl.LTE.LICENSE.Author": "Device.FAP.License.Author",
	})
	s := NewPathTranslationService(
		&stubDeviceLookup{dev: dev},
		&stubProductMatcher{res: &product.MatchResult{Product: prod}},
		&stubTranslatorFactory{tr: tr},
		zap.NewNop(),
	)

	requestParams := json.RawMessage(`{"names":["Device.Services.FAPService.1.FAPControl.LTE.LICENSE.Author"]}`)
	out, changed := s.TranslateResponseNamesForRequest(
		context.Background(),
		"SN1",
		[]string{"Device.FAP.License.Author"},
		requestParams,
	)

	require.True(t, changed)
	assert.Equal(t, []string{"Device.Services.FAPService.1.FAPControl.LTE.LICENSE.Author"}, out)
	assert.NotContains(t, out[0], "{i}")
}

func TestTranslateResponseNamesForRequest_PreservesConcreteRequestedStandardPrefix(t *testing.T) {
	dev := &coremodel.Device{ProductClass: "X-BNQ", FirmwareVersion: "1.0.0"}
	modelID := uuid.New()
	prod := &product.Product{ID: uuid.New(), ParamModelID: &modelID}
	tr := buildTranslator(t, map[string]string{
		"Device.Services.FAPService.{i}.FAPControl.LTE.LICENSE.Author": "Device.FAP.License.Author",
		"Device.Services.FAPService.{i}.FAPControl.LTE.LICENSE.Code":   "Device.FAP.License.Code",
	})
	s := NewPathTranslationService(
		&stubDeviceLookup{dev: dev},
		&stubProductMatcher{res: &product.MatchResult{Product: prod}},
		&stubTranslatorFactory{tr: tr},
		zap.NewNop(),
	)

	requestParams := json.RawMessage(`{"names":["Device.Services.FAPService.1.FAPControl.LTE.LICENSE."]}`)
	out, changed := s.TranslateResponseNamesForRequest(
		context.Background(),
		"SN1",
		[]string{"Device.FAP.License.Author", "Device.FAP.License.Code"},
		requestParams,
	)

	require.True(t, changed)
	assert.Equal(t, []string{
		"Device.Services.FAPService.1.FAPControl.LTE.LICENSE.Author",
		"Device.Services.FAPService.1.FAPControl.LTE.LICENSE.Code",
	}, out)
}

func TestTranslateResponseNames_PassthroughWhenDisabled(t *testing.T) {
	s := NewPathTranslationService(nil, nil, nil, zap.NewNop())
	in := []string{"a", "b"}
	out, changed := s.TranslateResponseNames(context.Background(), "SN", in)
	require.False(t, changed)
	assert.Equal(t, in, out)
	out[0] = "x" // 改返回值不影响入参（拷贝语义）
	assert.Equal(t, "a", in[0])
}

func TestTranslateTaskParams_TranslatesSPVValues(t *testing.T) {
	dev := &coremodel.Device{ProductClass: "X-BLQ", FirmwareVersion: "1.0.0"}
	modelID := uuid.New()
	prod := &product.Product{ID: uuid.New(), ParamModelID: &modelID}
	matcher := &stubProductMatcher{res: &product.MatchResult{Product: prod}}
	tr := buildTranslator(t, map[string]string{
		"Device.WiFi.SSID.": "X_COM.WiFi.SSID.",
	})

	s := NewPathTranslationService(
		&stubDeviceLookup{dev: dev},
		matcher,
		&stubTranslatorFactory{tr: tr},
		zap.NewNop(),
	)
	in := json.RawMessage(`{"values":[{"name":"Device.WiFi.SSID.","value":"home","type":"xsd:string"}]}`)
	tk := &task.Task{ID: "t1", DeviceSN: "SN1", Method: "SetParameterValues", Params: in}
	out, changed := s.TranslateTaskParams(context.Background(), tk)
	require.True(t, changed)
	assert.Contains(t, string(out), `"name":"X_COM.WiFi.SSID."`)
	assert.Contains(t, string(out), `"value":"home"`)
}

func TestTranslateTaskParams_TranslatesAddObjectName(t *testing.T) {
	dev := &coremodel.Device{ProductClass: "X-BLQ", FirmwareVersion: "1.0.0"}
	modelID := uuid.New()
	prod := &product.Product{ID: uuid.New(), ParamModelID: &modelID}
	matcher := &stubProductMatcher{res: &product.MatchResult{Product: prod}}
	tr := buildTranslator(t, map[string]string{
		"Device.Carrier.": "X_COM.Carrier.",
	})

	s := NewPathTranslationService(
		&stubDeviceLookup{dev: dev},
		matcher,
		&stubTranslatorFactory{tr: tr},
		zap.NewNop(),
	)
	in := json.RawMessage(`{"object_name":"Device.Carrier."}`)
	tk := &task.Task{ID: "t1", DeviceSN: "SN1", Method: "AddObject", Params: in}
	out, changed := s.TranslateTaskParams(context.Background(), tk)
	require.True(t, changed)
	assert.Contains(t, string(out), `"object_name":"X_COM.Carrier."`)
}

func TestTranslateTaskParams_OrphanFallback(t *testing.T) {
	// product matcher returns ErrOrphan → passthrough, no error
	s := NewPathTranslationService(
		&stubDeviceLookup{dev: &coremodel.Device{ProductClass: "UNKNOWN"}},
		&stubProductMatcher{err: product.ErrOrphan},
		&stubTranslatorFactory{},
		zap.NewNop(),
	)
	in := json.RawMessage(`{"names":["Device.X."]}`)
	tk := &task.Task{ID: "t1", DeviceSN: "SN1", Method: "GetParameterValues", Params: in}
	out, changed := s.TranslateTaskParams(context.Background(), tk)
	assert.False(t, changed)
	assert.JSONEq(t, string(in), string(out))
}

func TestTranslateTaskParams_SkipsUnknownMethod(t *testing.T) {
	// Reboot has no path field → skip without lookup
	called := false
	dev := &stubDeviceLookup{dev: &coremodel.Device{}}
	wrapped := devLookupCounter{inner: dev, called: &called}
	s := NewPathTranslationService(wrapped, &stubProductMatcher{}, &stubTranslatorFactory{}, zap.NewNop())
	in := json.RawMessage(`{}`)
	tk := &task.Task{ID: "t1", DeviceSN: "SN1", Method: "Reboot", Params: in}
	out, changed := s.TranslateTaskParams(context.Background(), tk)
	assert.False(t, changed)
	assert.JSONEq(t, string(in), string(out))
	assert.False(t, called, "device lookup should not be invoked for non-path RPCs")
}

type devLookupCounter struct {
	inner  pathTranslatorDeviceLookup
	called *bool
}

func (d devLookupCounter) GetBySerialNumber(ctx context.Context, sn string) (*coremodel.Device, error) {
	*d.called = true
	return d.inner.GetBySerialNumber(ctx, sn)
}
