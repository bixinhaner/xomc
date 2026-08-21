package components

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	persistentQueueStatusPending    = "pending"
	persistentQueueStatusSent       = "sent"
	persistentQueueStatusRunning    = "running"
	persistentQueueStatusSucceeded  = "succeeded"
	persistentQueueStatusFailed     = "failed"
	persistentQueueStatusDeadLetter = "dead_letter"

	persistentQueueResultSucceeded = "succeeded"
	persistentQueueResultFailed    = "failed"
)

var persistentQueueNames = []string{
	"device_tasks",
	"async_jobs",
	"parameter_sync_outbox",
	"northbound_outbox",
	"pm_kpi_export",
	"trace_export",
	"backup_tasks",
	"dead_letters",
}

var persistentQueueStatuses = []string{
	persistentQueueStatusPending,
	persistentQueueStatusSent,
	persistentQueueStatusRunning,
	persistentQueueStatusSucceeded,
	persistentQueueStatusFailed,
	persistentQueueStatusDeadLetter,
}

// PersistentQueueMetrics is the low-cardinality Prometheus contract for
// PostgreSQL-backed queues. Queue names and statuses are fixed constants; no
// device identity, row ID, object path, or job payload is emitted as a label.
type PersistentQueueMetrics struct {
	Pending                 *prometheus.GaugeVec
	OldestAgeSeconds        *prometheus.GaugeVec
	OverdueOldestAgeSeconds *prometheus.GaugeVec
	FailedTotal             *prometheus.GaugeVec
	DeadLetterTotal         *prometheus.GaugeVec
	ProcessedTotal          *prometheus.GaugeVec
	ObserverFailuresTotal   *prometheus.CounterVec
	Up                      *prometheus.GaugeVec
	SampleTimestamp         *prometheus.GaugeVec
}

// NewPersistentQueueMetrics registers and primes the persistent queue metric
// families. Priming ensures an empty queue is exported as zero immediately.
func NewPersistentQueueMetrics(reg prometheus.Registerer) *PersistentQueueMetrics {
	m := &PersistentQueueMetrics{
		Pending: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_persistent_queue_pending",
			Help: "Current PostgreSQL-backed queue entries by bounded queue and status.",
		}, []string{"queue", "status"}),
		OldestAgeSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_persistent_queue_oldest_age_seconds",
			Help: "Age in seconds of the oldest PostgreSQL-backed queue entry by bounded queue and status.",
		}, []string{"queue", "status"}),
		OverdueOldestAgeSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_persistent_queue_overdue_oldest_age_seconds",
			Help: "Seconds past the row-specific expiry of the most overdue PostgreSQL-backed queue entry by bounded queue and status.",
		}, []string{"queue", "status"}),
		FailedTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_persistent_queue_failed_total",
			Help: "Current failed PostgreSQL-backed queue entries by bounded queue.",
		}, []string{"queue"}),
		DeadLetterTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_persistent_queue_dead_letter_total",
			Help: "Current dead-letter PostgreSQL-backed queue entries by bounded queue.",
		}, []string{"queue"}),
		ProcessedTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_persistent_queue_processed_total",
			Help: "Current terminal PostgreSQL-backed queue entries by bounded queue and result; high-volume success states use PostgreSQL planner estimates.",
		}, []string{"queue", "result"}),
		ObserverFailuresTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_persistent_queue_observer_failures_total",
			Help: "PostgreSQL-backed queue observation failures by bounded queue.",
		}, []string{"queue"}),
		Up: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_persistent_queue_up",
			Help: "Whether the latest PostgreSQL-backed queue observation succeeded.",
		}, []string{"queue"}),
		SampleTimestamp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_persistent_queue_sample_timestamp_seconds",
			Help: "Unix timestamp of the latest successful PostgreSQL-backed queue observation.",
		}, []string{"queue"}),
	}
	for _, queue := range persistentQueueNames {
		for _, status := range persistentQueueStatuses {
			m.Pending.WithLabelValues(queue, status).Set(0)
			m.OldestAgeSeconds.WithLabelValues(queue, status).Set(0)
			m.OverdueOldestAgeSeconds.WithLabelValues(queue, status).Set(0)
		}
		m.FailedTotal.WithLabelValues(queue).Set(0)
		m.DeadLetterTotal.WithLabelValues(queue).Set(0)
		m.ProcessedTotal.WithLabelValues(queue, persistentQueueResultSucceeded).Set(0)
		m.ProcessedTotal.WithLabelValues(queue, persistentQueueResultFailed).Set(0)
		m.Up.WithLabelValues(queue).Set(0)
		m.SampleTimestamp.WithLabelValues(queue).Set(0)
	}
	if reg != nil {
		reg.MustRegister(
			m.Pending, m.OldestAgeSeconds, m.OverdueOldestAgeSeconds, m.FailedTotal, m.DeadLetterTotal,
			m.ProcessedTotal, m.ObserverFailuresTotal, m.Up, m.SampleTimestamp,
		)
	}
	return m
}

