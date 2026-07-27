// Package tsdbsync 把主库（PgPool）的维度表周期性刷入时序库（TsPool）的影子维度表，
// 让 PM/告警等时序查询无需跨库 JOIN（KPI/时序库物理分离）。
//
// 设计要点：
//   - 全量快照、增量落盘：源端读取完整快照，目标端经 staging 做 UPSERT + anti-join DELETE，
//     既保证不漏删，又避免每分钟 TRUNCATE 和未变化行的 WAL 重写。
//   - 单表事务：每张影子表一个 dst 事务（staging + merge 同一 tx 原子提交）；
//     单表失败只 log 不中断其余表（健壮性 > 一次全成）。
//   - device_dim 保留软删行（含 deleted_at），让影子表 JOIN 行为与原 devices 一致。
//   - cell_band_dim 是派生表（主库无 cell_band），从 device_parameters 的
//     CellIdentity / FreqBandIndicator 按 (device_id, fap_instance) 配对派生。
package tsdbsync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// DefaultInterval 是影子维度表默认同步周期。
const DefaultInterval = 60 * time.Second

// SyncRunner 周期性把主库维度表全量刷入时序库影子表。
type SyncRunner struct {
	src      *pgxpool.Pool // 主库 PgPool（维度真值源）
	dst      *pgxpool.Pool // 时序库 TsPool（影子表）
	interval time.Duration
	logger   *zap.Logger
}

// NewSyncRunner 构造影子维度同步器。interval <= 0 时回退 DefaultInterval。
func NewSyncRunner(src, dst *pgxpool.Pool, interval time.Duration, logger *zap.Logger) *SyncRunner {
	if interval <= 0 {
		interval = DefaultInterval
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SyncRunner{
		src:      src,
		dst:      dst,
		interval: interval,
		logger:   logger.Named("tsdbsync"),
	}
}

// Run 启动同步循环：先同步一次，再按 interval tick 循环，ctx.Done 退出。
func (r *SyncRunner) Run(ctx context.Context) {
	r.logger.Info("tsdb shadow-dim sync runner started", zap.Duration("interval", r.interval))

	// 启动先同步一次（让影子表立即可用，不等第一个 tick）。
	if err := r.Sync(ctx); err != nil {
		// Sync 内部已对单表失败做了 log 隔离；这里返回的是 ctx 取消之类的整体性错误。
		r.logger.Warn("initial tsdb shadow-dim sync returned error", zap.Error(err))
	}

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			r.logger.Info("tsdb shadow-dim sync runner stopped")
			return
		case <-ticker.C:
			if err := r.Sync(ctx); err != nil {
				r.logger.Warn("tsdb shadow-dim sync returned error", zap.Error(err))
			}
		}
	}
}

