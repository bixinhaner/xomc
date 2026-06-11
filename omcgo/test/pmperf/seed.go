package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// deviceSN 把设备序号映射成 SN，例如 ("KPILT", 1) → "KPILT-0000001"。
// 7 位零填充支撑到千万级设备；测试设备一律带该前缀，便于一键清理时按 LIKE 命中。
func deviceSN(prefix string, idx int) string {
	return fmt.Sprintf("%s-%07d", prefix, idx)
}

// snPattern 返回清理/统计用的 LIKE 模式，例如 "KPILT" → "KPILT-%"。
func snPattern(prefix string) string {
	return prefix + "-%"
}

func connectDB(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	// 压测注入 / 统计都是短事务，连接数不用很大；给足并发批量插入即可。
	cfg.MaxConns = 16
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(cctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}
	if err := pool.Ping(cctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return pool, nil
}

// seedDevices 向 devices 表注入 n 个测试设备（SN = prefix-000000X）。
// 用 ON CONFLICT DO NOTHING 幂等：重复跑不报错、已存在的设备跳过。
// 返回实际新插入行数（已存在的不计）。
func seedDevices(ctx context.Context, pool *pgxpool.Pool, prefix string, n int, oui, carrier, tech, productClass string) (int64, error) {
	const chunk = 500
	const cols = 7 // serial_number, oui, carrier, technology, product_class, manufacturer, model_name
	var inserted int64

	for start := 1; start <= n; start += chunk {
		end := start + chunk
		if end > n+1 {
			end = n + 1
		}
		var sb strings.Builder
		sb.WriteString(`INSERT INTO devices (serial_number, oui, carrier, technology, product_class, manufacturer, model_name) VALUES `)
		args := make([]any, 0, (end-start)*cols)
		ai := 1
		for i := start; i < end; i++ {
			if i > start {
				sb.WriteByte(',')
			}
			fmt.Fprintf(&sb, "($%d,$%d,$%d,$%d,$%d,$%d,$%d)", ai, ai+1, ai+2, ai+3, ai+4, ai+5, ai+6)
			ai += cols
			args = append(args, deviceSN(prefix, i), oui, carrier, tech, productClass, "KPILoadTest", "KPILoadTest-CPE")
		}
		// 分区表上不带冲突目标的 ON CONFLICT DO NOTHING（PG11+ 支持）兜住
		// (serial_number, carrier) 的部分唯一索引冲突。
		sb.WriteString(" ON CONFLICT DO NOTHING")

		cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		tag, err := pool.Exec(cctx, sb.String(), args...)
		cancel()
		if err != nil {
			return inserted, fmt.Errorf("seed devices [%d,%d): %w", start, end, err)
		}
		inserted += tag.RowsAffected()
	}
	return inserted, nil
}

// dbCounts 统计带测试前缀的入库现状：pm_metrics 行数、pm_files 已解析数、pm_files 总数。
type dbCounts struct {
	metricsRows int64
	filesParsed int64
	filesTotal  int64
}

func queryDBCounts(ctx context.Context, pool *pgxpool.Pool, prefix string) (dbCounts, error) {
	var c dbCounts
	pat := snPattern(prefix)
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := pool.QueryRow(cctx,
		`SELECT count(*) FROM pm_metrics WHERE device_sn LIKE $1`, pat,
	).Scan(&c.metricsRows); err != nil {
		return c, fmt.Errorf("count pm_metrics: %w", err)
	}
	if err := pool.QueryRow(cctx,
		`SELECT count(*) FILTER (WHERE parsed), count(*) FROM pm_files WHERE device_sn LIKE $1`, pat,
	).Scan(&c.filesParsed, &c.filesTotal); err != nil {
		return c, fmt.Errorf("count pm_files: %w", err)
	}
	return c, nil
}

// cleanupResult 记录一键清理删除的各表行数。
type cleanupResult struct {
	metrics int64
	files   int64
	devices int64
}

// cleanupAll 一键清除测试数据：pm_metrics（含 metric_type='kpi' 的 KPI 行）、
// pm_files、devices —— 全部按 SN 前缀 LIKE 命中。pm_metrics 无外键，删除顺序无要求，
// 设备放最后删。
func cleanupAll(ctx context.Context, pool *pgxpool.Pool, prefix string) (cleanupResult, error) {
	var r cleanupResult
	pat := snPattern(prefix)

	exec := func(sql string) (int64, error) {
		cctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		tag, err := pool.Exec(cctx, sql, pat)
		if err != nil {
			return 0, err
		}
		return tag.RowsAffected(), nil
	}

	var err error
	if r.metrics, err = exec(`DELETE FROM pm_metrics WHERE device_sn LIKE $1`); err != nil {
		return r, fmt.Errorf("delete pm_metrics: %w", err)
	}
	if r.files, err = exec(`DELETE FROM pm_files WHERE device_sn LIKE $1`); err != nil {
		return r, fmt.Errorf("delete pm_files: %w", err)
	}
	if r.devices, err = exec(`DELETE FROM devices WHERE serial_number LIKE $1`); err != nil {
		return r, fmt.Errorf("delete devices: %w", err)
	}
	return r, nil
}
