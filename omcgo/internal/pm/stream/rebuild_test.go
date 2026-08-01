package stream

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/prometheus/client_golang/prometheus"
)

func TestRebuildLeaseBatchRenewalUsesDatabaseClockAndRejectsExpired(t *testing.T) {
	owner := uuid.New()
	jobs := []RebuildJob{
		{ID: 11, LeaseOwner: owner},
		{ID: 12, LeaseOwner: owner},
	}

	query, args, err := rebuildLeaseBatchUpdate(
		jobs, rebuildLeaseDuration,
	).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "lease_expires_at = now() +") {
		t.Fatalf("lease renewal does not use database clock: %s", query)
	}
	if !strings.Contains(query, "lease_expires_at > now()") {
		t.Fatalf("expired rebuild lease can be revived: %s", query)
	}
	if strings.Contains(fmt.Sprint(args), time.Now().UTC().Format("2006-01-02")) {
		t.Fatalf("lease renewal unexpectedly carries a worker wall-clock timestamp: %v", args)
	}
	for _, job := range jobs {
		if !strings.Contains(fmt.Sprint(args), fmt.Sprint(job.ID)) {
			t.Fatalf("batch renewal args lack job %d: %v", job.ID, args)
		}
	}
}

func TestRebuildLeaseBatchFailureCancelsWorkImmediately(t *testing.T) {
	workCtx, cancelWork := context.WithCancel(context.Background())
	defer cancelWork()
	ticks := make(chan time.Time, 1)
	wantErr := errors.New("lease ownership lost")
	done := make(chan error, 1)
	go func() {
		done <- maintainRebuildLeases(
			workCtx,
			ticks,
			func(context.Context) error { return wantErr },
			cancelWork,
		)
	}()

	ticks <- time.Now()

	select {
	case <-workCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("batch work was not canceled after lease renewal failed")
	}
	if err := <-done; !errors.Is(err, wantErr) {
		t.Fatalf("heartbeat error = %v, want %v", err, wantErr)
	}
}

func TestRebuildLeaseHeartbeatIgnoresConcurrentShutdownCancellation(t *testing.T) {
	workCtx, cancelWork := context.WithCancel(context.Background())
	ticks := make(chan time.Time, 1)
	ticks <- time.Now()

	err := maintainRebuildLeases(
		workCtx,
		ticks,
		func(context.Context) error {
			cancelWork()
			return context.Canceled
		},
		cancelWork,
	)

	if err != nil {
		t.Fatalf("normal batch shutdown became a renewal failure: %v", err)
	}
}

func TestRebuildLeaseSetKeepsTailAliveDuringHundredSlowCompletions(t *testing.T) {
	const (
		jobCount = 100
		leaseTTL = 100 * time.Millisecond
	)
	owner := uuid.New()
	jobs := make([]RebuildJob, jobCount)
	deadlines := make(map[int64]time.Time, jobCount)
	now := time.Now()
	for index := range jobs {
		jobs[index] = RebuildJob{ID: int64(index + 1), LeaseOwner: owner}
		deadlines[jobs[index].ID] = now.Add(leaseTTL)
	}
	active := newRebuildActiveLeases(jobs)
	var deadlineMu sync.Mutex
	var renewCalls atomic.Int64
	workCtx, cancelWork := context.WithCancel(context.Background())
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	done := make(chan error, 1)
	go func() {
		done <- maintainRebuildLeases(
			workCtx,
			ticker.C,
			func(ctx context.Context) error {
				return active.renew(ctx, func(_ context.Context, leased []RebuildJob) error {
					renewCalls.Add(1)
					deadlineMu.Lock()
					defer deadlineMu.Unlock()
					next := time.Now().Add(leaseTTL)
					for _, job := range leased {
						deadlines[job.ID] = next
					}
					return nil
				})
			},
			cancelWork,
		)
	}()

	tailID := jobs[len(jobs)-1].ID
	for _, job := range jobs {
		// Production holds this guard until SELECT ... FOR UPDATE has fenced
		// the completing row, then removes only that row from future renewals.
		release := active.beginCompletion(job.ID)
		release()
		time.Sleep(4 * time.Millisecond)

		deadlineMu.Lock()
		jobDeadline := deadlines[job.ID]
		tailDeadline := deadlines[tailID]
		deadlineMu.Unlock()
		if !jobDeadline.After(time.Now()) {
			t.Fatalf("job %d lease expired during slow completion", job.ID)
		}
		if job.ID != tailID && !tailDeadline.After(time.Now()) {
			t.Fatalf("tail job lease expired while %d earlier jobs completed", job.ID)
		}
	}
	cancelWork()
	if err := <-done; err != nil {
		t.Fatalf("lease heartbeat failed: %v", err)
	}
	if renewCalls.Load() < 20 {
		t.Fatalf("renewals stopped before completion tail: %d", renewCalls.Load())
	}
}

