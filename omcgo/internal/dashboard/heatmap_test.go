package dashboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestGetAlarmHeatmap tests the GetAlarmHeatmap function.
func TestGetAlarmHeatmap(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testCases := []struct {
		name      string
		days      int
		wantError bool
	}{
		{
			name:      "default 30 days",
			days:      30,
			wantError: false,
		},
		{
			name:      "7 days",
			days:      7,
			wantError: false,
		},
		{
			name:      "90 days",
			days:      90,
			wantError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// TODO: Implement with test database
			// service := createTestService(t)
			// result, err := service.GetAlarmHeatmap(context.Background(), tc.days)
			// if tc.wantError {
			//     assert.Error(t, err)
			// } else {
			//     assert.NoError(t, err)
			//     assert.NotNil(t, result)
			// }
		})
	}
}

// TestGetAlarmHeatmapBySeverity tests the GetAlarmHeatmapBySeverity function.
func TestGetAlarmHeatmapBySeverity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testCases := []struct {
		name      string
		days      int
		severity  string
		wantError bool
	}{
		{
			name:      "all severities",
			days:      30,
			severity:  "",
			wantError: false,
		},
		{
			name:      "critical only",
			days:      30,
			severity:  "critical",
			wantError: false,
		},
		{
			name:      "major only",
			days:      30,
			severity:  "major",
			wantError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// TODO: Implement with test database
		})
	}
}

// TestGetAlarmHeatmapAll tests the GetAlarmHeatmapAll function.
func TestGetAlarmHeatmapAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("returns data for all severities", func(t *testing.T) {
		// TODO: Implement with test database
	})
}

// TestPostgreSQLDOWConversion tests the DOW conversion logic.
func TestPostgreSQLDOWConversion(t *testing.T) {
	// PostgreSQL DOW: 0=Sunday, 1=Monday, ..., 6=Saturday
	// Our adjusted DOW: 0=Monday, 1=Tuesday, ..., 6=Sunday
	testCases := []struct {
		name        string
		pgDOW       int // PostgreSQL DOW
		expectedDOW int // Adjusted DOW (0=Monday)
	}{
		{"Sunday to Monday", 0, 6},
		{"Monday stays Monday", 1, 0},
		{"Tuesday stays Tuesday", 2, 1},
		{"Wednesday stays Wednesday", 3, 2},
		{"Thursday stays Thursday", 4, 3},
		{"Friday stays Friday", 5, 4},
		{"Saturday stays Saturday", 6, 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			adjustedDay := (tc.pgDOW + 6) % 7
			assert.Equal(t, tc.expectedDOW, adjustedDay)
		})
	}
}

// TestDayOfWeekDataStructure tests the DayOfWeekData structure.
func TestDayOfWeekDataStructure(t *testing.T) {
	data := DayOfWeekData{
		Day:   0,
		Hours: make([]int64, 24),
	}

	assert.Equal(t, 0, data.Day, "day should be initialized to 0")
	assert.Equal(t, 24, len(data.Hours), "should have 24 hours")

	// Test setting and getting hour values
	data.Hours[0] = 100
	data.Hours[23] = 50

	assert.Equal(t, int64(100), data.Hours[0])
	assert.Equal(t, int64(50), data.Hours[23])
}

// TestSeverityFilterValue 验证 severity 标签 → smallint 过滤值的整型语义转换
// （回归 #119：text 参数与 smallint 列直接比较导致 SQLSTATE 42883）。
func TestSeverityFilterValue(t *testing.T) {
	testCases := []struct {
		name     string
		severity string
		want     int
	}{
		{"empty means no filter (0 sentinel)", "", 0},
		{"critical maps to 1", "critical", 1},
		{"major maps to 2", "major", 2},
		{"minor maps to 3", "minor", 3},
		{"warning maps to 4", "warning", 4},
		{"unknown label maps to 5 (matches nothing)", "bogus", 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, severityFilterValue(tc.severity))
		})
	}
}

// fakeHeatmapScanner 模拟 pgx rows，供 scanHeatmapBySeverity 单测使用。
type fakeHeatmapScanner struct {
	rows [][4]int // severity, day_of_week(PG DOW), hour_of_day, alarm_count
	idx  int
}

func (f *fakeHeatmapScanner) Next() bool {
	if f.idx >= len(f.rows) {
		return false
	}
	f.idx++
	return true
}

func (f *fakeHeatmapScanner) Scan(dest ...any) error {
	row := f.rows[f.idx-1]
	*(dest[0].(*int)) = row[0]
	*(dest[1].(*int)) = row[1]
	*(dest[2].(*int)) = row[2]
	*(dest[3].(*int64)) = int64(row[3])
	return nil
}

