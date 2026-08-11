package mml

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

// fakePathTranslator 实现 PathTranslator 接口（T-0168 升级版返 *TranslationOutcome），
// 返回预置 standard → private 映射。
type fakePathTranslator struct {
	mapping map[string]string // standardPath → privatePath（缺失视为 passthrough）
	err     error
	// T-0168: 模拟 orphan_passthrough 场景（productClass 未识别）
	orphan       bool
	productClass string // 透传给 outcome.ProductClass
	calls        int
}

func (f *fakePathTranslator) TranslateForDevice(_ context.Context, productClass, _ string, paths []string) (*TranslationOutcome, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	if f.orphan {
		// T-0168 激进路线：productClass 未识别，全 orphan_passthrough，不返 error
		out := make([]TranslatedPath, 0, len(paths))
		for _, p := range paths {
			out = append(out, TranslatedPath{Standard: p, Private: p, Source: "orphan_passthrough"})
		}
		return &TranslationOutcome{
			Paths:           out,
			ProductResolved: false,
			ProductID:       uuid.Nil,
			ProductClass:    productClass,
			AggregateSource: "orphan_passthrough",
		}, nil
	}
	out := make([]TranslatedPath, 0, len(paths))
	hasHit, hasMiss := false, false
	for _, p := range paths {
		if priv, ok := f.mapping[p]; ok {
			out = append(out, TranslatedPath{Standard: p, Private: priv, Source: "discovered"})
			hasHit = true
		} else {
			out = append(out, TranslatedPath{Standard: p, Private: p, Source: "passthrough"})
			hasMiss = true
		}
	}
	aggregate := "discovered"
	switch {
	case hasHit && hasMiss:
		aggregate = "mixed"
	case !hasHit && hasMiss:
		aggregate = "passthrough"
	case !hasHit && !hasMiss: // 空 paths
		aggregate = ""
	}
	return &TranslationOutcome{
		Paths:           out,
		ProductResolved: true,
		ProductID:       uuid.New(), // 测试用 product.id；真实场景由 ProductRegistry 返
		ProductClass:    productClass,
		AggregateSource: aggregate,
	}, nil
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
				"param_refs": []MMLParamRef{
					{Tr069Path: "Device.X.SoftwareVersion"},
					{Tr069Path: "Device.X.UserLabel"},
				},
			},
		},
	}

	err := s.translateTaskPaths(context.Background(), task)
	require.NoError(t, err)
	assert.Equal(t, 1, tr.calls, "translator should be called once per task")

	refs := task.Commands[0]["param_refs"].([]MMLParamRef)
	assert.Equal(t, "Device.X_VENDOR.SwVer", refs[0].PrivatePath)
	assert.Equal(t, "discovered", refs[0].TranslationSource)
	// passthrough：fakeTranslator 仍返 outcome.Paths 含 entry，private = standardPath
	assert.Equal(t, "Device.X.UserLabel", refs[1].PrivatePath)
	assert.Equal(t, "passthrough", refs[1].TranslationSource)

	// translation_results 元数据已记录
	results := task.Commands[0]["translation_results"].([]TranslatedPath)
	assert.Len(t, results, 2)
}

// TestTranslateTaskPaths_MODUsesParamRefs 验证 MOD 命令的 path 翻译走 param_refs.Tr069Path，
// 不依赖 parameters keys（parameters 的 key 是 MMLCode leaf 段，非 standardPath，禁止当 path 用）。
func TestTranslateTaskPaths_MODUsesParamRefs(t *testing.T) {
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
				"param_refs": []MMLParamRef{
					{ParamCode: "UserLabel", Tr069Path: "Device.X.UserLabel"},
				},
				// parameters key 是 MMLCode（leaf 段），不是 standardPath；不参与翻译
				"parameters": map[string]interface{}{
					"UserLabel": "myDevice",
				},
			},
		},
	}

	err := s.translateTaskPaths(context.Background(), task)
	require.NoError(t, err)

	// param_refs 上正确写回 PrivatePath
	refs := task.Commands[0]["param_refs"].([]MMLParamRef)
	assert.Equal(t, "Device.X_VENDOR.Label", refs[0].PrivatePath)
	assert.Equal(t, "discovered", refs[0].TranslationSource)

	// parameters 不动（SOAP 层用 param_refs 查 MMLCode → PrivatePath）
	params := task.Commands[0]["parameters"].(map[string]interface{})
	assert.Equal(t, "myDevice", params["UserLabel"], "parameters keys are MMLCode and stay untouched")
}

