package indicator

import (
	"context"
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/appconfig"
)

func TestXMLIndicatorDoesNotModelEnabledAttribute(t *testing.T) {
	_, ok := reflect.TypeOf(xmlIndicator{}).FieldByName("Enabled")
	assert.False(t, ok, "Loader 不应再解析 XML enabled 属性或维护 enabled_pm_indicators_*")
}

func TestFlattenDocsByDeviceTypeIgnoresXMLEnabledAttribute(t *testing.T) {
	var doc xmlIndicatorModel
	require.NoError(t, xml.Unmarshal([]byte(`
<indicatorModel platform="BLQ">
  <indicators>
    <indicator id="C0001" enabled="false" isCounter="1" formula="C0001" />
  </indicators>
</indicatorModel>`), &doc))

	records, formulas := flattenDocsByDeviceType([]xmlIndicatorModel{doc}, false)

	require.Contains(t, records, "C0001")
	assert.Equal(t, "C0001", records["C0001"].Ind.ID)
	require.Len(t, formulas, 1)
	assert.Equal(t, "C0001", formulas[0].IndicatorID)
}

func TestSeedBaselineGNBDefaultEnabledMatchesGNBXML(t *testing.T) {
	docs := parseLibFiles(t, "GNB.xml")
	want := make([]string, 0, len(docs[0].Indicators))
	for _, ind := range docs[0].Indicators {
		if ind.ID != "" {
			want = append(want, ind.ID)
		}
	}
	sort.Strings(want)

	seed, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "seed", "000001_init_seed.sql"))
	require.NoError(t, err)
	re := regexp.MustCompile(`(?s)INSERT INTO public\.enabled_pm_indicators_gnb \(operator_code, indicator_id\).*?FROM regexp_split_to_table\(\$ids\$\n(.*?)\n\$ids\$`)
	m := re.FindSubmatch(seed)
	require.Len(t, m, 2, "seed 基线应包含 GNB 默认启用清单")

	got := regexp.MustCompile(`\s+`).Split(string(m[1]), -1)
	got = compactNonEmpty(got)
	sort.Strings(got)
	assert.Equal(t, want, got, "GNB 默认全部启用清单必须与 GNB.xml 保持一致")
}

// #98：reload 重灌 builtin 公式时，已被用户自定义公式覆盖的 (平台, 指标) 必须跳过，
// 避免与保留下来的自定义行重复 / 覆盖用户意图。
func TestExcludeCustomOverridden(t *testing.T) {
	formulas := []formulaRow{
		{PlatformName: "P1", IndicatorID: "I1", Formula: "a", LoadedFrom: "x.xml"},
		{PlatformName: "P1", IndicatorID: "I2", Formula: "b", LoadedFrom: "x.xml"},
		{PlatformName: "P2", IndicatorID: "I1", Formula: "c", LoadedFrom: "y.xml"},
	}

	// 无自定义键 → 原样返回
	assert.Len(t, excludeCustomOverridden(formulas, nil), 3)

	// 自定义覆盖 P1/I1 → 丢弃该 builtin 行，保留其余两条；
	// (平台,指标) 是联合键，P2/I1 不同键不应被误删。
	custom := map[formulaKey]struct{}{{platform: "P1", indicator: "I1"}: {}}
	got := excludeCustomOverridden(formulas, custom)
	assert.Len(t, got, 2)
	hasP1I1, hasP2I1 := false, false
	for _, f := range got {
		if f.PlatformName == "P1" && f.IndicatorID == "I1" {
			hasP1I1 = true
		}
		if f.PlatformName == "P2" && f.IndicatorID == "I1" {
			hasP2I1 = true
		}
	}
	assert.False(t, hasP1I1, "被自定义覆盖的 P1/I1 不应重插")
	assert.True(t, hasP2I1, "P2/I1 是不同键，不应被 P1/I1 的覆盖误删")
}

// findIndicatorByID 在已解析的 XML 文档集合里按 id 找指标（issue #389 阶段1 校验用）。
func findIndicatorByID(docs []xmlIndicatorModel, id string) (xmlIndicator, bool) {
	for _, doc := range docs {
		for _, ind := range doc.Indicators {
			if ind.ID == id {
				return ind, true
			}
		}
	}
	return xmlIndicator{}, false
}

