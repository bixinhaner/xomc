-- +goose Up
-- ============================================================
-- 000138_upgrade_sub_tasks_uploading_state.sql
-- 为 upgrade_sub_tasks.status 加入 'uploading' 取值。
--
-- 背景：原 'downloading' 同时承载 Download RPC（升级 / 回滚）与 Upload RPC
-- （备份 / 日志采集 / 配置恢复）两条流程，导致 UI 文案、reaper 超时口径、
-- 状态机校验都得"二次推断任务类型"。本迁移把 Upload 链路拆出独立状态。
--
-- 影响面：
--   · 旧行（仍写 'downloading'）保持有效；
--   · 新代码 ExecuteOneUpload 改写 'uploading'；
--   · ufte/normalizeDeviceStatus 按状态直译，无需 isUpload 参数；
--   · reaper FailStale SQL 用 TransferComplete cutoff（30min）兜底 uploading，
--     原 'downloading + RPCResponse 5min' 对大文件场景过严的 BUG 顺手修掉。
-- ============================================================

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_upgrade_sub_tasks_status') THEN
        ALTER TABLE upgrade_sub_tasks DROP CONSTRAINT chk_upgrade_sub_tasks_status;
    END IF;
    ALTER TABLE upgrade_sub_tasks ADD CONSTRAINT chk_upgrade_sub_tasks_status
        CHECK (status IN ('pending', 'downloading', 'uploading', 'rebooting', 'verifying',
                           'completed', 'failed', 'suspended', 'terminated'));
END $$;
-- +goose StatementEnd

-- +goose Down
-- 回退时：把残留的 'uploading' 行先归一回 'downloading'（旧语义），再缩约束。
-- 这一步会丢失 Upload 的细分语义，但保持表数据合法。
UPDATE upgrade_sub_tasks SET status = 'downloading' WHERE status = 'uploading';

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_upgrade_sub_tasks_status') THEN
        ALTER TABLE upgrade_sub_tasks DROP CONSTRAINT chk_upgrade_sub_tasks_status;
    END IF;
    ALTER TABLE upgrade_sub_tasks ADD CONSTRAINT chk_upgrade_sub_tasks_status
        CHECK (status IN ('pending', 'downloading', 'rebooting', 'verifying',
                           'completed', 'failed', 'suspended', 'terminated'));
END $$;
-- +goose StatementEnd
