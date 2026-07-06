package alarm

import (
	"context"
	"time"

	"go.uber.org/zap"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Reconciler is a background job that brings PostgreSQL active alarm stats
// into Redis to ensure eventual consistency, replacing the need for row-level locks on device_info.
type Reconciler struct {
	pool       *pgxpool.Pool
	redisStore *RedisAlarmStore
	logger     *zap.Logger
	stopChan   chan struct{}
}

func NewReconciler(pool *pgxpool.Pool, redisStore *RedisAlarmStore, logger *zap.Logger) *Reconciler {
	return &Reconciler{
		pool:       pool,
		redisStore: redisStore,
		logger:     logger.Named("alarm_reconciler"),
		stopChan:   make(chan struct{}),
	}
}

func (r *Reconciler) Start() {
	go r.run()
}

func (r *Reconciler) Stop() {
	close(r.stopChan)
}

func (r *Reconciler) run() {
	r.logger.Info("alarm reconciler started")
	// Run immediately once
	r.doReconcile()
	
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopChan:
			r.logger.Info("alarm reconciler stopped")
			return
		case <-ticker.C:
			r.doReconcile()
		}
	}
}

func (r *Reconciler) doReconcile() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Use a CTE to aggregate active_alarm_count, highest severity, and highest severity count per device.
	query := `
		WITH device_stats AS (
			SELECT 
				device_id,
				COUNT(*) as total_active,
				MIN(severity) as highest_severity
			FROM alarms_active
			GROUP BY device_id
		),
		highest_counts AS (
			SELECT 
				a.device_id, 
				COUNT(*) as highest_count
			FROM alarms_active a
			JOIN device_stats ds ON a.device_id = ds.device_id AND a.severity = ds.highest_severity
			GROUP BY a.device_id
		)
		SELECT 
			ds.device_id, 
			ds.total_active, 
			ds.highest_severity, 
			hc.highest_count
		FROM device_stats ds
		JOIN highest_counts hc ON ds.device_id = hc.device_id
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		r.logger.Error("failed to query alarm stats for reconciliation", zap.Error(err))
		return
	}
	defer rows.Close()

	pipe := r.redisStore.client.Pipeline()
	count := 0

	for rows.Next() {
		var (
			deviceID     string
			totalActive  int
			highestSev   int
			highestCount int
		)
		if err := rows.Scan(&deviceID, &totalActive, &highestSev, &highestCount); err != nil {
			r.logger.Error("failed to scan alarm stats row", zap.Error(err))
			continue
		}

		key := alarmStatsKey(deviceID)
		pipe.HSet(ctx, key, "active_count", totalActive)
		pipe.HSet(ctx, key, "highest_severity", highestSev)
		pipe.HSet(ctx, key, "highest_count", highestCount)
		pipe.Expire(ctx, key, 24*time.Hour)
		count++
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("row iteration error during alarm reconciliation", zap.Error(err))
	}

	if count > 0 {
		_, err := pipe.Exec(ctx)
		if err != nil {
			r.logger.Error("failed to execute pipeline during alarm reconciliation", zap.Error(err))
		} else {
			r.logger.Debug("successfully reconciled alarm stats to redis", zap.Int("devices_updated", count))
		}
	}
}
