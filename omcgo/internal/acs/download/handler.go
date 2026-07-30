package download

import (
	"bytes"
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net/http"
	pathpkg "path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/backup"
)

type downloadObjectClient interface {
	GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
}

// Handler handles HTTP file download requests from CPE devices.
// Endpoint: GET /smallcell/FileDownloadService/{bucket}/{objectPath...}
// Authentication: HTTP Basic Auth with global credentials from config.
// CPE accesses this endpoint using the URL and credentials provided in the Download SOAP RPC.
type Handler struct {
	minioClient     downloadObjectClient
	logger          *zap.Logger
	username        string
	password        string
	runtimeProvider transfercfg.Provider
	// T-0072: optional metrics for on-the-fly decompression. nil-safe — Record*
	// methods short-circuit on nil receiver, so wiring is optional.
	metrics *DecompressMetrics
	// T-0075: optional decryptor for objects with `.enc` suffix. nil-safe;
	// when nil, encrypted objects are served as-is (CPE will fail to consume
	// them — visible in download metrics). encMetrics is the shared
	// PolicyMetrics surface where decryption failures count.
	encryptor  backup.Encryptor
	encMetrics *backup.PolicyMetrics
	// T-0089: optional concurrency cap for maybeDecrypt 64MB×N memory
	// amplification. nil = unbounded (T-0075 baseline).
	decryptSem *DecryptSemaphore
}

// NewHandler creates a new download handler.
func NewHandler(minioClient downloadObjectClient, username, password string, logger *zap.Logger) *Handler {
	return &Handler{
		minioClient: minioClient,
		username:    username,
		password:    password,
		logger:      logger,
	}
}

func (h *Handler) SetRuntimeProvider(provider transfercfg.Provider) {
	h.runtimeProvider = provider
}

// SetDecompressMetrics wires Prometheus collectors for on-the-fly decompression.
// Pass nil to disable; the Record* methods are nil-safe.
func (h *Handler) SetDecompressMetrics(m *DecompressMetrics) {
	h.metrics = m
}

// SetEncryption wires the backup decryptor for `.enc` objects (T-0075).
// Both args may be nil; encrypted objects without a wired decryptor are
// served as raw ciphertext (CPE will fail to apply — visible in metrics).
func (h *Handler) SetEncryption(enc backup.Encryptor, m *backup.PolicyMetrics) {
	h.encryptor = enc
	h.encMetrics = m
}

// SetDecryptSemaphore wires the concurrency cap that bounds the 64MB×N
// memory amplification on maybeDecrypt (T-0089). Pass nil to disable
// (preserves T-0075 baseline). Constructed separately from SetEncryption
// so deployments can tune concurrency without touching the encryption key
// path.
func (h *Handler) SetDecryptSemaphore(s *DecryptSemaphore) {
	h.decryptSem = s
}

