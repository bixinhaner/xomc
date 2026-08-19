// Package backup — SnapshotService (T-0164 / B2).
//
// 业务职责：维护 config_snapshots（一设备一行最新配置快照），与 backup_tasks
// 的逐任务文件归档正交。
//
// 写入路径有两条：
//
//  1. 自动 promote：备份任务完成后 FilePathRecorder 调用 PromoteFromBackup，
//     读 config_backup bucket 的源文件 → 解密(.enc) + 解压(.gz 等) 还原成明文 →
//     PutObject 写到 config-snapshots bucket，命名统一为 <SN>_CFG.<ext>，
//     然后 Upsert 表行（MD5=明文哈希）。原任务文件保留不动。失败不影响主流程。
//     （#61：早期用 server-side CopyObject 换名复制会丢 .enc/.gz 后缀并错配
//     AEAD AAD，导致加密/压缩场景下快照永久不可用——已改为明文解码管线。）
//
//  2. 手动导入：用户上传 multipart 文件，逐项强校验命名 <SN>_CFG.{xml,nv}，
//     PutObject 到 config-snapshots，然后 Upsert 表行。DB 写失败时补偿删
//     MinIO 对象，避免孤儿。批量返回 succeeded + failed 结构化结果。
//
// 历史数据不回填（用户确认 2026-05-22）—— 本 service 只对新备份事件 / 新导入
// 操作生效，不扫旧 backup_tasks / 旧 backup_restore_file。
package backup

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/storageprotection"
)

// SnapshotMover 是 SnapshotService 对 MinIO 的最小依赖。
//
// *minio.Client 天然满足：
//   - PutObject  用于 PromoteFromBackup（写明文快照）与 ImportFromUpload
//   - RemoveObject 用于补偿 / Delete
//
// 测试通过实现该接口注入 fake。
//
// 注（#61）：PromoteFromBackup 不再用 server-side CopyObject——那条路径会把
// 源备份对象（可能压缩 + 信封加密）原样换名复制到快照桶，丢掉 .gz/.enc 后缀
// 与 AEAD 的 AAD 绑定，导致按快照恢复时下发给 CPE 的是密文/压缩字节、永久不可
// 用。现改为读源对象 → 解密 → 解压 → 明文 PutObject，故 CopyObject 已从接口移除。
type SnapshotMover interface {
	PutObject(ctx context.Context, bucket, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	RemoveObject(ctx context.Context, bucket, objectName string, opts minio.RemoveObjectOptions) error
}

// SnapshotSourceReader 读取源备份对象的完整字节，供 PromoteFromBackup 解密 +
// 解压成明文。生产用 *minio.Client 适配（见 modules.go），测试注入 fake。
// 读取必须设上限（解密/解压前先卡 64MB 量级），避免恶意/异常大对象打爆内存。
type SnapshotSourceReader interface {
	ReadObject(ctx context.Context, bucket, object string) ([]byte, error)
}

// snapshotPromoteMaxBytes 是 promote 解密/解压后明文的字节上限，对齐 acs/upload
// 与 acs/download 的 64MB 天花板（备份现实 <10MB）。
const snapshotPromoteMaxBytes = 64 * 1024 * 1024

// minioSnapshotSourceReader 用 *minio.Client 实现 SnapshotSourceReader：
// GetObject + 带上限的全量读，供 PromoteFromBackup 解码源备份对象。
type minioSnapshotSourceReader struct {
	client *minio.Client
}

// NewMinIOSnapshotSourceReader 构造生产用的源对象读取器（modules.go 注入）。
func NewMinIOSnapshotSourceReader(client *minio.Client) SnapshotSourceReader {
	return &minioSnapshotSourceReader{client: client}
}

func (r *minioSnapshotSourceReader) ReadObject(ctx context.Context, bucket, object string) ([]byte, error) {
	obj, err := r.client.GetObject(ctx, bucket, object, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("minio GetObject %s/%s: %w", bucket, object, err)
	}
	defer obj.Close()
	// 源对象是"压缩+加密"后的字节，比明文略大——上限取明文天花板 + 1KB 信封开销，
	// 再多读 1 字节用于判溢出。
	const maxBytes = snapshotPromoteMaxBytes + 1024
	data, err := io.ReadAll(io.LimitReader(obj, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read backup object %s/%s: %w", bucket, object, err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("backup object %s/%s exceeds %d bytes: %w",
			bucket, object, maxBytes, ErrEncryptionInputTooLarge)
	}
	return data, nil
}

// SnapshotBackupFileLookup 用于 PromoteFromBackup 找到刚落地的备份文件元数据。
// *PgFileRepository 通过 ListBySerial 满足（service 内部按 task_id 二次过滤）。
type SnapshotBackupFileLookup interface {
	ListBySerial(ctx context.Context, sn string) ([]BackupRestoreFile, error)
}

// SnapshotDeviceLookup 给快照行补充展示字段（enb_name / product_type）。
// 复用与 RestoreService 同形的窄接口；nil 时跳过补充（行字段留空）。
type SnapshotDeviceLookup interface {
	GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error)
}

