package aggregator

import "time"

// ChainedBucket 是上游 bucket 成功后要入队的下游 bucket。
type ChainedBucket struct {
	JobType string
	Start   time.Time
	End     time.Time
}

// RollupBucketsFor 返回设备级跨粒度后继 bucket。
func RollupBucketsFor(jobType string, start, end time.Time, loc *time.Location) []ChainedBucket {
	if loc == nil {
		loc = time.UTC
	}
	localEnd := end.In(loc)
	if !isAlignedBoundary(localEnd) {
		return nil
	}

	switch jobType {
	case JobTypeHourly:
		if localEnd.Hour() != 0 {
			return nil
		}
		dayStart := localEnd.AddDate(0, 0, -1)
		return []ChainedBucket{{JobType: JobTypeDaily, Start: dayStart, End: localEnd}}
	case JobTypeDaily:
		out := make([]ChainedBucket, 0, 2)
		if localEnd.Weekday() == time.Monday {
			out = append(out, ChainedBucket{
				JobType: JobTypeWeekly,
				Start:   localEnd.AddDate(0, 0, -7),
				End:     localEnd,
			})
		}
		if localEnd.Day() == 1 {
			out = append(out, ChainedBucket{
				JobType: JobTypeMonthly,
				Start:   localEnd.AddDate(0, -1, 0),
				End:     localEnd,
			})
		}
		return out
	default:
		return nil
	}
}

func isAlignedBoundary(t time.Time) bool {
	return t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0
}
