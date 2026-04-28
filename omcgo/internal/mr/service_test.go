package mr

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/mr/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestService_IngestParsedFile_Success 验证 IngestParsedFile 顺序调用
// SaveFile → BatchInsertRecords → UpdateFileParsed 三个 store 方法。
func TestService_IngestParsedFile_Success(t *testing.T) {
	t.Parallel()

	fileID := uuid.New()
	deviceID := uuid.New()

	var saveCalled, batchCalled, updateCalled bool
	var seenRecordCount int
	var seenMRType string

	store := &mockMRStore{
		saveFileFn: func(_ context.Context, f *MRFileInfo) error {
			saveCalled = true
			assert.Equal(t, fileID, f.ID)
			return nil
		},
		batchInsertRecordsFn: func(_ context.Context, fid, did uuid.UUID, mrType string, recs []parser.MRRecord) error {
			batchCalled = true
			assert.Equal(t, fileID, fid)
			assert.Equal(t, deviceID, did)
			seenMRType = mrType
			assert.Len(t, recs, 2)
			return nil
		},
		updateFileParsedFn: func(_ context.Context, fid uuid.UUID, cnt int) error {
			updateCalled = true
			assert.Equal(t, fileID, fid)
			seenRecordCount = cnt
			return nil
		},
	}

	svc := NewService(store, &mockIndicatorRepo{}, nil)

	file := &MRFileInfo{
		ID:          fileID,
		DeviceID:    deviceID,
		MRType:      "MRO",
		FileName:    "mro.xml.gz",
		CollectTime: time.Now(),
	}
	records := []parser.MRRecord{{}, {}}

	err := svc.IngestParsedFile(context.Background(), file, records)
	require.NoError(t, err)
	assert.True(t, saveCalled, "SaveFile should be called")
	assert.True(t, batchCalled, "BatchInsertRecords should be called")
	assert.True(t, updateCalled, "UpdateFileParsed should be called")
	assert.Equal(t, "MRO", seenMRType)
	assert.Equal(t, 2, seenRecordCount)
}

// TestService_IngestParsedFile_NoRecordsSkipsBatchInsert 验证：
// 当 records 为空时，跳过 BatchInsertRecords 但仍写入 SaveFile + UpdateFileParsed(0)。
func TestService_IngestParsedFile_NoRecordsSkipsBatchInsert(t *testing.T) {
	t.Parallel()

	var saveCalled, batchCalled, updateCalled bool
	var seenRecordCount int

	store := &mockMRStore{
		saveFileFn:  func(_ context.Context, _ *MRFileInfo) error { saveCalled = true; return nil },
		batchInsertRecordsFn: func(_ context.Context, _, _ uuid.UUID, _ string, _ []parser.MRRecord) error {
			batchCalled = true
			return nil
		},
		updateFileParsedFn: func(_ context.Context, _ uuid.UUID, cnt int) error {
			updateCalled = true
			seenRecordCount = cnt
			return nil
		},
	}

	svc := NewService(store, &mockIndicatorRepo{}, nil)

	file := &MRFileInfo{
		ID:       uuid.New(),
		DeviceID: uuid.New(),
		MRType:   "MRS",
	}

	err := svc.IngestParsedFile(context.Background(), file, nil)
	require.NoError(t, err)
	assert.True(t, saveCalled)
	assert.False(t, batchCalled, "BatchInsertRecords should NOT be called when records is empty")
	assert.True(t, updateCalled)
	assert.Equal(t, 0, seenRecordCount)
}