// SnapshotImportItem 描述一项手动导入候选。
type SnapshotImportItem struct {
	// FileName 是用户上传时的原始文件名（前端传入，必须严格匹配
	// <SN>_CFG.{xml,nv}）。SerialNumber 由文件名解析得到，无需另传。
	FileName string
	// Content 是文件全字节内容。注意：手动导入路径走完整字节流以便算 md5
	// 与 PutObject ContentLength，因此调用方需把 multipart part 读完整。
	Content []byte
}

// SnapshotImportFailure 描述单文件导入失败的结构化原因，供前端逐项展示。
type SnapshotImportFailure struct {
	FileName string `json:"file_name"`
	// SerialNumber 在文件名解析失败时为空。
	SerialNumber string `json:"serial_number,omitempty"`
	ErrorCode    string `json:"error_code"`
	Message      string `json:"message"`
}

// SnapshotImportResult 是 ImportFromUpload 的批量结果。
type SnapshotImportResult struct {
	Succeeded []string                `json:"succeeded"`
	Failed    []SnapshotImportFailure `json:"failed"`
}

// Sentinel error codes returned in SnapshotImportFailure.ErrorCode.
const (
	ImportErrInvalidName   = "INVALID_FILE_NAME"
	ImportErrEmptyBody     = "EMPTY_FILE_BODY"
	ImportErrPutObject     = "MINIO_PUT_FAILED"
	ImportErrUpsert        = "DB_UPSERT_FAILED"
	ImportErrUnknownDevice = "UNKNOWN_DEVICE"
)

// SnapshotService 是 ConfigSnapshot 的对外门面。
type SnapshotService struct {
	repo         SnapshotRepository
	mover        SnapshotMover
	fileLookup   SnapshotBackupFileLookup
	deviceLookup SnapshotDeviceLookup
	bucket       string
	logger       *zap.Logger

	// #61 promote 解码管线：源对象读取器 + 可选解密器。
	// SetPromoteDecoder 注入；未注入时 PromoteFromBackup 返回 ErrPromoteNotConfigured。
	sourceReader SnapshotSourceReader
	decryptor    Encryptor
	admission    storageprotection.WriteAdmission
}

func (s *SnapshotService) SetStorageAdmission(admission storageprotection.WriteAdmission) {
	s.admission = admission
}

// SetPromoteDecoder 注入 promote 路径的源对象读取器与（可选）解密器（#61）。
//
//   - reader 必须非 nil 才能 promote（读源备份对象字节）。
//   - decryptor 为 nil 时只能 promote 未加密备份；遇到带 .enc 的源对象会拒绝
//     （拒绝写一个不可解密的坏快照，正是 #61 要消除的失败模式）。
//
// 生产在 modules.go 用 *minio.Client 适配 reader + AES-256-GCM Encryptor 注入。
func (s *SnapshotService) SetPromoteDecoder(reader SnapshotSourceReader, decryptor Encryptor) {
	s.sourceReader = reader
	s.decryptor = decryptor
}

// NewSnapshotService 装配依赖。bucket 留空时回退到 SnapshotBucketDefault。
//
// deviceLookup 与 fileLookup 都允许为 nil：
//   - deviceLookup=nil：快照行不带 enb_name / product_type（仍可正常使用）
//   - fileLookup=nil  ：禁用 PromoteFromBackup（直接返回 ErrNotConfigured）
//
// mover 必须非 nil；nil 直接 panic（启动期问题应尽早暴露）。
func NewSnapshotService(
	repo SnapshotRepository,
	mover SnapshotMover,
	fileLookup SnapshotBackupFileLookup,
	deviceLookup SnapshotDeviceLookup,
	bucket string,
	logger *zap.Logger,
) *SnapshotService {
	if repo == nil {
		panic("SnapshotService: repo is required")
	}
	if mover == nil {
		panic("SnapshotService: mover is required")
	}
	if bucket == "" {
		bucket = SnapshotBucketDefault
	}
	return &SnapshotService{
		repo:         repo,
		mover:        mover,
		fileLookup:   fileLookup,
		deviceLookup: deviceLookup,
		bucket:       bucket,
		logger:       logger.Named("config-snapshot"),
	}
}