// ServeHTTP handles download requests.
// Route: GET /smallcell/FileDownloadService/{bucket}/{objectPath...}
// The bucket and object path are extracted from the URL path.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	h.clearWriteDeadline(w)

	// Validate Basic Auth if configured (constant-time comparison to prevent timing attacks)
	runtimeCfg := h.currentSettings(r.Context())
	if runtimeCfg.Username != "" {
		username, password, ok := r.BasicAuth()
		userMatch := subtle.ConstantTimeCompare([]byte(username), []byte(runtimeCfg.Username)) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(password), []byte(runtimeCfg.Password)) == 1
		if !ok || !userMatch || !passMatch {
			w.Header().Set("WWW-Authenticate", `Basic realm="FileDownload"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Parse bucket and object path from URL: /smallcell/FileDownloadService/{bucket}/{path...}
	const prefix = "/smallcell/FileDownloadService/"
	path := r.URL.Path
	if !strings.HasPrefix(path, prefix) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	remainder := path[len(prefix):]
	if remainder == "" {
		http.Error(w, "missing bucket and path", http.StatusBadRequest)
		return
	}

	// Split into bucket and object path
	slashIdx := strings.IndexByte(remainder, '/')
	if slashIdx <= 0 {
		http.Error(w, "missing object path", http.StatusBadRequest)
		return
	}
	bucket := remainder[:slashIdx]
	objectPath := remainder[slashIdx+1:]
	if objectPath == "" {
		http.Error(w, "missing object path", http.StatusBadRequest)
		return
	}

	// Path traversal protection: clean paths and reject any ".." traversal
	bucket = pathpkg.Clean(bucket)
	objectPath = pathpkg.Clean(objectPath)
	if strings.Contains(bucket, "..") || strings.Contains(objectPath, "..") ||
		strings.HasPrefix(bucket, "/") || strings.HasPrefix(objectPath, "/") {
		h.logger.Warn("path traversal attempt blocked",
			zap.String("bucket", bucket),
			zap.String("path", objectPath),
			zap.String("remote_addr", r.RemoteAddr))
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	bucket = appconfig.NormalizeConfigBackupBucket(bucket)

	// Fetch object from MinIO
	obj, err := h.minioClient.GetObject(r.Context(), bucket, objectPath, minio.GetObjectOptions{})
	if err != nil {
		h.logger.Error("get object from minio",
			zap.Error(err),
			zap.String("bucket", bucket),
			zap.String("path", objectPath))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer obj.Close()

	// Get object info for Content-Length and validation
	info, err := obj.Stat()
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" || errResp.Code == "NoSuchBucket" {
			h.logger.Warn("object not found",
				zap.String("bucket", bucket),
				zap.String("path", objectPath))
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		h.logger.Error("stat object",
			zap.Error(err),
			zap.String("bucket", bucket),
			zap.String("path", objectPath))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("file download",
		zap.String("bucket", bucket),
		zap.String("path", objectPath),
		zap.Int64("size", info.Size),
		zap.String("remote_addr", r.RemoteAddr))

	// T-0075: peel optional `.enc` suffix first. If matched and the
	// decryptor is wired, buffer the entire object (capped) and decrypt
	// using AAD = filename (without `.enc`). The result feeds back into
	// the T-0072 decompression path as a normal io.Reader, so a file
	// like `cfg.xml.gz.enc` becomes plaintext XML at the wire.
	pathForCompression := objectPath
	innerName := filepath.Base(objectPath)
	encReader, encActive, encErr := h.maybeDecrypt(r.Context(), obj, info.Size, objectPath)
	if encErr != nil {
		// Decryption failure is not fall-through: failing closed stops the
		// CPE from getting ciphertext-shaped bytes that would never apply.
		// Metrics recorded inside maybeDecrypt / DecryptSemaphore.
		h.logger.Warn("decrypt failed; aborting download",
			zap.String("path", objectPath), zap.Error(encErr))
		// T-0089 status mapping. Order matters: ctx cancel → client gone,
		// don't bother writing; semaphore timeout → 503 + Retry-After;
		// auth/format → 422; everything else → 500.
		switch {
		case errors.Is(encErr, context.Canceled), errors.Is(encErr, context.DeadlineExceeded):
			return
		case errors.Is(encErr, ErrDecryptSemaphoreTimeout):
			w.Header().Set("Retry-After", "5")
			http.Error(w, "decrypt overload; retry later", http.StatusServiceUnavailable)
			return
		case errors.Is(encErr, backup.ErrEncryptionAuthFailed),
			errors.Is(encErr, backup.ErrEncryptionFormatInvalid):
			http.Error(w, "decryption failed", http.StatusUnprocessableEntity)
			return
		default:
			http.Error(w, "decryption failed", http.StatusInternalServerError)
			return
		}
	}
	if encActive {
		pathForCompression = strings.TrimSuffix(objectPath, "."+h.encryptor.Extension())
		innerName = filepath.Base(pathForCompression)
	}

	// T-0072: detect compressed extension; if matched, wrap the (possibly
	// decrypted) stream with a decompressor so CPE receives plaintext.
	// Filename header loses the compressed extension so consumers see the
	// canonical name.
	decomp, cleanName, doDecompress := detectCompression(pathForCompression)
	var sourceReader io.Reader
	if encActive {
		sourceReader = encReader
	} else {
		sourceReader = obj
	}
	filename := innerName
	w.Header().Set("Content-Type", "application/octet-stream")

	if doDecompress {
		body, err := decomp.Wrap(sourceReader)
		if err != nil {
			h.metrics.RecordError(decomp.Format(), "open")
			h.logger.Warn("decompress wrap failed; falling back to raw stream",
				zap.String("format", decomp.Format()),
				zap.String("path", objectPath),
				zap.Error(err))
			// Fall through to pass-through path (preserve compressed bytes).
		} else {
			defer body.Close()
			filename = filepath.Base(cleanName)
			// Content-Length is unknown after streaming decompression; do NOT
			// set it (chunked transfer or close-delimited body is fine).
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
			if r.Method == http.MethodHead {
				return
			}
			if _, copyErr := io.Copy(w, body); copyErr != nil {
				h.metrics.RecordError(decomp.Format(), "copy")
				h.logger.Warn("decompressed stream to client failed",
					zap.Error(copyErr),
					zap.String("format", decomp.Format()),
					zap.String("path", objectPath))
				return
			}
			h.metrics.RecordSuccess(decomp.Format())
			return
		}
	}

	// Pass-through path: serve the (possibly decrypted) bytes. When
	// encryption was active but compression was not, the decrypted plaintext
	// has unknown size — Content-Length is omitted; otherwise we use the
	// original MinIO object size.
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if !encActive {
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
	}

	if r.Method == http.MethodHead {
		return
	}

	if _, err := io.Copy(w, sourceReader); err != nil {
		h.logger.Warn("stream file to client",
			zap.Error(err),
			zap.String("bucket", bucket),
			zap.String("path", objectPath))
	}
}

func (h *Handler) clearWriteDeadline(w http.ResponseWriter) {
	err := http.NewResponseController(w).SetWriteDeadline(time.Time{})
	if err != nil && !errors.Is(err, http.ErrNotSupported) {
		h.logger.Warn("clear file download write deadline", zap.Error(err))
	}
}

func (h *Handler) currentSettings(ctx context.Context) transfercfg.DownloadSettings {
	if h.runtimeProvider != nil {
		return h.runtimeProvider.Snapshot(ctx).Download
	}
	return transfercfg.DownloadSettings{
		Username: h.username,
		Password: h.password,
	}
}

// maybeDecrypt returns (reader, encActive, error). When the object path ends
// with the encryptor's extension AND a decryptor is wired, the entire object
// is buffered (capped at 64MB + a little slack for the encryption envelope
// overhead) and decrypted. Returns the plaintext bytes wrapped in
// bytes.Reader so subsequent decompression can stream from it.
//
// encActive=false means "no decryption applied" — caller should serve the
// raw obj. error is non-nil only when an encrypted object failed to decrypt
// (auth tag mismatch, malformed format, key unavailable, semaphore reject).
// Decryption is fail-closed: the caller does NOT fall through to ciphertext
// on error.
//
// T-0089: when h.decryptSem is wired, Acquire is called BEFORE the 64MB
// buffer allocation — cap the worst-case memory amplification to
// (limit × 64MB). Acquired slot released via defer.
func (h *Handler) maybeDecrypt(ctx context.Context, obj io.Reader, size int64, objectPath string) (io.Reader, bool, error) {
	if h.encryptor == nil {
		return obj, false, nil
	}
	suffix := "." + h.encryptor.Extension()
	if !strings.HasSuffix(objectPath, suffix) {
		return obj, false, nil
	}
	// Cap at 64MB + 1KB envelope tolerance — same ceiling as upload-side.
	// Review fix MED-3 (T-0089): the size pre-check happens BEFORE
	// semaphore Acquire so attackers spamming oversize objects can't churn
	// slots — they get the 422 without ever taking a slot.
	const maxEncryptedSize = 64*1024*1024 + 1024
	if size > 0 && size > maxEncryptedSize {
		h.encMetrics.RecordBackupDecryptionError("format_invalid")
		return nil, false, fmt.Errorf("encrypted object exceeds %d bytes: %w",
			maxEncryptedSize, backup.ErrEncryptionFormatInvalid)
	}
	// T-0089: bound concurrent buffer allocations. nil-safe — when the
	// semaphore is not wired this is a no-op.
	if _, err := h.decryptSem.Acquire(ctx); err != nil {
		return nil, false, err
	}
	defer h.decryptSem.Release()
	limited := io.LimitReader(obj, maxEncryptedSize+1)
	blob, err := io.ReadAll(limited)
	if err != nil {
		h.encMetrics.RecordBackupDecryptionError("format_invalid")
		return nil, false, fmt.Errorf("buffer encrypted object: %w", err)
	}
	if int64(len(blob)) > maxEncryptedSize {
		h.encMetrics.RecordBackupDecryptionError("format_invalid")
		return nil, false, fmt.Errorf("encrypted object stream truncation read >%d: %w",
			maxEncryptedSize, backup.ErrEncryptionFormatInvalid)
	}
	innerPath := strings.TrimSuffix(objectPath, suffix)
	aad := []byte(filepath.Base(innerPath))
	plaintext, err := h.encryptor.Decrypt(blob, aad)
	if err != nil {
		switch {
		case errors.Is(err, backup.ErrEncryptionAuthFailed):
			h.encMetrics.RecordBackupDecryptionError("tamper")
		case errors.Is(err, backup.ErrEncryptionKeyUnavailable):
			h.encMetrics.RecordBackupDecryptionError("key_unavailable")
		default:
			h.encMetrics.RecordBackupDecryptionError("format_invalid")
		}
		return nil, false, err
	}
	return bytes.NewReader(plaintext), true, nil
}
