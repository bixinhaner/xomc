package dictsource

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// SyncRowReader 是 SyncEngine 对底层数据库的窄依赖。
// pgxpool.Pool 直接实现这两个方法,测试可注入内存 mock。
type SyncRowReader interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// SyncWriter 写侧依赖(独立接口,便于事务/批量写法演进)。
// 当前由 SyncRunner 实现;未来如要切到 pgxpool.Begin → CopyFrom 也只动这一层。
type SyncWriter interface {
	// UpsertAuto 用 (sys_dictionary_id, label, value) 自然键 upsert origin='auto' 行。
	// status=true / sort=0 / parent_id=NULL / level=0(托管字典强制扁平,无层级)。
	// 返回 (inserted, updated)。
	UpsertAuto(ctx context.Context, dictID int64, rows []PairLV) (inserted, updated int, err error)
	// DeleteAutoNotIn 删除 (sys_dictionary_id, origin='auto') 中 value 不在 keepValues 的行。
	DeleteAutoNotIn(ctx context.Context, dictID int64, keepValues []string) (deleted int, err error)
	// CountAuto 同步完成后回读当前 auto 行数,写到 sys_dictionaries.last_refresh_count。
	CountAuto(ctx context.Context, dictID int64) (int, error)
	// UpdateMetadata 写 last_refresh_* 元数据(失败也得能写)。
	UpdateMetadata(ctx context.Context, dictID int64, status, errMsg string, count int) error
}

// PairLV 是 SQL 抽出的一对 (label, value)。
type PairLV struct {
	Label string
	Value string
}

// SyncResult 是 SyncOne 返给调用方的执行明细。
type SyncResult struct {
	Inserted  int
	Updated   int
	Deleted   int
	TotalAuto int
	DurationMS int64
}

// SyncDict 是 SyncEngine 操作字典所需的最小信息。
// 调用方(DictionaryService)从 sys_dictionaries 行映射过来。
type SyncDict struct {
	ID          int64
	Name        string
	SourceTable string
	LabelField  string
	ValueField  string
}

// Errors —— 全部归 service 层翻译为 commonerrors.BusinessError 暴露到 HTTP。
var (
	ErrInvalidSourceTable = errors.New("source_table not in whitelist")
	ErrInvalidLabelField  = errors.New("source_label_field not in whitelist")
	ErrInvalidValueField  = errors.New("source_value_field not in whitelist")
	ErrLimitExceeded      = errors.New("source row count exceeds hard limit")
)

// MaxSourceRows 单次同步从源表读取的硬上限。
// PRD §3.4 容量保护:超过即截断 + 日志告警。10 万设备规模下绝大多数业务字典 << 5000。
const MaxSourceRows = 5000

// SingleDictTimeout 单字典同步硬超时。
// PRD §3.4:5 秒内必须完成,否则记 failed/timeout 跳过。
const SingleDictTimeout = 5 * time.Second

// SyncEngine 是手动 + 定时刷新共用的同步引擎。
// 调用约定:
//   - SyncOne 用于"用户手动点刷新"路径,调用方传单字典模型。
//   - SyncAll 用于 worker daily cron,内部遍历调 SyncOne(本包不直接查 sys_dictionaries
//     列表 —— 那是 service/repo 层职责;SyncAll 接受 lister 函数注入)。
type SyncEngine struct {
	reader   SyncRowReader
	writer   SyncWriter
	registry *Registry
	log      *zap.Logger
	metrics  *Metrics // 可为 nil(测试 / 未注入)
}

