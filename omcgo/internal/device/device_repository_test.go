package device

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// fakeGeoRow 模拟 pgx 行扫描语义供 scanGeoDeviceRow 单测使用：
// vals 中 nil 表示 SQL NULL；与 pgx 一致，NULL 只能扫进"指针的指针"目标
// （**T → 置 nil），扫进非空目标（*float64 / *string 等）直接报错。
// 这保证若有人把坐标列扫描目标改回 &d.Latitude（*float64），测试会像
// 线上 pgx 一样失败（#117 回归保护）。
type fakeGeoRow struct {
	vals []any
}

func (f fakeGeoRow) Scan(dest ...interface{}) error {
	if len(dest) != len(f.vals) {
		return fmt.Errorf("number of field descriptions must equal number of destinations, got %d and %d", len(f.vals), len(dest))
	}
	for i, v := range f.vals {
		if v == nil {
			switch d := dest[i].(type) {
			case **float64:
				*d = nil
			case **string:
				*d = nil
			case **uuid.UUID:
				*d = nil
			case **int:
				*d = nil
			default:
				return fmt.Errorf("can't scan into dest[%d]: cannot scan NULL into %T", i, dest[i])
			}
			continue
		}
		switch d := dest[i].(type) {
		case *uuid.UUID:
			*d = v.(uuid.UUID)
		case **uuid.UUID:
			u := v.(uuid.UUID)
			*d = &u
		case *string:
			*d = v.(string)
		case **string:
			s := v.(string)
			*d = &s
		case *float64:
			*d = v.(float64)
		case **float64:
			fv := v.(float64)
			*d = &fv
		case *bool:
			*d = v.(bool)
		case *int:
			*d = v.(int)
		case **int:
			i := v.(int)
			*d = &i
		case *model.DeviceLifecycle:
			*d = v.(model.DeviceLifecycle)
		default:
			return fmt.Errorf("fakeGeoRow: unsupported dest[%d] type %T", i, dest[i])
		}
	}
	return nil
}

// geoRowVals 按 ListGeo / SearchDevices 的 SELECT 列顺序构造一行（19 列）。
func geoRowVals(id uuid.UUID, lat, lng any) []any {
	return []any{
		id, "SN-001", "SN-001", // id, serial_number, name
		model.LifecycleCommissioned, true, // lifecycle_state, is_online
		lat, lng, // latitude, longitude（可为 nil = SQL NULL）
		nil, nil, // group_id, group_name
		"site-a", 0, "FAP-100", // address, alarm_count, type
		"10.0.0.1", "AA:BB:CC", "120", "dev-a", // ip, mac, pci, device_name
		0, nil, 0, // ue_count, highest_alarm_severity, highest_severity_alarm_count
	}
}

// TestScanGeoDeviceRow_NullableCoordinates 是 #117 的回归测试：
// GET /devices/search 命中 NULL 坐标设备时，repository 扫描曾把可空的
// latitude/longitude 列扫进非空 float64 目标，报
// "scan search result: can't scan into dest[5] (col: latitude)" 导致整个搜索 500。
func TestScanGeoDeviceRow_NullableCoordinates(t *testing.T) {
	id := uuid.New()

	tests := []struct {
		name    string
		lat     any // float64 或 nil（SQL NULL）
		lng     any
		wantLat float64
		wantLng float64
	}{
		{
			name:    "NULL 坐标设备不再扫描失败（#117 主回归）",
			lat:     nil,
			lng:     nil,
			wantLat: 0,
			wantLng: 0,
		},
		{
			name:    "有坐标设备坐标正常透传",
			lat:     30.5928,
			lng:     114.3055,
			wantLat: 30.5928,
			wantLng: 114.3055,
		},
		{
			name:    "单侧 NULL（仅 longitude 缺失）也容忍",
			lat:     30.5928,
			lng:     nil,
			wantLat: 30.5928,
			wantLng: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := scanGeoDeviceRow(fakeGeoRow{vals: geoRowVals(id, tt.lat, tt.lng)})
			require.NoError(t, err)

			// 响应字段兼容：Latitude/Longitude 仍是 float64，NULL → 零值
			assert.Equal(t, tt.wantLat, d.Latitude)
			assert.Equal(t, tt.wantLng, d.Longitude)

			// 其余字段不受坐标可空性影响
			assert.Equal(t, id, d.ID)
			assert.Equal(t, "SN-001", d.SerialNumber)
			assert.Equal(t, model.DeviceActive, d.Status, "commissioned+online 派生 active")
			require.NotNil(t, d.IPAddress)
			assert.Equal(t, "10.0.0.1", *d.IPAddress)
		})
	}
}