// Sync 逐表同步影子维度表。单表失败只 log 不中断其余表；返回的 error 仅在 ctx 取消时非 nil。
func (r *SyncRunner) Sync(ctx context.Context) error {
	start := time.Now()

	// 1) device_dim ← devices（显式列子集，含软删行）。
	r.runTable(ctx, "device_dim", r.syncDeviceDim)

	// 2) device_group_member_dim ← device_group_members（全列镜像）。
	r.runTable(ctx, "device_group_member_dim", func(ctx context.Context) (int64, error) {
		return r.syncFullMirror(ctx, "device_group_members", "device_group_member_dim")
	})

	// 3) product_dim ← products（全列镜像）。
	r.runTable(ctx, "product_dim", func(ctx context.Context) (int64, error) {
		return r.syncFullMirror(ctx, "products", "product_dim")
	})

	// 4) device_group_dim ← device_groups（仅 id, name；源表列多，用显式列）。
	r.runTable(ctx, "device_group_dim", r.syncDeviceGroupDim)

	// 5) alarm_definition_dim ← alarm_definitions（全列镜像）。
	r.runTable(ctx, "alarm_definition_dim", func(ctx context.Context) (int64, error) {
		return r.syncFullMirror(ctx, "alarm_definitions", "alarm_definition_dim")
	})

	// 5b) mr_customize_task_dim ← mr_customize_task（列子集镜像：task_id, task_status,
	//     target_device_sns）。mr_files 迁时序库后，MR 文件设备聚合的「上报中」徽标判定需本库
	//     JOIN 任务订阅状态；mr_customize_task 配置仍留主库，这里只读镜像。
	r.runTable(ctx, "mr_customize_task_dim", func(ctx context.Context) (int64, error) {
		return r.syncFullMirror(ctx, "mr_customize_task", "mr_customize_task_dim")
	})

	// 6) cell_band_dim ← 从 device_parameters 派生（device_id, cell_id, band）。
	r.runTable(ctx, "cell_band_dim", r.syncCellBandDim)

	// 7) perf_indicators_{enb,gnb,gsm} ← 同名镜像（指标名解析表，时序库查询直读）。
	for _, t := range []string{"perf_indicators_enb", "perf_indicators_gnb", "perf_indicators_gsm"} {
		table := t
		r.runTable(ctx, table, func(ctx context.Context) (int64, error) {
			return r.syncFullMirror(ctx, table, table)
		})
	}
	r.runTable(ctx, "pm_metric_dictionary", r.syncMetricDictionary)

	// ctx 取消时统一返回 ctx.Err()，让 Run 走退出分支（单表失败已在 runTable 内隔离 log）。
	if err := ctx.Err(); err != nil {
		return err
	}
	r.logger.Debug("tsdb shadow-dim sync cycle finished", zap.Duration("took", time.Since(start)))
	return nil
}

const metricDictionarySyncSQL = `WITH source AS (
    SELECT DISTINCT ON (id) id,report_key,en_name,cn_name,unit_id,statis_type,is_counter
      FROM (
        SELECT id,report_key,en_name,cn_name,unit_id,statis_type,is_counter,updated_at FROM perf_indicators_enb
        UNION ALL
        SELECT id,report_key,en_name,cn_name,unit_id,statis_type,is_counter,updated_at FROM perf_indicators_gnb
        UNION ALL
        SELECT id,report_key,en_name,cn_name,unit_id,statis_type,is_counter,updated_at FROM perf_indicators_gsm
      ) all_indicators
     ORDER BY id, updated_at DESC NULLS LAST
),
registered AS (
    INSERT INTO pm_metric_dictionary
        (metric_path,report_key,metric_type,metric_name,unit,statis_type)
    SELECT id,COALESCE(NULLIF(report_key,''),id),
           CASE WHEN is_counter='1' THEN 'counter' ELSE 'kpi' END,
           COALESCE(NULLIF(cn_name,''),en_name),unit_id,statis_type
      FROM source
    ON CONFLICT (metric_path) DO UPDATE SET
      report_key=COALESCE(NULLIF(EXCLUDED.report_key,''),pm_metric_dictionary.report_key),
      metric_type=EXCLUDED.metric_type,
      metric_name=COALESCE(NULLIF(EXCLUDED.metric_name,''),pm_metric_dictionary.metric_name),
      unit=COALESCE(NULLIF(EXCLUDED.unit,''),pm_metric_dictionary.unit),
      statis_type=COALESCE(NULLIF(EXCLUDED.statis_type,''),pm_metric_dictionary.statis_type),
      updated_at=now()
    WHERE (pm_metric_dictionary.report_key,pm_metric_dictionary.metric_type,
           pm_metric_dictionary.metric_name,pm_metric_dictionary.unit,
           pm_metric_dictionary.statis_type)
      IS DISTINCT FROM
          (COALESCE(NULLIF(EXCLUDED.report_key,''),pm_metric_dictionary.report_key),
           EXCLUDED.metric_type,
           COALESCE(NULLIF(EXCLUDED.metric_name,''),pm_metric_dictionary.metric_name),
           COALESCE(NULLIF(EXCLUDED.unit,''),pm_metric_dictionary.unit),
           COALESCE(NULLIF(EXCLUDED.statis_type,''),pm_metric_dictionary.statis_type))
    RETURNING metric_id
),
enriched_unknowns AS (
    UPDATE pm_metric_dictionary d
       SET report_key=s.report_key,
           metric_type='counter',
           metric_name=COALESCE(NULLIF(s.cn_name,''),NULLIF(s.en_name,''),d.metric_name),
           unit=COALESCE(NULLIF(s.unit_id,''),d.unit),
           statis_type=COALESCE(NULLIF(s.statis_type,''),d.statis_type),
           updated_at=now()
      FROM source s
     WHERE NULLIF(s.report_key,'')=d.metric_path AND d.metric_path<>s.id
       AND s.is_counter='1'
       AND (d.report_key,d.metric_type,d.metric_name,d.unit,d.statis_type)
         IS DISTINCT FROM
             (s.report_key,'counter',
              COALESCE(NULLIF(s.cn_name,''),NULLIF(s.en_name,''),d.metric_name),
              COALESCE(NULLIF(s.unit_id,''),d.unit),
              COALESCE(NULLIF(s.statis_type,''),d.statis_type))
    RETURNING d.metric_id
)
SELECT (SELECT count(*) FROM registered)+(SELECT count(*) FROM enriched_unknowns)`