type fakeRebuildCompletionRow struct {
	generation int64
	err        error
}

func (row fakeRebuildCompletionRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	*(dest[0].(*int64)) = row.generation
	return nil
}

type fakeRebuildCompletionTx struct {
	generation  int64
	failParent  int
	parentExecs int
	queries     []string
	args        [][]any
	committed   bool
	rolledBack  bool
}

func (tx *fakeRebuildCompletionTx) QueryRow(
	_ context.Context,
	query string,
	args ...any,
) pgx.Row {
	tx.queries = append(tx.queries, query)
	tx.args = append(tx.args, args)
	return fakeRebuildCompletionRow{generation: tx.generation}
}

func (tx *fakeRebuildCompletionTx) Exec(
	_ context.Context,
	query string,
	args ...any,
) (pgconn.CommandTag, error) {
	tx.queries = append(tx.queries, query)
	tx.args = append(tx.args, args)
	if strings.Contains(query, "INSERT INTO pm_aggregation_rebuilds") {
		tx.parentExecs++
		if tx.failParent > 0 && tx.parentExecs == tx.failParent {
			return pgconn.CommandTag{}, errors.New("injected parent cascade failure")
		}
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (tx *fakeRebuildCompletionTx) Commit(context.Context) error {
	tx.committed = true
	return nil
}

func (tx *fakeRebuildCompletionTx) Rollback(context.Context) error {
	tx.rolledBack = true
	return nil
}

func TestRebuildCompletionAndAllParentCascadesRollbackTogether(t *testing.T) {
	job := RebuildJob{
		ID: 91, LeaseOwner: uuid.New(), RequestGeneration: 7,
		Key: WindowKey{
			TaskID: uuid.New(), TaskVersionID: uuid.New(),
			EntityKey: "Network", Granularity: GranularityDaily,
			Start: time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		},
	}
	first := &fakeRebuildCompletionTx{generation: 7, failParent: 2}

	stable, err := completeRebuildAtomically(
		context.Background(),
		func(context.Context) (rebuildCompletionTx, error) { return first, nil },
		job,
		nil,
		nil,
	)

	if err == nil || stable {
		t.Fatalf("partial parent cascade = (stable=%v, err=%v), want rollback error", stable, err)
	}
	if first.committed || !first.rolledBack {
		t.Fatalf("partial cascade transaction commit=%v rollback=%v",
			first.committed, first.rolledBack)
	}

	retry := &fakeRebuildCompletionTx{generation: 7}
	stable, err = completeRebuildAtomically(
		context.Background(),
		func(context.Context) (rebuildCompletionTx, error) { return retry, nil },
		job,
		nil,
		nil,
	)
	if err != nil || !stable || !retry.committed {
		t.Fatalf("retry = (stable=%v, committed=%v, err=%v), want atomic success",
			stable, retry.committed, err)
	}
	if retry.parentExecs != 2 {
		t.Fatalf("retry parent cascades = %d, want weekly and monthly once", retry.parentExecs)
	}
	for _, query := range retry.queries {
		if strings.Contains(query, "INSERT INTO pm_aggregation_rebuilds") &&
			!strings.Contains(query, "source_event_id IS DISTINCT FROM") {
			t.Fatalf("parent upsert is not generation-idempotent: %s", query)
		}
	}
	joinedArgs := fmt.Sprint(retry.args)
	if !strings.Contains(joinedArgs, "cascade:91:7:weekly") ||
		!strings.Contains(joinedArgs, "cascade:91:7:monthly") {
		t.Fatalf("parent source IDs are not generation-idempotent: %s", joinedArgs)
	}
}

func TestRebuildCompletionDefersParentsWhenGenerationChanged(t *testing.T) {
	job := RebuildJob{
		ID: 92, LeaseOwner: uuid.New(), RequestGeneration: 7,
		Key: WindowKey{
			TaskID: uuid.New(), TaskVersionID: uuid.New(),
			EntityKey: "Network", Granularity: GranularityHourly,
			Start: time.Date(2026, 7, 30, 13, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 7, 30, 14, 0, 0, 0, time.UTC),
		},
	}
	tx := &fakeRebuildCompletionTx{generation: 8}

	stable, err := completeRebuildAtomically(
		context.Background(),
		func(context.Context) (rebuildCompletionTx, error) { return tx, nil },
		job,
		nil,
		nil,
	)

	if err != nil || stable {
		t.Fatalf("changed generation = (stable=%v, err=%v), want pending", stable, err)
	}
	if !tx.committed || tx.parentExecs != 0 {
		t.Fatalf("changed generation commit=%v parent cascades=%d, want pending without parent",
			tx.committed, tx.parentExecs)
	}
}

func TestRebuildClaimBatchCoalescesUntilDatabaseQuietPeriod(t *testing.T) {
	query, args, err := rebuildClaimBatchSelect(2*time.Minute, 100).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "requested_at <= now() - (") ||
		!strings.Contains(query, "interval '1 microsecond'") {
		t.Fatalf("claim query does not enforce a database-clock quiet period: %s", query)
	}
	if !strings.Contains(query, "FOR UPDATE SKIP LOCKED") {
		t.Fatalf("claim query is not safe for concurrent rebuilders: %s", query)
	}
	if !strings.Contains(fmt.Sprint(args), "120000000") {
		t.Fatalf("claim query does not carry the two-minute quiet period: %v", args)
	}
}

func TestHourlyRebuildQuietPeriodYieldsBeforePublicationDeadline(t *testing.T) {
	query, args, err := rebuildClaimBatchSelect(2*time.Minute, 8).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "granularity =") ||
		!strings.Contains(query, "window_end +") {
		t.Fatalf("claim query is not publication-deadline aware: %s", query)
	}
	if !strings.Contains(fmt.Sprint(args), "690000000") {
		t.Fatalf("claim query does not reserve 30 seconds before the 12-minute deadline: %v", args)
	}
}

