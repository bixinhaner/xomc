package stationlog

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

// DeviceLookup 仅需要按 SN 查设备的最小接口，避免引入整个 device.DeviceRepository。
type DeviceLookup interface {
	GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error)
}

// Service 基站日志采集服务，负责：
//  1. 订阅 SubjectLogFileReceived 事件，按日志类型写入对应表
//  2. 故障日志配额管理（全局最多 FaultLogMaxCount 条，超出删除最旧文件）
//  3. 对外提供日志文件查询和预签名下载 URL
//
// runningRepo 对应 station_running_logs（运行日志）；
// faultRepo   对应 station_fault_logs（故障/异常重启日志）。
type Service struct {
	runningRepo  Repository
	faultRepo    Repository
	deviceLookup DeviceLookup
	minioClient  *minio.Client
	buckets      appconfig.BucketConfig
	logger       *zap.Logger
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

// HandleLogFileReceived 处理 SubjectLogFileReceived 事件，按日志类型写入对应表。
func (s *Service) HandleLogFileReceived(ctx context.Context, evt event.Event) error {
	var p LogFileReceivedPayload
	if err := evt.DecodePayload(&p); err != nil {
		s.logger.Warn("decode log.file.received payload", zap.Error(err))
		return nil
	}

	logType := fileTypeToLogType(p.FileType)

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

	// 故障日志：执行全局配额管理
	if logType == LogTypeFault {
		if err := s.enforceFaultLogQuota(ctx); err != nil {
			// 配额清理失败不影响主流程，只记录警告
			s.logger.Warn("enforce fault log quota", zap.Error(err))
		}
	}

	return nil
}

// enforceFaultLogQuota 确保 station_fault_logs 表中未删除记录不超过 FaultLogMaxCount。
// 超出时从最早的文件开始清理（MinIO 删除 + 标记 is_deleted=true）。
func (s *Service) enforceFaultLogQuota(ctx context.Context) error {
	count, err := s.faultRepo.Count(ctx)
	if err != nil {
		return fmt.Errorf("count fault logs: %w", err)
	}
	if count <= FaultLogMaxCount {
		return nil
	}

	excess := int(count) - FaultLogMaxCount
	oldest, err := s.faultRepo.ListOldest(ctx, excess)
	if err != nil {
		return fmt.Errorf("list oldest fault logs: %w", err)
	}

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
	return nil
}

// List 查询日志文件列表。filter.LogType 决定查哪张表（默认运行日志）。
func (s *Service) List(ctx context.Context, filter LogFileFilter) ([]*LogFile, int64, error) {
	return s.repoFor(filter.LogType).List(ctx, filter)
}

// GetByID 按 ID 获取日志文件记录，需指定 logType 以确定查哪张表。
func (s *Service) GetByID(ctx context.Context, id uuid.UUID, logType LogType) (*LogFile, error) {
	return s.repoFor(logType).GetByID(ctx, id)
}

// LatestByDevice 获取指定设备最近一次采集的某类型日志文件。
func (s *Service) LatestByDevice(ctx context.Context, deviceID uuid.UUID, logType LogType) (*LogFile, error) {
	return s.repoFor(logType).LatestByDevice(ctx, deviceID)
}

// DownloadURL 为指定日志文件生成 MinIO 预签名下载 URL（有效期 1 小时）。
// logType 用于路由到正确的表。
func (s *Service) DownloadURL(ctx context.Context, id uuid.UUID, logType LogType) (string, error) {
	f, err := s.repoFor(logType).GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get log file: %w", err)
	}
	if f == nil {
		return "", fmt.Errorf("log file not found")
	}
	if f.IsDeleted {
		return "", fmt.Errorf("log file has been deleted")
	}

	presignedURL, err := s.minioClient.PresignedGetObject(ctx, f.Bucket, f.ObjectPath, time.Hour, nil)
	if err != nil {
		return "", fmt.Errorf("presign download url: %w", err)
	}
	return presignedURL.String(), nil
}

// Delete 删除日志文件（MinIO 文件 + 标记 is_deleted）。
// logType 用于路由到正确的表。
func (s *Service) Delete(ctx context.Context, id uuid.UUID, logType LogType) error {
	repo := s.repoFor(logType)
	f, err := repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get log file: %w", err)
	}
	if f == nil {
		return fmt.Errorf("log file not found")
	}

	if !f.IsDeleted {
		if removeErr := s.minioClient.RemoveObject(ctx, f.Bucket, f.ObjectPath, minio.RemoveObjectOptions{}); removeErr != nil {
			s.logger.Warn("remove log file from minio",
				zap.String("id", id.String()),
				zap.Error(removeErr),
			)
		}
	}
	return repo.MarkDeleted(ctx, id)
}
