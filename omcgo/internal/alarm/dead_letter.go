package alarm

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// DeadLetterRecord 是 webhook 派发耗尽重试后的死信记录。
// W2 T-0011 仅落库，扫描重投递留 T-0044。
type DeadLetterRecord struct {
	ID         uuid.UUID `db:"id" json:"id"`
	FilterID   uuid.UUID `db:"filter_id" json:"filter_id"`
	AlarmID    uuid.UUID `db:"alarm_id" json:"alarm_id"`
	Payload    []byte    `db:"payload" json:"payload"`
	LastError  string    `db:"last_error" json:"last_error"`
	RetryCount int       `db:"retry_count" json:"retry_count"`
	FailedAt   time.Time `db:"failed_at" json:"failed_at"`
}

// DeadLetterRepository 死信记录仓储接口。仅 Insert（最小必要面）。
type DeadLetterRepository interface {
	Insert(ctx context.Context, rec *DeadLetterRecord) error
}
