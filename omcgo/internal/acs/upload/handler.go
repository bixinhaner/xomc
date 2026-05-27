package upload

import (
	"bytes"
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/backup"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.uber.org/zap"
)

// Handler handles HTTP file upload requests from CPE devices.
// Endpoint: POST /smallcell/FileUploadService?fileType=PM&filename=xxx
// Authentication: HTTP Basic Auth with global credentials from config
type Handler struct {
	tokenManager *TokenManager
	sessionStore *SessionStore
	minioClient  *minio.Client
	maxFileSize  int64
	buckets      appconfig.BucketConfig
	eventBus     event.EventBus
	logger       *zap.Logger
	// Global credentials for upload authentication
	username        string
	password        string
	runtimeProvider transfercfg.Provider
	// T-0074: optional backup compression hooks. When both fields are set and
	// the inbound file_type is FileTypeConfig with policy.EnableCompression=true,
	// the body stream is wrapped with the configured compressor before MinIO
	// PutObject. Both fields are nil-safe — a nil getter disables compression.
	policyGetter       backup.PolicyGetter
	compressionMetrics *backup.PolicyMetrics
	// T-0075: optional encryptor for AES-256-GCM envelope encryption applied
	// after the compression wrap. nil-safe: nil disables encryption (the
	// pre-T-0075 plaintext-or-compressed pipeline). When wired AND policy.
	// EnableEncryption=true, ServeHTTP buffers the (possibly compressed) body
	// (up to 64MB), encrypts in-memory, appends ".enc" to the object path.
	encryptor backup.Encryptor
}

// NewHandler creates a new upload Handler.
func NewHandler(
	tokenManager *TokenManager,
	sessionStore *SessionStore,
	minioClient *minio.Client,
	maxFileSize int64,
	buckets appconfig.BucketConfig,
	username, password string,
	eventBus event.EventBus,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		tokenManager: tokenManager,
		sessionStore: sessionStore,
		minioClient:  minioClient,
		maxFileSize:  maxFileSize,
		buckets:      buckets,
		username:     username,
		password:     password,
		eventBus:     eventBus,
		logger:       logger,
	}
}

func (h *Handler) SetRuntimeProvider(provider transfercfg.Provider) {
	h.runtimeProvider = provider
}

