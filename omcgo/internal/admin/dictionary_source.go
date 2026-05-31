package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin/dictsource"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// === T-0182 数据字典数据源 ===
// 该文件提供 service 层粘合:
//   ① syncWriterAdapter: 把 DictionaryDetailRepository + DictionaryRepository
//      适配为 dictsource.SyncWriter(写侧依赖)。
//   ② pgxpool.Pool 直接实现 dictsource.SyncRowReader,无需额外适配。
//   ③ DictionaryService 的 4 个新方法:
//      - ListDictionarySources    GET /admin/sysDictionary/sources
//      - PreviewDictionarySource  GET /admin/sysDictionary/sources/preview
//      - RefreshDictionarySource  POST /admin/sysDictionary/refreshSource
//      - SyncSourceBoundAll       daily cron 调用(本期已加,P2 接入 worker)
//   ④ validateSourceFields:校验 source_* 三字段同时空/同时填 + 白名单。

// syncWriterAdapter 桥接 DictionaryDetailRepository → dictsource.SyncWriter。
type syncWriterAdapter struct {
	detailRepo DictionaryDetailRepository
	dictRepo   DictionaryRepository
}

func newSyncWriterAdapter(detailRepo DictionaryDetailRepository, dictRepo DictionaryRepository) *syncWriterAdapter {
	return &syncWriterAdapter{detailRepo: detailRepo, dictRepo: dictRepo}
}

func (a *syncWriterAdapter) UpsertAuto(ctx context.Context, dictID int64, rows []dictsource.PairLV) (int, int, error) {
	autoRows := make([]AutoDetailRow, len(rows))
	for i, r := range rows {
		autoRows[i] = AutoDetailRow{Label: r.Label, Value: r.Value}
	}
	return a.detailRepo.UpsertAutoBatch(ctx, dictID, autoRows)
}

func (a *syncWriterAdapter) DeleteAutoNotIn(ctx context.Context, dictID int64, keepValues []string) (int, error) {
	return a.detailRepo.DeleteAutoNotIn(ctx, dictID, keepValues)
}

func (a *syncWriterAdapter) CountAuto(ctx context.Context, dictID int64) (int, error) {
	return a.detailRepo.CountAutoActive(ctx, dictID)
}

func (a *syncWriterAdapter) UpdateMetadata(ctx context.Context, dictID int64, status, errMsg string, count int) error {
	return a.dictRepo.UpdateRefreshMetadata(ctx, dictID, status, errMsg, count)
}

// === 错误码 ===
// 沿用现有 dictionary handler 的 7XXX 段:
//   7020 source 三字段部分填写(必须同时空或同时填)
//   7021 source_table 不在白名单
//   7022 source_label_field / source_value_field 不在白名单
//   7023 同步执行失败
const (
	errCodeSourcePartialFields = 7020
	errCodeSourceTableInvalid  = 7021
	errCodeSourceFieldInvalid  = 7022
	errCodeSourceSyncFailed    = 7023
)

