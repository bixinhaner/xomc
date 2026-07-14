package stationlog

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
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

func newMemFaultRepo() *memRepo { return &memRepo{logType: LogTypeFault, isFault: true} }
func newMemRunRepo() *memRepo   { return &memRepo{logType: LogTypeRunning, isFault: false} }

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
		if r.IsDeleted {
			continue
		}
		if m.isFault && r.RecordStatus != FaultRecordStatusFileReceived {
			continue
		}
		n++
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

func (m *memRepo) ListExpired(_ context.Context, cutoff time.Time, limit int) ([]*LogFile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*LogFile
	for _, r := range m.rows {
		if r.IsDeleted || !r.CreatedAt.Before(cutoff) {
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

// CountByDevice 统计指定设备未删除记录数（#798 每设备配额单测支持）。故障日志表口径
// 与 ListOldest 一致：只算 file_received（detected 占位不计入配额）。
func (m *memRepo) CountByDevice(_ context.Context, deviceID uuid.UUID) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var n int64
	for _, r := range m.rows {
		if r.IsDeleted || r.DeviceID == nil || *r.DeviceID != deviceID {
			continue
		}
		if m.isFault && r.RecordStatus != FaultRecordStatusFileReceived {
			continue
		}
		n++
	}
	return n, nil
}

// ListOldestByDevice 按采集时间升序返回指定设备未删除记录（#798 每设备配额单测支持）。
func (m *memRepo) ListOldestByDevice(_ context.Context, deviceID uuid.UUID, limit int) ([]*LogFile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var candidates []*LogFile
	for _, r := range m.rows {
		if r.IsDeleted || r.DeviceID == nil || *r.DeviceID != deviceID {
			continue
		}
		if m.isFault && r.RecordStatus != FaultRecordStatusFileReceived {
			continue
		}
		clone := *r
		candidates = append(candidates, &clone)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].CollectedAt.Before(candidates[j].CollectedAt) })
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates, nil
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

// fakeMinioClient 是单测用的 minioObjectClient 替身：不真连对象存储，只记录调用，
// 避免 nil *minio.Client 在触发真实删除路径时空指针崩溃。
type fakeMinioClient struct {
	mu      sync.Mutex
	removed []string
}

func (f *fakeMinioClient) RemoveObject(_ context.Context, bucket, object string, _ minio.RemoveObjectOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removed = append(f.removed, bucket+"/"+object)
	return nil
}

func (f *fakeMinioClient) PresignedGetObject(_ context.Context, bucket, object string, _ time.Duration, _ url.Values) (*url.URL, error) {
	return url.Parse("https://minio.local/" + bucket + "/" + object)
}

// newTestService 用 fakeMinioClient 替身接住 MinIO 调用；HandleLogFileReceived 的 fault
// 路径会走 detected 补全分支，运行日志路径写 repository 即可，两者都可能触发配额清理
// 从而调用到 minioClient，用真实 nil 会 panic，故用 fake 替身。
func newTestService() (*Service, *memRepo, *memRepo) {
	runRepo := newMemRunRepo()
	faultRepo := newMemFaultRepo()
	svc := NewService(runRepo, faultRepo, stubDeviceLookup{}, nil, appconfig.BucketConfig{}, zap.NewNop())
	svc.minioClient = &fakeMinioClient{}
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

func TestEnforceFaultLogQuota_IgnoresDetectedPlaceholdersForGlobalCount(t *testing.T) {
	// detected 是无文件占位记录，不应把全局文件数配额顶满；
	// 否则 ListOldest 只返回 file_received 时会误删刚到达的真实文件。
	svc, _, faultRepo := newTestService()
	minio := svc.minioClient.(*fakeMinioClient)

	base := time.Now().Add(-time.Hour)
	for i := 0; i < DefaultMaxFileCount+10; i++ {
		require.NoError(t, faultRepo.Create(context.Background(), &LogFile{
			DeviceSN:     fmt.Sprintf("SN-DETECTED-%02d", i),
			RecordStatus: FaultRecordStatusDetected,
			CollectedAt:  base.Add(time.Duration(i) * time.Minute),
		}))
	}

	deviceID := uuid.New()
	fileID := uuid.New()
	require.NoError(t, faultRepo.Create(context.Background(), &LogFile{
		ID:           fileID,
		DeviceID:     &deviceID,
		DeviceSN:     "SN-FILE-ONLY",
		FileName:     "abnormalLog_SN-FILE-ONLY.tar.gz",
		ObjectPath:   "fault/SN-FILE-ONLY.tar.gz",
		Bucket:       "logs",
		FileSize:     1024,
		RecordStatus: FaultRecordStatusFileReceived,
		CollectedAt:  time.Now(),
	}))

	require.NoError(t, svc.enforceFaultLogQuota(context.Background(), &deviceID))

	got, err := faultRepo.GetByID(context.Background(), fileID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.False(t, got.IsDeleted, "detected 占位不应触发全局配额清理真实文件")
	assert.Empty(t, minio.removed, "不应删除 MinIO 中刚上传的故障日志对象")
}

// TestServiceDelete_NotFoundMapsToSentinel 锁定 #125 修复：Delete 对格式合法但
// 不存在的 UUID 必须返回 wrap 了 commonerrors.ErrNotFound 的哨兵错误，使 handler
// 能映射到 404 而非 500。运行日志 / 故障日志两张表分别覆盖。
func TestServiceDelete_NotFoundMapsToSentinel(t *testing.T) {
	cases := []struct {
		name    string
		logType LogType
	}{
		{"running", LogTypeRunning},
		{"fault", LogTypeFault},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _ := newTestService()
			err := svc.Delete(context.Background(), uuid.New(), tc.logType, nil)
			require.Error(t, err)
			assert.ErrorIs(t, err, commonerrors.ErrNotFound)
		})
	}
}

// TestServiceDelete_ExistingRowNoSentinel 成功路径：已存在记录删除返回 nil，
// 不应误报 NotFound。
func TestServiceDelete_ExistingRowNoSentinel(t *testing.T) {
	svc, _, faultRepo := newTestService()
	require.NoError(t, svc.RecordAbnormalReboot(context.Background(), device.AbnormalRebootSnapshot{
		DeviceID:       uuid.New(),
		DeviceSN:       "SN-DEL-OK",
		HaltMainReason: "halt_reboot",
		DetectedAt:     time.Now(),
	}))
	id := faultRepo.rows[0].ID

	err := svc.Delete(context.Background(), id, LogTypeFault, nil)
	require.NoError(t, err)
	assert.False(t, errors.Is(err, commonerrors.ErrNotFound))
	assert.True(t, faultRepo.rows[0].IsDeleted)
}

// TestServiceDelete_AlreadyDeletedMapsToConflict 锁定 finding 5/6：对已软删记录再次
// 删除返回 wrap commonerrors.ErrAlreadyExists（handler 映射 409），而非静默 200 / 重复删 MinIO。
func TestServiceDelete_AlreadyDeletedMapsToConflict(t *testing.T) {
	svc, _, faultRepo := newTestService()
	require.NoError(t, svc.RecordAbnormalReboot(context.Background(), device.AbnormalRebootSnapshot{
		DeviceID:       uuid.New(),
		DeviceSN:       "SN-DEL-TWICE",
		HaltMainReason: "halt_reboot",
		DetectedAt:     time.Now(),
	}))
	id := faultRepo.rows[0].ID
	faultRepo.rows[0].IsDeleted = true

	err := svc.Delete(context.Background(), id, LogTypeFault, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrAlreadyExists)
}

// TestServiceDownloadURL_NotFoundMapsToSentinel 锁定 #125 修复：DownloadURL 对
// 不存在 UUID 返回 wrap ErrNotFound 的哨兵错误（handler 映射 404）。
func TestServiceDownloadURL_NotFoundMapsToSentinel(t *testing.T) {
	cases := []struct {
		name    string
		logType LogType
	}{
		{"running", LogTypeRunning},
		{"fault", LogTypeFault},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _ := newTestService()
			_, err := svc.DownloadURL(context.Background(), uuid.New(), tc.logType, nil)
			require.Error(t, err)
			assert.ErrorIs(t, err, commonerrors.ErrNotFound)
		})
	}
}

