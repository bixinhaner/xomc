-- +goose Up
-- ============================================================
-- 000195_mr_customize_task.sql
-- MR 测量任务主表
--
-- 背景：F05 MR 模块已具备文件接收/解析/查询链路（internal/transfer/bridge.go
-- → internal/mr/collector → MRO/MRS/MRE parser），但缺少"触发设备主动上报"
-- 的任务编排能力。设备默认不上报 MR 文件，必须由 OMC 通过 TR-069
-- SetParameterValues 写 Device.FAP.MRMgmt.Config.{i}.* 6 个参数（启用 + URL
-- + 周期）后，设备才会按 UploadPeriod 周期 HTTP POST 上传。
--
-- 本表记录用户创建的 MR 任务策略：测量类型、统计周期、上报周期、起止时间
-- 等元数据。worker 进程的 scheduler 按 start_time 自动开启、按 end_time
-- 自动关闭，状态机：waitting → on → off（或被手动 stop 至 termination）。
--
-- 字段命名遵循 MR_Feature_Analysis.md §10 规范（保留 statis_period /
-- report_period / task_status 等原文命名），便于与文档对照。
--
-- 相关：PRD docs/project/prd/F05-mr-task-management.md
--      开发计划 docs/project/mr-task-management-development-plan-20260525.md
-- ============================================================

CREATE TABLE IF NOT EXISTS mr_customize_task (
    task_id        UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    task_name      VARCHAR(128)    NOT NULL,
    -- 测量类型组合（如 "MRS,MRE,MRO"），文档 §3.1 强制三项全选；
    -- 不通过 TR-069 下发，仅做业务分类。
    mr_type        VARCHAR(64)     NOT NULL DEFAULT 'MRS,MRE,MRO',
    -- 统计周期：PeriodicReportInterval 下发原值
    -- 合法集合 {ms2048, ms5120, ms10240, min1, min6, min12, min30, min60}
    -- 存储下发值（"2048" / "5120" / "10240" / "1" / "6" / "12" / "30" / "60"）
    statis_period  VARCHAR(16)     NOT NULL DEFAULT '5120',
    -- 上报周期（分钟）：UI 值，下发时 × 60 转为 UploadPeriod 秒
    -- 合法集合 {"15", "30", "60"}
    report_period  VARCHAR(8)      NOT NULL DEFAULT '15',
    -- UTC 时间存储；前端按用户时区转换
    start_time     TIMESTAMPTZ     NOT NULL,
    end_time       TIMESTAMPTZ,    -- NULL = 无限制
    task_status    VARCHAR(16)     NOT NULL DEFAULT 'waitting',
    task_result    VARCHAR(16),    -- 任务整体结果，关闭后由 scheduler 汇总
    operator_code  VARCHAR(16)     NOT NULL,
    creator        VARCHAR(64)     NOT NULL,
    note           TEXT,
    created_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_mr_task_status CHECK (
        task_status IN ('waitting','on','off','suspend','termination')
    ),
    CONSTRAINT chk_mr_task_time_order CHECK (
        end_time IS NULL OR end_time > start_time
    ),
    CONSTRAINT chk_mr_task_report_period CHECK (
        report_period IN ('15','30','60')
    )
);

COMMENT ON TABLE  mr_customize_task IS 'MR 测量任务主表（F05），由用户创建，worker scheduler 按 start_time/end_time 自动下发开启/关闭 SPV';
COMMENT ON COLUMN mr_customize_task.mr_type IS '测量类型组合，逗号分隔，强制含 MRS,MRE,MRO';
COMMENT ON COLUMN mr_customize_task.statis_period IS 'PeriodicReportInterval 下发原值';
COMMENT ON COLUMN mr_customize_task.report_period IS '上报周期（分钟），下发为 UploadPeriod = report_period × 60 秒';
COMMENT ON COLUMN mr_customize_task.task_status IS 'waitting=待执行/on=执行中/off=已关闭/suspend=已挂起/termination=终止中';

-- scheduler 每 30s 扫描 "待执行且已到期" 与 "执行中且已到期 end_time"
-- 复合索引覆盖两条查询路径
CREATE INDEX IF NOT EXISTS idx_mr_task_status_starttime
    ON mr_customize_task (task_status, start_time);
CREATE INDEX IF NOT EXISTS idx_mr_task_status_endtime
    ON mr_customize_task (task_status, end_time)
    WHERE end_time IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_mr_task_operator
    ON mr_customize_task (operator_code, created_at DESC);

-- updated_at 自动更新（复用 000001 创建的 update_updated_at_column）
CREATE TRIGGER trg_mr_customize_task_updated_at
    BEFORE UPDATE ON mr_customize_task
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
DROP TRIGGER IF EXISTS trg_mr_customize_task_updated_at ON mr_customize_task;
DROP INDEX IF EXISTS idx_mr_task_operator;
DROP INDEX IF EXISTS idx_mr_task_status_endtime;
DROP INDEX IF EXISTS idx_mr_task_status_starttime;
DROP TABLE IF EXISTS mr_customize_task;
