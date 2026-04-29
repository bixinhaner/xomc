package download

import (
	"crypto/subtle"
	"fmt"
	"io"
	"net/http"
	pathpkg "path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

// Handler handles HTTP file download requests from CPE devices.
// Endpoint: GET /smallcell/FileDownloadService/{bucket}/{objectPath...}
// Authentication: HTTP Basic Auth with global credentials from config.
// CPE accesses this endpoint using the URL and credentials provided in the Download SOAP RPC.
type Handler struct {
	minioClient *minio.Client
	logger      *zap.Logger
	username    string
	password    string
	// T-0072: optional metrics for on-the-fly decompression. nil-safe — Record*
	// methods short-circuit on nil receiver, so wiring is optional.
	metrics *DecompressMetrics
}

// NewHandler creates a new download handler.
func NewHandler(minioClient *minio.Client, username, password string, logger *zap.Logger) *Handler {
	return &Handler{
		minioClient: minioClient,
		username:    username,
		password:    password,
		logger:      logger,
	}
}

// SetDecompressMetrics wires Prometheus collectors for on-the-fly decompression.
// Pass nil to disable; the Record* methods are nil-safe.
func (h *Handler) SetDecompressMetrics(m *DecompressMetrics) {
	h.metrics = m
}

// ServeHTTP handles download requests.
// Route: GET /smallcell/FileDownloadService/{bucket}/{objectPath...}
// The bucket and object path are extracted from the URL path.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate Basic Auth if configured (constant-time comparison to prevent timing attacks)
	if h.username != "" {
		username, password, ok := r.BasicAuth()
		userMatch := subtle.ConstantTimeCompare([]byte(username), []byte(h.username)) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(password), []byte(h.password)) == 1
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

	// T-0072: detect compressed extension; if matched, wrap the MinIO object
	// stream with a decompressor so CPE receives plaintext. Filename header
	// loses the compressed extension so consumers see the canonical name.
	decomp, cleanName, doDecompress := detectCompression(objectPath)
	filename := filepath.Base(objectPath)
	w.Header().Set("Content-Type", "application/octet-stream")

	if doDecompress {
		body, err := decomp.Wrap(obj)
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

	// Pass-through path: serve raw object with Content-Length.
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))

	if r.Method == http.MethodHead {
		return
	}

	if _, err := io.Copy(w, obj); err != nil {
		h.logger.Warn("stream file to client",
			zap.Error(err),
			zap.String("bucket", bucket),
			zap.String("path", objectPath))
	}
}