// ErrPromoteNotConfigured 标识 PromoteFromBackup 在 fileLookup 未注入时被调用。
// FilePathRecorder 处理时降级为 warn 而非 error。
var ErrPromoteNotConfigured = errors.New("snapshot promote not configured: fileLookup not wired")

// ─────────────────────────────────────────────────────────────────────────
// 1) Promote — 备份链路自动写
// ─────────────────────────────────────────────────────────────────────────

// PromoteFromBackup 在备份任务完成、文件已落 config_backup bucket 之后被
// FilePathRecorder 调用。
//
// 流程（#61 改造 —— 由 server-side CopyObject 换名复制，改为解码成明文再写）：
//  1. 从 backup_restore_file 取该 (sn, task_id) 对应的最新一行得到源 bucket/path
//  2. 读源对象完整字节（capped）
//  3. decodeToPlaintext：按 object_path 后缀解密(.enc) + 解压(.gz/.zst/.lz4/.bz2)
//     还原成"CPE 实际应拿到的明文配置"，并得到规范扩展名（xml/nv）
//  4. 以明文计算 MD5、PutObject 写到 (snapshotBucket, <SN>_CFG.<ext>)
//  5. 若 deviceLookup 可用，补 enb_name / product_type
//  6. Upsert config_snapshots（source=backup, source_task_id，MD5=明文哈希）
//
// 为何不再 CopyObject（#61）：源备份对象在开启压缩/加密时是 .gz/.enc 字节，且
// AEAD 的 AAD 绑定到源 basename；换名复制会丢后缀（下载侧据后缀决定是否解压/解密）
// 且 AAD 错配，导致按快照恢复时 CPE 收到的是不可用的密文/压缩流。统一存明文后，
// 按快照恢复（ACS 下载明文直发）、运维 presigned 下载都拿到可用配置，
// config_snapshots.MD5 也回归"明文配置指纹"语义（与手动导入 ImportFromUpload 一致）。
//
// 行为契约：
//   - 源文件不存在（fileLookup 返回 0 行）→ 返回 ErrNotFound（FilePathRecorder 降级 warn）
//   - 解码管线未注入 / 读源 / 解密 / 解压 / PutObject 失败 → 不写 DB；调用方 metric+warn
//   - DB Upsert 失败 → 不补偿删 MinIO（下一次 Promote 会覆盖同 key）
func (s *SnapshotService) PromoteFromBackup(
	ctx context.Context, backupTaskID uuid.UUID, deviceSN string,
) error {
	if deviceSN == "" {
		return ErrEmptySerialNumber
	}
	if s.fileLookup == nil {
		return ErrPromoteNotConfigured
	}
	if s.sourceReader == nil {
		// 解码管线未注入：拒绝走老的"原样复制"路径（那会产生不可用快照，#61）。
		return ErrPromoteNotConfigured
	}

	files, err := s.fileLookup.ListBySerial(ctx, deviceSN)
	if err != nil {
		return fmt.Errorf("lookup backup_restore_file: %w", err)
	}
	src := pickBackupFileForTask(files, backupTaskID)
	if src == nil {
		return fmt.Errorf("no backup file found for sn=%s task_id=%s: %w",
			deviceSN, backupTaskID, commonerrors.ErrNotFound)
	}

	srcBucket, srcPath, err := splitBucketAndPath(src.ObjectPath)
	if err != nil {
		// object_path 在写入时已经是 "bucket/key" 形态，这里出错属于数据损坏。
		return fmt.Errorf("parse backup_restore_file.object_path: %w", err)
	}

	blob, err := s.sourceReader.ReadObject(ctx, srcBucket, srcPath)
	if err != nil {
		return fmt.Errorf("read backup object %s/%s: %w", srcBucket, srcPath, err)
	}

	plaintext, canonicalName, err := s.decodeToPlaintext(srcPath, blob)
	if err != nil {
		return fmt.Errorf("decode backup object %s/%s: %w", srcBucket, srcPath, err)
	}

	dstName, ext, err := NormalizeSnapshotFileName(canonicalName, deviceSN)
	if err != nil {
		return fmt.Errorf("normalize file name (decoded=%q): %w", canonicalName, err)
	}
	dstPath := SnapshotObjectPath(deviceSN, ext)

	if err := s.checkStorageAdmission(ctx, storageprotection.WriteScopeBackup); err != nil {
		return err
	}
	if _, err := s.mover.PutObject(ctx,
		s.bucket, dstPath,
		bytes.NewReader(plaintext), int64(len(plaintext)),
		minio.PutObjectOptions{ContentType: contentTypeFor(ext)},
	); err != nil {
		return fmt.Errorf("minio PutObject snapshot %s/%s: %w", s.bucket, dstPath, err)
	}

	sum := md5.Sum(plaintext)
	md5Hex := hex.EncodeToString(sum[:])
	snap := &ConfigSnapshot{
		SerialNumber: deviceSN,
		FileName:     dstName,
		FileExt:      ext,
		ObjectBucket: s.bucket,
		ObjectPath:   dstPath,
		MD5:          &md5Hex,
		FileSize:     int64(len(plaintext)),
		Source:       SnapshotSourceBackup,
		SourceTaskID: cloneUUIDPtr(backupTaskID),
	}
	s.applyDeviceFields(ctx, snap, deviceSN)

	if err := s.repo.Upsert(ctx, snap); err != nil {
		return fmt.Errorf("upsert config_snapshot: %w", err)
	}

	s.logger.Info("snapshot promoted from backup",
		zap.String("sn", deviceSN),
		zap.String("task_id", backupTaskID.String()),
		zap.String("file_name", dstName),
		zap.String("bucket", s.bucket),
		zap.Int64("plaintext_size", int64(len(plaintext))),
	)
	return nil
}

