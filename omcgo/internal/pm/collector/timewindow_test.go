package collector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateTimeWindow_Valid(t *testing.T) {
	end := time.Now()
	start := end.Add(-15 * time.Minute)
	err := ValidateTimeWindow(start, end, 15*time.Minute, end.Add(5*time.Second))
	require.NoError(t, err)
}

func TestValidateTimeWindow_Valid_WithinTolerance(t *testing.T) {
	end := time.Now()
	// 窗口 = 15min + 30s（容差内）
	start := end.Add(-15*time.Minute - 30*time.Second)
	err := ValidateTimeWindow(start, end, 15*time.Minute, end)
	require.NoError(t, err)
}

func TestValidateTimeWindow_MissingFields(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name        string
		start, end  time.Time
		ingest      time.Time
	}{
		{"both zero", time.Time{}, time.Time{}, now},
		{"start zero", time.Time{}, now, now},
		{"end zero", now.Add(-15 * time.Minute), time.Time{}, now},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTimeWindow(tc.start, tc.end, 15*time.Minute, tc.ingest)
			require.ErrorIs(t, err, ErrTimeWindowMissing)
		})
	}
}

func TestValidateTimeWindow_Reversed(t *testing.T) {
	end := time.Now()
	start := end.Add(15 * time.Minute) // 反了
	err := ValidateTimeWindow(start, end, 15*time.Minute, end)
	require.ErrorIs(t, err, ErrTimeWindowReversed)

	// start == end 也算反序
	err = ValidateTimeWindow(end, end, 15*time.Minute, end)
	require.ErrorIs(t, err, ErrTimeWindowReversed)
}

func TestValidateTimeWindow_SpanDeviates(t *testing.T) {
	end := time.Now()
	// 窗口 16min，但 granDuration 15min，超过 60s 容差
	start := end.Add(-16*time.Minute - 5*time.Second)
	err := ValidateTimeWindow(start, end, 15*time.Minute, end)
	require.ErrorIs(t, err, ErrTimeWindowSpanDeviates)

	// 窗口 13min，granDuration 15min，超过 60s 容差
	start = end.Add(-13*time.Minute - 30*time.Second)
	err = ValidateTimeWindow(start, end, 15*time.Minute, end)
	require.ErrorIs(t, err, ErrTimeWindowSpanDeviates)
}

func TestValidateTimeWindow_StaleFile(t *testing.T) {
	end := time.Now().Add(-48 * time.Hour) // 2 天前
	start := end.Add(-15 * time.Minute)
	ingest := time.Now()
	err := ValidateTimeWindow(start, end, 15*time.Minute, ingest)
	require.ErrorIs(t, err, ErrTimeWindowStale)
}

func TestValidateTimeWindow_GranDurationZero_SkipsSpanCheck(t *testing.T) {
	// granDuration=0 时跳过窗口长度校验（兼容老文件无 duration）
	end := time.Now()
	start := end.Add(-1 * time.Hour) // 1 小时窗口
	err := ValidateTimeWindow(start, end, 0, end)
	require.NoError(t, err)
}

func TestValidateTimeWindow_IngestZero_SkipsStaleCheck(t *testing.T) {
	// ingest=zero 时跳过 stale 校验（合理：未填充时不判定）
	end := time.Now().Add(-48 * time.Hour)
	start := end.Add(-15 * time.Minute)
	err := ValidateTimeWindow(start, end, 15*time.Minute, time.Time{})
	require.NoError(t, err)
}
