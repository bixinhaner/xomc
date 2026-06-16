package dashboard

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/pm/indicator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// resolveKPIAliases —— issue #227 别名解析层核心（symbolic → K 编号）
// ---------------------------------------------------------------------------

// 成功路径：已登记的 symbolic key 翻成正确 K 编号，反查表能把 K 编号映射回原 symbolic。
func TestResolveKPIAliases_Hit(t *testing.T) {
	kcodes, reverse := resolveKPIAliases([]string{"LTE_PDCP_VOLUME_DL", "RRC_CONN_SETUP_SR"})

	assert.ElementsMatch(t, []string{"K900010015", "K900010002"}, kcodes)
	assert.Equal(t, []string{"LTE_PDCP_VOLUME_DL"}, reverse["K900010015"])
	assert.Equal(t, []string{"RRC_CONN_SETUP_SR"}, reverse["K900010002"])
}

// issue #389 阶段1：LTE_CELL_AVAILABLE 原为 none 项（硬缺口、返回空序列），现已接到
// 新建“小区可用率”指标编号 K900010076，应正常解析、不再返回空序列。
func TestResolveKPIAliases_CellAvailableResolved(t *testing.T) {
	kcodes, reverse := resolveKPIAliases([]string{"LTE_CELL_AVAILABLE"})

	assert.Equal(t, []string{"K900010076"}, kcodes, "LTE_CELL_AVAILABLE 应解析到 K900010076")
	assert.Equal(t, []string{"LTE_CELL_AVAILABLE"}, reverse["K900010076"])
}

// 未命中别名表的值（调用方直接传 K 编号 / 非 Dashboard 调用方）：原样透传作为 K 编号，
// 自映射回原 key，保持既有等值查询语义不被破坏。
func TestResolveKPIAliases_PassthroughUnknown(t *testing.T) {
	kcodes, reverse := resolveKPIAliases([]string{"K900010099"})

	assert.Equal(t, []string{"K900010099"}, kcodes)
	assert.Equal(t, []string{"K900010099"}, reverse["K900010099"])
}

// 混合输入：命中（含已补齐的小区可用率）+ 透传 同时出现，各自归位。
func TestResolveKPIAliases_Mixed(t *testing.T) {
	kcodes, reverse := resolveKPIAliases([]string{
		"GSM_CALL_SETUP_SR",  // 命中 → KGSM0102
		"LTE_CELL_AVAILABLE", // 命中 → K900010076（issue #389 已补齐）
		"SOME_RAW_KEY",       // 透传
	})

	assert.ElementsMatch(t, []string{"KGSM0102", "K900010076", "SOME_RAW_KEY"}, kcodes)
	assert.Equal(t, []string{"GSM_CALL_SETUP_SR"}, reverse["KGSM0102"])
	assert.Equal(t, []string{"LTE_CELL_AVAILABLE"}, reverse["K900010076"])
	assert.Equal(t, []string{"SOME_RAW_KEY"}, reverse["SOME_RAW_KEY"])
	_, hasEmpty := reverse[""]
	assert.False(t, hasEmpty, "不应在反查表留空键")
}

// 别名表自洽性：除 none 项外，所有 symbolic 都有非空 K 编号且 K 编号全局唯一
// （回填依赖 K 编号唯一，撞 K 编号会把数据错配到多个 symbolic）。
func TestDashboardKPIAliases_KCodeUniqueAndNonEmpty(t *testing.T) {
	seen := map[string]string{}
	for _, a := range dashboardKPIAliases {
		assert.NotEmpty(t, a.Symbolic, "symbolic 不可为空")
		assert.NotEmpty(t, a.Tech, "tech 不可为空: %s", a.Symbolic)
		assert.NotEmpty(t, a.Panel, "panel 不可为空: %s", a.Symbolic)
		if a.KCode == "" {
			// none 项必须标 NeedsReview（提醒领域复核硬缺口）。
			assert.True(t, a.NeedsReview, "none 项应标 NeedsReview: %s", a.Symbolic)
			continue
		}
		if prev, dup := seen[a.KCode]; dup {
			t.Fatalf("K 编号 %s 被 %s 与 %s 重复引用，回填会错配", a.KCode, prev, a.Symbolic)
		}
		seen[a.KCode] = a.Symbolic
	}
}

// ---------------------------------------------------------------------------
// GetKPIDefinitions —— issue #213 Phase1 动态定义端点
// ---------------------------------------------------------------------------

// dashFakeIndicatorRepo 是给 GetKPIDefinitions 富化路径用的最小 mock：
// 只让 ListByIDs 按 byID 返回登记的 PerfIndicator，其余方法为接口满足用的空桩。
type dashFakeIndicatorRepo struct {
	byID          map[string]*indicator.PerfIndicator
	listByIDsErr  error
	listByIDsCall int
}

func (f *dashFakeIndicatorRepo) ListByIDs(ctx context.Context, dt indicator.DeviceType, ids []string) ([]*indicator.PerfIndicator, error) {
	f.listByIDsCall++
	if f.listByIDsErr != nil {
		return nil, f.listByIDsErr
	}
	out := make([]*indicator.PerfIndicator, 0, len(ids))
	for _, id := range ids {
		if ind, ok := f.byID[id]; ok {
			out = append(out, ind)
		}
	}
	return out, nil
}

