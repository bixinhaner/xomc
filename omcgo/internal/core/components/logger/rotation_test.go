package logger

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func resetOverride() { rotationOverride.Store(nil) }

func TestEffectiveRotation_NoOverride(t *testing.T) {
	resetOverride()
	keep, age, maxSize := effectiveRotation(10, 7*24*time.Hour)
	assert.Equal(t, 10, keep)
	assert.Equal(t, 7*24*time.Hour, age)
	assert.Equal(t, 0, maxSize, "无 override 时不额外做 size 切割")
}

func TestEffectiveRotation_PartialOverride(t *testing.T) {
	resetOverride()
	// 只覆盖 keep + maxSize；maxAge 字段为 0 → 维持启动值。
	SetRotationOverride(RotationOverride{KeepUncompressed: 3, MaxSizeMB: 20})
	keep, age, maxSize := effectiveRotation(10, 7*24*time.Hour)
	assert.Equal(t, 3, keep, "keep 被覆盖")
	assert.Equal(t, 7*24*time.Hour, age, "maxAge 未配 → 启动值")
	assert.Equal(t, 20, maxSize, "maxSize 被覆盖")
	resetOverride()
}

func TestEffectiveRotation_FullOverride(t *testing.T) {
	resetOverride()
	SetRotationOverride(RotationOverride{KeepUncompressed: 5, MaxAgeDays: 14, MaxSizeMB: 100})
	keep, age, maxSize := effectiveRotation(10, 7*24*time.Hour)
	assert.Equal(t, 5, keep)
	assert.Equal(t, 14*24*time.Hour, age)
	assert.Equal(t, 100, maxSize)
	resetOverride()
}

func TestEffectiveRotateInterval(t *testing.T) {
	resetOverride()
	assert.Equal(t, 5*time.Minute, effectiveRotateInterval(5*time.Minute))
	assert.Equal(t, DefaultRotateInterval, effectiveRotateInterval(0))

	SetRotationOverride(RotationOverride{RotateIntervalMinutes: 30})
	assert.Equal(t, 30*time.Minute, effectiveRotateInterval(5*time.Minute))
	SetRotationOverride(RotationOverride{RotateIntervalMinutes: -1})
	assert.Equal(t, 5*time.Minute, effectiveRotateInterval(5*time.Minute))
	assert.Equal(t, DefaultRotateInterval, effectiveRotateInterval(0))
	resetOverride()
}

func TestReadIntCfg(t *testing.T) {
	ctx := context.Background()
	lookup := func(_ context.Context, category, key string) (string, bool) {
		if category != RotationCategory {
			return "", false
		}
		switch key {
		case KeyMaxSizeMB:
			return "64", true
		case KeyMaxAgeDays:
			return "-1", true // ≤0 → 视为未覆盖
		case KeyKeepFiles:
			return "abc", true // 非法 → 0
		case KeyRotateIntervalMinutes:
			return "15", true
		}
		return "", false
	}
	assert.Equal(t, 64, readIntCfg(ctx, lookup, KeyMaxSizeMB))
	assert.Equal(t, 0, readIntCfg(ctx, lookup, KeyMaxAgeDays))
	assert.Equal(t, 0, readIntCfg(ctx, lookup, KeyKeepFiles))
	assert.Equal(t, 15, readIntCfg(ctx, lookup, KeyRotateIntervalMinutes))
	assert.Equal(t, 0, readIntCfg(ctx, lookup, "missing"))
	assert.Equal(t, 14, readIntCfgInCategory(ctx, func(_ context.Context, category, key string) (string, bool) {
		if category == RetentionCategory && key == KeyServiceDays {
			return "14", true
		}
		return "", false
	}, RetentionCategory, KeyServiceDays))
}

func TestStartRotationConfigWatcher_AppliesImmediately(t *testing.T) {
	resetOverride()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lookup := func(_ context.Context, category, key string) (string, bool) {
		if category == RotationCategory && key == KeyKeepFiles {
			return "7", true
		}
		return "", false
	}
	StartRotationConfigWatcher(ctx, lookup, nil)
	// 启动即应用一次（同步），无需等 tick。
	keep, _, _ := effectiveRotation(10, time.Hour)
	assert.Equal(t, 7, keep)
	resetOverride()
}

func TestStartRotationConfigWatcher_ServiceDaysOverridesLegacyArchiveAge(t *testing.T) {
	resetOverride()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lookup := func(_ context.Context, category, key string) (string, bool) {
		switch {
		case category == RotationCategory && key == KeyMaxAgeDays:
			return "7", true
		case category == RotationCategory && key == KeyServiceDays:
			return "14", true
		case category == RetentionCategory && key == KeyServiceDays:
			return "9", true
		}
		return "", false
	}
	StartRotationConfigWatcher(ctx, lookup, nil)

	_, age, _ := effectiveRotation(10, 3*24*time.Hour)
	assert.Equal(t, 14*24*time.Hour, age,
		"log.rotation.service_days 应优先于旧 log.rotation.max_age_days 和旧分类 service_days")
	resetOverride()
}

func TestStartRotationConfigWatcher_FallsBackToLegacyRetentionServiceDays(t *testing.T) {
	resetOverride()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lookup := func(_ context.Context, category, key string) (string, bool) {
		if category == RetentionCategory && key == KeyServiceDays {
			return "9", true
		}
		return "", false
	}
	StartRotationConfigWatcher(ctx, lookup, nil)

	_, age, _ := effectiveRotation(10, 3*24*time.Hour)
	assert.Equal(t, 9*24*time.Hour, age)
	resetOverride()
}

func TestValidateServiceLogDays(t *testing.T) {
	assert.NoError(t, ValidateServiceLogDays("1"))
	assert.NoError(t, ValidateServiceLogDays("3650"))
	assert.Error(t, ValidateServiceLogDays("0"))
	assert.Error(t, ValidateServiceLogDays("3651"))
	assert.Error(t, ValidateServiceLogDays("not-a-number"))
	assert.NoError(t, ValidateDatabaseLogDays("180"))
}