// decodeToPlaintext 把存盘的备份对象（可能信封加密 + 压缩）还原为"CPE 实际应
// 拿到的明文配置字节"，并返回规范文件名（后缀已剥离）。镜像 ACS 下载管线
// （acs/download/handler.go 的 maybeDecrypt + detectCompression），保证快照桶里
// 的对象就是恢复时直发 CPE 的字节（#61）。
//
//   - 解密：当对象名以解密器扩展名（.enc）结尾时，AAD = 去掉 .enc 后的 basename，
//     与上传/下载契约完全一致（acs/upload encAAD、acs/download aad）。
//   - 解压：解密后若仍带 .gz/.zst/.lz4/.bz2 则解压。
//   - 带 .enc 但未注入解密器 → 报错（拒绝写不可解密的坏快照）。
func (s *SnapshotService) decodeToPlaintext(srcKey string, blob []byte) (plaintext []byte, canonicalName string, err error) {
	name := srcKey
	work := blob

	switch {
	case s.decryptor != nil && strings.HasSuffix(name, "."+s.decryptor.Extension()):
		inner := strings.TrimSuffix(name, "."+s.decryptor.Extension())
		aad := []byte(filepath.Base(inner))
		pt, derr := s.decryptor.Decrypt(work, aad)
		if derr != nil {
			return nil, "", fmt.Errorf("decrypt backup object: %w", derr)
		}
		work = pt
		name = inner
	case s.decryptor == nil && strings.HasSuffix(name, ".enc"):
		// 源是加密对象但没有解密器：明确拒绝，而不是把密文当明文写快照。
		return nil, "", fmt.Errorf("encrypted backup object %q but no decryptor configured: %w",
			srcKey, ErrEncryptionKeyUnavailable)
	}

	if format, stripped, ok := stripCompressionSuffix(name); ok {
		pt, derr := decompress(format, work, snapshotPromoteMaxBytes)
		if derr != nil {
			return nil, "", fmt.Errorf("decompress backup object: %w", derr)
		}
		work = pt
		name = stripped
	}

	return work, filepath.Base(name), nil
}

