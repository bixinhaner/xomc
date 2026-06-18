package provision

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/model"
)

// ── countStorableFieldsUnderPrefix ─────────────────────────────────────────

func TestCountStorableFieldsUnderPrefix_MatchesUsingILikeITemplate(t *testing.T) {
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "DeviceGSM.Bts.{i}.CellId", IsStorable: true, IsSupported: true},
		{PrivatePath: "DeviceGSM.Bts.{i}.Band", IsStorable: true, IsSupported: true},
		{PrivatePath: "DeviceGSM.Bts.{i}.RachCellBarred", IsStorable: true, IsSupported: true},
		// 不可存储 / 不支持 → 不计入
		{PrivatePath: "DeviceGSM.Bts.{i}.RuntimeStatus", IsStorable: false, IsSupported: true},
		{PrivatePath: "DeviceGSM.Bts.{i}.UnsupportedX", IsStorable: true, IsSupported: false},
		// 其它对象 → 不计入
		{PrivatePath: "DeviceGSM.Bsc.SystemInfo", IsStorable: true, IsSupported: true},
		{PrivatePath: "Device.System.Mode", IsStorable: true, IsSupported: true},
	}
	got := countStorableFieldsUnderPrefix(mappings, "DeviceGSM.Bts.")
	assert.Equal(t, 3, got)
}

func TestCountStorableFieldsUnderPrefix_NormalizesFAPServiceSingleton(t *testing.T) {
	// extractStorablePrefixes 把 FAPService.{i}. 替换为 FAPService.1.,所以传入 prefix
	// 已经是 FAPService.1.<X>.  形态。countStorableFieldsUnderPrefix 必须对 mapping 也
	// 做 normalize 才能匹配。
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PCI",
			IsStorable: true, IsSupported: true},
		{PrivatePath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.EARFCN",
			IsStorable: true, IsSupported: true},
	}
	prefix := "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell."
	got := countStorableFieldsUnderPrefix(mappings, prefix)
	assert.Equal(t, 2, got)
}

func TestCountStorableFieldsUnderPrefix_NoMatch(t *testing.T) {
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "Device.System.Mode", IsStorable: true, IsSupported: true},
	}
	got := countStorableFieldsUnderPrefix(mappings, "DeviceGSM.Bts.")
	assert.Equal(t, 0, got)
}

// ── splitObjectAndInstancePrefix ───────────────────────────────────────────

func TestSplitObjectAndInstancePrefix(t *testing.T) {
	cases := []struct {
		in       string
		wantObj  string
		wantInst string
	}{
		// 深度 >= 4 段的真实多实例对象 → 正常返回
		{"Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.2.PCI",
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.",
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.2."},
		// 嵌套对象:取最深数字段(Trx 后的 "1",不是 Bts.5)
		{"DeviceGSM.Bts.5.Trx.1.Rf",
			"DeviceGSM.Bts.5.Trx.",
			"DeviceGSM.Bts.5.Trx.1."},
		// 顶层多实例对象:DeviceGSM.Bts.5.CellId (parts=[DeviceGSM,Bts,5,CellId],i=2 < 4)
		// 数字段位置太浅 → 返回 ("","") 保守不 reconcile
		{"DeviceGSM.Bts.5.CellId", "", ""},
		// 第二段就出现数字 → 弃 (防 BLQ 真机 DeviceInfo.2.UE_Count 误删)
		{"Device.DeviceInfo.2.UE_Count", "", ""},
		// 无数字段
		{"Device.System.Mode", "", ""},
		// 空 path
		{"", "", ""},
	}
	for _, c := range cases {
		gotObj, gotInst := splitObjectAndInstancePrefix(c.in)
		assert.Equal(t, c.wantObj, gotObj, "objPrefix for %q", c.in)
		assert.Equal(t, c.wantInst, gotInst, "instPrefix for %q", c.in)
	}
}

// ── deriveReconcilePrefixes ────────────────────────────────────────────────

func TestDeriveReconcilePrefixes_SingleInstanceBatchUsesInstancePrefix(t *testing.T) {
	// 模拟 expand 后的 batch: 全部属于 LTECell.2 一个实例
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.2.PCI"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.2.EARFCN"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.2.CID"},
	}
	got := deriveReconcilePrefixes(params)
	assert.Equal(t, []string{
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.2.",
	}, got, "单实例 batch 应输出 instance-level prefix")
}

