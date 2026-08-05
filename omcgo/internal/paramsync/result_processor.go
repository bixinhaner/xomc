package paramsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
)

type ResultProcessOutcome struct {
	Duplicate bool
	Finalized bool
	Failed    bool
}

// ErrTaskResultNotDurable means the terminal event reached APP before the
// device_tasks PostgreSQL projection became terminal. It is intentionally
// retryable: returning it rolls back the result insert and makes the NATS
// consumer redeliver instead of converting replication lag into run failure.
var ErrTaskResultNotDurable = errors.New("parameter sync task result is not durable yet")

type ResultProcessor interface {
	Process(ctx context.Context, result event.ParamSyncTaskResultPayload) (ResultProcessOutcome, error)
}

type PGResultProcessor struct {
	pool    *pgxpool.Pool
	metrics *Metrics
	now     func() time.Time
}

func NewPGResultProcessor(pool *pgxpool.Pool) *PGResultProcessor {
	return &PGResultProcessor{pool: pool, now: func() time.Time { return time.Now().UTC() }}
}

func (p *PGResultProcessor) WithMetrics(metrics *Metrics) *PGResultProcessor {
	p.metrics = metrics
	return p
}

func (p *PGResultProcessor) Process(ctx context.Context, result event.ParamSyncTaskResultPayload) (ResultProcessOutcome, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return ResultProcessOutcome{}, fmt.Errorf("begin parameter sync result: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	duplicate, err := resultEventExists(ctx, tx, result)
	if err != nil {
		return ResultProcessOutcome{}, err
	}
	if duplicate {
		if err := tx.Commit(ctx); err != nil {
			return ResultProcessOutcome{}, fmt.Errorf("commit duplicate parameter sync result: %w", err)
		}
		if p.metrics != nil {
			p.metrics.ResultRedelivery.Inc()
		}
		return ResultProcessOutcome{Duplicate: true}, nil
	}

	run, err := loadRunForUpdate(ctx, tx, result.RunID)
	if err != nil {
		return ResultProcessOutcome{}, err
	}
	if run.RequestID != result.RequestID || run.DeviceSN != result.DeviceSN || !run.Status.AcceptsTaskResults() {
		return p.failUnsafeResult(ctx, tx, run, result, "task/run/request/device association mismatch")
	}

	inserted, err := insertResultEvent(ctx, tx, result, p.now())
	if err != nil {
		return ResultProcessOutcome{}, err
	}
	if !inserted {
		if err := tx.Commit(ctx); err != nil {
			return ResultProcessOutcome{}, fmt.Errorf("commit duplicate parameter sync result: %w", err)
		}
		if p.metrics != nil {
			p.metrics.ResultRedelivery.Inc()
		}
		return ResultProcessOutcome{Duplicate: true}, nil
	}

	rawValues, taskStatus, taskMetadata, validationErr := loadTaskValues(ctx, tx, result)
	if errors.Is(validationErr, ErrTaskResultNotDurable) {
		return ResultProcessOutcome{}, validationErr
	}
	failed := !result.Success || validationErr != nil || run.Status == RunStatusCancelling
	errorMessage := result.ErrorMessage
	if validationErr != nil {
		errorMessage = validationErr.Error()
	}
	if run.Status == RunStatusCancelling && errorMessage == "" {
		errorMessage = run.ErrorMessage
	}
	if !failed {
		var recoveredValues []projectedValue
		if taskMetadata.Recovered {
			if standardPath, learnable := learnableRecovered9005StandardPath(taskMetadata, run.Coverage); learnable {
				if err := recordRecoveredReadUnsupportedPath(ctx, tx, run, standardPath, result.DeviceSN, p.now()); err != nil {
					return ResultProcessOutcome{}, err
				}
			}
			if value, ok := projectRecoveredMLNDCRFUnknown(taskMetadata, run.Coverage); ok {
				isMLNDC, err := isMLNDCDevice(ctx, tx, run.DeviceID)
				if err != nil {
					return ResultProcessOutcome{}, err
				}
				if isMLNDC {
					recoveredValues = append(recoveredValues, value)
				}
			}
			if err := markRecoveredCoverageIncomplete(ctx, tx, run, taskMetadata); err != nil {
				return ResultProcessOutcome{}, err
			}
		}
		values := projectTaskValues(rawValues, run.Coverage)
		values, err = filterMLNDCRawRFShadowValues(
			ctx, tx, run.DeviceID, values, run.Coverage,
		)
		if err != nil {
			return ResultProcessOutcome{}, err
		}
		values = append(values, recoveredValues...)
		if run.SyncScope.IsFull() {
			if err := writeStagingValues(ctx, tx, run, result.TaskID, values, p.now()); err != nil {
				return ResultProcessOutcome{}, err
			}
		} else if err := upsertOfficialValues(ctx, tx, run.DeviceID, values, p.now()); err != nil {
			return ResultProcessOutcome{}, err
		}
	}

	if failed || taskStatus != task.TaskStatusCompleted {
		failed = true
	}
	processedStatus := "processed"
	if failed {
		processedStatus = "failed"
	}
	resultUpdate, resultArgs, err := storage.Psql.Update("parameter_sync_task_results").
		Set("status", processedStatus).Set("processed_at", p.now()).Set("error_message", errorMessage).
		Where(sq.Eq{"run_id": run.ID, "task_id": result.TaskID}).ToSql()
	if err != nil {
		return ResultProcessOutcome{}, fmt.Errorf("build mark parameter sync result processed: %w", err)
	}
	if _, err := tx.Exec(ctx, resultUpdate, resultArgs...); err != nil {
		return ResultProcessOutcome{}, fmt.Errorf("mark parameter sync result processed: %w", err)
	}

	if failed && run.Status != RunStatusCancelling {
		if err := beginCancellingRun(ctx, tx, run, errorMessage, p.now()); err != nil {
			return ResultProcessOutcome{}, err
		}
		if err := cancelUnsentRunTasks(ctx, tx, run, p.now()); err != nil {
			return ResultProcessOutcome{}, err
		}
	}

	authoritative, err := reconcileRunCountsForInsertedResult(
		ctx, tx, run, failed,
		func(ctx context.Context, tx pgx.Tx, run SyncRun) (authoritativeRunCounts, error) {
			return loadAuthoritativeRunCounts(ctx, tx, run.ID)
		},
	)
	if err != nil {
		return ResultProcessOutcome{}, err
	}
	if p.metrics != nil {
		mode := "incremental"
		if authoritative {
			mode = "authoritative"
		}
		p.metrics.ResultCountMode.WithLabelValues(mode).Inc()
	}
	progressStatus := RunStatusExecuting
	if run.Status == RunStatusCancelling {
		progressStatus = RunStatusCancelling
	}
	convergence, err := convergeLoadedRunTx(ctx, tx, run, progressStatus, true, p.now())
	if err != nil {
		return ResultProcessOutcome{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ResultProcessOutcome{}, fmt.Errorf("commit parameter sync result: %w", err)
	}
	if p.metrics != nil && convergence.Finalized {
		result := "succeeded"
		if convergence.Failed {
			result = "failed"
		}
		p.metrics.FinalizeTotal.WithLabelValues(result).Inc()
		duration := p.now().Sub(run.StartedAt).Seconds()
		p.metrics.RunDuration.Observe(duration)
		p.metrics.RequestDuration.Observe(duration)
	}
	return ResultProcessOutcome{
		Finalized: convergence.Finalized,
		Failed:    convergence.Failed,
	}, nil
}

func resultEventExists(
	ctx context.Context,
	tx pgx.Tx,
	result event.ParamSyncTaskResultPayload,
) (bool, error) {
	var exists bool
	if err := tx.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM parameter_sync_task_results WHERE run_id=$1 AND task_id=$2
)`, result.RunID, result.TaskID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check duplicate parameter sync result: %w", err)
	}
	return exists, nil
}

func markRunProcessing(ctx context.Context, tx pgx.Tx, run *SyncRun) error {
	if run.Status == RunStatusProcessing {
		return nil
	}
	if err := ValidateRunTransition(run.Status, RunStatusProcessing); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE parameter_sync_runs SET status='processing', version=version+1 WHERE id=$1`, run.ID); err != nil {
		return fmt.Errorf("mark parameter sync run processing: %w", err)
	}
	run.Status = RunStatusProcessing
	return nil
}