// TestScanGeoDeviceRow_NullableJoinColumns 验证 LEFT JOIN 可空列
// （group_id/group_name/address/type）与 COALESCE 空串字段的处理不回退。
func TestScanGeoDeviceRow_NullableJoinColumns(t *testing.T) {
	id := uuid.New()
	vals := []any{
		id, "SN-002", "SN-002",
		model.LifecycleCommissioned, false,
		nil, nil, // 坐标 NULL
		nil, nil, // group_id, group_name NULL
		nil, 0, nil, // address, alarm_count, type（site_name/model_name 可空）
		"", "", "", "", // COALESCE 空串 → 响应里省略（nil 指针）
		0, nil, 0, // ue_count, highest_alarm_severity, highest_severity_alarm_count
	}

	d, err := scanGeoDeviceRow(fakeGeoRow{vals: vals})
	require.NoError(t, err)

	assert.Nil(t, d.GroupID)
	assert.Empty(t, d.GroupName)
	assert.Empty(t, d.Address)
	assert.Empty(t, d.Type)
	assert.Equal(t, model.DeviceOffline, d.Status, "commissioned+offline 派生 offline")
	assert.Nil(t, d.IPAddress)
	assert.Nil(t, d.MAC)
	assert.Nil(t, d.PCI)
	assert.Nil(t, d.DeviceName)
}

// TestGetCoordinatesQueryUsesDeviceAlias guards the shared notDeleted predicate,
// which references d.deleted_at. The FROM clause must therefore declare alias d.
func TestGetCoordinatesQueryUsesDeviceAlias(t *testing.T) {
	query, _, err := buildGetCoordinatesQuery(uuid.New())
	require.NoError(t, err)
	assert.Contains(t, query, "FROM devices d")
	assert.Contains(t, query, "d.deleted_at")
}

func TestApplyGeoBoundsFilter(t *testing.T) {
	bounds := &GeoBounds{
		MinLng: 73.5,
		MaxLng: 135.1,
		MinLat: 3.8,
		MaxLat: 53.6,
	}

	query, args, err := applyGeoBoundsFilter(
		storage.Psql.Select("d.id").From("devices d"),
		bounds,
	).ToSql()
	require.NoError(t, err)

	assert.Contains(t, query, "d.longitude >= $1")
	assert.Contains(t, query, "d.longitude <= $2")
	assert.Contains(t, query, "d.latitude >= $3")
	assert.Contains(t, query, "d.latitude <= $4")
	assert.Equal(t, []interface{}{73.5, 135.1, 3.8, 53.6}, args)
}

func TestApplyGeoBoundsFilter_NilKeepsQueryUnbounded(t *testing.T) {
	query, args, err := applyGeoBoundsFilter(
		storage.Psql.Select("d.id").From("devices d"),
		nil,
	).ToSql()
	require.NoError(t, err)

	assert.NotContains(t, query, "longitude")
	assert.NotContains(t, query, "latitude")
	assert.Empty(t, args)
}

// ---------------------------------------------------------------------------
// issue #203：FindStaleDevicesByClass 的 SQL 构建断言（DB-free）
//
// FindStaleDevicesByClass 在同一方法里内联构建 SQL 后立即 r.pool.Query，pool
// 是具体 *pgxpool.Pool（非接口），无法注入 fake 捕获 SQL；故此处按生产方法
// 完全相同的方式重建同一个 squirrel builder 再 ToSql 断言。CASE / CPE 谓词
// 直接引用生产常量 cpeProductClassPredicate，CPE-分支顺序与谓词内容锚定生产；
// make_interval / 占位符绑定顺序逐字镜像 device_repository.go，二者改动须同步。
// ---------------------------------------------------------------------------

