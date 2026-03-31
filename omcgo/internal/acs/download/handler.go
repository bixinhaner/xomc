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

	// Set response headers
	filename := filepath.Base(objectPath)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))

	if r.Method == http.MethodHead {
		return
	}

	// Stream file to client.
	// Note: once the first byte is written, the 200 status is committed and
	// any subsequent io.Copy error cannot be surfaced to the client via HTTP status.
	if _, err := io.Copy(w, obj); err != nil {
		h.logger.Warn("stream file to client",
			zap.Error(err),
			zap.String("bucket", bucket),
			zap.String("path", objectPath))
	}
}
