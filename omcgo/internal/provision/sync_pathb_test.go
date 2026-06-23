package provision

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/model"
)

func TestExtractStorablePrefixes_HappyPath(t *testing.T) {
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "Dev.WiFi.SSID.{i}.Enabled", IsStorable: true, IsSupported: true, EntryType: "parameter"},
		{PrivatePath: "Dev.WiFi.SSID.{i}.Name", IsStorable: true, IsSupported: true, EntryType: "parameter"},
		{PrivatePath: "Dev.WiFi.Radio.{i}.Channel", IsStorable: true, IsSupported: true, EntryType: "parameter"},
		{PrivatePath: "Dev.System.Mode", IsStorable: true, IsSupported: true, EntryType: "parameter"},
		{PrivatePath: "Dev.NotStorable", IsStorable: false, IsSupported: true, EntryType: "parameter"}, // 应被过滤
	}
	got := extractStorablePrefixes(mappings)
	// {i} 模板截到对象前缀；普通叶子原样下发
	want := []string{"Dev.System.Mode", "Dev.WiFi.Radio.", "Dev.WiFi.SSID."}
	sort.Strings(got)
	assert.Equal(t, want, got)
}

func TestExtractStorablePrefixes_DedupesIdenticalPrefixes(t *testing.T) {
	// 多条同前缀 → 只保留一条
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "Dev.A.{i}.X", IsStorable: true, IsSupported: true, EntryType: "parameter"},
		{PrivatePath: "Dev.A.{i}.Y", IsStorable: true, IsSupported: true, EntryType: "parameter"},
		{PrivatePath: "Dev.A.{i}.Z", IsStorable: true, IsSupported: true, EntryType: "parameter"},
	}
	got := extractStorablePrefixes(mappings)
	assert.Equal(t, []string{"Dev.A."}, got)
}

func TestExtractStorablePrefixes_AllNonStorable_Empty(t *testing.T) {
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "Dev.A.B", IsStorable: false, IsSupported: true},
		{PrivatePath: "Dev.X.Y", IsStorable: false, IsSupported: true},
	}
	got := extractStorablePrefixes(mappings)
	assert.Empty(t, got)
}

func TestExtractStorablePrefixes_ObjectPaths(t *testing.T) {
	// 末尾 "." 的 object 路径原样使用
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "Dev.WiFi.", IsStorable: true, IsSupported: true, EntryType: "object"},
		{PrivatePath: "Dev.System.", IsStorable: true, IsSupported: true, EntryType: "object"},
	}
	got := extractStorablePrefixes(mappings)
	sort.Strings(got)
	assert.Equal(t, []string{"Dev.System.", "Dev.WiFi."}, got)
}

func TestExtractStorablePrefixes_EmptyFiltered_NoDotKept(t *testing.T) {
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "", IsStorable: true, IsSupported: true},      // 空串 → 过滤
		{PrivatePath: "NoDot", IsStorable: true, IsSupported: true}, // 无 "." → 原样保留，由基站判定
	}
	got := extractStorablePrefixes(mappings)
	assert.Equal(t, []string{"NoDot"}, got)
}

func TestExtractStorablePrefixes_LeafKeptAsIs(t *testing.T) {
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "Device.X.Y.Z", IsStorable: true, IsSupported: true, EntryType: "parameter"},
	}
	got := extractStorablePrefixes(mappings)
	assert.Equal(t, []string{"Device.X.Y.Z"}, got)
}

func TestExtractStorablePrefixes_SingletonLeafExpanded(t *testing.T) {
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "Device.Services.FAPService.{i}.AmfsStatus", IsStorable: true, IsSupported: true, EntryType: "parameter"},
	}
	got := extractStorablePrefixes(mappings)
	assert.Equal(t, []string{"Device.Services.FAPService.1.AmfsStatus"}, got)
}