func TestDeriveReconcilePrefixes_MultiInstanceBatchUsesObjectPrefix(t *testing.T) {
	// 模拟未展开 batch: CPE 一次返多个实例的字段
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.2.PCI"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.3.PCI"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.5.PCI"},
	}
	got := deriveReconcilePrefixes(params)
	assert.Equal(t, []string{
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.",
	}, got, "多实例 batch 应输出 object-level prefix")
}

func TestDeriveReconcilePrefixes_MixedObjects(t *testing.T) {
	// 单 batch 内既有单实例对象(LTECell.2),又有多实例对象(WCDMA.4/5/6)
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.2.PCI"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.WCDMA.4.PSC"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.WCDMA.5.PSC"},
	}
	got := deriveReconcilePrefixes(params)
	sort.Strings(got)
	want := []string{
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.2.", // 单实例 → instance prefix
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.WCDMA.",      // 多实例 → object prefix
	}
	sort.Strings(want)
	assert.Equal(t, want, got)
}

func TestDeriveReconcilePrefixes_DropsShallowAndLeaf(t *testing.T) {
	// 数字段位置浅 / 无数字段 / 空 都被丢弃
	params := []model.DeviceParameter{
		{ParameterPath: "DeviceGSM.Bts.5.CellId"},      // i=2 < 4 → drop
		{ParameterPath: "Device.System.Mode"},          // 无数字段 → drop
		{ParameterPath: "Device.DeviceInfo.2.UE_Count"}, // i=2 < 4 → drop
		{ParameterPath: ""},                            // 空 → drop
	}
	got := deriveReconcilePrefixes(params)
	assert.Empty(t, got, "全部应被丢弃,不参与 reconcile")
}

// ── expandLargeObjectPrefixes ──────────────────────────────────────────────

// fakeRepoWithHint 实现 instanceHintRepo 接口,供 expand 测试控制 hint 值。
// 同时实现 DeviceParameterRepository 主接口的最小子集(其它方法不会被 expand 触发)。
type fakeRepoWithHint struct {
	maxByPrefix map[string]int
	err         error
}

func (f *fakeRepoWithHint) MaxInstanceNumberByPrefix(_ context.Context, _ uuid.UUID, prefix string) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	if v, ok := f.maxByPrefix[prefix]; ok {
		return v, nil
	}
	return 0, nil
}

// 占位实现:满足主接口,不会被 expand 路径触发
func (f *fakeRepoWithHint) BatchUpsert(_ context.Context, _ uuid.UUID, _ []model.DeviceParameter) error {
	return nil
}
func (f *fakeRepoWithHint) GetByDevice(_ context.Context, _ uuid.UUID) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (f *fakeRepoWithHint) GetByPath(_ context.Context, _ uuid.UUID, _ string) (*model.DeviceParameter, error) {
	return nil, nil
}
func (f *fakeRepoWithHint) DeleteByDevice(_ context.Context, _ uuid.UUID) error { return nil }
func (f *fakeRepoWithHint) DeleteByPathPrefix(_ context.Context, _ uuid.UUID, _ string) (int64, error) {
	return 0, nil
}
func (f *fakeRepoWithHint) GetByPathPrefix(_ context.Context, _ uuid.UUID, _ string) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (f *fakeRepoWithHint) CountByPathPrefix(_ context.Context, _ uuid.UUID, _ string) (int, error) {
	return 0, nil
}
func (f *fakeRepoWithHint) SearchByKeyword(_ context.Context, _ uuid.UUID, _ string, _ int) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (f *fakeRepoWithHint) GetDirectChildLeaves(_ context.Context, _ uuid.UUID, _ string, _ int, _ int) ([]model.DeviceParameter, int, error) {
	return nil, 0, nil
}
func (f *fakeRepoWithHint) GetByGroup(_ context.Context, _ uuid.UUID, _ string) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (f *fakeRepoWithHint) GetByFAPInstance(_ context.Context, _ uuid.UUID, _ int) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (f *fakeRepoWithHint) GetByFAPInstanceAndGroup(_ context.Context, _ uuid.UUID, _ int, _ string) ([]model.DeviceParameter, error) {
	return nil, nil
}