// ServeHTTP handles upload requests.
// Route: POST /smallcell/FileUploadService?fileType={type}&filename={name}
// Auth: HTTP Basic Authentication with global credentials
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Allow POST and PUT (TR-069 specifies PUT for Upload, some CPEs use POST)
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		h.logger.Warn("upload rejected: unsupported method", zap.String("method", r.Method), zap.String("remote_addr", r.RemoteAddr))
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 2. Validate Basic Auth credentials (skip if no credentials configured)
	runtimeCfg := h.currentSettings(r.Context())
	if runtimeCfg.Username != "" {
		username, password, ok := r.BasicAuth()
		if !ok {
			h.logger.Warn("missing basic auth credentials")
			w.Header().Set("WWW-Authenticate", `Basic realm="FileUpload"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if subtle.ConstantTimeCompare([]byte(username), []byte(runtimeCfg.Username)) != 1 ||
			subtle.ConstantTimeCompare([]byte(password), []byte(runtimeCfg.Password)) != 1 {
			h.logger.Warn("invalid upload credentials",
				zap.String("username", username),
			)
			w.Header().Set("WWW-Authenticate", `Basic realm="FileUpload"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// 3. Extract fileType and filename from query params.
	// FAULT_LOG_COLLECT (SPV 触发) 走厂商私有 URL 模板：
	//   /FileUploadService?fileType=RL&id={id}&sn={sn}&fileName=
	// 与现网 Upload RPC 链路的 `taskId` / `filename` 大小写不同，下面统一兜底。
	fileType := r.URL.Query().Get("fileType")
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		filename = r.URL.Query().Get("fileName")
	}

	if fileType == "" {
		http.Error(w, "missing fileType parameter", http.StatusBadRequest)
		return
	}

	// filename 留空是合法路径：厂商真实样本里 URL 末尾 `&filename=` 都是空的——
	// 设备自身决定上传时的文件名（裸 binary PUT，无 multipart envelope）。
	// 服务端按业务规则生成与 UFTE target_file_name_template 一致的命名：
	//   - 备份类（FileType "3" / CONFIGBACKUP_*）：backup-{taskId8}-{sn}.nv|xml
	//   - 日志类（FileType "6"/"8" Vendor Log）：log-{taskId8}-{sn}.tar.gz
	//   - 其它（默认兜底）：upload-{taskId8}-{sn}-{ts}
	// 保证 ACS 落地名与 UFTE 端 DeviceItem.TargetFile 渲染结果一致——后续
	// fileLandedLookup / downloadURLLookup 用 (sn, target_file) 反查能命中。
	// FAULT_LOG_COLLECT 用 ?id=<task_uuid>，其他链路用 ?taskId=<task_uuid>；取 id 兜底 taskId。
	queryTaskID := r.URL.Query().Get("taskId")
	if queryTaskID == "" {
		queryTaskID = r.URL.Query().Get("id")
	}
	if filename == "" {
		snQ := r.URL.Query().Get("sn")
		if queryTaskID == "" || snQ == "" {
			http.Error(w, "missing filename, and cannot derive: taskId/sn query params also empty", http.StatusBadRequest)
			return
		}
		filename = deriveUploadFilename(fileType, queryTaskID, snQ)
		h.logger.Info("derived filename from sn+taskId (URL filename was empty)",
			zap.String("file_type", fileType),
			zap.String("sn", snQ),
			zap.String("task_id", queryTaskID),
			zap.String("derived_filename", filename),
		)
	}

	// 3.1 Path traversal protection: strip directory components and reject suspicious filenames
	filename = filepath.Base(filename)
	if filename == "." || filename == ".." || strings.Contains(filename, "..") {
		h.logger.Warn("path traversal attempt blocked", zap.String("filename", r.URL.Query().Get("filename")))
		http.Error(w, "invalid filename", http.StatusBadRequest)
		return
	}

	// 4. Check file size
	if runtimeCfg.MaxFileSize > 0 && r.ContentLength > runtimeCfg.MaxFileSize {
		http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
		return
	}

	// 5. Determine bucket and object path
	ft := normalizeFileType(fileType)
	bucket, category := storage.BucketAndCategory(ft, h.buckets)
	now := time.Now()
	// 配置备份类（FileType=3 / CONFIGBACKUP_*）和故障日志类（FileType=8 / RL）加 taskId8
	// 目录层级，避免同设备不同任务上传同名文件互相覆盖：
	//   - 配置备份：厂商如 baicells 每次都用 "mib-home-fap.nv" 名
	//   - 故障日志：FAULT_LOG_COLLECT 同一设备可能在多个采集任务中各采集一次，
	//     CPE 自己生成的文件名相同时会被后写者覆盖
	// 其它类型（PM/MR/RunningLog/Firmware）保持原路径——它们本身命名带时间戳
	// 或唯一标识，不存在重名问题。
	taskSubdir := ""
	if ft == tr069.FileTypeConfig || ft == tr069.FileTypeFaultLog {
		if tid := strings.ReplaceAll(queryTaskID, "-", ""); len(tid) >= 8 {
			taskSubdir = tid[:8] + "/"
		}
	}
	var objectPath string
	// F05 MR 走专用路径 {deviceSN}/{filename}（桶名 mr-files 自带模块归属，无需日期层级）。
	// SN 优先取 ?sn=，回退 ?cellCode=（dispatcher 拼的 MrUrl 只带 cellCode 没 sn，
	// 而 MR 任务里 cellCode == 设备 SN，CellTarget 复用 SerialNumber）。
	// 不回退的话所有 MR 文件都进 mr-files/unknown/ 桶子目录，跟前端按 SN 聚合 / 下载链路对不上。
	if ft == tr069.FileTypeMR {
		querySN := r.URL.Query().Get("sn")
		if querySN == "" {
			querySN = r.URL.Query().Get("cellCode")
		}
		if querySN == "" {
			querySN = "unknown"
		}
		objectPath = storage.MRObjectPath(querySN, filename)
	} else if category != "" {
		objectPath = fmt.Sprintf("%s/%s/%s%s", category, now.Format("2006/01/02"), taskSubdir, filename)
	} else {
		objectPath = fmt.Sprintf("%s/%s%s", now.Format("2006/01/02"), taskSubdir, filename)
	}

	// 6. Stream upload to MinIO. For FileTypeConfig (TR-069 "3" Vendor
	// Configuration File = backup), optionally wrap the body in a streaming
	// compressor per BackupPolicy (T-0074).
	//
	// Note: r.Body is owned by net/http; the server closes it on handler
	// return. We do NOT close r.Body explicitly here — the compressor's pump
	// goroutine reads from a counting wrapper around r.Body, and ctx
	// cancellation (handler return) interrupts the pump.
	ctx := r.Context()

	body := io.Reader(r.Body)
	contentLength := r.ContentLength
	uploadOpts := minio.PutObjectOptions{ContentType: "application/octet-stream"}

	cmp := h.maybeWrapForCompression(ctx, ft, r.Body)
	if cmp.applied {
		defer cmp.body.Close()
		body = cmp.body
		contentLength = -1 // streaming, compressed size unknown
		objectPath += cmp.ext
		// ContentEncoding documents the on-disk compression so the restore
		// side (T-0072) can read it from object metadata as a backup signal
		// to the .gz/.zst filename suffix.
		uploadOpts.ContentEncoding = cmp.format
	}

	// T-0075: encryption layer (after compression). Buffers fully into memory
	// up to 64MB (encMaxPlaintext); rejects oversize uploads.
	//
	// AAD must equal the on-disk basename minus the .enc suffix so that the
	// download handler — which only knows the MinIO object path — can
	// reconstruct the same value. When compression is active the on-disk
	// name is `<filename>.<cmp.ext>.enc`, so AAD = filename+cmp.ext. Without
	// compression AAD = filename. This binding survives MinIO-level rename
	// attacks (review HIGH-1 fix).
	encApplied := false
	if h.encryptor != nil && ft == tr069.FileTypeConfig && h.policyGetter != nil {
		pol, perr := h.policyGetter.Get(ctx)
		if perr == nil && pol != nil && pol.EnableEncryption && pol.EncryptionAlgorithm == "AES-256-GCM" {
			encAAD := filename
			if cmp.applied {
				encAAD = filename + cmp.ext
			}
			encryptedBlob, encErr := h.encryptUpload(body, encAAD)
			if encErr != nil {
				// Fail closed: never fall through to plaintext when policy
				// asked for encryption — that would silently weaken security.
				h.compressionMetrics.RecordBackupEncryptionError(classifyEncryptError(encErr))
				h.logger.Error("backup encryption failed; aborting upload",
					zap.Error(encErr), zap.String("filename", filename))
				status := http.StatusInternalServerError
				if errors.Is(encErr, backup.ErrEncryptionInputTooLarge) {
					status = http.StatusRequestEntityTooLarge
				}
				http.Error(w, "encryption failed", status)
				return
			}
			body = bytes.NewReader(encryptedBlob)
			contentLength = int64(len(encryptedBlob))
			objectPath += "." + h.encryptor.Extension()
			// ContentEncoding chains: e.g. "gzip+aes-256-gcm".
			if uploadOpts.ContentEncoding != "" {
				uploadOpts.ContentEncoding += "+" + h.encryptor.Format()
			} else {
				uploadOpts.ContentEncoding = h.encryptor.Format()
			}
			encApplied = true
		}
	}

	startUpload := time.Now()
	info, err := h.minioClient.PutObject(ctx, bucket, objectPath, body, contentLength, uploadOpts)
	if err != nil {
		// Do NOT record this as a compression error: the failure could be
		// MinIO-side (network, auth, bucket missing). Compression-internal
		// errors propagate through the pipe to PutObject as body-read errors,
		// but distinguishing them at this layer is unreliable. Restrict the
		// "copy" reason to genuine compression-stream issues; track upload
		// failures via existing logging.
		h.logger.Error("upload to minio failed",
			zap.Error(err),
			zap.String("file_type", fileType),
			zap.String("path", objectPath),
		)
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}
	if cmp.applied {
		// pump goroutine has finished by the time PutObject returns (EOF
		// propagation through the pipe), so atomic.Int64 reads are safe.
		h.compressionMetrics.RecordCompressionBytes(cmp.format, cmp.bytesIn(), info.Size)
		h.compressionMetrics.RecordCompressionDuration(cmp.format, time.Since(startUpload).Seconds())
	}
	if encApplied {
		h.compressionMetrics.RecordBackupEncrypted()
	}

	h.logger.Info("file uploaded",
		zap.String("file_type", fileType),
		zap.String("filename", filename),
		zap.String("path", objectPath),
		zap.Int64("size", info.Size),
		zap.String("compression", cmp.format),
	)

	// 6.1. For parameter model uploads (FileType "11"), publish event for processing.
	if ft == tr069.FileTypeDataModel && h.eventBus != nil {
		h.publishDataModelEvent(ctx, bucket, objectPath, filename, info.Size)
	}

	// 6.2. T-0079: For backup config uploads (FileType "3"), publish
	// `backup.file.received` so the backup module can write
	// backup_tasks.file_path. Decoupled via EventBus to keep the ACS process
	// from importing backup module directly.
	if ft == tr069.FileTypeConfig && h.eventBus != nil {
		// 把 URL query 里的 sn / taskId 也传进去 —— 设备实际 PUT 时可能用自己内部
		// NV 文件名（如 baicells/MMMM 系列固件回传的 "mib-home-fap.nv"），文件名
		// 不匹配 backup-{taskId8}-{sn}.{ext} 模板时 parseBackupFilename 失效，
		// 此时回退到 URL query 兜底是唯一可靠路径。
		h.publishBackupFileReceivedEvent(ctx, bucket, objectPath, filename, info.Size, info.ETag,
			r.URL.Query().Get("sn"), queryTaskID)
	}

	// 6.3. For station log uploads (FileType "6" running log, "8" fault log),
	// publish log.file.received so the stationlog module can record the file
	// and enforce quotas.
	//
	// 同时发 backup.file.received —— stationlog 模块还没落地（station_log_files
	// 表不存在 + NATS 流未注册），但前端 UFTE 设备列表的「文件名 / 下载」UI 已经
	// 走通了基于 backup_restore_file 表的反查链路。让 LOG 上传也写一行通用元数据
	// 到 backup_restore_file，前端复用现有反查 + presigned URL 下载逻辑，
	// 不阻塞业务（用户能看到文件名 + 下载）。等 stationlog 模块上线后这里可以收口
	// 到单一事件，但当前阶段同时双发更稳。
	if (ft == tr069.FileTypeRunningLog || ft == tr069.FileTypeFaultLog) && h.eventBus != nil {
		h.publishLogFileReceivedEvent(ctx, bucket, objectPath, filename, string(ft), info.Size)
		h.publishBackupFileReceivedEvent(ctx, bucket, objectPath, filename, info.Size, info.ETag,
			r.URL.Query().Get("sn"), queryTaskID)
	}

	// 6.4. T-0164 G1 真机闭环修复点：FileType=PM (4) 文件入库后发 pm.file.received，
	// pm.Collector 订阅后解析 XML 写 pm_metrics / 触发 KPI 反算。
	// 此前只有 transfer-bridge 路径（订阅 AutonomousTransferComplete + 下载文件）
	// 会发这个事件，CPE 直接 HTTP POST 路径不经过 bridge，导致 pm_metrics 永远空。
	//
	// SN 提取优先级：
	//   1. URL query `sn=`（cpe_simulator.py 模板带；部分厂商私有实现也带）
	//   2. 文件名兜底（真机 Baicells 实测：`A{ts}_{OUI}.{SN}.xml(.gz)?`，URL 不带 sn）
	//
	// payload 走精简版（minio_path / bucket / device_sn / file_size / file_name），
	// 设备 UUID / OUI / carrier / technology 由 collector 用 SN 查 device 表回填。
	if ft == tr069.FileTypePM && h.eventBus != nil {
		deviceSN := r.URL.Query().Get("sn")
		if deviceSN == "" {
			deviceSN = extractDeviceSNFromPMFilename(filename)
		}
		h.publishPMFileReceivedEvent(ctx, bucket, objectPath, filename, info.Size, deviceSN)
	}

	// 6.5. F05 MR Task: publish mr.file.uploaded with cellCode so the mr/task
	// HeartbeatSubscriber can set Redis MRFileReport_{cellCode} TTL and bump
	// PG last_heartbeat. cellCode comes from the URL query that OMC put into
	// the device's MrUrl when opening the task.
	if ft == tr069.FileTypeMR && h.eventBus != nil {
		// MR 任务模型里 cellCode == 设备 SN（CellTarget.SmallCellCode 复用 SerialNumber）。
		// dispatcher 拼的 MrUrl 只带 ?cellCode=...&filename=（没 ?sn=），设备 PUT 上来时
		// URL query 也只有 cellCode。所以 SN 优先取 ?sn= 显式参数，空时回退 ?cellCode=。
		// 不回退的话 mr.file.received payload device_sn 永远空 → worker Collector
		// "missing device_id and no DeviceLookup wired" 报错 → NATS 重试 5 次后丢弃。
		mrSN := r.URL.Query().Get("sn")
		if mrSN == "" {
			mrSN = r.URL.Query().Get("cellCode")
		}
		h.publishMRFileUploadedEvent(ctx, bucket, objectPath, filename,
			r.URL.Query().Get("cellCode"), mrSN, info.Size)
		// 同时发 mr.file.received，让 mr.Collector 走"下载 + 解析 + 入库"链路。
		// device_id 留空，由 collector 注入的 DeviceLookup 按 SN 反查。
		h.publishMRFileReceivedEvent(ctx, bucket, objectPath, filename,
			mrSN, info.Size)
	}

	// 7. Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","path":"%s","size":%d}`, objectPath, info.Size)
}

func (h *Handler) currentSettings(ctx context.Context) transfercfg.UploadSettings {
	if h.runtimeProvider != nil {
		return h.runtimeProvider.Snapshot(ctx).Upload
	}
	return transfercfg.UploadSettings{
		Username:    h.username,
		Password:    h.password,
		MaxFileSize: h.maxFileSize,
	}
}

// normalizeFileType converts the fileType query parameter to a tr069.FileType.
// Handles both numeric codes ("4") and text aliases ("PM", "CONFIGBACKUP_XML", etc.).
func normalizeFileType(raw string) tr069.FileType {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "1":
		return tr069.FileTypeFirmware
	case "2":
		return tr069.FileTypePatch
	case "3", "CONFIGBACKUP_XML", "CONFIGBACKUP_NV":
		// CONFIGBACKUP_XML / CONFIGBACKUP_NV 均为配置备份，路由到 config_backup bucket。
		// 区别仅在于 CPE 侧的文件格式（XML vs NV）；从 ACS 视角两者都是配置文件。
		return tr069.FileTypeConfig
	case "4", "PM":
		return tr069.FileTypePM
	case "5", "MR":
		return tr069.FileTypeMR
	case "6", "LOG":
		return tr069.FileTypeRunningLog
	case "7":
		return tr069.FileTypeSecurityLog
	case "8", "RL":
		// "RL" 是 FAULT_LOG_COLLECT 链路（SPV 触发，FaultLogURL 参数）使用的厂商私有标识，
		// 等价于 TR-069 标准 FileType "8"（异常 / 故障日志）。
		return tr069.FileTypeFaultLog
	case "9":
		return tr069.FileTypePCAP
	case "10":
		return tr069.FileTypeWeb
	case "11", "PARAMETER MODEL":
		return tr069.FileTypeDataModel
	case "SSL":
		return tr069.FileTypeSSLCert
	default:
		return tr069.FileTypeRunningLog
	}
}