func TestRebuildCoalesceGenerationRunsAtMostOneFollowUp(t *testing.T) {
	if got := rebuildCompletionStatus(100, 100); got != "completed" {
		t.Fatalf("stable generation status = %q, want completed", got)
	}
	if got := rebuildCompletionStatus(100, 101); got != "pending" {
		t.Fatalf("one late generation status = %q, want pending", got)
	}
	if got := rebuildCompletionStatus(100, 200); got != "pending" {
		t.Fatalf("many late generations status = %q, want one pending follow-up", got)
	}
}

func TestLateEventMarksPreparedWindowDirtyBeforePublication(t *testing.T) {
	key := WindowKey{
		TaskVersionID: uuid.New(), EntityKey: "SN-dirty", Granularity: GranularityHourly,
		Start: time.Date(2026, 8, 1, 14, 0, 0, 0, time.UTC),
	}

	query, _, err := rebuildMarkRequestedUpdate(key).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "CASE WHEN status = 'prepared' THEN 'rebuilding' ELSE status END") {
		t.Fatalf("prepared window can publish before its late event is replayed: %s", query)
	}
}

func TestRebuildCoalesceDefersParentUntilChildGenerationStable(t *testing.T) {
	if rebuildGenerationStable(RebuildJob{RequestGeneration: 7}, 8) {
		t.Fatal("parent rebuild must not be enqueued while the child generation changed")
	}
	if !rebuildGenerationStable(RebuildJob{RequestGeneration: 8}, 8) {
		t.Fatal("stable child generation must allow exactly one parent enqueue")
	}
}