func TestExtractStorablePrefixes_FAPServiceNeighborObjectsStayScoped(t *testing.T) {
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.5GCell.", IsStorable: true, IsSupported: true, EntryType: "object"},
		{PrivatePath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.", IsStorable: true, IsSupported: true, EntryType: "object"},
		{PrivatePath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.CID", IsStorable: true, IsSupported: true, EntryType: "parameter"},
	}
	got := extractStorablePrefixes(mappings)
	sort.Strings(got)
	assert.Equal(t, []string{
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.5GCell.",
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.",
	}, got)
}

func TestExtractStorablePrefixes_LeavesNotMerged(t *testing.T) {
	// 同父对象下的多个叶子参数 → 各自原样保留，不合并到父前缀
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "Device.X.Y", IsStorable: true, IsSupported: true, EntryType: "parameter"},
		{PrivatePath: "Device.X.Z", IsStorable: true, IsSupported: true, EntryType: "parameter"},
	}
	got := extractStorablePrefixes(mappings)
	sort.Strings(got)
	assert.Equal(t, []string{"Device.X.Y", "Device.X.Z"}, got)
}

func TestExtractStorablePrefixes_UnsupportedFiltered(t *testing.T) {
	// T-0103：is_supported=false 条目即便 is_storable=true 也被过滤掉
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "Dev.A.Good", IsStorable: true, IsSupported: true, EntryType: "parameter"},
		{PrivatePath: "Dev.A.AmbrLimitSwitch", IsStorable: true, IsSupported: false, EntryType: "parameter"},
		{PrivatePath: "Dev.A.RunningStatus", IsStorable: true, IsSupported: false, EntryType: "parameter"},
		{PrivatePath: "Dev.B.{i}.Bad", IsStorable: true, IsSupported: false, EntryType: "parameter"},
	}
	got := extractStorablePrefixes(mappings)
	assert.Equal(t, []string{"Dev.A.Good"}, got)
}

func TestExtractStorablePrefixes_AllUnsupported_Empty(t *testing.T) {
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "Dev.X", IsStorable: true, IsSupported: false},
		{PrivatePath: "Dev.Y", IsStorable: true, IsSupported: false},
	}
	got := extractStorablePrefixes(mappings)
	assert.Empty(t, got)
}

func TestBasePrefix_Cases(t *testing.T) {
	cases := []struct {
		in, out string
	}{
		{"Dev.WiFi.SSID.{i}.Enabled", "Dev.WiFi.SSID."}, // {i} 模板截到对象前缀
		{"Dev.WiFi.SSID.{i}.{i}.Foo", "Dev.WiFi.SSID."}, // 截到第一个 {i}
		{"Dev.WiFi.SSID.", "Dev.WiFi.SSID."},            // object 原样
		{"Dev.System.Mode", "Dev.System.Mode"},          // 叶子原样
		{"Dev", "Dev"},                                  // 无 "." 原样
		{"", ""},                                        // 空串
		{"Dev.", "Dev."},                                // 末尾点原样
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			assert.Equal(t, c.out, basePrefix(c.in))
		})
	}
}

func TestNormalizeSingletonFAPServicePath_Cases(t *testing.T) {
	cases := []struct {
		in, out string
	}{
		{"Device.Services.FAPService.{i}.AmfsStatus", "Device.Services.FAPService.1.AmfsStatus"},
		{"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.5GCell.", "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.5GCell."},
		{"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.CID", "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.CID"},
		{"Device.DeviceInfo.SerialNumber", "Device.DeviceInfo.SerialNumber"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			assert.Equal(t, c.out, normalizeSingletonFAPServicePath(c.in))
		})
	}
}

func TestInstantiateStandardPath_NoPlaceholder(t *testing.T) {
	// 模板 standardPath 无 {i} → 直接返回
	got := instantiateStandardPath("Dev.System.Mode", "Dev.System.Mode", "Device.System.Mode")
	assert.Equal(t, "Device.System.Mode", got)
}

func TestInstantiateStandardPath_SingleInstance(t *testing.T) {
	got := instantiateStandardPath(
		"Dev.WiFi.SSID.7.Enabled",
		"Dev.WiFi.SSID.{i}.Enabled",
		"Device.WiFi.SSID.{i}.Enable",
	)
	assert.Equal(t, "Device.WiFi.SSID.7.Enable", got)
}

func TestInstantiateStandardPath_MultipleInstances(t *testing.T) {
	got := instantiateStandardPath(
		"Foo.1.Bar.2.Baz",
		"Foo.{i}.Bar.{i}.Baz",
		"Foo.{i}.Bar.{i}.Baz",
	)
	assert.Equal(t, "Foo.1.Bar.2.Baz", got)
}