// buildFindStaleByClassSQLForTest 镜像 FindStaleDevicesByClass 的 builder 构建段
// （device_repository.go），用于 DB-free 的 ToSql 断言。返回渲染后的 SQL 与 args。
//
// 注意：这是生产 builder 段的逐字镜像。若 FindStaleDevicesByClass 改了 SELECT 列、
// WHERE 谓词、staleExpr 表达式或占位符绑定顺序，必须同步本函数，否则本测试失去回归意义。
func buildFindStaleByClassSQLForTest(t *testing.T, cpeThresholdSec, enbThresholdSec, limit int) (string, []interface{}) {
	t.Helper()
	staleExpr := fmt.Sprintf(
		"d.last_inform_at < NOW() - make_interval(secs => (CASE WHEN %s THEN (?)::int ELSE (?)::int END))",
		cpeProductClassPredicate,
	)
	builder := storage.Psql.Select(deviceColumns()...).
		From("devices d").
		Where(sq.Eq{"d.is_online": true}).
		Where(notDeleted).
		Where("d.last_inform_at IS NOT NULL").
		Where(staleExpr, cpeThresholdSec, enbThresholdSec).
		OrderBy("d.last_inform_at ASC").
		Limit(uint64(limit))

	query, args, err := builder.ToSql()
	require.NoError(t, err)
	return query, args
}

// TestFindStaleDevicesByClass_SQLBuild_MakeIntervalAndCaseOrdering 断言离线判定
// SQL 用 make_interval(secs => (?)::int) 造 interval（绕开 text*interval，issue #203
// 回合2），CASE WHEN 把 CPE 谓词分支排在 ELSE（基站 eNB）之前，且占位参数按
// cpeThresholdSec → enbThresholdSec → limit 的顺序绑定。
func TestFindStaleDevicesByClass_SQLBuild_MakeIntervalAndCaseOrdering(t *testing.T) {
	const (
		cpe   = 600
		enb   = 100
		limit = 1000
	)
	query, args := buildFindStaleByClassSQLForTest(t, cpe, enb, limit)

	// 1) make_interval(secs => (?)::int) 表达式必须出现（修复 text*interval 42883）。
	assert.Contains(t, query, "make_interval(secs =>",
		"必须用 make_interval 造 interval，绕开 text*interval（issue #203 回合2）")
	assert.Contains(t, query, "::int",
		"秒数必须显式 cast 成 int 再喂 make_interval")

	// 2) CASE WHEN <CPE 谓词> THEN ... ELSE ...：CPE 分支必须排在 ELSE（eNB）之前。
	assert.Contains(t, query, "CASE WHEN", "阈值选取必须用 CASE WHEN 分类")
	caseIdx := strings.Index(query, "CASE WHEN")
	cpeIdx := strings.Index(query, "ILIKE '%cpe%'")
	elseIdx := strings.Index(query, "ELSE")
	require.GreaterOrEqual(t, caseIdx, 0, "SQL 应含 CASE WHEN")
	require.GreaterOrEqual(t, cpeIdx, 0, "SQL 应含 CPE 谓词 ILIKE '%cpe%'")
	require.GreaterOrEqual(t, elseIdx, 0, "SQL 应含 ELSE 兜底（基站 eNB）分支")
	assert.Less(t, caseIdx, cpeIdx, "CPE 谓词必须在 CASE WHEN 之后")
	assert.Less(t, cpeIdx, elseIdx, "CPE 分支（THEN）必须排在 ELSE（eNB）之前")

	// CPE 谓词的四个关键字都在 THEN 之前（即 CPE 分支内）。
	for _, kw := range []string{"%cpe%", "%home%", "%residential%", "%indoor%"} {
		kwIdx := strings.Index(query, "ILIKE '"+kw+"'")
		require.GreaterOrEqual(t, kwIdx, 0, "CPE 谓词应含关键字 %s", kw)
		assert.Less(t, kwIdx, elseIdx, "关键字 %s 必须落在 ELSE 之前的 CPE 分支", kw)
	}

	// 3) 占位参数绑定顺序：WHERE 子句按出现顺序收集 args——
	//    is_online=true（$1）→ staleExpr 的 CPE 分支(THEN, $2)=cpe → ELSE($3)=enb。
	//    notDeleted / last_inform_at IS NOT NULL 不带占位参数；
	//    Limit 由 squirrel 渲染为字面量 `LIMIT 1000`，不占位（故 args 只有 3 个）。
	require.Len(t, args, 3, "占位参数：is_online、cpe 阈值、enb 阈值（limit 是字面量不占位）")
	assert.EqualValues(t, true, args[0], "args[0] 是 is_online=true 谓词")
	assert.EqualValues(t, cpe, args[1], "args[1] 必须绑 cpeThresholdSec（CPE 分支 THEN，$2 在 ELSE 之前）")
	assert.EqualValues(t, enb, args[2], "args[2] 必须绑 enbThresholdSec（ELSE 基站分支，$3）")

	// 4) 基本结构：在线 + 未软删 + last_inform_at 非空 + 按 last_inform_at 升序 + LIMIT 字面量。
	assert.Contains(t, query, "d.is_online")
	assert.Contains(t, query, "d.deleted_at")
	assert.Contains(t, query, "d.last_inform_at IS NOT NULL")
	assert.Contains(t, query, "ORDER BY d.last_inform_at ASC")
	assert.Contains(t, query, fmt.Sprintf("LIMIT %d", limit), "limit 渲染为字面量 LIMIT 子句")
}

