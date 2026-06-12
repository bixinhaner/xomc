package dashboard

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupMTTRTestService 连接 OMCGO_TEST_DB_DSN 指向的真实 PG，构造一个仅需
// pgPool + logger 的 dashboard.Service（GetOverallEfficiencyMetrics 只用这两项）。
// 未设置 DSN 时 SKIP（不算 PASS，符合验收要求）。
func setupMTTRTestService(t *testing.T) *Service {
	t.Helper()

	dsn := os.Getenv("OMCGO_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_TEST_DB_DSN not set, skipping MTTR integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err, "connect test DB")
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping test DB: %v", err)
	}
	t.Cleanup(pool.Close)

	// KPI/时序库分离后效率指标查询走 tsPool；单 DSN 集成测试里 alarms_history
	// 与业务表同库，两池都指向同一连接即可保持测试语义。
	return &Service{
		pgPool: pool,
		tsPool: pool,
		logger: zap.NewNop(),
	}
}

// insertAlarmHistory 往 alarms_history 插入一条记录。cleared 为 nil 表示未解决。
func insertAlarmHistory(t *testing.T, pool *pgxpool.Pool, raised time.Time, cleared *time.Time, marker string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO alarms_history (
			"time", alarm_id, device_id, device_sn, carrier, severity,
			alarm_identifier, status, raised_at, cleared_at, probable_cause, description
		) VALUES (
			$1, $2, $3, $4, 'cmcc', 1,
			$5, 'cleared', $6, $7, '', $8
		)`,
		raised, id, uuid.New(), "MTTRSN-"+id.String()[:8],
		"MTTR-"+id.String()[:8], raised, cleared, marker,
	)
	require.NoError(t, err, "insert alarms_history (%s)", marker)
}

// TestGetOverallEfficiencyMetrics_MTTRNonNegative 构造 正常 / 倒挂 / 未解决 三类
// 已清除告警，验证 GetOverallEfficiencyMetrics 的 MTTR≥0，且时间倒挂记录不计入平均
// （平均应≈正常告警的真实解决时长）。覆盖死判 mttr_test。
func TestGetOverallEfficiencyMetrics_MTTRNonNegative(t *testing.T) {
	svc := setupMTTRTestService(t)
	pool := svc.pgPool

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 隔离：清空 alarms_history，避免既有数据干扰断言
	_, err := pool.Exec(ctx, "TRUNCATE TABLE alarms_history")
	require.NoError(t, err, "truncate alarms_history")

	now := time.Now().UTC()

	// 1) 正常告警：解决时长 60 分钟（raised < cleared）。在近30天窗口内。
	normalRaised := now.Add(-2 * time.Hour)
	normalCleared := normalRaised.Add(60 * time.Minute)
	insertAlarmHistory(t, pool, normalRaised, &normalCleared, "normal-60min")

	// 2) 倒挂告警：cleared 早于 raised（脏数据），解决时长为 -120 分钟。
	//    cleared_at 也需落在近30天窗口（WHERE cleared_at > NOW()-30d）才会进聚合候选。
	invRaised := now.Add(-1 * time.Hour)
	invCleared := invRaised.Add(-120 * time.Minute) // 早于 raised
	insertAlarmHistory(t, pool, invRaised, &invCleared, "inverted-neg")

	// 3) 未解决告警：cleared_at 为空，不应计入 MTTR。
	//    注意：本视图/查询按 WHERE cleared_at > NOW()-30d 过滤，未解决记录天然被排除，
	//    这里仍插入一条以确认不会污染结果。
	unresolvedRaised := now.Add(-30 * time.Minute)
	insertAlarmHistory(t, pool, unresolvedRaised, nil, "unresolved")

	metrics, err := svc.GetOverallEfficiencyMetrics(ctx)
	require.NoError(t, err, "GetOverallEfficiencyMetrics")
	require.NotNil(t, metrics)

	// 死判核心：MTTR 不为负
	require.GreaterOrEqual(t, metrics.AvgResolveMinutes, 0.0,
		"MTTR 必须 >= 0（倒挂脏数据不得拉成负值），实测=%v", metrics.AvgResolveMinutes)

	// 倒挂记录不计入：平均应≈正常告警的 60 分钟。
	// 若倒挂的 -120 被计入，(60 + -120)/2 = -30；排除后应为 60。
	require.InDelta(t, 60.0, metrics.AvgResolveMinutes, 0.5,
		"MTTR 应≈正常告警 60 分钟（倒挂记录不计入），实测=%v", metrics.AvgResolveMinutes)
}