func TestTranslateTaskPaths_ADDNormalizesNewInstancePlaceholder(t *testing.T) {
	s := newServiceForTest()
	s.SetDeviceLookup(&fakeDeviceLookup{devs: map[string]*model.Device{
		"SN1": {SerialNumber: "SN1", ProductClass: "FAP/BU1810"},
	}})
	modelPath := "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.5GCell.{i}.CellId"
	runtimePath := strings.Replace(modelPath, "{i}", "{NEW}", 1)
	translationPath := strings.Replace(runtimePath, "{NEW}", newInstanceTranslationSentinel, 1)
	s.SetPathTranslator(&fakePathTranslator{mapping: map[string]string{
		translationPath: translationPath,
	}})

	task := &MMLTask{
		DeviceSNs: []string{"SN1"},
		Commands: []map[string]interface{}{{
			"command_code":   "ADD 5G_CELL",
			"operation_type": "MOD",
			"param_refs": []MMLParamRef{{
				ParamCode: "CELL_ID",
				Tr069Path: runtimePath,
			}},
		}},
	}

	require.NoError(t, s.translateTaskPaths(context.Background(), task))
	refs := task.Commands[0]["param_refs"].([]MMLParamRef)
	assert.Equal(t, "discovered", refs[0].TranslationSource)
	assert.Contains(t, refs[0].PrivatePath, ".{NEW}.")
}

func TestTranslateTaskPaths_ADDHandlesEveryMultiInstanceDepth(t *testing.T) {
	s := newServiceForTest()
	s.SetDeviceLookup(&fakeDeviceLookup{devs: map[string]*model.Device{
		"SN1": {SerialNumber: "SN1", ProductClass: "FAP/BU1810"},
	}})

	runtimePaths := []string{
		"Device.Ethernet.Interface.{NEW}.Name",
		"Device.Ethernet.Interface.2.IPv4Address.{NEW}.IPAddress",
		"Device.Services.FAPService.1.CellConfig.2.NR.RAN.NeighborList.NRCell.{NEW}.CID",
	}
	mapping := make(map[string]string, len(runtimePaths))
	for _, path := range runtimePaths {
		lookup := normalizeNewInstancePlaceholder(path)
		mapping[lookup] = strings.Replace(lookup, "Device.", "Device.X_VENDOR.", 1)
	}
	s.SetPathTranslator(&fakePathTranslator{mapping: mapping})

	refs := make([]MMLParamRef, len(runtimePaths))
	for i, path := range runtimePaths {
		refs[i] = MMLParamRef{ParamCode: fmt.Sprintf("P%d", i), Tr069Path: path}
	}
	task := &MMLTask{
		DeviceSNs: []string{"SN1"},
		Commands: []map[string]interface{}{{
			"command_code":   "ADD MULTI_INSTANCE",
			"operation_type": "MOD",
			"param_refs":     refs,
		}},
	}

	require.NoError(t, s.translateTaskPaths(context.Background(), task))
	translated := task.Commands[0]["param_refs"].([]MMLParamRef)
	for i := range translated {
		assert.Equal(t, "discovered", translated[i].TranslationSource)
		assert.Contains(t, translated[i].PrivatePath, ".{NEW}.")
		assert.NotContains(t, translated[i].PrivatePath, newInstanceTranslationSentinel)
		assert.Equal(t, runtimePaths[i], translated[i].Tr069Path)
	}
}