func TestExpandLargeObjectPrefixes_SmallObjectsKeptAsIs(t *testing.T) {
	// 小对象: fields=5, hintFloor=32 → est = 60 * 5 * 32 = 9600 << 600KB → 不展开
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "Dev.WiFi.SSID.{i}.Enabled", IsStorable: true, IsSupported: true},
		{PrivatePath: "Dev.WiFi.SSID.{i}.Name", IsStorable: true, IsSupported: true},
		{PrivatePath: "Dev.WiFi.SSID.{i}.Channel", IsStorable: true, IsSupported: true},
		{PrivatePath: "Dev.WiFi.SSID.{i}.SecMode", IsStorable: true, IsSupported: true},
		{PrivatePath: "Dev.WiFi.SSID.{i}.MaxBitRate", IsStorable: true, IsSupported: true},
	}
	svc := &SyncService{
		paramRepo: &fakeRepoWithHint{maxByPrefix: map[string]int{}}, // 无 DB 历史 → hintFloor=32
		logger:    zap.NewNop(),
	}
	got := svc.expandLargeObjectPrefixes(context.Background(), uuid.New(), mappings,
		[]string{"Dev.WiFi.SSID."})
	assert.Equal(t, []string{"Dev.WiFi.SSID."}, got, "小对象应原样保留")
}

func TestExpandLargeObjectPrefixes_LargeObjectExpandedToInstances(t *testing.T) {
	// 大对象: 70 字段 × hintFloor=32 instances × 60B = 134400 < 600KB? 算不上大
	// 试 70 × 254 (BTS 真实历史) × 60 = 1066800 > 600KB → 展开
	mappings := make([]parammodel.ParamMapping, 0, 70)
	for i := 0; i < 70; i++ {
		mappings = append(mappings, parammodel.ParamMapping{
			PrivatePath: "DeviceGSM.Bts.{i}.Field" + string(rune('A'+i%26)),
			IsStorable:  true,
			IsSupported: true,
		})
	}
	svc := &SyncService{
		paramRepo: &fakeRepoWithHint{maxByPrefix: map[string]int{
			"DeviceGSM.Bts.": 254, // 历史 254 BTS
		}},
		logger: zap.NewNop(),
	}
	got := svc.expandLargeObjectPrefixes(context.Background(), uuid.New(), mappings,
		[]string{"DeviceGSM.Bts."})
	// hint = 254 + 254/4 = 317; expand 出 317 个 instance prefix
	require.Len(t, got, 317)
	assert.Equal(t, "DeviceGSM.Bts.1.", got[0])
	assert.Equal(t, "DeviceGSM.Bts.317.", got[316])
}

func TestExpandLargeObjectPrefixes_FirstTimeSyncUsesHintFloor(t *testing.T) {
	// 首次同步 DB 无历史 → maxByPrefix 返回 0 → hint=hintFloor=32
	// 70 字段 × 32 × 60 = 134400 < 600KB → 此 case 不展开
	// 我们用更大的 200 字段触发: 200 × 32 × 60 = 384000 < 600KB 仍不展开
	// 必须 fields × hintFloor × 60 > 600 * 1024 → fields > 600*1024/(32*60) ≈ 320
	// 用 400 字段
	mappings := make([]parammodel.ParamMapping, 0, 400)
	for i := 0; i < 400; i++ {
		mappings = append(mappings, parammodel.ParamMapping{
			PrivatePath: "DeviceGiant.{i}.Field_" + string(rune('A'+i%26)) + "_" + string(rune('0'+i%10)),
			IsStorable:  true,
			IsSupported: true,
		})
	}
	svc := &SyncService{
		paramRepo: &fakeRepoWithHint{maxByPrefix: map[string]int{}}, // 无历史
		logger:    zap.NewNop(),
	}
	got := svc.expandLargeObjectPrefixes(context.Background(), uuid.New(), mappings,
		[]string{"DeviceGiant."})
	// hint = hintFloor = 32; 期望展开 32 个
	require.Len(t, got, 32)
	assert.Equal(t, "DeviceGiant.1.", got[0])
	assert.Equal(t, "DeviceGiant.32.", got[31])
}

