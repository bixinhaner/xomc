package provision

import (
	"context"
	"errors"
	"sort"
	"testing"

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