func updateRunProgress(ctx context.Context, tx pgx.Tx, run *SyncRun, status RunStatus) error {
	query, args, err := storage.Psql.Update("parameter_sync_runs").
		Set("status", status).Set("expected_task_count", run.ExpectedTaskCount).
		Set("terminal_task_count", run.TerminalTaskCount).
		Set("processed_task_count", run.ProcessedTaskCount).Set("failed_task_count", run.FailedTaskCount).
		Set("error_message", run.ErrorMessage).Set("version", sq.Expr("version + 1")).
		Where(sq.Eq{"id": run.ID}).ToSql()
	if err != nil {
		return fmt.Errorf("build update parameter sync progress: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update parameter sync progress: %w", err)
	}
	return nil
}

type authoritativeRunCounts struct {
	expected  int
	terminal  int
	processed int
	failed    int
}

type authoritativeRunCountLoader func(context.Context, pgx.Tx, SyncRun) (authoritativeRunCounts, error)

// reconcileRunCountsForInsertedResult advances counters exactly once for the
// result row that was just inserted. The run row is locked by Process, while
// duplicate result events return before this function. A full durable-state
// count is therefore needed only at a failure/terminal boundary or to repair
// invalid persisted counters, instead of once per task result.
func reconcileRunCountsForInsertedResult(
	ctx context.Context,
	tx pgx.Tx,
	run *SyncRun,
	failed bool,
	load authoritativeRunCountLoader,
) (bool, error) {
	run.TerminalTaskCount++
	run.ProcessedTaskCount++
	if failed {
		run.FailedTaskCount++
	}

	invalid := run.ExpectedTaskCount < 0 ||
		run.TerminalTaskCount < 0 || run.ProcessedTaskCount < 0 || run.FailedTaskCount < 0 ||
		run.TerminalTaskCount > run.ExpectedTaskCount ||
		run.ProcessedTaskCount > run.TerminalTaskCount ||
		run.FailedTaskCount > run.ProcessedTaskCount
	if !failed && !invalid && !run.ReadyToFinalize() {
		return false, nil
	}

	counts, err := load(ctx, tx, *run)
	if err != nil {
		return false, err
	}
	applyAuthoritativeRunCounts(run, counts)
	return true, nil
}

// loadAuthoritativeRunCounts derives counters from durable task/result state.
// The caller uses it only at failure/final boundaries or to repair invalid
// counters; duplicate events are rejected before incremental counting.
func loadAuthoritativeRunCounts(ctx context.Context, tx pgx.Tx, runID uuid.UUID) (authoritativeRunCounts, error) {
	const query = `
SELECT
  count(t.id)::int,
  count(t.id) FILTER (WHERE t.status IN ('completed','failed','expired','cancelled'))::int,
  count(res.task_id) FILTER (
    WHERE t.status IN ('completed','failed','expired','cancelled')
      AND res.status IN ('processed','failed')
  )::int,
  count(res.task_id) FILTER (
    WHERE t.status IN ('completed','failed','expired','cancelled')
      AND res.status IN ('processed','failed')
      AND (res.status='failed' OR NOT res.success)
  )::int
FROM device_tasks t
LEFT JOIN parameter_sync_task_results res ON res.run_id=$1 AND res.task_id=t.id
WHERE t.source='param_sync' AND t.source_id=$1`
	var counts authoritativeRunCounts
	if err := tx.QueryRow(ctx, query, runID).Scan(
		&counts.expected,
		&counts.terminal,
		&counts.processed,
		&counts.failed,
	); err != nil {
		return authoritativeRunCounts{}, fmt.Errorf("derive parameter sync run counts: %w", err)
	}
	return counts, nil
}

func applyAuthoritativeRunCounts(run *SyncRun, counts authoritativeRunCounts) {
	// ExpectedTaskCount records the dispatch plan cardinality. Durable task rows
	// can repair it upward after a crash between task creation and run update,
	// but must never shrink it: a partial dispatch is an incomplete plan, not a
	// smaller successful run.
	if counts.expected > run.ExpectedTaskCount {
		run.ExpectedTaskCount = counts.expected
	}
	run.TerminalTaskCount = counts.terminal
	run.ProcessedTaskCount = counts.processed
	run.FailedTaskCount = counts.failed
}

func loadRunForUpdate(ctx context.Context, tx pgx.Tx, runID uuid.UUID) (*SyncRun, error) {
	query, args, err := storage.Psql.Select(runColumns()...).From("parameter_sync_runs").
		Where(sq.Eq{"id": runID}).Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build load parameter sync run for result: %w", err)
	}
	run, err := scanRun(tx.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, fmt.Errorf("load parameter sync run for result: %w", err)
	}
	return run, nil
}

