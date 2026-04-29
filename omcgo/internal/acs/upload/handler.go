package upload

import (
	"context"
	"crypto/subtle"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/minio/minio-go/v7"
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
	username string
	password string
	// T-0074: optional backup compression hooks. When both fields are set and
	// the inbound file_type is FileTypeConfig with policy.EnableCompression=true,
	// the body stream is wrapped with the configured compressor before MinIO
	// PutObject. Both fields are nil-safe — a nil getter disables compression.
	policyGetter       backup.PolicyGetter
	compressionMetrics *backup.PolicyMetrics
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
	if h.username != "" {
		username, password, ok := r.BasicAuth()
		if !ok {
			h.logger.Warn("missing basic auth credentials")
			w.Header().Set("WWW-Authenticate", `Basic realm="FileUpload"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if subtle.ConstantTimeCompare([]byte(username), []byte(h.username)) != 1 ||
			subtle.ConstantTimeCompare([]byte(password), []byte(h.password)) != 1 {
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
	if h.maxFileSize > 0 && r.ContentLength > h.maxFileSize {
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

	// 7. Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","path":"%s","size":%d}`, objectPath, info.Size)
}

// normalizeFileType converts the fileType query parameter to a tr069.FileType.
// Handles both numeric codes ("4") and text aliases ("PM").
func normalizeFileType(raw string) tr069.FileType {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "1":
		return tr069.FileTypeFirmware
	case "2":
		return tr069.FileTypePatch
	case "3":
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
//   - format ∈ {lz4, bzip2} (not yet implemented; service layer should have
//     rejected this combination on PUT, but defend in depth), OR
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
	if pol.CompressionFormat == "lz4" || pol.CompressionFormat == "bzip2" {
		h.logger.Warn("compression format not implemented; passing through",
			zap.String("format", pol.CompressionFormat))
		return compressionWrap{}
	}
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
