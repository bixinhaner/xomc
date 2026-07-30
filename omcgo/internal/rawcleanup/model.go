package rawcleanup

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type LockLease interface {
	Valid(context.Context) error
	Release(context.Context)
}

type Kind string

const (
	KindPM Kind = "pm"
	KindMR Kind = "mr"
)

type Candidate struct {
	ID          uuid.UUID
	Kind        Kind
	ObjectPath  string
	CollectTime time.Time
	Attempts    int
}

type DeleteResult struct {
	Candidate Candidate
	Err       error
}

type Repository interface {
	TryLock(context.Context) (LockLease, bool, error)
	ListCandidates(context.Context, time.Time, int) ([]Candidate, error)
	MarkResults(context.Context, []DeleteResult, time.Time) error
	RecentCreatedCount(context.Context, time.Time) (int64, error)
	OldestExpired(context.Context, time.Time, Kind) (time.Time, error)
	CleanupMetadata(context.Context, time.Time, int) (int64, error)
}

type ObjectDeleter interface {
	Delete(context.Context, []Candidate) []DeleteResult
}

type Pressure struct {
	MonitoringAvailable bool
	CPUPercent          float64
	DiskAwaitMillis     float64
	DiskQueue           float64
	NATSBacklog         bool
	NATSUnavailable     bool
	SlowDelete          bool
}

type PressureProbe interface {
	Sample(context.Context) Pressure
}
