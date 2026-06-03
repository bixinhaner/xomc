package indicator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PM-P1: 验证装载器把 reportKey 落到 report_key 列(与 en_name 各就各位),
// 含 reportKey≠enName 的派生 KPI 场景。这里不依赖 DB,直接验证
// indicatorColumns / indicatorRowValues 这对纯函数的列↔值对位关系。

// colIndex 返回列名在列清单中的下标,找不到返回 -1。
func colIndex(cols []string, name string) int {
	for i, c := range cols {
		if c == name {
			return i
		}
	}
	return -1
}

func TestIndicatorColumns_HasReportKey(t *testing.T) {
	for _, isGnb := range []bool{false, true} {
		cols := indicatorInsertColumns(isGnb)
		assert.Contains(t, cols, "report_key", "report_key 必须进 cols(isGnb=%v)", isGnb)
		// report_key 紧跟 loaded_from(同侧)
		li := colIndex(cols, "loaded_from")
		ri := colIndex(cols, "report_key")
		require.GreaterOrEqual(t, li, 0)
		require.GreaterOrEqual(t, ri, 0)
		assert.Equal(t, li+1, ri, "report_key 应紧跟 loaded_from")
		if !isGnb {
			// 非 GNB 时 indicator_level 仍是尾列,report_key 在其之前
			ii := colIndex(cols, "indicator_level")
			require.GreaterOrEqual(t, ii, 0)
			assert.Greater(t, ii, ri, "indicator_level 仍是尾列(在 report_key 之后)")
		}
	}
}

func TestIndicatorRowValues_ColsRowAligned(t *testing.T) {
	// 列数与值数严格相等(对位的前提)
	for _, isGnb := range []bool{false, true} {
		rec := indicatorRecord{
			Ind: xmlIndicator{
				ID:        "C000060216",
				EnName:    "OTHER.CellServiceTime",
				ReportKey: "OTHER.CellServiceTime",
			},
			LoadedFrom: "indicator-library/enb/ALL.xml",
		}
		cols := indicatorInsertColumns(isGnb)
		row := indicatorInsertRow(rec, isGnb)
		assert.Len(t, row, len(cols), "isGnb=%v: 值数必须等于列数", isGnb)
	}
}

func TestIndicatorRowValues_ReportKeyAndEnNameSeparateSlots(t *testing.T) {
	// 派生 KPI:reportKey 与 enName 不同,验证两字段分别落到各自列槽。
	rec := indicatorRecord{
		Ind: xmlIndicator{
			ID:        "K000010001",
			EnName:    "L.E-RAB.SuccEst.Ratio",      // 展示名
			ReportKey: "L.E-RAB.SuccEst.Ratio.RAW",  // 上报名,刻意与 enName 不同
		},
		LoadedFrom: "indicator-library/enb/ALL.xml",
	}
	cols := indicatorInsertColumns(false)
	row := indicatorInsertRow(rec, false)
	require.Len(t, row, len(cols))

	enIdx := colIndex(cols, "en_name")
	rkIdx := colIndex(cols, "report_key")
	require.GreaterOrEqual(t, enIdx, 0)
	require.GreaterOrEqual(t, rkIdx, 0)

	assert.Equal(t, "L.E-RAB.SuccEst.Ratio", row[enIdx], "en_name 列落展示名")
	assert.Equal(t, "L.E-RAB.SuccEst.Ratio.RAW", row[rkIdx], "report_key 列落上报名")
	assert.NotEqual(t, row[enIdx], row[rkIdx], "派生 KPI 两字段值不同,确认未串列")
}

func TestIndicatorRowValues_EmptyReportKeyNullified(t *testing.T) {
	// reportKey 为空时走 nullIfEmpty → nil(可空列,不强制非空)。
	rec := indicatorRecord{
		Ind: xmlIndicator{
			ID:        "X000000001",
			EnName:    "Some.Indicator",
			ReportKey: "", // 缺省
		},
		LoadedFrom: "indicator-library/enb/ALL.xml",
	}
	cols := indicatorInsertColumns(false)
	row := indicatorInsertRow(rec, false)
	rkIdx := colIndex(cols, "report_key")
	require.GreaterOrEqual(t, rkIdx, 0)
	assert.Nil(t, row[rkIdx], "空 reportKey 落 NULL(nullIfEmpty)")
}

func TestIndicatorRowValues_EnNameFallsBackToID(t *testing.T) {
	// enName 缺省回退 ID,但 report_key 独立(不应被回退污染)。
	rec := indicatorRecord{
		Ind: xmlIndicator{
			ID:        "C000060216",
			EnName:    "", // 缺省 → 回退 ID
			ReportKey: "OTHER.CellServiceTime",
		},
	}
	cols := indicatorInsertColumns(false)
	row := indicatorInsertRow(rec, false)
	enIdx := colIndex(cols, "en_name")
	rkIdx := colIndex(cols, "report_key")
	assert.Equal(t, "C000060216", row[enIdx], "en_name 缺省回退 ID")
	assert.Equal(t, "OTHER.CellServiceTime", row[rkIdx], "report_key 不受 en_name 回退影响")
}