// GetSession retrieves an upload session (for TC handler use).
func (h *Handler) GetSession(ctx context.Context, deviceSN, commandKey string) (*Session, error) {
	return h.sessionStore.Get(ctx, deviceSN, commandKey)
}

// DeleteSession removes an upload session.
func (h *Handler) DeleteSession(ctx context.Context, deviceSN, commandKey string) error {
	return h.sessionStore.Delete(ctx, deviceSN, commandKey)
}

// UploadCredentials returns the global upload credentials.
// Used by Upload RPC to include in the SOAP message.
func (h *Handler) UploadCredentials() (username, password string) {
	return h.username, h.password
}

// publishDataModelEvent publishes a datamodel.file.received event after a parameter model file is uploaded.
// The filename is expected to contain the device SN: "datamodel_{deviceSN}_{uuid}.xml"
func (h *Handler) publishDataModelEvent(ctx context.Context, bucket, objectPath, filename string, fileSize int64) {
	// Extract device SN from filename pattern: datamodel_{deviceSN}_{uuid}.xml
	deviceSN := extractDeviceSNFromFilename(filename)

	payload := map[string]interface{}{
		"minio_bucket": bucket,
		"minio_path":   objectPath,
		"device_sn":    deviceSN,
		"file_size":    fileSize,
		"filename":     filename,
	}

	evt, err := event.NewEvent(event.SubjectDataModelFileReceived, payload)
	if err != nil {
		h.logger.Error("create datamodel event", zap.Error(err))
		return
	}
	if err := h.eventBus.Publish(ctx, event.SubjectDataModelFileReceived, evt); err != nil {
		h.logger.Error("publish datamodel.file.received", zap.Error(err))
		return
	}
	h.logger.Info("published datamodel.file.received",
		zap.String("device_sn", deviceSN),
		zap.String("path", objectPath))
}

