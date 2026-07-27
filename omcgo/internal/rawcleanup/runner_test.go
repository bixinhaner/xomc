package rawcleanup

import (
	"context"
	"testing"
	"time"
)

type runnerRepo struct {
	candidates []Candidate
	marked     []DeleteResult
	lock       bool
	metadata   int
	cutoff     time.Time
	leaseErr   error
}

type runnerLease struct{ err error }

func (l *runnerLease) Valid(context.Context) error { return l.err }
func (l *runnerLease) Release(context.Context)     {}

func (r *runnerRepo) TryLock(context.Context) (LockLease, bool, error) {
	return &runnerLease{err: r.leaseErr}, r.lock, nil
}
func (r *runnerRepo) ListCandidates(_ context.Context, cutoff time.Time, _ int) ([]Candidate, error) {
	r.cutoff = cutoff
	return r.candidates, nil
}
func (r *runnerRepo) MarkResults(_ context.Context, got []DeleteResult, _ time.Time) error {
	r.marked = append(r.marked, got...)
	return nil
}
func (r *runnerRepo) RecentCreatedCount(context.Context, time.Time) (int64, error) { return 36000, nil }
func (r *runnerRepo) OldestExpired(context.Context, time.Time, Kind) (time.Time, error) {
	return time.Time{}, nil
}
func (r *runnerRepo) CleanupMetadata(context.Context, time.Time, int) (int64, error) {
	r.metadata++
	return 0, nil
}

type runnerDeleter struct{ calls int }

func (d *runnerDeleter) Delete(_ context.Context, in []Candidate) []DeleteResult {
	d.calls++
	out := make([]DeleteResult, len(in))
	for i := range in {
		out[i].Candidate = in[i]
	}
	return out
}

func TestRunOnceShadowNeverDeletesOrMarks(t *testing.T) {
	repo := &runnerRepo{lock: true, candidates: []Candidate{{Kind: KindPM}}}
	deleter := &runnerDeleter{}
	runner := NewRunner(repo, deleter, Config{Mode: "shadow", BatchSize: 100, RetentionDays: 60})
	if _, err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if deleter.calls != 0 || len(repo.marked) != 0 || repo.metadata != 0 {
		t.Fatalf("shadow mode mutated data: calls=%d marked=%d metadata=%d", deleter.calls, len(repo.marked), repo.metadata)
	}
}

func TestRunOnceRequiresLockAndMarksExactResults(t *testing.T) {
	repo := &runnerRepo{lock: true, candidates: []Candidate{{Kind: KindPM}, {Kind: KindMR}}}
	deleter := &runnerDeleter{}
	runner := NewRunner(repo, deleter, Config{Mode: "fallback", BatchSize: 100, RetentionDays: 60})
	got, err := runner.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != 2 || deleter.calls != 1 || len(repo.marked) != 2 || repo.metadata != 1 {
		t.Fatalf("got=%d calls=%d marked=%d metadata=%d", got, deleter.calls, len(repo.marked), repo.metadata)
	}

	repo.lock = false
	if _, err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if deleter.calls != 1 {
		t.Fatalf("non-lock holder deleted objects")
	}
}

func TestRunOnceUsesLatestValidRetentionAndKeepsLastKnownOnFailure(t *testing.T) {
	repo := &runnerRepo{lock: true}
	days := 90
	runner := NewRunner(repo, &runnerDeleter{}, Config{
		Mode: "shadow", RetentionDays: 60,
		RetentionDaysLookup: func(context.Context) int { return days },
	})
	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	runner.now = func() time.Time { return now }
	if _, err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if want := now.AddDate(0, 0, -90); !repo.cutoff.Equal(want) {
		t.Fatalf("cutoff=%v want %v", repo.cutoff, want)
	}
	days = 0
	if _, err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if want := now.AddDate(0, 0, -90); !repo.cutoff.Equal(want) {
		t.Fatalf("invalid lookup replaced last known cutoff: %v", repo.cutoff)
	}
}

func TestRunOnceDoesNotDeleteAfterLockSessionLoss(t *testing.T) {
	repo := &runnerRepo{lock: true, leaseErr: context.Canceled, candidates: []Candidate{{Kind: KindPM}}}
	deleter := &runnerDeleter{}
	runner := NewRunner(repo, deleter, Config{Mode: "fallback"})
	if _, err := runner.RunOnce(context.Background()); err == nil {
		t.Fatal("lost lock must fail the round")
	}
	if deleter.calls != 0 || len(repo.marked) != 0 {
		t.Fatalf("lost lock mutated objects or status")
	}
}
