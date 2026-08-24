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
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/backup"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/imsparam"
	"github.com/omcgo/omcgo/internal/storageprotection"
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.uber.org/zap"
)

// Handler handles HTTP file upload requests from CPE devices.
// Endpoint: POST /smallcell/FileUploadService?fileType=PM&filename=xxx
// Authentication: HTTP Basic Auth with global credentials from config
type Handler struct {
	tokenManager *TokenManager
	sessionStore *SessionStore
	minioClient  objectStore
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
	// #318: optional PM upload backpressure gate. nil-safe — when wired and the
	// inbound fileType is PM, ServeHTTP rejects with 503 while the watchdog
	// reports backpressure (disk/CPU over high watermark). Devices retry per
	// TR-069 so no data is lost.
	backpressure BackpressureGate
	// PM 上传去重（压测观测到 omc_pm_files_processed_total{status=duplicate} 占比
	// 高达约78%）：CPE 网络抖动会在短时间内对同一份文件重复发起 HTTP 上传，worker
	// 侧 IsFileParsed 短路虽然接近零成本，但重复的 MinIO 落盘 + NATS 事件发布仍然
	// 浪费 IO、占用 pm-workers 并发槽位、放大队列积压观测值。nil-safe：未注入时
	// 不做去重（等价于原有行为）。复用 internal/core/event.Deduper（Redis SETNX +
	// TTL，fail-open），key 用 (device_sn, filename) 而不是 event ID——要拦的是
	// "同一份文件被多次上传"，此时还没有 event，天然不能用 event ID 去重。
	pmDedup          *event.Deduper
	storageAdmission storageprotection.WriteAdmission
}

