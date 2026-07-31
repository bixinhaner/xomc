package stationlog

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/authz"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
)

// DeviceLookup 仅需要按 SN 查设备的最小接口，避免引入整个 device.DeviceRepository。
type DeviceLookup interface {
	GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error)
}

// minioObjectClient 是 Service 需要的 MinIO 客户端子集（便于单测注入 fake，避免真连接对象
// 存储 / 空指针崩溃）。*minio.Client 满足该接口。
type minioObjectClient interface {
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
	PresignedGetObject(ctx context.Context, bucketName, objectName string, expires time.Duration, reqParams url.Values) (*url.URL, error)
}

// Service 基站日志采集服务，负责：
//  1. 订阅 SubjectLogFileReceived 事件，将运行日志写入 station_running_logs
//  2. 记录满足设备类型异常重启规则的重启事实
//  3. 对外提供日志文件查询和预签名下载 URL
//
// runningRepo 对应 station_running_logs（运行日志）；
// faultRepo   对应 station_fault_logs（故障/异常重启日志）。
type Service struct {
	runningRepo  Repository
	faultRepo    Repository
	deviceLookup DeviceLookup
	minioClient  minioObjectClient
	buckets      appconfig.BucketConfig
	logger       *zap.Logger
	// groupReader 是 #63 设备组可见性强制层的按设备归属读取器；GetByID/DownloadURL/
	// Delete 在拿到记录后用它校验记录归属设备是否在调用者可见组内。nil → dev/test 退化
	// 放行（authz nil-safe）。
	groupReader authz.GroupReader
	// retentionPolicy 提供可配的故障日志文件数配额（#320）。nil → 退化用常量 FaultLogMaxCount。
	retentionPolicy *RetentionPolicy
}

// SetRetentionPolicy 注入保留策略（#320）：enforceFaultLogQuota 据此读可配的文件数配额
// （sys_configs stationlog.retention.max_file_count，0=禁用配额仅按时间保留）。Nil-safe。
func (s *Service) SetRetentionPolicy(p *RetentionPolicy) {
	s.retentionPolicy = p
}

// SetGroupReader 注入设备组归属读取器（#63 租户隔离强制层）。
func (s *Service) SetGroupReader(reader authz.GroupReader) {
	s.groupReader = reader
}

// authorizeRecord 校验一条日志记录的归属设备是否在调用者可见组内。
//
//	visibleGroups == nil   → 超管：放行。
//	groupReader == nil     → dev/test 退化：放行。
//	记录 device_id 为空     → 未关联设备：非超管不可见 → ErrForbidden（fail-closed）。
//	否则                   → 委派 authz.AuthorizeDeviceAccess 判交集。
func (s *Service) authorizeRecord(ctx context.Context, f *LogFile, visibleGroups []uuid.UUID) error {
	if visibleGroups == nil || s.groupReader == nil {
		return nil
	}
	if f.DeviceID == nil {
		return commonerrors.ErrForbidden
	}
	return authz.AuthorizeDeviceAccess(ctx, s.groupReader, *f.DeviceID, visibleGroups)
}

func NewService(
	runningRepo Repository,
	faultRepo Repository,
	deviceLookup DeviceLookup,
	minioClient *minio.Client,
	buckets appconfig.BucketConfig,
	logger *zap.Logger,
) *Service {
	return &Service{
		runningRepo:  runningRepo,
		faultRepo:    faultRepo,
		deviceLookup: deviceLookup,
		minioClient:  minioClient,
		buckets:      buckets,
		logger:       logger.Named("stationlog"),
	}
}

// repoFor 根据日志类型返回对应的仓库。
func (s *Service) repoFor(lt LogType) Repository {
	if lt == LogTypeFault {
		return s.faultRepo
	}
	return s.runningRepo
}

// RecordAbnormalReboot 把"识别即落库"事件写入 station_fault_logs，状态 detected。
//
// 这是 device.AbnormalRebootRecorder 接口的实现，由 modules.go 在 wiring 时注入
// 到 DeviceService。设计要点见 docs/project/abnormal-reboot-log-plan-20260521.md §3.D3。
//
// 同 SN 1 秒内连续 detected 记录不去重 —— collected_at 区分；上游告警链路
// 已经有滑动窗口去抖。
func (s *Service) RecordAbnormalReboot(ctx context.Context, snap device.AbnormalRebootSnapshot) error {
	now := snap.DetectedAt
	if now.IsZero() {
		now = time.Now()
	}

	devID := snap.DeviceID
	rec := &LogFile{
		DeviceID:               &devID,
		DeviceSN:               snap.DeviceSN,
		LogType:                LogTypeFault,
		FaultReason:            snap.HaltMainReason,
		FaultDetail:            snap.HaltDetailReason,
		DeviceName:             snap.DeviceName,
		DeviceType:             snap.DeviceType,
		IsGNB:                  snap.IsGNB,
		OperateIP:              snap.OperateIP,
		SoftwareVersion:        snap.SoftwareVersion,
		RuntimeBeforeReboot:    snap.RuntimeBeforeReboot,
		RecordStatus:           FaultRecordStatusDetected,
		ManualCollectionStatus: ManualCollectionIdle,
		CollectedAt:            now,
	}

	if err := s.faultRepo.Create(ctx, rec); err != nil {
		return fmt.Errorf("create abnormal reboot record: %w", err)
	}

	s.logger.Info("abnormal reboot recorded (detected)",
		zap.String("id", rec.ID.String()),
		zap.String("device_sn", snap.DeviceSN),
		zap.String("halt_main_reason", snap.HaltMainReason),
		zap.String("halt_detail_reason", snap.HaltDetailReason),
	)
	return nil
}