// SetEncryption wires the optional T-0075 backup encryptor. nil disables.
// Caller is responsible for constructing the Encryptor with a working
// KeyProvider (see backup.NewEncryptor + backup.NewEnvKeyProvider).
func (h *Handler) SetEncryption(enc backup.Encryptor) {
	h.encryptor = enc
}

// encryptUpload reads the (possibly compressed) body fully into memory up to
// the encryption ceiling, then runs Encrypt with the supplied AAD. Returns
// the fully-formed encrypted blob suitable for bytes.Reader → PutObject.
//
// aad must equal the on-disk basename minus the `.enc` suffix so the
// downloader can reconstruct it from the object path (review HIGH-1 fix).
func (h *Handler) encryptUpload(body io.Reader, aad string) ([]byte, error) {
	// Limit + 1 lets us detect overflow without truncating silently.
	limited := io.LimitReader(body, int64(64*1024*1024)+1)
	buf, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("buffer body for encryption: %w", err)
	}
	if len(buf) > 64*1024*1024 {
		return nil, backup.ErrEncryptionInputTooLarge
	}
	return h.encryptor.Encrypt(buf, []byte(aad))
}

// classifyEncryptError maps an encryption error to a coarse metric reason
// label. Unknown errors fall to "encrypt_fail".
func classifyEncryptError(err error) string {
	switch {
	case errors.Is(err, backup.ErrEncryptionKeyUnavailable):
		return "key_unavailable"
	case errors.Is(err, backup.ErrEncryptionInputTooLarge):
		return "oversize"
	case errors.Is(err, backup.ErrEncryptionFormatInvalid):
		return "format_invalid"
	default:
		return "encrypt_fail"
	}
}

