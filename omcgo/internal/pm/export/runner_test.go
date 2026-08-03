package export

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
)

// ── stub repo / uploader / source ────────────────────────────────────────────

type stubTaskRepo struct {
	task *Task

	getErr     error
	runErr     error
	succErr    error
	failErr    error
	markedRun  int
	succeeded  *succArgs
	failedWith string
	failedN    int
}

type succArgs struct {
	bucket, filePath string
	fileSize, rowN   int64
}

func (s *stubTaskRepo) Get(_ context.Context, _ uuid.UUID) (*Task, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.task, nil
}
func (s *stubTaskRepo) MarkRunning(_ context.Context, _ uuid.UUID) error {
	s.markedRun++
	return s.runErr
}
func (s *stubTaskRepo) MarkSucceeded(_ context.Context, _ uuid.UUID, bucket, fp string, fs, rn int64) error {
	s.succeeded = &succArgs{bucket, fp, fs, rn}
	return s.succErr
}
func (s *stubTaskRepo) MarkFailed(_ context.Context, _ uuid.UUID, msg string) error {
	s.failedN++
	s.failedWith = msg
	return s.failErr
}

type stubUploader struct {
	uploadErr error
	gotBody   []byte
}

func (u *stubUploader) PutObject(_ context.Context, _, _ string, reader io.Reader, _ int64, _ minio.PutObjectOptions) (minio.UploadInfo, error) {
	b, _ := io.ReadAll(reader)
	u.gotBody = b
	if u.uploadErr != nil {
		return minio.UploadInfo{}, u.uploadErr
	}
	return minio.UploadInfo{Size: int64(len(b))}, nil
}

// earlyFailUploader 模拟对象存储在读取 pipe 前立即拒绝上传（例如桶不可用或鉴权失败）。
// 这与会先 io.ReadAll 的 stubUploader 不同，专门覆盖 #34 的生产卡死形态。
type earlyFailUploader struct {
	err error
}

func (u *earlyFailUploader) PutObject(_ context.Context, _, _ string, _ io.Reader, _ int64, _ minio.PutObjectOptions) (minio.UploadInfo, error) {
	return minio.UploadInfo{}, u.err
}

type stubTimezoneProvider struct {
	loc   *time.Location
	calls int
}

type exportMetaRow struct {
	dimension   string
	deviceCount int
	metricPaths []string
}

func (r *exportMetaRow) Scan(dest ...any) error {
	if len(dest) > 0 {
		*dest[0].(*string) = r.dimension
	}
	if len(dest) > 1 {
		*dest[1].(*int) = r.deviceCount
	}
	if len(dest) > 2 {
		*dest[2].(*[]string) = r.metricPaths
	}
	return nil
}

type recordedExportQuery struct {
	sql  string
	args []any
}

type recordingExportQuerier struct {
	row         pgx.Row
	queryRowSQL string
	queries     []recordedExportQuery
	results     []pgx.Rows
}

func (q *recordingExportQuerier) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (q *recordingExportQuerier) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	q.queries = append(q.queries, recordedExportQuery{sql: sql, args: args})
	if len(q.results) == 0 {
		return &adhocFakeRows{}, nil
	}
	rows := q.results[0]
	q.results = q.results[1:]
	return rows, nil
}

func (q *recordingExportQuerier) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	q.queryRowSQL = sql
	return q.row
}

func (s *stubTimezoneProvider) Location(_ context.Context) *time.Location {
	s.calls++
	return s.loc
}

// sliceSource 把固定批次的 ExportRow 当成 RowSource（测试用）。
type sliceSource struct {
	batches [][]ExportRow
	i       int
	err     error
}

func (s *sliceSource) Next(_ context.Context) ([]ExportRow, bool, error) {
	if s.err != nil {
		return nil, false, s.err
	}
	if s.i >= len(s.batches) {
		return nil, true, nil
	}
	b := s.batches[s.i]
	s.i++
	done := s.i >= len(s.batches)
	return b, done, nil
}

func newTestTask(src SourceType) *Task {
	return &Task{ID: uuid.New(), SourceType: src, Params: []byte(`{}`)}
}

