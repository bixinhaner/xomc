package imsparam

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/software"
	"github.com/omcgo/omcgo/internal/storageprotection"
	devtask "github.com/omcgo/omcgo/internal/task"
)

// DistributeTypeCode 是下发任务的 UFTE typeCode，也是 CommandKey 前缀。
// 必须与 ufte 内置模板 IMS_PARAM_DISTRIBUTE 一致（占位 sub_task 与 device_task
// 用同一 key 回推 TC）。
const DistributeTypeCode = "IMS_PARAM_DISTRIBUTE"

// CollectTypeCode 是采集任务的 UFTE typeCode。
const CollectTypeCode = "IMS_PARAM_COLLECT"

// LibraryObjectRoot 参数文件库在 MinIO 里的对象前缀：ims-params/{paramType}/{fileName}。
const LibraryObjectRoot = "ims-params"

// Mover：MinIO 的最小写依赖（与 backup.LicenseMover 对齐）。
type Mover interface {
	PutObject(ctx context.Context, bucket, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	RemoveObject(ctx context.Context, bucket, objectName string, opts minio.RemoveObjectOptions) error
}

// ObjectGetter：下载用的最小读依赖。
type ObjectGetter interface {
	GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
}

// DeviceLookup：派发时按 SN 反查设备（取 ID / 在线状态）。
type DeviceLookup interface {
	GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error)
}

// TransferAddressResolver 与 backup.transferAddressResolver 同形（per-device
// 传输地址决策）。nil-safe：未注入时退化为全局 transferProvider 配置。
type TransferAddressResolver interface {
	Resolve(ctx context.Context, deviceID uuid.UUID, direction transfercfg.TransferDirection) (transfercfg.AddressDecision, error)
}

// ImportItem 单文件导入候选。
type ImportItem struct {
	FileName    string
	Content     []byte
	Description string
}

// Service 是核心网参数文件库的对外门面 + UFTE IMS_PARAM_DISTRIBUTE 派发器。
type Service struct {
	repo             Repository
	mover            Mover
	deviceLookup     DeviceLookup
	bucket           string
	taskSvc          devtask.Enqueuer
	logger           *zap.Logger
	admission        storageprotection.WriteAdmission
	transferProvider transfercfg.Provider
	downloadResolver TransferAddressResolver
}

func (s *Service) SetStorageAdmission(admission storageprotection.WriteAdmission) {
	s.admission = admission
}

func (s *Service) SetTransferProvider(p transfercfg.Provider) {
	s.transferProvider = p
}

func (s *Service) SetDownloadAddressResolver(r TransferAddressResolver) {
	s.downloadResolver = r
}

func NewService(
	repo Repository,
	mover Mover,
	deviceLookup DeviceLookup,
	taskSvc devtask.Enqueuer,
	bucket string,
	logger *zap.Logger,
) *Service {
	if repo == nil {
		panic("imsparam.Service: repo is required")
	}
	if mover == nil {
		panic("imsparam.Service: mover is required")
	}
	if bucket == "" {
		bucket = "config-backup"
	}
	return &Service{
		repo:         repo,
		mover:        mover,
		deviceLookup: deviceLookup,
		taskSvc:      taskSvc,
		bucket:       bucket,
		logger:       logger.Named("ims-param"),
	}
}

// ─────────────────────────────────────────────────────────────────────────
// 1) 导入 / 查询 / 删除
// ─────────────────────────────────────────────────────────────────────────

// ImportFromUpload 批量导入同一类型下的文件（文件库 = 下发源，仅承载可下发
// 类型：参数 FT1~7 / 下载鉴权 FT14 / 备份 FT16；日志/License/恢复等仅采集类
// 不入库）。同 (param_type, file_name) 重复上传 = 覆盖更新（行 ID 稳定）。
func (s *Service) ImportFromUpload(
	ctx context.Context, paramType string, items []ImportItem, uploadBy string,
) (*ImportResult, error) {
	def, ok := Lookup(paramType)
	if !ok {
		return nil, fmt.Errorf("%w: unknown param type %q", commonerrors.ErrInvalidInput, paramType)
	}
	if !def.DownloadSupported {
		return nil, fmt.Errorf("%w: param type %s (%s) does not support download; files are only distributed to devices",
			commonerrors.ErrInvalidInput, def.Code, def.Name)
	}
	normalized := def.Code
	out := &ImportResult{Succeeded: make([]string, 0, len(items)), Failed: make([]ImportFailure, 0)}
	for _, item := range items {
		if fail := s.importOne(ctx, normalized, item, uploadBy); fail != nil {
			out.Failed = append(out.Failed, *fail)
			continue
		}
		out.Succeeded = append(out.Succeeded, item.FileName)
	}
	return out, nil
}