// SetCompression wires backup compression dependencies into the handler.
// Both arguments may be nil to disable compression (the default state).
// T-0074: keeps NewHandler signature backward-compatible (same pattern as
// BackupExecutor.SetPolicyEnforcement).
func (h *Handler) SetCompression(getter backup.PolicyGetter, metrics *backup.PolicyMetrics) {
	h.policyGetter = getter
	h.compressionMetrics = metrics
}

// compressionWrap is the result of maybeWrapForCompression. When applied=false
// the upstream code paths take the original body untouched.
type compressionWrap struct {
	applied bool
	body    io.ReadCloser // wrapped reader; caller must Close
	ext     string        // ".gz" / ".zst", appended to object path
	format  string        // metric label (gzip|zstd)
	counter *countingReader
}

// bytesIn reports the plaintext byte count consumed so far. Safe to call
// concurrently with the pump goroutine — counter.n is atomic.Int64.
func (c compressionWrap) bytesIn() int64 {
	if c.counter == nil {
		return 0
	}
	return c.counter.n.Load()
}

// maybeWrapForCompression decides whether the inbound upload body should be
// streaming-compressed. Returns applied=false (and no error) when:
//   - file type is not FileTypeConfig (only backup files compress today), OR
//   - no policyGetter wired, OR
//   - policy lookup failed, OR
//   - policy.EnableCompression=false, OR
//   - NewCompressor / Wrap failed (recorded as metric, fall back to plaintext).
//
// The fall-back-on-failure choice is deliberate: backup is a high-availability
// feature; we prefer storing larger uncompressed bytes over failing the upload.
func (h *Handler) maybeWrapForCompression(ctx context.Context, ft tr069.FileType, src io.Reader) compressionWrap {
	if ft != tr069.FileTypeConfig || h.policyGetter == nil {
		return compressionWrap{}
	}
	pol, err := h.policyGetter.Get(ctx)
	if err != nil || pol == nil || !pol.EnableCompression {
		if err != nil {
			h.logger.Warn("backup policy lookup failed; uploading without compression",
				zap.Error(err))
		}
		return compressionWrap{}
	}
	// T-0077: lz4 + bzip2 are now real implementations; the earlier
	// "format not implemented; passing through" guard was removed.
	c, err := backup.NewCompressor(pol.CompressionFormat, pol.CompressionLevel)
	if err != nil {
		h.compressionMetrics.RecordCompressionError(pol.CompressionFormat, "open")
		h.logger.Warn("compressor construction failed; passing through",
			zap.String("format", pol.CompressionFormat),
			zap.Int("level", pol.CompressionLevel),
			zap.Error(err))
		return compressionWrap{}
	}
	counter := &countingReader{r: src}
	wrapped, err := c.Wrap(ctx, counter)
	if err != nil {
		h.compressionMetrics.RecordCompressionError(c.Format(), "open")
		h.logger.Warn("compressor wrap failed; passing through",
			zap.String("format", c.Format()),
			zap.Error(err))
		return compressionWrap{}
	}
	return compressionWrap{
		applied: true,
		body:    wrapped,
		ext:     c.Extension(),
		format:  c.Format(),
		counter: counter,
	}
}

// countingReader counts plaintext bytes consumed so the compression metric
// can compute compressed/raw ratio after PutObject completes. n is atomic
// because the pump goroutine writes it from inside io.Copy while the request
// goroutine reads it after PutObject returns; pipe close establishes a
// happens-before but the race detector does not always recognize that
// synchronization for ad-hoc int fields.
type countingReader struct {
	r io.Reader
	n atomic.Int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n.Add(int64(n))
	return n, err
}