func TestTranslateTaskPaths_NoTranslatorInjected_Skips(t *testing.T) {
	s := newServiceForTest()
	s.SetDeviceLookup(&fakeDeviceLookup{devs: map[string]*model.Device{
		"SN1": {SerialNumber: "SN1", ProductClass: "Nova430E"},
	}})
	task := &MMLTask{
		DeviceSNs: []string{"SN1"},
		Commands: []map[string]interface{}{
			{"param_refs": []MMLParamRef{{Tr069Path: "Device.X.A"}}},
		},
	}
	err := s.translateTaskPaths(context.Background(), task)
	assert.NoError(t, err)
	// command entry 未被修改
	refs := task.Commands[0]["param_refs"].([]MMLParamRef)
	assert.Equal(t, "", refs[0].PrivatePath, "translator 未注入时不应写 PrivatePath")
}

func TestTranslateTaskPaths_OrphanError(t *testing.T) {
	// 向后兼容：旧版适配器仍返 ErrProductClassUnresolved 时上层正确传播。
	// T-0168 新适配器已改为内部消化为 orphan_passthrough（见
	// TestTranslateTaskPaths_OrphanPassthrough），保留此测试防退化。
	s := newServiceForTest()
	s.SetDeviceLookup(&fakeDeviceLookup{devs: map[string]*model.Device{
		"SN1": {SerialNumber: "SN1", ProductClass: "Unknown"},
	}})
	orphanErr := &ErrProductClassUnresolved{SN: "SN1", ProductClass: "Unknown"}
	s.SetPathTranslator(&fakePathTranslator{err: orphanErr})

	task := &MMLTask{
		DeviceSNs: []string{"SN1"},
		Commands: []map[string]interface{}{
			{"param_refs": []MMLParamRef{{Tr069Path: "Device.X.A"}}},
		},
	}
	err := s.translateTaskPaths(context.Background(), task)
	var ue *ErrProductClassUnresolved
	require.ErrorAs(t, err, &ue)
}

// TestTranslateTaskPaths_OrphanPassthrough 验证 T-0168 激进路线：productClass 未识别
// 时整任务全部走 orphan_passthrough，task 上的 4 个翻译审计字段被正确写入，且不返 error。
func TestTranslateTaskPaths_OrphanPassthrough(t *testing.T) {
	s := newServiceForTest()
	s.SetDeviceLookup(&fakeDeviceLookup{devs: map[string]*model.Device{
		"SN1": {SerialNumber: "SN1", ProductClass: "UNKNOWN-2025"},
	}})
	s.SetPathTranslator(&fakePathTranslator{orphan: true})

	task := &MMLTask{
		DeviceSNs: []string{"SN1"},
		Commands: []map[string]interface{}{
			{
				"param_refs": []MMLParamRef{
					{Tr069Path: "Device.X.Foo"},
					{Tr069Path: "Device.X.Bar"},
				},
			},
		},
	}
	err := s.translateTaskPaths(context.Background(), task)
	require.NoError(t, err, "orphan 路径应不阻塞 fanout")

	// 任务级翻译审计字段
	assert.False(t, task.ProductResolved, "ProductResolved 应为 false")
	assert.Nil(t, task.MatchedProductID, "MatchedProductID 应为 nil")
	assert.Equal(t, "UNKNOWN-2025", task.MatchedProductClass, "MatchedProductClass 应透传 productClass")
	assert.Equal(t, "orphan_passthrough", task.PathTranslationSource, "PathTranslationSource 应为 orphan_passthrough")

	// param_refs 上的 per-path 翻译标记
	refs := task.Commands[0]["param_refs"].([]MMLParamRef)
	for _, ref := range refs {
		assert.Equal(t, ref.Tr069Path, ref.PrivatePath, "orphan 时 PrivatePath 应等于 standardPath")
		assert.Equal(t, "orphan_passthrough", ref.TranslationSource, "Source 应为 orphan_passthrough")
	}
}