func (s *Service) importOne(
	ctx context.Context, paramType string, item ImportItem, uploadBy string,
) *ImportFailure {
	fileName := filepath.Base(strings.TrimSpace(item.FileName))
	if fileName == "" || fileName == "." || fileName == ".." || strings.Contains(fileName, "..") {
		return &ImportFailure{
			FileName: item.FileName, ErrorCode: ImportErrInvalidName,
			Message: "文件名不合法",
		}
	}
	if len(item.Content) == 0 {
		return &ImportFailure{
			FileName: item.FileName, ErrorCode: ImportErrEmptyBody,
			Message: "上传内容为空",
		}
	}
	objectPath := fmt.Sprintf("%s/%s/%s", LibraryObjectRoot, paramType, url.PathEscape(fileName))
	sum := md5.Sum(item.Content)
	md5Hex := hex.EncodeToString(sum[:])

	if s.admission != nil {
		decision, err := s.admission.CheckPath(ctx, storageprotection.ProtectedPathIDMinIO, storageprotection.WriteScopeUpload)
		if err != nil {
			return &ImportFailure{FileName: item.FileName, ErrorCode: ImportErrPutObject, Message: err.Error()}
		}
		if !decision.Allowed {
			return &ImportFailure{FileName: item.FileName, ErrorCode: ImportErrPutObject, Message: "storage write protected: " + decision.Reason}
		}
	}
	if _, err := s.mover.PutObject(ctx,
		s.bucket, objectPath,
		bytes.NewReader(item.Content), int64(len(item.Content)),
		minio.PutObjectOptions{ContentType: "application/octet-stream"},
	); err != nil {
		return &ImportFailure{FileName: item.FileName, ErrorCode: ImportErrPutObject, Message: err.Error()}
	}

	file := &ParamFile{
		ParamType:    paramType,
		FileName:     fileName,
		ObjectBucket: s.bucket,
		ObjectPath:   objectPath,
		MD5:          &md5Hex,
		FileSize:     int64(len(item.Content)),
		Description:  stringPtrOrNil(strings.TrimSpace(item.Description)),
		UploadedBy:   stringPtrOrNil(uploadBy),
	}
	if err := s.repo.Upsert(ctx, file); err != nil {
		// 补偿删 MinIO 对象
		if rmErr := s.mover.RemoveObject(ctx, s.bucket, objectPath, minio.RemoveObjectOptions{}); rmErr != nil {
			s.logger.Warn("compensating RemoveObject after ims_param_files upsert failure",
				zap.String("param_type", paramType), zap.String("object", objectPath), zap.Error(rmErr))
		}
		return &ImportFailure{FileName: item.FileName, ErrorCode: ImportErrUpsert, Message: err.Error()}
	}

	s.logger.Info("ims param file imported",
		zap.String("param_type", paramType),
		zap.String("file_name", fileName),
		zap.String("object", objectPath),
		zap.Int64("size", file.FileSize),
		zap.String("upload_by", uploadBy),
	)
	return nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*ParamFile, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, filter ParamFileFilter) ([]ParamFile, int64, error) {
	return s.repo.List(ctx, filter)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get ims_param_files before delete: %w", err)
	}
	if existing == nil {
		return commonerrors.ErrNotFound
	}
	if rmErr := s.mover.RemoveObject(ctx,
		existing.ObjectBucket, existing.ObjectPath, minio.RemoveObjectOptions{},
	); rmErr != nil {
		s.logger.Warn("delete: RemoveObject failed; proceeding with DB delete",
			zap.String("id", id.String()),
			zap.String("bucket", existing.ObjectBucket),
			zap.String("object", existing.ObjectPath),
			zap.Error(rmErr),
		)
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) BatchDelete(ctx context.Context, ids []uuid.UUID) (succeeded, failed []uuid.UUID, err error) {
	succeeded = make([]uuid.UUID, 0, len(ids))
	failed = make([]uuid.UUID, 0)
	for _, id := range ids {
		if delErr := s.Delete(ctx, id); delErr != nil {
			s.logger.Warn("batch delete: single ims_param_files failed",
				zap.String("id", id.String()), zap.Error(delErr))
			failed = append(failed, id)
			continue
		}
		succeeded = append(succeeded, id)
	}
	return succeeded, failed, nil
}