// validateSourceFields 校验 (sourceTable, sourceLabelField, sourceValueField) 三字段:
//   - 三个都为空 → 返 ("", "", "", false, nil) 表示"未绑定数据源"
//   - 三个都非空 + Registry 命中 → 返 (table, label, value, true, nil)
//   - 部分填写 → 返 ErrPartialSourceFields(commonerrors)
//   - 表 / 字段不在白名单 → 返 ErrSourceTableInvalid / ErrSourceFieldInvalid
//
// allowEmpty=true 时显式 ""(三字段同时为空字符串)代表"解绑"语义,合法;
// 调用方(Update)用 allowEmpty=true,Create 用 allowEmpty=false。
func validateSourceFields(reg *dictsource.Registry, sourceTable, labelField, valueField *string, allowEmpty bool) (table, label, value string, bound bool, err error) {
	// 标准化 — nil 视为未填,非 nil 取 trim 后的值。
	hasT := sourceTable != nil
	hasL := labelField != nil
	hasV := valueField != nil

	// 三个都未提交(JSON 缺字段) → 未绑定数据源,通过。
	if !hasT && !hasL && !hasV {
		return "", "", "", false, nil
	}

	// 三个字段都提交 — 进一步判内容。
	if !(hasT && hasL && hasV) {
		return "", "", "", false, commonerrors.NewBusinessError(errCodeSourcePartialFields,
			"source_table / source_label_field / source_value_field must be set together", nil)
	}
	t := strings.TrimSpace(*sourceTable)
	l := strings.TrimSpace(*labelField)
	v := strings.TrimSpace(*valueField)

	// 三个都空字符串 — Update 视为"解绑";Create 视为不合法(没必要 create 一个无源字典)。
	if t == "" && l == "" && v == "" {
		if allowEmpty {
			return "", "", "", false, nil
		}
		return "", "", "", false, commonerrors.NewBusinessError(errCodeSourcePartialFields,
			"empty source binding is only valid on update (unbind)", nil)
	}

	// 任一为空字符串而其它非空 → 部分填写。
	if t == "" || l == "" || v == "" {
		return "", "", "", false, commonerrors.NewBusinessError(errCodeSourcePartialFields,
			"source_table / source_label_field / source_value_field must be all set or all empty", nil)
	}

	// 白名单校验。
	if _, ok := reg.Resolve(t); !ok {
		return "", "", "", false, commonerrors.NewBusinessError(errCodeSourceTableInvalid,
			fmt.Sprintf("source_table %q is not in whitelist", t), nil)
	}
	if _, ok := reg.ResolveField(t, l); !ok {
		return "", "", "", false, commonerrors.NewBusinessError(errCodeSourceFieldInvalid,
			fmt.Sprintf("source_label_field %q not in whitelist for table %s", l, t), nil)
	}
	if _, ok := reg.ResolveField(t, v); !ok {
		return "", "", "", false, commonerrors.NewBusinessError(errCodeSourceFieldInvalid,
			fmt.Sprintf("source_value_field %q not in whitelist for table %s", v, t), nil)
	}
	return t, l, v, true, nil
}

// translateEngineError 把 dictsource 包错误翻译为 commonerrors.BusinessError。
// HTTP handler 直接对外返,前端按 code 出 toast 文案。
func translateEngineError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, dictsource.ErrInvalidSourceTable):
		return commonerrors.NewBusinessError(errCodeSourceTableInvalid, err.Error(), err)
	case errors.Is(err, dictsource.ErrInvalidLabelField), errors.Is(err, dictsource.ErrInvalidValueField):
		return commonerrors.NewBusinessError(errCodeSourceFieldInvalid, err.Error(), err)
	default:
		return commonerrors.NewBusinessError(errCodeSourceSyncFailed, err.Error(), err)
	}
}

// DefaultDictSourceDailyCron 是 worker 每日同步字典数据源的默认 cron 表达式。
// PRD §3.4:每日 02:00 BJT,业务低峰;robfig/cron 6 段语法(秒/分/时/日/月/周)。
const DefaultDictSourceDailyCron = "0 0 2 * * *"

// LoadDictSourceRegistry 加载内置白名单(embed sources.yaml)。
// 是 internal/admin 包向 cmd/app/provider 暴露的最小入口 — provider 不直接 import dictsource 包。
func LoadDictSourceRegistry() (*dictsource.Registry, error) {
	return dictsource.LoadDefault()
}

// NewDictSyncEngine 构造同步引擎,把 detail/dict repository 适配为 SyncWriter,
// pgxpool.Pool 自身实现 SyncRowReader(Query / QueryRow 方法签名匹配)。
func NewDictSyncEngine(
	reg *dictsource.Registry,
	pool *pgxpool.Pool,
	dictRepo DictionaryRepository,
	detailRepo DictionaryDetailRepository,
	log *zap.Logger,
) *dictsource.SyncEngine {
	writer := newSyncWriterAdapter(detailRepo, dictRepo)
	return dictsource.NewSyncEngine(pool, writer, reg, log)
}

// NewDictSourceMetrics 注册 dictionary_source_sync_total 指标。
// nil registerer 走匿名 Registry(测试)。app + worker 各自注册一份对应自己进程的
// /metrics 端点(否则 worker 失败永远不会暴露到 app 的 Prometheus)。
func NewDictSourceMetrics(reg prometheus.Registerer) *dictsource.Metrics {
	return dictsource.NewMetrics(reg)
}