func (r *SyncRunner) syncMetricDictionary(ctx context.Context) (int64, error) {
	var count int64
	err := r.dst.QueryRow(ctx, metricDictionarySyncSQL).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("sync PM metric dictionary: %w", err)
	}
	return count, nil
}

// runTable 跑单张表的同步，把成功/失败统一 log；单表失败不向上冒泡（隔离）。
func (r *SyncRunner) runTable(ctx context.Context, dimTable string, fn func(context.Context) (int64, error)) {
	if ctx.Err() != nil {
		return
	}
	n, err := fn(ctx)
	if err != nil {
		r.logger.Warn("shadow-dim table sync failed (other tables continue)",
			zap.String("table", dimTable), zap.Error(err))
		return
	}
	r.logger.Debug("shadow-dim table synced", zap.String("table", dimTable), zap.Int64("rows", n))
}

// syncFullMirror 镜像同步：以【dst 影子表的列】为准，只从源表 SELECT dst 拥有的列，
// 在 dst 单事务内 staging + 增量 merge。
//
// 为何按【src ∩ dst 列交集】而非 SELECT *：影子表 DDL 取自某次基线，与源表当前 schema 会双向漂移：
//   - 源表新增列（增量迁移）：如 perf_indicators.report_key / products.is_builtin / alarm_definitions.description
//     —— 这些不在影子表也不被时序库查询用到；
//   - 源表删除列：如 000025 删了 alarm_definitions 的 cn_suggestion/en_suggestion，但影子表 DDL 仍留着。
//
// 任一方向的不一致都会让"按单边列 SELECT+COPY"整表失败。取交集 → 只同步两边都有的列，对 schema
// 漂移完全免疫；时序库查询所需列（id/cn_name/en_name/identifier 等）始终在交集内。
// 适用：device_group_member_dim / product_dim / alarm_definition_dim / perf_indicators_*。
func (r *SyncRunner) syncFullMirror(ctx context.Context, srcTable, dstTable string) (int64, error) {
	dstCols, err := tableColumns(ctx, r.dst, dstTable)
	if err != nil {
		return 0, fmt.Errorf("introspect dst %s: %w", dstTable, err)
	}
	srcCols, err := tableColumns(ctx, r.src, srcTable)
	if err != nil {
		return 0, fmt.Errorf("introspect src %s: %w", srcTable, err)
	}
	cols := intersectCols(dstCols, srcCols) // 保持 dst ordinal 顺序
	if len(cols) == 0 {
		return 0, fmt.Errorf("no common columns between src %s and dst %s", srcTable, dstTable)
	}
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = pgx.Identifier{c}.Sanitize()
	}
	q := fmt.Sprintf("SELECT %s FROM %s", strings.Join(quoted, ", "), srcTable)
	data, err := r.collectRows(ctx, q, len(cols))
	if err != nil {
		return 0, fmt.Errorf("mirror %s→%s: %w", srcTable, dstTable, err)
	}
	return r.truncateAndCopy(ctx, dstTable, cols, data)
}

