package stationlog

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
)

func newEventPayload(p LogFileReceivedPayload) event.Event {
	evt, err := event.NewEvent("log.file.received", p)
	if err != nil {
		panic(err)
	}
	return evt
}

// memRepo 是内存版 Repository，单测专用。
// 通过实现 Repository + FaultExtraRepository 两个接口，能完整覆盖
// stationlog.Service 在故障日志路径上的两种 detected/file_received 状态切换。
type memRepo struct {
	mu      sync.Mutex
	rows    []*LogFile
	logType LogType
	isFault bool
}

func newMemFaultRepo() *memRepo  { return &memRepo{logType: LogTypeFault, isFault: true} }
func newMemRunRepo() *memRepo    { return &memRepo{logType: LogTypeRunning, isFault: false} }

func (m *memRepo) Create(_ context.Context, f *LogFile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	now := time.Now()
	f.CreatedAt = now
	f.UpdatedAt = now
	if f.CollectedAt.IsZero() {
		f.CollectedAt = now
	}
	if m.isFault {
		if f.RecordStatus == "" {
			if f.FileName == "" {
				f.RecordStatus = FaultRecordStatusDetected
			} else {
				f.RecordStatus = FaultRecordStatusFileReceived
			}
		}
		if f.ManualCollectionStatus == "" {
			f.ManualCollectionStatus = ManualCollectionIdle
		}
	}
	f.LogType = m.logType
	clone := *f
	m.rows = append(m.rows, &clone)
	return nil
}

func (m *memRepo) GetByID(_ context.Context, id uuid.UUID) (*LogFile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if r.ID == id {
			clone := *r
			return &clone, nil
		}
	}
	return nil, nil
}

func (m *memRepo) List(_ context.Context, filter LogFileFilter) ([]*LogFile, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*LogFile
	for _, r := range m.rows {
		if r.IsDeleted {
			continue
		}
		if filter.DeviceSN != "" && r.DeviceSN != filter.DeviceSN {
			continue
		}
		if filter.RecordStatus != "" && r.RecordStatus != filter.RecordStatus {
			continue
		}
		clone := *r
		out = append(out, &clone)
	}
	return out, int64(len(out)), nil
}

func (m *memRepo) MarkDeleted(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if r.ID == id {
			r.IsDeleted = true
			r.UpdatedAt = time.Now()
			return nil
		}
	}
	return nil
}

func (m *memRepo) Count(_ context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var n int64
	for _, r := range m.rows {
		if !r.IsDeleted {
			n++
		}
	}
	return n, nil
}

func (m *memRepo) ListOldest(_ context.Context, limit int) ([]*LogFile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*LogFile
	for _, r := range m.rows {
		if r.IsDeleted {
			continue
		}
		if m.isFault && r.RecordStatus != FaultRecordStatusFileReceived {
			continue
		}
		clone := *r
		out = append(out, &clone)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (m *memRepo) LatestByDevice(_ context.Context, deviceID uuid.UUID) (*LogFile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.rows) - 1; i >= 0; i-- {
		r := m.rows[i]
		if r.IsDeleted || r.DeviceID == nil || *r.DeviceID != deviceID {
			continue
		}
		clone := *r
		return &clone, nil
	}
	return nil, nil
}

func (m *memRepo) UpdateFile(_ context.Context, id uuid.UUID, fileName, objectPath, bucket string, size int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if r.ID == id {
			r.FileName = fileName
			r.ObjectPath = objectPath
			r.Bucket = bucket
			r.FileSize = size
			r.RecordStatus = FaultRecordStatusFileReceived
			r.UpdatedAt = time.Now()
			r.CollectedAt = time.Now()
			return nil
		}
	}
	return nil
}

func (m *memRepo) LatestDetectedByDeviceSN(_ context.Context, sn string) (*LogFile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.rows) - 1; i >= 0; i-- {
		r := m.rows[i]
		if r.IsDeleted || r.DeviceSN != sn || r.RecordStatus != FaultRecordStatusDetected {
			continue
		}
		clone := *r
		return &clone, nil
	}
	return nil, nil
}

type stubDeviceLookup struct{}

func (stubDeviceLookup) GetBySerialNumber(_ context.Context, _ string) (*model.Device, error) {
	return nil, nil
}

// newTestService 不接 MinIO；HandleLogFileReceived 的 fault 路径会走 detected
// 补全分支，不需要落 MinIO；运行日志路径写 repository 即可。
func newTestService() (*Service, *memRepo, *memRepo) {
	runRepo := newMemRunRepo()
	faultRepo := newMemFaultRepo()
	svc := NewService(runRepo, faultRepo, stubDeviceLookup{}, nil, appconfig.BucketConfig{}, zap.NewNop())
	return svc, runRepo, faultRepo
}