// TestEnforceFaultLogQuota_PerDeviceLimitIndependentOfOtherDevices 锁定 #798：每设备故障日志
// 配额（默认 5）与全局配额并存、互不替代。设备 A 连续 6 次异常重启+文件到达，超出后应只清理
// 设备 A 自己最旧的一条，设备 B 的记录不受影响。
func TestEnforceFaultLogQuota_PerDeviceLimitIndependentOfOtherDevices(t *testing.T) {
	svc, _, faultRepo := newTestService()

	deviceA := uuid.New()
	base := time.Now().Add(-time.Hour)
	for i := 0; i < 6; i++ {
		require.NoError(t, svc.RecordAbnormalReboot(context.Background(), device.AbnormalRebootSnapshot{
			DeviceID:       deviceA,
			DeviceSN:       "SN-DEVICE-A",
			HaltMainReason: "halt_reboot",
			DetectedAt:     base.Add(time.Duration(i) * time.Minute),
		}))
		require.NoError(t, svc.HandleLogFileReceived(context.Background(), newEventPayload(LogFileReceivedPayload{
			DeviceSN:   "SN-DEVICE-A",
			FileType:   "8",
			FileName:   fmt.Sprintf("abnormalLog_A_%d.tar.gz", i),
			ObjectPath: fmt.Sprintf("fault/A/%d.tar.gz", i),
			Bucket:     "logs",
			FileSize:   1024,
		})))
	}

	deviceB := uuid.New()
	require.NoError(t, svc.RecordAbnormalReboot(context.Background(), device.AbnormalRebootSnapshot{
		DeviceID:       deviceB,
		DeviceSN:       "SN-DEVICE-B",
		HaltMainReason: "halt_reboot",
		DetectedAt:     time.Now(),
	}))
	require.NoError(t, svc.HandleLogFileReceived(context.Background(), newEventPayload(LogFileReceivedPayload{
		DeviceSN:   "SN-DEVICE-B",
		FileType:   "8",
		FileName:   "abnormalLog_B_0.tar.gz",
		ObjectPath: "fault/B/0.tar.gz",
		Bucket:     "logs",
		FileSize:   1024,
	})))

	var aliveA, aliveB int
	var oldestAStillAlive bool
	for _, r := range faultRepo.rows {
		if r.IsDeleted {
			continue
		}
		switch r.DeviceSN {
		case "SN-DEVICE-A":
			aliveA++
			if r.FileName == "abnormalLog_A_0.tar.gz" {
				oldestAStillAlive = true
			}
		case "SN-DEVICE-B":
			aliveB++
		}
	}
	assert.Equal(t, DefaultMaxFileCountPerDevice, aliveA, "设备 A 未删除记录数应等于每设备配额默认值")
	assert.False(t, oldestAStillAlive, "设备 A 最旧的一条应被配额清理淘汰")
	assert.Equal(t, 1, aliveB, "设备 B 记录不应受设备 A 配额清理影响")
}

