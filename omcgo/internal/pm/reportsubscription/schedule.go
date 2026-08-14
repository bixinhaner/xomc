package reportsubscription

import (
	"fmt"
	"sort"
	"time"
)

// NormalizeSendTimes 校验 HH:mm、去重并排序。
func NormalizeSendTimes(values []string) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		parsed, err := time.Parse("15:04", value)
		if err != nil || parsed.Format("15:04") != value {
			return nil, fmt.Errorf("%w: send_times must use HH:mm", ErrInvalid)
		}
		normalized := parsed.Format("15:04")
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) == 0 || len(out) > 8 {
		return nil, fmt.Errorf("%w: send_times must contain 1 to 8 values", ErrInvalid)
	}
	sort.Strings(out)
	return out, nil
}

// NextRunAt 返回系统时区中严格晚于 after 的最近发送时刻。
func NextRunAt(after time.Time, sendTimes []string, location *time.Location) (time.Time, error) {
	normalized, err := NormalizeSendTimes(sendTimes)
	if err != nil {
		return time.Time{}, err
	}
	if location == nil {
		location = time.Local
	}
	local := after.In(location)
	for dayOffset := 0; dayOffset <= 1; dayOffset++ {
		for _, value := range normalized {
			parsed, _ := time.Parse("15:04", value)
			candidate := time.Date(local.Year(), local.Month(), local.Day()+dayOffset, parsed.Hour(), parsed.Minute(), 0, 0, location)
			if candidate.After(local) {
				return candidate, nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("%w: calculate next run", ErrInvalid)
}

// NaturalWindow 计算包含完整数据的最近一个自然窗口，使用 [start,end) 半开区间。
func NaturalWindow(period Period, now time.Time, location *time.Location) (time.Time, time.Time, error) {
	if location == nil {
		location = time.Local
	}
	local := now.In(location)
	var start, end time.Time
	switch period {
	case Period15Min:
		minute := (local.Minute() / 15) * 15
		end = time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), minute, 0, 0, location)
		start = end.Add(-15 * time.Minute)
	case PeriodHourly:
		end = time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), 0, 0, 0, location)
		start = end.Add(-time.Hour)
	case PeriodDaily:
		end = time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
		start = end.AddDate(0, 0, -1)
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("%w: unsupported period %q", ErrInvalid, period)
	}
	return start, end, nil
}
