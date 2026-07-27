package stream

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

const slotDuration = 15 * time.Minute

type Window struct {
	Start time.Time
	End   time.Time
}

type WindowKey struct {
	TaskID        uuid.UUID
	TaskVersionID uuid.UUID
	Granularity   Granularity
	Start         time.Time
	End           time.Time
}

func WindowFor(slotStart time.Time, granularity Granularity, location *time.Location) (Window, error) {
	if location == nil {
		location = time.UTC
	}
	local := slotStart.In(location)
	var start, end time.Time
	switch granularity {
	case GranularityHourly:
		start = time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), 0, 0, 0, location)
		end = start.Add(time.Hour)
	case GranularityDaily:
		start = time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
		end = start.AddDate(0, 0, 1)
	case GranularityWeekly:
		daysSinceMonday := (int(local.Weekday()) + 6) % 7
		day := local.AddDate(0, 0, -daysSinceMonday)
		start = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, location)
		end = start.AddDate(0, 0, 7)
	case GranularityMonthly:
		start = time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, location)
		end = start.AddDate(0, 1, 0)
	default:
		return Window{}, fmt.Errorf("unsupported PM aggregation granularity %q", granularity)
	}
	return Window{Start: start.UTC(), End: end.UTC()}, nil
}

func expectedSlots(window Window, memberCount int) int64 {
	if memberCount <= 0 {
		return 0
	}
	return int64(window.End.Sub(window.Start)/slotDuration) * int64(memberCount)
}