type persistentQueueQuery struct {
	name  string
	query string
}

// All statements are fixed at compile time. This avoids treating table names
// or statuses as SQL input and keeps the observer safe from identifier injection.
var persistentQueueQueries = []persistentQueueQuery{
	{name: "device_tasks", query: `
SELECT status, count, oldest_age_seconds, overdue_oldest_age_seconds
FROM (
WITH live_status(status) AS (
    VALUES ('pending'), ('sent'), ('failed')
),
status_estimates AS (
    SELECT v.value AS status, COALESCE(SUM(c.reltuples * f.freq), 0)::bigint AS estimate
    FROM pg_inherits i
    JOIN pg_class p ON p.oid = i.inhparent
    JOIN pg_class c ON c.oid = i.inhrelid
    JOIN pg_namespace n ON n.oid = c.relnamespace
    JOIN pg_stats s ON s.schemaname = n.nspname AND s.tablename = c.relname AND s.attname = 'status'
    CROSS JOIN LATERAL jsonb_array_elements_text(to_jsonb(s.most_common_vals)) WITH ORDINALITY v(value, ord)
    JOIN LATERAL unnest(s.most_common_freqs) WITH ORDINALITY f(freq, ord) USING (ord)
    WHERE p.oid = 'device_tasks'::regclass
    GROUP BY v.value
),
live_counts AS (
    SELECT s.status,
           CASE
             WHEN bounded.count <= 1000 THEN bounded.count
             ELSE GREATEST(COALESCE(e.estimate, 0), bounded.count)
           END::bigint AS count
    FROM live_status s
    LEFT JOIN status_estimates e ON e.status = s.status
    CROSS JOIN LATERAL (
        SELECT COUNT(*)::bigint AS count
        FROM (
            SELECT 1 FROM device_tasks WHERE status = s.status LIMIT 1001
        ) bounded_rows
    ) bounded
),
live_oldest AS (
    SELECT s.status,
           COALESCE(EXTRACT(EPOCH FROM (now() - oldest.created_at)), 0)::double precision AS oldest_age_seconds
    FROM live_status s
    LEFT JOIN LATERAL (
        SELECT created_at
        FROM device_tasks
        WHERE status = s.status
        ORDER BY created_at ASC
        LIMIT 1
    ) oldest ON true
),
live_overdue AS (
    SELECT s.status,
           COALESCE(EXTRACT(EPOCH FROM (now() - overdue.expires_at)), 0)::double precision AS overdue_oldest_age_seconds
    FROM live_status s
    LEFT JOIN LATERAL (
        SELECT expires_at
        FROM device_tasks
        WHERE status = s.status AND expires_at < now()
        ORDER BY expires_at ASC
        LIMIT 1
    ) overdue ON true
)
SELECT c.status, c.count, o.oldest_age_seconds, od.overdue_oldest_age_seconds
FROM live_counts c
JOIN live_oldest o USING (status)
JOIN live_overdue od USING (status)
WHERE c.count > 0
-- Historical terminal states dominate this partitioned table. Exact COUNT/MIN/MAX
-- walks millions of index entries every 30 seconds. PostgreSQL already maintains
-- a status histogram and row estimate for each child, so use those bounded-cost
-- planner statistics for terminal gauges.
UNION ALL SELECT status_estimates.status, status_estimates.estimate, 0::double precision, 0::double precision
FROM status_estimates
WHERE status_estimates.estimate > 0
  AND (status_estimates.status IN ('completed', 'expired', 'cancelled')
       OR status_estimates.status NOT IN ('pending', 'sent', 'completed', 'failed', 'expired', 'cancelled'))
) bounded_device_tasks`},
	{name: "async_jobs", query: `SELECT status, COUNT(*)::bigint, COALESCE(EXTRACT(EPOCH FROM (now() - MIN(created_at))), 0)::double precision, 0::double precision FROM async_jobs GROUP BY status`},
	{name: "parameter_sync_outbox", query: `
SELECT status, COUNT(*)::bigint, COALESCE(EXTRACT(EPOCH FROM (now() - MIN(created_at))), 0)::double precision, 0::double precision FROM parameter_sync_outbox WHERE status = 'pending' GROUP BY status
UNION ALL SELECT status, COUNT(*)::bigint, COALESCE(EXTRACT(EPOCH FROM (now() - MIN(created_at))), 0)::double precision, 0::double precision FROM parameter_sync_outbox WHERE status = 'failed' GROUP BY status
UNION ALL SELECT status, COUNT(*)::bigint, COALESCE(EXTRACT(EPOCH FROM (now() - MIN(created_at))), 0)::double precision, 0::double precision FROM parameter_sync_outbox WHERE status = 'delivering' GROUP BY status
-- delivered is likewise historical and dominates this table.  The terminal
-- partial index estimate avoids a multi-million-row index walk; subtract the
-- exact (normally tiny) dead count because the index covers both states.
UNION ALL SELECT 'delivered', GREATEST(c.reltuples::bigint - (SELECT COUNT(*) FROM parameter_sync_outbox WHERE status = 'dead'), 0), 0::double precision, 0::double precision
FROM pg_class c WHERE c.oid = 'idx_parameter_sync_outbox_terminal_status'::regclass
UNION ALL SELECT status, COUNT(*)::bigint, COALESCE(EXTRACT(EPOCH FROM (now() - MIN(created_at))), 0)::double precision, 0::double precision FROM parameter_sync_outbox WHERE status = 'dead' GROUP BY status`},
	{name: "northbound_outbox", query: `SELECT status, COUNT(*)::bigint, COALESCE(EXTRACT(EPOCH FROM (now() - MIN(created_at))), 0)::double precision, 0::double precision FROM northbound_outbox GROUP BY status`},
	{name: "pm_kpi_export", query: `SELECT status, COUNT(*)::bigint, COALESCE(EXTRACT(EPOCH FROM (now() - MIN(created_at))), 0)::double precision, 0::double precision FROM pm_kpi_export_tasks GROUP BY status`},
	{name: "trace_export", query: `SELECT status, COUNT(*)::bigint, COALESCE(EXTRACT(EPOCH FROM (now() - MIN(created_at))), 0)::double precision, 0::double precision FROM trace_export_jobs GROUP BY status`},
	{name: "backup_tasks", query: `SELECT status, COUNT(*)::bigint, COALESCE(EXTRACT(EPOCH FROM (now() - MIN(created_at))), 0)::double precision, 0::double precision FROM backup_tasks GROUP BY status`},
	{name: "dead_letters", query: `SELECT 'dead_letter', COUNT(*)::bigint, COALESCE(EXTRACT(EPOCH FROM (now() - MIN(created_at))), 0)::double precision, 0::double precision FROM dead_letters`},
}

