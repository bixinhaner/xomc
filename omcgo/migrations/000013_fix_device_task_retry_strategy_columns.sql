-- +goose Up
-- +goose StatementBegin
ALTER TABLE public.device_tasks
    ADD COLUMN IF NOT EXISTS retry_interval_seconds integer DEFAULT 0 NOT NULL,
    ADD COLUMN IF NOT EXISTS next_attempt_at timestamp with time zone;

COMMENT ON COLUMN public.device_tasks.retry_interval_seconds IS '自动重试间隔秒数；MML failed_retry_interval 下传后用于控制失败重试间隔。0 表示立即可重试。';
COMMENT ON COLUMN public.device_tasks.next_attempt_at IS '下一次允许出队时间；用于 MML 失败重试延迟，未到时间的 pending 任务不会被 Redis 队列弹出。';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
