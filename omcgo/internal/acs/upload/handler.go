package upload

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/omcgo/omcgo/internal/core/appconfig"
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
	logger       *zap.Logger
	// Global credentials for upload authentication
	username string
	password string
}

// NewHandler creates a new upload Handler.
func NewHandler(
	tokenManager *TokenManager,
	sessionStore *SessionStore,
	minioClient *minio.Client,
	maxFileSize int64,
	buckets appconfig.BucketConfig,
	username, password string,
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
		logger:       logger,
	}
}

// ServeHTTP handles upload requests.
// Route: POST /smallcell/FileUploadService?fileType={type}&filename={name}
// Auth: HTTP Basic Authentication with global credentials
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Only allow POST method
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 2. Validate Basic Auth credentials
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
	bucket := h.bucketForFileType(fileType)
	objectPath := h.objectPath(fileType, filename)

	// 6. Stream upload to MinIO
	ctx := r.Context()
	info, err := h.minioClient.PutObject(ctx, bucket, objectPath, r.Body, r.ContentLength, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		h.logger.Error("upload to minio failed",
			zap.Error(err),
			zap.String("file_type", fileType),
			zap.String("path", objectPath),
		)
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}

	h.logger.Info("file uploaded",
		zap.String("file_type", fileType),
		zap.String("filename", filename),
		zap.String("path", objectPath),
		zap.Int64("size", info.Size),
	)

	// 7. Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","path":"%s","size":%d}`, objectPath, info.Size)
}

func (h *Handler) bucketForFileType(fileType string) string {
	switch fileType {
	case "4", "PM":
		return h.buckets.PMFiles
	case "5", "MR":
		return h.buckets.MRFiles
	case "6", "Log", "LOG":
		return h.buckets.Logs
	default:
		return h.buckets.PMFiles
	}
}

// objectPath generates MinIO object path with organized directory structure.
// Format: {fileType}/{YYYY}/{MM}/{DD}/{filename}
// Example: pm/2026/03/20/pm_20260320.xml
func (h *Handler) objectPath(fileType, filename string) string {
	now := time.Now()
	typeDir := h.typeDirectory(fileType)
	return fmt.Sprintf("%s/%s/%s",
		typeDir,
		now.Format("2006/01/02"),
		filename,
	)
}

// typeDirectory maps file types to MinIO directory names
func (h *Handler) typeDirectory(fileType string) string {
	switch fileType {
	case "4", "PM":
		return "pm"
	case "5", "MR":
		return "mr"
	case "6", "Log", "LOG":
		return "logs"
	default:
		return "uploads"
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
