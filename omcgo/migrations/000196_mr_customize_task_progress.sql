-- +goose Up
-- ============================================================
-- 000196_mr_customize_task_progress.sql
-- MR 测量任务 — 小站级进度表
--
-- 一个 MR 任务包含 N 个目标小站（cell）。本表记录每个 cell 在任务生命
-- 周期内的状态：开启下发结果、关闭下发结果、平台支持判断、上报心跳。
--
-- 与主表的关系：N:1（每任务多 cell），主表删除时级联清理。
--
-- progress_status 状态语义（按 MR_Feature_Analysis.md §10）：
--   pending       — 初始，未触发下发
--   openSuccess   — 开启 SPV 设备应答成功
--   openFailure   — 开启 SPV 设备应答失败（fault_code 记录原因）
--   closeSuccess  — 关闭 SPV 应答成功
--   closeFailure  — 关闭 SPV 应答失败
--   unsupport     — 平台不在 IntelCR/BLQ/MLQ/MLN 支持列表
--   timeOut       — SPV 下发后无响应（内部码 1001）
--   noPermission  — 用户无 MR 功能权限
--
-- health_status 是 progress_status 之外的"运行时上报健康度"维度：
--   normal  — Redis MRFileReport_{cellCode} 存在（设备按周期上报中）
--   abnormal — 连续 N 个 UploadPeriod 周期未上报（scheduler 巡检判定）
--   unknown — 未开启或已关闭，不进行健康检测
--
-- 相关：开发计划 docs/project/mr-task-management-development-plan-20260525.md
-- ============================================================

CREATE TABLE IF NOT EXISTS mr_customize_task_progress (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id          UUID         NOT NULL
                                  REFERENCES mr_customize_task(task_id)
                                  ON DELETE CASCADE,
    small_cell_code  VARCHAR(64)  NOT NULL,
    serial_number    VARCHAR(64)  NOT NULL,
    host_name        VARCHAR(128),
    progress_status  VARCHAR(32)  NOT NULL DEFAULT 'pending',
    health_status    VARCHAR(16)  NOT NULL DEFAULT 'unknown',
    fault_code       VARCHAR(32),
    -- 最近一次设备主动上报 MR 文件的时间，由 transfer/bridge.go 写入
    last_heartbeat   TIMESTAMPTZ,
    -- 连续未命中心跳的次数，scheduler 巡检时递增 / 重置
    missed_heartbeat INTEGER      NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_mr_progress_status CHECK (
        progress_status IN (
            'pending',
            'openSuccess', 'openFailure',
            'closeSuccess', 'closeFailure',
            'unsupport', 'timeOut', 'noPermission'
        )
    ),
    CONSTRAINT chk_mr_progress_health CHECK (
        health_status IN ('normal','abnormal','unknown')
    ),
    -- 同一任务内同一 cell 不应出现两行（业务唯一约束）
    CONSTRAINT uq_mr_progress_task_cell UNIQUE (task_id, small_cell_code)
);

COMMENT ON TABLE  mr_customize_task_progress IS 'MR 任务 — 小站级进度，含下发结果与上报心跳健康度';
COMMENT ON COLUMN mr_customize_task_progress.progress_status IS 'pending/openSuccess/openFailure/closeSuccess/closeFailure/unsupport/timeOut/noPermission';
COMMENT ON COLUMN mr_customize_task_progress.health_status IS 'normal/abnormal/unknown — Redis MRFileReport TTL 巡检结果';
COMMENT ON COLUMN mr_customize_task_progress.missed_heartbeat IS '连续未命中心跳次数；scheduler 阈值 (默认 2) 时标记 abnormal';

-- 任务详情页常按 task_id 过滤
CREATE INDEX IF NOT EXISTS idx_mr_progress_task_status
    ON mr_customize_task_progress (task_id, progress_status);
-- transfer bridge 写心跳时按 small_cell_code 反查 task_id
CREATE INDEX IF NOT EXISTS idx_mr_progress_cell
    ON mr_customize_task_progress (small_cell_code);
-- scheduler 巡检健康度按 health_status 过滤
CREATE INDEX IF NOT EXISTS idx_mr_progress_health
    ON mr_customize_task_progress (health_status)
    WHERE progress_status = 'openSuccess';

CREATE TRIGGER trg_mr_customize_task_progress_updated_at
    BEFORE UPDATE ON mr_customize_task_progress
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
DROP TRIGGER IF EXISTS trg_mr_customize_task_progress_updated_at ON mr_customize_task_progress;
DROP INDEX IF EXISTS idx_mr_progress_health;
DROP INDEX IF EXISTS idx_mr_progress_cell;
DROP INDEX IF EXISTS idx_mr_progress_task_status;
DROP TABLE IF EXISTS mr_customize_task_progress;