// pickBackupFileForTask 在 ListBySerial 的结果中挑选与 backupTaskID 匹配的那行。
// 若任务 ID 在所有行的 task_id 中都不命中，回退到最新一行（ListBySerial 已按
// update_time DESC 排序）—— 这覆盖旧链路 task_id 写入晚于 file_name 的极端时序。
func pickBackupFileForTask(files []BackupRestoreFile, backupTaskID uuid.UUID) *BackupRestoreFile {
	if len(files) == 0 {
		return nil
	}
	want := backupTaskID.String()
	for i := range files {
		if files[i].TaskID != nil && *files[i].TaskID == want {
			return &files[i]
		}
	}
	return &files[0]
}

// ─────────────────────────────────────────────────────────────────────────
// 2) Import — 手动批量上传
// ─────────────────────────────────────────────────────────────────────────

// ImportFromUpload 接收已读完字节的多个文件，逐项校验、上传 MinIO、Upsert DB。
//
// 单项失败不影响其他项；返回结构化的 succeeded + failed 列表，调用方按需展示。
// 任一项 DB 写失败时**自动补偿删 MinIO 对象**，避免孤儿。
func (s *SnapshotService) ImportFromUpload(
	ctx context.Context, items []SnapshotImportItem, uploadBy string,
) (*SnapshotImportResult, error) {
	out := &SnapshotImportResult{
		Succeeded: make([]string, 0, len(items)),
		Failed:    make([]SnapshotImportFailure, 0),
	}
	for _, item := range items {
		fail := s.importOne(ctx, item, uploadBy)
		if fail != nil {
			out.Failed = append(out.Failed, *fail)
			continue
		}
		// 文件名解析成功保证 SN 提取也成功，importOne 内部已做 Upsert。
		sn, _, _ := ValidateImportFileName(item.FileName, "")
		out.Succeeded = append(out.Succeeded, sn)
	}
	return out, nil
}

// importOne 是 ImportFromUpload 单文件处理；成功返回 nil，失败返回结构化原因。
func (s *SnapshotService) importOne(
	ctx context.Context, item SnapshotImportItem, uploadBy string,
) *SnapshotImportFailure {
	sn, ext, err := ValidateImportFileName(item.FileName, "")
	if err != nil {
		return &SnapshotImportFailure{
			FileName:  item.FileName,
			ErrorCode: ImportErrInvalidName,
			Message:   err.Error(),
		}
	}
	if len(item.Content) == 0 {
		return &SnapshotImportFailure{
			FileName:     item.FileName,
			SerialNumber: sn,
			ErrorCode:    ImportErrEmptyBody,
			Message:      "上传内容为空",
		}
	}
	objectPath := SnapshotObjectPath(sn, ext)
	sum := md5.Sum(item.Content)
	md5Hex := hex.EncodeToString(sum[:])

	if err := s.checkStorageAdmission(ctx, storageprotection.WriteScopeBackup); err != nil {
		return &SnapshotImportFailure{
			FileName:     item.FileName,
			SerialNumber: sn,
			ErrorCode:    ImportErrPutObject,
			Message:      err.Error(),
		}
	}
	if _, err := s.mover.PutObject(ctx,
		s.bucket, objectPath,
		bytes.NewReader(item.Content), int64(len(item.Content)),
		minio.PutObjectOptions{ContentType: contentTypeFor(ext)},
	); err != nil {
		return &SnapshotImportFailure{
			FileName:     item.FileName,
			SerialNumber: sn,
			ErrorCode:    ImportErrPutObject,
			Message:      err.Error(),
		}
	}

	snap := &ConfigSnapshot{
		SerialNumber: sn,
		FileName:     SnapshotObjectPath(sn, ext), // 与 objectPath 同形（<SN>_CFG.<ext>）
		FileExt:      ext,
		ObjectBucket: s.bucket,
		ObjectPath:   objectPath,
		MD5:          &md5Hex,
		FileSize:     int64(len(item.Content)),
		Source:       SnapshotSourceManualUpload,
		UpdateBy:     stringPtrOrNil(uploadBy),
	}
	s.applyDeviceFields(ctx, snap, sn)

	if err := s.repo.Upsert(ctx, snap); err != nil {
		// 补偿：删除刚 Put 的对象，避免孤儿
		if rmErr := s.mover.RemoveObject(ctx, s.bucket, objectPath, minio.RemoveObjectOptions{}); rmErr != nil {
			s.logger.Warn("compensating RemoveObject after upsert failure",
				zap.String("sn", sn), zap.String("object", objectPath), zap.Error(rmErr))
		}
		return &SnapshotImportFailure{
			FileName:     item.FileName,
			SerialNumber: sn,
			ErrorCode:    ImportErrUpsert,
			Message:      err.Error(),
		}
	}

	s.logger.Info("snapshot imported manually",
		zap.String("sn", sn),
		zap.String("file_name", snap.FileName),
		zap.Int64("size", snap.FileSize),
		zap.String("upload_by", uploadBy),
	)
	return nil
}

