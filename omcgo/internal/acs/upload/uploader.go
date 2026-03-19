package upload

import (
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/appconfig"
)

// Uploader generates upload URLs and related information.
type Uploader struct {
	tokenManager *TokenManager
	sessionStore *SessionStore
	config       appconfig.UploadConfig
	buckets      appconfig.BucketConfig
}

// NewUploader creates a new Uploader.
func NewUploader(
	tokenManager *TokenManager,
	sessionStore *SessionStore,
	config appconfig.UploadConfig,
	buckets appconfig.BucketConfig,
) *Uploader {
	return &Uploader{
		tokenManager: tokenManager,
		sessionStore: sessionStore,
		config:       config,
		buckets:      buckets,
	}
}

// UploadInfo contains upload URL and related information.
type UploadInfo struct {
	URL        string // Full upload URL for CPE
	Token      string // JWT Token
	ObjectPath string // MinIO object path
	Bucket     string // MinIO bucket name
}

// GenerateUploadURL generates an upload URL for the given parameters.
func (u *Uploader) GenerateUploadURL(deviceSN, commandKey, fileType, filename string) (*UploadInfo, error) {
	// 1. Generate JWT token
	token, err := u.tokenManager.Generate(deviceSN, commandKey, fileType, filename)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// 2. Determine target bucket
	bucket := u.bucketForFileType(fileType)

	// 3. Generate object path
	objectPath := u.objectPath(deviceSN, filename)

	// 4. Build full upload URL
	uploadPath := u.config.Path
	if uploadPath == "" {
		uploadPath = "/upload"
	}
	uploadURL := fmt.Sprintf("%s%s/%s", u.config.BaseURL, uploadPath, token)

	return &UploadInfo{
		URL:        uploadURL,
		Token:      token,
		ObjectPath: objectPath,
		Bucket:     bucket,
	}, nil
}

func (u *Uploader) bucketForFileType(fileType string) string {
	switch fileType {
	case "4": // Vendor Configuration File / PM
		return u.buckets.PMFiles
	case "5": // Log File / MR
		return u.buckets.MRFiles
	case "6": // Log
		return u.buckets.Logs
	default:
		return u.buckets.PMFiles
	}
}

func (u *Uploader) objectPath(deviceSN, filename string) string {
	now := time.Now()
	return fmt.Sprintf("%s/%s/%s",
		now.Format("2006/01/02"),
		deviceSN,
		filename,
	)
}

// Buckets returns the bucket configuration for external use.
func (u *Uploader) Buckets() appconfig.BucketConfig {
	return u.buckets
}
