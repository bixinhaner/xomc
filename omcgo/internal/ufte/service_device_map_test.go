package ufte

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	coremodel "github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/software"
)

// 这组测试覆盖 BUG 修复：CONFIG_RESTORE / LICENSE_UPGRADE 设备列表
// "目标版本/目标文件"列以前始终显示 "-"，原因是 sub_task.DestVersion 留空
// 且 fileLandedLookup 对下行任务永远查不到。修复后 dispatcher 回写文件名到
// sub_task.dest_version，mapDeviceItem 把它当 TargetFile 暴露。

func mustCatalog(t *testing.T, typeCode string) []TaskType {
	t.Helper()
	for _, item := range builtInTaskTypes() {
		if item.TypeCode == typeCode {
			return []TaskType{item}
		}
	}
	t.Fatalf("typeCode %q missing from builtInTaskTypes()", typeCode)
	return nil
}

// stubDeviceCacheRepo 给 mapDeviceItem 走 cache 命中路径用——避免触发 deviceRepo
// 的实际 RPC，单测只关心 TargetFile 字段。
func newServiceForMap(t *testing.T) *Service {
	t.Helper()
	return NewService(nil, nil, nil, nil, &stubDeviceRepo{}, zap.NewNop())
}

func makeRestoreSubTask(taskID uuid.UUID, destFileName string) software.UpgradeSubTaskWithTaskName {
	return software.UpgradeSubTaskWithTaskName{
		UpgradeSubTask: software.UpgradeSubTask{
			ID:           uuid.New(),
			TaskID:       taskID,
			DeviceID:     uuid.New(),
			DeviceSN:     "SN-001",
			Status:       software.UpgradeCompleted,
			DestVersion:  destFileName,
			OriVersion:   "v1.0",
			CommandKey:   software.BuildDirectDispatchCommandKey("CONFIG_RESTORE", taskID, "SN-001"),
		},
		TaskName: "restore-task",
	}
}

func TestMapDeviceItem_ConfigRestore_SurfacesDestVersionAsTargetFile(t *testing.T) {
	svc := newServiceForMap(t)
	catalog := mustCatalog(t, "CONFIG_RESTORE")
	parent := &software.UpgradeTask{
		ID:               uuid.New(),
		TaskName:         "restore-task",
		TaskType:         software.TaskTypeLogCollect, // 占位任务沿用 LogCollect 业务码
		DownloadFileType: "10 <OUI> Configuration File",
		ProductClass:     "4G eNB",
	}
	sub := makeRestoreSubTask(parent.ID, "snap_SN-001_20260524.nv")
	sub.TaskID = parent.ID

	cache := map[uuid.UUID]*coremodel.Device{
		sub.DeviceID: {ID: sub.DeviceID, SerialNumber: "SN-001", ProductClass: "4G eNB", FirmwareVersion: "v1.0"},
	}
	item, err := svc.mapDeviceItem(context.Background(), catalog, sub, parent, cache)
	require.NoError(t, err)
	assert.Equal(t, "snap_SN-001_20260524.nv", item.TargetFile, "TargetFile 应回填 sub_task.dest_version")
	assert.Empty(t, item.TargetVersion, "LogCollect 类 TargetVersion 仍清空")
	assert.Empty(t, item.DownloadURL, "CONFIG_RESTORE 是下行方向，无 downloadURL")
}

func TestMapDeviceItem_LicenseUpgrade_SurfacesDestVersionAsTargetFile(t *testing.T) {
	svc := newServiceForMap(t)
	catalog := mustCatalog(t, "LICENSE_UPGRADE")
	parent := &software.UpgradeTask{
		ID:               uuid.New(),
		TaskName:         "license-task",
		TaskType:         software.TaskTypeLogCollect,
		DownloadFileType: "License File",
		ProductClass:     "4G eNB",
	}
	sub := software.UpgradeSubTaskWithTaskName{
		UpgradeSubTask: software.UpgradeSubTask{
			ID:          uuid.New(),
			TaskID:      parent.ID,
			DeviceID:    uuid.New(),
			DeviceSN:    "SN-002",
			Status:      software.UpgradeCompleted,
			DestVersion: "device-SN-002.lic",
			CommandKey:  software.BuildDirectDispatchCommandKey("LICENSE_UPGRADE", parent.ID, "SN-002"),
		},
		TaskName: "license-task",
	}
	cache := map[uuid.UUID]*coremodel.Device{
		sub.DeviceID: {ID: sub.DeviceID, SerialNumber: "SN-002", ProductClass: "4G eNB"},
	}
	item, err := svc.mapDeviceItem(context.Background(), catalog, sub, parent, cache)
	require.NoError(t, err)
	assert.Equal(t, "device-SN-002.lic", item.TargetFile)
	assert.Empty(t, item.TargetVersion)
	assert.Empty(t, item.DownloadURL, "LICENSE_UPGRADE 是下行方向，无 downloadURL")
}