func TestRebuildCoalesceGroupsSameSourceVersionsAndPeriod(t *testing.T) {
	deviceRollupVersionID := uuid.New()
	ruleVersionID := uuid.New()
	start := time.Date(2026, 7, 30, 13, 0, 0, 0, time.UTC)
	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{
		{
			TaskID: uuid.New(), VersionID: deviceRollupVersionID,
			Technology: "LTE", DeviceRollup: true,
		},
		{
			TaskID: uuid.New(), VersionID: ruleVersionID,
			Technology: "LTE", Enabled: true,
		},
	})
	jobs := make([]RebuildJob, 100)
	for index := range jobs {
		jobs[index].Key = WindowKey{
			TaskVersionID: ruleVersionID,
			EntityKey:     fmt.Sprintf("group-%03d", index),
			Granularity:   GranularityHourly,
			Start:         start,
			End:           start.Add(time.Hour),
		}
	}

	groups, ungrouped := groupRollupRebuildJobs(jobs, snapshot)

	if len(ungrouped) != 0 {
		t.Fatalf("rollup rebuilds left ungrouped: %d", len(ungrouped))
	}
	if len(groups) != 1 {
		t.Fatalf("shared source-period groups = %d, want 1", len(groups))
	}
	if len(groups[0].Jobs) != 100 {
		t.Fatalf("jobs in shared scan = %d, want 100", len(groups[0].Jobs))
	}
	if len(groups[0].SourceVersionIDs) != 1 ||
		groups[0].SourceVersionIDs[0] != deviceRollupVersionID {
		t.Fatalf("source versions = %v, want [%s]",
			groups[0].SourceVersionIDs, deviceRollupVersionID)
	}
}

func TestRebuildCoalesceMetricsAreRegistered(t *testing.T) {
	registry := prometheus.NewPedanticRegistry()
	metrics := NewMetrics(registry)
	metrics.RebuildBatchesTotal.Inc()
	metrics.RebuildJobsPerBatch.Observe(2)
	metrics.RebuildSnapshotRowsTotal.Add(3)
	metrics.RebuildSnapshotScanSeconds.Observe(0.01)
	metrics.RebuildCoalescedTotal.Add(4)

	families, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[string]bool, len(families))
	for _, family := range families {
		got[family.GetName()] = true
	}
	for _, name := range []string{
		"omc_pm_aggregation_rebuild_batches_total",
		"omc_pm_aggregation_rebuild_jobs_per_batch",
		"omc_pm_aggregation_rebuild_snapshot_rows_total",
		"omc_pm_aggregation_rebuild_snapshot_scan_seconds",
		"omc_pm_aggregation_rebuild_coalesced_total",
	} {
		if !got[name] {
			t.Errorf("metric %s is not registered", name)
		}
	}
}

func TestRebuildParentLineageUsesStableDeviceRollupVersion(t *testing.T) {
	taskID := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	hourlyVersionID := uuid.MustParse("20000000-0000-4000-8000-000000000001")
	rollupVersionID := uuid.MustParse("30000000-0000-4000-8000-000000000001")
	snapshot := &TaskSnapshot{ByVersion: map[uuid.UUID]*TaskVersionSnapshot{
		hourlyVersionID: {
			TaskID: taskID, VersionID: hourlyVersionID,
			DevicePipeline: true, RollupVersionID: rollupVersionID,
		},
	}}
	key := WindowKey{
		TaskID: taskID, TaskVersionID: hourlyVersionID,
		Granularity: GranularityHourly,
	}

	gotTaskID, gotVersionID := rebuildParentLineage(key, snapshot)

	if gotTaskID != taskID || gotVersionID != rollupVersionID {
		t.Fatalf(
			"device parent lineage = (%s, %s), want (%s, %s)",
			gotTaskID, gotVersionID, taskID, rollupVersionID,
		)
	}
}

func TestRebuildParentLineageKeepsOrdinaryTaskVersion(t *testing.T) {
	taskID, versionID := uuid.New(), uuid.New()
	snapshot := &TaskSnapshot{ByVersion: map[uuid.UUID]*TaskVersionSnapshot{
		versionID: {TaskID: taskID, VersionID: versionID},
	}}

	gotTaskID, gotVersionID := rebuildParentLineage(WindowKey{
		TaskID: taskID, TaskVersionID: versionID,
	}, snapshot)

	if gotTaskID != taskID || gotVersionID != versionID {
		t.Fatalf("ordinary lineage changed to (%s, %s)", gotTaskID, gotVersionID)
	}
}

