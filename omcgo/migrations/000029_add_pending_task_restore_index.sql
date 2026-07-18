-- +goose Up
-- Worker 启动按全局 (created_at, id) keyset 顺序恢复 pending 任务。原部分索引以
-- device_sn 开头，无法服务此排序，机械盘冷启动会扫描 16 个分区的大量历史任务。
-- 在分区父表创建索引会自动为每个现有分区创建并挂接对应的部分索引。
CREATE INDEX IF NOT EXISTS idx_device_tasks_pending_created_id
    ON public.device_tasks (created_at, id)
    WHERE status = 'pending';

-- +goose Down
DROP INDEX IF EXISTS public.idx_device_tasks_pending_created_id;
