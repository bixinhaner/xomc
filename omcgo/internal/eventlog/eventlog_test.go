package eventlog

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/device"
)

// --- Filter 分页边界（纯函数）---

func TestFilterOffset(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		pageSize int
		want     int
	}{
		{name: "第一页 offset 0", page: 1, pageSize: 20, want: 0},
		{name: "page<=0 视为首页", page: 0, pageSize: 20, want: 0},
		{name: "负页号视为首页", page: -5, pageSize: 20, want: 0},
		{name: "第三页", page: 3, pageSize: 20, want: 40},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := Filter{Page: tc.page, PageSize: tc.pageSize}
			assert.Equal(t, tc.want, f.Offset())
		})
	}
}

func TestFilterLimit(t *testing.T) {
	tests := []struct {
		name     string
		pageSize int
		want     int
	}{
		{name: "默认 20", pageSize: 0, want: 20},
		{name: "负值回退默认", pageSize: -1, want: 20},
		{name: "显式 50", pageSize: 50, want: 50},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := Filter{PageSize: tc.pageSize}
			assert.Equal(t, tc.want, f.Limit())
		})
	}
}

// --- 内存 fake Repository（无 DB），function-field 模式注入失败路径 ---

type fakeRepo struct {
	created      []*EventLog
	createErr    error
	listErr      error
	getByIDErr   error
	statErr      error
	listResult   []*EventLog
	listTotal    int64
	getByIDValue *EventLog
	statResult   []*DeviceRebootStat
}

func (f *fakeRepo) Create(_ context.Context, e *EventLog) error {
	if f.createErr != nil {
		return f.createErr
	}
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	f.created = append(f.created, e)
	return nil
}

func (f *fakeRepo) List(_ context.Context, _ Filter) ([]*EventLog, int64, error) {
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
	return f.listResult, f.listTotal, nil
}

func (f *fakeRepo) GetByID(_ context.Context, _ uuid.UUID) (*EventLog, error) {
	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}
	return f.getByIDValue, nil
}

func (f *fakeRepo) StatByDevice(_ context.Context, _ Filter) ([]*DeviceRebootStat, error) {
	if f.statErr != nil {
		return nil, f.statErr
	}
	return f.statResult, nil
}

func newService(repo Repository) *Service {
	return NewService(repo, zap.NewNop())
}

// --- RecordBootEvent 成功路径：落库内容契约 ---

func TestRecordBootEvent_Success(t *testing.T) {
	repo := &fakeRepo{}
	svc := newService(repo)

	when := time.Date(2026, 6, 10, 8, 0, 0, 0, time.UTC)
	devID := uuid.New()
	err := svc.RecordBootEvent(context.Background(), device.BootEventSnapshot{
		DeviceID:            devID,
		DeviceSN:            "SN-001",
		DeviceName:          "基站A",
		BootCount:           3,
		RuntimeBeforeReboot: 3600,
		Events:              []string{"1 BOOT"},
		OccurredAt:          when,
	})
	require.NoError(t, err)
	require.Len(t, repo.created, 1)

	e := repo.created[0]
	assert.Equal(t, "SN-001", e.DeviceSN)
	assert.Equal(t, EventTypeBoot, e.EventType)
	assert.Equal(t, EventLevelInfo, e.EventLevel)
	require.NotNil(t, e.DeviceID)
	assert.Equal(t, devID, *e.DeviceID)
	assert.Equal(t, when, e.OccurredAt)

	// event_data JSONB 必须编码 boot_count / events / runtime。
	var data map[string]any
	require.NoError(t, json.Unmarshal(e.EventData, &data))
	assert.EqualValues(t, 3, data["boot_count"])
	assert.EqualValues(t, 3600, data["runtime_before_reboot"])
}

func TestRecordBootEvent_ZeroTimeDefaultsToNow(t *testing.T) {
	repo := &fakeRepo{}
	svc := newService(repo)

	before := time.Now()
	err := svc.RecordBootEvent(context.Background(), device.BootEventSnapshot{
		DeviceSN: "SN-002",
		// OccurredAt 留零值 → service 用 time.Now() 兜底。
	})
	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	assert.False(t, repo.created[0].OccurredAt.IsZero())
	assert.False(t, repo.created[0].OccurredAt.Before(before))
}

// --- RecordBootEvent 失败路径：repo.Create 报错应被 wrap 并上抛 ---

func TestRecordBootEvent_RepoError(t *testing.T) {
	sentinel := errors.New("db down")
	svc := newService(&fakeRepo{createErr: sentinel})

	err := svc.RecordBootEvent(context.Background(), device.BootEventSnapshot{DeviceSN: "SN-003"})
	require.Error(t, err)
	assert.ErrorIs(t, err, sentinel, "底层错误应可通过 errors.Is 追溯")
	assert.Contains(t, err.Error(), "create event log")
}

// --- 只读查询的成功与失败转发 ---

func TestList(t *testing.T) {
	want := []*EventLog{{DeviceSN: "SN-001"}}
	svc := newService(&fakeRepo{listResult: want, listTotal: 1})
	got, total, err := svc.List(context.Background(), Filter{})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	assert.Equal(t, want, got)
}

func TestList_Error(t *testing.T) {
	sentinel := errors.New("query failed")
	svc := newService(&fakeRepo{listErr: sentinel})
	_, _, err := svc.List(context.Background(), Filter{})
	assert.ErrorIs(t, err, sentinel)
}

func TestGetByID_Error(t *testing.T) {
	sentinel := errors.New("not found")
	svc := newService(&fakeRepo{getByIDErr: sentinel})
	_, err := svc.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, sentinel)
}

func TestStatByDevice_Error(t *testing.T) {
	sentinel := errors.New("agg failed")
	svc := newService(&fakeRepo{statErr: sentinel})
	_, err := svc.StatByDevice(context.Background(), Filter{})
	assert.ErrorIs(t, err, sentinel)
}
