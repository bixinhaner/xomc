package minio

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/omcgo/omcgo/internal/core/appconfig"
)

// rawFileRetentionDays 是 PM/MR 原始文件桶（pm-files / mr-files）的自动过期天数（#169）。
// KPI/聚合结果落 PG/TimescaleDB；原始 XML 仅供回溯/重算，14 天后由 MinIO ILM 自动过期删除，
// 防「设备数 × 每天文件数 × 保留期」把盘单调撑满。其它桶（firmware/config_backup/logs 等）
// 内容需持久，不设此策略。
const rawFileRetentionDays = 14

// NewMinIOClient creates a new MinIO client for internal traffic
// (后端 ↔ MinIO，走 cfg.Endpoint，通常是 docker 内网名 / k8s service)。
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

// NewPresignClient returns a MinIO client whose endpoint is configured for
// browser / external SDK consumption. 仅用于生成预签名 URL (PresignedGetObject /
// PresignedPutObject)，不应用于实际 Put/Get/Stat（那些走 NewMinIOClient）。
//
// 行为：
//   - cfg.PublicEndpoint 非空 → 新建 client 用 PublicEndpoint 当 endpoint；
//     签名基于 endpoint host 计算，所以签出来的 URL 浏览器能直接 GET。
//   - cfg.PublicEndpoint 为空 → 回退用 NewMinIOClient 的内部 endpoint
//     （此时签的 URL 浏览器可能无法解析 host，需要部署 reverse proxy 或调用方
//     自行处理 — 仅适合后端 ↔ MinIO 同主机部署）。
//
// 关键细节：MinIO Go SDK 默认在 PresignedGetObject 前会发 GetBucketLocation
// 探测 region；当 endpoint 是 "宿主机 host"（如 localhost:9000）但调用方在
// docker 容器内时，这个探测会 dial ::1:9000 失败 → 整个预签名失败。
// 显式 Region: "us-east-1"（MinIO 默认 region）后 SDK 跳过探测，纯客户端
// 计算签名，不再发任何 HTTP 请求。
func NewPresignClient(cfg appconfig.MinIOConfig) (*minio.Client, error) {
	if cfg.PublicEndpoint == "" {
		return NewMinIOClient(cfg)
	}
	client, err := minio.New(cfg.PublicEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: "us-east-1",
	})
	if err != nil {
		return nil, fmt.Errorf("create MinIO presign client: %w", err)
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
		cfg.Exchange,
		cfg.UIAssets,
		cfg.TraceBulk,
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

	// #169：原始文件桶（pm-files / mr-files）设 14 天过期，防 PM/MR 文件把盘单调塞满；幂等。
	// 其它桶内容需持久，不设。失败由调用方降级为 warn（infra.go），不阻塞启动。
	for _, b := range []string{cfg.PMFiles, cfg.MRFiles} {
		if err := ensureRawFileLifecycle(ctx, client, b); err != nil {
			return err
		}
	}
	return nil
}

// rawFileLifecycleConfig 构造原始文件桶的 N 天过期 ILM 配置（纯函数，便于单测）。
func rawFileLifecycleConfig() *lifecycle.Configuration {
	lc := lifecycle.NewConfiguration()
	lc.Rules = []lifecycle.Rule{{
		ID:         fmt.Sprintf("omc-raw-expire-%dd", rawFileRetentionDays),
		Status:     "Enabled",
		RuleFilter: lifecycle.Filter{Prefix: ""}, // 空前缀 = 整桶所有对象
		Expiration: lifecycle.Expiration{Days: lifecycle.ExpirationDays(rawFileRetentionDays)},
	}}
	return lc
}

// ensureRawFileLifecycle 幂等设置原始文件桶的过期生命周期（重设覆盖）。bucket 为空跳过。
func ensureRawFileLifecycle(ctx context.Context, client *minio.Client, bucket string) error {
	if bucket == "" {
		return nil
	}
	if err := client.SetBucketLifecycle(ctx, bucket, rawFileLifecycleConfig()); err != nil {
		return fmt.Errorf("set lifecycle on bucket %s: %w", bucket, err)
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