// TestFindStaleDevicesByClass_NullProductClass_FallsIntoENBBranch 是 NULL
// product_class 边界的 DB-free 断言：CASE 谓词全部走 `product_class ILIKE '%...%'`，
// 当 product_class 为 SQL NULL 时，`NULL ILIKE '%cpe%'` 求值为 NULL（非 TRUE），
// CASE WHEN 的所有分支条件均不满足 → 落入 ELSE（基站 eNB 阈值）分支。
// 这从 SQL/CASE 形状层面证明：NULL product_class 设备按基站阈值（enb）判离线，
// 不会被误用 CPE 阈值（cpe）。真正在 PG 上的求值验证见下方 //go:build integration 版本。
func TestFindStaleDevicesByClass_NullProductClass_FallsIntoENBBranch(t *testing.T) {
	query, args := buildFindStaleByClassSQLForTest(t, 600, 100, 1000)

	// CPE 分类完全依赖 product_class 的 ILIKE 模式匹配——没有对 NULL 的显式兜底
	// （如 COALESCE(product_class,'') 或 IS NOT NULL），因此 NULL 三值逻辑下
	// CPE 分支不可能为 TRUE，必然落 ELSE。
	caseStart := strings.Index(query, "CASE WHEN")
	caseEnd := strings.Index(query, "END")
	require.GreaterOrEqual(t, caseStart, 0)
	require.Greater(t, caseEnd, caseStart, "SQL 应含完整 CASE ... END")
	caseExpr := query[caseStart : caseEnd+len("END")]

	// CASE 谓词只用 product_class ILIKE，未对 NULL 做 COALESCE/IS NULL 兜底，
	// 故 NULL product_class 三值逻辑下走 ELSE 分支。
	assert.Contains(t, caseExpr, "product_class ILIKE",
		"CPE 分类只靠 product_class ILIKE 模式匹配")
	assert.NotContains(t, caseExpr, "COALESCE",
		"未对 product_class NULL 做 COALESCE 兜底——NULL 必落 ELSE（eNB）")
	assert.NotContains(t, caseExpr, "IS NULL",
		"CASE 谓词未显式处理 NULL——NULL ILIKE 求值 NULL→非 TRUE→走 ELSE（eNB）")
	assert.Contains(t, caseExpr, "ELSE", "必须有 ELSE 兜底承接 NULL/非 CPE 设备")

	// ELSE 分支绑的是 enbThresholdSec（args[2]，$3）——即 NULL product_class 设备用基站阈值。
	// args 顺序见 SQLBuild 测试：[is_online, cpe($2,THEN), enb($3,ELSE)]。
	require.Len(t, args, 3)
	assert.EqualValues(t, 100, args[2], "ELSE（NULL/eNB）分支用 enbThresholdSec（$3）")
}

