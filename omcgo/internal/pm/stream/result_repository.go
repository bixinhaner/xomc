package stream

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const resultStagingTable = "pm_aggregation_results_staging"

var resultColumns = []string{
	"window_start", "window_end", "task_id", "task_version_id",
	"granularity", "dimension", "dimension_key", "dimension_name",
	"object_ldn", "device_oui", "device_sn", "technology",
	"metric_id", "metric_path", "metric_type",
	"aggregation_op", "metric_value", "sample_count", "complete", "missing_slots",
	"revision", "version_effective_from", "version_effective_to",
	"received_slots", "expected_slots", "version_expected_slots",
	"natural_expected_slots", "version_slice_complete", "period_complete",
}

type resultWriteTx interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	CopyFrom(
		context.Context,
		pgx.Identifier,
		[]string,
		pgx.CopyFromSource,
	) (int64, error)
}

type resultCompleteness struct {
	version       *TaskVersionSnapshot
	coverage      finalizationCoverage
	receivedSlots int64
}

type resultCopySource struct {
	key          WindowKey
	revision     int
	metrics      []finalizedMetric
	completeness resultCompleteness
	index        int
	values       []any
}

func newResultCopySource(
	key WindowKey,
	revision int,
	metrics []finalizedMetric,
	completeness resultCompleteness,
) *resultCopySource {
	return &resultCopySource{
		key: key, revision: revision, metrics: metrics,
		completeness: completeness, index: -1,
		values: make([]any, len(resultColumns)),
	}
}

func (source *resultCopySource) Next() bool {
	source.index++
	return source.index < len(source.metrics)
}

func (source *resultCopySource) Values() ([]any, error) {
	metric := source.metrics[source.index]
	definition := metric.Definition
	copy(source.values, []any{
		source.key.Start, source.key.End, source.key.TaskID, source.key.TaskVersionID,
		string(source.key.Granularity), string(definition.Dimension),
		definition.DimensionKey, definition.DimensionName,
		definition.ObjectLDN, definition.DeviceOUI, definition.DeviceSN,
		definition.Technology, metric.MetricID, definition.MetricPath,
		metric.MetricType, string(metric.Operation), metric.Value,
		metric.SampleCount,
		source.completeness.coverage.PeriodComplete && metric.FormulaComplete,
		source.completeness.coverage.MissingSlots,
		source.revision,
		versionEffectiveFrom(source.completeness.version),
		versionEffectiveTo(source.completeness.version),
		source.completeness.receivedSlots,
		source.completeness.coverage.NaturalSlots,
		source.completeness.coverage.VersionExpectedSlots,
		source.completeness.coverage.NaturalSlots,
		source.completeness.coverage.DataComplete && metric.FormulaComplete,
		source.completeness.coverage.PeriodComplete && metric.FormulaComplete,
	})
	return source.values, nil
}

func (source *resultCopySource) Err() error {
	return nil
}

// ReplaceWindowResults writes one complete window result set with a fixed
// number of database round trips. First publication never deletes rows.
// Revisions remove only business keys that are absent from the new staged set.
func ReplaceWindowResults(
	ctx context.Context,
	tx resultWriteTx,
	key WindowKey,
	revision int,
	metrics []finalizedMetric,
	completeness resultCompleteness,
) (int, error) {
	if _, err := tx.Exec(ctx, `
CREATE TEMP TABLE IF NOT EXISTS pm_aggregation_results_staging
  (LIKE public.pm_aggregation_results INCLUDING DEFAULTS)
  ON COMMIT DELETE ROWS`); err != nil {
		return 0, fmt.Errorf("create PM aggregation result staging table: %w", err)
	}
	if _, err := tx.Exec(ctx, "TRUNCATE pm_aggregation_results_staging"); err != nil {
		return 0, fmt.Errorf("clear PM aggregation result staging table: %w", err)
	}

	copied, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{resultStagingTable},
		resultColumns,
		newResultCopySource(key, revision, metrics, completeness),
	)
	if err != nil {
		return 0, fmt.Errorf("stage PM aggregation results: %w", err)
	}
	if copied != int64(len(metrics)) {
		return 0, fmt.Errorf(
			"stage PM aggregation results: copied %d rows, want %d",
			copied,
			len(metrics),
		)
	}

	tag, err := tx.Exec(ctx, `
INSERT INTO pm_aggregation_results (
  window_start, window_end, task_id, task_version_id,
  granularity, dimension, dimension_key, dimension_name,
  object_ldn, device_oui, device_sn, technology,
  metric_id, metric_path, metric_type,
  aggregation_op, metric_value, sample_count, complete, missing_slots,
  revision, version_effective_from, version_effective_to,
  received_slots, expected_slots, version_expected_slots,
  natural_expected_slots, version_slice_complete, period_complete
)
SELECT
  window_start, window_end, task_id, task_version_id,
  granularity, dimension, dimension_key, dimension_name,
  object_ldn, device_oui, device_sn, technology,
  metric_id, metric_path, metric_type,
  aggregation_op, metric_value, sample_count, complete, missing_slots,
  revision, version_effective_from, version_effective_to,
  received_slots, expected_slots, version_expected_slots,
  natural_expected_slots, version_slice_complete, period_complete
FROM pm_aggregation_results_staging
ON CONFLICT (
  task_version_id, granularity, window_start, dimension_key,
  object_ldn, technology, metric_id
) DO UPDATE SET
  window_end = EXCLUDED.window_end,
  task_id = EXCLUDED.task_id,
  dimension = EXCLUDED.dimension,
  dimension_name = EXCLUDED.dimension_name,
  device_oui = EXCLUDED.device_oui,
  device_sn = EXCLUDED.device_sn,
  metric_path = EXCLUDED.metric_path,
  metric_type = EXCLUDED.metric_type,
  aggregation_op = EXCLUDED.aggregation_op,
  metric_value = EXCLUDED.metric_value,
  sample_count = EXCLUDED.sample_count,
  complete = EXCLUDED.complete,
  missing_slots = EXCLUDED.missing_slots,
  revision = EXCLUDED.revision,
  version_effective_from = EXCLUDED.version_effective_from,
  version_effective_to = EXCLUDED.version_effective_to,
  received_slots = EXCLUDED.received_slots,
  expected_slots = EXCLUDED.expected_slots,
  version_expected_slots = EXCLUDED.version_expected_slots,
  natural_expected_slots = EXCLUDED.natural_expected_slots,
  version_slice_complete = EXCLUDED.version_slice_complete,
  period_complete = EXCLUDED.period_complete`)
	if err != nil {
		return 0, fmt.Errorf("upsert PM aggregation results: %w", err)
	}
	resultCount := int(tag.RowsAffected())

	if revision > 1 {
		if _, err := tx.Exec(ctx, `
DELETE FROM pm_aggregation_results AS existing
WHERE existing.task_version_id = $1
  AND existing.granularity = $2
  AND existing.window_start = $3
  AND existing.dimension_key = $4
  AND NOT EXISTS (
    SELECT 1
    FROM pm_aggregation_results_staging AS staged
    WHERE staged.task_version_id = existing.task_version_id
      AND staged.granularity = existing.granularity
      AND staged.window_start = existing.window_start
      AND staged.dimension_key = existing.dimension_key
      AND staged.object_ldn = existing.object_ldn
      AND staged.technology = existing.technology
      AND staged.metric_id = existing.metric_id
  )`,
			key.TaskVersionID,
			string(key.Granularity),
			key.Start,
			key.EntityKey,
		); err != nil {
			return 0, fmt.Errorf("delete stale PM aggregation results: %w", err)
		}
	}
	return resultCount, nil
}