func TestMapDeviceItem_ConfigRestore_EmptyDestVersionShowsEmptyTargetFile(t *testing.T) {
	// 兼容老数据：派发前 / dispatcher 写回前的瞬间，DestVersion 还是空——TargetFile 仍空，
	// 前端会照旧渲染 "-"。这里验证不会错误地用其它字段填充。
	svc := newServiceForMap(t)
	catalog := mustCatalog(t, "CONFIG_RESTORE")
	parent := &software.UpgradeTask{
		ID:               uuid.New(),
		TaskName:         "restore-task",
		TaskType:         software.TaskTypeLogCollect,
		DownloadFileType: "10 <OUI> Configuration File",
		FileName:         "fallback-shouldnt-leak.xml",
	}
	sub := software.UpgradeSubTaskWithTaskName{
		UpgradeSubTask: software.UpgradeSubTask{
			ID:       uuid.New(),
			TaskID:   parent.ID,
			DeviceID: uuid.New(),
			DeviceSN: "SN-003",
			Status:   software.UpgradePending,
		},
	}
	cache := map[uuid.UUID]*coremodel.Device{
		sub.DeviceID: {ID: sub.DeviceID, SerialNumber: "SN-003"},
	}
	item, err := svc.mapDeviceItem(context.Background(), catalog, sub, parent, cache)
	require.NoError(t, err)
	assert.Empty(t, item.TargetFile, "未写回前 TargetFile 保持空（不应回退用 parent.FileName）")
}

// 以下两测覆盖 issue #195：上报时间只在文件真正上报成功（终态 ended）时才填。
// sub_task.updated_at 在子任务生成 / 中间状态流转时都会刷新，但那不是
// "文件上报成功"时刻——直接拿 updated_at 会在任务刚创建时就显示一个误导值。

// 死判 empty-before：刚生成、未到终态的子任务，上报时间为空。
func TestMapDeviceItem_ReportTime_EmptyBeforeSuccess(t *testing.T) {
	svc := newServiceForMap(t)
	catalog := mustCatalog(t, "RUNTIME_LOG_COLLECT")
	parent := &software.UpgradeTask{
		ID:       uuid.New(),
		TaskName: "log-task",
		TaskType: software.TaskTypeLogCollect,
	}
	// updated_at 已是非零（子任务生成那一刻就写了），但 status 还没到 ended。
	sub := software.UpgradeSubTaskWithTaskName{
		UpgradeSubTask: software.UpgradeSubTask{
			ID:        uuid.New(),
			TaskID:    parent.ID,
			DeviceID:  uuid.New(),
			DeviceSN:  "SN-195A",
			Status:    software.UpgradeUploading, // 中间态，文件尚未上报成功
			UpdatedAt: coremodel.Time(time.Date(2026, 6, 1, 17, 35, 21, 0, time.UTC)),
		},
		TaskName: "log-task",
	}
	cache := map[uuid.UUID]*coremodel.Device{
		sub.DeviceID: {ID: sub.DeviceID, SerialNumber: "SN-195A"},
	}
	item, err := svc.mapDeviceItem(context.Background(), catalog, sub, parent, cache)
	require.NoError(t, err)
	assert.NotEqual(t, "ended", item.Status, "前置：该子任务尚未到成功终态")
	assert.Empty(t, item.LastReportAt, "未到成功终态时上报时间必须为空")
}

// 死判 filled-on-success：上报成功终态，上报时间被填为 updated_at。
func TestMapDeviceItem_ReportTime_FilledOnSuccess(t *testing.T) {
	svc := newServiceForMap(t)
	catalog := mustCatalog(t, "RUNTIME_LOG_COLLECT")
	parent := &software.UpgradeTask{
		ID:       uuid.New(),
		TaskName: "log-task",
		TaskType: software.TaskTypeLogCollect,
	}
	reportedAt := time.Date(2026, 6, 1, 17, 40, 9, 0, time.UTC)
	sub := software.UpgradeSubTaskWithTaskName{
		UpgradeSubTask: software.UpgradeSubTask{
			ID:        uuid.New(),
			TaskID:    parent.ID,
			DeviceID:  uuid.New(),
			DeviceSN:  "SN-195B",
			Status:    software.UpgradeCompleted, // 文件上报成功终态
			UpdatedAt: coremodel.Time(reportedAt),
		},
		TaskName: "log-task",
	}
	cache := map[uuid.UUID]*coremodel.Device{
		sub.DeviceID: {ID: sub.DeviceID, SerialNumber: "SN-195B"},
	}
	item, err := svc.mapDeviceItem(context.Background(), catalog, sub, parent, cache)
	require.NoError(t, err)
	assert.Equal(t, "ended", item.Status, "前置：该子任务已到成功终态")
	assert.Equal(t, reportedAt.Format(time.RFC3339), item.LastReportAt, "成功终态时上报时间应填为 updated_at")
}

// 以下两测覆盖 issue #655：三皮肤设备列表新增「开始时间 / 结束时间」两列，
// 后端 DeviceItem.StartedAt / EndedAt 透传 sub_task.started_at / completed_at。