// TestAlarmSeverityTextToCodes 锁定 #361 文本↔severity 编码映射：
// critical/major/minor/warning 同时兼容历史 1~4 与现行 31001~31004，未知→nil。
func TestAlarmSeverityTextToCodes(t *testing.T) {
	cases := map[string][]int{
		"critical": {1, 31001}, "major": {2, 31002}, "minor": {3, 31003}, "warning": {4, 31004},
		"CRITICAL": {1, 31001}, " Major ": {2, 31002}, // 大小写/空白容错
		"none": nil, "": nil, "bogus": nil,
	}
	for text, want := range cases {
		assert.Equalf(t, want, alarmSeverityTextToCodes(text),
			"alarmSeverityTextToCodes(%q)", text)
	}
}

// TestAlarmSeverityFilterCond_SQLShape 验证 #361 告警级别筛选用相关子查询匹配
// 「该设备未 cleared 活动告警最严重级别 = 请求级别」，与列表展示口径一致；
// 未知级别返回 nil（不过滤）。
func TestAlarmSeverityFilterCond_SQLShape(t *testing.T) {
	// 已知级别：构造出相关子查询条件，占位参数 = 该级别数值。
	cond := alarmSeverityFilterCond("major")
	require.NotNil(t, cond)
	sqlStr, args, err := cond.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sqlStr, "FROM alarms_active")
	assert.Contains(t, sqlStr, "MIN(aaf.severity)")
	assert.Contains(t, sqlStr, "status <> 'cleared'")
	assert.Contains(t, sqlStr, "IN (")
	require.Len(t, args, 2)
	assert.EqualValues(t, 2, args[0], "major → legacy severity=2")
	assert.EqualValues(t, 31002, args[1], "major → dictionary severity=31002")

	// 未知级别：不过滤。
	assert.Nil(t, alarmSeverityFilterCond("none"))
	assert.Nil(t, alarmSeverityFilterCond(""))
}

// TestDeviceWithInfoSelectColumns_AlarmAggregation 验证 #361 list select 列已切到
// alarms_active 实时聚合派生值（CASE aa.top_sev → 文本 + aa.active_alarm_count），
// 不再读无人维护的 di.alarm_severity 冗余列。
func TestDeviceWithInfoSelectColumns_AlarmAggregation(t *testing.T) {
	cols := deviceWithInfoSelectColumns()
	joined := strings.Join(cols, " || ")

	assert.Contains(t, joined, "COALESCE(dgm.source_type, 'auto')", "无真实归属行的默认组视图必须派生为 auto")
	assert.Contains(t, joined, "as source_type", "列表 DTO 必须暴露归属来源字段")

	assert.Contains(t, joined, "CASE aa.top_sev", "告警级别列必须来自聚合派生 CASE")
	assert.Contains(t, joined, "31001", "critical 需兼容 31001 编码")
	assert.Contains(t, joined, "31002", "major 需兼容 31002 编码")
	assert.Contains(t, joined, "31003", "minor 需兼容 31003 编码")
	assert.Contains(t, joined, "31004", "warning 需兼容 31004 编码")
	assert.Contains(t, joined, "AS alarm_severity", "派生列别名仍为 alarm_severity（前端契约不变）")
	assert.Contains(t, joined, "aa.active_alarm_count", "必须暴露活动告警数列")
	assert.Contains(t, joined, "d.last_param_sync_at", "列表 DTO 必须暴露最近参数同步完成时间")
	assert.Contains(t, joined, "AS param_sync_running", "列表 DTO 必须暴露参数同步动态状态列")
	assert.Contains(t, joined, "parameter_sync_requests", "durable paramsync 请求未终态时应显示同步中")
	assert.Contains(t, joined, "parameter_sync_runs", "durable paramsync 运行未终态时应显示同步中")
	assert.NotContains(t, joined, "device_tasks dt", "旧 Path B sync-gpv 任务已废弃，动态状态不得依赖 device_tasks")
	assert.NotContains(t, joined, "sync-gpv-", "动态状态只认 durable parameter_sync_* 数据面")
	assert.NotContains(t, joined, "di.alarm_severity",
		"#361：列表 select 不再读无人维护的 di.alarm_severity 冗余列")

	// aa JOIN 子句聚合 alarms_active：MIN(severity)=top_sev、COUNT(*)=active_alarm_count、排除 cleared。
	assert.Contains(t, alarmsActiveAggJoin, "MIN(severity)")
	assert.Contains(t, alarmsActiveAggJoin, "COUNT(*)")
	assert.Contains(t, alarmsActiveAggJoin, "status <> 'cleared'")
	assert.Contains(t, alarmsActiveAggJoin, "aa ON aa.device_id = d.id")
}

