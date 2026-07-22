package alarm

import (
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	appcontext "github.com/omcgo/omcgo/internal/core/context"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
)

func TestHistoryUpdatedAtPrefersLastUpdatedAt(t *testing.T) {
	updatedAt := time.Now().Add(-5 * time.Minute).UTC().Truncate(time.Second)
	lastUpdatedAt := time.Now().Add(-2 * time.Minute).UTC().Truncate(time.Second)

	got := historyUpdatedAt(&model.Alarm{
		UpdatedAt:     updatedAt,
		LastUpdatedAt: lastUpdatedAt,
	})

	assert.Equal(t, lastUpdatedAt, got)
}

func TestHistoryUpdatedAtFallsBackToUpdatedAt(t *testing.T) {
	updatedAt := time.Now().Add(-3 * time.Minute).UTC().Truncate(time.Second)

	got := historyUpdatedAt(&model.Alarm{UpdatedAt: updatedAt})

	assert.Equal(t, updatedAt, got)
}

func TestHistoryUpdatedAtReturnsZeroWhenNoBusinessTimestampExists(t *testing.T) {
	got := historyUpdatedAt(&model.Alarm{})

	assert.True(t, got.IsZero())
}

func TestLocalizedAlarmNameExprUsesDefinitionWhenIdentifierMatches(t *testing.T) {
	assert.Equal(t,
		"CASE WHEN ad.identifier IS NULL THEN COALESCE(alarms_active.description, '') "+
			"ELSE COALESCE(NULLIF(ad.en_name, ''), NULLIF(ad.cn_name, ''), '') END AS description",
		localizedAlarmNameExpr(appcontext.LocaleEN, "alarms_active"),
	)
	assert.Equal(t,
		"CASE WHEN ad.identifier IS NULL THEN COALESCE(alarms_history.description, '') "+
			"ELSE COALESCE(NULLIF(ad.cn_name, ''), NULLIF(ad.en_name, ''), '') END AS description",
		localizedAlarmNameExpr(appcontext.LocaleZH, "alarms_history"),
	)
}

func TestLocalizedProbableCauseExprUsesDefinitionRegardlessOfAlarmSource(t *testing.T) {
	assert.Equal(t,
		"CASE WHEN ad.identifier IS NULL THEN COALESCE(alarms_active.probable_cause, '') "+
			"ELSE COALESCE(NULLIF(ad.en_probable_cause, ''), NULLIF(ad.cn_probable_cause, ''), '') END AS probable_cause",
		localizedProbableCauseExpr(appcontext.LocaleEN, "alarms_active"),
	)
	assert.Equal(t,
		"CASE WHEN ad.identifier IS NULL THEN COALESCE(alarms_history.probable_cause, '') "+
			"ELSE COALESCE(NULLIF(ad.cn_probable_cause, ''), NULLIF(ad.en_probable_cause, ''), '') END AS probable_cause",
		localizedProbableCauseExpr(appcontext.LocaleZH, "alarms_history"),
	)
}

func TestLocalizedAlarmSelectsProjectProbableCause(t *testing.T) {
	activeSQL, _, err := activeAlarmLocalizedSelect(appcontext.LocaleEN).
		Where(squirrel.Eq{"alarms_active.id": uuid.Nil}).ToSql()
	assert.NoError(t, err)
	assert.Contains(t, activeSQL, "ad.en_probable_cause")
	assert.Contains(t, activeSQL, "LEFT JOIN alarm_definitions ad")

	historySQL, _, err := historyAlarmSelect(appcontext.LocaleEN).
		Where(squirrel.Eq{"alarms_history.alarm_id": uuid.Nil}).ToSql()
	assert.NoError(t, err)
	assert.Contains(t, historySQL, "ad.en_probable_cause")
	assert.Contains(t, historySQL, "LEFT JOIN alarm_definition_dim ad")
}

func TestInternalAlarmSelectRemainsRaw(t *testing.T) {
	activeSQL, _, err := activeAlarmSelect().Where(squirrel.Eq{"alarms_active.id": uuid.Nil}).ToSql()
	assert.NoError(t, err)
	assert.NotContains(t, activeSQL, "alarm_definitions ad")
	assert.NotContains(t, activeSQL, "CASE WHEN LOWER")

	historySQL, _, err := historyAlarmRawSelect().Where(squirrel.Eq{"alarms_history.alarm_id": uuid.Nil}).ToSql()
	assert.NoError(t, err)
	assert.NotContains(t, historySQL, "alarm_definition_dim ad")
	assert.NotContains(t, historySQL, "CASE WHEN LOWER")
}