// TestEnforceFaultLogQuota_PerDeviceDisabledSkipsCleanup 锁定 #798：每设备配额显式配成 0
// 表示禁用，此时超过默认值 5 条也不应触发设备维度清理（失败路径，与上一条成功路径对照）。
func TestEnforceFaultLogQuota_PerDeviceDisabledSkipsCleanup(t *testing.T) {
	svc, _, faultRepo := newTestService()
	values := map[string]string{
		KeyMaxFileCount:          "1000", // 全局配额调大，避免掩盖本测试关注的设备维度行为
		KeyMaxFileCountPerDevice: "0",    // 0 = 禁用设备维度配额
	}
	svc.SetRetentionPolicy(NewRetentionPolicy(func(_ context.Context, category, key string) (string, bool) {
		if category != RetentionCategory {
			return "", false
		}
		v, ok := values[key]
		return v, ok
	}, nil))

	deviceA := uuid.New()
	base := time.Now().Add(-time.Hour)
	for i := 0; i < 6; i++ {
		require.NoError(t, svc.RecordAbnormalReboot(context.Background(), device.AbnormalRebootSnapshot{
			DeviceID:       deviceA,
			DeviceSN:       "SN-DEVICE-DISABLED",
			HaltMainReason: "halt_reboot",
			DetectedAt:     base.Add(time.Duration(i) * time.Minute),
		}))
		require.NoError(t, svc.HandleLogFileReceived(context.Background(), newEventPayload(LogFileReceivedPayload{
			DeviceSN:   "SN-DEVICE-DISABLED",
			FileType:   "8",
			FileName:   fmt.Sprintf("abnormalLog_D_%d.tar.gz", i),
			ObjectPath: fmt.Sprintf("fault/D/%d.tar.gz", i),
			Bucket:     "logs",
			FileSize:   1024,
		})))
	}

	var alive int
	for _, r := range faultRepo.rows {
		if !r.IsDeleted {
			alive++
		}
	}
	assert.Equal(t, 6, alive, "设备维度配额禁用后 6 条记录都不应被清理")
}

// TestServiceDownloadURL_DeletedMapsToConflict 已软删的记录下载按冲突态返回
// commonerrors.ErrAlreadyExists（finding 5/6：handler 映射 409，区别于"从不存在" 404）。
func TestServiceDownloadURL_DeletedMapsToConflict(t *testing.T) {
	svc, _, faultRepo := newTestService()
	require.NoError(t, svc.RecordAbnormalReboot(context.Background(), device.AbnormalRebootSnapshot{
		DeviceID:       uuid.New(),
		DeviceSN:       "SN-DL-DELETED",
		HaltMainReason: "halt_reboot",
		DetectedAt:     time.Now(),
	}))
	id := faultRepo.rows[0].ID
	faultRepo.rows[0].IsDeleted = true

	_, err := svc.DownloadURL(context.Background(), id, LogTypeFault, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrAlreadyExists)
}