func TestInstantiateStandardPath_PartialOverlap_LeftFirst(t *testing.T) {
	// privatePath 首个 {i} 段对应实例号"3"，第二个对应"5"
	got := instantiateStandardPath(
		"P.3.Q.5.R",
		"P.{i}.Q.{i}.R",
		"S.{i}.T.{i}.U",
	)
	assert.Equal(t, "S.3.T.5.U", got)
}

func TestInstantiateStandardPath_LengthMismatch_Fallback(t *testing.T) {
	// 段数不一致 → 容错返回模板原文
	got := instantiateStandardPath(
		"Foo.1",
		"Foo.{i}.Bar",
		"Device.{i}.Bar",
	)
	assert.Equal(t, "Device.{i}.Bar", got)
}

func TestInstantiateStandardPath_NoPlaceholderInTemplate(t *testing.T) {
	// 即便 actualPrivate 含数字段，templateStandard 无 {i} → 原样返回
	got := instantiateStandardPath(
		"Dev.WiFi.SSID.7.Enabled",
		"Dev.WiFi.SSID.{i}.Enabled",
		"Device.WiFi.SSID.Enable",
	)
	assert.Equal(t, "Device.WiFi.SSID.Enable", got)
}

// ---------------------------------------------------------------------------
// Tests: T-0127 Path B 同步差异日志
// ---------------------------------------------------------------------------

// fakeParamRepoForDiff 提供 GetByDevice 桩供 snapshotStandardPaths 测试用。
type fakeParamRepoForDiff struct {
	deviceParameterRepoStub
	paths []string
	err   error
}

func (f *fakeParamRepoForDiff) GetByDevice(_ context.Context, _ uuid.UUID) ([]model.DeviceParameter, error) {
	if f.err != nil {
		return nil, f.err
	}
	params := make([]model.DeviceParameter, 0, len(f.paths))
	for _, p := range f.paths {
		params = append(params, model.DeviceParameter{ParameterPath: p})
	}
	return params, nil
}

// deviceParameterRepoStub 实现 device.DeviceParameterRepository 接口的空桩（仅用 GetByDevice）。
type deviceParameterRepoStub struct{}

func (deviceParameterRepoStub) BatchUpsert(_ context.Context, _ uuid.UUID, _ []model.DeviceParameter) error {
	return nil
}
func (deviceParameterRepoStub) GetByDevice(_ context.Context, _ uuid.UUID) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (deviceParameterRepoStub) GetByPath(_ context.Context, _ uuid.UUID, _ string) (*model.DeviceParameter, error) {
	return nil, nil
}
func (deviceParameterRepoStub) DeleteByDevice(_ context.Context, _ uuid.UUID) error { return nil }
func (deviceParameterRepoStub) DeleteByPathPrefix(_ context.Context, _ uuid.UUID, _ string) (int64, error) {
	return 0, nil
}
func (deviceParameterRepoStub) GetByPathPrefix(_ context.Context, _ uuid.UUID, _ string) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (deviceParameterRepoStub) CountByPathPrefix(_ context.Context, _ uuid.UUID, _ string) (int, error) {
	return 0, nil
}
func (deviceParameterRepoStub) SearchByKeyword(_ context.Context, _ uuid.UUID, _ string, _ int) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (deviceParameterRepoStub) GetDirectChildLeaves(_ context.Context, _ uuid.UUID, _ string, _ int, _ int) ([]model.DeviceParameter, int, error) {
	return nil, 0, nil
}
func (deviceParameterRepoStub) GetByGroup(_ context.Context, _ uuid.UUID, _ string) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (deviceParameterRepoStub) GetByFAPInstance(_ context.Context, _ uuid.UUID, _ int) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (deviceParameterRepoStub) GetByFAPInstanceAndGroup(_ context.Context, _ uuid.UUID, _ int, _ string) ([]model.DeviceParameter, error) {
	return nil, nil
}

// newDiffTestService 构造一个可观测 logger + 可注入 paramRepo/redisClient 的 SyncService。
func newDiffTestService(t *testing.T, paths []string, paramErr error) (*SyncService, *observer.ObservedLogs, *miniredis.Miniredis) {
	t.Helper()
	core, recorded := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	svc := &SyncService{
		paramRepo:   &fakeParamRepoForDiff{paths: paths, err: paramErr},
		redisClient: rdb,
		logger:      logger,
	}
	return svc, recorded, mr
}

