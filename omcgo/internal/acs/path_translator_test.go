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

func TestTranslateTaskParams_Passthrough_WhenDisabled(t *testing.T) {
	// no dependencies → Enabled=false → original Params returned
	s := NewPathTranslationService(nil, nil, nil, zap.NewNop())
	in := json.RawMessage(`{"names":["Device.X."]}`)
	tk := &task.Task{ID: "t1", DeviceSN: "SN1", Method: "GetParameterValues", Params: in}
	out, changed := s.TranslateTaskParams(context.Background(), tk)
	assert.False(t, changed)
	assert.JSONEq(t, string(in), string(out))
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
