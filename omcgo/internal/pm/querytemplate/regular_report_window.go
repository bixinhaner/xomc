package querytemplate

import (
	"fmt"
	"strings"
	"time"
)

func RegularReportWindow(period RegularReportPeriod, now time.Time, location *time.Location) (time.Time, time.Time, error) {
	if location == nil {
		location = time.UTC
	}
	localNow := now.In(location)
	var end time.Time
	switch period {
	case RegularReport15Min:
		minute := (localNow.Minute() / 15) * 15
		end = time.Date(localNow.Year(), localNow.Month(), localNow.Day(), localNow.Hour(), minute, 0, 0, location)
	case RegularReportHour:
		end = time.Date(localNow.Year(), localNow.Month(), localNow.Day(), localNow.Hour(), 0, 0, 0, location)
	case RegularReportDay:
		end = time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, location)
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("unsupported regular report period %q", period)
	}
	switch period {
	case RegularReport15Min:
		return end.Add(-15 * time.Minute), end, nil
	case RegularReportHour:
		return end.Add(-time.Hour), end, nil
	case RegularReportDay:
		return end.AddDate(0, 0, -1), end, nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("unsupported regular report period %q", period)
	}
}

func RegularReportNextRun(period RegularReportPeriod, sendTime string, after time.Time, location *time.Location) (time.Time, error) {
	if location == nil {
		location = time.UTC
	}
	parsed, err := time.Parse("15:04", strings.TrimSpace(sendTime))
	if err != nil || !validRegularReportPeriod(period) {
		return time.Time{}, fmt.Errorf("invalid regular report schedule")
	}
	localAfter := after.In(location)
	switch period {
	case RegularReportDay:
		candidate := time.Date(localAfter.Year(), localAfter.Month(), localAfter.Day(), parsed.Hour(), parsed.Minute(), 0, 0, location)
		if !candidate.After(localAfter) {
			candidate = candidate.AddDate(0, 0, 1)
		}
		return candidate, nil
	case RegularReportHour:
		candidate := time.Date(localAfter.Year(), localAfter.Month(), localAfter.Day(), localAfter.Hour(), parsed.Minute(), 0, 0, location)
		if !candidate.After(localAfter) {
			candidate = candidate.Add(time.Hour)
		}
		return candidate, nil
	case RegularReport15Min:
		base := time.Date(localAfter.Year(), localAfter.Month(), localAfter.Day(), localAfter.Hour(), 0, 0, 0, location)
		candidate := base.Add(time.Duration(parsed.Minute()) * time.Minute)
		for !candidate.After(localAfter) {
			candidate = candidate.Add(15 * time.Minute)
		}
		return candidate, nil
	default:
		return time.Time{}, fmt.Errorf("unsupported regular report period %q", period)
	}
}