// NewSyncEngine 构造同步引擎。registry 必须不为 nil(否则 SyncOne nil-deref)。
// metrics 可为 nil(测试不需要 Prometheus);生产由 provider/worker 用 NewMetrics(reg) 注入。
func NewSyncEngine(reader SyncRowReader, writer SyncWriter, registry *Registry, log *zap.Logger) *SyncEngine {
	if registry == nil {
		// Fail-fast:让启动期立刻发现 wiring bug,而不是 runtime nil-deref。
		// S5 review MEDIUM-3 — defensive guard,与 provider 的 silent fallback 互补:
		// provider 在 LoadDefault 失败时根本不调本构造函数,所以正常路径不会触发 panic。
		panic("dictsource.NewSyncEngine: registry must not be nil (use LoadDefault or LoadFromBytes)")
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &SyncEngine{reader: reader, writer: writer, registry: registry, log: log}
}

// SetMetrics 注入 Prometheus 指标(provider 启动期调用一次)。
// 留作 setter 而非构造参数,避免 NewSyncEngine 签名变动影响现有测试。
func (e *SyncEngine) SetMetrics(m *Metrics) {
	e.metrics = m
}

// SyncOne 同步单张托管字典。
// 失败时 SyncResult 仍返回(零值字段),err 携带原因;调用方负责 UpdateMetadata。
// 已包 5s 超时;调用方传入的 ctx 是 daily cron 的整体 ctx,本方法 derive。
//
// trigger 默认 "manual";调用方(daily cron/initial/switch)传特定标签让 Prometheus
// 维度更精确(SyncAll 内部传 "daily",service Create/Update 路径用 SyncOneWithTrigger)。
func (e *SyncEngine) SyncOne(parentCtx context.Context, d SyncDict) (SyncResult, error) {
	return e.SyncOneWithTrigger(parentCtx, d, TriggerManual)
}

// SyncOneWithTrigger 与 SyncOne 等价,但允许调用方显式声明 trigger label。
// 留作扩展点,默认 SyncOne 把 trigger 钉在 manual(用户手动刷新场景)。
func (e *SyncEngine) SyncOneWithTrigger(parentCtx context.Context, d SyncDict, trigger string) (SyncResult, error) {
	ctx, cancel := context.WithTimeout(parentCtx, SingleDictTimeout)
	defer cancel()

	start := time.Now()
	res := SyncResult{}

	// 包装,任何 path 退出前都打 metric。
	observeFn := func(result string) {
		e.metrics.Observe(result, trigger)
	}

	// 1. 白名单校验 —— 必须三个字段都过,任意失败立即拒绝。
	if d.SourceTable == "" || d.LabelField == "" || d.ValueField == "" {
		observeFn(ResultFailed)
		return res, fmt.Errorf("sync dict %d: missing source binding", d.ID)
	}
	if _, ok := e.registry.Resolve(d.SourceTable); !ok {
		observeFn(ResultFailed)
		return res, fmt.Errorf("sync dict %d: %w (%q)", d.ID, ErrInvalidSourceTable, d.SourceTable)
	}
	if _, ok := e.registry.ResolveField(d.SourceTable, d.LabelField); !ok {
		observeFn(ResultFailed)
		return res, fmt.Errorf("sync dict %d: %w (%q)", d.ID, ErrInvalidLabelField, d.LabelField)
	}
	if _, ok := e.registry.ResolveField(d.SourceTable, d.ValueField); !ok {
		observeFn(ResultFailed)
		return res, fmt.Errorf("sync dict %d: %w (%q)", d.ID, ErrInvalidValueField, d.ValueField)
	}

	e.log.Info("dict_source_sync_started",
		zap.Int64("dict_id", d.ID), zap.String("table", d.SourceTable),
		zap.String("label", d.LabelField), zap.String("value", d.ValueField),
	)

	// 2. 拉源数据。
	//    - 白名单已校验 → 标识符直接插入 SQL 安全(pgx 不接受表名占位符);
	//    - WHERE label/value 非 NULL,避免空 label 灌进字典;
	//    - DISTINCT 去重(同 product_class 出现多次只入字典一条);
	//    - LIMIT 5000 硬上限 —— 超出即返 ErrLimitExceeded(调用方记 failed)。
	pairs, err := e.fetchSource(ctx, d)
	if err != nil {
		// 区分 ctx 超时 (timeout) vs 真错误 (failed)
		if errors.Is(err, context.DeadlineExceeded) {
			observeFn(ResultTimeout)
		} else {
			observeFn(ResultFailed)
		}
		return res, err
	}
	if len(pairs) >= MaxSourceRows {
		observeFn(ResultFailed)
		return res, fmt.Errorf("sync dict %d: %w (got %d rows)", d.ID, ErrLimitExceeded, len(pairs))
	}

	// S5 review MEDIUM-2:防御源表"瞬时为空"导致全量清空 auto 项。
	// 触发场景:长事务/FK cascade in progress/replication lag。让管理员看到
	// last_refresh_error 而不是默默丢数据,等下次 cron 再尝试。
	// 排除合法场景:首次同步(currentCount==0)或源表本来就空,不视为瞬时。
	if len(pairs) == 0 {
		currentCount, cntErr := e.writer.CountAuto(ctx, d.ID)
		if cntErr == nil && currentCount > 0 {
			observeFn(ResultFailed)
			return res, fmt.Errorf("sync dict %d: source returned 0 rows but existing auto count is %d (transient empty? aborting to avoid mass-delete)", d.ID, currentCount)
		}
	}

	// 3. 写库:upsert 入库 + 删孤儿 + 回读总数。
	inserted, updated, err := e.writer.UpsertAuto(ctx, d.ID, pairs)
	if err != nil {
		observeFn(ResultFailed)
		return res, fmt.Errorf("upsert auto details: %w", err)
	}
	res.Inserted = inserted
	res.Updated = updated

	keepValues := make([]string, len(pairs))
	for i, p := range pairs {
		keepValues[i] = p.Value
	}
	deleted, err := e.writer.DeleteAutoNotIn(ctx, d.ID, keepValues)
	if err != nil {
		observeFn(ResultFailed)
		return res, fmt.Errorf("delete orphan auto details: %w", err)
	}
	res.Deleted = deleted

	total, err := e.writer.CountAuto(ctx, d.ID)
	if err != nil {
		observeFn(ResultFailed)
		return res, fmt.Errorf("count auto details: %w", err)
	}
	res.TotalAuto = total
	res.DurationMS = time.Since(start).Milliseconds()
	observeFn(ResultOK)

	e.log.Info("dict_source_sync_completed",
		zap.Int64("dict_id", d.ID),
		zap.Int("inserted", res.Inserted), zap.Int("updated", res.Updated),
		zap.Int("deleted", res.Deleted), zap.Int("total", res.TotalAuto),
		zap.Int64("duration_ms", res.DurationMS),
	)
	return res, nil
}

// fetchSource 从源表抽 (label, value) distinct 行。
// 该方法是字典数据进库前的"过滤层":
//   - 白名单已校验,字段名直接拼 SQL 安全
//   - 同时过滤 NULL value(无值的字典项无意义)
//   - LIMIT N+1 让调用方判 N+1 == MaxSourceRows 时算超限
func (e *SyncEngine) fetchSource(ctx context.Context, d SyncDict) ([]PairLV, error) {
	sql := fmt.Sprintf(
		`SELECT DISTINCT %s::TEXT AS label, %s::TEXT AS value
		 FROM %s
		 WHERE %s IS NOT NULL AND %s::TEXT <> ''
		 LIMIT %d`,
		d.LabelField, d.ValueField, d.SourceTable,
		d.ValueField, d.ValueField,
		MaxSourceRows,
	)
	rows, err := e.reader.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("query source %s: %w", d.SourceTable, err)
	}
	defer rows.Close()

	out := make([]PairLV, 0, 32)
	for rows.Next() {
		var label, value string
		if err := rows.Scan(&label, &value); err != nil {
			return nil, fmt.Errorf("scan source row: %w", err)
		}
		out = append(out, PairLV{Label: label, Value: value})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iter source rows: %w", err)
	}
	return out, nil
}