// ---------------------------------------------------------------------------
// Tests: line 52 reconcile 差集删除（全量同步语义）
// ---------------------------------------------------------------------------

// fakeParamRepoForReconcile 实现 DeleteByPathPrefix 跟踪以验证差集删除调用。
type fakeParamRepoForReconcile struct {
	deviceParameterRepoStub
	getPaths     []string
	deletedPaths []string
	deleteErr    error
}

func (f *fakeParamRepoForReconcile) GetByDevice(_ context.Context, _ uuid.UUID) ([]model.DeviceParameter, error) {
	out := make([]model.DeviceParameter, 0, len(f.getPaths))
	for _, p := range f.getPaths {
		out = append(out, model.DeviceParameter{ParameterPath: p})
	}
	return out, nil
}

func (f *fakeParamRepoForReconcile) DeleteByPathPrefix(_ context.Context, _ uuid.UUID, prefix string) (int64, error) {
	if f.deleteErr != nil {
		return 0, f.deleteErr
	}
	f.deletedPaths = append(f.deletedPaths, prefix)
	return 1, nil
}

func TestNearestObjectPrefix_Cases(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		// 深层多实例：i=5，prefix 5 段，>=4 OK
		{"Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.2.Pci",
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell."},
		// i=4，prefix 4 段，>=4 OK
		{"Device.X.Y.Z.1.W", "Device.X.Y.Z."},
		// i=3，prefix 3 段，<4 拒
		{"Device.WiFi.SSID.1.Enable", ""},
		// i=2，prefix 2 段，<4 拒 — 防误删大爆炸（实测 BLQ 触发场景）
		{"Device.DeviceInfo.2.UE_Count", ""},
		// 叶子（无数字段）→ 不参与
		{"Device.System.Mode", ""},
		{"Device.DeviceInfo.SoftwareVersion", ""},
		{"", ""},
		// 全数字段（不太可能但兜底）— i=2, 拒
		{"1.2.3", ""},
		// 多层嵌套实例：取最深的数字段 "2"（i=5），>=4 OK
		{"Device.A.B.C.1.X.2.Y", "Device.A.B.C.1.X."},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			assert.Equal(t, c.want, nearestObjectPrefix(c.in))
		})
	}
}

func TestDeriveObjectPrefixesFromParams_DedupAndSkipLeaves(t *testing.T) {
	params := []model.DeviceParameter{
		// 深层 prefix OK
		{ParameterPath: "Device.A.B.C.1.X.Y"},
		{ParameterPath: "Device.A.B.C.2.X.Y"},
		{ParameterPath: "Device.A.B.D.1.X.Y"},
		// 浅 prefix（<4 段）→ 拒
		{ParameterPath: "Device.X.1.Y"},
		// 叶子 → 跳过
		{ParameterPath: "Device.System.Mode"},
		{ParameterPath: ""},
	}
	got := deriveObjectPrefixesFromParams(params)
	assert.ElementsMatch(t, []string{"Device.A.B.C.", "Device.A.B.D."}, got)
}

func TestPathInAnyPrefix(t *testing.T) {
	prefixes := []string{"Device.WiFi.SSID.", "Device.WiFi.Radio."}
	assert.True(t, pathInAnyPrefix("Device.WiFi.SSID.1.Enable", prefixes))
	assert.True(t, pathInAnyPrefix("Device.WiFi.SSID.99.Name", prefixes))
	assert.True(t, pathInAnyPrefix("Device.WiFi.Radio.1.Channel", prefixes))
	assert.False(t, pathInAnyPrefix("Device.System.Mode", prefixes))
	assert.False(t, pathInAnyPrefix("Device.Services.FAPService.1.X", prefixes))
}

