-- +goose Up
-- +goose StatementBegin

-- Keep publish/ack crash-gap recovery proportional to the unacknowledged tail
-- instead of scanning the full retained outbox.
CREATE INDEX IF NOT EXISTS idx_pm_aggregation_outbox_unacknowledged
    ON public.pm_aggregation_outbox (published_at, event_id)
    WHERE consumed_at IS NULL
      AND barrier_eligible
      AND published_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_pm_aggregation_rollup_unacknowledged
    ON public.pm_aggregation_rollup_outbox (published_at, event_id)
    WHERE consumed_at IS NULL
      AND barrier_eligible
      AND published_at IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS public.idx_pm_aggregation_rollup_unacknowledged;
DROP INDEX IF EXISTS public.idx_pm_aggregation_outbox_unacknowledged;

-- +goose StatementEnd
