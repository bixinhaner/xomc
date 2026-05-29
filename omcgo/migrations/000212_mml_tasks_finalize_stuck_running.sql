-- +goose Up
-- ============================================================
-- 000205_mml_tasks_finalize_stuck_running.sql
--
-- 2026-05-28 一次性补 finalize 历史"卡 running"任务。
--
-- 根因(已在同 commit 修复):
--   pg_repository.go PgTaskRepository.Update 把 []byte 传入 JSONB 列,
--   pgx v5 把 []byte 编为 BYTEA wire format,PG 把字节流当 text 反解 JSON
--   失败 → "invalid input syntax for type json (SQLSTATE 22P02)" →
--   整条 UPDATE 失败 → finalizeIfComplete 无法把 status / result /
--   finished_at 写回(success_count 增加在另一条独立 UPDATE 中,不受影响)。
--   结果:大量任务 status='running' 但 success_count + failed_count =
--   total_devices,前端看上去"任务执行完但状态卡在执行中"。
--
-- 修复策略:
--   把所有"进度满但 status 仍 running"的 mml_tasks 一次性 finalize,
--   与 ResultAggregator.finalizeIfComplete 规则保持一致:
--     · failed=0          → status=completed, result=success
--     · success=0         → status=failed,    result=failed
--     · 兼有              → status=completed, result=partial
--   finished_at 取 updated_at 兜底(IncrementStats 写了 updated_at),
--   updated_at 为 NULL 时回退到 NOW()。
--
-- 注意:total_devices > 0 防御性过滤,避免 0 设备任务(理论上不应存在)
-- 误判为已完成。
-- ============================================================

-- 注意:result 列是 JSONB,app 侧 TaskResult string 经 json.Marshal 写入
-- 实际是 JSON 字符串(含外层引号,如 "success")。直接给 JSONB 列赋裸 text
-- 会报 SQLSTATE 42804:column "result" is of type jsonb but expression is of
-- type text。用 to_jsonb(text) 在 server 侧转 JSON 字符串。
UPDATE mml_tasks
   SET status      = CASE
                       WHEN failed_count  = 0 THEN 'completed'
                       WHEN success_count = 0 THEN 'failed'
                       ELSE 'completed'
                     END,
       result      = CASE
                       WHEN failed_count  = 0 THEN to_jsonb('success'::text)
                       WHEN success_count = 0 THEN to_jsonb('failed'::text)
                       ELSE to_jsonb('partial'::text)
                     END,
       finished_at = COALESCE(finished_at, updated_at, NOW())
 WHERE status = 'running'
   AND total_devices > 0
   AND (success_count + failed_count) >= total_devices;


-- +goose Down
-- 无法精确还原原始 'running' 状态(updated_at / finished_at 已被覆盖),
-- Down 为 noop。如必须回滚此修复,请配合 pg_repository.go 一并回退到
-- []byte 写 JSONB 的版本。
SELECT 1;