func TestDeviceWithInfoListContractIncludesUECount(t *testing.T) {
	assert.Contains(t, deviceWithInfoSelectColumns(), "COALESCE(di.ue_count, 0)",
		"设备列表查询必须读取 device_info.ue_count")

	body, err := json.Marshal(DeviceWithInfo{UECount: 9})
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.EqualValues(t, 9, payload["ue_count"],
		"设备列表响应必须向前端输出 ue_count")
}

func TestBuildRecycleBinListBuilders_SearchIncludesMACInListAndCount(t *testing.T) {
	search := "48:BF"
	listBuilder, countBuilder := buildRecycleBinListBuilders(RecycleBinFilter{
		Search: &search,
	})

	listSQL, listArgs, err := listBuilder.ToSql()
	require.NoError(t, err)
	countSQL, countArgs, err := countBuilder.ToSql()
	require.NoError(t, err)

	assert.Contains(t, listSQL, "d.serial_number ILIKE", "回收站搜索仍需支持 SN")
	assert.Contains(t, listSQL, "d.site_name ILIKE", "回收站搜索仍需支持名称")
	assert.Contains(t, listSQL, "di.mac ILIKE", "回收站搜索框承诺支持 MAC，应查询 device_info.mac")
	assert.Contains(t, countSQL, "LEFT JOIN device_info di ON di.device_id = d.id",
		"分页总数查询也必须 join device_info，否则 MAC 搜索下 count SQL 无法引用 di.mac")
	assert.Contains(t, countSQL, "di.mac ILIKE", "分页总数口径必须与列表查询一致")
	assert.Equal(t, []interface{}{"%48:BF%", "%48:BF%", "%48:BF%"}, listArgs)
	assert.Equal(t, listArgs, countArgs)
}

func TestBuildRecycleBinListBuilders_GroupFilterUsesPreservedMemberships(t *testing.T) {
	groupID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	listBuilder, countBuilder := buildRecycleBinListBuilders(RecycleBinFilter{
		GroupID: &groupID,
	})

	listSQL, listArgs, err := listBuilder.ToSql()
	require.NoError(t, err)
	countSQL, countArgs, err := countBuilder.ToSql()
	require.NoError(t, err)

	assert.Contains(t, listSQL, "LEFT JOIN device_group_members dgm ON d.id = dgm.device_id")
	assert.Contains(t, listSQL, "dgm.group_id = $1")
	assert.Contains(t, countSQL, "JOIN device_group_members dgm ON d.id = dgm.device_id")
	assert.Contains(t, countSQL, "dgm.group_id = $1")
	assert.Equal(t, []interface{}{groupID.String()}, listArgs)
	assert.Equal(t, listArgs, countArgs)
}

