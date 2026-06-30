-- +goose Up
-- +goose StatementBegin

-- BUG-02 fix (#707)：创建 mml_audit_log 表。
--
-- 背景：MMLService.writeAuditLogs 在 PgAuditRepository.CreateBatch 中执行
-- INSERT INTO mml_audit_log，但该表未在 schema 中创建，导致每次执行 MML 命令
-- 后出现 "ERROR: relation "mml_audit_log" does not exist (SQLSTATE 42P01)"
-- 并追加 ~40 行堆栈到日志，严重污染日志并触发告警噪音。
--
-- 字段来自 pg_repository.go:PgAuditRepository.CreateBatch INSERT 语句：
--   (task_id, command_code, operation_type, device_sn, parameters, param_paths,
--    result_status, result_message, creator, duration_ms)

CREATE TABLE IF NOT EXISTS public.mml_audit_log (
    id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id        uuid        REFERENCES public.mml_tasks(id) ON DELETE SET NULL,
    command_code   varchar(255),
    operation_type varchar(20),
    device_sn      varchar(64),
    parameters     jsonb,
    param_paths    jsonb,
    result_status  varchar(20),
    result_message text,
    creator        varchar(100),
    duration_ms    numeric(12,3),
    created_at     timestamptz NOT NULL DEFAULT now()
);

-- 主要查询：按任务查审计、按设备+时间范围查历史
CREATE INDEX IF NOT EXISTS idx_mml_audit_log_task_id
    ON public.mml_audit_log (task_id)
    WHERE task_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_mml_audit_log_device_sn_created
    ON public.mml_audit_log (device_sn, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_mml_audit_log_command_code
    ON public.mml_audit_log (command_code)
    WHERE command_code IS NOT NULL;

COMMENT ON TABLE public.mml_audit_log IS
    'MML 命令执行审计日志。每次 ExecuteCommand 对每个 device+command 组合写一条记录，
     记录下发时的参数、最终结果状态与耗时，供合规审计与运维回溯。';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_mml_audit_log_command_code;
DROP INDEX IF EXISTS idx_mml_audit_log_device_sn_created;
DROP INDEX IF EXISTS idx_mml_audit_log_task_id;
DROP TABLE IF EXISTS public.mml_audit_log;

-- +goose StatementEnd
