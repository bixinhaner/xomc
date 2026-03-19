package upload

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"go.uber.org/zap"
)

// Handler handles HTTP file upload requests.
type Handler struct {
	tokenManager *TokenManager
	sessionStore *SessionStore
	minioClient  *minio.Client
	maxFileSize  int64
	buckets      appconfig.BucketConfig
	logger       *zap.Logger
}

// NewHandler creates a new upload Handler.
func NewHandler(
	tokenManager *TokenManager,
	sessionStore *SessionStore,
	minioClient *minio.Client,
	maxFileSize int64,
	buckets appconfig.BucketConfig,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		tokenManager: tokenManager,
		sessionStore: sessionStore,
		minioClient:  minioClient,
		maxFileSize:  maxFileSize,
		buckets:      buckets,
		logger:       logger,
	}
}

// ServeHTTP handles upload requests.
// Route: PUT /upload/{token}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Extract token from path
	token := h.extractToken(r.URL.Path)
	if token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	// 2. Validate token
	claims, err := h.tokenManager.Validate(token)
	if err != nil {
		h.logger.Warn("invalid upload token", zap.Error(err))
		http.Error(w, "invalid token", http.StatusBadRequest)
		return
	}

	// 3. Check file size
	if h.maxFileSize > 0 && r.ContentLength > h.maxFileSize {
		http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
		return
	}

	// 4. Determine bucket and object path
	bucket := h.bucketForFileType(claims.FileType)
	objectPath := h.objectPath(claims.DeviceSN, claims.TargetFileName)

	// 5. Stream upload to MinIO
	ctx := r.Context()
	info, err := h.minioClient.PutObject(ctx, bucket, objectPath, r.Body, r.ContentLength, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		h.logger.Error("upload to minio failed",
			zap.Error(err),
			zap.String("device_sn", claims.DeviceSN),
			zap.String("path", objectPath),
		)
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}

	// 6. Save session for later TC correlation
	session := &Session{
		DeviceSN:       claims.DeviceSN,
		CommandKey:     claims.CommandKey,
		FileType:       claims.FileType,
		Bucket:         bucket,
		ObjectPath:     objectPath,
		FileSize:       info.Size,
		UploadedAt:     time.Now(),
		TargetFileName: claims.TargetFileName,
	}
	if err := h.sessionStore.Save(ctx, session); err != nil {
		h.logger.Warn("save upload session failed", zap.Error(err))
		// Don't fail the request - upload succeeded
	}

	h.logger.Info("file uploaded",
		zap.String("device_sn", claims.DeviceSN),
		zap.String("command_key", claims.CommandKey),
		zap.String("path", objectPath),
		zap.Int64("size", info.Size),
	)

	// 7. Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","path":"%s","size":%d}`, objectPath, info.Size)
}

func (h *Handler) extractToken(path string) string {
	// Path format: /upload/{token}
	parts := strings.Split(strings.TrimPrefix(path, "/upload/"), "/")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return ""
}

func (h *Handler) bucketForFileType(fileType string) string {
	switch fileType {
	case "4":
		return h.buckets.PMFiles
	case "5":
		return h.buckets.MRFiles
	case "6":
		return h.buckets.Logs
	default:
		return h.buckets.PMFiles
	}
}

func (h *Handler) objectPath(deviceSN, filename string) string {
	return fmt.Sprintf("%s/%s/%s",
		time.Now().Format("2006/01/02"),
		deviceSN,
		filename,
	)
}

// GetSession retrieves an upload session (for TC handler use).
func (h *Handler) GetSession(ctx context.Context, deviceSN, commandKey string) (*Session, error) {
	return h.sessionStore.Get(ctx, deviceSN, commandKey)
}

// DeleteSession removes an upload session.
func (h *Handler) DeleteSession(ctx context.Context, deviceSN, commandKey string) error {
	return h.sessionStore.Delete(ctx, deviceSN, commandKey)
}