// #198：adhoc/性能仪表盘导出的指标列全集直接取任务 meta.metricPaths，保序去重；
// 当前筛选零行也不能把配置指标列裁掉，且不再依赖结果表 DISTINCT 发现列。
func TestRunner_BuildSource_AdhocUsesConfiguredMetricColumns(t *testing.T) {
	for _, source := range []SourceType{SourcePMDashboard, SourceAdhocResult} {
		t.Run(string(source), func(t *testing.T) {
			taskID := uuid.New()
			productID := uuid.New()
			metricPaths := []string{" KGSM0102 ", "KGSM0102", "CGSM0001"}
			normalizedMetricPaths := []string{"KGSM0102", "CGSM0001"}
			params, err := json.Marshal(AdhocParams{
				TaskID:     taskID.String(),
				ProductIDs: []string{productID.String()},
				ObjectLDNs: []string{"DeviceGroup=11111111-1111-1111-1111-111111111111,Tech=lte"},
				Weekdays:   []int{1, 2},
				Hours:      []int{8, 9},
			})
			require.NoError(t, err)

			metaDB := &recordingExportQuerier{row: &exportMetaRow{
				dimension:   "product",
				deviceCount: 0,
				metricPaths: metricPaths,
			}}
			adhocDB := &recordingExportQuerier{results: []pgx.Rows{
				&adhocFakeRows{rows: [][]any{
					{"KGSM0102", "指标二"},
					{"CGSM0001", "计数器一"},
				}},
			}}
			runner := NewRunner(RunnerDeps{AdhocDB: adhocDB, TaskMetaDB: metaDB})

			src, cols, _, err := runner.buildSource(context.Background(), &Task{
				ID:         uuid.New(),
				SourceType: source,
				Params:     params,
			})
			require.NoError(t, err)

			assert.Contains(t, metaDB.queryRowSQL, "metric_paths")
			require.Len(t, cols, 2)
			assert.Equal(t, []string{"KGSM0102", "CGSM0001"}, []string{cols[0].Code, cols[1].Code})
			assert.Equal(t, []string{"kpi", "counter"}, []string{cols[0].Type, cols[1].Type})
			assert.Equal(t, []string{"指标二", "计数器一"}, []string{cols[0].Name, cols[1].Name})
			require.Len(t, adhocDB.queries, 1)
			assert.NotContains(t, adhocDB.queries[0].sql, "SELECT DISTINCT r.metric_path")
			assert.Contains(t, adhocDB.queries[0].sql, "perf_indicators_enb")
			adhocSrc := src.(*adhocSource)
			assert.Equal(t, normalizedMetricPaths, adhocSrc.metricPaths)
			assert.Equal(t, []uuid.UUID{productID}, adhocSrc.filter.ProductIDs)
			assert.Equal(t, []string{"DeviceGroup=11111111-1111-1111-1111-111111111111,Tech=lte"}, adhocSrc.filter.ObjectLDNs)
			assert.Equal(t, []int{1, 2}, adhocSrc.filter.Weekdays)
			assert.Equal(t, []int{8, 9}, adhocSrc.filter.Hours)

			_, _, err = src.Next(context.Background())
			require.NoError(t, err)
			require.Len(t, adhocDB.queries, 2)
			assert.Contains(t, adhocDB.queries[1].sql, "r.metric_path IN")
			assert.Contains(t, adhocDB.queries[1].args, "KGSM0102")
			assert.Contains(t, adhocDB.queries[1].args, "CGSM0001")
			assert.NotContains(t, adhocDB.queries[1].args, " KGSM0102 ")
		})
	}
}

