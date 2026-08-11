// Package backup — LicenseService (T-0165).
//
// 设备 license 库管理 + LICENSE_UPGRADE 任务派发。结构对齐 SnapshotService
// 但只支持"手动导入"路径（license 不来自备份链路）。
//
// 派发：DispatchLicenseUpgradeBySN 实现 ufte.LicenseUpgradeDispatcher 接口，
// 与 SnapshotConfigRestoreDispatcher 同形——按 SN 批量从 device_licenses 取
// 最新一份 license → 整批拒绝校验缺失 → 逐设备入队 Download device task
// (FileType="License File", URL=device-licenses/<SN>_LIC.<ext>)。
package backup

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	pathpkg "path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/storageprotection"
	devtask "github.com/omcgo/omcgo/internal/task"
)

// LicenseMover：MinIO 的最小依赖（与 SnapshotMover 对齐）。
type LicenseMover interface {
	PutObject(ctx context.Context, bucket, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	RemoveObject(ctx context.Context, bucket, objectName string, opts minio.RemoveObjectOptions) error
}

// LicenseDeviceLookup：补充展示字段（enb_name / product_type）+ 取 OUI 用。
type LicenseDeviceLookup interface {
	GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error)
}

// LicenseImportItem 单文件导入候选。
type LicenseImportItem struct {
	FileName    string
	Content     []byte
	Description string
}

// LicenseImportFailure 单文件失败原因。
type LicenseImportFailure struct {
	FileName     string `json:"file_name"`
	SerialNumber string `json:"serial_number,omitempty"`
	ErrorCode    string `json:"error_code"`
	Message      string `json:"message"`
}

// LicenseImportResult 批量导入结果。
type LicenseImportResult struct {
	Succeeded []string               `json:"succeeded"`
	Failed    []LicenseImportFailure `json:"failed"`
}

// LicenseService 是 DeviceLicense 的对外门面 + UFTE LICENSE_UPGRADE 派发器。
type LicenseService struct {
	repo         LicenseRepository
	mover        LicenseMover
	deviceLookup LicenseDeviceLookup
	bucket       string
	// taskSvc 与 restoreSvc 同款，用 devtask.Enqueuer 入队 Download。
	taskSvc   devtask.Enqueuer
	logger    *zap.Logger
	admission storageprotection.WriteAdmission
}

func (s *LicenseService) SetStorageAdmission(admission storageprotection.WriteAdmission) {
	s.admission = admission
}

// NewLicenseService 装配。
//
// taskSvc / deviceLookup 必填——LICENSE_UPGRADE 派发需要这两者；
// taskSvc=nil 时 DispatchLicenseUpgradeBySN 直接返 ErrNotConfigured。
func NewLicenseService(
	repo LicenseRepository,
	mover LicenseMover,
	deviceLookup LicenseDeviceLookup,
	taskSvc devtask.Enqueuer,
	bucket string,
	logger *zap.Logger,
) *LicenseService {
	if repo == nil {
		panic("LicenseService: repo is required")
	}
	if mover == nil {
		panic("LicenseService: mover is required")
	}
	if bucket == "" {
		bucket = LicenseBucketDefault
	}
	return &LicenseService{
		repo:         repo,
		mover:        mover,
		deviceLookup: deviceLookup,
		taskSvc:      taskSvc,
		bucket:       bucket,
		logger:       logger.Named("device-license"),
	}
}

// ─────────────────────────────────────────────────────────────────────────
// 1) Import — 手动批量上传
// ─────────────────────────────────────────────────────────────────────────

// ImportFromUpload 接收文件批量导入。
func (s *LicenseService) ImportFromUpload(
	ctx context.Context, items []LicenseImportItem, uploadBy string,
) (*LicenseImportResult, error) {
	out := &LicenseImportResult{
		Succeeded: make([]string, 0, len(items)),
		Failed:    make([]LicenseImportFailure, 0),
	}
	for _, item := range items {
		fail := s.importOne(ctx, item, uploadBy)
		if fail != nil {
			out.Failed = append(out.Failed, *fail)
			continue
		}
		sn, _, _ := ValidateImportLicenseFileName(item.FileName, "")
		out.Succeeded = append(out.Succeeded, sn)
	}
	return out, nil
}