func TestPublishedParentsSelectRestrictsTaskAndVersionLineage(t *testing.T) {
	taskID := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	versionID := uuid.MustParse("20000000-0000-4000-8000-000000000001")
	job := RebuildJob{Key: WindowKey{
		EntityKey: "Network", Granularity: GranularityHourly,
		Start: time.Date(2026, 7, 29, 13, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 7, 29, 14, 0, 0, 0, time.UTC),
	}}

	query, args, err := publishedParentsSelect(
		job, GranularityDaily, taskID, versionID, "cascade:1:daily",
	).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "task_id = ") ||
		!strings.Contains(query, "task_version_id = ") {
		t.Fatalf("parent select lacks lineage predicates: %s", query)
	}
	joinedArgs := fmt.Sprint(args)
	if !strings.Contains(joinedArgs, taskID.String()) ||
		!strings.Contains(joinedArgs, versionID.String()) {
		t.Fatalf("parent select args lack lineage IDs: %v", args)
	}
}

func TestRebuildCascadeTargets(t *testing.T) {
	start := time.Date(2026, 7, 29, 13, 0, 0, 0, time.UTC)
	hour := WindowKey{
		TaskID: uuid.New(), TaskVersionID: uuid.New(),
		EntityKey: "device-1", Granularity: GranularityHourly,
		Start: start, End: start.Add(time.Hour),
	}
	got := RebuildCascadeTargets(hour)
	want := []Granularity{GranularityDaily}
	if !equalGranularities(got, want) {
		t.Fatalf("hour cascade = %v, want %v", got, want)
	}

	day := hour
	day.Granularity = GranularityDaily
	got = RebuildCascadeTargets(day)
	want = []Granularity{GranularityWeekly, GranularityMonthly}
	if !equalGranularities(got, want) {
		t.Fatalf("day cascade = %v, want %v", got, want)
	}
}

func TestRebuildSourceRoutingUsesRawOnlyForDevicePipelineHours(t *testing.T) {
	pipelineID, ruleID := uuid.New(), uuid.New()
	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{
		{TaskID: uuid.New(), VersionID: pipelineID, DevicePipeline: true},
		{TaskID: uuid.New(), VersionID: ruleID},
	})

	if !rebuildUsesRawSources(WindowKey{
		TaskVersionID: pipelineID, Granularity: GranularityHourly,
	}, snapshot) {
		t.Fatal("device-pipeline hour must use normalized raw replay")
	}
	if rebuildUsesRawSources(WindowKey{
		TaskVersionID: ruleID, EntityKey: "Network",
		Granularity: GranularityHourly,
	}, snapshot) {
		t.Fatal("ordinary rule hour must use stable device-hour rollups")
	}
	if rebuildUsesRawSources(WindowKey{
		TaskVersionID: pipelineID, Granularity: GranularityDaily,
	}, snapshot) {
		t.Fatal("daily rebuild must use compact hourly rollups")
	}
}

func TestDeviceRollupVersionsForRuleFiltersTechnology(t *testing.T) {
	lteID, gsmID, ordinaryID := uuid.New(), uuid.New(), uuid.New()
	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{
		{
			TaskID: uuid.New(), VersionID: lteID,
			Technology: "LTE", DeviceRollup: true,
		},
		{
			TaskID: uuid.New(), VersionID: gsmID,
			Technology: "gsm", DeviceRollup: true,
		},
		{TaskID: uuid.New(), VersionID: ordinaryID, Technology: "lte"},
	})

	got := deviceRollupVersionsForRule(snapshot, &TaskVersionSnapshot{
		Technology: "lte",
	})

	if len(got) != 1 || got[0] != lteID {
		t.Fatalf("LTE replay versions = %v, want [%s]", got, lteID)
	}
}

func TestRebuildRetryDelayBacksOffAndCaps(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{8, 256 * time.Second},
		{20, 5 * time.Minute},
	}
	for _, test := range tests {
		if got := rebuildRetryDelay(test.attempt); got != test.want {
			t.Fatalf("rebuildRetryDelay(%d) = %s, want %s", test.attempt, got, test.want)
		}
	}
}

func TestRebuildCompletedAtExpressionUsesTimestampType(t *testing.T) {
	query, _, err := storage.Psql.Update("pm_aggregation_rebuilds").
		Set("completed_at", rebuildCompletedAtExpr(3, time.Now().UTC())).
		ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "NULL::timestamptz") ||
		!strings.Contains(query, "::timestamptz") {
		t.Fatalf("completed_at expression lacks timestamp casts: %s", query)
	}
}

func equalGranularities(got, want []Granularity) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