// PersistentQueueObserver periodically samples fixed PostgreSQL queue tables.
// A failed table query preserves its previous business gauges and marks only
// the table unavailable, so a database outage cannot look like an empty queue.
type PersistentQueueObserver struct {
	pool     *pgxpool.Pool
	metrics  *PersistentQueueMetrics
	interval time.Duration
	logger   *zap.Logger

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func NewPersistentQueueObserver(pool *pgxpool.Pool, metrics *PersistentQueueMetrics, interval time.Duration, logger *zap.Logger) *PersistentQueueObserver {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &PersistentQueueObserver{pool: pool, metrics: metrics, interval: interval, logger: logger.Named("persistent-queue-observer")}
}

func (o *PersistentQueueObserver) Start(ctx context.Context) {
	if o == nil || o.pool == nil || o.metrics == nil {
		return
	}
	o.mu.Lock()
	if o.cancel != nil {
		o.mu.Unlock()
		return
	}
	workerCtx, cancel := context.WithCancel(ctx)
	o.cancel = cancel
	o.done = make(chan struct{})
	done := o.done
	o.mu.Unlock()
	go func() {
		defer close(done)
		o.Collect(workerCtx)
		ticker := time.NewTicker(o.interval)
		defer ticker.Stop()
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				o.Collect(workerCtx)
			}
		}
	}()
}

func (o *PersistentQueueObserver) Stop() {
	if o == nil {
		return
	}
	o.mu.Lock()
	cancel, done := o.cancel, o.done
	o.cancel, o.done = nil, nil
	o.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	if done != nil {
		<-done
	}
}