// TestService_IngestParsedFile_NilFile 验证：file 为 nil 直接返回错误，不调用 store。
func TestService_IngestParsedFile_NilFile(t *testing.T) {
	t.Parallel()

	store := &mockMRStore{
		saveFileFn: func(_ context.Context, _ *MRFileInfo) error {
			t.Fatal("SaveFile must not be called for nil file")
			return nil
		},
	}
	svc := NewService(store, &mockIndicatorRepo{}, nil)

	err := svc.IngestParsedFile(context.Background(), nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file is nil")
}

// TestService_IngestParsedFile_SaveFails 验证：SaveFile 失败 → 不再调用 BatchInsert/Update，
// 错误带上下文包装。
func TestService_IngestParsedFile_SaveFails(t *testing.T) {
	t.Parallel()

	saveErr := errors.New("disk full")
	var batchCalled bool
	store := &mockMRStore{
		saveFileFn: func(_ context.Context, _ *MRFileInfo) error { return saveErr },
		batchInsertRecordsFn: func(_ context.Context, _, _ uuid.UUID, _ string, _ []parser.MRRecord) error {
			batchCalled = true
			return nil
		},
	}
	svc := NewService(store, &mockIndicatorRepo{}, nil)

	file := &MRFileInfo{ID: uuid.New(), DeviceID: uuid.New()}
	err := svc.IngestParsedFile(context.Background(), file, []parser.MRRecord{{}})
	require.Error(t, err)
	assert.ErrorIs(t, err, saveErr, "should wrap original save error")
	assert.False(t, batchCalled, "BatchInsertRecords should NOT be called after SaveFile failure")
}

// TestService_ListFilesByDevice 验证 store.ListFiles 接收正确的 device 过滤。
func TestService_ListFilesByDevice(t *testing.T) {
	t.Parallel()

	deviceID := uuid.New()
	expected := &model.ListResponse[MRFileInfo]{
		Items:      []MRFileInfo{{ID: uuid.New(), DeviceID: deviceID}},
		Total:      1,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}

	store := &mockMRStore{
		listFilesFn: func(_ context.Context, f MRFileFilter) (*model.ListResponse[MRFileInfo], error) {
			require.NotNil(t, f.DeviceID)
			assert.Equal(t, deviceID, *f.DeviceID)
			assert.Equal(t, 1, f.Page)
			assert.Equal(t, 50, f.PageSize)
			return expected, nil
		},
	}
	svc := NewService(store, &mockIndicatorRepo{}, nil)

	resp, err := svc.ListFilesByDevice(context.Background(), &deviceID, model.ListRequest{Page: 1, PageSize: 50})
	require.NoError(t, err)
	assert.Same(t, expected, resp)
}

// TestService_QueryRecordsByDevice 验证 store.QueryRecords 接收正确的 device 过滤。
func TestService_QueryRecordsByDevice(t *testing.T) {
	t.Parallel()

	deviceID := uuid.New()
	expected := &model.ListResponse[MRRecordEntry]{
		Items:      []MRRecordEntry{{DeviceID: deviceID, CellID: "C1", MRType: "MRO"}},
		Total:      1,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}
	store := &mockMRStore{
		queryRecordsFn: func(_ context.Context, f MRRecordFilter) (*model.ListResponse[MRRecordEntry], error) {
			require.NotNil(t, f.DeviceID)
			assert.Equal(t, deviceID, *f.DeviceID)
			return expected, nil
		},
	}
	svc := NewService(store, &mockIndicatorRepo{}, nil)

	resp, err := svc.QueryRecordsByDevice(context.Background(), &deviceID, model.ListRequest{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Same(t, expected, resp)
}

// TestService_QueryRecordsByDevice_NoFilter 验证 deviceID = nil 时不过滤设备。
func TestService_QueryRecordsByDevice_NoFilter(t *testing.T) {
	t.Parallel()

	store := &mockMRStore{
		queryRecordsFn: func(_ context.Context, f MRRecordFilter) (*model.ListResponse[MRRecordEntry], error) {
			assert.Nil(t, f.DeviceID, "device filter should be nil when caller passes nil")
			return &model.ListResponse[MRRecordEntry]{Items: []MRRecordEntry{}, Page: 1, PageSize: 20}, nil
		},
	}
	svc := NewService(store, &mockIndicatorRepo{}, nil)

	resp, err := svc.QueryRecordsByDevice(context.Background(), nil, model.ListRequest{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

// TestService_GetIndicatorStats_Success 验证统计基于 indicator 的值域计算。
func TestService_GetIndicatorStats_Success(t *testing.T) {
	t.Parallel()

	minVal := -140.0
	maxVal := -44.0
	indicator := &MRIndicator{
		IndicatorCode: "MR.RSRP",
		ValueRangeMin: &minVal,
		ValueRangeMax: &maxVal,
	}
	indRepo := &mockIndicatorRepo{
		getByCodeFn: func(_ context.Context, code string) (*MRIndicator, error) {
			assert.Equal(t, "MR.RSRP", code)
			return indicator, nil
		},
	}
	svc := NewService(&mockMRStore{}, indRepo, nil)

	stats, err := svc.GetIndicatorStats(context.Background(), "MR.RSRP")
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, "MR.RSRP", stats.IndicatorCode)
	assert.InDelta(t, -92.0, stats.Avg, 0.001)
	assert.InDelta(t, -140.0, stats.Min, 0.001)
	assert.InDelta(t, -44.0, stats.Max, 0.001)
	assert.InDelta(t, -92.0, stats.P50, 0.001)
	assert.InDelta(t, -39.6, stats.P95, 0.001) // -44 * 0.9
	assert.Equal(t, int64(0), stats.SampleCount)
}

// TestService_GetIndicatorStats_NotFound 验证 indicator 不存在时返回 ErrIndicatorNotFound。
func TestService_GetIndicatorStats_NotFound(t *testing.T) {
	t.Parallel()

	indRepo := &mockIndicatorRepo{
		getByCodeFn: func(_ context.Context, _ string) (*MRIndicator, error) {
			return nil, nil
		},
	}
	svc := NewService(&mockMRStore{}, indRepo, nil)

	stats, err := svc.GetIndicatorStats(context.Background(), "MR.UNKNOWN")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrIndicatorNotFound)
	assert.Nil(t, stats)
}

// TestService_GetIndicatorStats_EmptyCode 验证空 code 直接返回错误，不查仓储。
func TestService_GetIndicatorStats_EmptyCode(t *testing.T) {
	t.Parallel()

	indRepo := &mockIndicatorRepo{
		getByCodeFn: func(_ context.Context, _ string) (*MRIndicator, error) {
			t.Fatal("repo.GetByCode must not be called for empty code")
			return nil, nil
		},
	}
	svc := NewService(&mockMRStore{}, indRepo, nil)

	_, err := svc.GetIndicatorStats(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "code is empty")
}

// TestService_GetIndicatorStats_RepoError 验证仓储错误被包装上下文返回。
func TestService_GetIndicatorStats_RepoError(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("db unavailable")
	indRepo := &mockIndicatorRepo{
		getByCodeFn: func(_ context.Context, _ string) (*MRIndicator, error) {
			return nil, repoErr
		},
	}
	svc := NewService(&mockMRStore{}, indRepo, nil)

	_, err := svc.GetIndicatorStats(context.Background(), "MR.RSRP")
	require.Error(t, err)
	assert.ErrorIs(t, err, repoErr)
	assert.NotErrorIs(t, err, ErrIndicatorNotFound)
}