func TestBuildRecycleBinListBuilders_DefaultGroupIncludesLegacyUngroupedDevices(t *testing.T) {
	groupID := uuid.MustParse(global.DefaultLevel2GroupID)
	listBuilder, countBuilder := buildRecycleBinListBuilders(RecycleBinFilter{
		GroupID: &groupID,
	})

	listSQL, _, err := listBuilder.ToSql()
	require.NoError(t, err)
	countSQL, _, err := countBuilder.ToSql()
	require.NoError(t, err)

	assert.Contains(t, listSQL, "dgm.group_id = $1")
	assert.Contains(t, listSQL, ungroupedDevicesWhere)
	assert.Contains(t, countSQL, "LEFT JOIN device_group_members dgm ON d.id = dgm.device_id")
	assert.Contains(t, countSQL, "dgm.group_id = $1")
	assert.Contains(t, countSQL, ungroupedDevicesWhere)
}

func TestApplyDeviceGroupFilter_DefaultGroupIncludesLegacyUngroupedDevices(t *testing.T) {
	defaultGroup := uuid.MustParse(global.DefaultLevel2GroupID)
	builder := sq.Select("d.id").
		From("devices d").
		LeftJoin("device_group_members dgm ON dgm.device_id = d.id").
		PlaceholderFormat(sq.Dollar)

	got := applyDeviceGroupFilter(builder, DeviceFilter{GroupID: &defaultGroup})
	sql, args, err := got.ToSql()
	require.NoError(t, err)

	assert.Contains(t, sql, "dgm.group_id IN ($1)")
	assert.Contains(t, sql, ungroupedDevicesWhere)
	assert.Equal(t, []interface{}{defaultGroup}, args)
}

func TestApplyDeviceGroupFilter_RealGroupDoesNotIncludeLegacyUngroupedDevices(t *testing.T) {
	groupID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	builder := sq.Select("d.id").
		From("devices d").
		LeftJoin("device_group_members dgm ON dgm.device_id = d.id").
		PlaceholderFormat(sq.Dollar)

	got := applyDeviceGroupFilter(builder, DeviceFilter{GroupID: &groupID})
	sql, args, err := got.ToSql()
	require.NoError(t, err)

	assert.Contains(t, sql, "dgm.group_id IN ($1)")
	assert.NotContains(t, sql, ungroupedDevicesWhere)
	assert.Equal(t, []interface{}{groupID}, args)
}

func TestRecycleBinSelectColumns_ExposeStableRecycleMetadata(t *testing.T) {
	joined := strings.Join(recycleBinSelectColumns(), "\n")

	assert.Equal(t, len(deviceWithInfoSelectColumns()), len(recycleBinSelectColumns()),
		"回收站复用 scanDeviceWithInfoRow，SELECT 列数必须与共享 scanner 对齐")
	assert.Contains(t, joined, "d.recycle_type")
	assert.Contains(t, joined, "d.recycle_executor")
	assert.Contains(t, joined, "d.last_param_sync_at")
	assert.Contains(t, joined, "FALSE AS param_sync_running")
	assert.Contains(t, joined, "d.deleted_at - d.last_inform_at",
		"离线时长必须固定在移入回收站时刻，不能随查询时间继续增长")
	assert.NotContains(t, joined, "NOW() - di.last_offline_time",
		"回收站离线时长不得使用当前时间动态重算")
}

func TestRecycleBinListIncludesLocationObservationColumnsForSharedScanner(t *testing.T) {
	joined := strings.Join(recycleBinSelectColumns(), "\n")
	assert.Contains(t, joined, "dlo.latitude AS reported_latitude")
	assert.Contains(t, joined, "dlo.longitude AS reported_longitude")
	assert.Contains(t, joined, "dlo.gps_height AS reported_gps_height")
	assert.Contains(t, joined, "dlo.observed_at AS reported_observed_at")
	assert.Contains(t, joined, "dlo.version AS reported_version")
	assert.Contains(t, joined, "dlo.source_path AS reported_source_path")

	listBuilder, _ := buildRecycleBinListBuilders(RecycleBinFilter{})
	listSQL, _, err := listBuilder.ToSql()
	require.NoError(t, err)
	assert.Contains(t, listSQL, "LEFT JOIN device_location_observations dlo ON d.id = dlo.device_id")
}