// HandleLogFileReceived 处理 SubjectLogFileReceived 事件。
//
// 运行日志（LogTypeRunning）写入 station_running_logs。
// 故障日志（LogTypeFault）属于文件传输任务链路；重启记录只记录正常/异常重启事实，
// 不承载故障日志附件，因此这里不更新 station_fault_logs。
func (s *Service) HandleLogFileReceived(ctx context.Context, evt event.Event) error {
	var p LogFileReceivedPayload
	if err := evt.DecodePayload(&p); err != nil {
		s.logger.Warn("decode log.file.received payload", zap.Error(err))
		return nil
	}

	logType := fileTypeToLogType(p.FileType)
	p.DeviceSN = strings.TrimSpace(p.DeviceSN)
	if logType == LogTypeFault {
		s.logger.Info("ignore fault log file for reboot records",
			zap.String("device_sn", p.DeviceSN),
			zap.String("file_name", p.FileName),
			zap.String("path", p.ObjectPath),
		)
		return nil
	}

	// 从文件名解析到的 device_sn 可能为空（非标准命名），此时跳过设备关联
	var deviceID *uuid.UUID
	if p.DeviceSN != "" {
		if dev, err := s.deviceLookup.GetBySerialNumber(ctx, p.DeviceSN); err == nil && dev != nil {
			deviceID = &dev.ID
		}
	}

	f := &LogFile{
		DeviceID:    deviceID,
		DeviceSN:    p.DeviceSN,
		LogType:     logType,
		FileName:    p.FileName,
		ObjectPath:  p.ObjectPath,
		Bucket:      p.Bucket,
		FileSize:    p.FileSize,
		CollectedAt: time.Now(),
	}

	if err := s.repoFor(logType).Create(ctx, f); err != nil {
		return fmt.Errorf("create station log record (%s): %w", logType, err)
	}

	s.logger.Info("station log file recorded",
		zap.String("id", f.ID.String()),
		zap.String("log_type", string(logType)),
		zap.String("device_sn", p.DeviceSN),
		zap.String("path", p.ObjectPath),
	)

	return nil
}

// enforceFaultLogQuota 确保 station_fault_logs 表中未删除记录不超过文件数配额，并额外确保
// deviceID（若非空）对应设备自己的记录不超过每设备配额（#798）。超出时从最早的文件开始清理
// （MinIO 删除 + 标记 is_deleted=true）。
//
// #320：全局配额值改为可配（sys_configs stationlog.retention.max_file_count，默认 20）。
// maxCount<=0 表示禁用文件数配额（仅靠按时间保留 cron 治理）。本配额与时间保留并存：
// 配额管短时洪泛，时间保留管长期留存。
//
// #798：每设备配额（sys_configs stationlog.retention.max_file_count_per_device，默认 5）与
// 全局配额并存、互不替代——全局兜底总量失控，设备维度防止单台设备刷屏挤占其他设备的保留空间。
// deviceID 为 nil（无法关联设备的历史兼容路径）时跳过设备维度检查。
func (s *Service) enforceFaultLogQuota(ctx context.Context, deviceID *uuid.UUID) error {
	maxCount := FaultLogMaxCount
	maxPerDevice := DefaultMaxFileCountPerDevice
	if s.retentionPolicy != nil {
		maxCount = s.retentionPolicy.MaxFileCount(ctx)
		maxPerDevice = s.retentionPolicy.MaxFileCountPerDevice(ctx)
	}

	if maxCount > 0 {
		if err := s.enforceGlobalFaultLogQuota(ctx, maxCount); err != nil {
			return err
		}
	}

	if deviceID != nil && maxPerDevice > 0 {
		if err := s.enforcePerDeviceFaultLogQuota(ctx, *deviceID, maxPerDevice); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) enforceGlobalFaultLogQuota(ctx context.Context, maxCount int) error {
	count, err := s.faultRepo.Count(ctx)
	if err != nil {
		return fmt.Errorf("count fault logs: %w", err)
	}
	if count <= int64(maxCount) {
		return nil
	}

	excess := int(count - int64(maxCount))
	oldest, err := s.faultRepo.ListOldest(ctx, excess)
	if err != nil {
		return fmt.Errorf("list oldest fault logs: %w", err)
	}
	s.removeOldestFaultLogs(ctx, oldest)
	return nil
}

func (s *Service) enforcePerDeviceFaultLogQuota(ctx context.Context, deviceID uuid.UUID, maxPerDevice int) error {
	count, err := s.faultRepo.CountByDevice(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("count fault logs by device: %w", err)
	}
	if count <= int64(maxPerDevice) {
		return nil
	}

	excess := int(count - int64(maxPerDevice))
	oldest, err := s.faultRepo.ListOldestByDevice(ctx, deviceID, excess)
	if err != nil {
		return fmt.Errorf("list oldest fault logs by device: %w", err)
	}
	s.removeOldestFaultLogs(ctx, oldest)
	return nil
}

// removeOldestFaultLogs 逐条删除 MinIO 对象并软删 PG 行，供全局/设备两种配额清理复用。
func (s *Service) removeOldestFaultLogs(ctx context.Context, oldest []*LogFile) {
	for _, old := range oldest {
		if removeErr := s.minioClient.RemoveObject(ctx, old.Bucket, old.ObjectPath, minio.RemoveObjectOptions{}); removeErr != nil {
			s.logger.Warn("remove fault log from minio",
				zap.String("id", old.ID.String()),
				zap.String("path", old.ObjectPath),
				zap.Error(removeErr),
			)
		}
		if markErr := s.faultRepo.MarkDeleted(ctx, old.ID); markErr != nil {
			s.logger.Warn("mark fault log deleted",
				zap.String("id", old.ID.String()),
				zap.Error(markErr),
			)
		} else {
			s.logger.Info("fault log quota: removed old file",
				zap.String("id", old.ID.String()),
				zap.String("device_sn", old.DeviceSN),
				zap.String("path", old.ObjectPath),
			)
		}
	}
}

// List 查询日志文件列表。filter.LogType 决定查哪张表（默认运行日志）。
func (s *Service) List(ctx context.Context, filter LogFileFilter) ([]*LogFile, int64, error) {
	return s.repoFor(filter.LogType).List(ctx, filter)
}

// GetByID 按 ID 获取日志文件记录，需指定 logType 以确定查哪张表。
// #63：拿到记录后按 visibleGroups 校验归属（记录存在但越权 → ErrForbidden）。
func (s *Service) GetByID(ctx context.Context, id uuid.UUID, logType LogType, visibleGroups []uuid.UUID) (*LogFile, error) {
	f, err := s.repoFor(logType).GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, nil
	}
	if authzErr := s.authorizeRecord(ctx, f, visibleGroups); authzErr != nil {
		return nil, authzErr
	}
	return f, nil
}

