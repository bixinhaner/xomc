package mml

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// ────────────────────────────────────────────────────────────────────────────
// fakeDeviceLookup 实现 DeviceLookup 接口，返回预置 SN → product_class 映射。
// ────────────────────────────────────────────────────────────────────────────
type fakeDeviceLookup struct {
	devs map[string]*model.Device // sn → device
	err  error                    // 全局错误（模拟 DB 故障）
}

func (f *fakeDeviceLookup) GetBySerialNumber(_ context.Context, sn string) (*model.Device, error) {
	if f.err != nil {
		return nil, f.err
	}
	dev, ok := f.devs[sn]
	if !ok {
		return nil, nil
	}
	return dev, nil
}

// fakePathTranslator 实现 PathTranslator 接口，返回预置 standard → private 映射。
type fakePathTranslator struct {
	mapping map[string]string // standardPath → privatePath（缺失视为 passthrough）
	err     error
	calls   int
}

func (f *fakePathTranslator) TranslateForDevice(_ context.Context, _, _ string, paths []string) ([]TranslatedPath, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	out := make([]TranslatedPath, 0, len(paths))
	for _, p := range paths {
		if priv, ok := f.mapping[p]; ok {
			out = append(out, TranslatedPath{Standard: p, Private: priv, Source: "discovered"})
		} else {
			out = append(out, TranslatedPath{Standard: p, Private: p, Source: "passthrough"})
		}
	}
	return out, nil
}

func newServiceForTest() *Service {
	return &Service{logger: zap.NewNop()}
}

// ────────────────────────────────────────────────────────────────────────────
// R-8.4: product_class 一致性校验
// ────────────────────────────────────────────────────────────────────────────

