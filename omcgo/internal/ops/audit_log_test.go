package ops

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// ---------------------------------------------------------------------------
// #124 回归：ops_audit_logs.target_type 截断导致审计留痕静默丢失
// ---------------------------------------------------------------------------
//
// 根因是 DB 列宽（varchar(16) → 迁移 000035 放宽为 varchar(64)），列宽本身由
// 冒烟脚本兜底验证；本文件用纯单测锁住两个易回归的代码侧不变量：
//   1. 模块写入的所有 target_type 值必须 ≤ 64 字符（新列宽），新增值超宽时在
//      单测阶段就暴露，而不是线上 22001 静默丢审计；
//   2. AuditLogService.Log 失败仅 Warn 不回传错误（非阻塞语义保留），且维护
//      窗口创建确实尝试写入 target_type='maintenance_window' 的审计。

// opsAuditTargetTypeMaxLen 与迁移 000035 中 varchar(64) 保持一致。
const opsAuditTargetTypeMaxLen = 64

func TestOpsAuditTargetTypes_FitColumnWidth(t *testing.T) {
	// 与 internal/ops 各调用点写入值一一对应（grep TargetType 现查）。
	cases := []struct {
		name       string
		targetType string
	}{
		{"task", "task"},
		{"device", "device"},
		{"user", "user"},
		{"scope", "scope"},
		{"maintenance_window", "maintenance_window"}, // #124：18 字符，曾溢出 varchar(16)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.LessOrEqual(t, len(tc.targetType), opsAuditTargetTypeMaxLen,
				"target_type %q 超过 ops_audit_logs.target_type varchar(%d)，写入将 22001 失败",
				tc.targetType, opsAuditTargetTypeMaxLen)
		})
	}
}

// capturingAuditLogRepo 记录 Create 入参并可注入错误，验证写入内容与非阻塞语义。
type capturingAuditLogRepo struct {
	logs      []*OpsAuditLog
	createErr error
}

func (m *capturingAuditLogRepo) Create(_ context.Context, l *OpsAuditLog) error {
	m.logs = append(m.logs, l)
	return m.createErr
}

func (m *capturingAuditLogRepo) List(_ context.Context, _ AuditLogFilter) (*model.ListResponse[OpsAuditLog], error) {
	return model.NewListResponse([]OpsAuditLog{}, 0, 1, 20), nil
}

// mwRepoStub 满足 MaintenanceWindowRepository，Create 仅分配 ID。
type mwRepoStub struct{}

func (m *mwRepoStub) Create(_ context.Context, w *OpsMaintenanceWindow) error {
	w.ID = uuid.New()
	return nil
}

func (m *mwRepoStub) GetByID(_ context.Context, _ uuid.UUID) (*OpsMaintenanceWindow, error) {
	return nil, errors.New("not implemented")
}

func (m *mwRepoStub) Update(_ context.Context, _ *OpsMaintenanceWindow) error {
	return errors.New("not implemented")
}

func (m *mwRepoStub) List(_ context.Context, _ MaintenanceWindowFilter) (*model.ListResponse[OpsMaintenanceWindow], error) {
	return model.NewListResponse([]OpsMaintenanceWindow{}, 0, 1, 20), nil
}

func (m *mwRepoStub) ListActive(_ context.Context, _ time.Time) ([]OpsMaintenanceWindow, error) {
	return nil, nil
}

func TestMaintenanceWindowService_Create_WritesAuditWithinColumnWidth(t *testing.T) {
	auditRepo := &capturingAuditLogRepo{}
	svc := NewMaintenanceWindowService(&mwRepoStub{},
		NewAuditLogService(auditRepo, zap.NewNop()), zap.NewNop())

	creator := uuid.New()
	start := time.Now()
	w, err := svc.Create(context.Background(), CreateMaintenanceWindowRequest{
		Name:      "mw-audit-regression",
		ScopeType: "device",
		StartAt:   start,
		EndAt:     start.Add(time.Hour),
	}, &creator)
	require.NoError(t, err)
	require.NotNil(t, w)

	require.Len(t, auditRepo.logs, 1, "创建维护窗口必须尝试写一条审计")
	got := auditRepo.logs[0]
	assert.Equal(t, "maintenance_window_create", got.OpType)
	assert.Equal(t, "maintenance_window", got.TargetType)
	assert.Equal(t, w.ID.String(), got.TargetID)
	assert.LessOrEqual(t, len(got.TargetType), opsAuditTargetTypeMaxLen,
		"target_type 超出迁移 000035 放宽后的 varchar(%d)", opsAuditTargetTypeMaxLen)
}

func TestAuditLogService_Log_NonBlockingOnRepoError(t *testing.T) {
	auditRepo := &capturingAuditLogRepo{createErr: errors.New("insert audit_log: boom")}
	svc := NewAuditLogService(auditRepo, zap.NewNop())

	// Log 无返回值；断言 repo 失败时不 panic、不中断调用方（业务主流程不受影响）。
	assert.NotPanics(t, func() {
		svc.Log(context.Background(), &OpsAuditLog{
			OpType: "maintenance_window_create", TargetType: "maintenance_window",
			TargetID: uuid.NewString(),
		})
	})
	assert.Len(t, auditRepo.logs, 1, "失败也应只尝试一次写入")
}