// tableColumns 返回某库某表的列名（public schema，按 ordinal 顺序）。
func tableColumns(ctx context.Context, pool *pgxpool.Pool, table string) ([]string, error) {
	rows, err := pool.Query(ctx,
		`SELECT column_name FROM information_schema.columns
		 WHERE table_schema = 'public' AND table_name = $1
		 ORDER BY ordinal_position`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

// intersectCols 返回 a、b 的列名交集，保持 a（dst）的顺序。
func intersectCols(a, b []string) []string {
	set := make(map[string]struct{}, len(b))
	for _, c := range b {
		set[c] = struct{}{}
	}
	out := make([]string, 0, len(a))
	for _, c := range a {
		if _, ok := set[c]; ok {
			out = append(out, c)
		}
	}
	return out
}

// syncDeviceDim 同步 device_dim（显式列子集，含软删行让 JOIN 行为与 devices 一致）。
func (r *SyncRunner) syncDeviceDim(ctx context.Context) (int64, error) {
	cols := []string{"id", "oui", "serial_number", "product_id", "technology", "carrier", "site_name", "product_class", "deleted_at"}
	const q = `SELECT id, oui, serial_number, product_id, technology, carrier, site_name, product_class, deleted_at FROM devices`
	data, err := r.collectRows(ctx, q, len(cols))
	if err != nil {
		return 0, err
	}
	return r.truncateAndCopy(ctx, "device_dim", cols, data)
}

// syncDeviceGroupDim 同步 device_group_dim（源 device_groups 列多，仅取 id, name）。
func (r *SyncRunner) syncDeviceGroupDim(ctx context.Context) (int64, error) {
	cols := []string{"id", "name"}
	const q = `SELECT id, name FROM device_groups`
	data, err := r.collectRows(ctx, q, len(cols))
	if err != nil {
		return 0, err
	}
	return r.truncateAndCopy(ctx, "device_group_dim", cols, data)
}

// syncCellBandDim 派生 cell_band_dim（device_id, cell_id, band）。
//
// 主库无 cell_band 表：从 device_parameters 按参数路径后缀提取，
// 每个小区实例 (device_id, fap_instance) 取两行配对：
//   - cell_id = CellIdentity(LTE) / NrcellIdentity(NR) / IpaUnitId(GSM)
//   - band    = FreqBandIndicator(LTE) / FreqBandIndicatorNR(NR) / DeviceGSM.Bts.%.Band(GSM)
//
// 语义参考 internal/pm/aggregator/query.go 的 bandPathSuffixes / cellIDPathSuffixes。
// 两参数同属一个 (device_id, fap_instance) → INNER JOIN 配成一行；band 缺失的小区不产出
// （与原 band 维度 INNER JOIN 跳过未命中小区的行为一致）。CellIdentity 为空串跳过。
//
// cell_band_dim 主键 (device_id, cell_id)：同一设备同一小区号若有多 fap_instance 取任一
// （DISTINCT ON 去重，正常一个小区号对一个 band）。
func (r *SyncRunner) syncCellBandDim(ctx context.Context) (int64, error) {
	// LIKE 后缀匹配（{i} 实例号在路径中已实例化为具体数字，故用 '%suffix'）。
	const q = `
WITH cell_id AS (
    -- LTE
    SELECT device_id, fap_instance, parameter_value AS cell_id
    FROM device_parameters
    WHERE parameter_path LIKE '%.CellConfig.LTE.RAN.Common.CellIdentity'
      AND parameter_value IS NOT NULL
      AND parameter_value <> ''
    UNION ALL
    -- NR
    SELECT device_id, fap_instance, parameter_value AS cell_id
    FROM device_parameters
    WHERE parameter_path LIKE '%.NrcellIdentity'
      AND parameter_value IS NOT NULL
      AND parameter_value <> ''
    UNION ALL
    -- GSM (PM Uid= corresponds to IpaUnitId)
    SELECT device_id, fap_instance, parameter_value AS cell_id
    FROM device_parameters
    WHERE parameter_path LIKE '%IpaUnitId'
      AND parameter_value IS NOT NULL
      AND parameter_value <> ''
),
band AS (
    SELECT device_id, fap_instance, parameter_value AS band
    FROM device_parameters
    WHERE (parameter_path LIKE '%.CellConfig.LTE.RAN.RF.FreqBandIndicator'
           OR parameter_path LIKE '%.FreqBandIndicatorNR'
           OR parameter_path LIKE 'DeviceGSM.Bts.%.Band')
      AND parameter_value IS NOT NULL
      AND parameter_value <> ''
)
SELECT DISTINCT ON (c.device_id, c.cell_id)
       c.device_id, c.cell_id, b.band
FROM cell_id c
JOIN band b
  ON b.device_id = c.device_id AND b.fap_instance = c.fap_instance
ORDER BY c.device_id, c.cell_id, c.fap_instance`

	cols := []string{"device_id", "cell_id", "band"}
	data, err := r.collectRows(ctx, q, len(cols))
	if err != nil {
		return 0, err
	}
	return r.truncateAndCopy(ctx, "cell_band_dim", cols, data)
}

// collectRows 跑一条 src 查询，把每行的值收集成 [][]any（供 CopyFrom）。
func (r *SyncRunner) collectRows(ctx context.Context, query string, colCount int) ([][]any, error) {
	rows, err := r.src.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query src: %w", err)
	}
	defer rows.Close()

	data := make([][]any, 0, 256)
	for rows.Next() {
		vals, verr := rows.Values()
		if verr != nil {
			return nil, fmt.Errorf("read row: %w", verr)
		}
		if len(vals) != colCount {
			return nil, fmt.Errorf("unexpected column count: got %d want %d", len(vals), colCount)
		}
		data = append(data, vals)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iter src: %w", err)
	}
	return data, nil
}

// truncateAndCopy 保留历史函数名，但实现改为 staging + 增量 merge，避免每分钟
// TRUNCATE 触发 DataFileImmediateSync 并重写整张维表。无主键的旧影子表才安全回退
// 到原 TRUNCATE + COPY 语义。
func (r *SyncRunner) truncateAndCopy(ctx context.Context, dstTable string, cols []string, data [][]any) (int64, error) {
	tx, err := r.dst.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin dst tx for %s: %w", dstTable, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit 成功后 rollback 是 no-op；失败路径用它回滚

	keyCols, err := primaryKeyColumns(ctx, tx, dstTable)
	if err != nil {
		return 0, fmt.Errorf("load primary key for %s: %w", dstTable, err)
	}
	if len(keyCols) == 0 || !containsAllColumns(cols, keyCols) {
		r.logger.Warn("shadow table has no usable primary key; falling back to truncate copy",
			zap.String("table", dstTable))
		if _, err := tx.Exec(ctx, fmt.Sprintf("TRUNCATE %s", pgx.Identifier{dstTable}.Sanitize())); err != nil {
			return 0, fmt.Errorf("truncate %s: %w", dstTable, err)
		}
		if len(data) > 0 {
			if _, err := tx.CopyFrom(ctx, pgx.Identifier{dstTable}, cols, pgx.CopyFromRows(data)); err != nil {
				return 0, fmt.Errorf("copy into %s: %w", dstTable, err)
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit fallback %s: %w", dstTable, err)
		}
		return int64(len(data)), nil
	}

	stageTable := "sync_stage_" + dstTable
	quotedCols := quoteColumns(cols)
	createStage := fmt.Sprintf(
		"CREATE TEMP TABLE %s ON COMMIT DROP AS SELECT %s FROM %s WITH NO DATA",
		pgx.Identifier{stageTable}.Sanitize(),
		strings.Join(quotedCols, ", "),
		pgx.Identifier{dstTable}.Sanitize(),
	)
	if _, err := tx.Exec(ctx, createStage); err != nil {
		return 0, fmt.Errorf("create staging table for %s: %w", dstTable, err)
	}
	if len(data) > 0 {
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{stageTable}, cols, pgx.CopyFromRows(data)); err != nil {
			return 0, fmt.Errorf("copy staging for %s: %w", dstTable, err)
		}
	}
	upsertSQL, pruneSQL, err := buildMirrorMergeSQL(dstTable, stageTable, cols, keyCols)
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, upsertSQL); err != nil {
		return 0, fmt.Errorf("merge staging into %s: %w", dstTable, err)
	}
	if _, err := tx.Exec(ctx, pruneSQL); err != nil {
		return 0, fmt.Errorf("prune stale rows from %s: %w", dstTable, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit %s: %w", dstTable, err)
	}
	return int64(len(data)), nil
}

type queryRower interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func primaryKeyColumns(ctx context.Context, q queryRower, table string) ([]string, error) {
	rows, err := q.Query(ctx, `
		SELECT a.attname
		FROM pg_index i
		CROSS JOIN LATERAL unnest(i.indkey) WITH ORDINALITY AS k(attnum, ord)
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = k.attnum
		WHERE i.indrelid = $1::regclass AND i.indisprimary
		ORDER BY k.ord`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			return nil, err
		}
		cols = append(cols, col)
	}
	return cols, rows.Err()
}

