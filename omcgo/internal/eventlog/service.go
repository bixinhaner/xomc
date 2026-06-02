package eventlog

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/device"
)

// Service 事件日志服务：识别即落库 + 列表查询。
//
// 与 stationlog.Service 的关系：两套独立的"识别即落库"链路。
//   - stationlog: 异常重启（1 BOOT + HaltReason 非空）→ station_fault_logs（带文件管理）
//   - eventlog:   普通 1 BOOT（无 HaltReason）→ event_logs（轻量审计流水）
//
// device.RecordBootFromInform 根据 abnormal 标志分流：
//   - abnormal=true  → stationlog.RecordAbnormalReboot
//   - abnormal=false → eventlog.RecordBootEvent  ← 本服务
type Service struct {
	repo   Repository
	logger *zap.Logger
}

func NewService(repo Repository, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger.Named("eventlog")}
}

// RecordBootEvent 实现 device.BootEventRecorder 接口：把一次普通 1 BOOT 写入 event_logs。
//
// 跟 stationlog.RecordAbnormalReboot 共用 device.AbnormalRebootSnapshot 不同，
// 这里复用 device.BootEventSnapshot（更瘦的 snapshot 类型，无 HaltReason）。
func (s *Service) RecordBootEvent(ctx context.Context, snap device.BootEventSnapshot) error {
	now := snap.OccurredAt
	if now.IsZero() {
		now = time.Now()
	}

	// event_data 编码 boot_count 与事件列表（前端列表暂不展示，留给未来扩展/排错）
	dataMap := map[string]interface{}{
		"boot_count": snap.BootCount,
		"events":     snap.Events,
	}
	dataBytes, _ := json.Marshal(dataMap)

	devID := snap.DeviceID
	e := &EventLog{
		DeviceID:        &devID,
		DeviceSN:        snap.DeviceSN,
		DeviceName:      snap.DeviceName,
		DeviceType:      snap.DeviceType,
		IsGNB:           snap.IsGNB,
		OperateIP:       snap.OperateIP,
		SoftwareVersion: snap.SoftwareVersion,
		EventType:       EventTypeBoot,
		EventReason:     "设备重启完成", // 与前端 Mock "log.event.reboot" 描述对齐
		EventLevel:      EventLevelInfo,
		EventData:       dataBytes,
		OccurredAt:      now,
	}

	if err := s.repo.Create(ctx, e); err != nil {
		return fmt.Errorf("create event log: %w", err)
	}

	s.logger.Info("event log recorded",
		zap.String("id", e.ID.String()),
		zap.String("device_sn", snap.DeviceSN),
		zap.String("event_type", EventTypeBoot),
		zap.Int("boot_count", snap.BootCount),
	)
	return nil
}

// List 查询事件日志列表（分页 + 过滤）。
func (s *Service) List(ctx context.Context, filter Filter) ([]*EventLog, int64, error) {
	return s.repo.List(ctx, filter)
}

// GetByID 按 ID 获取单条事件日志。
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*EventLog, error) {
	return s.repo.GetByID(ctx, id)
}

// StatByDevice 按设备聚合事件日志重启次数（跟随过滤条件，不分页）。
func (s *Service) StatByDevice(ctx context.Context, filter Filter) ([]*DeviceRebootStat, error) {
	return s.repo.StatByDevice(ctx, filter)
}
