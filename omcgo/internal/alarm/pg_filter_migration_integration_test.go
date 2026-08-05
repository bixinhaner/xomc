package alarm

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestPgAlarmFilterMigrationBarrier_Integration(t *testing.T) {
	dsn := os.Getenv("ALARM_FILTER_MIGRATION_TEST_DSN")
	if dsn == "" {
		t.Skip("set ALARM_FILTER_MIGRATION_TEST_DSN to a dedicated database")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	legacyRuleID, notificationRuleID, versionID := uuid.New(), uuid.New(), uuid.New()
	updatedAt := time.Now().UTC().Truncate(time.Microsecond)
	_, err = pool.Exec(ctx,
		"INSERT INTO notification_rules (id,name,priority,created_by) VALUES ($1,$2,1,'integration')",
		notificationRuleID, "migration-"+notificationRuleID.String())
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		"INSERT INTO notification_rule_versions (id,rule_id,version_no,created_by,published_at) VALUES ($1,$2,1,'integration',now())",
		versionID, notificationRuleID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO alarm_filters
  (id,name,filter_type,alarm_identifiers,action,priority,enabled,created_at,updated_at)
VALUES ($1,$2,'alarm_identifier',ARRAY['DEVICE_OFFLINE'],'notify_email',10,true,$3,$3)`,
		legacyRuleID, "legacy-"+legacyRuleID.String(), updatedAt)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM alarm_filters WHERE id=$1", legacyRuleID)
		_, _ = pool.Exec(context.Background(), "DELETE FROM notification_rules WHERE id=$1", notificationRuleID)
	})

	repository := NewPgAlarmFilterRuleRepository(pool)
	err = repository.ReplaceNotifyEmailWithBarrier(
		ctx, legacyRuleID, updatedAt, notificationRuleID, versionID, "integration",
	)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrAlarmFilterMigrationPrecondition))

	_, err = pool.Exec(ctx,
		"UPDATE notification_rules SET current_enabled_version_id=$2 WHERE id=$1",
		notificationRuleID, versionID)
	require.NoError(t, err)
	require.NoError(t, repository.ReplaceNotifyEmailWithBarrier(
		ctx, legacyRuleID, updatedAt, notificationRuleID, versionID, "integration",
	))

	var action, updatedBy string
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT action,updated_by FROM alarm_filters WHERE id=$1", legacyRuleID,
	).Scan(&action, &updatedBy))
	require.Equal(t, FilterActionLegacyNotificationBarrier, action)
	require.Equal(t, "integration", updatedBy)

	barrier, err := repository.GetByID(ctx, legacyRuleID)
	require.NoError(t, err)
	barrier.Name = "ordinary-update-must-not-touch-barrier"
	require.ErrorIs(t, repository.Update(ctx, barrier), ErrAlarmFilterBarrierManaged)
	require.ErrorIs(t, repository.Toggle(ctx, legacyRuleID), ErrAlarmFilterBarrierManaged)
	require.ErrorIs(t, repository.Delete(ctx, legacyRuleID), ErrAlarmFilterBarrierManaged)
	require.ErrorIs(t, repository.Create(ctx, &AlarmFilterRule{
		ID: uuid.New(), Name: "ordinary-create-must-not-create-barrier", FilterType: FilterTypeDevice,
		Action: FilterActionLegacyNotificationBarrier,
	}), ErrAlarmFilterBarrierManaged)
}