// in-progress：started_at 已写、completed_at 还未写 → StartedAt 有值 / EndedAt 空。
func TestMapDeviceItem_StartedAndEndedAt_InProgress(t *testing.T) {
	svc := newServiceForMap(t)
	catalog := mustCatalog(t, "RUNTIME_LOG_COLLECT")
	parent := &software.UpgradeTask{
		ID:       uuid.New(),
		TaskName: "log-task",
		TaskType: software.TaskTypeLogCollect,
	}
	startedAt := time.Date(2026, 6, 25, 10, 30, 0, 0, time.UTC)
	startedAtModel := coremodel.Time(startedAt)
	sub := software.UpgradeSubTaskWithTaskName{
		UpgradeSubTask: software.UpgradeSubTask{
			ID:          uuid.New(),
			TaskID:      parent.ID,
			DeviceID:    uuid.New(),
			DeviceSN:    "SN-655A",
			Status:      software.UpgradeUploading,
			StartedAt:   &startedAtModel,
			CompletedAt: nil, // 未到终态 → repo 未写
			UpdatedAt:   coremodel.Time(startedAt.Add(time.Second)),
		},
		TaskName: "log-task",
	}
	cache := map[uuid.UUID]*coremodel.Device{
		sub.DeviceID: {ID: sub.DeviceID, SerialNumber: "SN-655A"},
	}
	item, err := svc.mapDeviceItem(context.Background(), catalog, sub, parent, cache)
	require.NoError(t, err)
	assert.Equal(t, startedAt.Format(time.RFC3339), item.StartedAt, "首次进入执行态后 StartedAt 应有值")
	assert.Empty(t, item.EndedAt, "未到终态时 EndedAt 必须为空")
}

// ended：started_at + completed_at 都已写 → 两字段都有值且不相等。
func TestMapDeviceItem_StartedAndEndedAt_Ended(t *testing.T) {
	svc := newServiceForMap(t)
	catalog := mustCatalog(t, "RUNTIME_LOG_COLLECT")
	parent := &software.UpgradeTask{
		ID:       uuid.New(),
		TaskName: "log-task",
		TaskType: software.TaskTypeLogCollect,
	}
	startedAt := time.Date(2026, 6, 25, 10, 30, 0, 0, time.UTC)
	completedAt := startedAt.Add(2 * time.Minute)
	startedAtModel := coremodel.Time(startedAt)
	completedAtModel := coremodel.Time(completedAt)
	sub := software.UpgradeSubTaskWithTaskName{
		UpgradeSubTask: software.UpgradeSubTask{
			ID:          uuid.New(),
			TaskID:      parent.ID,
			DeviceID:    uuid.New(),
			DeviceSN:    "SN-655B",
			Status:      software.UpgradeCompleted,
			StartedAt:   &startedAtModel,
			CompletedAt: &completedAtModel,
			UpdatedAt:   coremodel.Time(completedAt),
		},
		TaskName: "log-task",
	}
	cache := map[uuid.UUID]*coremodel.Device{
		sub.DeviceID: {ID: sub.DeviceID, SerialNumber: "SN-655B"},
	}
	item, err := svc.mapDeviceItem(context.Background(), catalog, sub, parent, cache)
	require.NoError(t, err)
	assert.Equal(t, startedAt.Format(time.RFC3339), item.StartedAt)
	assert.Equal(t, completedAt.Format(time.RFC3339), item.EndedAt)
	assert.NotEqual(t, item.StartedAt, item.EndedAt, "StartedAt 与 EndedAt 必须能区分开（避免回归到都用 updated_at）")
}

// pending：未进入执行态 → 两字段都为空。
func TestMapDeviceItem_StartedAndEndedAt_Pending(t *testing.T) {
	svc := newServiceForMap(t)
	catalog := mustCatalog(t, "RUNTIME_LOG_COLLECT")
	parent := &software.UpgradeTask{
		ID:       uuid.New(),
		TaskName: "log-task",
		TaskType: software.TaskTypeLogCollect,
	}
	sub := software.UpgradeSubTaskWithTaskName{
		UpgradeSubTask: software.UpgradeSubTask{
			ID:        uuid.New(),
			TaskID:    parent.ID,
			DeviceID:  uuid.New(),
			DeviceSN:  "SN-655C",
			Status:    software.UpgradePending,
			UpdatedAt: coremodel.Time(time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)),
		},
		TaskName: "log-task",
	}
	cache := map[uuid.UUID]*coremodel.Device{
		sub.DeviceID: {ID: sub.DeviceID, SerialNumber: "SN-655C"},
	}
	item, err := svc.mapDeviceItem(context.Background(), catalog, sub, parent, cache)
	require.NoError(t, err)
	assert.Empty(t, item.StartedAt, "pending 态 StartedAt 必须为空")
	assert.Empty(t, item.EndedAt, "pending 态 EndedAt 必须为空")
}
