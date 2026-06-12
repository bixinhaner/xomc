package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
)

// deviceSN 把 (前缀, 制式, 序号) 映射成 SN，例如 ("KPILT","lte",1) → "KPILT-LTE-0000001"。
// 制式段便于人工辨识；清理仍按前缀 "KPILT-%" 一把命中三制式。
func deviceSN(prefix, rat string, idx int) string {
	return fmt.Sprintf("%s-%s-%07d", prefix, strings.ToUpper(rat), idx)
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

// seedDevices 为某制式注入 n 个测试设备（SN = prefix-RAT-000000X），带 productClass +
// technology。**必须在上传任何文件前注入并带上 productClass**：worker 的 KPIRouter 会按
// device_sn 缓存路由结果（含“未匹配”的负缓存），若设备首次被看到时 productClass 为空，
// 之后再补也不会重算 → 既算不出 KPI，又因白名单为空放过重名 counter 撞自然键。
// 用 ON CONFLICT DO NOTHING 幂等。返回实际新插入行数。
func seedDevices(ctx context.Context, pool *pgxpool.Pool, prefix, rat string, n int, oui, carrier, tech, productClass string) (int64, error) {
	const chunk = 500
	const cols = 7
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
			args = append(args, deviceSN(prefix, rat, i), oui, carrier, tech, productClass, "KPILoadTest", "KPILoadTest-"+strings.ToUpper(rat))
		}
		sb.WriteString(" ON CONFLICT DO NOTHING")

		cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		tag, err := pool.Exec(cctx, sb.String(), args...)
		cancel()
		if err != nil {
			return inserted, fmt.Errorf("seed %s devices [%d,%d): %w", rat, start, end, err)
		}
		inserted += tag.RowsAffected()
	}
	return inserted, nil
}

// dbCounts 统计带测试前缀的入库现状。
type dbCounts struct {
	metricsRows int64
	kpiRows     int64
	filesParsed int64
	filesTotal  int64
}

func queryDBCounts(ctx context.Context, pool *pgxpool.Pool, prefix string) (dbCounts, error) {
	var c dbCounts
	pat := snPattern(prefix)
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := pool.QueryRow(cctx,
		`SELECT count(*), count(*) FILTER (WHERE metric_type='kpi') FROM pm_metrics WHERE device_sn LIKE $1`, pat,
	).Scan(&c.metricsRows, &c.kpiRows); err != nil {
		return c, fmt.Errorf("count pm_metrics: %w", err)
	}
	if err := pool.QueryRow(cctx,
		`SELECT count(*) FILTER (WHERE parsed), count(*) FROM pm_files WHERE device_sn LIKE $1`, pat,
	).Scan(&c.filesParsed, &c.filesTotal); err != nil {
		return c, fmt.Errorf("count pm_files: %w", err)
	}
	return c, nil
}

// queryKPIByRat 返回每个 RAT 前缀下的 KPI 行数（device_sn LIKE 'prefix-RAT-%'）。
func queryKPIByRat(ctx context.Context, pool *pgxpool.Pool, prefix, rat string) (counter, kpi int64) {
	pat := fmt.Sprintf("%s-%s-%%", prefix, strings.ToUpper(rat))
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_ = pool.QueryRow(cctx,
		`SELECT count(*) FILTER (WHERE metric_type='counter'), count(*) FILTER (WHERE metric_type='kpi') FROM pm_metrics WHERE device_sn LIKE $1`, pat,
	).Scan(&counter, &kpi)
	return
}

type cleanupResult struct {
	metrics int64
	files   int64
	devices int64
	dlq     int64
}

// cleanupAll 一键清除测试数据：pm_metrics（含 KPI 行）、pm_files、devices、dead_letters
// （按 SN 前缀 / payload 命中）。pm_metrics 无外键，删除顺序无要求。
// KPI/时序库物理分离后 pm_metrics/pm_files 在时序库（tsPool），devices/dead_letters 在主库
// （mainPool）；未分离部署时两者指向同一池。
func cleanupAll(ctx context.Context, mainPool, tsPool *pgxpool.Pool, prefix string) (cleanupResult, error) {
	var r cleanupResult
	pat := snPattern(prefix)

	exec := func(pool *pgxpool.Pool, sql string, arg string) (int64, error) {
		cctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		tag, err := pool.Exec(cctx, sql, arg)
		if err != nil {
			return 0, err
		}
		return tag.RowsAffected(), nil
	}

	var err error
	if r.metrics, err = exec(tsPool, `DELETE FROM pm_metrics WHERE device_sn LIKE $1`, pat); err != nil {
		return r, fmt.Errorf("delete pm_metrics: %w", err)
	}
	if r.files, err = exec(tsPool, `DELETE FROM pm_files WHERE device_sn LIKE $1`, pat); err != nil {
		return r, fmt.Errorf("delete pm_files: %w", err)
	}
	if r.devices, err = exec(mainPool, `DELETE FROM devices WHERE serial_number LIKE $1`, pat); err != nil {
		return r, fmt.Errorf("delete devices: %w", err)
	}
	// dead_letters 里测试文件的失败记录（payload 含 device_sn）。best-effort：表/列不符时忽略。
	if n, derr := exec(mainPool, `DELETE FROM dead_letters WHERE payload::text LIKE $1`, "%"+prefix+"-%"); derr == nil {
		r.dlq = n
	}
	return r, nil
}

// purgePMStream 清空 NATS JetStream 的 PM 流（pm.file.received / pm.file.parsed）。
// 压测失败文件（未带 productClass 的旧批次）会在 PM 流里反复重投，拖垮 worker；
// 重测前 purge 一次给 worker 干净起点。
func purgePMStream(natsURL string) (int64, error) {
	nc, err := nats.Connect(natsURL, nats.Timeout(5*time.Second))
	if err != nil {
		return 0, fmt.Errorf("connect nats %s: %w", natsURL, err)
	}
	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		return 0, fmt.Errorf("jetstream ctx: %w", err)
	}
	before, _ := js.StreamInfo("PM")
	var n int64
	if before != nil {
		n = int64(before.State.Msgs)
	}
	if err := js.PurgeStream("PM"); err != nil {
		return 0, fmt.Errorf("purge PM stream: %w", err)
	}
	return n, nil
}
