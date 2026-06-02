-- +goose Up
-- 修复：MML 任务状态永远卡在 "执行中"（running）。
--
-- 根因：mml_tasks.result 列被建成 jsonb，但 Go 模型 MMLTask.Result 是标量枚举
-- (*TaskResult: success/partial/failed)，repository 读写都按文本处理——
-- Update 写 `Set("result", task.Result)` 传裸串、Scan(&t.Result) 直接读裸串。
-- jsonb 列收到裸串 "failed"（不是合法 JSON，JSON 字符串需带引号）→ PG 报
-- `invalid input syntax for type json (SQLSTATE 22P02)` → ResultAggregator
-- .finalizeIfComplete 的 UPDATE 整条失败 → 任务 status 永远停在 running
-- （success_count/failed_count 由独立的 IncrementStats UPDATE 维护，不受影响，
--  所以计数正确、状态却不动）。
--
-- 修复：把 result 改成 text，与读写代码的标量语义一致（device_sns/commands/
-- results 仍是 jsonb，它们确实是数组/对象，不在本次改动范围）。

-- +goose StatementBegin
DO $$
BEGIN
    IF (SELECT data_type
          FROM information_schema.columns
         WHERE table_name = 'mml_tasks' AND column_name = 'result') = 'jsonb' THEN
        -- 历史值基本均为 NULL（写入一直失败）；#>>'{}' 把可能存在的 jsonb 标量字符串
        -- 提取为不带引号的 text，NULL 仍为 NULL。
        ALTER TABLE mml_tasks
            ALTER COLUMN result TYPE text USING (result #>> '{}');
    END IF;
END $$;
-- +goose StatementEnd

-- 数据补偿：把此前因上述 bug 卡在 running、但其 device_tasks 已全部进入终态的
-- 任务一次性收尾（判据与 ResultAggregator.finalizeIfComplete 一致：实际派发的
-- device_tasks 全部终态、且至少派发过一条）。仍有在途 device_task 的任务不动。
-- +goose StatementBegin
WITH agg AS (
    SELECT source_id::uuid AS mml_id,
           COUNT(*)                                                   AS total,
           COUNT(*) FILTER (WHERE status NOT IN ('completed','failed','expired')) AS active,
           COUNT(*) FILTER (WHERE status = 'completed')               AS completed,
           COUNT(*) FILTER (WHERE status IN ('failed','expired'))     AS failed
      FROM device_tasks
     WHERE source = 'mml'
     GROUP BY source_id
)
UPDATE mml_tasks m
   SET status        = CASE WHEN a.failed = 0 THEN 'completed'
                            WHEN a.completed = 0 THEN 'failed'
                            ELSE 'completed' END,
       result        = CASE WHEN a.failed = 0 THEN 'success'
                            WHEN a.completed = 0 THEN 'failed'
                            ELSE 'partial' END,
       success_count = a.completed,
       failed_count  = a.failed,
       finished_at   = COALESCE(m.finished_at, now()),
       updated_at    = now()
  FROM agg a
 WHERE m.id = a.mml_id
   AND a.total > 0
   AND a.active = 0
   AND m.status = 'running';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF (SELECT data_type
          FROM information_schema.columns
         WHERE table_name = 'mml_tasks' AND column_name = 'result') = 'text' THEN
        ALTER TABLE mml_tasks
            ALTER COLUMN result TYPE jsonb USING (to_jsonb(result));
    END IF;
END $$;
-- +goose StatementEnd