func TestRunner_BuildSource_AdhocBlankMetricPathsFallsBackWithoutFilteringRows(t *testing.T) {
	for _, source := range []SourceType{SourcePMDashboard, SourceAdhocResult} {
		t.Run(string(source), func(t *testing.T) {
			taskID := uuid.New()
			params, err := json.Marshal(AdhocParams{TaskID: taskID.String()})
			require.NoError(t, err)
			metaDB := &recordingExportQuerier{row: &exportMetaRow{
				dimension:   "network",
				metricPaths: []string{"", "  "},
			}}
			adhocDB := &recordingExportQuerier{results: []pgx.Rows{
				&adhocFakeRows{rows: [][]any{{"K_STORED", "kpi"}}},
				&adhocFakeRows{rows: [][]any{{"K_STORED", "已存指标"}}},
				&adhocFakeRows{},
			}}
			runner := NewRunner(RunnerDeps{AdhocDB: adhocDB, TaskMetaDB: metaDB})

			src, cols, _, err := runner.buildSource(context.Background(), &Task{
				ID:         uuid.New(),
				SourceType: source,
				Params:     params,
			})
			require.NoError(t, err)
			require.Len(t, cols, 1)
			assert.Equal(t, "K_STORED", cols[0].Code)
			assert.Empty(t, src.(*adhocSource).metricPaths)

			_, _, err = src.Next(context.Background())
			require.NoError(t, err)
			require.Len(t, adhocDB.queries, 3)
			assert.Contains(t, adhocDB.queries[0].sql, "SELECT DISTINCT r.metric_path")
			assert.NotContains(t, adhocDB.queries[0].sql, "r.metric_path IN")
			assert.NotContains(t, adhocDB.queries[2].sql, "r.metric_path IN")
		})
	}
}

func TestRunner_BuildSource_UsesStoredEnglishLocaleWithoutRequestContext(t *testing.T) {
	taskID := uuid.New()
	metaDB := &recordingExportQuerier{row: &exportMetaRow{
		dimension:   "product",
		metricPaths: []string{"KGSM0143"},
	}}
	adhocDB := &recordingExportQuerier{results: []pgx.Rows{
		&adhocFakeRows{rows: [][]any{{"KGSM0143", "English KPI Name"}}},
	}}
	runner := NewRunner(RunnerDeps{AdhocDB: adhocDB, TaskMetaDB: metaDB})

	_, _, _, err := runner.buildSource(context.Background(), &Task{
		ID:         uuid.New(),
		SourceType: SourceAdhocResult,
		Params: []byte(fmt.Sprintf(
			`{"task_id":%q,"locale":"en-US"}`,
			taskID.String(),
		)),
	})
	require.NoError(t, err)
	require.Len(t, adhocDB.queries, 1)
	assert.Contains(t, adhocDB.queries[0].sql, "COALESCE(NULLIF(en_name, ''), cn_name)")
}