// TestScanHeatmapBySeverity 验证 smallint severity 扫描为整型后转标签作 map key
// （回归 #119：原实现把 smallint 扫进 string，pgx 扫描必失败）。
func TestScanHeatmapBySeverity(t *testing.T) {
	svc := &Service{logger: zap.NewNop()}

	testCases := []struct {
		name       string
		rows       [][4]int
		wantKeys   []string
		wantChecks func(t *testing.T, m map[string]*HeatmapData)
	}{
		{
			name: "severity smallint values map to label keys",
			rows: [][4]int{
				{1, 1, 0, 10}, // critical, PG Monday, 00h
				{2, 1, 5, 3},  // major
				{4, 0, 23, 7}, // warning, PG Sunday, 23h
			},
			wantKeys: []string{"critical", "major", "warning"},
			wantChecks: func(t *testing.T, m map[string]*HeatmapData) {
				// PG DOW 1=Monday → adjusted 0；PG DOW 0=Sunday → adjusted 6
				assert.Equal(t, int64(10), m["critical"].DaysOfWeek[0].Hours[0])
				assert.Equal(t, int64(10), m["critical"].MaxCount)
				assert.Equal(t, int64(7), m["warning"].DaysOfWeek[6].Hours[23])
			},
		},
		{
			name: "dictionary severity codes map to label keys",
			rows: [][4]int{
				{31001, 1, 0, 10},
				{31002, 1, 5, 3},
				{31004, 0, 23, 7},
			},
			wantKeys: []string{"critical", "major", "warning"},
			wantChecks: func(t *testing.T, m map[string]*HeatmapData) {
				assert.Equal(t, int64(10), m["critical"].DaysOfWeek[0].Hours[0])
				assert.Equal(t, int64(7), m["warning"].DaysOfWeek[6].Hours[23])
			},
		},
		{
			name: "same bucket accumulates counts",
			rows: [][4]int{
				{1, 1, 0, 10},
				{31001, 1, 0, 5},
			},
			wantKeys: []string{"critical"},
			wantChecks: func(t *testing.T, m map[string]*HeatmapData) {
				assert.Equal(t, int64(15), m["critical"].DaysOfWeek[0].Hours[0])
				assert.Equal(t, int64(15), m["critical"].MaxCount)
			},
		},
		{
			name: "MaxCount tracks the largest bucket per severity",
			rows: [][4]int{
				{3, 2, 8, 5},
				{3, 2, 9, 12},
				{3, 3, 1, 4},
			},
			wantKeys: []string{"minor"},
			wantChecks: func(t *testing.T, m map[string]*HeatmapData) {
				assert.Equal(t, int64(12), m["minor"].MaxCount)
			},
		},
		{
			name:     "no rows yields empty map",
			rows:     nil,
			wantKeys: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.scanHeatmapBySeverity(&fakeHeatmapScanner{rows: tc.rows})
			require.NoError(t, err)
			assert.Len(t, result, len(tc.wantKeys))
			for _, key := range tc.wantKeys {
				assert.Contains(t, result, key)
			}
			if tc.wantChecks != nil {
				tc.wantChecks(t, result)
			}
		})
	}

	t.Run("invalid rows type returns error", func(t *testing.T) {
		_, err := svc.scanHeatmapBySeverity("not a scanner")
		require.Error(t, err)
	})
}

// TestSelectHeatmapResult 验证标签 key 与请求侧 severity 过滤参数对齐后能正确选中结果。
func TestSelectHeatmapResult(t *testing.T) {
	svc := &Service{logger: zap.NewNop()}

	critical := initializeEmptyHeatmap()
	critical.MaxCount = 10
	major := initializeEmptyHeatmap()
	major.MaxCount = 3

	heatmaps := map[string]*HeatmapData{
		"critical": critical,
		"major":    major,
	}

	testCases := []struct {
		name         string
		severity     string
		wantSeverity string
		wantMaxCount int64
	}{
		{"filter by critical returns its data", "critical", "critical", 10},
		{"empty filter returns severity with max count", "", "critical", 10},
		{"unmatched filter returns empty structure", "warning", "warning", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := svc.selectHeatmapResult(heatmaps, tc.severity)
			require.NotNil(t, got)
			assert.Equal(t, tc.wantSeverity, got.Severity)
			require.NotNil(t, got.Data)
			assert.Equal(t, tc.wantMaxCount, got.Data.MaxCount)
		})
	}

	t.Run("empty map with empty filter returns all placeholder", func(t *testing.T) {
		got := svc.selectHeatmapResult(map[string]*HeatmapData{}, "")
		require.NotNil(t, got)
		assert.Equal(t, "all", got.Severity)
	})
}

// BenchmarkGetAlarmHeatmap benchmarks the GetAlarmHeatmap function.
func BenchmarkGetAlarmHeatmap(b *testing.B) {
	// TODO: Set up benchmark with realistic data volume
	b.ReportAllocs()
}

// BenchmarkGetAlarmHeatmapBySeverity benchmarks the GetAlarmHeatmapBySeverity function.
func BenchmarkGetAlarmHeatmapBySeverity(b *testing.B) {
	// TODO: Set up benchmark with realistic data volume
	b.ReportAllocs()
}
