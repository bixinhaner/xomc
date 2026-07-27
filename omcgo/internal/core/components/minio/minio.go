package minio

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"go.uber.org/zap"
)

// DefaultRawFileRetentionDays 是 PM/MR 原始文件桶（pm-files / mr-files）自动过期天数的
// 兜底默认值（#169 / #319）。KPI/聚合结果落 PG/TimescaleDB；原始 XML 仅供回溯/重算，到期后
// 由 MinIO ILM 自动过期删除，防「设备数 × 每天文件数 × 保留期」把盘单调撑满。其它桶
// （firmware/config-backup/logs 等）内容需持久，不设此策略。
//
// issue #319：天数改为可配（sys_configs minio.retention.raw_object_days）。只有 app
// 进程的持久化 ILM applier 有权设置它；ACS/worker 的 EnsureBuckets 只建桶，不能覆盖配置。
const DefaultRawFileRetentionDays = 60

const lifecycleRollbackTimeout = 10 * time.Second

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
//
// 可选 logger（变参，传 0 或 1 个）：当 PublicEndpoint 为空回退内部 endpoint 时
// 打 Warn 级日志，把「该配 public_endpoint 否则浏览器解析不了」显式化（qa-614 #377）。
// 不传 logger 时静默回退（兼容既有调用点）。
func NewPresignClient(cfg appconfig.MinIOConfig, log ...*zap.Logger) (*minio.Client, error) {
	if cfg.PublicEndpoint == "" {
		if len(log) > 0 && log[0] != nil {
			log[0].Warn("MinIO public_endpoint 未配置，预签名 URL 将回退使用内部 endpoint；"+
				"浏览器 / 外部 SDK 可能解析失败（ERR_NAME_NOT_RESOLVED），"+
				"请在 prod/test/k8s 配置 minio.public_endpoint 指向对外可达地址",
				zap.String("internal_endpoint", cfg.Endpoint))
		}
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

type bucketCreator interface {
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error
}

// EnsureBuckets creates all required buckets if they don't exist.
func EnsureBuckets(ctx context.Context, client bucketCreator, cfg appconfig.BucketConfig) error {
	buckets := []string{
		cfg.PMFiles,
		cfg.MRFiles,
		cfg.Firmware,
		appconfig.NormalizeConfigBackupBucket(cfg.ConfigBackup),
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

	// Lifecycle is intentionally not written here. EnsureBuckets runs in app,
	// ACS, and worker; only the app-side persistent ILM applier is authoritative.
	// Otherwise any ACS/worker restart could overwrite the configured value.
	return nil
}

// rawFileLifecycleConfig 构造原始文件桶的 days 天过期 ILM 配置（纯函数，便于单测）。
func rawFileLifecycleConfig(days int) *lifecycle.Configuration {
	lc := lifecycle.NewConfiguration()
	lc.Rules = []lifecycle.Rule{{
		ID:         fmt.Sprintf("omc-raw-expire-%dd", days),
		Status:     "Enabled",
		RuleFilter: lifecycle.Filter{Prefix: ""}, // 空前缀 = 整桶所有对象
		Expiration: lifecycle.Expiration{Days: lifecycle.ExpirationDays(days)},
	}}
	return lc
}

// ensureRawFileLifecycle 幂等设置原始文件桶的 days 天过期生命周期（重设覆盖）。bucket 为空跳过。
type bucketLifecycleClient interface {
	GetBucketLifecycle(ctx context.Context, bucketName string) (*lifecycle.Configuration, error)
	SetBucketLifecycle(ctx context.Context, bucketName string, config *lifecycle.Configuration) error
}

func ensureRawFileLifecycle(ctx context.Context, client bucketLifecycleClient, bucket string, days int) error {
	if bucket == "" {
		return nil
	}
	if err := client.SetBucketLifecycle(ctx, bucket, rawFileLifecycleConfig(days)); err != nil {
		return fmt.Errorf("set lifecycle on bucket %s: %w", bucket, err)
	}
	actual, err := client.GetBucketLifecycle(ctx, bucket)
	if err != nil {
		return fmt.Errorf("read back lifecycle on bucket %s: %w", bucket, err)
	}
	if !rawFileLifecycleMatches(actual, days) {
		return fmt.Errorf("verify lifecycle on bucket %s: expected one enabled %d-day raw-file rule", bucket, days)
	}
	return nil
}

// ApplyRawFileLifecycle 对一组原始文件桶幂等重设 days 天过期 ILM（issue #319）。
// SetBucketLifecycle 是服务端整桶覆盖操作，可随时重设——app 进程在启动期及
// sys_configs(minio.retention.raw_object_days) 变更时调用本函数热更新天数。
// 多桶更新无法依赖 S3 事务；先保存原配置，任一设置/回读失败时执行补偿回滚，
// 避免 pm-files 与 mr-files 长时间处于不同保留策略。
func ApplyRawFileLifecycle(ctx context.Context, client *minio.Client, buckets []string, days int) error {
	if client == nil {
		return fmt.Errorf("MinIO lifecycle client is nil")
	}
	return applyRawFileLifecycle(ctx, client, buckets, days)
}

func applyRawFileLifecycle(ctx context.Context, client bucketLifecycleClient, buckets []string, days int) error {
	if client == nil {
		return fmt.Errorf("MinIO lifecycle client is nil")
	}
	if days <= 0 {
		return fmt.Errorf("raw-file lifecycle days must be positive")
	}

	previous := make(map[string]*lifecycle.Configuration, len(buckets))
	ordered := make([]string, 0, len(buckets))
	for _, bucket := range buckets {
		if bucket == "" {
			continue
		}
		config, err := getBucketLifecycleOrEmpty(ctx, client, bucket)
		if err != nil {
			return fmt.Errorf("read existing lifecycle on bucket %s: %w", bucket, err)
		}
		previous[bucket] = config
		ordered = append(ordered, bucket)
	}

	updated := make([]string, 0, len(ordered))
	for _, bucket := range ordered {
		// Include the current bucket in compensation: Set may have succeeded even
		// when the subsequent read-back verification fails.
		updated = append(updated, bucket)
		if err := ensureRawFileLifecycle(ctx, client, bucket, days); err != nil {
			rollbackCtx, cancelRollback := context.WithTimeout(context.WithoutCancel(ctx), lifecycleRollbackTimeout)
			rollbackErr := rollbackRawFileLifecycles(rollbackCtx, client, updated, previous)
			cancelRollback()
			if rollbackErr != nil {
				return fmt.Errorf("%w; compensating rollback failed: %v", err, rollbackErr)
			}
			return err
		}
	}
	return nil
}

func rollbackRawFileLifecycles(ctx context.Context, client bucketLifecycleClient, updated []string, previous map[string]*lifecycle.Configuration) error {
	var rollbackErrors []error
	for i := len(updated) - 1; i >= 0; i-- {
		bucket := updated[i]
		if err := client.SetBucketLifecycle(ctx, bucket, previous[bucket]); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("restore lifecycle on bucket %s: %w", bucket, err))
			continue
		}
		actual, err := getBucketLifecycleOrEmpty(ctx, client, bucket)
		if err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("read back restored lifecycle on bucket %s: %w", bucket, err))
			continue
		}
		if !lifecycleConfigurationsEqual(actual, previous[bucket]) {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("verify restored lifecycle on bucket %s: configuration mismatch", bucket))
		}
	}
	return errors.Join(rollbackErrors...)
}

func getBucketLifecycleOrEmpty(ctx context.Context, client bucketLifecycleClient, bucket string) (*lifecycle.Configuration, error) {
	config, err := client.GetBucketLifecycle(ctx, bucket)
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchLifecycleConfiguration" {
			return lifecycle.NewConfiguration(), nil
		}
		return nil, err
	}
	if config == nil {
		return lifecycle.NewConfiguration(), nil
	}
	return config, nil
}

func lifecycleConfigurationsEqual(left, right *lifecycle.Configuration) bool {
	leftXML, leftErr := xml.Marshal(left)
	rightXML, rightErr := xml.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftXML) == string(rightXML)
}

func rawFileLifecycleMatches(config *lifecycle.Configuration, days int) bool {
	if config == nil || len(config.Rules) != 1 {
		return false
	}
	rule := config.Rules[0]
	return rule.Status == "Enabled" && rule.RuleFilter.Prefix == "" && int(rule.Expiration.Days) == days
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