// publishBackupFileReceivedEvent emits SubjectBackupFileReceived after a
// FileType=3 (Vendor Configuration File) upload lands in MinIO. The backup
// module subscribes to this and writes backup_tasks.file_path (T-0079). The
// filename is expected to follow the executor-generated pattern:
//
//	backup-{taskID8}-{deviceSN}.xml(.gz|.zst|.lz4|.bz2)?
//
// Filenames not matching the pattern still publish the event with empty
// backup_task_id_prefix; the subscriber treats that as "no-match skip" and
// won't error — this preserves operator-uploaded ad-hoc config files (rare).
//
// M1 of backup-restore-alignment-plan: 透传 MinIO ETag 作为 MD5。单块 PutObject
// 下 ETag = MD5(hex)；multipart 上传时 ETag 带 `-N` 后缀，订阅者据此过滤。
func (h *Handler) publishBackupFileReceivedEvent(
	ctx context.Context, bucket, objectPath, filename string, fileSize int64, etag string,
	queryDeviceSN, queryTaskID string,
) {
	taskIDPrefix, deviceSN := parseBackupFilename(filename)
	// Fallback 1：filename 不符合 backup-{taskId8}-{sn}.{ext} 模板时（如设备用了
	// 自己的 NV 文件名 "mib-home-fap.nv"），从 URL query 里兜底拿真实 sn 和
	// taskId 前缀——这才是 backup_restore_file metadata upsert 的唯一可靠源。
	if deviceSN == "" && queryDeviceSN != "" {
		deviceSN = queryDeviceSN
	}
	if taskIDPrefix == "" && queryTaskID != "" {
		hex := strings.ReplaceAll(queryTaskID, "-", "")
		if len(hex) >= 8 {
			taskIDPrefix = hex[:8]
		}
	}
	// Payload 还透传完整 task_id（UUID 字符串）——下游 FilePathRecorder 用它
	// 写 backup_restore_file.task_id（精确隔离不同任务的同名文件），prefix
	// 只够给历史 backup_tasks 表前缀匹配兼容用。

	// 仅当 ETag 形如 32-hex 字符串时视为可信 MD5；multipart ETag 形如
	// "xxxxxxxxx-N" — 后缀带块数，与 MD5 不符。
	md5 := ""
	if isHexMD5(etag) {
		md5 = etag
	}

	payload := map[string]interface{}{
		"bucket":                bucket,
		"object_path":           objectPath,
		"filename":              filename,
		"backup_task_id_prefix": taskIDPrefix,
		"task_id":               queryTaskID, // 完整 UUID，由 FilePathRecorder 写入 backup_restore_file.task_id
		"device_sn":             deviceSN,
		"file_size":             fileSize,
		"md5":                   md5,
	}

	evt, err := event.NewEvent(event.SubjectBackupFileReceived, payload)
	if err != nil {
		h.logger.Error("create backup.file.received event", zap.Error(err))
		return
	}
	if err := h.eventBus.Publish(ctx, event.SubjectBackupFileReceived, evt); err != nil {
		h.logger.Error("publish backup.file.received", zap.Error(err))
		return
	}
	h.logger.Info("published backup.file.received",
		zap.String("path", objectPath),
		zap.String("backup_task_id_prefix", taskIDPrefix),
		zap.String("device_sn", deviceSN))
}