func quoteColumns(cols []string) []string {
	out := make([]string, len(cols))
	for i, col := range cols {
		out[i] = pgx.Identifier{col}.Sanitize()
	}
	return out
}

func containsAllColumns(cols, required []string) bool {
	set := make(map[string]struct{}, len(cols))
	for _, col := range cols {
		set[col] = struct{}{}
	}
	for _, col := range required {
		if _, ok := set[col]; !ok {
			return false
		}
	}
	return true
}

func buildMirrorMergeSQL(dstTable, stageTable string, cols, keyCols []string) (string, string, error) {
	if len(cols) == 0 || len(keyCols) == 0 || !containsAllColumns(cols, keyCols) {
		return "", "", fmt.Errorf("mirror merge requires selected primary-key columns")
	}
	dst := pgx.Identifier{dstTable}.Sanitize()
	stage := pgx.Identifier{stageTable}.Sanitize()
	quotedCols := quoteColumns(cols)
	quotedKeys := quoteColumns(keyCols)
	keySet := make(map[string]struct{}, len(keyCols))
	for _, key := range keyCols {
		keySet[key] = struct{}{}
	}
	var assignments, changed []string
	for _, col := range cols {
		if _, key := keySet[col]; key {
			continue
		}
		qcol := pgx.Identifier{col}.Sanitize()
		assignments = append(assignments, fmt.Sprintf("%s = EXCLUDED.%s", qcol, qcol))
		changed = append(changed, fmt.Sprintf("target.%s IS DISTINCT FROM EXCLUDED.%s", qcol, qcol))
	}
	action := "DO NOTHING"
	if len(assignments) > 0 {
		action = fmt.Sprintf("DO UPDATE SET %s WHERE %s",
			strings.Join(assignments, ", "), strings.Join(changed, " OR "))
	}
	upsert := fmt.Sprintf(
		"INSERT INTO %s AS target (%s) SELECT %s FROM %s ON CONFLICT (%s) %s",
		dst, strings.Join(quotedCols, ", "), strings.Join(quotedCols, ", "), stage,
		strings.Join(quotedKeys, ", "), action,
	)
	matches := make([]string, 0, len(keyCols))
	for _, key := range keyCols {
		qkey := pgx.Identifier{key}.Sanitize()
		matches = append(matches, fmt.Sprintf("d.%s IS NOT DISTINCT FROM s.%s", qkey, qkey))
	}
	prune := fmt.Sprintf(
		"DELETE FROM %s AS d WHERE NOT EXISTS (SELECT 1 FROM %s AS s WHERE %s)",
		dst, stage, strings.Join(matches, " AND "),
	)
	return upsert, prune, nil
}