// LatestByDevice 获取指定设备最近一次采集的某类型日志文件。
func (s *Service) LatestByDevice(ctx context.Context, deviceID uuid.UUID, logType LogType) (*LogFile, error) {
	return s.repoFor(logType).LatestByDevice(ctx, deviceID)
}

// DownloadURL 为指定日志文件生成 MinIO 预签名下载 URL（有效期 1 小时）。
// logType 用于路由到正确的表。
//
// 错误语义（finding 5/6）：记录不存在 → ErrNotFound（handler 映射 404）；记录已删除
// → ErrAlreadyExists（409，资源处于"已删除"冲突态，不再可下载）。#63：越权 → ErrForbidden。
func (s *Service) DownloadURL(ctx context.Context, id uuid.UUID, logType LogType, visibleGroups []uuid.UUID) (string, error) {
	f, err := s.repoFor(logType).GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get log file: %w", err)
	}
	if f == nil {
		return "", fmt.Errorf("log file not found: %w", commonerrors.ErrNotFound)
	}
	if authzErr := s.authorizeRecord(ctx, f, visibleGroups); authzErr != nil {
		return "", authzErr
	}
	if f.IsDeleted {
		return "", fmt.Errorf("log file has been deleted: %w", commonerrors.ErrAlreadyExists)
	}

	presignedURL, err := s.minioClient.PresignedGetObject(ctx, f.Bucket, f.ObjectPath, time.Hour, nil)
	if err != nil {
		return "", fmt.Errorf("presign download url: %w", err)
	}
	return presignedURL.String(), nil
}

// Delete 删除日志文件（MinIO 文件 + 标记 is_deleted）。
// logType 用于路由到正确的表。
//
// 错误语义（finding 5/6）：记录不存在 → ErrNotFound（404）；记录已删除 →
// ErrAlreadyExists（409，幂等冲突态，避免重复删 MinIO + 误返 200）。#63：越权 → ErrForbidden。
func (s *Service) Delete(ctx context.Context, id uuid.UUID, logType LogType, visibleGroups []uuid.UUID) error {
	repo := s.repoFor(logType)
	f, err := repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get log file: %w", err)
	}
	if f == nil {
		return fmt.Errorf("log file not found: %w", commonerrors.ErrNotFound)
	}
	if authzErr := s.authorizeRecord(ctx, f, visibleGroups); authzErr != nil {
		return authzErr
	}
	if f.IsDeleted {
		return fmt.Errorf("log file already deleted: %w", commonerrors.ErrAlreadyExists)
	}

	if removeErr := s.minioClient.RemoveObject(ctx, f.Bucket, f.ObjectPath, minio.RemoveObjectOptions{}); removeErr != nil {
		s.logger.Warn("remove log file from minio",
			zap.String("id", id.String()),
			zap.Error(removeErr),
		)
	}
	return repo.MarkDeleted(ctx, id)
}
