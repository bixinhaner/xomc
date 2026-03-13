package minio

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/omcgo/omcgo/internal/core/appconfig"
)

// NewMinIOClient creates a new MinIO client.
func NewMinIOClient(cfg appconfig.MinIOConfig) (*minio.Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create MinIO client: %w", err)
	}
	return client, nil
}

// EnsureBuckets creates all required buckets if they don't exist.
func EnsureBuckets(ctx context.Context, client *minio.Client, cfg appconfig.BucketConfig) error {
	buckets := []string{
		cfg.PMFiles,
		cfg.MRFiles,
		cfg.Firmware,
		cfg.ConfigBackup,
		cfg.Logs,
		cfg.Reports,
	}

	for _, bucket := range buckets {
		if bucket == "" {
			continue
		}
		exists, err := client.BucketExists(ctx, bucket)
		if err != nil {
			return fmt.Errorf("check bucket %s: %w", bucket, err)
		}
		if !exists {
			if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
				return fmt.Errorf("create bucket %s: %w", bucket, err)
			}
		}
	}
	return nil
}

// MinIOHealthCheck verifies the MinIO connection is alive.
func MinIOHealthCheck(ctx context.Context, client *minio.Client) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("MinIO health check: %w", err)
	}
	return nil
}