func TestReconcileDeletedPaths_DeletesMissingInstances(t *testing.T) {
	repo := &fakeParamRepoForReconcile{}
	svc := &SyncService{paramRepo: repo, logger: zap.NewNop()}
	dev := &model.Device{ID: uuid.New(), SerialNumber: "BLQ-001"}
	// 用真实 BLQ 场景 path（>=4 段深层多实例，nearestObjectPrefix 会推导出 prefix）
	// CPE 返回 LTECell.1、LTECell.3；DB 中还有 LTECell.2、LTECell.4（被 CPE 删了）+ 其他范围数据
	lt := "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell."
	prev := map[string]struct{}{
		lt + "1.Pci":                {},
		lt + "2.Pci":                {},
		lt + "3.Pci":                {},
		lt + "4.Pci":                {},
		"Device.WiFi.SSID.1.Enable": {}, // 浅 prefix（3 段）不在 reconcile 范围
		"Device.System.Mode":        {}, // 叶子
	}
	params := []model.DeviceParameter{
		{ParameterPath: lt + "1.Pci"},
		{ParameterPath: lt + "3.Pci"},
	}
	svc.reconcileDeletedPaths(context.Background(), dev, prev, params)
	assert.ElementsMatch(t, []string{
		lt + "2.Pci",
		lt + "4.Pci",
	}, repo.deletedPaths)
}

func TestReconcileDeletedPaths_NoMissing_NoDelete(t *testing.T) {
	repo := &fakeParamRepoForReconcile{}
	svc := &SyncService{paramRepo: repo, logger: zap.NewNop()}
	dev := &model.Device{ID: uuid.New(), SerialNumber: "BLQ-001"}
	lt := "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell."
	prev := map[string]struct{}{lt + "1.Pci": {}}
	params := []model.DeviceParameter{{ParameterPath: lt + "1.Pci"}}
	svc.reconcileDeletedPaths(context.Background(), dev, prev, params)
	assert.Empty(t, repo.deletedPaths)
}

func TestReconcileDeletedPaths_AllLeavesNoOp(t *testing.T) {
	// CPE 全返回叶子 path（无数字段）→ 推导不出 prefix → 不删
	repo := &fakeParamRepoForReconcile{}
	svc := &SyncService{paramRepo: repo, logger: zap.NewNop()}
	dev := &model.Device{ID: uuid.New(), SerialNumber: "BLQ-001"}
	prev := map[string]struct{}{
		"Device.System.Mode": {},
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.5.Stale": {},
	}
	params := []model.DeviceParameter{{ParameterPath: "Device.System.Mode"}}
	svc.reconcileDeletedPaths(context.Background(), dev, prev, params)
	assert.Empty(t, repo.deletedPaths, "全叶子响应不应触发任何删除（避免误删未涵盖的多实例数据）")
}

func TestReconcileDeletedPaths_RejectsShallowInstances(t *testing.T) {
	// 浅层 prefix（如 Device.DeviceInfo.2.UE_Count，数字段在第 2 位）门槛拒，不删
	// 防止 BLQ 实测中 Device.DeviceInfo.2.* 路径推导出 Device.DeviceInfo. 覆盖整个子树
	repo := &fakeParamRepoForReconcile{}
	svc := &SyncService{paramRepo: repo, logger: zap.NewNop()}
	dev := &model.Device{ID: uuid.New(), SerialNumber: "BLQ-001"}
	prev := map[string]struct{}{
		"Device.DeviceInfo.WAN_CONFIG11_DEFAULTGW": {},
		"Device.DeviceInfo.SoftwareVersion":        {},
	}
	params := []model.DeviceParameter{
		{ParameterPath: "Device.DeviceInfo.2.UE_Count"}, // 浅 prefix（i=2 < 4 门槛）→ skip
	}
	svc.reconcileDeletedPaths(context.Background(), dev, prev, params)
	assert.Empty(t, repo.deletedPaths)
}

func TestLogPathBSyncDiff_LogsMissingPaths_InfoLevel(t *testing.T) {
	svc, recorded, mr := newDiffTestService(t, nil, nil)
	deviceID := uuid.New()
	dev := &model.Device{ID: deviceID, SerialNumber: "SN-DIFF-001"}

	// 写 reason 标签
	require.NoError(t, mr.Set("provision:syncreason:"+deviceID.String(), "device_online"))

	prev := map[string]struct{}{
		"Device.WiFi.SSID.1.Enable": {},
		"Device.WiFi.SSID.2.Enable": {},
		"Device.WiFi.SSID.3.Enable": {},
		"Device.WiFi.SSID.4.Enable": {},
	}
	params := []model.DeviceParameter{
		{ParameterPath: "Device.WiFi.SSID.1.Enable"},
		{ParameterPath: "Device.WiFi.SSID.2.Enable"},
		{ParameterPath: "Device.WiFi.SSID.3.Enable"},
		// SSID.4 缺失
	}
	svc.logPathBSyncDiff(context.Background(), dev, prev, params)

	entries := recorded.FilterMessage("param_sync_missing").All()
	require.Len(t, entries, 1, "应产出一条 param_sync_missing 日志")
	assert.Equal(t, zap.InfoLevel, entries[0].Level)

	fields := entries[0].ContextMap()
	assert.Equal(t, "device_online", fields["reason"])
	assert.Equal(t, int64(4), fields["prev_total"])
	assert.Equal(t, int64(3), fields["current_total"])
	assert.Equal(t, int64(1), fields["missing_count"])
}

