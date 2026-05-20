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

	// 3. Extract fileType and filename from query params
	fileType := r.URL.Query().Get("fileType")
	filename := r.URL.Query().Get("filename")

	if fileType == "" {
		http.Error(w, "missing fileType parameter", http.StatusBadRequest)
		return
	}
	if filename == "" {
		http.Error(w, "missing filename parameter", http.StatusBadRequest)
		return
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
	var objectPath string
	if category != "" {
		objectPath = fmt.Sprintf("%s/%s/%s", category, now.Format("2006/01/02"), filename)
	} else {
		objectPath = fmt.Sprintf("%s/%s", now.Format("2006/01/02"), filename)
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
		h.publishBackupFileReceivedEvent(ctx, bucket, objectPath, filename, info.Size, info.ETag)
	}

	// 6.3. For station log uploads (FileType "6" running log, "8" fault log),
	// publish log.file.received so the stationlog module can record the file
	// and enforce quotas.
	if (ft == tr069.FileTypeRunningLog || ft == tr069.FileTypeFaultLog) && h.eventBus != nil {
		h.publishLogFileReceivedEvent(ctx, bucket, objectPath, filename, string(ft), info.Size)
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
	case "8":
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
) {
	taskIDPrefix, deviceSN := parseBackupFilename(filename)

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
var backupFilenameRe = regexp.MustCompile(`^backup-([0-9a-f]{8})-(.+)\.xml(\.[a-z0-9]+)?$`)

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