func TestRunner_BuildSource_KpiQueryDoesNotCreateSyntheticSkeletonRows(t *testing.T) {
	start := time.Date(2026, 7, 14, 7, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	params, err := json.Marshal(DashboardParams{
		Granularity: "hourly",
		Dimension:   "device",
		DeviceSNs:   []string{"SN1"},
		MetricPaths: []string{"K001"},
		StartTime:   start.Format(time.RFC3339),
		EndTime:     end.Format(time.RFC3339),
	})
	require.NoError(t, err)

	metricDB := &recordingExportQuerier{results: []pgx.Rows{
		&adhocFakeRows{}, // 指标名解析无命中，列名回退指标编号
	}}
	runner := NewRunner(RunnerDeps{
		MetricDB: metricDB,
		Aggr:     aggregator.New(metricDB, nil, nil),
	})

	src, _, _, err := runner.buildSource(context.Background(), &Task{
		ID:         uuid.New(),
		SourceType: SourceKpiQuery,
		Params:     params,
	})
	require.NoError(t, err)

	deviceSrc, ok := src.(*dashboardDeviceSource)
	require.True(t, ok)
	assert.Empty(t, deviceSrc.objectLDNs)
	require.NotEmpty(t, metricDB.queries)
	assert.NotContains(t, metricDB.queries[0].sql, "SELECT DISTINCT object_ldn")
}

func TestRunner_BuildSource_KpiQueryPreservesNonHourlySkeletonExport(t *testing.T) {
	start := time.Date(2026, 7, 14, 7, 0, 0, 0, time.UTC)
	end := start.Add(30 * time.Minute)
	params, err := json.Marshal(DashboardParams{
		Granularity: "15min",
		Dimension:   "device",
		DeviceSNs:   []string{"SN1"},
		MetricPaths: []string{"K001"},
		StartTime:   start.Format(time.RFC3339),
		EndTime:     end.Format(time.RFC3339),
	})
	require.NoError(t, err)

	metricDB := &recordingExportQuerier{results: []pgx.Rows{
		&adhocFakeRows{rows: [][]any{{"Cellid=1"}, {"Cellid=2"}}}, // DiscoverObjectLDNs
		&adhocFakeRows{}, // 指标名解析无命中，列名回退指标编号
	}}
	runner := NewRunner(RunnerDeps{
		MetricDB: metricDB,
		Aggr:     aggregator.New(metricDB, nil, nil),
	})

	src, _, _, err := runner.buildSource(context.Background(), &Task{
		ID:         uuid.New(),
		SourceType: SourceKpiQuery,
		Params:     params,
	})
	require.NoError(t, err)

	filled, ok := src.(*fillEmptySource)
	require.True(t, ok)
	assert.Equal(t, []string{"Cellid=1", "Cellid=2"}, filled.req.ObjectLDNs)

	deviceSrc, ok := filled.src.(*dashboardDeviceSource)
	require.True(t, ok)
	assert.Equal(t, []string{"Cellid=1", "Cellid=2"}, deviceSrc.objectLDNs)
	require.NotEmpty(t, metricDB.queries)
	assert.Contains(t, metricDB.queries[0].sql, "SELECT DISTINCT object_ldn")
	assert.NotContains(t, metricDB.queries[0].sql, "metric_path", "当前指标完全没数据时也要能发现对象全集")
}

func TestRunner_BuildSource_DeviceKPIExportUsesStoredResultSource(t *testing.T) {
	start := time.Date(2026, 7, 14, 7, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	params, err := json.Marshal(DashboardParams{
		Granularity: "hourly",
		Dimension:   "device",
		DeviceSNs:   []string{"SN1"},
		MetricPaths: []string{"K001"},
		StartTime:   start.Format(time.RFC3339),
		EndTime:     end.Format(time.RFC3339),
	})
	require.NoError(t, err)

	metricDB := &recordingExportQuerier{results: []pgx.Rows{
		&adhocFakeRows{}, // 指标名解析无命中，列名回退指标编号。
	}}
	runner := NewRunner(RunnerDeps{MetricDB: metricDB})

	src, _, _, err := runner.buildSource(context.Background(), &Task{
		ID:         uuid.New(),
		SourceType: SourceDashboard,
		Params:     params,
	})
	require.NoError(t, err)
	_, ok := src.(*dashboardDeviceSource)
	require.True(t, ok)
}

func TestRunner_BuildSource_MixedKpiCounterClearsSingleMetricTypeFilter(t *testing.T) {
	start := time.Date(2026, 7, 14, 7, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	params, err := json.Marshal(DashboardParams{
		Granularity: "hourly",
		Dimension:   "device",
		DeviceSNs:   []string{"SN1"},
		MetricPaths: []string{"KGSM0101", "CGSM0010001"},
		MetricType:  "kpi",
		StartTime:   start.Format(time.RFC3339),
		EndTime:     end.Format(time.RFC3339),
	})
	require.NoError(t, err)

	metricDB := &recordingExportQuerier{results: []pgx.Rows{
		&adhocFakeRows{},
	}}
	runner := NewRunner(RunnerDeps{MetricDB: metricDB})

	src, _, _, err := runner.buildSource(context.Background(), &Task{
		ID:         uuid.New(),
		SourceType: SourceDeviceView,
		Params:     params,
	})
	require.NoError(t, err)
	deviceSrc, ok := src.(*dashboardDeviceSource)
	require.True(t, ok)
	assert.Nil(t, deviceSrc.req.MetricType)
}

func TestRunner_BuildSource_DeviceDailyKPIExportUsesStoredOffsetSource(t *testing.T) {
	start := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	params, err := json.Marshal(DashboardParams{
		Granularity: "daily",
		Dimension:   "device",
		DeviceSNs:   []string{"SN1"},
		MetricPaths: []string{"KGSM0101"},
		StartTime:   start.Format(time.RFC3339),
		EndTime:     end.Format(time.RFC3339),
	})
	require.NoError(t, err)

	metricDB := &recordingExportQuerier{results: []pgx.Rows{
		&adhocFakeRows{},
	}}
	runner := NewRunner(RunnerDeps{MetricDB: metricDB})

	src, _, _, err := runner.buildSource(context.Background(), &Task{
		ID:         uuid.New(),
		SourceType: SourceDashboard,
		Params:     params,
	})
	require.NoError(t, err)
	_, ok := src.(*dashboardDeviceOffsetSource)
	require.True(t, ok)
}

func TestRunner_BuildSource_DeviceViewUsesMeasurementObjectOnlyLayout(t *testing.T) {
	start := time.Date(2026, 7, 14, 7, 0, 0, 0, time.UTC)
	end := start.Add(30 * time.Minute)
	params, err := json.Marshal(DashboardParams{
		Granularity: "15min",
		Dimension:   "device",
		DeviceSNs:   []string{"SN1"},
		MetricPaths: []string{"C001"},
		StartTime:   start.Format(time.RFC3339),
		EndTime:     end.Format(time.RFC3339),
	})
	require.NoError(t, err)

	metricDB := &recordingExportQuerier{results: []pgx.Rows{
		&adhocFakeRows{}, // 指标名解析无命中，列名回退指标编号。
	}}
	runner := NewRunner(RunnerDeps{MetricDB: metricDB})

	src, _, layout, err := runner.buildSource(context.Background(), &Task{
		ID:         uuid.New(),
		SourceType: SourceDeviceView,
		Params:     params,
	})
	require.NoError(t, err)

	_, ok := src.(*dashboardDeviceSource)
	require.True(t, ok)
	assert.False(t, layout.IncludeCell)
	assert.True(t, layout.IncludeMeasurementObject)
	assert.Equal(t, "设备 SN", layout.FirstColHeader)
}

func TestRunner_BuildSource_DeviceViewCSVIncludesMeasurementObjectOriginalLDN(t *testing.T) {
	start := time.Date(2026, 7, 14, 7, 0, 0, 0, time.UTC)
	end := start.Add(15 * time.Minute)
	params, err := json.Marshal(DashboardParams{
		Granularity: "15min",
		Dimension:   "device",
		DeviceSNs:   []string{"NR-SN"},
		MetricPaths: []string{"C001"},
		StartTime:   start.Format(time.RFC3339),
		EndTime:     end.Format(time.RFC3339),
	})
	require.NoError(t, err)

	metricDB := &recordingExportQuerier{results: []pgx.Rows{
		&adhocFakeRows{}, // 指标名解析无命中，列名回退指标编号。
	}}
	runner := NewRunner(RunnerDeps{MetricDB: metricDB})
	_, _, layout, err := runner.buildSource(context.Background(), &Task{
		ID:         uuid.New(),
		SourceType: SourceDeviceView,
		Params:     params,
	})
	require.NoError(t, err)

	objectLDN := "Type=Cell,Mode=SA,gNBID=123,NrCGI=46068123456"
	up := &stubUploader{}
	_, err = streamCSVToObject(
		context.Background(),
		up,
		"reports",
		"device-view.csv",
		&sliceSource{batches: [][]ExportRow{{
			{
				Device:     "NR-SN",
				CellPLMN:   objectLDN,
				Time:       start,
				StartTime:  start,
				EndTime:    end,
				MetricCode: "C001",
				Value:      42,
			},
		}}},
		[]WideColumn{{Code: "C001", Type: "counter", Name: "下行包数"}},
		layout,
		nil,
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"开始时间", "结束时间", "设备 SN", "测量对象", "下行包数"}, nthCSVRow(t, up.gotBody, 0))
	row := nthCSVRow(t, up.gotBody, 1)
	require.Len(t, row, 5)
	assert.Equal(t, objectLDN, row[3])
	assert.Equal(t, "42.00", row[4])
}

func TestRunner_BuildSource_DeviceViewEnglishCSVIncludesMeasurementObjectOriginalLDN(t *testing.T) {
	start := time.Date(2026, 7, 14, 7, 0, 0, 0, time.UTC)
	end := start.Add(15 * time.Minute)
	params := []byte(fmt.Sprintf(`{
		"granularity": "15min",
		"dimension": "device",
		"device_sns": ["NR-SN"],
		"metric_paths": ["C001"],
		"start_time": %q,
		"end_time": %q,
		"locale": "en-US"
	}`, start.Format(time.RFC3339), end.Format(time.RFC3339)))

	metricDB := &recordingExportQuerier{results: []pgx.Rows{
		&adhocFakeRows{}, // 指标名解析无命中，列名回退指标编号。
	}}
	runner := NewRunner(RunnerDeps{MetricDB: metricDB})
	_, _, layout, err := runner.buildSource(context.Background(), &Task{
		ID:         uuid.New(),
		SourceType: SourceDeviceView,
		Params:     params,
	})
	require.NoError(t, err)

	objectLDN := "Type=Cell,Mode=SA,gNBID=123,NrCGI=46068123456"
	up := &stubUploader{}
	_, err = streamCSVToObject(
		context.Background(),
		up,
		"reports",
		"device-view-en.csv",
		&sliceSource{batches: [][]ExportRow{{
			{
				Device:     "NR-SN",
				CellPLMN:   objectLDN,
				Time:       start,
				StartTime:  start,
				EndTime:    end,
				MetricCode: "C001",
				Value:      42,
			},
		}}},
		[]WideColumn{{Code: "C001", Type: "counter", Name: "Downlink Packets"}},
		layout,
		nil,
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"Start Time", "End Time", "Device SN", "Measurement Object", "Downlink Packets"}, nthCSVRow(t, up.gotBody, 0))
	row := nthCSVRow(t, up.gotBody, 1)
	require.Len(t, row, 5)
	assert.Equal(t, objectLDN, row[3])
	assert.Equal(t, "42.00", row[4])
}

// ── 预 running 守门：payload 坏 / 缺 task_id 直接返 error，不动任务 ─────────────

func TestRunner_JobType(t *testing.T) {
	r := NewRunner(RunnerDeps{Repo: &stubTaskRepo{}})
	assert.Equal(t, JobType, r.JobType())
}

func TestRunner_Run_MissingTaskID(t *testing.T) {
	r := NewRunner(RunnerDeps{Repo: &stubTaskRepo{}})
	_, err := r.Run(context.Background(), &asyncjob.Job{Payload: json.RawMessage(`{}`)})
	require.Error(t, err)
}

func TestRunner_Run_InvalidPayload(t *testing.T) {
	r := NewRunner(RunnerDeps{Repo: &stubTaskRepo{}})
	_, err := r.Run(context.Background(), &asyncjob.Job{Payload: json.RawMessage(`not-json`)})
	require.Error(t, err)
}

func TestRunner_Run_MarkRunningError_Propagates(t *testing.T) {
	repo := &stubTaskRepo{task: newTestTask(SourceDashboard), runErr: errors.New("not pending")}
	r := NewRunner(RunnerDeps{Repo: repo})
	payload, _ := BuildJobPayload(uuid.New())
	_, err := r.Run(context.Background(), &asyncjob.Job{Payload: payload})
	require.Error(t, err)
}

// ── 成功路径：running → 取数 → 上传 → succeeded 回填 ────────────────────────

func TestRunner_Run_Success(t *testing.T) {
	task := newTestTask(SourceDashboard)
	repo := &stubTaskRepo{task: task}
	up := &stubUploader{}
	r := NewRunner(RunnerDeps{Repo: repo, Uploader: up, Bucket: "reports"})
	// 注入 stub 源：2 个数据点，同设备同时间（零值）→ 横表摊成 1 行、2 指标列。
	r.buildSourceFn = func(_ context.Context, _ *Task) (RowSource, []WideColumn, csvLayout, error) {
		return &sliceSource{batches: [][]ExportRow{{
				{Device: "ABCDEF/SN1", MetricCode: "K001", Value: 1.5},
				{Device: "ABCDEF/SN1", MetricCode: "K002", Value: 2.5},
			}}},
			[]WideColumn{{Code: "K001", Type: "kpi", Name: "上行吞吐"}, {Code: "K002", Type: "kpi", Name: "下行吞吐"}},
			csvLayout{FirstColHeader: "设备", IncludeCell: true}, nil
	}

	payload, _ := BuildJobPayload(task.ID)
	out, err := r.Run(context.Background(), &asyncjob.Job{Payload: payload})
	require.NoError(t, err)
	assert.Equal(t, 1, repo.markedRun)
	require.NotNil(t, repo.succeeded)
	assert.Equal(t, "reports", repo.succeeded.bucket)
	assert.Equal(t, int64(1), repo.succeeded.rowN) // 横表：同行键 2 指标摊成 1 横行
	assert.Greater(t, repo.succeeded.fileSize, int64(0))
	assert.Equal(t, 0, repo.failedN)

	// 上传体头三字节为 UTF-8 BOM。
	require.GreaterOrEqual(t, len(up.gotBody), 3)
	assert.Equal(t, []byte{0xEF, 0xBB, 0xBF}, up.gotBody[:3])

	var res map[string]any
	require.NoError(t, json.Unmarshal(out, &res))
	assert.Equal(t, "succeeded", res["status"])
}

func TestRunner_Run_EarlyUploadFailureMarksTaskFailedWithoutHanging(t *testing.T) {
	task := newTestTask(SourceDashboard)
	repo := &stubTaskRepo{task: task}
	r := NewRunner(RunnerDeps{
		Repo:     repo,
		Uploader: &earlyFailUploader{err: errors.New("bucket unavailable")},
		Bucket:   "reports",
	})
	r.buildSourceFn = func(_ context.Context, _ *Task) (RowSource, []WideColumn, csvLayout, error) {
		return &sliceSource{batches: [][]ExportRow{{
			{Device: "ABCDEF/SN1", MetricCode: "K001", Value: 1.5},
		}}}, []WideColumn{{Code: "K001", Type: "kpi", Name: "K001"}}, csvLayout{FirstColHeader: "设备"}, nil
	}

	done := make(chan error, 1)
	go func() {
		payload, _ := BuildJobPayload(task.ID)
		_, err := r.Run(context.Background(), &asyncjob.Job{Payload: payload})
		done <- err
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
		assert.Equal(t, 1, repo.failedN)
		assert.Contains(t, repo.failedWith, "bucket unavailable")
	case <-time.After(250 * time.Millisecond):
		t.Fatal("export runner hung after uploader returned before reading the pipe")
	}
}

func TestRunner_Run_FormatsCSVTimesInTimezoneProviderLocation(t *testing.T) {
	task := newTestTask(SourceDashboard)
	repo := &stubTaskRepo{task: task}
	up := &stubUploader{}
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	tzp := &stubTimezoneProvider{loc: shanghai}
	r := NewRunner(RunnerDeps{Repo: repo, Uploader: up, Bucket: "reports", TimezoneProvider: tzp})
	start := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	r.buildSourceFn = func(_ context.Context, _ *Task) (RowSource, []WideColumn, csvLayout, error) {
		return &sliceSource{batches: [][]ExportRow{{
				{Device: "SN1", MetricCode: "K001", Time: start, StartTime: start, EndTime: start.Add(time.Hour), Value: 1},
			}}},
			[]WideColumn{{Code: "K001", Type: "kpi", Name: "上行吞吐"}},
			csvLayout{FirstColHeader: "设备", IncludeCell: true}, nil
	}

	payload, _ := BuildJobPayload(task.ID)
	_, err = r.Run(context.Background(), &asyncjob.Job{Payload: payload})
	require.NoError(t, err)

	row := nthCSVRow(t, up.gotBody, 1)
	assert.Equal(t, "2026-06-04 18:00:00", row[0])
	assert.Equal(t, "2026-06-04 19:00:00", row[1])
	assert.Equal(t, 1, tzp.calls)
}

func TestRunner_Run_NilTimezoneFallsBackToUTC(t *testing.T) {
	tests := []struct {
		name string
		deps RunnerDeps
	}{
		{name: "nil provider", deps: RunnerDeps{}},
		{name: "nil location", deps: RunnerDeps{TimezoneProvider: &stubTimezoneProvider{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := newTestTask(SourceDashboard)
			repo := &stubTaskRepo{task: task}
			up := &stubUploader{}
			tt.deps.Repo = repo
			tt.deps.Uploader = up
			tt.deps.Bucket = "reports"
			r := NewRunner(tt.deps)
			start := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
			r.buildSourceFn = func(_ context.Context, _ *Task) (RowSource, []WideColumn, csvLayout, error) {
				return &sliceSource{batches: [][]ExportRow{{
						{Device: "SN1", MetricCode: "K001", Time: start, StartTime: start, EndTime: start.Add(time.Hour), Value: 1},
					}}},
					[]WideColumn{{Code: "K001", Type: "kpi", Name: "上行吞吐"}},
					csvLayout{FirstColHeader: "设备", IncludeCell: true}, nil
			}

			payload, _ := BuildJobPayload(task.ID)
			_, err := r.Run(context.Background(), &asyncjob.Job{Payload: payload})
			require.NoError(t, err)

			row := nthCSVRow(t, up.gotBody, 1)
			assert.Equal(t, "2026-06-04 10:00:00", row[0])
			assert.Equal(t, "2026-06-04 11:00:00", row[1])
		})
	}
}

// ── 失败路径：取数报错 → MarkFailed + error 落库，job 不重试（返回 nil） ───────

func TestRunner_Run_GenerateError_MarksFailed(t *testing.T) {
	task := newTestTask(SourceDashboard)
	repo := &stubTaskRepo{task: task}
	r := NewRunner(RunnerDeps{Repo: repo, Uploader: &stubUploader{}, Bucket: "reports"})
	r.buildSourceFn = func(_ context.Context, _ *Task) (RowSource, []WideColumn, csvLayout, error) {
		return &sliceSource{err: errors.New("db read failed")}, nil, csvLayout{FirstColHeader: "设备", IncludeCell: true}, nil
	}

	payload, _ := BuildJobPayload(task.ID)
	out, err := r.Run(context.Background(), &asyncjob.Job{Payload: payload})
	require.NoError(t, err) // 框架视为已处理，不重试
	assert.Equal(t, 1, repo.failedN)
	assert.Contains(t, repo.failedWith, "db read failed")
	assert.Nil(t, repo.succeeded)

	var res map[string]any
	require.NoError(t, json.Unmarshal(out, &res))
	assert.Equal(t, "failed", res["status"])
}

// 上传失败也走 failed。
func TestRunner_Run_UploadError_MarksFailed(t *testing.T) {
	task := newTestTask(SourceDashboard)
	repo := &stubTaskRepo{task: task}
	up := &stubUploader{uploadErr: errors.New("minio down")}
	r := NewRunner(RunnerDeps{Repo: repo, Uploader: up, Bucket: "reports"})
	r.buildSourceFn = func(_ context.Context, _ *Task) (RowSource, []WideColumn, csvLayout, error) {
		return &sliceSource{batches: [][]ExportRow{{{Device: "d", MetricCode: "K1"}}}}, nil, csvLayout{FirstColHeader: "设备", IncludeCell: true}, nil
	}
	payload, _ := BuildJobPayload(task.ID)
	_, err := r.Run(context.Background(), &asyncjob.Job{Payload: payload})
	require.NoError(t, err)
	assert.Equal(t, 1, repo.failedN)
}

// 取数失败 + MarkFailed 也失败 → 返回 error 让框架重试。
func TestRunner_Run_MarkFailedAlsoFails_ReturnsError(t *testing.T) {
	task := newTestTask(SourceDashboard)
	repo := &stubTaskRepo{task: task, failErr: errors.New("db down")}
	r := NewRunner(RunnerDeps{Repo: repo, Uploader: &stubUploader{}, Bucket: "reports"})
	r.buildSourceFn = func(_ context.Context, _ *Task) (RowSource, []WideColumn, csvLayout, error) {
		return &sliceSource{err: errors.New("read error")}, nil, csvLayout{FirstColHeader: "设备", IncludeCell: true}, nil
	}
	payload, _ := BuildJobPayload(task.ID)
	_, err := r.Run(context.Background(), &asyncjob.Job{Payload: payload})
	require.Error(t, err)
}

// 无 uploader / 无 bucket → generate 报错 → MarkFailed。
func TestRunner_Run_NoUploader_MarksFailed(t *testing.T) {
	task := newTestTask(SourceDashboard)
	repo := &stubTaskRepo{task: task}
	r := NewRunner(RunnerDeps{Repo: repo, Bucket: "reports"}) // uploader nil
	r.buildSourceFn = func(_ context.Context, _ *Task) (RowSource, []WideColumn, csvLayout, error) {
		return &sliceSource{}, nil, csvLayout{FirstColHeader: "设备", IncludeCell: true}, nil
	}
	payload, _ := BuildJobPayload(task.ID)
	_, err := r.Run(context.Background(), &asyncjob.Job{Payload: payload})
	require.NoError(t, err)
	assert.Equal(t, 1, repo.failedN)
}
