//go:build integration

package device

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #378：回收站批量恢复因 serial_number 部分唯一索引 (serial_number, carrier)
// WHERE deleted_at IS NULL 冲突致整批 UPDATE 回滚 500。本测试在真实 PostgreSQL
// 上验证唯一索引行为 + NOT EXISTS 守卫的「可部分成功」语义（纯 mock 无法验证
// 部分唯一索引）。
//
// 无 PG 时（OMCGO_DB_DSN 未设）Skip。
// 运行：OMCGO_DB_DSN=postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable \
//        go test -tags integration -count=1 -run TestRestoreDevices ./internal/device/...

// insertDeviceRow 插入一行最小 devices 记录（仅 NOT NULL 列必填），返回 id。
func insertDeviceRow(t *testing.T, ctx context.Context, r *PgDeviceRepository, sn, carrier string, deleted bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	var deletedExpr string
	if deleted {
		deletedExpr = "NOW()"
	} else {
		deletedExpr = "NULL"
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO devices (id, serial_number, oui, carrier, technology, deleted_at)
		 VALUES ($1, $2, '0019AB', $3, 'lte', `+deletedExpr+`)`,
		id, sn, carrier)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = r.pool.Exec(context.Background(), `DELETE FROM devices WHERE id = $1`, id)
	})
	return id
}

func TestRestoreDevices_PartialSuccessWithSNConflict_RealPG(t *testing.T) {
	pool := openPoolOrSkipDevice(t)
	ctx := context.Background()
	repo := &PgDeviceRepository{pool: pool}

	const carrier = "cmcc"
	snConflict := "QA614-378-CONFLICT-" + uuid.NewString()[:8]
	snClean := "QA614-378-CLEAN-" + uuid.NewString()[:8]

	// 场景：snConflict 既有活跃行又有软删行（恢复会撞部分唯一索引）；snClean 只有软删行。
	_ = insertDeviceRow(t, ctx, repo, snConflict, carrier, false)            // 活跃行
	deletedConflict := insertDeviceRow(t, ctx, repo, snConflict, carrier, true)  // 软删行（冲突）
	deletedClean := insertDeviceRow(t, ctx, repo, snClean, carrier, true)        // 软删行（可恢复）

	t.Run("batch with one SN conflict -> 其余成功 + 冲突回传，不整批回滚", func(t *testing.T) {
		res, err := repo.RestoreDevices(ctx, []uuid.UUID{deletedConflict, deletedClean})
		require.NoError(t, err, "冲突场景不应返回 DB 错误（不再 23505 整批回滚）")
		assert.EqualValues(t, 1, res.Restored, "干净设备应恢复")
		assert.EqualValues(t, 1, res.Skipped, "冲突设备应被跳过")
		require.Len(t, res.Conflicts, 1)
		assert.Equal(t, deletedConflict.String(), res.Conflicts[0].ID)
		assert.Equal(t, snConflict, res.Conflicts[0].SerialNumber)
		assert.Equal(t, "serial_number_conflict", res.Conflicts[0].Reason)

		// 校验落库：干净行 deleted_at 已置 NULL，冲突行仍软删。
		var cleanDeleted, conflictDeleted *string
		require.NoError(t, pool.QueryRow(ctx, `SELECT deleted_at::text FROM devices WHERE id=$1`, deletedClean).Scan(&cleanDeleted))
		require.NoError(t, pool.QueryRow(ctx, `SELECT deleted_at::text FROM devices WHERE id=$1`, deletedConflict).Scan(&conflictDeleted))
		assert.Nil(t, cleanDeleted, "干净设备 deleted_at 应被置 NULL")
		assert.NotNil(t, conflictDeleted, "冲突设备应仍保持软删")
	})
}

func TestRestoreDevices_AllCleanAllSucceed_RealPG(t *testing.T) {
	pool := openPoolOrSkipDevice(t)
	ctx := context.Background()
	repo := &PgDeviceRepository{pool: pool}

	const carrier = "cmcc"
	id1 := insertDeviceRow(t, ctx, repo, "QA614-378-A-"+uuid.NewString()[:8], carrier, true)
	id2 := insertDeviceRow(t, ctx, repo, "QA614-378-B-"+uuid.NewString()[:8], carrier, true)

	res, err := repo.RestoreDevices(ctx, []uuid.UUID{id1, id2})
	require.NoError(t, err)
	assert.EqualValues(t, 2, res.Restored)
	assert.EqualValues(t, 0, res.Skipped)
	assert.Empty(t, res.Conflicts)
}

func TestRestoreDevices_SingleConflict_NoErrorPartialResult_RealPG(t *testing.T) {
	pool := openPoolOrSkipDevice(t)
	ctx := context.Background()
	repo := &PgDeviceRepository{pool: pool}

	const carrier = "cmcc"
	sn := "QA614-378-SINGLE-" + uuid.NewString()[:8]
	_ = insertDeviceRow(t, ctx, repo, sn, carrier, false)        // 活跃行
	deleted := insertDeviceRow(t, ctx, repo, sn, carrier, true)  // 软删行（冲突）

	res, err := repo.RestoreDevices(ctx, []uuid.UUID{deleted})
	require.NoError(t, err, "单台冲突应返回明确业务结果而非 500/DB error")
	assert.EqualValues(t, 0, res.Restored)
	assert.EqualValues(t, 1, res.Skipped)
	require.Len(t, res.Conflicts, 1)
	assert.Equal(t, sn, res.Conflicts[0].SerialNumber)
}
