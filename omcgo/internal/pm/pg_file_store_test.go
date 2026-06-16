package pm

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
)

// Verify PgPMFileStore implements PMFileStore interface at compile time.
var _ PMFileStore = (*PgPMFileStore)(nil)

func TestPMFileInfo_Fields(t *testing.T) {
	now := time.Now()
	parsedAt := now.Add(time.Minute)
	info := PMFileInfo{
		ID:           uuid.New(),
		DeviceID:     uuid.New(),
		DeviceSN:     "eNB001",
		Carrier:      "cmcc",
		Technology:   "lte",
		FileName:     "A20260311.1500+0800-1515+0800_eNB001.xml",
		FileSize:     4096,
		CollectTime:  now,
		MinioPath:    "pm-files/cmcc/lte/eNB001/2026-03-11/file.xml",
		Parsed:       true,
		ParsedAt:     &parsedAt,
		CounterCount: 42,
		CreatedAt:    now,
	}

	assert.NotEqual(t, uuid.Nil, info.ID)
	assert.Equal(t, "eNB001", info.DeviceSN)
	assert.Equal(t, "cmcc", info.Carrier)
	assert.Equal(t, "lte", info.Technology)
	assert.Equal(t, int64(4096), info.FileSize)
	assert.True(t, info.Parsed)
	assert.Equal(t, 42, info.CounterCount)
}

func TestPMFileFilter_Defaults(t *testing.T) {
	filter := PMFileFilter{
		ListRequest: model.DefaultListRequest(),
	}

	assert.Nil(t, filter.DeviceID)
	assert.Nil(t, filter.StartTime)
	assert.Nil(t, filter.EndTime)
	assert.Equal(t, 1, filter.Page)
	assert.Equal(t, 20, filter.PageSize)
}

func TestPMFileFilter_WithValues(t *testing.T) {
	devID := uuid.New()
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

	filter := PMFileFilter{
		DeviceID:    &devID,
		StartTime:   &start,
		EndTime:     &end,
		ListRequest: model.ListRequest{Page: 2, PageSize: 50},
	}

	assert.Equal(t, &devID, filter.DeviceID)
	assert.Equal(t, &start, filter.StartTime)
	assert.Equal(t, &end, filter.EndTime)
	assert.Equal(t, 2, filter.Page)
	assert.Equal(t, 50, filter.PageSize)
}

// mockPMFileStore is a test-only mock implementing PMFileStore.
type mockPMFileStore struct {
	saveFileFn        func(ctx context.Context, info *PMFileInfo) error
	getFileByIDFn     func(ctx context.Context, id uuid.UUID) (*PMFileInfo, error)
	listFilesFn       func(ctx context.Context, filter PMFileFilter) (*model.ListResponse[PMFileInfo], error)
	updateFileParsedFn func(ctx context.Context, id uuid.UUID, counterCount int) error
}

func (m *mockPMFileStore) SaveFile(ctx context.Context, info *PMFileInfo) error {
	if m.saveFileFn != nil {
		return m.saveFileFn(ctx, info)
	}
	info.ID = uuid.New()
	return nil
}
func (m *mockPMFileStore) GetFileByID(ctx context.Context, id uuid.UUID) (*PMFileInfo, error) {
	if m.getFileByIDFn != nil {
		return m.getFileByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockPMFileStore) ListFiles(ctx context.Context, filter PMFileFilter) (*model.ListResponse[PMFileInfo], error) {
	if m.listFilesFn != nil {
		return m.listFilesFn(ctx, filter)
	}
	return model.NewListResponse([]PMFileInfo{}, 0, 1, 20), nil
}
func (m *mockPMFileStore) UpdateFileParsed(ctx context.Context, id uuid.UUID, counterCount int) error {
	if m.updateFileParsedFn != nil {
		return m.updateFileParsedFn(ctx, id, counterCount)
	}
	return nil
}

func (m *mockPMFileStore) ListUncompressed(ctx context.Context, olderThan time.Time, limit int) ([]string, error) {
	return nil, nil
}

func (m *mockPMFileStore) MarkCompressed(ctx context.Context, renames map[string]string) error {
	return nil
}

func TestMockPMFileStore_SaveAndRetrieve(t *testing.T) {
	store := make(map[uuid.UUID]*PMFileInfo)

	mock := &mockPMFileStore{
		saveFileFn: func(_ context.Context, info *PMFileInfo) error {
			info.ID = uuid.New()
			info.CreatedAt = time.Now()
			store[info.ID] = info
			return nil
		},
		getFileByIDFn: func(_ context.Context, id uuid.UUID) (*PMFileInfo, error) {
			if f, ok := store[id]; ok {
				return f, nil
			}
			return nil, nil
		},
		updateFileParsedFn: func(_ context.Context, id uuid.UUID, counterCount int) error {
			if f, ok := store[id]; ok {
				now := time.Now()
				f.Parsed = true
				f.ParsedAt = &now
				f.CounterCount = counterCount
			}
			return nil
		},
	}

	ctx := context.Background()

	// Save
	info := &PMFileInfo{
		DeviceID:   uuid.New(),
		DeviceSN:   "eNB001",
		Carrier:    "cmcc",
		Technology: "lte",
		FileName:   "test.xml",
		MinioPath:  "pm-files/test.xml",
	}
	err := mock.SaveFile(ctx, info)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, info.ID)

	// Retrieve
	got, err := mock.GetFileByID(ctx, info.ID)
	assert.NoError(t, err)
	assert.Equal(t, info.DeviceSN, got.DeviceSN)
	assert.False(t, got.Parsed)

	// Update parsed
	err = mock.UpdateFileParsed(ctx, info.ID, 100)
	assert.NoError(t, err)

	got, _ = mock.GetFileByID(ctx, info.ID)
	assert.True(t, got.Parsed)
	assert.Equal(t, 100, got.CounterCount)
	assert.NotNil(t, got.ParsedAt)
}