func (s *LicenseService) importOne(
	ctx context.Context, item LicenseImportItem, uploadBy string,
) *LicenseImportFailure {
	sn, ext, err := ValidateImportLicenseFileName(item.FileName, "")
	if err != nil {
		return &LicenseImportFailure{
			FileName: item.FileName, ErrorCode: ImportErrInvalidName, Message: err.Error(),
		}
	}
	if len(item.Content) == 0 {
		return &LicenseImportFailure{
			FileName: item.FileName, SerialNumber: sn,
			ErrorCode: ImportErrEmptyBody, Message: "上传内容为空",
		}
	}
	objectPath := LicenseObjectPath(sn, ext)
	sum := md5.Sum(item.Content)
	md5Hex := hex.EncodeToString(sum[:])

	if err := s.checkStorageAdmission(ctx); err != nil {
		return &LicenseImportFailure{
			FileName: item.FileName, SerialNumber: sn,
			ErrorCode: ImportErrPutObject, Message: err.Error(),
		}
	}
	if _, err := s.mover.PutObject(ctx,
		s.bucket, objectPath,
		bytes.NewReader(item.Content), int64(len(item.Content)),
		minio.PutObjectOptions{ContentType: "application/octet-stream"},
	); err != nil {
		return &LicenseImportFailure{
			FileName: item.FileName, SerialNumber: sn,
			ErrorCode: ImportErrPutObject, Message: err.Error(),
		}
	}

	lic := &DeviceLicense{
		SerialNumber: sn,
		FileName:     LicenseObjectPath(sn, ext),
		FileExt:      ext,
		ObjectBucket: s.bucket,
		ObjectPath:   objectPath,
		MD5:          &md5Hex,
		FileSize:     int64(len(item.Content)),
		Source:       LicenseSourceManualUpload,
		UpdateBy:     stringPtrOrNil(uploadBy),
		Description:  stringPtrOrNil(strings.TrimSpace(item.Description)),
	}
	s.applyDeviceFields(ctx, lic, sn)

	if err := s.repo.Upsert(ctx, lic); err != nil {
		// 补偿删 MinIO 对象
		if rmErr := s.mover.RemoveObject(ctx, s.bucket, objectPath, minio.RemoveObjectOptions{}); rmErr != nil {
			s.logger.Warn("compensating RemoveObject after upsert failure",
				zap.String("sn", sn), zap.String("object", objectPath), zap.Error(rmErr))
		}
		return &LicenseImportFailure{
			FileName: item.FileName, SerialNumber: sn,
			ErrorCode: ImportErrUpsert, Message: err.Error(),
		}
	}

	s.logger.Info("license imported",
		zap.String("sn", sn), zap.String("file_name", lic.FileName),
		zap.Int64("size", lic.FileSize), zap.String("upload_by", uploadBy),
	)
	// Import is successful once the preinstall is durable. If the device is
	// already online, deliver immediately; a transient enqueue failure leaves
	// the row pending for the next registered/online event.
	if err := s.DispatchPendingLicense(ctx, sn); err != nil {
		s.logger.Warn("immediate preinstalled license dispatch failed; left pending",
			zap.String("sn", sn), zap.Error(err))
	}
	return nil
}