func compactNonEmpty(in []string) []string {
	out := in[:0]
	for _, s := range in {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// parseLibFiles 读取并解析给定相对路径（相对 data/indicator-library）的 XML 文件集合。
func parseLibFiles(t *testing.T, rels ...string) []xmlIndicatorModel {
	t.Helper()
	docs := make([]xmlIndicatorModel, 0, len(rels))
	for _, rel := range rels {
		var doc xmlIndicatorModel
		require.NoError(t, readXML(filepath.Join("..", "..", "..", "data", "indicator-library", rel), &doc), rel)
		docs = append(docs, doc)
	}
	return docs
}

// TestStatisDurationAndCellAvailabilityRegistered 守 issue #389 阶段1 死判
// [indicators-registered-3tech]：指标库三制式各自登记「统计时长」计数器（is_counter=1 /
// statisType=sum）与「小区可用率」派生指标（is_counter=0 / statisType=pct / 公式含
// 在服时长÷统计时长×100）。直接解析出厂 XML 资产校验，与 dictloader 落库前的真相源对齐。
func TestStatisDurationAndCellAvailabilityRegistered(t *testing.T) {
	cases := []struct {
		tech       string
		files      []string
		durationID string
		availID    string
		availArith string // 期望小区可用率公式（arithmetic，C 编号口径）
	}{
		{
			tech:       "4G/ENB",
			files:      []string{"enb/BLX.xml", "enb/BM.xml", "enb/MLQ.xml", "enb/BLQ.xml", "enb/MLN.xml"},
			durationID: "C000060273",
			availID:    "K900010076",
			availArith: "C000060216/C000060273*100",
		},
		{
			tech:       "5G/GNB",
			files:      []string{"GNB.xml"},
			durationID: "C010120025",
			availID:    "KGNB0570",
			availArith: "OTHER.CellServiceTime/C010120025*100",
		},
		{
			tech:       "GSM",
			files:      []string{"GSM.xml"},
			durationID: "CGSM0080001",
			availID:    "KGSM0143",
			availArith: "OTHER.CellServiceTime/CGSM0080001*100",
		},
	}
	for _, c := range cases {
		t.Run(c.tech, func(t *testing.T) {
			docs := parseLibFiles(t, c.files...)

			// 统计时长：计数器 + sum + 合成（reportKey=OTHER.StatisDuration）。
			dur, ok := findIndicatorByID(docs, c.durationID)
			require.True(t, ok, "%s 统计时长 %s 应登记", c.tech, c.durationID)
			assert.Equal(t, "1", dur.IsCounter, "统计时长应为计数器")
			assert.Equal(t, "sum", dur.StatisType, "统计时长应累加")
			assert.Equal(t, "OTHER.StatisDuration", dur.ReportKey)
			assert.Equal(t, "统计时长", dur.CnName)

			// 小区可用率：派生 + pct + 公式 = 在服时长÷统计时长×100。
			av, ok := findIndicatorByID(docs, c.availID)
			require.True(t, ok, "%s 小区可用率 %s 应登记", c.tech, c.availID)
			assert.Equal(t, "0", av.IsCounter, "小区可用率应为派生指标")
			assert.Equal(t, "pct", av.StatisType, "小区可用率统计类型应为 pct")
			assert.Equal(t, c.availArith, av.Arithmetic, "%s 小区可用率公式应为在服÷统计×100", c.tech)
			assert.Equal(t, "OTHER.CellServiceTime/OTHER.StatisDuration*100", av.Formula)
			assert.Equal(t, "小区可用率", av.CnName)
		})
	}
}

func TestGSMBuiltInDurationKPICompilesToRuntimeCounter(t *testing.T) {
	docs := parseLibFiles(t, "GSM.xml")
	cases := []struct {
		id             string
		rawArithmetic  string
		runtimeFormula string
		deps           []string
	}{
		{
			id:             "KGSM0108",
			rawArithmetic:  "((CGSM0040004/1000)/(CGSM0040003*Duration))*100",
			runtimeFormula: "((CGSM0040004/1000)/(CGSM0040003*CGSM0080001))*100",
			deps:           []string{"CGSM0040004", "CGSM0040003", "CGSM0080001"},
		},
		{
			id:             "KGSM0109",
			rawArithmetic:  "(1-(CGSM0040005)/(CGSM0040003*Duration))*100",
			runtimeFormula: "(1-(CGSM0040005)/(CGSM0040003*CGSM0080001))*100",
			deps:           []string{"CGSM0040005", "CGSM0040003", "CGSM0080001"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			ind, ok := findIndicatorByID(docs, tc.id)
			require.True(t, ok, "GSM 内置 KPI %s 应存在", tc.id)
			require.Equal(t, "0", ind.IsCounter)
			require.Equal(t, tc.rawArithmetic, ind.Arithmetic, "XML 资产保留 Duration 可读写法")

			runtimeFormula := CompileRuntimeArithmetic(DeviceTypeGSM, ind.Arithmetic)
			require.Equal(t, tc.runtimeFormula, runtimeFormula)
			assert.NotContains(t, runtimeFormula, "Duration")
			for _, dep := range tc.deps {
				assert.Contains(t, runtimeFormula, dep)
			}
		})
	}
}

// ISSUE-389 阶段2 修复：指标库加载成功后必须 bump 下游 KPI 路由缓存，
// 否则新增的「统计时长」计数器 report_key 不在已缓存的旧路由白名单里，落库被丢弃。
// 这里覆盖 bump 调用的三条语义：成功调用、bump 报错不影响加载、nil bumper 安全空跑。

func TestBumpDownstreamCache_InvokesBumperOnSuccess(t *testing.T) {
	called := 0
	l := NewLoader(nil, appcfg(), "", nil).WithCacheBumper(func(_ context.Context) error {
		called++
		return nil
	})
	l.bumpDownstreamCache(context.Background())
	assert.Equal(t, 1, called, "加载成功后应调用一次 cacheBumper（bump KPI 路由 cache_version）")
}

func TestBumpDownstreamCache_BumpErrorDoesNotPanic(t *testing.T) {
	l := NewLoader(nil, appcfg(), "", nil).WithCacheBumper(func(_ context.Context) error {
		return errors.New("redis down")
	})
	// bump 失败仅告警（下游靠 24h TTL 兜底），不得 panic / 不抛错。
	assert.NotPanics(t, func() { l.bumpDownstreamCache(context.Background()) })
}

func TestBumpDownstreamCache_NilBumperIsNoOp(t *testing.T) {
	l := NewLoader(nil, appcfg(), "", nil) // 未注入 bumper（无 Redis 部署）
	assert.NotPanics(t, func() { l.bumpDownstreamCache(context.Background()) })
}

func appcfg() appconfig.IndicatorLoaderConfig { return appconfig.IndicatorLoaderConfig{} }