func insertResultEvent(ctx context.Context, tx pgx.Tx, result event.ParamSyncTaskResultPayload, now time.Time) (bool, error) {
	query, args, err := storage.Psql.Insert("parameter_sync_task_results").
		Columns("run_id", "task_id", "event_id", "success", "result_ref", "status", "error_code", "error_message", "created_at").
		Values(result.RunID, result.TaskID, result.EventID, result.Success, result.ResultRef, "received", result.ErrorCode, result.ErrorMessage, now).
		Suffix("ON CONFLICT DO NOTHING").ToSql()
	if err != nil {
		return false, fmt.Errorf("build record parameter sync result: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("record parameter sync result: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

type storedTaskResult struct {
	StandardParameterValues []tr069.ParameterValueStruct `json:"standard_parameter_values"`
	PrivateParameterValues  []tr069.ParameterValueStruct `json:"private_parameter_values"`
	Recovered               bool                         `json:"recovered"`
	BadPath                 string                       `json:"bad_path"`
	BadPaths                []string                     `json:"bad_paths"`
	FaultCode               int                          `json:"fault_code"`
	RequestedNames          []string                     `json:"-"`
}

func loadTaskValues(ctx context.Context, tx pgx.Tx, result event.ParamSyncTaskResultPayload) ([]tr069.ParameterValueStruct, task.TaskStatus, storedTaskResult, error) {
	query, args, err := storage.Psql.Select("result", "status", "source", "source_id", "creator_id", "params").
		From("device_tasks").Where(sq.Eq{"id": result.TaskID, "device_sn": result.DeviceSN}).ToSql()
	if err != nil {
		return nil, "", storedTaskResult{}, fmt.Errorf("build load parameter sync task result: %w", err)
	}
	var raw, rawParams []byte
	var status task.TaskStatus
	var source task.TaskSource
	var sourceID, creatorID string
	if err := tx.QueryRow(ctx, query, args...).Scan(&raw, &status, &source, &sourceID, &creatorID, &rawParams); err != nil {
		return nil, status, storedTaskResult{}, fmt.Errorf("load parameter sync task result: %w", err)
	}
	if source != task.TaskSourceParamSync || sourceID != result.RunID.String() || creatorID != result.RequestID.String() {
		return nil, status, storedTaskResult{}, fmt.Errorf("task source association mismatch")
	}
	if err := validateDurableTaskState(status, result.Success); err != nil {
		return nil, status, storedTaskResult{}, err
	}
	if !result.Success {
		return nil, status, storedTaskResult{}, nil
	}
	var stored storedTaskResult
	if err := json.Unmarshal(raw, &stored); err != nil {
		return nil, status, storedTaskResult{}, fmt.Errorf("decode durable parameter sync task result: %w", err)
	}
	var taskParams struct {
		Names []string `json:"names"`
	}
	if len(rawParams) > 0 {
		if err := json.Unmarshal(rawParams, &taskParams); err != nil {
			return nil, status, storedTaskResult{}, fmt.Errorf("decode durable parameter sync task params: %w", err)
		}
		stored.RequestedNames = taskParams.Names
	}
	if len(stored.PrivateParameterValues) > 0 {
		return stored.PrivateParameterValues, status, stored, nil
	}
	// Rolling-upgrade compatibility for tasks completed by an older ACS.
	return stored.StandardParameterValues, status, stored, nil
}

// learnableRecovered9005StandardPath deliberately accepts only an exact leaf
// that appeared in the GPV request. A concrete child returned for an object
// prefix may merely mean that the current instance is absent, so persisting it
// as a product capability would be unsafe.
func learnableRecovered9005StandardPath(stored storedTaskResult, coverage []CoverageScope) (string, bool) {
	badPath := strings.TrimSpace(stored.BadPath)
	if !stored.Recovered || stored.FaultCode != 9005 || badPath == "" || strings.HasSuffix(badPath, ".") {
		return "", false
	}
	exactRequest := false
	for _, requested := range stored.RequestedNames {
		if strings.TrimSpace(requested) == badPath {
			exactRequest = true
			break
		}
	}
	if !exactRequest {
		return "", false
	}
	mappings := make([]parammodel.ParamMapping, 0)
	for _, scope := range coverage {
		for _, frozen := range scope.Mappings {
			mappings = append(mappings, parammodel.ParamMapping{
				StandardPath: frozen.StandardPath, PrivatePath: frozen.PrivatePath,
				IsStorable: frozen.IsStorable, IsSupported: true,
			})
		}
	}
	translated := parammodel.NewTranslator(&parammodel.MappingSet{Mappings: mappings}, nil, nil).ToStandard(badPath)
	if !translated.Found || translated.Mapping == nil || !translated.Mapping.IsStorable {
		return "", false
	}
	standardPath := strings.TrimSpace(translated.Translated)
	return standardPath, standardPath != "" && !strings.HasSuffix(standardPath, ".")
}

func recordRecoveredReadUnsupportedPath(
	ctx context.Context, tx pgx.Tx, run *SyncRun, standardPath, deviceSN string, now time.Time,
) error {
	if run == nil || run.DeviceID == uuid.Nil || strings.TrimSpace(standardPath) == "" {
		return nil
	}
	const query = `
INSERT INTO product_unsupported_paths (
  product_id, firmware_version, standard_path, read_unsupported, write_unsupported,
  last_fault_code, last_device_sn, first_seen_at, last_seen_at, created_at, updated_at
)
SELECT product_id, COALESCE(NULLIF($2, ''), firmware_version, ''), $3, true, true,
       9005, $4, $5, $5, $5, $5
FROM devices
WHERE id=$1 AND product_id IS NOT NULL AND deleted_at IS NULL
ON CONFLICT (product_id, firmware_version, standard_path) DO UPDATE SET
  read_unsupported=true,
  write_unsupported=true,
  last_fault_code=9005,
  last_device_sn=EXCLUDED.last_device_sn,
  hit_count=product_unsupported_paths.hit_count + 1,
  last_seen_at=EXCLUDED.last_seen_at,
  updated_at=EXCLUDED.updated_at`
	if _, err := tx.Exec(ctx, query, run.DeviceID, run.MappingVersion, standardPath, deviceSN, now); err != nil {
		return fmt.Errorf("record recovered firmware read-unsupported path: %w", err)
	}
	return nil
}

func markRecoveredCoverageIncomplete(ctx context.Context, tx pgx.Tx, run *SyncRun, stored storedTaskResult) error {
	indexes := recoveredIncompleteCoverageIndexes(run.Coverage, stored)
	if len(indexes) == 0 {
		return nil
	}
	changed := false
	for _, i := range indexes {
		if i >= 0 && i < len(run.Coverage) && run.Coverage[i].Complete {
			run.Coverage[i].Complete = false
			changed = true
		}
	}
	if !changed {
		return nil
	}
	encoded, err := json.Marshal(run.Coverage)
	if err != nil {
		return fmt.Errorf("encode incomplete parameter sync coverage: %w", err)
	}
	query, args, err := storage.Psql.Update("parameter_sync_runs").
		Set("coverage", encoded).Set("version", sq.Expr("version + 1")).
		Where(sq.Eq{"id": run.ID}).ToSql()
	if err != nil {
		return fmt.Errorf("build mark parameter sync coverage incomplete: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("mark parameter sync coverage incomplete: %w", err)
	}
	return nil
}

// recoveredIncompleteCoverageIndexes resolves device-private fault paths only
// against the immutable mappings captured by the run. Returning indexes keeps
// the path interpretation separate from persistence and prevents a registry
// reload from changing the meaning of an in-flight result.
func recoveredIncompleteCoverageIndexes(coverage []CoverageScope, stored storedTaskResult) []int {
	paths := append([]string(nil), stored.BadPaths...)
	if len(paths) == 0 && strings.TrimSpace(stored.BadPath) != "" {
		paths = append(paths, stored.BadPath)
	}
	matched := make(map[int]struct{})
	for _, path := range paths {
		markFrozenCoverageMatches(coverage, path, matched)
	}
	if len(matched) == 0 {
		for _, requested := range stored.RequestedNames {
			markFrozenCoverageMatches(coverage, requested, matched)
		}
	}
	if len(matched) == 0 && stored.Recovered && stored.FaultCode == 9005 && len(paths) > 0 {
		for i := range coverage {
			matched[i] = struct{}{}
		}
	}
	indexes := make([]int, 0, len(matched))
	for i := range coverage {
		if _, ok := matched[i]; ok {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func markFrozenCoverageMatches(coverage []CoverageScope, path string, matched map[int]struct{}) {
	normalized := normalizeRuntimePath(strings.TrimSpace(path))
	if normalized == "" {
		return
	}
	for i, scope := range coverage {
		for _, mapping := range scope.Mappings {
			if frozenPathsRelated(normalized, mapping.PrivatePath) ||
				frozenPathsRelated(normalized, mapping.StandardPath) {
				matched[i] = struct{}{}
				break
			}
		}
	}
}

func frozenPathsRelated(left, right string) bool {
	left = normalizeRuntimePath(strings.TrimSpace(left))
	right = normalizeRuntimePath(strings.TrimSpace(right))
	if left == "" || right == "" {
		return false
	}
	return left == right ||
		(strings.HasSuffix(left, ".") && strings.HasPrefix(right, left)) ||
		(strings.HasSuffix(right, ".") && strings.HasPrefix(left, right))
}

func validateDurableTaskState(status task.TaskStatus, success bool) error {
	if success {
		switch status {
		case task.TaskStatusPending, task.TaskStatusSent:
			return fmt.Errorf("%w: success event observed while task is %s", ErrTaskResultNotDurable, status)
		case task.TaskStatusCompleted:
			return nil
		default:
			return fmt.Errorf("successful parameter sync event contradicts terminal task status %s", status)
		}
	}
	switch status {
	case task.TaskStatusPending, task.TaskStatusSent:
		return fmt.Errorf("%w: failure event observed while task is %s", ErrTaskResultNotDurable, status)
	case task.TaskStatusFailed, task.TaskStatusExpired, task.TaskStatusCancelled:
		return nil
	default:
		return fmt.Errorf("failed parameter sync event contradicts terminal task status %s", status)
	}
}

type projectedValue struct {
	ParameterPath string
	PrivatePath   string
	Value         string
	ParameterType string
	Writable      bool
	FAPInstance   int
	ParamGroup    string
}

func projectTaskValues(values []tr069.ParameterValueStruct, coverage []CoverageScope) []projectedValue {
	mappings := make([]parammodel.ParamMapping, 0)
	for _, scope := range coverage {
		for _, frozen := range scope.Mappings {
			mappings = append(mappings, parammodel.ParamMapping{
				StandardPath: frozen.StandardPath, PrivatePath: frozen.PrivatePath,
				Access: frozen.Access, DataType: frozen.DataType,
				IsStorable: frozen.IsStorable, IsSupported: true,
			})
		}
	}
	translator := parammodel.NewTranslator(&parammodel.MappingSet{Mappings: mappings}, nil, nil)
	projected := make([]projectedValue, 0, len(values))
	for _, value := range values {
		if value.Name == "" {
			continue
		}
		path := value.Name
		writable := false
		if translated := translator.ToStandard(value.Name); translated.Found {
			if !translated.Mapping.IsStorable {
				continue
			}
			path = translated.Translated
			writable = parammodel.IsAccessWritable(translated.Mapping.Access)
		}
		projected = append(projected, projectedValue{
			ParameterPath: path, PrivatePath: value.Name, Value: value.Value,
			ParameterType: inferProjectedParameterType(value.Type), Writable: writable,
			FAPInstance: device.ExtractFAPInstance(path), ParamGroup: device.ClassifyParamGroup(path),
		})
	}
	return projected
}

func filterMLNDCRawRFShadowValues(
	ctx context.Context,
	tx pgx.Tx,
	deviceID uuid.UUID,
	values []projectedValue,
	coverage []CoverageScope,
) ([]projectedValue, error) {
	if !hasFrozenMLNDCRFStatusMapping(coverage) || !containsRawMLNDCRFShadow(values) {
		return values, nil
	}
	isMLNDC, err := isMLNDCDevice(ctx, tx, deviceID)
	if err != nil {
		return nil, err
	}
	if !isMLNDC {
		return values, nil
	}

	filtered := make([]projectedValue, 0, len(values))
	for _, value := range values {
		if isRawMLNDCRFShadow(value) {
			continue
		}
		filtered = append(filtered, value)
	}
	return filtered, nil
}

func containsRawMLNDCRFShadow(values []projectedValue) bool {
	for _, value := range values {
		if isRawMLNDCRFShadow(value) {
			return true
		}
	}
	return false
}

func isRawMLNDCRFShadow(value projectedValue) bool {
	return value.ParameterPath == value.PrivatePath &&
		normalizeRuntimePath(value.ParameterPath) == mlnDCRFStatusStandardPath
}

func hasFrozenMLNDCRFStatusMapping(coverage []CoverageScope) bool {
	for _, scope := range coverage {
		for _, mapping := range scope.Mappings {
			if mapping.IsStorable &&
				mapping.StandardPath == mlnDCRFStatusStandardPath &&
				mapping.PrivatePath == mlnDCRFStatusPrivatePath {
				return true
			}
		}
	}
	return false
}

func projectRecoveredMLNDCRFUnknown(
	stored storedTaskResult,
	coverage []CoverageScope,
) (projectedValue, bool) {
	if !stored.Recovered || stored.FaultCode != 9005 || len(stored.RequestedNames) != 1 {
		return projectedValue{}, false
	}
	privatePath := strings.TrimSpace(stored.RequestedNames[0])
	if privatePath == "" ||
		normalizeRuntimePath(privatePath) != mlnDCRFStatusPrivatePath {
		return projectedValue{}, false
	}
	if badPath := strings.TrimSpace(stored.BadPath); badPath != "" && badPath != privatePath {
		return projectedValue{}, false
	}

	values := projectTaskValues([]tr069.ParameterValueStruct{{
		Name:  privatePath,
		Value: "unknown",
		Type:  "string",
	}}, coverage)
	if len(values) != 1 ||
		normalizeRuntimePath(values[0].ParameterPath) != mlnDCRFStatusStandardPath {
		return projectedValue{}, false
	}
	return values[0], true
}

func isMLNDCDevice(ctx context.Context, tx pgx.Tx, deviceID uuid.UUID) (bool, error) {
	query, args, err := storage.Psql.
		Select("COALESCE(product_class, '')").
		From("devices").
		Where(sq.Eq{"id": deviceID}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build load MLN/DC product class: %w", err)
	}
	var productClass string
	if err := tx.QueryRow(ctx, query, args...).Scan(&productClass); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("load MLN/DC product class: %w", err)
	}
	return strings.EqualFold(strings.TrimSpace(productClass), mlnDCProductClass), nil
}

func inferProjectedParameterType(soapType string) string {
	switch soapType {
	case "xsd:unsignedInt", "unsignedInt":
		return "unsignedInt"
	case "xsd:int", "int":
		return "int"
	case "xsd:boolean", "boolean":
		return "boolean"
	case "xsd:dateTime", "dateTime":
		return "dateTime"
	default:
		return "string"
	}
}

// paramSyncUpsertBatchSize 是 writeStagingValues/upsertOfficialValues 单条多行 INSERT 携带的最大
// 参数值行数。此前逐值单条 tx.Exec，一次任务结果几十~上百个参数值就是几十~上百次网络往返 +
// 语句 parse/plan，是压测（万级设备并发同步）下 postgres CPU 被打满的主因之一——已有整改方案
// 明确要求（docs/review-report/20260714/PARAM_SYNC_REMEDIATION_PLAN.md §16.10）"正式参数写入
// 使用 pgx.CopyFrom、unnest 或等价批量方式，禁止逐参数单条 upsert"。这里选多行 VALUES 而非
// CopyFrom：批量通常几十到几百行（单task的GetParameterValues响应），COPY 需要额外落临时表才能
// 支持 ON CONFLICT 语义，收益不及实现复杂度；多行 INSERT 已经把 N 次往返收敛成 1 次。500 行 ×
// 8/11 列远低于 PG 65535 参数上限，留足安全边际。
const paramSyncUpsertBatchSize = 500

// deviceParameterUnchangedGuard 是 device_parameters 的 ON CONFLICT DO UPDATE 附加 WHERE
// 条件：只有 value/type/writable/fap_instance/param_group 任一实际变化才真正重写这一行
// （从而才推进 last_updated_at）。压测实测 device_parameters 单表在 postgres 上有 ~13000
// 行更新/秒的持续写入量，绝大多数是设备重复上报同样的值——不加这层判断，即使已经把多个
// 参数值合并成一条多行 INSERT（见 paramSyncUpsertBatchSize），每次同步仍会对每一行做一次
// 实际的 UPDATE（新版本、索引维护、WAL），是批量合并解决不了的成本，只能从"要不要写"这层
// 拦住。代价：`last_updated_at`（前端展示为"最后更新"）语义从"最近一次确认同步"变成
// "最近一次值实际变化"——已确认前端标签本就是"最后更新/Last Updated"，与新语义一致，
// 唯一另一处 lastSynced 展示（QuickSettingsTab）用的是独立本地状态，不读这个字段，不受影响。
const deviceParameterUnchangedGuard = `(device_parameters.parameter_value IS DISTINCT FROM EXCLUDED.parameter_value
    OR device_parameters.parameter_type IS DISTINCT FROM EXCLUDED.parameter_type
    OR device_parameters.writable IS DISTINCT FROM EXCLUDED.writable
    OR device_parameters.fap_instance IS DISTINCT FROM EXCLUDED.fap_instance
    OR device_parameters.param_group IS DISTINCT FROM EXCLUDED.param_group)`

// dedupeProjectedValuesByPath 按 ParameterPath 去重，同路径保留最后一个值——和原来逐条
// tx.Exec 顺序执行、后写覆盖前写的语义一致。同一批多行 INSERT ... ON CONFLICT DO UPDATE
// 不允许在同一条语句里两次命中同一冲突目标（PG 报 "ON CONFLICT DO UPDATE command cannot
// affect row a second time"），而 translator 把不同 private path 映射到同一 StandardPath
// 是已知可能场景（多实例映射未命中时的兜底），必须先去重才能安全批量。
func dedupeProjectedValuesByPath(values []projectedValue) []projectedValue {
	if len(values) < 2 {
		return values
	}
	order := make([]string, 0, len(values))
	byPath := make(map[string]projectedValue, len(values))
	for _, value := range values {
		if _, exists := byPath[value.ParameterPath]; !exists {
			order = append(order, value.ParameterPath)
		}
		byPath[value.ParameterPath] = value
	}
	deduped := make([]projectedValue, 0, len(order))
	for _, path := range order {
		deduped = append(deduped, byPath[path])
	}
	return deduped
}

func writeStagingValues(ctx context.Context, tx pgx.Tx, run *SyncRun, taskID string, values []projectedValue, now time.Time) error {
	values = dedupeProjectedValuesByPath(values)
	for start := 0; start < len(values); start += paramSyncUpsertBatchSize {
		end := min(start+paramSyncUpsertBatchSize, len(values))
		builder := storage.Psql.Insert("parameter_sync_staging_values").
			Columns("run_id", "parameter_path", "private_path", "value", "value_type", "writable", "fap_instance", "param_group", "task_id", "created_at", "updated_at")
		for _, value := range values[start:end] {
			encoded, err := json.Marshal(value.Value)
			if err != nil {
				return fmt.Errorf("encode staging parameter value: %w", err)
			}
			builder = builder.Values(run.ID, value.ParameterPath, value.PrivatePath, encoded, value.ParameterType, value.Writable, value.FAPInstance, value.ParamGroup, taskID, now, now)
		}
		query, args, err := builder.
			Suffix("ON CONFLICT (run_id, parameter_path) DO UPDATE SET private_path = EXCLUDED.private_path, value = EXCLUDED.value, value_type = EXCLUDED.value_type, writable = EXCLUDED.writable, fap_instance = EXCLUDED.fap_instance, param_group = EXCLUDED.param_group, task_id = EXCLUDED.task_id, updated_at = EXCLUDED.updated_at").
			ToSql()
		if err != nil {
			return fmt.Errorf("build stage parameter sync value batch: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("stage parameter sync value batch: %w", err)
		}
	}
	return nil
}

func upsertOfficialValues(ctx context.Context, tx pgx.Tx, deviceID uuid.UUID, values []projectedValue, now time.Time) error {
	values = dedupeProjectedValuesByPath(values)
	if err := device.AcquireParameterWriteLocks(ctx, tx, deviceID); err != nil {
		return err
	}
	for start := 0; start < len(values); start += paramSyncUpsertBatchSize {
		end := min(start+paramSyncUpsertBatchSize, len(values))
		builder := storage.Psql.Insert("device_parameters").
			Columns("device_id", "parameter_path", "parameter_value", "parameter_type", "writable", "last_updated_at", "fap_instance", "param_group")
		for _, value := range values[start:end] {
			builder = builder.Values(deviceID, value.ParameterPath, value.Value, value.ParameterType, value.Writable, now, value.FAPInstance, value.ParamGroup)
		}
		query, args, err := builder.
			Suffix("ON CONFLICT (device_id, parameter_path) DO UPDATE SET parameter_value = EXCLUDED.parameter_value, parameter_type = EXCLUDED.parameter_type, writable = EXCLUDED.writable, last_updated_at = EXCLUDED.last_updated_at, fap_instance = EXCLUDED.fap_instance, param_group = EXCLUDED.param_group WHERE " + deviceParameterUnchangedGuard).
			ToSql()
		if err != nil {
			return fmt.Errorf("build upsert parameter sync value batch: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("upsert parameter sync value batch: %w", err)
		}
	}
	return nil
}

func finalizeSuccessfulRun(ctx context.Context, tx pgx.Tx, run *SyncRun, now time.Time) error {
	if run.SyncScope.IsFull() {
		if err := device.AcquireParameterWriteLocks(ctx, tx, run.DeviceID); err != nil {
			return err
		}
		const mergeSQL = `
INSERT INTO device_parameters (device_id, parameter_path, parameter_value, parameter_type, writable, last_updated_at, fap_instance, param_group)
SELECT $1, parameter_path, value #>> '{}', value_type, writable, $2, fap_instance, param_group
FROM parameter_sync_staging_values WHERE run_id = $3
ON CONFLICT (device_id, parameter_path) DO UPDATE SET
  parameter_value = EXCLUDED.parameter_value,
  parameter_type = EXCLUDED.parameter_type,
  writable = EXCLUDED.writable,
  last_updated_at = EXCLUDED.last_updated_at,
  fap_instance = EXCLUDED.fap_instance,
  param_group = EXCLUDED.param_group
WHERE ` + deviceParameterUnchangedGuard
		if _, err := tx.Exec(ctx, mergeSQL, run.DeviceID, now, run.ID); err != nil {
			return fmt.Errorf("merge full parameter sync staging: %w", err)
		}
		// A tolerated 9005 marks the affected frozen coverage incomplete. Merge
		// values from successful batches, but never delete older values for a
		// coverage point the device did not return completely. All complete
		// coverage is reconciled in one indexed DELETE to avoid one statement and
		// one repeated device-parameter index walk per mapping group.
		query, args, ok, err := buildFullSyncReconcileDelete(run.DeviceID, run.ID, run.Coverage)
		if err != nil {
			return err
		}
		if ok {
			if _, err := tx.Exec(ctx, query, args...); err != nil {
				return fmt.Errorf("reconcile full parameter sync: %w", err)
			}
		}
	}
	deviceUpdate, deviceArgs, err := storage.Psql.Update("devices").Set("last_param_sync_at", now).
		Set("last_param_sync_failed_at", nil).Set("last_param_sync_error", nil).
		Set("updated_at", now).Where(sq.Eq{"id": run.DeviceID}).ToSql()
	if err != nil {
		return fmt.Errorf("build mark parameter sync time: %w", err)
	}
	if _, err := tx.Exec(ctx, deviceUpdate, deviceArgs...); err != nil {
		return fmt.Errorf("mark parameter sync time: %w", err)
	}
	if err := updateTerminalState(ctx, tx, run, RequestStatusSucceeded, RunStatusSucceeded, ResultCodeOK, "", now); err != nil {
		return err
	}
	if err := recordAutomaticSyncOutcome(ctx, tx, run, true, "", now); err != nil {
		return err
	}
	if err := insertRunTerminalOutbox(ctx, tx, run, event.SubjectParamSyncRunCompleted, now); err != nil {
		return err
	}
	cleanup, cleanupArgs, err := storage.Psql.Delete("parameter_sync_staging_values").Where(sq.Eq{"run_id": run.ID}).ToSql()
	if err != nil {
		return fmt.Errorf("build clean parameter sync staging: %w", err)
	}
	if _, err := tx.Exec(ctx, cleanup, cleanupArgs...); err != nil {
		return fmt.Errorf("clean parameter sync staging: %w", err)
	}
	return nil
}

func buildFullSyncReconcileDelete(deviceID, runID uuid.UUID, coverage []CoverageScope) (string, []any, bool, error) {
	completeMappings := make([]FrozenMapping, 0)
	for _, scope := range coverage {
		if !scope.Complete || scope.Path == "" {
			continue
		}
		completeMappings = append(completeMappings, scope.Mappings...)
	}
	completePredicate := frozenCoverageMappingsPredicate(completeMappings)
	if completePredicate == nil {
		return "", nil, false, nil
	}
	query, args, err := storage.Psql.Delete("device_parameters").
		Where(sq.Eq{"device_id": deviceID}).
		Where(completePredicate).
		Where("NOT EXISTS (SELECT 1 FROM parameter_sync_staging_values s WHERE s.run_id = ? AND s.parameter_path = device_parameters.parameter_path)", runID).
		ToSql()
	if err != nil {
		return "", nil, false, fmt.Errorf("build reconcile full parameter sync: %w", err)
	}
	return query, args, true, nil
}

func frozenCoveragePathPredicate(coverage CoverageScope) sq.Sqlizer {
	return frozenCoverageMappingsPredicate(coverage.Mappings)
}

func frozenCoverageMappingsPredicate(mappings []FrozenMapping) sq.Sqlizer {
	predicates := sq.Or{}
	exactPaths := make([]string, 0, len(mappings))
	exactSeen := make(map[string]struct{}, len(mappings))
	runtimePatterns := make([]string, 0, len(mappings))
	runtimeSeen := make(map[string]struct{}, len(mappings))
	for _, mapping := range mappings {
		if !mapping.IsStorable || mapping.StandardPath == "" {
			continue
		}
		if idx := strings.Index(mapping.StandardPath, "{i}"); idx >= 0 {
			pattern := regexp.QuoteMeta(mapping.StandardPath)
			pattern = strings.ReplaceAll(pattern, regexp.QuoteMeta("{i}"), `[0-9]+`)
			if _, exists := runtimeSeen[pattern]; !exists {
				runtimeSeen[pattern] = struct{}{}
				runtimePatterns = append(runtimePatterns, pattern)
			}
		} else {
			if _, exists := exactSeen[mapping.StandardPath]; !exists {
				exactSeen[mapping.StandardPath] = struct{}{}
				exactPaths = append(exactPaths, mapping.StandardPath)
			}
		}
	}
	if len(exactPaths) > 0 {
		predicates = append(predicates, sq.Expr("parameter_path = ANY(?::text[])", exactPaths))
	}
	if len(runtimePatterns) > 0 {
		// device_id is the leading key of the PK, so PostgreSQL first narrows the
		// candidate set to this one device. A compact alternation avoids
		// hundreds of OR branches, parameters and bitmap scans. One anchored
		// alternation also compiles once per candidate row instead of compiling
		// every individual pattern through regex ANY.
		predicates = append(predicates, sq.Expr(
			"parameter_path ~ ?",
			"^("+strings.Join(runtimePatterns, "|")+")$",
		))
	}
	if len(predicates) == 0 {
		return nil
	}
	return predicates
}

func beginCancellingRun(ctx context.Context, tx pgx.Tx, run *SyncRun, message string, now time.Time) error {
	if message == "" {
		message = "parameter sync task failed"
	}
	if err := ValidateRunTransition(run.Status, RunStatusCancelling); err != nil {
		return err
	}
	run.Status = RunStatusCancelling
	run.ErrorMessage = message
	query, args, err := storage.Psql.Update("parameter_sync_runs").Set("status", RunStatusCancelling).
		Set("error_message", message).Set("version", sq.Expr("version + 1")).
		Where(sq.Eq{"id": run.ID}).ToSql()
	if err != nil {
		return fmt.Errorf("build begin parameter sync cancellation: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("begin parameter sync cancellation: %w", err)
	}
	return nil
}

func cancelUnsentRunTasks(ctx context.Context, tx pgx.Tx, run *SyncRun, now time.Time) error {
	const cancelQuery = `
UPDATE device_tasks SET status='cancelled', completed_at=$2,
  error_message='parameter sync run cancelling'
WHERE source='param_sync' AND source_id=$1 AND status='pending'
RETURNING id, to_jsonb(device_tasks)`
	rows, err := tx.Query(ctx, cancelQuery, run.ID, now)
	if err != nil {
		return fmt.Errorf("cancel unsent parameter sync tasks: %w", err)
	}
	type cancelledTask struct {
		id      uuid.UUID
		payload []byte
	}
	var cancelled []cancelledTask
	for rows.Next() {
		var item cancelledTask
		if err := rows.Scan(&item.id, &item.payload); err != nil {
			rows.Close()
			return fmt.Errorf("scan cancelled parameter sync task: %w", err)
		}
		cancelled = append(cancelled, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate cancelled parameter sync tasks: %w", err)
	}
	rows.Close()
	for _, item := range cancelled {
		resultQuery, resultArgs, err := storage.Psql.Insert("parameter_sync_task_results").
			Columns("run_id", "task_id", "event_id", "success", "result_ref", "status", "error_message", "processed_at", "created_at").
			Values(run.ID, item.id, item.id.String()+":cancelled", false, "device_tasks:"+item.id.String(), "failed", run.ErrorMessage, now, now).
			Suffix("ON CONFLICT (run_id, task_id) DO NOTHING").ToSql()
		if err != nil {
			return fmt.Errorf("build cancelled parameter sync result: %w", err)
		}
		if _, err := tx.Exec(ctx, resultQuery, resultArgs...); err != nil {
			return fmt.Errorf("insert cancelled parameter sync result: %w", err)
		}
		outboxQuery, outboxArgs, err := storage.Psql.Insert("parameter_sync_outbox").
			Columns("event_type", "aggregate_type", "aggregate_id", "dedupe_key", "payload", "created_at", "updated_at").
			Values("param_sync.task.cancel", "task", item.id, "task-cancel:"+item.id.String(), item.payload, now, now).
			Suffix("ON CONFLICT (dedupe_key) DO NOTHING").ToSql()
		if err != nil {
			return fmt.Errorf("build parameter sync task cancellation outbox: %w", err)
		}
		if _, err := tx.Exec(ctx, outboxQuery, outboxArgs...); err != nil {
			return fmt.Errorf("insert parameter sync task cancellation outbox: %w", err)
		}
	}
	return nil
}

func finalizeConvergedFailedRun(ctx context.Context, tx pgx.Tx, run *SyncRun, now time.Time) error {
	message := run.ErrorMessage
	if message == "" {
		message = "parameter sync task failed"
	}
	var requestStatus RequestStatus
	if err := tx.QueryRow(ctx, "SELECT status FROM parameter_sync_requests WHERE id=$1", run.RequestID).Scan(&requestStatus); err != nil {
		return fmt.Errorf("load converging parameter sync request: %w", err)
	}
	preserveRequest := requestStatus == RequestStatusCancelled || requestStatus == RequestStatusTimedOut
	terminalStatus := RunStatusFailed
	if requestStatus == RequestStatusCancelled {
		terminalStatus = RunStatusCancelled
	}
	if preserveRequest {
		if err := updateRunTerminalOnly(ctx, tx, run, terminalStatus, message, now); err != nil {
			return err
		}
	} else if err := updateTerminalState(ctx, tx, run, RequestStatusFailed, RunStatusFailed, ResultCodeTaskFailed, message, now); err != nil {
		return err
	}
	if requestStatus != RequestStatusCancelled {
		if err := recordAutomaticSyncOutcome(ctx, tx, run, false, message, now); err != nil {
			return err
		}
	}
	if run.SyncScope.IsFull() && requestStatus != RequestStatusCancelled {
		query, args, err := storage.Psql.Update("devices").Set("last_param_sync_failed_at", now).
			Set("last_param_sync_error", message).Where(sq.Eq{"id": run.DeviceID}).ToSql()
		if err != nil {
			return fmt.Errorf("build mark full parameter sync failed: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("mark full parameter sync failed: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, "DELETE FROM parameter_sync_staging_values WHERE run_id=$1", run.ID); err != nil {
		return fmt.Errorf("clean failed parameter sync staging: %w", err)
	}
	return insertRunTerminalOutbox(ctx, tx, run, event.SubjectParamSyncRunFailed, now)
}

func updateRunTerminalOnly(ctx context.Context, tx pgx.Tx, run *SyncRun, status RunStatus, message string, now time.Time) error {
	if err := ValidateRunTransition(run.Status, status); err != nil {
		return err
	}
	run.Status = status
	query, args, err := storage.Psql.Update("parameter_sync_runs").Set("status", status).
		Set("expected_task_count", run.ExpectedTaskCount).
		Set("terminal_task_count", run.TerminalTaskCount).Set("processed_task_count", run.ProcessedTaskCount).
		Set("failed_task_count", run.FailedTaskCount).Set("error_message", message).
		Set("completed_at", now).Set("version", sq.Expr("version + 1")).Where(sq.Eq{"id": run.ID}).ToSql()
	if err != nil {
		return fmt.Errorf("build finalize converged parameter sync run: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("finalize converged parameter sync run: %w", err)
	}
	return nil
}

func automaticSyncBackoff(consecutiveFailures int) time.Duration {
	switch {
	case consecutiveFailures <= 1:
		return time.Minute
	case consecutiveFailures == 2:
		return 5 * time.Minute
	case consecutiveFailures == 3:
		return 15 * time.Minute
	case consecutiveFailures == 4:
		return time.Hour
	default:
		return 6 * time.Hour
	}
}

func recordAutomaticSyncOutcome(ctx context.Context, tx pgx.Tx, run *SyncRun, success bool, message string, now time.Time) error {
	if !run.TriggerReason.Automatic() {
		return nil
	}
	if success {
		const query = `
INSERT INTO parameter_sync_device_state
  (device_id, consecutive_failures, last_success_at, next_auto_sync_at, last_error, updated_at)
VALUES ($1, 0, $2, $2, NULL, $2)
ON CONFLICT (device_id) DO UPDATE SET consecutive_failures=0,
  last_success_at=EXCLUDED.last_success_at, next_auto_sync_at=EXCLUDED.next_auto_sync_at,
  last_error=NULL, updated_at=EXCLUDED.updated_at`
		if _, err := tx.Exec(ctx, query, run.DeviceID, now); err != nil {
			return fmt.Errorf("record automatic parameter sync success: %w", err)
		}
		return nil
	}
	var failures int
	if err := tx.QueryRow(ctx, `SELECT consecutive_failures FROM parameter_sync_device_state WHERE device_id=$1 FOR UPDATE`, run.DeviceID).Scan(&failures); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("lock automatic parameter sync state: %w", err)
	}
	failures++
	next := now.Add(automaticSyncBackoff(failures))
	const query = `
INSERT INTO parameter_sync_device_state
  (device_id, consecutive_failures, last_failure_at, next_auto_sync_at, last_error, updated_at)
VALUES ($1, $2, $3, $4, $5, $3)
ON CONFLICT (device_id) DO UPDATE SET consecutive_failures=EXCLUDED.consecutive_failures,
  last_failure_at=EXCLUDED.last_failure_at, next_auto_sync_at=EXCLUDED.next_auto_sync_at,
  last_error=EXCLUDED.last_error, updated_at=EXCLUDED.updated_at`
	if _, err := tx.Exec(ctx, query, run.DeviceID, failures, now, next, message); err != nil {
		return fmt.Errorf("record automatic parameter sync failure: %w", err)
	}
	return nil
}

func updateTerminalState(ctx context.Context, tx pgx.Tx, run *SyncRun, requestStatus RequestStatus, runStatus RunStatus, code ResultCode, message string, now time.Time) error {
	if err := ValidateRunTransition(run.Status, runStatus); err != nil {
		return err
	}
	var currentRequestStatus RequestStatus
	if err := tx.QueryRow(ctx, `SELECT status FROM parameter_sync_requests WHERE id=$1`, run.RequestID).Scan(&currentRequestStatus); err != nil {
		return fmt.Errorf("load parameter sync request transition: %w", err)
	}
	if err := ValidateRequestTransition(currentRequestStatus, requestStatus); err != nil {
		return err
	}
	run.Status = runStatus
	runUpdate, runArgs, err := storage.Psql.Update("parameter_sync_runs").Set("status", runStatus).
		Set("expected_task_count", run.ExpectedTaskCount).
		Set("terminal_task_count", run.TerminalTaskCount).Set("processed_task_count", run.ProcessedTaskCount).
		Set("failed_task_count", run.FailedTaskCount).Set("error_message", message).
		Set("completed_at", now).Set("version", sq.Expr("version + 1")).Where(sq.Eq{"id": run.ID}).ToSql()
	if err != nil {
		return fmt.Errorf("build finalize parameter sync run: %w", err)
	}
	if _, err := tx.Exec(ctx, runUpdate, runArgs...); err != nil {
		return fmt.Errorf("finalize parameter sync run: %w", err)
	}
	reqUpdate, reqArgs, err := storage.Psql.Update("parameter_sync_requests").Set("status", requestStatus).
		Set("result_code", code).Set("error_message", message).Set("active_run_id", nil).
		Set("completed_at", now).Set("updated_at", now).Where(sq.Eq{"id": run.RequestID}).ToSql()
	if err != nil {
		return fmt.Errorf("build finalize parameter sync request: %w", err)
	}
	if _, err := tx.Exec(ctx, reqUpdate, reqArgs...); err != nil {
		return fmt.Errorf("finalize parameter sync request: %w", err)
	}
	return nil
}

func insertRunTerminalOutbox(ctx context.Context, tx pgx.Tx, run *SyncRun, subject string, now time.Time) error {
	payload, err := json.Marshal(map[string]any{
		"request_id": run.RequestID, "run_id": run.ID, "device_id": run.DeviceID,
		"status": run.Status, "trigger_reason": run.TriggerReason, "sync_scope": run.SyncScope,
	})
	if err != nil {
		return fmt.Errorf("marshal parameter sync terminal event: %w", err)
	}
	query, args, err := storage.Psql.Insert("parameter_sync_outbox").
		Columns("event_type", "aggregate_type", "aggregate_id", "dedupe_key", "payload", "created_at", "updated_at").
		Values(subject, "run", run.ID, subject+":"+run.ID.String(), payload, now, now).
		Suffix("ON CONFLICT (dedupe_key) DO NOTHING").ToSql()
	if err != nil {
		return fmt.Errorf("build parameter sync terminal outbox: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert parameter sync terminal outbox: %w", err)
	}
	return nil
}

func (p *PGResultProcessor) failUnsafeResult(ctx context.Context, tx pgx.Tx, run *SyncRun, result event.ParamSyncTaskResultPayload, message string) (ResultProcessOutcome, error) {
	inserted, err := insertResultEvent(ctx, tx, result, p.now())
	if err != nil {
		return ResultProcessOutcome{}, err
	}
	if inserted {
		query, args, buildErr := storage.Psql.Update("parameter_sync_task_results").
			Set("status", "failed").Set("processed_at", p.now()).Set("error_message", message).
			Where(sq.Eq{"run_id": result.RunID, "task_id": result.TaskID}).ToSql()
		if buildErr != nil {
			return ResultProcessOutcome{}, fmt.Errorf("build mark unsafe parameter sync result failed: %w", buildErr)
		}
		if _, execErr := tx.Exec(ctx, query, args...); execErr != nil {
			return ResultProcessOutcome{}, fmt.Errorf("mark unsafe parameter sync result failed: %w", execErr)
		}
	}
	if inserted {
		counts, countErr := loadAuthoritativeRunCounts(ctx, tx, run.ID)
		if countErr != nil {
			return ResultProcessOutcome{}, countErr
		}
		applyAuthoritativeRunCounts(run, counts)
		status := run.Status
		if !status.Terminal() && status != RunStatusCancelling {
			if err := beginCancellingRun(ctx, tx, run, message, p.now()); err != nil {
				return ResultProcessOutcome{}, err
			}
			status = RunStatusCancelling
		}
		if status == RunStatusCancelling && run.ReadyToFinalize() {
			if err := finalizeConvergedFailedRun(ctx, tx, run, p.now()); err != nil {
				return ResultProcessOutcome{}, err
			}
		} else if err := updateRunProgress(ctx, tx, run, status); err != nil {
			return ResultProcessOutcome{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return ResultProcessOutcome{}, fmt.Errorf("commit unsafe parameter sync result: %w", err)
	}
	if p.metrics != nil && !inserted {
		p.metrics.ResultRedelivery.Inc()
	}
	return ResultProcessOutcome{Duplicate: !inserted, Finalized: inserted, Failed: inserted}, nil
}