func TestValidateDeviceProductClassUniform(t *testing.T) {
	mkDev := func(sn, pc string) *model.Device {
		return &model.Device{ID: uuid.New(), SerialNumber: sn, ProductClass: pc}
	}

	tests := []struct {
		name      string
		devs      map[string]*model.Device
		sns       []string
		lookupErr error
		want      error
		wantMixed []string // ErrMixedProductClass.Classes
	}{
		{
			name: "uniform single class → no error",
			devs: map[string]*model.Device{
				"SN1": mkDev("SN1", "Nova430E"),
				"SN2": mkDev("SN2", "Nova430E"),
			},
			sns: []string{"SN1", "SN2"},
		},
		{
			name: "mixed → ErrMixedProductClass with sorted classes",
			devs: map[string]*model.Device{
				"SN1": mkDev("SN1", "Nova430E"),
				"SN2": mkDev("SN2", "Nova227"),
				"SN3": mkDev("SN3", "Nova430E"),
			},
			sns:       []string{"SN1", "SN2", "SN3"},
			wantMixed: []string{"Nova227", "Nova430E"}, // sorted asc
		},
		{
			name: "all empty product_class → ErrNoValidDevices",
			devs: map[string]*model.Device{
				"SN1": mkDev("SN1", ""),
				"SN2": mkDev("SN2", ""),
			},
			sns:  []string{"SN1", "SN2"},
			want: ErrNoValidDevices,
		},
		{
			name: "unknown SNs → ErrNoValidDevices",
			devs: map[string]*model.Device{},
			sns:  []string{"GHOST"},
			want: ErrNoValidDevices,
		},
		{
			name: "empty sns → ErrNoValidDevices",
			sns:  []string{},
			want: ErrNoValidDevices,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newServiceForTest()
			s.SetDeviceLookup(&fakeDeviceLookup{devs: tc.devs, err: tc.lookupErr})

			err := s.validateDeviceProductClassUniform(context.Background(), tc.sns)

			if tc.want != nil {
				assert.ErrorIs(t, err, tc.want)
				return
			}
			if tc.wantMixed != nil {
				var mixedErr *ErrMixedProductClass
				require.ErrorAs(t, err, &mixedErr)
				assert.Equal(t, tc.wantMixed, mixedErr.Classes)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestValidateDeviceProductClassUniform_NoLookupInjected_Skips(t *testing.T) {
	s := newServiceForTest()
	// DeviceLookup 未注入 → 跳过校验
	err := s.validateDeviceProductClassUniform(context.Background(), []string{"SN1"})
	assert.NoError(t, err)
}

func TestErrMixedProductClass_CodeAndMessage(t *testing.T) {
	e := &ErrMixedProductClass{Classes: []string{"A", "B"}}
	assert.Equal(t, 17001, e.Code())
	assert.Contains(t, e.Error(), "A, B")
}

func TestErrProductClassUnresolved_CodeAndMessage(t *testing.T) {
	e := &ErrProductClassUnresolved{SN: "SN1", ProductClass: "Unknown"}
	assert.Equal(t, 17006, e.Code())
	assert.Contains(t, e.Error(), "SN1")
	assert.Contains(t, e.Error(), "Unknown")
}

// ────────────────────────────────────────────────────────────────────────────
// R-9.3: Translator 注入翻译
// ────────────────────────────────────────────────────────────────────────────

func TestTranslateTaskPaths_PassthroughAndDiscovered(t *testing.T) {
	s := newServiceForTest()
	s.SetDeviceLookup(&fakeDeviceLookup{devs: map[string]*model.Device{
		"SN1": {SerialNumber: "SN1", ProductClass: "Nova430E", FirmwareVersion: "v1.0"},
	}})
	tr := &fakePathTranslator{mapping: map[string]string{
		"Device.X.SoftwareVersion": "Device.X_VENDOR.SwVer",
	}}
	s.SetPathTranslator(tr)

	task := &MMLTask{
		DeviceSNs: []string{"SN1"},
		Commands: []map[string]interface{}{
			{
				"command_code":   "LST_DEVICE",
				"operation_type": "LST",
				"param_refs": []interface{}{
					map[string]interface{}{"param_path": "Device.X.SoftwareVersion"},
					map[string]interface{}{"param_path": "Device.X.UserLabel"},
				},
			},
		},
	}

	err := s.translateTaskPaths(context.Background(), task)
	require.NoError(t, err)
	assert.Equal(t, 1, tr.calls, "translator should be called once per task")

	refs := task.Commands[0]["param_refs"].([]interface{})
	r0 := refs[0].(map[string]interface{})
	assert.Equal(t, "Device.X_VENDOR.SwVer", r0["private_path"])
	assert.Equal(t, "discovered", r0["translation_source"])
	r1 := refs[1].(map[string]interface{})
	// passthrough：未命中 mapping，private_path 与 standardPath 相同
	assert.Equal(t, "Device.X.UserLabel", r1["private_path"])
	assert.Equal(t, "passthrough", r1["translation_source"])

	// translation_results 元数据已记录
	results := task.Commands[0]["translation_results"].([]TranslatedPath)
	assert.Len(t, results, 2)
}

func TestTranslateTaskPaths_ParametersKeyReplacement(t *testing.T) {
	s := newServiceForTest()
	s.SetDeviceLookup(&fakeDeviceLookup{devs: map[string]*model.Device{
		"SN1": {SerialNumber: "SN1", ProductClass: "Nova430E"},
	}})
	s.SetPathTranslator(&fakePathTranslator{mapping: map[string]string{
		"Device.X.UserLabel": "Device.X_VENDOR.Label",
	}})

	task := &MMLTask{
		DeviceSNs: []string{"SN1"},
		Commands: []map[string]interface{}{
			{
				"command_code":   "MOD_DEVICE",
				"operation_type": "MOD",
				"parameters": map[string]interface{}{
					"Device.X.UserLabel": "myDevice",
				},
			},
		},
	}

	err := s.translateTaskPaths(context.Background(), task)
	require.NoError(t, err)

	params := task.Commands[0]["parameters"].(map[string]interface{})
	// key 已替换为 privatePath
	assert.Equal(t, "myDevice", params["Device.X_VENDOR.Label"])
	_, hasOld := params["Device.X.UserLabel"]
	assert.False(t, hasOld, "old standardPath key should be removed")
}

func TestTranslateTaskPaths_NoTranslatorInjected_Skips(t *testing.T) {
	s := newServiceForTest()
	s.SetDeviceLookup(&fakeDeviceLookup{devs: map[string]*model.Device{
		"SN1": {SerialNumber: "SN1", ProductClass: "Nova430E"},
	}})
	task := &MMLTask{
		DeviceSNs: []string{"SN1"},
		Commands: []map[string]interface{}{
			{"param_refs": []interface{}{map[string]interface{}{"param_path": "Device.X.A"}}},
		},
	}
	err := s.translateTaskPaths(context.Background(), task)
	assert.NoError(t, err)
	// command entry 未被修改
	refs := task.Commands[0]["param_refs"].([]interface{})
	r0 := refs[0].(map[string]interface{})
	_, hasPriv := r0["private_path"]
	assert.False(t, hasPriv)
}

func TestTranslateTaskPaths_OrphanError(t *testing.T) {
	s := newServiceForTest()
	s.SetDeviceLookup(&fakeDeviceLookup{devs: map[string]*model.Device{
		"SN1": {SerialNumber: "SN1", ProductClass: "Unknown"},
	}})
	orphanErr := &ErrProductClassUnresolved{SN: "SN1", ProductClass: "Unknown"}
	s.SetPathTranslator(&fakePathTranslator{err: orphanErr})

	task := &MMLTask{
		DeviceSNs: []string{"SN1"},
		Commands: []map[string]interface{}{
			{"param_refs": []interface{}{map[string]interface{}{"param_path": "Device.X.A"}}},
		},
	}
	err := s.translateTaskPaths(context.Background(), task)
	var ue *ErrProductClassUnresolved
	require.ErrorAs(t, err, &ue)
}

// ────────────────────────────────────────────────────────────────────────────
// 路径收集函数：sort + dedup
// ────────────────────────────────────────────────────────────────────────────

func TestCollectStandardPaths(t *testing.T) {
	commands := []map[string]interface{}{
		{
			"param_refs": []interface{}{
				map[string]interface{}{"param_path": "Device.B"},
				map[string]interface{}{"param_path": "Device.A"},
			},
		},
		{
			"parameters": map[string]interface{}{
				"Device.A":    "v1",
				"Device.C":    "v2",
				"object_name": "Device.X.{i=1}.", // 不参与翻译
			},
		},
	}
	out := collectStandardPaths(commands)
	assert.Equal(t, []string{"Device.A", "Device.B", "Device.C"}, out)
}

// sanity check: import 路径完整
var _ = errors.New