func TestRecordAbnormalReboot_WritesDetectedRow(t *testing.T) {
	svc, _, faultRepo := newTestService()

	snap := device.AbnormalRebootSnapshot{
		DeviceID:            uuid.New(),
		DeviceSN:            "SN-001",
		DeviceName:          "Beijing-eNB-0001",
		DeviceType:          "eNB",
		IsGNB:               false,
		OperateIP:           "10.1.0.101",
		SoftwareVersion:     "V1.3.0",
		HaltMainReason:      "halt_reboot",
		HaltDetailReason:    "watchdog_timeout",
		RuntimeBeforeReboot: 3600,
		DetectedAt:          time.Now(),
	}

	require.NoError(t, svc.RecordAbnormalReboot(context.Background(), snap))

	require.Len(t, faultRepo.rows, 1)
	row := faultRepo.rows[0]
	assert.Equal(t, FaultRecordStatusDetected, row.RecordStatus)
	assert.Equal(t, ManualCollectionIdle, row.ManualCollectionStatus)
	assert.Equal(t, "halt_reboot", row.FaultReason)
	assert.Equal(t, "watchdog_timeout", row.FaultDetail)
	assert.Equal(t, "Beijing-eNB-0001", row.DeviceName)
	assert.Equal(t, "eNB", row.DeviceType)
	assert.False(t, row.IsGNB)
	assert.Equal(t, int64(3600), row.RuntimeBeforeReboot)
	assert.Empty(t, row.FileName)
	assert.Empty(t, row.ObjectPath)
}

func TestRecordAbnormalReboot_AllowsRepeatRecords(t *testing.T) {
	// 同设备短时间连续异常重启应分别落库，靠 CollectedAt 区分；
	// 去重应留给上游告警的滑动窗口，不在持久化层做。
	svc, _, faultRepo := newTestService()
	snap := device.AbnormalRebootSnapshot{
		DeviceID:       uuid.New(),
		DeviceSN:       "SN-DUP",
		HaltMainReason: "halt_reboot",
		DetectedAt:     time.Now(),
	}
	require.NoError(t, svc.RecordAbnormalReboot(context.Background(), snap))
	require.NoError(t, svc.RecordAbnormalReboot(context.Background(), snap))
	assert.Len(t, faultRepo.rows, 2)
}

func TestHandleLogFileReceived_PromotesDetectedRecord(t *testing.T) {
	// 文件到达时优先把已有 detected 占位记录推进到 file_received，避免 UI 重复条目。
	svc, _, faultRepo := newTestService()

	// 先写一条 detected 占位
	snap := device.AbnormalRebootSnapshot{
		DeviceID:       uuid.New(),
		DeviceSN:       "SN-PROMOTE",
		HaltMainReason: "halt_reboot",
		DetectedAt:     time.Now(),
	}
	require.NoError(t, svc.RecordAbnormalReboot(context.Background(), snap))
	require.Len(t, faultRepo.rows, 1)

	// 文件到达事件
	payloadRaw := newEventPayload(LogFileReceivedPayload{
		DeviceSN:   "SN-PROMOTE",
		FileType:   "8",
		FileName:   "abnormalLog_SN-PROMOTE.tar.gz",
		ObjectPath: "fault/2026/05/21/abnormalLog_SN-PROMOTE.tar.gz",
		Bucket:     "logs",
		FileSize:   1024,
	})

	require.NoError(t, svc.HandleLogFileReceived(context.Background(), payloadRaw))

	// 仍然只有一行，但已经升级到 file_received
	require.Len(t, faultRepo.rows, 1)
	row := faultRepo.rows[0]
	assert.Equal(t, FaultRecordStatusFileReceived, row.RecordStatus)
	assert.Equal(t, "abnormalLog_SN-PROMOTE.tar.gz", row.FileName)
	assert.Equal(t, int64(1024), row.FileSize)
}

func TestHandleLogFileReceived_InsertsWhenNoDetectedRecord(t *testing.T) {
	// 直接到达的文件（无前置 detected 占位）应按旧路径 INSERT 一行 file_received。
	svc, _, faultRepo := newTestService()

	payloadRaw := newEventPayload(LogFileReceivedPayload{
		DeviceSN:   "SN-ORPHAN",
		FileType:   "8",
		FileName:   "abnormalLog_SN-ORPHAN.tar.gz",
		ObjectPath: "fault/2026/05/21/abnormalLog_SN-ORPHAN.tar.gz",
		Bucket:     "logs",
		FileSize:   2048,
	})

	require.NoError(t, svc.HandleLogFileReceived(context.Background(), payloadRaw))

	require.Len(t, faultRepo.rows, 1)
	row := faultRepo.rows[0]
	assert.Equal(t, FaultRecordStatusFileReceived, row.RecordStatus)
	assert.Equal(t, "abnormalLog_SN-ORPHAN.tar.gz", row.FileName)
}