func (s *SnapshotService) checkStorageAdmission(ctx context.Context, scope storageprotection.WriteScope) error {
	if s.admission == nil {
		return nil
	}
	decision, err := s.admission.CheckPath(ctx, storageprotection.ProtectedPathIDMinIO, scope)
	if err != nil {
		return fmt.Errorf("storage admission check: %w", err)
	}
	if !decision.Allowed {
		return fmt.Errorf("storage write protected: %s", decision.Reason)
	}
	return nil
}

func contentTypeFor(ext string) string {
	switch ext {
	case string(SnapshotExtXML):
		return "application/xml"
	default:
		return "application/octet-stream"
	}
}

// ─────────────────────────────────────────────────────────────────────────
// 3) Read — 查询
// ─────────────────────────────────────────────────────────────────────────

// GetBySerialNumber 代理 repo。
func (s *SnapshotService) GetBySerialNumber(ctx context.Context, sn string) (*ConfigSnapshot, error) {
	return s.repo.GetBySerialNumber(ctx, sn)
}

// ValidateDeviceSNs 批量校验 SN 在 devices 表中是否存在（T-0164 导入校验）。
// 给前端 ImportDrawer 在拖入文件时立即标红"未知 SN"的项。
// deviceLookup 未注入时所有 SN 都视为 existing（降级，避免误拦）。
func (s *SnapshotService) ValidateDeviceSNs(
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
		// err 非 nil 视为"暂时无法确认"，降级为 existing 避免误拦。
		existing = append(existing, sn)
	}
	return existing, missing
}

// BatchGetBySerialNumbers 代理 repo。
func (s *SnapshotService) BatchGetBySerialNumbers(
	ctx context.Context, sns []string,
) (map[string]*ConfigSnapshot, error) {
	return s.repo.BatchGetBySerialNumbers(ctx, sns)
}

// List 代理 repo。
func (s *SnapshotService) List(
	ctx context.Context, filter SnapshotFilter,
) ([]ConfigSnapshot, int64, error) {
	return s.repo.List(ctx, filter)
}

// ─────────────────────────────────────────────────────────────────────────
// 4) Delete
// ─────────────────────────────────────────────────────────────────────────

// Delete 删除单设备快照行 + MinIO 对象。
// MinIO 删除失败仅 warn（"DB 是真相"），DB 删除失败返回 error。
func (s *SnapshotService) Delete(ctx context.Context, sn string) error {
	if sn == "" {
		return ErrEmptySerialNumber
	}
	existing, err := s.repo.GetBySerialNumber(ctx, sn)
	if err != nil {
		return fmt.Errorf("get config_snapshot before delete: %w", err)
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

// BatchDelete 删除多设备快照。逐个调 Delete 以保证 MinIO 与 DB 一致。
// 返回成功删除 / 失败 SN 列表（失败为 DB 删除阶段失败）。
func (s *SnapshotService) BatchDelete(
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
// 5) helpers
// ─────────────────────────────────────────────────────────────────────────

// applyDeviceFields 在 Upsert 之前把 enb_name / product_type / source_version
// 填进快照行。deviceLookup=nil 或查询失败时静默跳过 —— 这些列允许为 NULL。
func (s *SnapshotService) applyDeviceFields(ctx context.Context, snap *ConfigSnapshot, sn string) {
	if s.deviceLookup == nil {
		return
	}
	dev, err := s.deviceLookup.GetBySerialNumber(ctx, sn)
	if err != nil || dev == nil {
		return
	}
	if dev.DeviceName != "" {
		name := dev.DeviceName
		snap.EnbName = &name
	}
	if dev.ProductClass != "" {
		pc := dev.ProductClass
		snap.ProductType = &pc
	}
	// #70：记录快照捕获时的设备软件版本，作为后续按快照恢复的跨版本检查源版本。
	if dev.FirmwareVersion != "" {
		fw := dev.FirmwareVersion
		snap.SourceVersion = &fw
	}
}

func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func cloneUUIDPtr(id uuid.UUID) *uuid.UUID {
	out := id
	return &out
}
