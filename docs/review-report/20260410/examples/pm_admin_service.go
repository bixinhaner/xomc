// Package pm 演示 Goose + sqlc + pgx 三者配合使用
//
// 本示例展示:
// 1. Goose: 执行 DDL 迁移 (创建超表)
// 2. sqlc: 生成类型安全的查询代码
// 3. pgx: 运行时管理操作 (设置保留策略)
package pm

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// PMAdminService PM 管理服務
// 演示如何使用 pgx 原生调用 TimescaleDB 管理函数
type PMAdminService struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewPMAdminService 创建 PM 管理服务
func NewPMAdminService(pool *pgxpool.Pool, logger *zap.Logger) *PMAdminService {
	return &PMAdminService{
		pool:   pool,
		logger: logger,
	}
}

// SetRetentionPolicy 设置数据保留策略
// 这是运行时管理操作,不能使用 Goose 或 sqlc
func (s *PMAdminService) SetRetentionPolicy(ctx context.Context, tableName string, retentionDays int) error {
	s.logger.Info("setting retention policy",
		zap.String("table", tableName),
		zap.Int("retention_days", retentionDays))

	_, err := s.pool.Exec(ctx, `
		SELECT add_retention_policy($1, 
			drop_after => INTERVAL $2 || ' days',
			if_not_exists => TRUE
		)
	`, tableName, retentionDays)

	if err != nil {
		return fmt.Errorf("set retention policy: %w", err)
	}

	s.logger.Info("retention policy set successfully",
		zap.String("table", tableName),
		zap.Int("retention_days", retentionDays))

	return nil
}

// SetCompressionPolicy 设置数据压缩策略
// 这是运行时管理操作
func (s *PMAdminService) SetCompressionPolicy(ctx context.Context, tableName string, compressAfterDays int) error {
	s.logger.Info("setting compression policy",
		zap.String("table", tableName),
		zap.Int("compress_after_days", compressAfterDays))

	_, err := s.pool.Exec(ctx, `
		SELECT add_compression_policy($1, 
			compress_after => INTERVAL $2 || ' days',
			if_not_exists => TRUE
		)
	`, tableName, compressAfterDays)

	if err != nil {
		return fmt.Errorf("set compression policy: %w", err)
	}

	s.logger.Info("compression policy set successfully",
		zap.String("table", tableName),
		zap.Int("compress_after_days", compressAfterDays))

	return nil
}

// RemoveRetentionPolicy 移除数据保留策略
func (s *PMAdminService) RemoveRetentionPolicy(ctx context.Context, tableName string) error {
	s.logger.Info("removing retention policy", zap.String("table", tableName))

	_, err := s.pool.Exec(ctx, `
		SELECT remove_retention_policy($1, if_exists => TRUE)
	`, tableName)

	if err != nil {
		return fmt.Errorf("remove retention policy: %w", err)
	}

	s.logger.Info("retention policy removed", zap.String("table", tableName))
	return nil
}

// GetHypertableInfo 获取超表信息
// 演示如何查询 TimescaleDB 系统视图
func (s *PMAdminService) GetHypertableInfo(ctx context.Context, tableName string) (*HypertableInfo, error) {
	query := `
		SELECT 
			hypertable_name,
			num_dimensions,
			num_chunks,
			compression_status,
			total_bytes,
			dictionary_bytes,
			index_bytes,
			toast_bytes
		FROM timescaledb_information.hypertables
		WHERE hypertable_name = $1
	`

	var info HypertableInfo
	err := s.pool.QueryRow(ctx, query, tableName).Scan(
		&info.HypertableName,
		&info.NumDimensions,
		&info.NumChunks,
		&info.CompressionStatus,
		&info.TotalBytes,
		&info.DictionaryBytes,
		&info.IndexBytes,
		&info.ToastBytes,
	)

	if err != nil {
		return nil, fmt.Errorf("query hypertable info: %w", err)
	}

	return &info, nil
}

// HypertableInfo 超表信息
type HypertableInfo struct {
	HypertableName    string
	NumDimensions     int32
	NumChunks         int32
	CompressionStatus string
	TotalBytes        int64
	DictionaryBytes   int64
	IndexBytes        int64
	ToastBytes        int64
}

// CreateContinuousAggregate 创建连续聚合
// 这是 DDL 操作,通常在迁移文件中执行,但也可以通过 pgx 动态创建
func (s *PMAdminService) CreateContinuousAggregate(ctx context.Context, viewName string, tableName string, bucketInterval time.Duration) error {
	s.logger.Info("creating continuous aggregate",
		zap.String("view", viewName),
		zap.String("table", tableName),
		zap.Duration("bucket_interval", bucketInterval))

	query := fmt.Sprintf(`
		CREATE MATERIALIZED VIEW IF NOT EXISTS %s
		WITH (timescaledb.continuous) AS
		SELECT
			time_bucket('%s', time) AS bucket,
			device_id,
			counter_name,
			AVG(counter_value) AS avg_value,
			MAX(counter_value) AS max_value,
			MIN(counter_value) AS min_value,
			COUNT(*) AS sample_count
		FROM %s
		GROUP BY bucket, device_id, counter_name
		WITH NO DATA;
	`, viewName, bucketInterval.String(), tableName)

	_, err := s.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("create continuous aggregate: %w", err)
	}

	s.logger.Info("continuous aggregate created", zap.String("view", viewName))
	return nil
}

// SetContinuousAggregatePolicy 设置连续聚合刷新策略
func (s *PMAdminService) SetContinuousAggregatePolicy(ctx context.Context, viewName string, refreshInterval time.Duration) error {
	s.logger.Info("setting continuous aggregate policy",
		zap.String("view", viewName),
		zap.Duration("refresh_interval", refreshInterval))

	query := fmt.Sprintf(`
		SELECT add_continuous_aggregate_policy('%s',
			start_offset => INTERVAL '%s',
			end_offset => INTERVAL '1 hour',
			schedule_interval => INTERVAL '%s',
			if_not_exists => TRUE
		)
	`, viewName, (refreshInterval * 2).String(), refreshInterval.String())

	_, err := s.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("set continuous aggregate policy: %w", err)
	}

	s.logger.Info("continuous aggregate policy set", zap.String("view", viewName))
	return nil
}
package examples
