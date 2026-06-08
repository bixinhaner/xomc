package mml

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// 本文件端到端验证 mml_custom_command 的命名空间规则（连真实/容器 PG，连不上自动 t.Skip）。
// 规则（方案 X）：
//   - 私有命令：同 owner 不重名（service NameExistsForPrivate 查询 + DB 部分唯一索引兜底）
//   - 公共命令：全局不重名，跨所有用户（service NameExistsForPublic 查询；DB 无 public 约束）
//   - 私有 ↔ 公共：跨 scope 同名可共存
//
// owner_user_id 有 FK → users.id（fk_mml_custom_command_owner），故 owner 取 DB 现有真实
// 用户 id（不足 2 个则 t.Skip），不用随机 UUID。

// newScopeEnv 装配测试环境：真实 PG repo 接进 Service、两个真实 owner、隔离命名 + 退出清理。
func newScopeEnv(t *testing.T) (svc *Service, pool *pgxpool.Pool, ownerA, ownerB uuid.UUID, name string) {
	t.Helper()
	pool = newMMLTestPool(t) // 连不上 PG → t.Skip
	ctx := context.Background()

	var hasTbl bool
	if err := pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='mml_custom_command')",
	).Scan(&hasTbl); err != nil || !hasTbl {
		t.Skipf("mml_custom_command not present: err=%v exists=%v", err, hasTbl)
	}

	rows, err := pool.Query(ctx, "SELECT id FROM users ORDER BY created_at LIMIT 2")
	require.NoError(t, err)
	var owners []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		require.NoError(t, rows.Scan(&id))
		owners = append(owners, id)
	}
	rows.Close()
	require.NoError(t, rows.Err())
	if len(owners) < 2 {
		t.Skipf("需要至少 2 个真实用户做 owner，实际 %d", len(owners))
	}

	svc = NewService(&mockCommandRepo{}, &mockScriptRepo{}, &mockTaskRepo{},
		NewPgCustomCommandRepository(pool), nil, zap.NewNop())
	name = "itest_scope_" + uuid.NewString()[:8]
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM mml_custom_command WHERE command_name=$1", name)
	})
	return svc, pool, owners[0], owners[1], name
}

// scopeCmd 构造一条自定义命令。codeSuffix 仅为让 command_code 互不相同（command_code 非唯一约束，但便于排查）。
func scopeCmd(name, scope string, owner uuid.UUID, codeSuffix string) *MMLCustomCommand {
	return &MMLCustomCommand{
		CommandName:   name,
		CommandCode:   "LST " + codeSuffix,
		OperationType: "LST",
		CommandScope:  scope,
		Creator:       "itest",
		OwnerUserID:   &owner,
	}
}

func countByScope(t *testing.T, pool *pgxpool.Pool, name, scope string) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(context.Background(),
		"SELECT count(*) FROM mml_custom_command WHERE command_name=$1 AND command_scope=$2",
		name, scope).Scan(&n))
	return n
}

// 第一种：用户A 私有 test + 用户B 私有 test → 都允许（私有按 owner 隔离，跨用户同名 OK）。
func TestIntegration_CustomCommand_Scenario1_TwoUsersPrivateSameName(t *testing.T) {
	svc, pool, ownerA, ownerB, name := newScopeEnv(t)
	ctx := context.Background()

	_, err := svc.CreateCustomCommand(ctx, scopeCmd(name, "private", ownerA, "A"))
	require.NoError(t, err, "用户A 私有 test 应成功")

	_, err = svc.CreateCustomCommand(ctx, scopeCmd(name, "private", ownerB, "B"))
	require.NoError(t, err, "用户B 私有 test 应成功（跨用户私有同名允许）")

	assert.Equal(t, 2, countByScope(t, pool, name, "private"), "两条私有 test 应共存")
}

// 第二种：用户A 私有 test + 用户A 公共 test → 都允许（私有/公共不同命名空间，跨 scope 共存）。
func TestIntegration_CustomCommand_Scenario2_SameUserPrivateThenPublicSameName(t *testing.T) {
	svc, pool, ownerA, _, name := newScopeEnv(t)
	ctx := context.Background()

	_, err := svc.CreateCustomCommand(ctx, scopeCmd(name, "private", ownerA, "A"))
	require.NoError(t, err, "用户A 私有 test 应成功")

	_, err = svc.CreateCustomCommand(ctx, scopeCmd(name, "public", ownerA, "A2"))
	require.NoError(t, err, "用户A 公共 test 应成功（与自己的私有 test 跨 scope 共存）")

	assert.Equal(t, 1, countByScope(t, pool, name, "private"), "私有 1 条")
	assert.Equal(t, 1, countByScope(t, pool, name, "public"), "公共 1 条")
}

// 第三种：用户A 公共 test + 用户B 公共 test → 第二条被拒（公共全局唯一，跨所有用户）。
func TestIntegration_CustomCommand_Scenario3_TwoUsersPublicSameName(t *testing.T) {
	svc, pool, ownerA, ownerB, name := newScopeEnv(t)
	ctx := context.Background()

	_, err := svc.CreateCustomCommand(ctx, scopeCmd(name, "public", ownerA, "A"))
	require.NoError(t, err, "用户A 公共 test 应成功")

	_, err = svc.CreateCustomCommand(ctx, scopeCmd(name, "public", ownerB, "B"))
	require.Error(t, err, "用户B 公共 test 应被拒（公共全局唯一）")
	assert.True(t, errors.Is(err, commonerrors.ErrAlreadyExists), "应映射为 409 Conflict")

	assert.Equal(t, 1, countByScope(t, pool, name, "public"), "公共仍只有 1 条")
}

// 私有索引兜底：绕过 service 查询预检直连 repo，同 owner 私有重名应被 DB 索引
// uq_mml_custom_command_private_name_per_owner 以 23505 拒绝（证明 race 兜底真实存在）。
func TestIntegration_CustomCommand_PrivateIndexBackstop(t *testing.T) {
	svc, pool, ownerA, _, name := newScopeEnv(t)
	ctx := context.Background()

	_, err := svc.CreateCustomCommand(ctx, scopeCmd(name, "private", ownerA, "A"))
	require.NoError(t, err)

	repo := NewPgCustomCommandRepository(pool)
	err = repo.Create(ctx, scopeCmd(name, "private", ownerA, "B")) // 绕过 service 预检
	require.Error(t, err, "DB 索引应拒绝同 owner 私有重名")
	assert.True(t, isUniqueViolation(err), "应为 unique_violation(23505)；got=%v", err)
}