// isHexMD5 reports whether s 由 32 位十六进制字符组成 (大小写均可)。
func isHexMD5(s string) bool {
	if len(s) != 32 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

// backupFilenameRe matches the executor's `backup-{taskID8}-{deviceSN}.xml`
// pattern with optional T-0074/T-0077 compression extension. Capture groups:
//
//	1: taskID8 (8 hex chars)
//	2: deviceSN (any chars up to .xml)
//	3: optional compression extension (.gz/.zst/.lz4/.bz2) — discarded
// backupFilenameRe 匹配两种备份扩展名：
//   .xml — 标准平台（BLQ/QLS）的 CONFIG_BACKUP_XML 走 FileType=10 {OUI} Configuration File
//   .nv  — NV 平台（MLQ/MLN_SC）的 CONFIG_BACKUP_NV 走 FileType=12 {OUI} Configuration File
// 可选 .gz/.zst/.bz2/.lz4 等压缩后缀（T-0074）。
var backupFilenameRe = regexp.MustCompile(`^backup-([0-9a-f]{8})-(.+?)\.(xml|nv)(\.[a-z0-9]+)?$`)

// parseBackupFilename returns (taskIDPrefix, deviceSN) extracted from a
// backup filename. Returns ("", "") when the filename does not match the
// executor-generated pattern (e.g. operator-uploaded ad-hoc config) — the
// subscriber will treat the empty prefix as "no-match skip" without erroring.
func parseBackupFilename(filename string) (taskIDPrefix, deviceSN string) {
	m := backupFilenameRe.FindStringSubmatch(filename)
	if len(m) >= 3 {
		return m[1], m[2]
	}
	return "", ""
}

// publishPMFileReceivedEvent emits SubjectPMFileReceived after a FileType=PM
// upload lands in MinIO. The PM collector (in the worker process) subscribes
// to this and parses the XML / writes pm_metrics / triggers KPI rollups.
//
// Payload is the "thin" variant: device_id / device_oui / carrier / technology
// are intentionally omitted — the collector resolves them from device_sn via
// its DeviceLookup fallback (see internal/pm/collector/collector.go). The
// transfer-bridge path publishes the "fat" variant with all fields pre-filled
// because it already has a DeviceRepository on hand; ACS does not, and we
// don't want to add a synchronous DB lookup on the upload hot path.
//
// If device_sn is empty the publish is skipped with a WARN — the collector
// can't resolve the device without it, and an event with no SN would fail
// downstream anyway.
func (h *Handler) publishPMFileReceivedEvent(
	ctx context.Context, bucket, objectPath, filename string, fileSize int64, deviceSN string,
) {
	if deviceSN == "" {
		h.logger.Warn("PM upload missing device_sn query param; skipping pm.file.received publish",
			zap.String("path", objectPath),
			zap.String("filename", filename))
		return
	}

	payload := map[string]interface{}{
		"minio_path": objectPath,
		"bucket":     bucket,
		"device_sn":  deviceSN,
		"file_size":  fileSize,
		"file_name":  filename,
	}

	evt, err := event.NewEvent(event.SubjectPMFileReceived, payload)
	if err != nil {
		h.logger.Error("create pm.file.received event", zap.Error(err))
		return
	}
	if err := h.eventBus.Publish(ctx, event.SubjectPMFileReceived, evt); err != nil {
		h.logger.Error("publish pm.file.received", zap.Error(err))
		return
	}
	h.logger.Info("published pm.file.received",
		zap.String("device_sn", deviceSN),
		zap.String("path", objectPath),
		zap.Int64("size", fileSize))
}

// pmFilenameSNRe matches the two PM filename layouts we've seen in the wild:
//
//	A{date}.{startTime}-{endTime}_{OUI}.{SN}.xml(.gz)?   — Baicells real CPE
//	pm-{SN}.xml(.gz)?                                    — cpe_simulator.py
//
// Capture group 1 is the deviceSN. Anchored to the end (after stripping the
// optional .gz) so it can't confuse intermediate dot-segments with the SN.
//
// 3GPP 32.435 names PM files this way (`A{date}.{period}_{vendorTag}.{neId}`
// or similar); the regex below captures the Baicells dialect that uses
// `{OUI}.{SN}` as the vendor/NE tag. New vendor dialects should add an
// alternative branch here rather than scattering parsing logic at the call site.
var pmFilenameSNRe = regexp.MustCompile(`(?:^A.+?_[0-9a-fA-F]{6,}\.|^pm-)([^.]+)\.xml(\.gz)?$`)

// extractDeviceSNFromPMFilename returns the device SN parsed from a PM upload
// filename, or "" when none of the recognised vendor patterns match. The
// caller already strips the directory portion, so `filename` is the basename.
func extractDeviceSNFromPMFilename(filename string) string {
	m := pmFilenameSNRe.FindStringSubmatch(filename)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

// publishMRFileReceivedEvent emits SubjectMRFileReceived for fileType=MR direct
// uploads (CPE → OMC HTTP POST). With this, mr.Collector treats direct-upload
// MR files identically to the transfer/bridge AutonomousTransferComplete path:
// download from MinIO, detect MRO/MRS/MRE, parse XML, batch-insert mr_records.
//
// device_id is left empty because the upload handler doesn't have device repo
// injection; mr.Collector's DeviceLookup (wired in cmd/worker) resolves it
// from device_sn → device_id / carrier at consumption time.
func (h *Handler) publishMRFileReceivedEvent(
	ctx context.Context, bucket, objectPath, filename, deviceSN string, fileSize int64,
) {
	payload := map[string]interface{}{
		"minio_path": objectPath,
		"bucket":     bucket,
		"device_id":  "", // 留空，让 collector 按 device_sn 反查
		"device_sn":  deviceSN,
		"carrier":    "", // 同上，由 collector 从 device 实体回填
		"file_name":  filename,
		"file_size":  fileSize,
	}
	evt, err := event.NewEvent(event.SubjectMRFileReceived, payload)
	if err != nil {
		h.logger.Error("create mr.file.received event", zap.Error(err))
		return
	}
	if err := h.eventBus.Publish(ctx, event.SubjectMRFileReceived, evt); err != nil {
		h.logger.Error("publish mr.file.received", zap.Error(err))
		return
	}
	h.logger.Info("published mr.file.received",
		zap.String("device_sn", deviceSN),
		zap.String("path", objectPath))
}

// publishMRFileUploadedEvent emits SubjectMRFileUploaded after a fileType=MR
// upload lands in MinIO. The mr/task HeartbeatSubscriber consumes it to refresh
// the Redis MRFileReport_{cellCode} TTL and call repo.TouchHeartbeat.
//
// cellCode is sourced from the URL query (OMC set it in MrUrl when opening the
// MR task via SPV). Empty cellCode → skip publish (defensive — the device sent
// a non-task-driven MR file; no progress row to update).
func (h *Handler) publishMRFileUploadedEvent(
	ctx context.Context, bucket, objectPath, filename, cellCode, deviceSN string, fileSize int64,
) {
	if cellCode == "" {
		h.logger.Debug("skip mr.file.uploaded: empty cellCode",
			zap.String("path", objectPath), zap.String("filename", filename))
		return
	}
	payload := map[string]interface{}{
		"bucket":      bucket,
		"object_path": objectPath,
		"file_name":   filename,
		"cell_code":   cellCode,
		"device_sn":   deviceSN,
		"file_size":   fileSize,
	}
	evt, err := event.NewEvent(event.SubjectMRFileUploaded, payload)
	if err != nil {
		h.logger.Error("create mr.file.uploaded event", zap.Error(err))
		return
	}
	if err := h.eventBus.Publish(ctx, event.SubjectMRFileUploaded, evt); err != nil {
		h.logger.Error("publish mr.file.uploaded", zap.Error(err))
		return
	}
	h.logger.Info("published mr.file.uploaded",
		zap.String("cell_code", cellCode),
		zap.String("path", objectPath))
}

// publishLogFileReceivedEvent emits SubjectLogFileReceived after a
// running-log (FileType "6") or fault-log (FileType "8") upload lands in MinIO.
// The stationlog module subscribes and creates station_log_files records.
//
// Expected filename patterns (generated by software/executor.go):
//
//	running log: runtime-{taskID8}-{deviceSN}.tar.gz
//	fault log:   fault-{taskID8}-{deviceSN}.tar.gz
func (h *Handler) publishLogFileReceivedEvent(
	ctx context.Context, bucket, objectPath, filename, fileType string, fileSize int64,
) {
	taskID8, deviceSN := parseLogFilename(filename)

	payload := map[string]interface{}{
		"bucket":      bucket,
		"object_path": objectPath,
		"file_name":   filename,
		"file_type":   fileType,
		"file_size":   fileSize,
		"task_id8":    taskID8,
		"device_sn":   deviceSN,
	}

	evt, err := event.NewEvent(event.SubjectLogFileReceived, payload)
	if err != nil {
		h.logger.Error("create log.file.received event", zap.Error(err))
		return
	}
	if err := h.eventBus.Publish(ctx, event.SubjectLogFileReceived, evt); err != nil {
		h.logger.Error("publish log.file.received", zap.Error(err))
		return
	}
	h.logger.Info("published log.file.received",
		zap.String("path", objectPath),
		zap.String("file_type", fileType),
		zap.String("device_sn", deviceSN))
}

// logFilenameRe matches executor-generated log filenames:
//
//	runtime-{taskID8}-{deviceSN}.tar.gz
//	fault-{taskID8}-{deviceSN}.tar.gz
var logFilenameRe = regexp.MustCompile(`^(?:runtime|fault)-([0-9a-f]{8})-(.+)\.tar\.gz$`)

// parseLogFilename extracts (taskID8, deviceSN) from a log filename.
// Returns ("", "") when the filename does not match (e.g. ad-hoc uploads).
func parseLogFilename(filename string) (taskID8, deviceSN string) {
	m := logFilenameRe.FindStringSubmatch(filename)
	if len(m) >= 3 {
		return m[1], m[2]
	}
	return "", ""
}

// extractDeviceSNFromFilename extracts device SN from filename pattern.
// Expected format: "datamodel_{deviceSN}_{uuid}.xml" or "{deviceSN}_datamodel.xml"
func extractDeviceSNFromFilename(filename string) string {
	name := strings.TrimSuffix(filename, ".xml")
	name = strings.TrimSuffix(name, ".gz")

	// Try pattern: datamodel_{SN}_{suffix}
	if strings.HasPrefix(name, "datamodel_") {
		rest := strings.TrimPrefix(name, "datamodel_")
		// Find the last underscore (UUID separator)
		if idx := strings.LastIndex(rest, "_"); idx > 0 {
			return rest[:idx]
		}
		return rest
	}

	// Try pattern: {SN}_datamodel
	if idx := strings.Index(name, "_datamodel"); idx > 0 {
		return name[:idx]
	}

	// Fallback: return the full name without extension
	return name
}

// deriveUploadFilename 在设备 URL `filename=` 留空时，按业务规则生成与 UFTE 端
// target_file_name_template 一致的文件名。与 software/executor.taskIDPrefix +
// ufte_task_types.target_file_name_template 的渲染规则保持对齐：
//   - 备份配置（FileType 3 / CONFIGBACKUP_*）→ backup-{taskId8}-{sn}.<nv|xml>
//   - 日志采集（FileType 6 运行日志 / 8 故障日志）→ log-{taskId8}-{sn}.tar.gz
//   - 其它/兜底 → upload-{taskId8}-{sn}-{unix}
// 一致性保证：ACS 落地的 filename 与 UFTE DeviceItem.TargetFile 渲染结果同名，
// 后续 fileLandedLookup / downloadURLLookup 用 (sn, target_file) 反查能命中。
func deriveUploadFilename(fileType, taskID, sn string) string {
	taskID8 := taskID
	if hex := strings.ReplaceAll(taskID, "-", ""); len(hex) >= 8 {
		taskID8 = hex[:8]
	}
	ft := strings.ToUpper(strings.TrimSpace(fileType))
	switch ft {
	case "CONFIGBACKUP_NV":
		return fmt.Sprintf("backup-%s-%s.nv", taskID8, sn)
	case "CONFIGBACKUP_XML", "3":
		return fmt.Sprintf("backup-%s-%s.xml", taskID8, sn)
	case "6", "LOG":
		return fmt.Sprintf("runtime-%s-%s.tar.gz", taskID8, sn)
	case "8", "RL":
		return fmt.Sprintf("fault-%s-%s.tar.gz", taskID8, sn)
	default:
		return fmt.Sprintf("upload-%s-%s-%d", taskID8, sn, time.Now().Unix())
	}
}