// TestTranslateTaskPaths_MixedSource 验证 task.PathTranslationSource 在
// discovered + passthrough 混合时正确判定为 "mixed"。
func TestTranslateTaskPaths_MixedSource(t *testing.T) {
	s := newServiceForTest()
	s.SetDeviceLookup(&fakeDeviceLookup{devs: map[string]*model.Device{
		"SN1": {SerialNumber: "SN1", ProductClass: "BaiBLQ_5.0.16"},
	}})
	// 第一条命中 mapping → discovered；第二条未命中 → passthrough
	s.SetPathTranslator(&fakePathTranslator{mapping: map[string]string{
		"Device.Standard.Path1": "InternetGw.X_VENDOR.Path1",
	}})

	task := &MMLTask{
		DeviceSNs: []string{"SN1"},
		Commands: []map[string]interface{}{
			{
				"param_refs": []MMLParamRef{
					{Tr069Path: "Device.Standard.Path1"},
					{Tr069Path: "Device.Standard.Path2"},
				},
			},
		},
	}
	err := s.translateTaskPaths(context.Background(), task)
	require.NoError(t, err)

	assert.True(t, task.ProductResolved)
	assert.NotNil(t, task.MatchedProductID, "命中 product 时 MatchedProductID 应非 nil")
	assert.Equal(t, "mixed", task.PathTranslationSource, "discovered+passthrough 混合应为 mixed")
}

// TestExtractPathTranslations 验证 service.GetTaskResults 调用的 extractPathTranslations
// 从 task.Commands JSONB 中正确抽出去重的 PathTranslationView 列表（T-0168）。
func TestExtractPathTranslations(t *testing.T) {
	commands := []map[string]interface{}{
		{
			"param_refs": []interface{}{
				map[string]interface{}{
					"tr069_path":         "Device.A",
					"private_path":       "InternetGw.X_VENDOR.A",
					"translation_source": "discovered",
				},
				map[string]interface{}{
					"tr069_path":         "Device.B",
					"private_path":       "Device.B",
					"translation_source": "passthrough",
				},
			},
		},
		{
			"param_refs": []interface{}{
				// 同 standardPath 去重保留一份
				map[string]interface{}{
					"tr069_path":         "Device.A",
					"private_path":       "InternetGw.X_VENDOR.A",
					"translation_source": "discovered",
				},
			},
		},
	}
	got := extractPathTranslations(commands)
	require.Len(t, got, 2, "Device.A / Device.B 去重应为 2 条")
	// 按 standardPath 字母序排序
	assert.Equal(t, "Device.A", got[0].StandardPath)
	assert.Equal(t, "InternetGw.X_VENDOR.A", got[0].PrivatePath)
	assert.Equal(t, "discovered", got[0].TranslationSource)
	assert.True(t, got[0].Translated)
	assert.Equal(t, "Device.B", got[1].StandardPath)
	assert.Equal(t, "passthrough", got[1].TranslationSource)
	assert.False(t, got[1].Translated, "passthrough 算未翻译")
}

// ────────────────────────────────────────────────────────────────────────────
// 路径收集函数：sort + dedup
// ────────────────────────────────────────────────────────────────────────────

func TestCollectStandardPaths(t *testing.T) {
	commands := []map[string]interface{}{
		{
			"param_refs": []MMLParamRef{
				{Tr069Path: "Device.B"},
				{Tr069Path: "Device.A"},
			},
		},
		{
			// 模拟 JSONB 反序列化路径：[]interface{} + 字段 tr069_path
			"param_refs": []interface{}{
				map[string]interface{}{"tr069_path": "Device.C"},
			},
			// parameters key 不再视作 standardPath（MOD 的 key 是 MMLCode leaf）
			"parameters": map[string]interface{}{
				"UserLabel":   "v1",
				"object_name": "Device.X.{i=1}.",
			},
		},
	}
	out := collectStandardPaths(commands)
	assert.Equal(t, []string{"Device.A", "Device.B", "Device.C"}, out)
}

// sanity check: import 路径完整
var _ = errors.New