func TestLogPathBSyncDiff_SkipsOnFirstSync(t *testing.T) {
	svc, recorded, _ := newDiffTestService(t, nil, nil)
	dev := &model.Device{ID: uuid.New(), SerialNumber: "SN-FIRST"}

	prev := map[string]struct{}{} // 空 — 首次同步
	params := []model.DeviceParameter{
		{ParameterPath: "Device.X.Y.1.A"},
	}
	svc.logPathBSyncDiff(context.Background(), dev, prev, params)

	assert.Empty(t, recorded.FilterMessage("param_sync_missing").All(),
		"集合 B 为空（首次同步）应跳过差异日志避免噪声")
}

func TestLogPathBSyncDiff_NoMissing_NoLog(t *testing.T) {
	svc, recorded, _ := newDiffTestService(t, nil, nil)
	dev := &model.Device{ID: uuid.New(), SerialNumber: "SN-NOMISS"}

	prev := map[string]struct{}{
		"Device.X.A": {},
		"Device.X.B": {},
	}
	params := []model.DeviceParameter{
		{ParameterPath: "Device.X.A"},
		{ParameterPath: "Device.X.B"},
		{ParameterPath: "Device.X.C"}, // 多出的不算 missing
	}
	svc.logPathBSyncDiff(context.Background(), dev, prev, params)

	assert.Empty(t, recorded.FilterMessage("param_sync_missing").All(),
		"无 missing 时不应输出日志")
}

func TestLogPathBSyncDiff_WarnLevel_WhenSetTooSmall(t *testing.T) {
	svc, recorded, _ := newDiffTestService(t, nil, nil)
	dev := &model.Device{ID: uuid.New(), SerialNumber: "SN-WARN"}

	prev := make(map[string]struct{}, 10)
	for i := 0; i < 10; i++ {
		prev[uuid.New().String()] = struct{}{}
	}
	params := []model.DeviceParameter{
		{ParameterPath: "Device.X.A"}, // 仅 1 个 — 远小于 prev 的一半（5）
	}
	svc.logPathBSyncDiff(context.Background(), dev, prev, params)

	entries := recorded.FilterMessage("param_sync_missing").All()
	require.Len(t, entries, 1)
	assert.Equal(t, zap.WarnLevel, entries[0].Level,
		"current < prev/2 时应升级到 Warn 级别")
}

func TestLogPathBSyncDiff_SamplingTruncatesTo20(t *testing.T) {
	svc, recorded, _ := newDiffTestService(t, nil, nil)
	dev := &model.Device{ID: uuid.New(), SerialNumber: "SN-SAMPLE"}

	prev := make(map[string]struct{}, 50)
	for i := 0; i < 50; i++ {
		prev[uuid.New().String()] = struct{}{}
	}
	params := []model.DeviceParameter{} // 全部缺失 — 50 个

	svc.logPathBSyncDiff(context.Background(), dev, prev, params)

	entries := recorded.FilterMessage("param_sync_missing").All()
	require.Len(t, entries, 1)

	// missing_paths_sample 在 ObservedLogs.ContextMap() 中类型为 []interface{}（zap.Strings 序列化）
	fields := entries[0].ContextMap()
	sample, ok := fields["missing_paths_sample"].([]interface{})
	if !ok {
		// 兼容路径：某些 zap 版本保留 []string
		if s, isStr := fields["missing_paths_sample"].([]string); isStr {
			assert.Equal(t, 20, len(s), "采样应截断到前 20 条")
		} else {
			t.Fatalf("missing_paths_sample 类型预期 []interface{} 或 []string，实际 %T", fields["missing_paths_sample"])
		}
	} else {
		assert.Equal(t, 20, len(sample), "采样应截断到前 20 条")
	}
	assert.Equal(t, int64(50), fields["missing_count"])
}

