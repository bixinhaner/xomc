package rawcleanup

import (
	"math"
	"time"
)

type Decision struct {
	Rate   float64
	Paused bool
	Reason string
}

func TargetRate(predicted, observed, headroom, minRate, maxRate float64) float64 {
	rate := math.Ceil(math.Max(predicted, observed) * headroom)
	return math.Min(maxRate, math.Max(minRate, rate))
}

func BatchInterval(batch int, rate float64) time.Duration {
	if batch <= 0 || rate <= 0 {
		return time.Minute
	}
	return time.Duration(float64(time.Second) * float64(batch) / rate)
}

func ApplyPressure(rate float64, p Pressure, now time.Time) Decision {
	if p.NATSBacklog {
		return Decision{Paused: true, Reason: "nats_backlog"}
	}
	if p.DiskAwaitMillis > 40 || p.DiskQueue > 2 {
		return Decision{Paused: true, Reason: "disk_overload"}
	}
	reason := ""
	if !p.MonitoringAvailable {
		rate = math.Min(rate, 20)
		reason = "monitoring_unavailable"
	}
	if p.NATSUnavailable {
		rate = math.Min(rate, 20)
		reason = "nats_unavailable"
	}
	if p.CPUPercent > 80 {
		rate *= .5
		reason = "cpu"
	}
	if p.DiskAwaitMillis > 20 || p.DiskQueue > 1 {
		rate *= .5
		reason = "disk"
	}
	if p.SlowDelete {
		rate *= .5
		reason = "minio_slow"
	}
	seconds := now.Minute()*60 + now.Second()
	distance := seconds % (15 * 60)
	if distance <= 2*60 || distance >= 13*60 {
		rate *= .25
		reason = "pm_boundary"
	}
	return Decision{Rate: rate, Reason: reason}
}