// ─────────────────────────────────────────────────────────────────────────
// 2) UFTE IMS_PARAM_DISTRIBUTE 派发
// ─────────────────────────────────────────────────────────────────────────

// DispatchByFileID 按 SN 批量下发指定参数文件：逐设备入队 Download device task
// （FileType=CWMPFileTypeImsParam，URL={base}/FileDownloadService/{bucket}/{path}?paramType=FT_ImsCore_*）。
//
// upgradeTaskID 必须由调用方（UFTE）提前 CreatePlaceholderTrackingTask 后传入，
// 用于派生 CommandKey（software.BuildDirectDispatchCommandKey）让 sub_task 与
// device_task 对齐 → TC 到达自动推进。传 uuid.Nil 时退化用一次性 dispatch ID。
//
// 返回 dispatchedFiles（sn → 下发文件名），供 UFTE 写回 sub_task.dest_version。
// 设备查不到等逐台跳过仅记日志（skipped 不阻断整体）。
func (s *Service) DispatchByFileID(
	ctx context.Context, targetDeviceSNs []string, fileID uuid.UUID, createUser string, upgradeTaskID uuid.UUID,
) (uuid.UUID, map[string]string, error) {
	if s.taskSvc == nil {
		return uuid.Nil, nil, fmt.Errorf("ims param dispatcher not configured: %w", commonerrors.ErrInvalidInput)
	}
	if len(targetDeviceSNs) == 0 {
		return uuid.Nil, nil, fmt.Errorf("at least one target device required: %w", commonerrors.ErrInvalidInput)
	}
	file, err := s.repo.GetByID(ctx, fileID)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("lookup ims_param_files: %w", err)
	}
	if file == nil {
		return uuid.Nil, nil, fmt.Errorf("%w: ims param file %s not found", commonerrors.ErrNotFound, fileID)
	}
	def, ok := Lookup(file.ParamType)
	if !ok || !def.DownloadSupported {
		return uuid.Nil, nil, fmt.Errorf("%w: param type %s does not support download", commonerrors.ErrInvalidInput, file.ParamType)
	}

	dispatchID := upgradeTaskID
	if dispatchID == uuid.Nil {
		dispatchID = uuid.New()
	}

	enqueued := 0
	skipped := make([]string, 0)
	dispatchedFiles := make(map[string]string, len(targetDeviceSNs))
	for _, sn := range targetDeviceSNs {
		if s.deviceLookup == nil {
			skipped = append(skipped, sn)
			continue
		}
		dev, dErr := s.deviceLookup.GetBySerialNumber(ctx, sn)
		if dErr != nil || dev == nil {
			skipped = append(skipped, sn)
			s.logger.Warn("device not found for ims param distribute; skipping",
				zap.String("device_sn", sn), zap.Error(dErr))
			continue
		}
		commandKey := software.BuildDirectDispatchCommandKey(DistributeTypeCode, dispatchID, dev.SerialNumber)
		if qErr := s.enqueueParamDownload(ctx, file, dev, createUser, dispatchID, commandKey); qErr != nil {
			skipped = append(skipped, sn)
			s.logger.Warn("enqueue Download device task failed (ims param distribute)",
				zap.String("device_sn", sn), zap.Error(qErr))
			continue
		}
		dispatchedFiles[dev.SerialNumber] = file.FileName
		enqueued++
	}

	s.logger.Info("ims param distribute dispatched",
		zap.String("dispatch_id", dispatchID.String()),
		zap.String("param_type", file.ParamType),
		zap.String("file_name", file.FileName),
		zap.Int("target_count", len(targetDeviceSNs)),
		zap.Int("enqueued", enqueued),
		zap.Int("skipped", len(skipped)),
	)
	return dispatchID, dispatchedFiles, nil
}