// ── unused stubs (interface satisfaction only) ──────────────────────────────
func (f *dashFakeIndicatorRepo) List(ctx context.Context, filter indicator.IndicatorListFilter) (*model.ListResponse[indicator.IndicatorListItem], error) {
	return nil, nil
}
func (f *dashFakeIndicatorRepo) GetByID(ctx context.Context, dt indicator.DeviceType, id string) (*indicator.PerfIndicator, error) {
	return nil, nil
}
func (f *dashFakeIndicatorRepo) Create(ctx context.Context, dt indicator.DeviceType, ind *indicator.PerfIndicator, tx pgx.Tx) error {
	return nil
}
func (f *dashFakeIndicatorRepo) Update(ctx context.Context, dt indicator.DeviceType, id string, req *indicator.UpdateIndicatorRequest, tx pgx.Tx) error {
	return nil
}
func (f *dashFakeIndicatorRepo) Delete(ctx context.Context, dt indicator.DeviceType, id string, tx pgx.Tx) error {
	return nil
}
func (f *dashFakeIndicatorRepo) DeleteByGroupID(ctx context.Context, dt indicator.DeviceType, groupID string, tx pgx.Tx) error {
	return nil
}
func (f *dashFakeIndicatorRepo) GetIDsByGroupID(ctx context.Context, dt indicator.DeviceType, groupID string) ([]string, error) {
	return nil, nil
}
func (f *dashFakeIndicatorRepo) GetNextKPIID(ctx context.Context, dt indicator.DeviceType, operatorCode string) (string, error) {
	return "", nil
}
func (f *dashFakeIndicatorRepo) GetNextCounterID(ctx context.Context) (string, error) {
	return "", nil
}
func (f *dashFakeIndicatorRepo) ListAll(ctx context.Context, filter indicator.IndicatorListFilter) ([]indicator.IndicatorListItem, error) {
	return nil, nil
}

func strptr(s string) *string { return &s }

// 富化路径：indicatorRepo 返回 cnName / unit，被叠加到对应 symbolic 定义上。
func TestGetKPIDefinitions_Enriched(t *testing.T) {
	repo := &dashFakeIndicatorRepo{
		byID: map[string]*indicator.PerfIndicator{
			"K900010015": {ID: "K900010015", CnName: strptr("下行数据业务流量"), UnitID: strptr("MByte")},
			"K900010076": {ID: "K900010076", CnName: strptr("小区可用率"), UnitID: strptr("%")},
			"KGSM0102":   {ID: "KGSM0102", CnName: strptr("电话成功率"), UnitID: strptr("%")},
		},
	}
	svc := &Service{indicatorRepo: repo, logger: zap.NewNop()}

	resp, err := svc.GetKPIDefinitions(context.Background())
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, len(dashboardKPIAliases), resp.Total)
	// 至少 lte / nr / gsm 三组都在。
	techs := map[string]bool{}
	var dlVolume, cellAvail *KPIDefinitionItem
	for i := range resp.Technologies {
		td := &resp.Technologies[i]
		techs[td.Tech] = true
		for j := range td.Items {
			it := &td.Items[j]
			switch it.Key {
			case "LTE_PDCP_VOLUME_DL":
				dlVolume = it
			case "LTE_CELL_AVAILABLE":
				cellAvail = it
			}
		}
	}
	assert.True(t, techs["lte"] && techs["nr"] && techs["gsm"], "三制式都应出现")

	require.NotNil(t, dlVolume)
	assert.Equal(t, "K900010015", dlVolume.KCode)
	assert.Equal(t, "下行数据业务流量", dlVolume.CnName)
	assert.Equal(t, "MByte", dlVolume.Unit)
	assert.True(t, dlVolume.Available)

	// issue #389 阶段1：小区可用率已补齐 → KCode=K900010076 / Available=true / cnName 富化。
	require.NotNil(t, cellAvail)
	assert.Equal(t, "K900010076", cellAvail.KCode)
	assert.True(t, cellAvail.Available)
	assert.Equal(t, "小区可用率", cellAvail.CnName)
}

// 退化路径：indicatorRepo 为 nil → 仍返回完整别名表结构（仅缺 cnName / unit 富化），端点不报错。
func TestGetKPIDefinitions_NilRepoDegraded(t *testing.T) {
	svc := &Service{indicatorRepo: nil, logger: zap.NewNop()}

	resp, err := svc.GetKPIDefinitions(context.Background())
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, len(dashboardKPIAliases), resp.Total)
	for _, td := range resp.Technologies {
		for _, it := range td.Items {
			assert.Empty(t, it.CnName, "nil repo 不应有 cnName 富化: %s", it.Key)
			assert.NotEmpty(t, it.Key)
			assert.NotEmpty(t, it.Panel)
		}
	}
}

// 富化失败容错：ListByIDs 报错时记日志、对应制式退化为静态字段，端点仍成功返回。
func TestGetKPIDefinitions_EnrichErrorTolerated(t *testing.T) {
	repo := &dashFakeIndicatorRepo{listByIDsErr: errors.New("db down")}
	svc := &Service{indicatorRepo: repo, logger: zap.NewNop()}

	resp, err := svc.GetKPIDefinitions(context.Background())
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, len(dashboardKPIAliases), resp.Total)
}