func TestLogPathBSyncDiff_ReasonUnknownWhenRedisKeyAbsent(t *testing.T) {
	svc, recorded, _ := newDiffTestService(t, nil, nil) // mr 上无 reason key
	dev := &model.Device{ID: uuid.New(), SerialNumber: "SN-UNKNOWN-REASON"}

	prev := map[string]struct{}{"Device.X.A": {}, "Device.X.B": {}}
	params := []model.DeviceParameter{{ParameterPath: "Device.X.A"}}
	svc.logPathBSyncDiff(context.Background(), dev, prev, params)

	entries := recorded.FilterMessage("param_sync_missing").All()
	require.Len(t, entries, 1)
	assert.Equal(t, "unknown", entries[0].ContextMap()["reason"],
		"Redis 无 reason key 时应降级为 unknown")
}

func TestSnapshotStandardPaths_ReturnsNilOnError(t *testing.T) {
	svc, _, _ := newDiffTestService(t, nil, errors.New("db down"))
	got := svc.snapshotStandardPaths(context.Background(), uuid.New())
	assert.Nil(t, got, "GetByDevice 失败应返 nil（差异日志静默跳过）")
}

func TestSnapshotStandardPaths_ReturnsEmptyOnNoRows(t *testing.T) {
	svc, _, _ := newDiffTestService(t, nil, nil) // paths empty
	got := svc.snapshotStandardPaths(context.Background(), uuid.New())
	assert.Nil(t, got, "无现有 standardPath 应返 nil（首次同步等价）")
}

func TestSnapshotStandardPaths_BuildsMap(t *testing.T) {
	paths := []string{"Device.X.A", "Device.X.B", "Device.X.C"}
	svc, _, _ := newDiffTestService(t, paths, nil)
	got := svc.snapshotStandardPaths(context.Background(), uuid.New())
	require.Len(t, got, 3)
	for _, p := range paths {
		_, ok := got[p]
		assert.Truef(t, ok, "path %s should be in snapshot map", p)
	}
}

// ---------------------------------------------------------------------------
// Tests: T-0124 last_param_sync_at 回写口径
// ---------------------------------------------------------------------------

type fakeParamSyncWriter struct {
	mu    sync.Mutex // guards calls：延迟 finalize goroutine 写 / 测试断言读
	calls []paramSyncWriteCall
	err   error
}

type paramSyncWriteCall struct {
	deviceID uuid.UUID
	at       time.Time
}

func (f *fakeParamSyncWriter) UpdateLastParamSyncAt(_ context.Context, id uuid.UUID, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, paramSyncWriteCall{deviceID: id, at: at})
	return f.err
}

// callCount 加锁读取 calls 数量，供 Eventually/Never 等并发断言使用。
func (f *fakeParamSyncWriter) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

type fakeDeviceInfoRefresher struct{}

func (f *fakeDeviceInfoRefresher) SyncFromParameters(_ context.Context, _ uuid.UUID, _ model.CarrierCode, _ model.Technology) ([]string, error) {
	return nil, nil
}

type fakePathBSyncTaskReader struct {
	hasOpen bool
	err     error
	calls   []string
}

func (f *fakePathBSyncTaskReader) HasIncompleteSyncGPVTasksByDevice(_ context.Context, deviceSN string) (bool, error) {
	f.calls = append(f.calls, deviceSN)
	return f.hasOpen, f.err
}

func TestSetParamSyncWriter_ChainableReturnsSyncService(t *testing.T) {
	svc := &SyncService{}
	got := svc.SetParamSyncWriter(&fakeParamSyncWriter{})
	assert.Same(t, svc, got, "SetParamSyncWriter 应返回 *SyncService 支持链式调用")
}

func TestSetParamSyncWriter_NilSafe(t *testing.T) {
	svc := &SyncService{}
	assert.NotPanics(t, func() { svc.SetParamSyncWriter(nil) }, "nil 注入应不 panic（dev/test 场景接受禁用回写）")
	assert.Nil(t, svc.paramSyncWriter, "nil 注入后字段应为 nil")
}