func (s *LicenseService) checkStorageAdmission(ctx context.Context) error {
	if s.admission == nil {
		return nil
	}
	decision, err := s.admission.Check(ctx, storageprotection.TargetFilesystem, storageprotection.UnifiedStorageTargetID, storageprotection.WriteScopeUpload)
	if err != nil {
		return fmt.Errorf("storage admission check: %w", err)
	}
	if !decision.Allowed {
		return fmt.Errorf("storage write protected: %s", decision.Reason)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────
// 2) Read
// ─────────────────────────────────────────────────────────────────────────

func (s *LicenseService) GetBySerialNumber(ctx context.Context, sn string) (*DeviceLicense, error) {
	return s.repo.GetBySerialNumber(ctx, sn)
}

func (s *LicenseService) List(ctx context.Context, filter LicenseFilter) ([]DeviceLicense, int64, error) {
	return s.repo.List(ctx, filter)
}

// ValidateDeviceSNs 与 SnapshotService.ValidateDeviceSNs 对齐：前端导入抽屉用。
func (s *LicenseService) ValidateDeviceSNs(
	ctx context.Context, sns []string,
) (existing []string, missing []string) {
	existing = make([]string, 0, len(sns))
	missing = make([]string, 0)
	if s.deviceLookup == nil {
		existing = append(existing, sns...)
		return existing, missing
	}
	for _, sn := range sns {
		if sn == "" {
			continue
		}
		dev, err := s.deviceLookup.GetBySerialNumber(ctx, sn)
		if err == nil && dev == nil {
			missing = append(missing, sn)
			continue
		}
		existing = append(existing, sn)
	}
	return existing, missing
}

// ─────────────────────────────────────────────────────────────────────────
// 3) Delete
// ─────────────────────────────────────────────────────────────────────────

func (s *LicenseService) Delete(ctx context.Context, sn string) error {
	if sn == "" {
		return fmt.Errorf("serial_number 不能为空")
	}
	existing, err := s.repo.GetBySerialNumber(ctx, sn)
	if err != nil {
		return fmt.Errorf("get device_license before delete: %w", err)
	}
	if existing != nil && existing.ObjectBucket != "" && existing.ObjectPath != "" {
		if rmErr := s.mover.RemoveObject(ctx,
			existing.ObjectBucket, existing.ObjectPath, minio.RemoveObjectOptions{},
		); rmErr != nil {
			s.logger.Warn("delete: RemoveObject failed; proceeding with DB delete",
				zap.String("sn", sn),
				zap.String("bucket", existing.ObjectBucket),
				zap.String("object", existing.ObjectPath),
				zap.Error(rmErr))
		}
	}
	return s.repo.Delete(ctx, sn)
}

func (s *LicenseService) BatchDelete(
	ctx context.Context, sns []string,
) (succeeded, failed []string, err error) {
	succeeded = make([]string, 0, len(sns))
	failed = make([]string, 0)
	for _, sn := range sns {
		if delErr := s.Delete(ctx, sn); delErr != nil {
			s.logger.Warn("batch delete: single sn failed",
				zap.String("sn", sn), zap.Error(delErr))
			failed = append(failed, sn)
			continue
		}
		succeeded = append(succeeded, sn)
	}
	return succeeded, failed, nil
}

// ─────────────────────────────────────────────────────────────────────────
// 4) DispatchLicenseUpgradeBySN — UFTE LICENSE_UPGRADE 派发
// ─────────────────────────────────────────────────────────────────────────

// LicenseUpgradeDispatchResult 同 SnapshotConfigRestore 的返回形态。
type LicenseUpgradeDispatchResult struct {
	DispatchedID uuid.UUID
	Missing      []string
}

// DispatchLicenseUpgradeBySN 批量按 SN 取 license → 缺失整批拒绝 → 逐设备
// 入队 Download device task。FileType="License File"，URL=bucket/path。
//
// upgradeTaskID 必须由调用方（UFTE）提前 CreatePlaceholderTrackingTask 后传入，
// 用于派生 CommandKey 让 sub_task 和 device_task 对齐 → TC 到达可自动推进。
// 传 uuid.Nil 时退化用一次性 dispatch ID（兼容旧调用，没有 TC 跟踪能力）。
//
// 返回：
//   - dispatchedID：占位（当前不持久化主任务，UFTE 上层用 ufte.Task.ID 替代）
//   - missing：缺失 license 的 SN 列表；非空时返 ErrNotFound 包装
//   - err：基础设施错误
func (s *LicenseService) DispatchLicenseUpgradeBySN(
	ctx context.Context, targetDeviceSNs []string, createUser string, upgradeTaskID uuid.UUID,
) (uuid.UUID, map[string]string, []string, error) {
	if s.taskSvc == nil {
		return uuid.Nil, nil, nil, fmt.Errorf("license upgrade dispatcher not configured: %w", commonerrors.ErrInvalidInput)
	}
	if len(targetDeviceSNs) == 0 {
		return uuid.Nil, nil, nil, fmt.Errorf("at least one target device required: %w", commonerrors.ErrInvalidInput)
	}

	licMap, err := s.repo.BatchGetBySerialNumbers(ctx, targetDeviceSNs)
	if err != nil {
		return uuid.Nil, nil, nil, fmt.Errorf("lookup device_licenses: %w", err)
	}
	missing := make([]string, 0)
	for _, sn := range targetDeviceSNs {
		if _, ok := licMap[sn]; !ok {
			missing = append(missing, sn)
		}
	}
	if len(missing) > 0 {
		return uuid.Nil, nil, missing,
			fmt.Errorf("the following devices have no license: %v: %w",
				missing, commonerrors.ErrNotFound)
	}

	// dispatchID 是兜底（upgradeTaskID==Nil 时用），主路径用 upgradeTaskID 派生 CommandKey
	dispatchID := upgradeTaskID
	if dispatchID == uuid.Nil {
		dispatchID = uuid.New()
	}
	tidShort := strings.ReplaceAll(dispatchID.String(), "-", "")
	if len(tidShort) >= 8 {
		tidShort = tidShort[:8]
	}

	enqueued := 0
	skipped := make([]string, 0)
	// dispatchedFiles：返回给 UFTE 写回 sub_task.dest_version，作为"目标文件"列展示。
	// 只收成功 enqueue 的设备；skipped 不进 map 以免覆盖。
	dispatchedFiles := make(map[string]string, len(targetDeviceSNs))
	for _, sn := range targetDeviceSNs {
		lic := licMap[sn]
		if s.deviceLookup == nil {
			skipped = append(skipped, sn)
			continue
		}
		dev, dErr := s.deviceLookup.GetBySerialNumber(ctx, sn)
		if dErr != nil || dev == nil {
			skipped = append(skipped, sn)
			s.logger.Warn("device not found for license upgrade; skipping",
				zap.String("device_sn", sn), zap.Error(dErr))
			continue
		}
		// CommandKey 与 software.BuildDirectDispatchCommandKey 严格对齐：
		// "<typeCode>_<upgradeTaskID8>_<sn>"。这样 handleTCBody.GetByCommandKey
		// 能在 upgrade_sub_tasks 表里命中本任务的子任务并自动推进。
		commandKey := fmt.Sprintf("LICENSE_UPGRADE_%s_%s", tidShort, dev.SerialNumber)
		targetFileName, qErr := s.enqueueLicenseDownload(ctx, lic, dev, createUser, dispatchID, commandKey)
		if qErr != nil {
			skipped = append(skipped, sn)
			s.logger.Warn("enqueue Download device task failed (license upgrade)",
				zap.String("device_sn", sn), zap.Error(qErr))
			continue
		}
		dispatchedFiles[dev.SerialNumber] = targetFileName
		enqueued++
	}

	s.logger.Info("license upgrade dispatched",
		zap.String("dispatch_id", dispatchID.String()),
		zap.Int("target_count", len(targetDeviceSNs)),
		zap.Int("enqueued", enqueued),
		zap.Int("skipped", len(skipped)),
	)
	return dispatchID, dispatchedFiles, nil, nil
}

// DispatchPendingLicense sends a preinstalled license once the target device
// exists and is online. The repository claim makes registered+online delivery
// idempotent across app replicas.
func (s *LicenseService) DispatchPendingLicense(ctx context.Context, sn string) error {
	if sn == "" || s.taskSvc == nil || s.deviceLookup == nil {
		return nil
	}
	lic, err := s.repo.GetBySerialNumber(ctx, sn)
	if err != nil || lic == nil {
		return err
	}
	dev, err := s.deviceLookup.GetBySerialNumber(ctx, sn)
	if err != nil {
		return fmt.Errorf("lookup device for preinstalled license: %w", err)
	}
	if dev == nil || !dev.IsOnline {
		return nil
	}
	claimed, err := s.repo.ClaimAutoDispatch(ctx, sn)
	if err != nil || !claimed {
		return err
	}
	dispatchID := uuid.New()
	shortID := strings.ReplaceAll(dispatchID.String(), "-", "")[:8]
	commandKey := fmt.Sprintf("LICENSE_PREINSTALL_%s_%s", shortID, sn)
	if _, err := s.enqueueLicenseDownload(ctx, lic, dev, "", dispatchID, commandKey); err != nil {
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		if releaseErr := s.repo.ReleaseAutoDispatch(releaseCtx, sn); releaseErr != nil {
			s.logger.Error("release failed preinstall dispatch claim",
				zap.String("sn", sn), zap.Error(releaseErr))
		}
		return err
	}
	s.logger.Info("preinstalled license dispatched",
		zap.String("sn", sn), zap.String("command_key", commandKey))
	return nil
}

func (s *LicenseService) enqueueLicenseDownload(
	ctx context.Context,
	lic *DeviceLicense,
	dev *model.Device,
	createUser string,
	dispatchID uuid.UUID,
	commandKey string,
) (string, error) {
	targetFileName := pathpkg.Base(lic.ObjectPath)
	licMD5 := ""
	if lic.MD5 != nil {
		licMD5 = *lic.MD5
	}
	params, err := json.Marshal(map[string]interface{}{
		"file_type":        "License File",
		"url":              lic.ObjectBucket + "/" + lic.ObjectPath,
		"target_file_name": targetFileName,
		"md5":              licMD5,
	})
	if err != nil {
		return "", fmt.Errorf("marshal Download params for %s: %w", dev.SerialNumber, err)
	}
	description := "License upgrade download"
	if strings.HasPrefix(commandKey, "LICENSE_PREINSTALL_") {
		description = "Auto-install preloaded license on device online"
	}
	if _, err := s.taskSvc.CreateTask(ctx, &devtask.CreateTaskRequest{
		DeviceSN:    dev.SerialNumber,
		Method:      "Download",
		Params:      params,
		Source:      devtask.TaskSourceSystem,
		SourceID:    dispatchID.String(),
		CreatorID:   createUser,
		CommandKey:  commandKey,
		Description: description,
	}); err != nil {
		return "", err
	}
	return targetFileName, nil
}

// PreviewLicenseFiles 不入队任何 device_task，只返回 sn → 预期下发的 license 文件名。
// 给 UFTE 挂起 / 定时模式的占位任务用：创建时就把目标文件名写到 sub_task.dest_version，
// 让前端列表"目标文件"列即时可见，无需等 dispatcher 真正派发。
// 缺失 license 的设备整批拒绝（同 DispatchLicenseUpgradeBySN 语义）。
func (s *LicenseService) PreviewLicenseFiles(
	ctx context.Context, targetDeviceSNs []string,
) (map[string]string, error) {
	if len(targetDeviceSNs) == 0 {
		return nil, fmt.Errorf("at least one target device required: %w", commonerrors.ErrInvalidInput)
	}
	licMap, err := s.repo.BatchGetBySerialNumbers(ctx, targetDeviceSNs)
	if err != nil {
		return nil, fmt.Errorf("lookup device_licenses (preview): %w", err)
	}
	missing := make([]string, 0)
	for _, sn := range targetDeviceSNs {
		if _, ok := licMap[sn]; !ok {
			missing = append(missing, sn)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("the following devices have no license: %v: %w",
			missing, commonerrors.ErrNotFound)
	}
	out := make(map[string]string, len(targetDeviceSNs))
	for _, sn := range targetDeviceSNs {
		out[sn] = pathpkg.Base(licMap[sn].ObjectPath)
	}
	return out, nil
}

// ─────────────────────────────────────────────────────────────────────────
// 5) helpers
// ─────────────────────────────────────────────────────────────────────────

func (s *LicenseService) applyDeviceFields(ctx context.Context, lic *DeviceLicense, sn string) {
	if s.deviceLookup == nil {
		return
	}
	dev, err := s.deviceLookup.GetBySerialNumber(ctx, sn)
	if err != nil || dev == nil {
		return
	}
	if dev.DeviceName != "" {
		n := dev.DeviceName
		lic.EnbName = &n
	}
	if dev.ProductClass != "" {
		pc := dev.ProductClass
		lic.ProductType = &pc
	}
}
