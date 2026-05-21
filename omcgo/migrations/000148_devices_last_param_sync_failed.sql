-- +goose Up
-- 参数同步失败可见性:补两列承载"上次失败时刻 + 失败原因"。
-- 与 last_param_sync_at(成功时刻)互斥:成功时清空 failed_at + error,失败时清空 succeeded_at。
-- 前端 GetSyncStatus 据此判定 idle / failed 子态:failed_at 非空显红字"上次同步失败"。
ALTER TABLE devices ADD COLUMN IF NOT EXISTS last_param_sync_failed_at TIMESTAMPTZ NULL;
ALTER TABLE devices ADD COLUMN IF NOT EXISTS last_param_sync_error TEXT NULL;

-- +goose Down
ALTER TABLE devices DROP COLUMN IF EXISTS last_param_sync_error;
ALTER TABLE devices DROP COLUMN IF EXISTS last_param_sync_failed_at;