// PreviewDispatchFile 不入队任何 device_task，只返回待下发文件。
// 给 UFTE 挂起 / 定时模式的占位任务用：创建时把目标文件名写到 sub_task.dest_version。
func (s *Service) PreviewDispatchFile(ctx context.Context, fileID uuid.UUID) (*ParamFile, error) {
	file, err := s.repo.GetByID(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("lookup ims_param_files (preview): %w", err)
	}
	if file == nil {
		return nil, fmt.Errorf("%w: ims param file %s not found", commonerrors.ErrNotFound, fileID)
	}
	return file, nil
}

// DispatchImsParamByFileID 实现 ufte.ImsParamDispatcher（薄适配，转发 DispatchByFileID）。
func (s *Service) DispatchImsParamByFileID(
	ctx context.Context, targetDeviceSNs []string, fileID uuid.UUID, createUser string, upgradeTaskID uuid.UUID,
) (uuid.UUID, map[string]string, error) {
	return s.DispatchByFileID(ctx, targetDeviceSNs, fileID, createUser, upgradeTaskID)
}

// PreviewImsParamFile 实现 ufte.ImsParamDispatcher：返回 (fileName, paramType)。
func (s *Service) PreviewImsParamFile(ctx context.Context, fileID uuid.UUID) (string, string, error) {
	file, err := s.PreviewDispatchFile(ctx, fileID)
	if err != nil {
		return "", "", err
	}
	return file.FileName, file.ParamType, nil
}

func (s *Service) enqueueParamDownload(
	ctx context.Context,
	file *ParamFile,
	dev *model.Device,
	createUser string,
	dispatchID uuid.UUID,
	commandKey string,
) error {
	downloadURL, err := s.resolveDownloadURL(ctx, dev, file)
	if err != nil {
		return err
	}
	md5Hex := ""
	if file.MD5 != nil {
		md5Hex = *file.MD5
	}
	// CWMP FileType 统一 "Ims File"（方向由 Download RPC 表达）；具体类型由
	// <ParameterType> 元素携带，URL query paramType 供 ACS 下载服务旁路识别。
	if _, ok := Lookup(file.ParamType); !ok {
		return fmt.Errorf("ims param file %s has unknown param type %q", file.ID, file.ParamType)
	}
	paramsJSON, err := json.Marshal(map[string]interface{}{
		"file_type":        CWMPFileTypeImsParam,
		"url":              downloadURL,
		"target_file_name": file.FileName,
		"md5":              md5Hex,
		"param_type":       file.ParamType,
	})
	if err != nil {
		return fmt.Errorf("marshal Download params for %s: %w", dev.SerialNumber, err)
	}
	if _, err := s.taskSvc.CreateTask(ctx, &devtask.CreateTaskRequest{
		DeviceSN:    dev.SerialNumber,
		Method:      "Download",
		Params:      paramsJSON,
		Source:      devtask.TaskSourceSystem,
		SourceID:    dispatchID.String(),
		CreatorID:   createUser,
		CommandKey:  commandKey,
		Description: "IMS core file distribute",
	}); err != nil {
		return err
	}
	s.logger.Info("ims param Download task enqueued",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("command_key", commandKey),
		zap.String("param_type", file.ParamType),
	)
	return nil
}

func (s *Service) resolveDownloadURL(ctx context.Context, dev *model.Device, file *ParamFile) (string, error) {
	settings := transfercfg.DownloadSettings{Path: "/smallcell/FileDownloadService"}
	if s.transferProvider != nil {
		settings = s.transferProvider.Snapshot(ctx).Download
		if settings.Path == "" {
			settings.Path = "/smallcell/FileDownloadService"
		}
	}
	baseURL := settings.BaseURL
	if s.downloadResolver != nil {
		decision, err := s.downloadResolver.Resolve(ctx, dev.ID, transfercfg.TransferDirectionDownload)
		if err != nil {
			return "", fmt.Errorf("resolve ims param download address: %w", err)
		}
		baseURL = decision.BaseURL
	}
	segments := append([]string{file.ObjectBucket}, strings.Split(file.ObjectPath, "/")...)
	query := url.Values{}
	query.Set("paramType", file.ParamType)
	return transfercfg.BuildURL(baseURL, settings.Path, segments, query)
}

func stringPtrOrNil(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
