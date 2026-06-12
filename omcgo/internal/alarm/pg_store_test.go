package alarm

import (
	"strings"
	"testing"
	"time"

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

// localizedAlarmNameExpr 末尾必须有 '' 兜底：字典三源全 NULL 时整体退回空串，
// 否则 NULL 扫进非空 string 字段 model.Alarm.Description 会让历史告警列表 500。
func TestLocalizedAlarmNameExprHasEmptyStringFallback(t *testing.T) {
	for _, loc := range []appcontext.Locale{appcontext.LocaleZH, appcontext.LocaleEN} {
		expr := localizedAlarmNameExpr(loc, "alarms_history")
		assert.True(t, strings.Contains(expr, "alarms_history.description, '') AS description"),
			"locale %q 缺少 '' 兜底，可能让 NULL description 扫崩：%s", loc, expr)
	}
}