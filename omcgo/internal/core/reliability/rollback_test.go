package reliability

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// stubTx 实现 txRollbacker，返回预置的回滚错误。
type stubTx struct {
	err    error
	called int
}

func (s *stubTx) Rollback(_ context.Context) error {
	s.called++
	return s.err
}

func TestRollbackTx(t *testing.T) {
	tests := []struct {
		name      string
		rbErr     error
		wantWarns int
	}{
		{
			name:      "成功提交后回滚为 no-op（ErrTxClosed）不记录",
			rbErr:     pgx.ErrTxClosed,
			wantWarns: 0,
		},
		{
			name:      "回滚成功（nil）不记录",
			rbErr:     nil,
			wantWarns: 0,
		},
		{
			name:      "回滚真实失败（连接断裂）记 warn",
			rbErr:     errors.New("conn reset by peer"),
			wantWarns: 1,
		},
		{
			name:      "包装后的 ErrTxClosed 仍识别为 no-op",
			rbErr:     errors.Join(errors.New("wrap"), pgx.ErrTxClosed),
			wantWarns: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zap.WarnLevel)
			logger := zap.New(core)
			tx := &stubTx{err: tt.rbErr}

			RollbackTx(context.Background(), tx, logger, "unitTest")

			if tx.called != 1 {
				t.Fatalf("Rollback 应被调用 1 次，实际 %d 次", tx.called)
			}
			if got := logs.FilterMessage("transaction rollback failed").Len(); got != tt.wantWarns {
				t.Fatalf("warn 记录数 = %d，期望 %d", got, tt.wantWarns)
			}
		})
	}
}

// TestRollbackTx_NilLogger 验证 logger 为 nil 时不 panic 且仍消费回滚错误。
func TestRollbackTx_NilLogger(t *testing.T) {
	tx := &stubTx{err: errors.New("boom")}
	RollbackTx(context.Background(), tx, nil, "nilLogger")
	if tx.called != 1 {
		t.Fatalf("Rollback 应被调用 1 次，实际 %d 次", tx.called)
	}
}