// Collect samples all fixed tables once. It is exported for smoke tests and
// for operators that need an immediate sample after a database reconnect.
func (o *PersistentQueueObserver) Collect(ctx context.Context) {
	if o == nil || o.pool == nil || o.metrics == nil {
		return
	}
	for _, descriptor := range persistentQueueQueries {
		o.collectQueue(ctx, descriptor)
	}
}

type persistentQueueSnapshot struct {
	pending          map[string]float64
	oldestAge        map[string]float64
	overdueOldestAge map[string]float64
	failed           float64
	deadLetter       float64
	succeeded        float64
}

func (o *PersistentQueueObserver) collectQueue(ctx context.Context, descriptor persistentQueueQuery) {
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := o.pool.Query(queryCtx, descriptor.query)
	if err == nil {
		defer rows.Close()
	}
	snapshot := persistentQueueSnapshot{
		pending:          make(map[string]float64),
		oldestAge:        make(map[string]float64),
		overdueOldestAge: make(map[string]float64),
	}
	if err == nil {
		for rows.Next() {
			var rawStatus string
			var count int64
			var oldestAge float64
			var overdueOldestAge float64
			if scanErr := rows.Scan(&rawStatus, &count, &oldestAge, &overdueOldestAge); scanErr != nil {
				err = fmt.Errorf("scan %s queue row: %w", descriptor.name, scanErr)
				break
			}
			status, normalizeErr := normalizePersistentQueueStatus(rawStatus)
			if normalizeErr != nil {
				err = fmt.Errorf("normalize %s queue status: %w", descriptor.name, normalizeErr)
				break
			}
			snapshot.pending[status] += float64(count)
			if oldestAge > snapshot.oldestAge[status] {
				snapshot.oldestAge[status] = oldestAge
			}
			if overdueOldestAge > snapshot.overdueOldestAge[status] {
				snapshot.overdueOldestAge[status] = overdueOldestAge
			}
		}
		if rowsErr := rows.Err(); err == nil && rowsErr != nil {
			err = fmt.Errorf("iterate %s queue rows: %w", descriptor.name, rowsErr)
		}
	}
	if err != nil {
		o.metrics.ObserverFailuresTotal.WithLabelValues(descriptor.name).Inc()
		o.metrics.Up.WithLabelValues(descriptor.name).Set(0)
		o.logger.Warn("persistent queue observation failed", zap.String("queue", descriptor.name), zap.Error(err))
		return
	}

	for _, status := range persistentQueueStatuses {
		o.metrics.Pending.WithLabelValues(descriptor.name, status).Set(snapshot.pending[status])
		o.metrics.OldestAgeSeconds.WithLabelValues(descriptor.name, status).Set(snapshot.oldestAge[status])
		o.metrics.OverdueOldestAgeSeconds.WithLabelValues(descriptor.name, status).Set(snapshot.overdueOldestAge[status])
	}
	snapshot.failed = snapshot.pending[persistentQueueStatusFailed]
	snapshot.deadLetter = snapshot.pending[persistentQueueStatusDeadLetter]
	snapshot.succeeded = snapshot.pending[persistentQueueStatusSucceeded]
	o.metrics.FailedTotal.WithLabelValues(descriptor.name).Set(snapshot.failed)
	o.metrics.DeadLetterTotal.WithLabelValues(descriptor.name).Set(snapshot.deadLetter)
	o.metrics.ProcessedTotal.WithLabelValues(descriptor.name, persistentQueueResultSucceeded).Set(snapshot.succeeded)
	o.metrics.ProcessedTotal.WithLabelValues(descriptor.name, persistentQueueResultFailed).Set(snapshot.failed + snapshot.deadLetter)
	o.metrics.Up.WithLabelValues(descriptor.name).Set(1)
	o.metrics.SampleTimestamp.WithLabelValues(descriptor.name).Set(float64(time.Now().Unix()))
}

func normalizePersistentQueueStatus(raw string) (string, error) {
	switch raw {
	case "pending", "queued":
		return persistentQueueStatusPending, nil
	case "sent":
		return persistentQueueStatusSent, nil
	case "running", "processing", "delivering":
		return persistentQueueStatusRunning, nil
	case "succeeded", "completed", "delivered", "done":
		return persistentQueueStatusSucceeded, nil
	case "failed", "canceled", "cancelled", "expired":
		return persistentQueueStatusFailed, nil
	case "dead", "zombie", "dead_letter":
		return persistentQueueStatusDeadLetter, nil
	default:
		return "", fmt.Errorf("unsupported status %q", raw)
	}
}