// Preview 用于「测试 → 预览前 10 条」按钮,不写库,只读源表。
// PRD §3.2.3:用户绑数据源前可 dry-run 看真实数据形态。
// 与 fetchSource 复用 SQL 形状,但 LIMIT 由调用方传(常用 10),并额外返回总行数估算。
//
// 注意:总行数 SELECT COUNT(*) 在大表上有成本;v1 接受 ~50ms 的代价(绑定字典属于
// 低频操作,管理员一次操作),后续可改为 EXPLAIN 或者 pg_class.reltuples 估算。
func (e *SyncEngine) Preview(ctx context.Context, table, labelField, valueField string, limit int) ([]PairLV, int, error) {
	if _, ok := e.registry.Resolve(table); !ok {
		return nil, 0, ErrInvalidSourceTable
	}
	if _, ok := e.registry.ResolveField(table, labelField); !ok {
		return nil, 0, ErrInvalidLabelField
	}
	if _, ok := e.registry.ResolveField(table, valueField); !ok {
		return nil, 0, ErrInvalidValueField
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	pairs, err := e.fetchSource(ctx, SyncDict{
		SourceTable: table, LabelField: labelField, ValueField: valueField,
	})
	if err != nil {
		return nil, 0, err
	}
	if len(pairs) > limit {
		pairs = pairs[:limit]
	}

	// 估算总数:DISTINCT 去重后的总行数(可能小于物理 row count)。
	// 慢路径走 COUNT(DISTINCT),没接受 reltuples 估算逻辑因为白名单表都是小表。
	totalSQL := fmt.Sprintf(
		`SELECT COUNT(DISTINCT (%s::TEXT, %s::TEXT))
		 FROM %s
		 WHERE %s IS NOT NULL AND %s::TEXT <> ''`,
		labelField, valueField, table, valueField, valueField,
	)
	var total int
	if err := e.reader.QueryRow(ctx, totalSQL).Scan(&total); err != nil {
		// COUNT 失败不阻断 preview;让用户看到样本即可。
		e.log.Warn("dict_source_preview_count_failed",
			zap.String("table", table), zap.Error(err),
		)
		total = len(pairs)
	}
	return pairs, total, nil
}

// SyncAll 是 worker daily cron 的入口。
// lister 注入 "从 sys_dictionaries 取所有 source_table IS NOT NULL AND status=true 行"
// 的方法;避免本包反向依赖 admin 包的 repository。
// 返回 (ok 数, failed 数);全程不 panic,单字典失败不中断后续。
func (e *SyncEngine) SyncAll(parentCtx context.Context, lister func(context.Context) ([]SyncDict, error)) (ok, failed int) {
	start := time.Now()
	dicts, err := lister(parentCtx)
	if err != nil {
		e.log.Error("dict_source_daily_list_failed", zap.Error(err))
		return 0, 0
	}
	for _, d := range dicts {
		// S5 review MEDIUM-1:per-iteration panic recover。SyncOne 在 nil writer /
		// 上下游接口签名漂移 / pgx 反序列化异常等小概率路径可能 panic;不包裹
		// 会让整个 daily 同步在某张失败字典处中断,后续字典全部漏跑。
		// 包裹后单字典 panic 视同失败处理,继续下一条。
		func() {
			defer func() {
				if r := recover(); r != nil {
					failed++
					e.log.Error("dict_source_sync_panic",
						zap.Int64("dict_id", d.ID),
						zap.String("name", d.Name),
						zap.Any("panic", r),
					)
					_ = e.writer.UpdateMetadata(parentCtx, d.ID, "failed", truncateErr(fmt.Sprintf("panic: %v", r)), 0)
				}
			}()
			res, err := e.SyncOneWithTrigger(parentCtx, d, TriggerDaily)
			if err != nil {
				failed++
				e.log.Warn("dict_source_sync_failed",
					zap.Int64("dict_id", d.ID), zap.String("name", d.Name), zap.Error(err),
				)
				_ = e.writer.UpdateMetadata(parentCtx, d.ID, "failed", truncateErr(err.Error()), 0)
				return
			}
			ok++
			_ = e.writer.UpdateMetadata(parentCtx, d.ID, "ok", "", res.TotalAuto)
		}()
	}
	e.log.Info("dict_source_daily_summary",
		zap.Int("total", len(dicts)), zap.Int("ok", ok), zap.Int("failed", failed),
		zap.Int64("duration_ms", time.Since(start).Milliseconds()),
	)
	return ok, failed
}

// truncateErr 把错误摘要截到 500 字符避免 last_refresh_error 列爆掉。
func truncateErr(s string) string {
	const maxLen = 500
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
