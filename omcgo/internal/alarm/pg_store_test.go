package alarm

import (
	"strings"
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

// localizedAlarmNameExpr 末尾必须有 ” 兜底：字典三源全 NULL 时整体退回空串，
// 否则 NULL 扫进非空 string 字段 model.Alarm.Description 会让历史告警列表 500。
func TestLocalizedAlarmNameExprHasEmptyStringFallback(t *testing.T) {
	for _, loc := range []appcontext.Locale{appcontext.LocaleZH, appcontext.LocaleEN} {
		expr := localizedAlarmNameExpr(loc, "alarms_history")
		assert.True(t, strings.Contains(expr, "alarms_history.description, '') AS description"),
			"locale %q 缺少 '' 兜底，可能让 NULL description 扫崩：%s", loc, expr)
	}
}

func TestLocalizedProbableCauseExpr(t *testing.T) {
	t.Run("英文 OMC 告警优先英文并保留完整回退链", func(t *testing.T) {
		expr := localizedProbableCauseExpr(appcontext.LocaleEN, "alarms_active")
		assert.Contains(t, expr, "LOWER(TRIM(COALESCE(alarms_active.alarm_source, ''))) = 'omc'")
		assert.Less(t, strings.Index(expr, "ad.en_probable_cause"), strings.Index(expr, "ad.cn_probable_cause"))
		assert.Less(t, strings.Index(expr, "ad.cn_probable_cause"), strings.Index(expr, "alarms_active.probable_cause"))
		assert.Contains(t, expr, "ELSE COALESCE(alarms_active.probable_cause, '')")
		assert.Contains(t, expr, "AS probable_cause")
	})

	t.Run("中文 OMC 告警优先中文", func(t *testing.T) {
		expr := localizedProbableCauseExpr(appcontext.LocaleZH, "alarms_history")
		assert.Less(t, strings.Index(expr, "ad.cn_probable_cause"), strings.Index(expr, "ad.en_probable_cause"))
		assert.Contains(t, expr, "alarms_history.probable_cause")
	})
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