type objectStore interface {
	PutObject(context.Context, string, string, io.Reader, int64, minio.PutObjectOptions) (minio.UploadInfo, error)
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

// SetBackpressureGate 注入 PM 上传背压门闸（#318）。nil-safe：未注入时不做背压。
func (h *Handler) SetBackpressureGate(gate BackpressureGate) {
	h.backpressure = gate
}

func (h *Handler) SetStorageAdmission(admission storageprotection.WriteAdmission) {
	h.storageAdmission = admission
}

// SetPMUploadDedup 注入 PM 上传去重器。nil-safe：未注入时不做去重（原有行为）。
func (h *Handler) SetPMUploadDedup(d *event.Deduper) {
	h.pmDedup = d
}

// checkPMUploadDuplicate 判断 (deviceSN, filename) 这个 PM 上传在去重 TTL 窗口内
// 是否已经见过。返回 true 表示应该跳过本次上传（重复），调用方应直接答复设备成功，
// 不再落 MinIO、不再发布事件。
//
// nil-safe / fail-open：h.pmDedup 未注入、deviceSN 为空、或 Redis 出错时都返回
// false（不跳过，走原有正常上传流程），避免因为去重能力缺失或故障影响主链路。
func (h *Handler) checkPMUploadDuplicate(ctx context.Context, deviceSN, filename string) bool {
	if h.pmDedup == nil || deviceSN == "" {
		return false
	}
	first, err := h.pmDedup.FirstTime(ctx, "acs-pm-upload", deviceSN+":"+filename)
	if err != nil {
		h.logger.Warn("PM upload dedup check failed, failing open",
			zap.Error(err), zap.String("device_sn", deviceSN), zap.String("filename", filename))
		return false
	}
	if !first {
		h.logger.Info("PM upload deduplicated: same device_sn+filename seen recently, skipping store+publish",
			zap.String("device_sn", deviceSN), zap.String("filename", filename))
		return true
	}
	return false
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
	h.clearTransferDeadlines(w)

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
	// 服务端按业务规则生成可追踪且尽量可读的落地名：
	//   - 配置备份（FileType "3" / CONFIGBACKUP_*）：{sn}_CFG.{xml|nv}（issue #585）
	//   - 运行日志（FileType "6"/LOG）：runtime-{taskId8}-{sn}.tar.gz
	//   - 异常日志（FileType "8"/RL）：fault-{sn}-{yyyyMMddHHmmssSSS}.tar.gz
	//   - 其它（默认兜底）：upload-{taskId8}-{sn}-{ts}
	// UFTE 完成后通过 backup_restore_file 按 (sn, task_id) 反查真实落地文件名，
	// 再用该文件名换下载 URL，因此异常日志不再需要把 taskId8 暴露在路径或文件名里。
	// FAULT_LOG_COLLECT 用 ?id=<task_uuid>，其他链路用 ?taskId=<task_uuid>；取 id 兜底 taskId。
	queryTaskID := r.URL.Query().Get("taskId")
	if queryTaskID == "" {
		queryTaskID = r.URL.Query().Get("id")
	}
	querySN := r.URL.Query().Get("sn")
	ft := normalizeFileType(fileType)
	now := time.Now()

	// issue #585：配置备份强制规范命名 {sn}_CFG.{xml|nv}。
	// 不论设备给的 filename 是什么（baicells 用 "provisioning.xml" / "mib-home-fap.nv"，
	// 其它厂商各异），ACS 在落 MinIO 之前一律覆盖为规范名。
	// 收益：
	//   - 任务管理 presigned URL 末段直接是 {sn}_CFG.{ext}，不再需要 response-content-disposition 黑魔法
	//   - 文件管理 snapshot 桶 promote 可退化为 server-side CopyObject（命名口径一致）
	//   - MinIO 直查 / 运维定位 / 排错都用规范名
	// 仅在 sn 可用时强制；sn 缺失则保留原 filename / derive 兜底（不应发生，OMC 派发的 URL 一定带 sn）。
	if canonical, ok := canonicalConfigBackupFilename(fileType, querySN); ok {
		if filename != canonical {
			h.logger.Info("config backup filename rewritten to canonical form",
				zap.String("file_type", fileType),
				zap.String("sn", querySN),
				zap.String("device_supplied", filename),
				zap.String("canonical", canonical),
			)
		}
		filename = canonical
	} else if ft == tr069.FileTypeFaultLog && querySN != "" && queryTaskID != "" {
		// 异常日志去掉 taskId8 目录后，不能再信任设备自带文件名的唯一性。
		// OMC 统一二次命名为 fault-{sn}-{yyyyMMddHHmmssSSS}.tar.gz，
		// 既让 MinIO 目录可读，又用毫秒级时间避免同设备多次采集互相覆盖。
		derived := deriveUploadFilenameAt(fileType, queryTaskID, querySN, now)
		if filename != derived {
			h.logger.Info("fault log filename rewritten to timestamped canonical form",
				zap.String("file_type", fileType),
				zap.String("sn", querySN),
				zap.String("task_id", queryTaskID),
				zap.String("device_supplied", filename),
				zap.String("canonical", derived),
			)
		}
		filename = derived
	} else if isImsCoreUpload(ft) && filename == "" {
		// 核心网采集类（参数/日志/License/恢复）：URL filename= 留空（设备自决文件名）。
		// 设备也没给 filename 时按类型 + sn 派生兜底名。
		if queryTaskID == "" || querySN == "" {
			http.Error(w, "missing filename, and cannot derive: taskId/sn query params also empty", http.StatusBadRequest)
			return
		}
		subType := r.URL.Query().Get("paramType")
		filename = deriveImsFilename(subType, querySN, now)
		h.logger.Info("derived ims filename from sn+taskId (URL filename was empty)",
			zap.String("file_type", fileType),
			zap.String("sn", querySN),
			zap.String("task_id", queryTaskID),
			zap.String("derived_filename", filename),
		)
	} else if filename == "" {
		if queryTaskID == "" || querySN == "" {
			http.Error(w, "missing filename, and cannot derive: taskId/sn query params also empty", http.StatusBadRequest)
			return
		}
		filename = deriveUploadFilenameAt(fileType, queryTaskID, querySN, now)
		h.logger.Info("derived filename from sn+taskId (URL filename was empty)",
			zap.String("file_type", fileType),
			zap.String("sn", querySN),
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
	// 4.1 纵深防御：ContentLength 可被伪造或缺失（chunked 传输为 -1），仅靠上面的
	// 头部检查不足。用 MaxBytesReader 在流式读取层强制同一上限，超限时读取返回
	// 错误并中止上传，防止绕过 ContentLength 的超大上传耗尽内存。
	if runtimeCfg.MaxFileSize > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, runtimeCfg.MaxFileSize)
	}

	// 5. Determine bucket and object path

	// #318：PM 上传背压门闸。磁盘/CPU 超高水位时（watchdog 后台维护态，热路径仅读原子标志）
	// 对 PM 文件早返回 503——在落 MinIO 前拒收，TR-069 设备会重传，不丢数据；回落自动恢复。
	//
	// 必须在去重检查之前做（2026-07-20 修复 #833 复测暴露的严重 bug）：去重用的是
	// Redis SETNX + 24h TTL，一旦某次请求先被去重标记为"已见过"，之后 24 小时内同一
	// (device_sn, filename) 的所有请求都会被去重短路直接回 200 OK，不再真正落
	// MinIO/发事件。如果去重检查在背压之前，背压生效期间收到的第一次请求会：先被
	// 去重标记为"已见过" → 再被背压拒收（503）。设备按 TR-069 语义重传时，重传请求
	// 却会被去重当成"已处理过的重复"直接吃掉、回 200 OK——设备以为上传成功，但这份
	// PM 文件从未真正落盘/入库，且 24h 内都无法再重传成功，等价于背压窗口内的 PM
	// 文件被静默永久丢弃，与背压设计初衷"设备重传、不丢数据"直接矛盾（omc78 压测环境
	// 实测复现：背压持续 2.5 小时期间该设备的 PM 文件在 pm_files 表里一条都没有，
	// 背压解除后也没有补上）。把背压检查挪到去重之前即可修复：背压拒收的请求根本
	// 不会走到去重这一步，去重标记只会在请求真正被接纳、准备落盘时才打上。
	if ft == tr069.FileTypePM && h.backpressure != nil {
		if allowed, reason := h.backpressure.Acquire(); !allowed {
			h.backpressure.RecordRejected(reason)
			h.logger.Warn("PM upload rejected: resource backpressure or inflight limit",
				zap.String("filename", filename), zap.String("remote_addr", r.RemoteAddr),
				zap.String("reason", reason))
			w.Header().Set("Retry-After", "60")
			http.Error(w, "PM upload temporarily paused due to resource backpressure", http.StatusServiceUnavailable)
			return
		}
		defer h.backpressure.Release()
	}

	// PM 上传去重短路：SN 提取优先级与 6.4 节发布事件时一致（URL query `sn=` 优先，
	// 回退文件名解析），保证同一次真实上传（含 CPE 重传）在这里算出的 key 稳定一致。
	if ft == tr069.FileTypePM && h.pmDedup != nil {
		pmSN := r.URL.Query().Get("sn")
		if pmSN == "" {
			pmSN = extractDeviceSNFromPMFilename(filename)
		}
		if h.checkPMUploadDuplicate(r.Context(), pmSN, filename) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status":"ok","dedup":true,"filename":"%s"}`, filename)
			return
		}
	}

	bucket, category := storage.BucketAndCategory(ft, h.buckets)
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
	} else {
		objectPath = buildUploadObjectPath(ft, category, now, queryTaskID, filename)
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

	cmp := h.maybeWrapForCompression(ctx, ft, filename, r.Body)
	if cmp.applied {
		objectPath += cmp.ext
		// ContentEncoding documents the on-disk compression so the restore
		// side (T-0072) can read it from object metadata as a backup signal
		// to the .gz/.zst filename suffix.
		uploadOpts.ContentEncoding = cmp.format
		if ft == tr069.FileTypePM || ft == tr069.FileTypeMR {
			// PM/MR 压缩体积通常不大（几十KB~几MB），提前读完拿到确切压缩后
			// 大小，走已知 size 的单次 PutObject。若像下面 else 分支那样
			// 传 contentLength=-1（未知大小），minio-go 对未知大小走
			// putObjectMultipartStreamParallel，会为每次调用分配
			// opts.NumThreads*PartSize（可达数百 MB）的内存缓冲区——高
			// 并发下（PM 上传背压 max_inflight 调大后）迅速把内存打爆，
			// 触发 cgroup OOM killer 循环杀死 ACS 进程（2026-07-21 omc78
			// 压测实测：4G 内存限额下仍持续 OOMKilled，根因在此，不是文件
			// 本身大，是 SDK 对"未知大小"的缓冲策略）。MR 与 PM 同
			// 构（高频、结构化、重复性强的性能/测量数据），同样会在
			// max_inflight 调大后遇到同样的高并发惊群，因此与 PM 统一
			// 处理。
			compressed, readErr := io.ReadAll(cmp.body)
			cmp.body.Close()
			if readErr != nil {
				h.logger.Error("read compressed PM/MR upload body failed",
					zap.Error(readErr), zap.String("path", objectPath))
				http.Error(w, "read upload body failed", http.StatusBadRequest)
				return
			}
			body = bytes.NewReader(compressed)
			contentLength = int64(len(compressed))
		} else {
			defer cmp.body.Close()
			body = cmp.body
			contentLength = -1 // streaming, compressed size unknown（备份大文件场景仍走流式，保留原行为）
		}
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

	// 直传路径（未压缩、未加密——PM/MR 等绝大多数上传都走这条路径）此时 body 仍是
	// r.Body 本身：一个不支持 Seek 的 io.ReadCloser。minio-go 在遇到瞬时网络错误
	// （连接被 MinIO 端复用/重置）时会内部重试整个 PUT；重试要求 body 能从头重读，
	// 而 r.Body 已经在第一次尝试中被读到 EOF，重试读到 0 字节，minio-go 报
	// "http: ContentLength=N with Body length 0"（生产环境批量 PM 上传失败的根因）。
	// 这里把 body 缓冲进内存再包成 bytes.Reader（实现 io.ReadSeeker），让重试可以
	// Seek 回起点安全重放。体积已被上面的 MaxBytesReader 卡住上限，缓冲内存可控。
	if !cmp.applied && !encApplied {
		buffered, readErr := io.ReadAll(body)
		if readErr != nil {
			h.logger.Error("read upload body failed",
				zap.Error(readErr),
				zap.String("file_type", fileType),
				zap.String("path", objectPath),
			)
			http.Error(w, "read upload body failed", http.StatusBadRequest)
			return
		}
		body = bytes.NewReader(buffered)
		contentLength = int64(len(buffered))
	}

	if h.storageAdmission != nil {
		scope := storageprotection.WriteScopeUpload
		switch ft {
		case tr069.FileTypePM:
			scope = storageprotection.WriteScopePM
		case tr069.FileTypeMR:
			scope = storageprotection.WriteScopeMR
		case tr069.FileTypeConfig:
			scope = storageprotection.WriteScopeBackup
		}
		decision, admissionErr := h.storageAdmission.CheckPath(ctx, storageprotection.ProtectedPathIDMinIO, scope)
		if admissionErr != nil {
			h.logger.Error("storage write admission check failed", zap.Error(admissionErr), zap.String("scope", string(scope)))
			http.Error(w, "storage admission unavailable", http.StatusServiceUnavailable)
			return
		}
		if !decision.Allowed {
			if decision.RetryAfter > 0 {
				w.Header().Set("Retry-After", strconv.Itoa(int(decision.RetryAfter.Seconds())))
			}
			http.Error(w, "storage write protected", http.StatusInsufficientStorage)
			return
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
	// publish log.file.received. The stationlog module persists running logs;
	// fault-log task/file state is maintained by the file-transfer task chain
	// and must not mutate reboot records.
	//
	// 同时发 backup.file.received —— 前端 UFTE 设备列表的「文件名 / 下载」UI 已经
	// 走通了基于 backup_restore_file 表的反查链路。让 LOG 上传也写一行通用元数据
	// 到 backup_restore_file，前端复用现有反查 + presigned URL 下载逻辑。
	if (ft == tr069.FileTypeRunningLog || ft == tr069.FileTypeFaultLog) && h.eventBus != nil {
		h.publishLogFileReceivedEvent(ctx, bucket, objectPath, filename, string(ft), info.Size, querySN, queryTaskID)
		h.publishBackupFileReceivedEvent(ctx, bucket, objectPath, filename, info.Size, info.ETag,
			r.URL.Query().Get("sn"), queryTaskID)
	}

	// 6.3.1 核心网参数文件（IMS_PARAM）：同 LOG 链路发 backup.file.received ——
	// FilePathRecorder 写 backup_restore_file（UFTE 设备列表 presigned 下载反查）+
	// 通知 software.HandleFileLandedForCollect 把采集子任务推 Completed。
	// TC 随后到达时走 handleTCBody 幂等收口。
	// 参数（FT1~12）/ 日志（FT13/18/19）/ License（FT15）/ 恢复（FT17）四类同链路。
	if isImsCoreUpload(ft) && h.eventBus != nil {
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
	if ft == tr069.FileTypePM && h.backpressure != nil {
		h.backpressure.RecordAccepted(time.Now())
	}
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

func (h *Handler) clearTransferDeadlines(w http.ResponseWriter) {
	controller := http.NewResponseController(w)
	if err := controller.SetReadDeadline(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
		h.logger.Warn("clear file upload read deadline", zap.Error(err))
	}
	if err := controller.SetWriteDeadline(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
		h.logger.Warn("clear file upload write deadline", zap.Error(err))
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
	case "IMS_FILE", "IMS FILE", "IMS_PARAM", "IMSCORE_PARAMETERS_TYPE", "IMSCORE PARAMETERS FILE",
		"IMS_LOG", "IMS LOG FILE", "IMS_LICENSE", "IMS LICENSE FILE", "IMS_RECOVERY", "IMS RECOVERY FILE":
		// 核心网文件传输（CWMP FileType 统一 "Ims File"；历史别名保留兼容在途 URL）。
		// 具体文件类型由 URL query paramType=FT_ImsCore_* 携带
		// （见 internal/imsparam / docs/design/imscore-file-transfer.md）。
		return tr069.FileTypeImsCoreParam
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
//
// Deprecated (issue #585): production ACS wiring no longer calls SetCompression.
// Config backup uploads must remain plaintext so the Task Management presigned
// URL download is consistent across browsers/CLI tools. The method + the
// downstream maybeWrapForCompression branch are kept (and unit-tested) so the
// implementation remains audit-able, but any new caller must justify why the
// objects on storage should diverge from the device-provided raw bytes.
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
//   - file type is PM or MR: delegates to compressPMUpload (unconditional
//     gzip, no policy gate — see compressPMUpload doc), OR
//   - file type is not FileTypeConfig (only backup files compress today), OR
//   - no policyGetter wired, OR
//   - policy lookup failed, OR
//   - policy.EnableCompression=false, OR
//   - NewCompressor / Wrap failed (recorded as metric, fall back to plaintext).
//
// The fall-back-on-failure choice is deliberate: backup is a high-availability
// feature; we prefer storing larger uncompressed bytes over failing the upload.
func (h *Handler) maybeWrapForCompression(ctx context.Context, ft tr069.FileType, filename string, src io.Reader) compressionWrap {
	if ft == tr069.FileTypePM || ft == tr069.FileTypeMR {
		return h.compressPMUpload(ctx, filename, src)
	}
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

// pmUploadGzipLevel is the fixed gzip level used for compressPMUpload —
// matches the level-6 default used elsewhere in this codebase (backup.
// DefaultPolicy), a balanced choice given ACS handles many concurrent PM
// uploads and shouldn't burn excessive CPU per file on max compression.
const pmUploadGzipLevel = 6

// compressPMUpload gzip-wraps a PM (FileType=4) or MR (FileType=5) upload
// body before it lands in MinIO.
//
// Unlike maybeWrapForCompression's backup-config branch (policy-gated per
// sys_configs, currently dormant per issue #585), PM/MR compression here is
// unconditional: PM/KPI and MR XML are highly repetitive (many similar
// counter/tag names) and gzip well, directly cutting MinIO write IO on the
// ACS hot path and MinIO read IO when the worker downloads it to parse — a
// stress-test diagnosis found tsdb/host IO pressure (PSI) as the dominant
// bottleneck behind PM ingestion lag, not worker CPU/concurrency. MR shares
// the same shape (high-frequency, structured, repetitive measurement data),
// so it gets the same treatment (docs/project/pm-mr-gzip-rekey-plan-20260616.md
// §2 已锁定「范围：PM + MR 一起改（同构链路）」).
//
// No changes are required downstream:
//   - internal/pm/collector.go and internal/mr/collector/collector.go already
//     transparently gunzip on download via core/compress.MaybeGunzip (issue
//     #321 — real CPEs already upload .xml.gz in the wild, so this path was
//     already exercised for PM; MR collector shares the same helper).
//   - internal/core/rawarchive.Archiver's async re-compression pass already
//     probes the gzip magic number and short-circuits to outcomeSkippedGz
//     (a single 2-byte ReadHead, no re-read/re-write) for objects that are
//     already gzip, so it won't double-compress.
//   - internal/pm/handler.go's DownloadPMFile and internal/mr/handler.go's
//     DownloadFile already sniff the gzip magic number at download time and
//     adjust the served filename/Content-Type accordingly, so the existing
//     download UI keeps working unchanged for both PM and MR.
//
// Guards against double-compression: if filename already ends in ".gz" the
// CPE (or simulator) is already sending gzip bytes (issue #321) — this
// returns compressionWrap{} (not applied) so we never wrap gzip in gzip.
func (h *Handler) compressPMUpload(ctx context.Context, filename string, src io.Reader) compressionWrap {
	if strings.HasSuffix(strings.ToLower(filename), ".gz") {
		return compressionWrap{}
	}
	c, err := backup.NewCompressor("gzip", pmUploadGzipLevel)
	if err != nil {
		h.logger.Warn("PM upload gzip compressor construction failed; uploading plaintext", zap.Error(err))
		return compressionWrap{}
	}
	counter := &countingReader{r: src}
	wrapped, err := c.Wrap(ctx, counter)
	if err != nil {
		h.logger.Warn("PM upload gzip wrap failed; uploading plaintext", zap.Error(err))
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
//
// backupFilenameRe 匹配两种备份扩展名：
//
//	.xml — 标准平台（BLQ/QLS）的 CONFIG_BACKUP_XML 走 FileType=10 {OUI} Configuration File
//	.nv  — NV 平台（MLQ/MLN_SC）的 CONFIG_BACKUP_NV 走 FileType=12 {OUI} Configuration File
//
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

// pmFilenameSNRe matches the PM filename layouts we've seen in the wild. The
// deviceSN is always the final dot-segment before `.xml`; the leading branches
// just gate which prefixes we trust as PM files (anchored so an arbitrary
// `{anything}.xml` won't be mistaken for a PM upload):
//
//	A{date}.{period}_{OUI}.{SN}.xml(.gz)?   — Baicells real CPE (4G/LTE)
//	A{date}.{period}.{SN}.xml(.gz)?         — 3GPP 32.435 `A`-form without an
//	                                          {OUI} vendor tag (#364: BSC/2G GSM
//	                                          real-CPE dialect, no underscore-OUI)
//	pm-{SN}.xml(.gz)? / pm_{SN}.xml(.gz)?   — cpe_simulator.py + simulator dialect
//	PM-{SN}.xml(.gz)? / PM_{SN}.xml(.gz)?   — vendor PM-prefix dialect (#364)
//
// Capture group 1 is the deviceSN. Anchored to the end (after stripping the
// optional .gz) so it can't confuse intermediate dot-segments with the SN.
//
// 3GPP 32.435 names PM files `A{date}.{period}_{vendorTag}.{neId}` (or without
// the vendor tag). The `^A` branch is generalised to "starts with A, capture the
// last dot-segment", so both the OUI and no-OUI dialects route through one rule.
//
// New vendor dialects should add an alternative prefix branch in this single
// regex rather than scattering parsing logic at the call site. The safest path
// for any new vendor remains the explicit `?sn=` URL query (see handler PM
// branch), which bypasses filename-dialect guessing entirely.
var pmFilenameSNRe = regexp.MustCompile(`(?i:^A.+\.|^pm[-_])([^.]+)\.xml(\.gz)?$`)

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
// The stationlog module subscribes and records running logs only; fault-log
// upload state stays in the file-transfer task chain.
//
// Expected filename patterns (generated by software/executor.go):
//
//	running log: runtime-{taskID8}-{deviceSN}.tar.gz
//	fault log:   fault-{deviceSN}-{yyyyMMddHHmmssSSS}.tar.gz
//
// Some CPEs upload vendor-generated names such as ErrorLog_...dieLog.tar.gz.
// For those, fall back to the URL query values OMC placed in the upload URL.
func (h *Handler) publishLogFileReceivedEvent(
	ctx context.Context, bucket, objectPath, filename, fileType string, fileSize int64, querySN, queryTaskID string,
) {
	var taskID8, deviceSN string
	if normalizeFileType(fileType) == tr069.FileTypeFaultLog {
		// Fault-log filenames no longer include taskID8. Query params are the
		// authoritative identity for OMC-triggered FaultLogURL uploads.
		deviceSN = strings.TrimSpace(querySN)
		taskID8 = taskID8FromQuery(queryTaskID)
		if deviceSN == "" || taskID8 == "" {
			fallbackTaskID8, fallbackSN := parseLogFilename(filename)
			if taskID8 == "" {
				taskID8 = fallbackTaskID8
			}
			if deviceSN == "" {
				deviceSN = fallbackSN
			}
		}
	} else {
		taskID8, deviceSN = parseLogFilename(filename)
		if deviceSN == "" {
			deviceSN = strings.TrimSpace(querySN)
		}
		if taskID8 == "" {
			taskID8 = taskID8FromQuery(queryTaskID)
		}
	}

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

func taskID8FromQuery(taskID string) string {
	compact := strings.ReplaceAll(strings.TrimSpace(taskID), "-", "")
	if len(compact) < 8 {
		return ""
	}
	return compact[:8]
}

// buildUploadObjectPath creates the MinIO object key for non-MR uploads.
// Config backups keep the taskID8 directory because vendors often reuse the
// same source filename. Fault logs are timestamp-renamed before this point, so
// they intentionally stay directly under fault/YYYY/MM/DD/.
func buildUploadObjectPath(ft tr069.FileType, category string, now time.Time, queryTaskID, filename string) string {
	taskSubdir := ""
	if ft == tr069.FileTypeConfig {
		if tid := strings.ReplaceAll(queryTaskID, "-", ""); len(tid) >= 8 {
			taskSubdir = tid[:8] + "/"
		}
	}
	if category != "" {
		return fmt.Sprintf("%s/%s/%s%s", category, now.Format("2006/01/02"), taskSubdir, filename)
	}
	return fmt.Sprintf("%s/%s%s", now.Format("2006/01/02"), taskSubdir, filename)
}

// logFilenameRe matches executor-generated log filenames:
//
//	runtime-{taskID8}-{deviceSN}.tar.gz
//	legacy fault fallback: fault-{taskID8}-{deviceSN}.tar.gz
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

// deriveUploadFilename 在设备 URL `filename=` 留空且文件类型不属于配置备份时，
// 按业务规则生成上传兜底文件名。
// 配置备份类（FileType "3" / CONFIGBACKUP_*）不再走这里——上游已强制规范为
// {sn}_CFG.{xml|nv}（见 canonicalConfigBackupFilename），即使设备给了 filename
// 也会被覆盖。本函数仅服务日志类 / 默认兜底。
func deriveUploadFilename(fileType, taskID, sn string) string {
	return deriveUploadFilenameAt(fileType, taskID, sn, time.Now())
}

func deriveUploadFilenameAt(fileType, taskID, sn string, now time.Time) string {
	taskID8 := taskID
	if hex := strings.ReplaceAll(taskID, "-", ""); len(hex) >= 8 {
		taskID8 = hex[:8]
	}
	ft := strings.ToUpper(strings.TrimSpace(fileType))
	switch ft {
	case "6", "LOG":
		return fmt.Sprintf("runtime-%s-%s.tar.gz", taskID8, sn)
	case "8", "RL":
		return fmt.Sprintf("fault-%s-%s.tar.gz", sn, timestampMillis(now))
	default:
		return fmt.Sprintf("upload-%s-%s-%d", taskID8, sn, now.Unix())
	}
}

func timestampMillis(t time.Time) string {
	return t.Format("20060102150405") + fmt.Sprintf("%03d", t.Nanosecond()/int(time.Millisecond))
}

// isImsCoreUpload 判断是否核心网上传（FileType 别名统一归一到 IMS_PARAM 后，
// 历史上 Log/License/Recovery 枚举不再由 URL 触达，仅保留枚举定义兼容存量对象）。
func isImsCoreUpload(ft tr069.FileType) bool {
	switch ft {
	case tr069.FileTypeImsCoreParam, tr069.FileTypeImsCoreLog,
		tr069.FileTypeImsCoreLicense, tr069.FileTypeImsCoreRecovery:
		return true
	}
	return false
}

// deriveImsFilename 为 URL filename= 留空的核心网上传派生兜底文件名（与
// executor.imsParamUploadFileName 同款格式）：paramType 去 FT_ImsCore_ 前缀 +
// 年月日时分秒；日志类（*_Logs_U）.log 后缀，其余 .dat。paramType 非法时退化
// sn + 毫秒时间戳，保证文件仍可落地可追踪。
func deriveImsFilename(subType, sn string, now time.Time) string {
	code := imsparam.NormalizeParamType(subType)
	if _, ok := imsparam.Lookup(code); ok {
		name := strings.TrimPrefix(code, "FT_ImsCore_")
		ext := ".dat"
		if imsparam.IsLogType(code) {
			ext = ".log"
		}
		return fmt.Sprintf("%s_%s%s", name, now.Format("20060102150405"), ext)
	}
	return fmt.Sprintf("ims-%s-%s.dat", sn, timestampMillis(now))
}

// canonicalConfigBackupFilename 推导配置备份上传的规范文件名 {sn}_CFG.{ext}。
// 返回 ok=false 表示当前 fileType 不是配置备份（或 sn 缺失），调用方按其它分支处理。
//
// fileType 取值来源（OMC 派发 Upload RPC 时由 UFTE TransportPath 模板写死）：
//   - "CONFIGBACKUP_XML" → .xml
//   - "CONFIGBACKUP_NV"  → .nv
//   - "3"               → .xml（数字编号无法区分 XML/NV，默认走 XML——
//     OMC 自家派发都用文本 alias，数字编号仅为协议兼容兜底）
func canonicalConfigBackupFilename(fileType, sn string) (string, bool) {
	sn = strings.TrimSpace(sn)
	if sn == "" {
		return "", false
	}
	switch strings.ToUpper(strings.TrimSpace(fileType)) {
	case "CONFIGBACKUP_NV":
		return fmt.Sprintf("%s_CFG.nv", sn), true
	case "CONFIGBACKUP_XML", "3":
		return fmt.Sprintf("%s_CFG.xml", sn), true
	default:
		return "", false
	}
}
