// Package dlq 提供 worker 进程级的死信队列（dead-letter queue）抽象。
//
// 与 internal/alarm/dead_letter.go（webhook 派发死信）属于不同语义层：
//
//   - alarm.DeadLetterRecord — 告警北向 webhook 推送耗尽重试后的 outbound DLQ
//   - dlq.DeadLetter        — worker 订阅 EventBus 处理失败耗尽重试后的 inbound DLQ
//
// 两者并存、不合并。本包仅提供 inbound 事件 DLQ 的类型定义、过滤器和 Repository
// 接口，具体 PostgreSQL 实现见 pg_repository.go。Runner 在 retry 耗尽时调用
// Repository.Insert 落库；admin handler 调用 List/Get/Delete 提供运维能力。
package dlq

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/model"
)

// DeadLetter 表示一次 worker 处理失败耗尽重试后的死信记录。
//
// 字段语义：
//   - SourceModule  — 业务模块名（"pm" / "mr" / "alarm" 等），用于分组与过滤
//   - SourceSubject — EventBus 主题（"pm.file.received" 等），replay 时复用此 subject
//   - Payload       — 原始事件载荷（json.RawMessage），replay 时直接 publish 出去
//   - Error         — 最后一次失败的错误信息（已 TRUNCATE 到 4096 字符）
//   - RetryCount    — 累计重试次数（含最后一次失败），通常等于 RetryConfig.MaxAttempts
//   - CreatedAt     — 入库时间
//   - LastAttemptAt — 最后一次重试时间（入库 = 最后一次失败的时间）
type DeadLetter struct {
	ID            uuid.UUID `db:"id"              json:"id"`
	SourceModule  string    `db:"source_module"   json:"source_module"`
	SourceSubject string    `db:"source_subject"  json:"source_subject"`
	Payload       []byte    `db:"payload"         json:"payload"`
	Error         string    `db:"error"           json:"error"`
	RetryCount    int       `db:"retry_count"     json:"retry_count"`
	CreatedAt     time.Time `db:"created_at"      json:"created_at"`
	LastAttemptAt time.Time `db:"last_attempt_at" json:"last_attempt_at"`
}

// MaxErrorLength 为 error 字段在入库前 truncate 的最大字节数。
// 防止 stack trace 撑爆 PG 行（PRD §8.6 安全考虑）。
const MaxErrorLength = 4096

// TruncateError 截断错误信息到 MaxErrorLength 字节。
// 超出长度时追加 "...[truncated]" 提示。
func TruncateError(s string) string {
	if len(s) <= MaxErrorLength {
		return s
	}
	const suffix = "...[truncated]"
	cut := MaxErrorLength - len(suffix)
	if cut < 0 {
		cut = 0
	}
	return s[:cut] + suffix
}

// Filter 是 List 查询的过滤器，所有指针字段为 nil 时不参与 WHERE 过滤。
// 嵌入 model.ListRequest 复用 Page/PageSize/SortBy/SortDir 标准分页字段。
type Filter struct {
	Module  *string
	Subject *string
	model.ListRequest
}

// Repository 是死信记录的持久化抽象，由 admin handler / runner 共同消费。
//
// 实现保证：
//   - Insert 是幂等的（PRIMARY KEY 是 uuid，调用方传 ID 即可重入）
//   - List 按 created_at DESC 排序
//   - Count 是 module 维度的轻量计数（用于 worker_dlq_size 指标拉取）
type Repository interface {
	Insert(ctx context.Context, entry *DeadLetter) error
	List(ctx context.Context, filter Filter) (*model.ListResponse[DeadLetter], error)
	Get(ctx context.Context, id uuid.UUID) (*DeadLetter, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context, sourceModule string) (int64, error)
}