func TestExpandLargeObjectPrefixes_DBLookupErrorFallsBackToHintFloor(t *testing.T) {
	mappings := make([]parammodel.ParamMapping, 0, 400)
	for i := 0; i < 400; i++ {
		mappings = append(mappings, parammodel.ParamMapping{
			PrivatePath: "DeviceGiant.{i}.F" + string(rune('a'+i%26)) + string(rune('0'+i%10)),
			IsStorable:  true,
			IsSupported: true,
		})
	}
	svc := &SyncService{
		paramRepo: &fakeRepoWithHint{err: errors.New("db down")},
		logger:    zap.NewNop(),
	}
	got := svc.expandLargeObjectPrefixes(context.Background(), uuid.New(), mappings,
		[]string{"DeviceGiant."})
	require.Len(t, got, 32, "DB 错误应回退 hintFloor=32")
}

func TestExpandLargeObjectPrefixes_NonObjectPrefixUnchanged(t *testing.T) {
	// 标量参数(无尾点)不能展开,原样返回
	svc := &SyncService{
		paramRepo: &fakeRepoWithHint{},
		logger:    zap.NewNop(),
	}
	got := svc.expandLargeObjectPrefixes(context.Background(), uuid.New(),
		[]parammodel.ParamMapping{}, []string{"Device.System.Mode"})
	assert.Equal(t, []string{"Device.System.Mode"}, got)
}

func TestExpandLargeObjectPrefixes_MaxHintCap(t *testing.T) {
	// 历史里出现异常 1000 个实例,放大 25% = 1250,但 maxHintCap=512 → 截断
	mappings := make([]parammodel.ParamMapping, 0, 100)
	for i := 0; i < 100; i++ {
		mappings = append(mappings, parammodel.ParamMapping{
			PrivatePath: "Huge.{i}.F" + string(rune('a'+i%26)),
			IsStorable:  true,
			IsSupported: true,
		})
	}
	svc := &SyncService{
		paramRepo: &fakeRepoWithHint{maxByPrefix: map[string]int{"Huge.": 1000}},
		logger:    zap.NewNop(),
	}
	got := svc.expandLargeObjectPrefixes(context.Background(), uuid.New(), mappings,
		[]string{"Huge."})
	require.Len(t, got, maxHintCap, "超过 maxHintCap 应截断")
}

// ── 集成:expand 决策被记录到日志 ────────────────────────────────────────

func TestExpandLargeObjectPrefixes_LogsExpansion(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	mappings := make([]parammodel.ParamMapping, 0, 70)
	for i := 0; i < 70; i++ {
		mappings = append(mappings, parammodel.ParamMapping{
			PrivatePath: "DeviceGSM.Bts.{i}.F" + string(rune('a'+i%26)),
			IsStorable:  true,
			IsSupported: true,
		})
	}
	svc := &SyncService{
		paramRepo: &fakeRepoWithHint{maxByPrefix: map[string]int{"DeviceGSM.Bts.": 254}},
		logger:    zap.New(core),
	}
	_ = svc.expandLargeObjectPrefixes(context.Background(), uuid.New(), mappings,
		[]string{"DeviceGSM.Bts."})
	logs := recorded.FilterMessage("path-b expand: object prefix expanded to instance level").All()
	require.Len(t, logs, 1)
	fields := logs[0].ContextMap()
	assert.Equal(t, "DeviceGSM.Bts.", fields["object_prefix"])
	assert.EqualValues(t, 70, fields["fields_per_instance"])
	assert.EqualValues(t, 317, fields["max_instance_hint"])
	assert.EqualValues(t, 317, fields["expanded_count"])
}