func TestSetDeviceInfoRefresher_ChainableReturnsSyncService(t *testing.T) {
	svc := &SyncService{}
	got := svc.SetDeviceInfoRefresher(&fakeDeviceInfoRefresher{})
	assert.Same(t, svc, got, "SetDeviceInfoRefresher 应返回 *SyncService 支持链式调用")
}

func TestSetDeviceInfoRefresher_NilSafe(t *testing.T) {
	svc := &SyncService{}
	assert.NotPanics(t, func() { svc.SetDeviceInfoRefresher(nil) }, "nil 注入应不 panic")
	assert.Nil(t, svc.deviceInfoRefresher, "nil 注入后字段应为 nil")
}

func TestSetPathBSyncTaskReader_ChainableReturnsSyncService(t *testing.T) {
	svc := &SyncService{}
	got := svc.SetPathBSyncTaskReader(&fakePathBSyncTaskReader{})
	assert.Same(t, svc, got)
}

func TestSetPathBSyncTaskReader_NilSafe(t *testing.T) {
	svc := &SyncService{}
	assert.NotPanics(t, func() { svc.SetPathBSyncTaskReader(nil) })
	assert.Nil(t, svc.pathBSyncTaskReader)
}

func TestShouldFinalizePathBSync_WaitsUntilLastBatch(t *testing.T) {
	svc, _, _ := newDiffTestService(t, nil, nil)
	ctx := context.Background()
	dev := &model.Device{ID: uuid.New(), SerialNumber: "SN-1"}

	svc.recordPathBSyncPendingBatches(ctx, dev.ID, 3)

	assert.False(t, svc.shouldFinalizePathBSync(ctx, dev, "sync-gpv-sn-0"))
	assert.False(t, svc.shouldFinalizePathBSync(ctx, dev, "sync-gpv-sn-1"))
	assert.True(t, svc.shouldFinalizePathBSync(ctx, dev, "sync-gpv-sn-2"))
	assert.True(t, svc.shouldFinalizePathBSync(ctx, dev, "sync-gpv-sn-3"), "缺少 pending key 时应降级为允许 finalize")
}

func TestShouldFinalizePathBSync_NonFullSyncBypassesPendingCounter(t *testing.T) {
	svc, _, _ := newDiffTestService(t, nil, nil)
	assert.True(t, svc.shouldFinalizePathBSync(context.Background(), &model.Device{ID: uuid.New(), SerialNumber: "SN-2"}, "auto-gpv-after-spv-1"))
}

func TestShouldFinalizePathBSync_UsesTaskReaderWhenAvailable(t *testing.T) {
	svc, _, _ := newDiffTestService(t, nil, nil)
	reader := &fakePathBSyncTaskReader{hasOpen: true}
	svc.SetPathBSyncTaskReader(reader)
	dev := &model.Device{ID: uuid.New(), SerialNumber: "SN-reader"}

	assert.False(t, svc.shouldFinalizePathBSync(context.Background(), dev, "sync-gpv-sn-0"))
	assert.Equal(t, []string{"SN-reader"}, reader.calls)

	reader.hasOpen = false
	assert.True(t, svc.shouldFinalizePathBSync(context.Background(), dev, "sync-gpv-sn-1"))
}

func TestHandleSyncResultPathB_EmptyFullSyncWaitsForRemainingTasks(t *testing.T) {
	svc, _, _ := newDiffTestService(t, nil, nil)
	writer := &fakeParamSyncWriter{}
	svc.SetParamSyncWriter(writer)
	reader := &fakePathBSyncTaskReader{hasOpen: true}
	svc.SetPathBSyncTaskReader(reader)
	dev := &model.Device{ID: uuid.New(), SerialNumber: "SN-empty", Carrier: model.CarrierCMCC, Technology: model.TechNR}

	handled, err := svc.HandleSyncResultPathB(context.Background(), dev, nil, "sync-gpv-sn-empty-0")
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, []string{"SN-empty"}, reader.calls)
	assert.Never(t, func() bool {
		return writer.callCount() > 0
	}, 200*time.Millisecond, 20*time.Millisecond, "empty full-sync response must not finalize while sync-gpv tasks remain open")
}
