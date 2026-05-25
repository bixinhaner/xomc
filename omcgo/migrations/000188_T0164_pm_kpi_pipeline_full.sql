-- T-0164 PM/KPI 流水线 G1-G8 DDL 合并迁移（draft/pm-kpi-impl → main 合并整理）
--
-- 合并自原 draft 分支 11 个 sub-migration（按 Up 顺序应用 / Down 反向回滚）：
--   - 000159_async_jobs
--   - 000160_create_pm_metrics_and_drop_legacy
--   - 000161_create_pm_aggregation_tables
--   - 000162_extend_pm_tasks_and_create_adhoc_results
--   - 000163_create_pm_dashboards_and_panels
--   - 000164_async_jobs_cron_state
--   - 000165_pm_user_prefs_by_technology
--   - 000167_pm_tasks_last_fire_at
--   - 000168_pm_panels_topn_bignumber_builtin_flag
--   - 000170_pm_panels_granularities_array
--   - 000171_pm_metrics_natural_key_object_ldn
--
-- 设计文档：docs/design/pm-kpi-pipeline-improvements.md G1-G8
-- 合并原因：draft 分支独立 PM 改造期间 main 并行演进（T-0173 等推到 000187），
-- 原 11 个文件占用 000159-000171 编号与 main 冲突无法干净 merge；合并到单一
-- 000188（接 main 末尾）后顺位入 main。
--
-- 各 sub-migration 完整历史保留在 git log（fc5c0336 之前的 commit 链路）。

-- +goose Up

-- ============================================================
-- 000159_async_jobs
-- ============================================================
-- +goose StatementBegin
-- T-0164-P8 / G8 通用任务框架：async_jobs 表
-- 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.8
--
-- 用途：进程级通用异步任务总线，承载 G5 cron 实例（小时/日/周/月聚合）+ 未来批量计算任务。
-- 状态机：pending → running → succeeded / failed / zombie / canceled
--   - 心跳上报：running 中每 30s 更新 heartbeat_at
--   - 僵尸检测：sweeper 每 60s 扫 heartbeat_at < now() - 5min 的 running 任务，重置 pending
--   - 重启续跑：worker 崩溃后 conn 归还触发 PG session 结束，advisory lock 自动释放；
--     僵尸任务下一 sweeper 回合自动 pending 让其他 worker 抢
--   - cron 触发：scheduler 内 cron tick（@由 LeaderElector 单实例选主，避免多 worker 重复入队）
--
-- 与 G7 自定义聚合任务的分工（设计 §4.7/§4.8 锁定）：
--   - G7 走 PM 模块 per-module 表 pm_tasks（独立 worker 池）
--   - G8 仅承载"系统层"任务：G5 cron + 未来批量计算

CREATE TABLE async_jobs (
    id              UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    job_type        TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending','running','succeeded','failed','zombie','canceled')),
    schedule_expr   TEXT,                       -- cron expression（仅 cron 触发的任务有值）
    scheduled_at    TIMESTAMPTZ NOT NULL,       -- 计划执行时刻（cron 解析后或手动设定）
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    heartbeat_at    TIMESTAMPTZ,                -- running 期间每 30s 更新
    lock_owner      TEXT,                       -- worker 进程 ID (hostname-pid)
    attempt         INT NOT NULL DEFAULT 1,
    max_attempts    INT NOT NULL DEFAULT 3,
    payload         JSONB,                      -- job 参数（如聚合的时间窗）
    result          JSONB,                      -- 成功时的结果（如聚合行数）
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 关键查询：LockNextPending(job_type) 用 (status, scheduled_at, job_type) 三列定位
CREATE INDEX idx_async_jobs_pending_pickup
    ON async_jobs (job_type, status, scheduled_at)
    WHERE status = 'pending';

-- 关键查询：ListZombies 扫 running 且 heartbeat_at 过期
CREATE INDEX idx_async_jobs_zombie_check
    ON async_jobs (status, heartbeat_at)
    WHERE status = 'running';

-- 历史排查：按 job_type + created_at 倒序看最近运行历史
CREATE INDEX idx_async_jobs_type_created
    ON async_jobs (job_type, created_at DESC);

-- updated_at 触发器（与项目其他表一致）— 复用 000001 已创建的 update_updated_at_column()
CREATE TRIGGER trg_async_jobs_updated_at
    BEFORE UPDATE ON async_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE  async_jobs IS 'T-0164-P8 / G8 通用异步任务总线（承载 G5 cron + 未来批量计算）';
COMMENT ON COLUMN async_jobs.status IS 'pending/running/succeeded/failed/zombie/canceled';
COMMENT ON COLUMN async_jobs.heartbeat_at IS 'running 期间每 30s 更新；sweeper 用 heartbeat_at < now()-5min 识别僵尸';
COMMENT ON COLUMN async_jobs.lock_owner IS 'worker 进程标识 hostname-pid，便于排查"哪个 worker 抢到任务"';
-- +goose StatementEnd

-- ============================================================
-- 000160_create_pm_metrics_and_drop_legacy
-- ============================================================
-- +goose StatementBegin
-- T-0164-P3 / G3 合并 pm_counters + kpi_values 为单表 pm_metrics
-- 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.3
--
-- 变更：
--   1. DROP MATERIALIZED VIEW pm_counters_hourly（TimescaleDB continuous aggregate，CASCADE 删底层 _materialized_hypertable_4）
--   2. DROP TABLE pm_counters / kpi_values（项目未上生产，0 行数据；schema 备份在 /tmp/pm-kpi-pre-G3-backup-20260523.sql）
--   3. CREATE TABLE pm_metrics（统一 counter + kpi 存储，metric_type 区分）
--      + hypertable on `time`，chunk_time_interval=1day
--      + compression（segmentby device_sn/metric_type/granularity）+ 7d policy
--      + retention 30d policy（G2 保留策略默认值，G5 配置化时 alter_job 联动）
--
-- pm_metrics 设计要点：
--   - metric_type CHECK ('counter','kpi')：单表存两类指标
--   - statis_type CHECK ('sum','avg','max','pct') 或 NULL：counter 必填（驱动 G5 自然桶聚合），kpi 为 NULL
--   - granularity CHECK ('15min','hourly','daily','weekly','monthly')：G3 仅 '15min'，G5 加 'hourly'+ 等
--   - 三时间字段 NOT NULL（T-0164-P4 / G4 已在 parser 层填充 FileBeginTime/FileEndTime/IngestTime）
--   - **设备唯一标识 (device_oui, device_sn)**：TR-069 标准双键
--       device_oui — TR-069 DeviceId.OUI（6 位十六进制大写厂商标识），如 '48BF74'
--       device_sn  — TR-069 DeviceId.SerialNumber（厂商内序列号），如 '1202000240194DP0026'
--       全系统设备唯一标识由这两列组合决定（详见 docs/project/plan-T-0165-system-wide-oui-sn-migration.md）
--   - 自然键唯一索引 (device_oui, device_sn, metric_path, granularity, end_time, time)：补传去重 + ON CONFLICT 幂等
--     （TimescaleDB 要求 UNIQUE 索引必须包含分区列 `time`；业务上 time = end_time，约束意义不变）
--   - PRIMARY KEY (id, time)：TS hypertable 要求 PK 含分区列
--   - object_ldn 替代旧 cell_id：更通用的 LDN 对象标识（cell / sector / 等）
--   - extra JSONB：扩展（carrier / technology / counter_group 等）
--
-- 现场聚合接口（pm/handler.go）暂走 pm_metrics granularity='15min'，G5 后路由到聚合表。

DROP MATERIALIZED VIEW IF EXISTS pm_counters_hourly CASCADE;
DROP TABLE IF EXISTS pm_counters CASCADE;
DROP TABLE IF EXISTS kpi_values CASCADE;

CREATE TABLE pm_metrics (
    id            UUID NOT NULL DEFAULT gen_random_uuid(),
    device_oui    TEXT NOT NULL,
    device_sn     TEXT NOT NULL,
    metric_path   TEXT NOT NULL,
    metric_type   TEXT NOT NULL CHECK (metric_type IN ('counter', 'kpi')),
    metric_value  DOUBLE PRECISION NOT NULL,
    statis_type   TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum', 'avg', 'max', 'min', 'pct')),
    granularity   TEXT NOT NULL CHECK (granularity IN ('15min', 'hourly', 'daily', 'weekly', 'monthly')),
    time          TIMESTAMPTZ NOT NULL,
    start_time    TIMESTAMPTZ NOT NULL,
    end_time      TIMESTAMPTZ NOT NULL,
    ingest_time   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    object_ldn    TEXT,
    extra         JSONB,
    PRIMARY KEY (id, time)
);

-- TimescaleDB 要求 UNIQUE 索引必须包含分区列 `time`；业务上 time = end_time，
-- 增加 time 列不改变自然键去重语义。
-- 设备唯一标识用 (device_oui, device_sn) 双键（TR-069 标准）
CREATE UNIQUE INDEX uq_pm_metrics_natural
    ON pm_metrics (device_oui, device_sn, metric_path, granularity, end_time, time);

CREATE INDEX idx_pm_metrics_path_time   ON pm_metrics (metric_path, time DESC);
CREATE INDEX idx_pm_metrics_device_time ON pm_metrics (device_oui, device_sn, time DESC);
CREATE INDEX idx_pm_metrics_ingest_time ON pm_metrics (ingest_time DESC);
CREATE INDEX idx_pm_metrics_object_ldn  ON pm_metrics (object_ldn) WHERE object_ldn IS NOT NULL;

SELECT create_hypertable('pm_metrics', 'time', chunk_time_interval => INTERVAL '1 day');

ALTER TABLE pm_metrics SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'device_oui,device_sn,metric_type,granularity'
);

SELECT add_compression_policy('pm_metrics', INTERVAL '7 days');
SELECT add_retention_policy('pm_metrics', INTERVAL '30 days');
-- +goose StatementEnd

-- ============================================================
-- 000161_create_pm_aggregation_tables
-- ============================================================
-- +goose StatementBegin
-- T-0164-P5 / G5 自然日历桶聚合表（设备维度 + 设备组维度）
-- 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.5
-- 实施 plan：docs/project/plan-T-0164-P5-natural-bucket-aggregation.md
--
-- 变更：
--   1. 新增 4 张设备维度聚合表：
--        pm_metrics_hourly  — hypertable（chunk 7d，compress 14d，retention 180d）
--        pm_metrics_daily   — 普通表，G2 cron 按 sys_configs 清理
--        pm_metrics_weekly  — 普通表
--        pm_metrics_monthly — 普通表
--   2. 新增 4 张设备组维度聚合表（同粒度）：
--        pm_group_metrics_hourly / daily / weekly / monthly
--
-- 设计要点（与 pm_metrics 一致）：
--   - 设备唯一标识 (device_oui, device_sn) TR-069 双键（T-0164-P3 / G3 引入）
--   - statis_type / granularity / metric_type 等枚举约束与 pm_metrics 完全对齐
--   - hourly 仿 pm_metrics 走 hypertable + PRIMARY KEY (id, time) + UNIQUE 自然键，
--     支持 ON CONFLICT 幂等 UPSERT（cron 重跑 / 补算同窗口安全）
--   - daily/weekly/monthly 量小（年级别 ~1500 行/设备/KPI），普通表 PRIMARY KEY
--     直接落自然键，不再需要 id 列，节省一次 gen_random_uuid 调用
--   - device_group 维度同 device 维度但维度键换成 device_group_id（UUID）；
--     device_group_id 不建外键（device_groups 是普通表但避免删 group 级联连锁）
--   - extra JSONB 与 pm_metrics 一致，可装 carrier/technology/cluster 等扩展
--
-- 与 pm_metrics 的关系：
--   - 数据流：pm_metrics(15min) → pm_metrics_hourly → pm_metrics_daily → weekly → monthly
--   - cron runner（G5 §3 Task 3 + worker main G8 wiring）按整点对齐自然桶聚合
--   - QueryAggregated 路由：粒度 15min → pm_metrics，其余 → 对应粒度表（device 维度）
--                          或对应 pm_group_metrics_* 表（device_group 维度）

-- ==================== pm_metrics_hourly：hypertable ====================
CREATE TABLE pm_metrics_hourly (
    id            UUID NOT NULL DEFAULT gen_random_uuid(),
    device_oui    TEXT NOT NULL,
    device_sn     TEXT NOT NULL,
    metric_path   TEXT NOT NULL,
    metric_type   TEXT NOT NULL CHECK (metric_type IN ('counter', 'kpi')),
    metric_value  DOUBLE PRECISION NOT NULL,
    statis_type   TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum', 'avg', 'max', 'min', 'pct')),
    granularity   TEXT NOT NULL CHECK (granularity IN ('15min', 'hourly', 'daily', 'weekly', 'monthly')),
    time          TIMESTAMPTZ NOT NULL,
    start_time    TIMESTAMPTZ NOT NULL,
    end_time      TIMESTAMPTZ NOT NULL,
    ingest_time   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    object_ldn    TEXT,
    extra         JSONB,
    PRIMARY KEY (id, time)
);

CREATE UNIQUE INDEX uq_pm_metrics_hourly_natural
    ON pm_metrics_hourly (device_oui, device_sn, metric_path, granularity, end_time, time);
CREATE INDEX idx_pm_metrics_hourly_path_time   ON pm_metrics_hourly (metric_path, time DESC);
CREATE INDEX idx_pm_metrics_hourly_device_time ON pm_metrics_hourly (device_oui, device_sn, time DESC);
CREATE INDEX idx_pm_metrics_hourly_object_ldn  ON pm_metrics_hourly (object_ldn) WHERE object_ldn IS NOT NULL;

SELECT create_hypertable('pm_metrics_hourly', 'time', chunk_time_interval => INTERVAL '7 days');

ALTER TABLE pm_metrics_hourly SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'device_oui,device_sn,metric_type,granularity'
);

SELECT add_compression_policy('pm_metrics_hourly', INTERVAL '14 days');
SELECT add_retention_policy('pm_metrics_hourly', INTERVAL '180 days');

-- ==================== pm_metrics_daily：普通表 ====================
CREATE TABLE pm_metrics_daily (
    device_oui    TEXT NOT NULL,
    device_sn     TEXT NOT NULL,
    metric_path   TEXT NOT NULL,
    metric_type   TEXT NOT NULL CHECK (metric_type IN ('counter', 'kpi')),
    metric_value  DOUBLE PRECISION NOT NULL,
    statis_type   TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum', 'avg', 'max', 'min', 'pct')),
    granularity   TEXT NOT NULL CHECK (granularity IN ('15min', 'hourly', 'daily', 'weekly', 'monthly')),
    time          TIMESTAMPTZ NOT NULL,
    start_time    TIMESTAMPTZ NOT NULL,
    end_time      TIMESTAMPTZ NOT NULL,
    ingest_time   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    object_ldn    TEXT,
    extra         JSONB,
    PRIMARY KEY (device_oui, device_sn, metric_path, granularity, end_time)
);
CREATE INDEX idx_pm_metrics_daily_path_time   ON pm_metrics_daily (metric_path, end_time DESC);
CREATE INDEX idx_pm_metrics_daily_device_time ON pm_metrics_daily (device_oui, device_sn, end_time DESC);
CREATE INDEX idx_pm_metrics_daily_object_ldn  ON pm_metrics_daily (object_ldn) WHERE object_ldn IS NOT NULL;

-- ==================== pm_metrics_weekly：普通表 ====================
CREATE TABLE pm_metrics_weekly (
    device_oui    TEXT NOT NULL,
    device_sn     TEXT NOT NULL,
    metric_path   TEXT NOT NULL,
    metric_type   TEXT NOT NULL CHECK (metric_type IN ('counter', 'kpi')),
    metric_value  DOUBLE PRECISION NOT NULL,
    statis_type   TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum', 'avg', 'max', 'min', 'pct')),
    granularity   TEXT NOT NULL CHECK (granularity IN ('15min', 'hourly', 'daily', 'weekly', 'monthly')),
    time          TIMESTAMPTZ NOT NULL,
    start_time    TIMESTAMPTZ NOT NULL,
    end_time      TIMESTAMPTZ NOT NULL,
    ingest_time   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    object_ldn    TEXT,
    extra         JSONB,
    PRIMARY KEY (device_oui, device_sn, metric_path, granularity, end_time)
);
CREATE INDEX idx_pm_metrics_weekly_path_time   ON pm_metrics_weekly (metric_path, end_time DESC);
CREATE INDEX idx_pm_metrics_weekly_device_time ON pm_metrics_weekly (device_oui, device_sn, end_time DESC);
CREATE INDEX idx_pm_metrics_weekly_object_ldn  ON pm_metrics_weekly (object_ldn) WHERE object_ldn IS NOT NULL;

-- ==================== pm_metrics_monthly：普通表 ====================
CREATE TABLE pm_metrics_monthly (
    device_oui    TEXT NOT NULL,
    device_sn     TEXT NOT NULL,
    metric_path   TEXT NOT NULL,
    metric_type   TEXT NOT NULL CHECK (metric_type IN ('counter', 'kpi')),
    metric_value  DOUBLE PRECISION NOT NULL,
    statis_type   TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum', 'avg', 'max', 'min', 'pct')),
    granularity   TEXT NOT NULL CHECK (granularity IN ('15min', 'hourly', 'daily', 'weekly', 'monthly')),
    time          TIMESTAMPTZ NOT NULL,
    start_time    TIMESTAMPTZ NOT NULL,
    end_time      TIMESTAMPTZ NOT NULL,
    ingest_time   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    object_ldn    TEXT,
    extra         JSONB,
    PRIMARY KEY (device_oui, device_sn, metric_path, granularity, end_time)
);
CREATE INDEX idx_pm_metrics_monthly_path_time   ON pm_metrics_monthly (metric_path, end_time DESC);
CREATE INDEX idx_pm_metrics_monthly_device_time ON pm_metrics_monthly (device_oui, device_sn, end_time DESC);
CREATE INDEX idx_pm_metrics_monthly_object_ldn  ON pm_metrics_monthly (object_ldn) WHERE object_ldn IS NOT NULL;

-- ==================== pm_group_metrics_hourly：hypertable（设备组维度）====================
CREATE TABLE pm_group_metrics_hourly (
    id              UUID NOT NULL DEFAULT gen_random_uuid(),
    device_group_id UUID NOT NULL,
    metric_path     TEXT NOT NULL,
    metric_type     TEXT NOT NULL CHECK (metric_type IN ('counter', 'kpi')),
    metric_value    DOUBLE PRECISION NOT NULL,
    statis_type     TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum', 'avg', 'max', 'min', 'pct')),
    granularity     TEXT NOT NULL CHECK (granularity IN ('15min', 'hourly', 'daily', 'weekly', 'monthly')),
    time            TIMESTAMPTZ NOT NULL,
    start_time      TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ NOT NULL,
    ingest_time     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    extra           JSONB,
    PRIMARY KEY (id, time)
);

CREATE UNIQUE INDEX uq_pm_group_metrics_hourly_natural
    ON pm_group_metrics_hourly (device_group_id, metric_path, granularity, end_time, time);
CREATE INDEX idx_pm_group_metrics_hourly_path_time ON pm_group_metrics_hourly (metric_path, time DESC);
CREATE INDEX idx_pm_group_metrics_hourly_group_time ON pm_group_metrics_hourly (device_group_id, time DESC);

SELECT create_hypertable('pm_group_metrics_hourly', 'time', chunk_time_interval => INTERVAL '7 days');

ALTER TABLE pm_group_metrics_hourly SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'device_group_id,metric_type,granularity'
);

SELECT add_compression_policy('pm_group_metrics_hourly', INTERVAL '14 days');
SELECT add_retention_policy('pm_group_metrics_hourly', INTERVAL '180 days');

-- ==================== pm_group_metrics_daily：普通表 ====================
CREATE TABLE pm_group_metrics_daily (
    device_group_id UUID NOT NULL,
    metric_path     TEXT NOT NULL,
    metric_type     TEXT NOT NULL CHECK (metric_type IN ('counter', 'kpi')),
    metric_value    DOUBLE PRECISION NOT NULL,
    statis_type     TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum', 'avg', 'max', 'min', 'pct')),
    granularity     TEXT NOT NULL CHECK (granularity IN ('15min', 'hourly', 'daily', 'weekly', 'monthly')),
    time            TIMESTAMPTZ NOT NULL,
    start_time      TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ NOT NULL,
    ingest_time     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    extra           JSONB,
    PRIMARY KEY (device_group_id, metric_path, granularity, end_time)
);
CREATE INDEX idx_pm_group_metrics_daily_path_time ON pm_group_metrics_daily (metric_path, end_time DESC);
CREATE INDEX idx_pm_group_metrics_daily_group_time ON pm_group_metrics_daily (device_group_id, end_time DESC);

-- ==================== pm_group_metrics_weekly：普通表 ====================
CREATE TABLE pm_group_metrics_weekly (
    device_group_id UUID NOT NULL,
    metric_path     TEXT NOT NULL,
    metric_type     TEXT NOT NULL CHECK (metric_type IN ('counter', 'kpi')),
    metric_value    DOUBLE PRECISION NOT NULL,
    statis_type     TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum', 'avg', 'max', 'min', 'pct')),
    granularity     TEXT NOT NULL CHECK (granularity IN ('15min', 'hourly', 'daily', 'weekly', 'monthly')),
    time            TIMESTAMPTZ NOT NULL,
    start_time      TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ NOT NULL,
    ingest_time     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    extra           JSONB,
    PRIMARY KEY (device_group_id, metric_path, granularity, end_time)
);
CREATE INDEX idx_pm_group_metrics_weekly_path_time ON pm_group_metrics_weekly (metric_path, end_time DESC);
CREATE INDEX idx_pm_group_metrics_weekly_group_time ON pm_group_metrics_weekly (device_group_id, end_time DESC);

-- ==================== pm_group_metrics_monthly：普通表 ====================
CREATE TABLE pm_group_metrics_monthly (
    device_group_id UUID NOT NULL,
    metric_path     TEXT NOT NULL,
    metric_type     TEXT NOT NULL CHECK (metric_type IN ('counter', 'kpi')),
    metric_value    DOUBLE PRECISION NOT NULL,
    statis_type     TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum', 'avg', 'max', 'min', 'pct')),
    granularity     TEXT NOT NULL CHECK (granularity IN ('15min', 'hourly', 'daily', 'weekly', 'monthly')),
    time            TIMESTAMPTZ NOT NULL,
    start_time      TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ NOT NULL,
    ingest_time     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    extra           JSONB,
    PRIMARY KEY (device_group_id, metric_path, granularity, end_time)
);
CREATE INDEX idx_pm_group_metrics_monthly_path_time ON pm_group_metrics_monthly (metric_path, end_time DESC);
CREATE INDEX idx_pm_group_metrics_monthly_group_time ON pm_group_metrics_monthly (device_group_id, end_time DESC);
-- +goose StatementEnd

-- ============================================================
-- 000162_extend_pm_tasks_and_create_adhoc_results
-- ============================================================
-- +goose StatementBegin
-- T-0164-P7 / G7 自定义聚合任务（oneshot + continuous）数据模型
--
-- 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.7
-- 实施 plan：docs/project/plan-T-0164-P7-adhoc-aggregation.md
--
-- 变更：
--   1. 扩 pm_tasks（已有，per-module，不进 G8）加 7 列承载 adhoc_aggregation 模式
--   2. 新建 pm_adhoc_aggregation_results hypertable 落任务结果
--
-- 设计要点：
--   - pm_tasks 既有 device_sns(JSONB)/kpi_codes(JSONB)/granularity/time_range/status/progress
--     对老 extraction/report/threshold-check 任务无影响（新列 NULL 即可）
--   - G7 adhoc 任务复用 status/progress；新增 task_subtype 区分（extraction vs adhoc_aggregation）
--   - metric_paths / granularities 用 TEXT[]（PG 原生数组，支持 ANY/&&/contains 算子，比 JSONB 更紧凑）
--   - window_start/window_end 单独列（adhoc 必填，老任务可空 — 老任务的 time_range JSONB 仍用）
--   - cron_expr 仅 continuous 模式填
--   - pm_adhoc_aggregation_results 是 hypertable：retention 365d（G2 sys_configs 后续可改）
--     drop_chunks 整块清；task 删除不级联（panel ↔ task 独立生命周期，G6 引用）

ALTER TABLE pm_tasks ADD COLUMN IF NOT EXISTS task_subtype  TEXT;
ALTER TABLE pm_tasks ADD COLUMN IF NOT EXISTS mode          TEXT;
ALTER TABLE pm_tasks ADD COLUMN IF NOT EXISTS cron_expr     TEXT;
ALTER TABLE pm_tasks ADD COLUMN IF NOT EXISTS metric_paths  TEXT[];
ALTER TABLE pm_tasks ADD COLUMN IF NOT EXISTS granularities TEXT[];
ALTER TABLE pm_tasks ADD COLUMN IF NOT EXISTS window_start  TIMESTAMPTZ;
ALTER TABLE pm_tasks ADD COLUMN IF NOT EXISTS window_end    TIMESTAMPTZ;

-- CHECK 约束（仅对新 adhoc 任务，老任务 task_subtype/mode IS NULL 不受约束）
ALTER TABLE pm_tasks DROP CONSTRAINT IF EXISTS chk_pm_tasks_mode;
ALTER TABLE pm_tasks ADD  CONSTRAINT chk_pm_tasks_mode
    CHECK (mode IS NULL OR mode IN ('oneshot', 'continuous'));

ALTER TABLE pm_tasks DROP CONSTRAINT IF EXISTS chk_pm_tasks_continuous_cron;
ALTER TABLE pm_tasks ADD  CONSTRAINT chk_pm_tasks_continuous_cron
    CHECK (mode IS DISTINCT FROM 'continuous' OR cron_expr IS NOT NULL);

CREATE INDEX IF NOT EXISTS idx_pm_tasks_subtype_status
    ON pm_tasks (task_subtype, status)
    WHERE task_subtype IS NOT NULL;

-- ==================== pm_adhoc_aggregation_results：hypertable ====================
CREATE TABLE pm_adhoc_aggregation_results (
    id           UUID NOT NULL DEFAULT gen_random_uuid(),
    task_id      UUID NOT NULL,
    device_oui   TEXT NOT NULL,
    device_sn    TEXT NOT NULL,
    metric_path  TEXT NOT NULL,
    metric_type  TEXT NOT NULL CHECK (metric_type IN ('counter', 'kpi')),
    metric_value DOUBLE PRECISION NOT NULL,
    statis_type  TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum', 'avg', 'max', 'min', 'pct')),
    granularity  TEXT NOT NULL CHECK (granularity IN ('15min', 'hourly', 'daily', 'weekly', 'monthly')),
    time         TIMESTAMPTZ NOT NULL,
    start_time   TIMESTAMPTZ NOT NULL,
    end_time     TIMESTAMPTZ NOT NULL,
    ingest_time  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    object_ldn   TEXT,
    extra        JSONB,
    PRIMARY KEY (id, time)
);

CREATE INDEX idx_pm_adhoc_results_task_time
    ON pm_adhoc_aggregation_results (task_id, time DESC);
CREATE INDEX idx_pm_adhoc_results_device
    ON pm_adhoc_aggregation_results (device_oui, device_sn, time DESC);

SELECT create_hypertable('pm_adhoc_aggregation_results', 'time', chunk_time_interval => INTERVAL '30 days');

ALTER TABLE pm_adhoc_aggregation_results SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'task_id,device_oui,device_sn,metric_type'
);

SELECT add_compression_policy('pm_adhoc_aggregation_results', INTERVAL '90 days');
-- G2 sys_configs 'pm.retention.adhoc_aggregation_days' 默认 365；retention policy 由 G2 reload 联动
SELECT add_retention_policy('pm_adhoc_aggregation_results', INTERVAL '365 days');
-- +goose StatementEnd

-- ============================================================
-- 000163_create_pm_dashboards_and_panels
-- ============================================================
-- +goose StatementBegin
-- T-0164-P6 / G6 PM 性能查看：Dashboard / Panel / 用户偏好
--
-- 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.6
-- 实施 plan：docs/project/plan-T-0164-P6-frontend-dashboard.md
--
-- 与现有 internal/dashboard（运营总览 — 设备/告警统计）解耦：
--   - 表名前缀 pm_ 区分（pm_dashboards / pm_panels / pm_user_dashboard_preferences）
--   - 业务域是"PM 性能查看"：用户可配置仪表盘 + 面板 + 对比 + 派生 + 分享
--
-- 设计要点：
--   - pm_dashboards.shared_with UUID[] 直接存被分享用户列表（小数量场景，单数组比关联表更轻）
--   - parent_dashboard_id FK 自身（ON DELETE SET NULL，源 dashboard 删除时 fork 仍可见但失去 parent 引用）
--   - technology lte/nr/gsm 顶层切换字段
--   - panels.dashboard_id CASCADE — 仪表盘删除时面板随删
--   - panels.adhoc_task_id 软引用 pm_tasks（FK 留 G7 task 表跨域；这里仅冗余 UUID 引用，避免循环依赖）
--   - kpi_card_layout per-user 独立持久化（与 dashboard 分离，避免每改 KPI 卡片就动 dashboard）

CREATE TABLE pm_dashboards (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                 TEXT NOT NULL,
    description          TEXT,
    owner_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    shared_with          UUID[] NOT NULL DEFAULT ARRAY[]::UUID[],
    parent_dashboard_id  UUID REFERENCES pm_dashboards(id) ON DELETE SET NULL,
    technology           TEXT NOT NULL CHECK (technology IN ('lte','nr','gsm')),
    layout               JSONB NOT NULL DEFAULT '{"panels":[]}'::JSONB,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pm_dashboards_owner ON pm_dashboards (owner_id);
CREATE INDEX idx_pm_dashboards_shared ON pm_dashboards USING GIN (shared_with);
CREATE INDEX idx_pm_dashboards_parent ON pm_dashboards (parent_dashboard_id) WHERE parent_dashboard_id IS NOT NULL;

CREATE TRIGGER trigger_pm_dashboards_updated_at
    BEFORE UPDATE ON pm_dashboards
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ==================== pm_panels ====================
CREATE TABLE pm_panels (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dashboard_id      UUID NOT NULL REFERENCES pm_dashboards(id) ON DELETE CASCADE,
    panel_type        TEXT NOT NULL CHECK (panel_type IN ('kpi_card','line_chart','bar_chart','table','gauge')),
    title             TEXT NOT NULL,
    metric_paths      TEXT[] NOT NULL,
    granularity       TEXT NOT NULL CHECK (granularity IN ('15min','hourly','daily','weekly','monthly')),
    dimension         TEXT NOT NULL CHECK (dimension IN ('device','device_group')),
    device_sns        TEXT[],
    device_group_ids  UUID[],
    time_range        JSONB NOT NULL DEFAULT '{}'::JSONB,
    compare_mode      TEXT CHECK (compare_mode IS NULL OR compare_mode IN ('same_window_other_devices','previous_window')),
    adhoc_task_id     UUID,   -- 软引用 pm_tasks(id) where task_subtype='adhoc_aggregation'（避免跨表 FK）
    config            JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pm_panels_dashboard ON pm_panels (dashboard_id);
CREATE INDEX idx_pm_panels_adhoc_task ON pm_panels (adhoc_task_id) WHERE adhoc_task_id IS NOT NULL;

CREATE TRIGGER trigger_pm_panels_updated_at
    BEFORE UPDATE ON pm_panels
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ==================== pm_user_dashboard_preferences ====================
-- per-user KPI 卡片 layout 持久化（与具体 dashboard 解耦，全局生效）
CREATE TABLE pm_user_dashboard_preferences (
    user_id          UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    kpi_card_layout  JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trigger_pm_user_dashboard_prefs_updated_at
    BEFORE UPDATE ON pm_user_dashboard_preferences
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- ============================================================
-- 000164_async_jobs_cron_state
-- ============================================================
-- +goose StatementBegin
-- T-0164 收尾 G8-Gap-1：cron 触发持久化 + 启动补跑
--
-- 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.8（G8 cron 错过补跑）
-- 实施 plan：docs/project/plan-T-0164-followup-gaps.md §G8-Gap-1
--
-- 用途：worker 重启 / 停机后，cron 调度器需要知道"上次成功触发到哪了"以补跑漏桶。
-- 没有这张表 = 停机期间所有 cron 触发完全丢失（robfig/cron 是内存调度器，无持久化）。
--
-- 设计要点：
--   - PK = job_type（每种 cron job 一行；多 worker 用 UPSERT 单 SQL CAS 防双触发）
--   - last_triggered_at = 上次成功 enqueue async_jobs 的时刻
--   - cron_expr = 当前生效的 cron expr（变更时也记录便于排查）
--   - updated_at 自动更新触发器

CREATE TABLE async_jobs_cron_state (
    job_type           TEXT PRIMARY KEY,
    cron_expr          TEXT NOT NULL,
    last_triggered_at  TIMESTAMPTZ NOT NULL,
    last_bucket_end    TIMESTAMPTZ,   -- 上次触发的 bucket 时间窗右界（便于按时间窗对齐查询）
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trigger_async_jobs_cron_state_updated_at
    BEFORE UPDATE ON async_jobs_cron_state
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- ============================================================
-- 000165_pm_user_prefs_by_technology
-- ============================================================
-- +goose StatementBegin
-- T-0164 收尾 G6-Gap-3 + G6-Gap-13：制式切换持久化 + KPI 卡片按制式分键
--
-- 设计文档：docs/design/pm-kpi-pipeline-improvements.md §7 G6 DoD
-- 实施 plan：docs/project/plan-T-0164-followup-gaps.md §G6-Gap-3, §G6-Gap-13
--
-- 变更：
--   把 pm_user_dashboard_preferences 主键从 user_id 改为 (user_id, technology)，
--   每个用户在每种制式下持有独立的 KPI 卡片 layout + 当前选中仪表盘 + 当前筛选条状态。
--
-- 新增列：
--   - technology TEXT NOT NULL CHECK (lte/nr/gsm)
--   - current_dashboard_id UUID — 切回该制式时自动打开的仪表盘
--   - shared_filters JSONB — 全局筛选条快照（时间窗 / 设备组 / 设备多选；G6-Gap-2 用）
--
-- 迁移已有数据：把现有行 LTE 默认。生产无数据所以是 nop。

-- 先 drop 老 PK + trigger，新建带 technology 的 PK
ALTER TABLE pm_user_dashboard_preferences DROP CONSTRAINT pm_user_dashboard_preferences_pkey;
ALTER TABLE pm_user_dashboard_preferences ADD COLUMN technology TEXT NOT NULL DEFAULT 'lte'
    CHECK (technology IN ('lte', 'nr', 'gsm'));
ALTER TABLE pm_user_dashboard_preferences ADD COLUMN current_dashboard_id UUID;
ALTER TABLE pm_user_dashboard_preferences ADD COLUMN shared_filters JSONB NOT NULL DEFAULT '{}'::JSONB;

ALTER TABLE pm_user_dashboard_preferences
    ADD CONSTRAINT pm_user_dashboard_preferences_pkey PRIMARY KEY (user_id, technology);

-- 软引用：dashboard 被删时 current_dashboard_id 走 SET NULL（不级联删 prefs 整行）
ALTER TABLE pm_user_dashboard_preferences
    ADD CONSTRAINT fk_pm_user_prefs_dashboard
    FOREIGN KEY (current_dashboard_id) REFERENCES pm_dashboards(id) ON DELETE SET NULL;

-- 既有 default 'lte' 占位完后，去掉 default 强制后续显式指定（避免误插）
ALTER TABLE pm_user_dashboard_preferences ALTER COLUMN technology DROP DEFAULT;
-- +goose StatementEnd

-- ============================================================
-- 000167_pm_tasks_last_fire_at
-- ============================================================
-- G7-Gap-9: continuous adhoc 任务 lossless 补跑
--
-- 增 last_fire_at 列追踪"上一次 cron 触发时刻"（由 ContinuousScheduler 写入），
-- 与 updated_at（worker 状态切换时刷新）解耦。
-- 调度器停机后启动：从 last_fire_at 开始一次一次推 Next(cron)，每个 worker 完成后
-- 下一个 sweep 继续推进一个 cron 窗口，直至追平 NOW()。
--
-- 列可为 NULL（老行无值），调度器对 NULL 走 fallback：使用 created_at 作起点。
ALTER TABLE pm_tasks ADD COLUMN IF NOT EXISTS last_fire_at TIMESTAMPTZ;

COMMENT ON COLUMN pm_tasks.last_fire_at IS 'G7 continuous 任务上次 cron 触发时刻；ContinuousScheduler 写入，worker 不动。';

-- ============================================================
-- 000168_pm_panels_topn_bignumber_builtin_flag
-- ============================================================
-- G6-Gap-5 + G6-Gap-4 (T-0164 收尾 P2 第 1 批)
--
-- 1. 扩展 pm_panels.panel_type CHECK 加 'topn' + 'big_number' 两种新类型
--    · topn        - 排行榜（前 N + 排序方向）
--    · big_number  - 数值大屏（单值放大显示 + 警戒色）
--
-- 2. pm_dashboards 增 is_builtin BOOLEAN 列，区分"系统内置 readonly"与用户自定义
--    · seed/000169 会插入 12 个内置 dashboard（3 制式 × 4 报表）
--    · 前端 DashboardList 按 is_builtin 分"系统内置 / 我的 / 来自分享"三组
--    · 前端 DashboardEditor 检测 is_builtin → hide 编辑 / 删除 按钮，仅允许"另存为派生"

ALTER TABLE pm_panels DROP CONSTRAINT IF EXISTS pm_panels_panel_type_check;
ALTER TABLE pm_panels ADD CONSTRAINT pm_panels_panel_type_check
    CHECK (panel_type IN ('kpi_card','line_chart','bar_chart','table','gauge','topn','big_number'));

ALTER TABLE pm_dashboards ADD COLUMN IF NOT EXISTS is_builtin BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_pm_dashboards_is_builtin
    ON pm_dashboards (is_builtin) WHERE is_builtin = TRUE;

-- ============================================================
-- 000170_pm_panels_granularities_array
-- ============================================================
-- G6-Gap-6 (T-0164 收尾 P2 第 2 批)
--
-- pm_panels.granularity TEXT → granularities TEXT[]
--
-- 设计：panel 可以同时绑定多个粒度（hourly / daily / weekly / monthly），
-- 前端 PanelHeader 显示 Tab 让用户切换粒度浏览同一指标的不同时间分辨率，
-- 不需要重新请求 panel CRUD 接口。
--
-- 迁移路径：
--   1) ADD COLUMN granularities TEXT[]
--   2) backfill granularities = ARRAY[granularity]（保留原单粒度行为）
--   3) DROP COLUMN granularity
--   4) NOT NULL + CHECK 元素白名单（15min/hourly/daily/weekly/monthly）+ array_length >= 1

ALTER TABLE pm_panels ADD COLUMN IF NOT EXISTS granularities TEXT[];

UPDATE pm_panels
   SET granularities = ARRAY[granularity]
 WHERE granularities IS NULL AND granularity IS NOT NULL;

ALTER TABLE pm_panels DROP COLUMN IF EXISTS granularity;

ALTER TABLE pm_panels ALTER COLUMN granularities SET NOT NULL;

ALTER TABLE pm_panels DROP CONSTRAINT IF EXISTS pm_panels_granularities_check;
ALTER TABLE pm_panels ADD CONSTRAINT pm_panels_granularities_check CHECK (
    granularities <@ ARRAY['15min','hourly','daily','weekly','monthly']::TEXT[]
    AND array_length(granularities, 1) >= 1
);

-- ============================================================
-- 000171_pm_metrics_natural_key_object_ldn
-- ============================================================
-- +goose StatementBegin
-- T-0164 BUG-6 修复：把 object_ldn 纳入 pm_metrics 自然键
--
-- 根因：
--   uq_pm_metrics_natural 原列清单 (device_oui, device_sn, metric_path, granularity, end_time, time)
--   不含 object_ldn。同一 PM 文件中同 metric_path 跨多个 cell（不同 object_ldn）的行
--   自然键完全相同 → INSERT ... ON CONFLICT DO UPDATE 同语句两次命中同一目标行 →
--   PG 抛 SQLSTATE 21000「ON CONFLICT DO UPDATE command cannot affect row a second time」。
--
-- 现象：批量 INSERT 全量 retry 3 次进 DLQ，pm_metrics 0 行。
--
-- 决策：
--   1. object_ldn 由 NULLABLE 改 NOT NULL DEFAULT ''
--      原因：UNIQUE 索引中两个 NULL 列值视为「不冲突」，相同 (oui, sn, path, gran, end_time, time, NULL)
--      可同时存在 → 破坏幂等。改 NOT NULL DEFAULT '' 后唯一性严格。
--   2. 重建 uq_pm_metrics_natural 索引，把 object_ldn 加到列末。
--   3. 聚合表 pm_metrics_hourly/daily/weekly/monthly 不需要改：
--      aggregator 通过 GROUP BY (oui, sn, metric_path, statis_type) 已把多 cell 合并为单行
--      （object_ldn 取 MIN 代表值），聚合层语义不需要按 cell 区分。

-- 1. 把现存的 NULL 填成 ''（DEFAULT 'd 不影响已有 NULL 行）
UPDATE pm_metrics SET object_ldn = '' WHERE object_ldn IS NULL;

-- 2. NOT NULL + DEFAULT
ALTER TABLE pm_metrics ALTER COLUMN object_ldn SET DEFAULT '';
ALTER TABLE pm_metrics ALTER COLUMN object_ldn SET NOT NULL;

-- 3. 重建 UNIQUE 索引，列末追加 object_ldn
--    TimescaleDB 要求 UNIQUE 含分区列 time（保留）
DROP INDEX IF EXISTS uq_pm_metrics_natural;
CREATE UNIQUE INDEX uq_pm_metrics_natural
    ON pm_metrics (device_oui, device_sn, metric_path, granularity, end_time, time, object_ldn);

-- 4. 部分索引 idx_pm_metrics_object_ldn 的 WHERE object_ldn IS NOT NULL 失效（永真）
--    改成无条件普通索引，并避免与新 UNIQUE 索引前缀重复浪费空间。
--    保留以兼容已有按 ldn 查询的代码（pm/counter/pg_repository.go 用 object_ldn 过滤 cellID）。
DROP INDEX IF EXISTS idx_pm_metrics_object_ldn;
CREATE INDEX idx_pm_metrics_object_ldn ON pm_metrics (object_ldn) WHERE object_ldn <> '';
-- +goose StatementEnd

-- +goose Down

-- ============================================================
-- 000171_pm_metrics_natural_key_object_ldn (Down)
-- ============================================================
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_pm_metrics_object_ldn;
CREATE INDEX idx_pm_metrics_object_ldn ON pm_metrics (object_ldn) WHERE object_ldn IS NOT NULL;

DROP INDEX IF EXISTS uq_pm_metrics_natural;
CREATE UNIQUE INDEX uq_pm_metrics_natural
    ON pm_metrics (device_oui, device_sn, metric_path, granularity, end_time, time);

ALTER TABLE pm_metrics ALTER COLUMN object_ldn DROP NOT NULL;
ALTER TABLE pm_metrics ALTER COLUMN object_ldn DROP DEFAULT;
-- +goose StatementEnd

-- ============================================================
-- 000170_pm_panels_granularities_array (Down)
-- ============================================================
ALTER TABLE pm_panels ADD COLUMN IF NOT EXISTS granularity TEXT;

UPDATE pm_panels
   SET granularity = COALESCE(granularities[1], 'hourly')
 WHERE granularity IS NULL;

ALTER TABLE pm_panels DROP CONSTRAINT IF EXISTS pm_panels_granularities_check;
ALTER TABLE pm_panels DROP COLUMN IF EXISTS granularities;

ALTER TABLE pm_panels ALTER COLUMN granularity SET NOT NULL;

-- ============================================================
-- 000168_pm_panels_topn_bignumber_builtin_flag (Down)
-- ============================================================
DROP INDEX IF EXISTS idx_pm_dashboards_is_builtin;
ALTER TABLE pm_dashboards DROP COLUMN IF EXISTS is_builtin;

ALTER TABLE pm_panels DROP CONSTRAINT IF EXISTS pm_panels_panel_type_check;
ALTER TABLE pm_panels ADD CONSTRAINT pm_panels_panel_type_check
    CHECK (panel_type IN ('kpi_card','line_chart','bar_chart','table','gauge'));

-- ============================================================
-- 000167_pm_tasks_last_fire_at (Down)
-- ============================================================
ALTER TABLE pm_tasks DROP COLUMN IF EXISTS last_fire_at;

-- ============================================================
-- 000165_pm_user_prefs_by_technology (Down)
-- ============================================================
-- +goose StatementBegin
ALTER TABLE pm_user_dashboard_preferences DROP CONSTRAINT IF EXISTS fk_pm_user_prefs_dashboard;
ALTER TABLE pm_user_dashboard_preferences DROP CONSTRAINT IF EXISTS pm_user_dashboard_preferences_pkey;
ALTER TABLE pm_user_dashboard_preferences DROP COLUMN IF EXISTS shared_filters;
ALTER TABLE pm_user_dashboard_preferences DROP COLUMN IF EXISTS current_dashboard_id;
ALTER TABLE pm_user_dashboard_preferences DROP COLUMN IF EXISTS technology;
ALTER TABLE pm_user_dashboard_preferences
    ADD CONSTRAINT pm_user_dashboard_preferences_pkey PRIMARY KEY (user_id);
-- +goose StatementEnd

-- ============================================================
-- 000164_async_jobs_cron_state (Down)
-- ============================================================
-- +goose StatementBegin
DROP TABLE IF EXISTS async_jobs_cron_state CASCADE;
-- +goose StatementEnd

-- ============================================================
-- 000163_create_pm_dashboards_and_panels (Down)
-- ============================================================
-- +goose StatementBegin
DROP TABLE IF EXISTS pm_user_dashboard_preferences CASCADE;
DROP TABLE IF EXISTS pm_panels CASCADE;
DROP TABLE IF EXISTS pm_dashboards CASCADE;
-- +goose StatementEnd

-- ============================================================
-- 000162_extend_pm_tasks_and_create_adhoc_results (Down)
-- ============================================================
-- +goose StatementBegin
SELECT remove_retention_policy('pm_adhoc_aggregation_results', if_exists => true);
SELECT remove_compression_policy('pm_adhoc_aggregation_results', if_exists => true);
DROP TABLE IF EXISTS pm_adhoc_aggregation_results CASCADE;

DROP INDEX IF EXISTS idx_pm_tasks_subtype_status;

ALTER TABLE pm_tasks DROP CONSTRAINT IF EXISTS chk_pm_tasks_continuous_cron;
ALTER TABLE pm_tasks DROP CONSTRAINT IF EXISTS chk_pm_tasks_mode;

ALTER TABLE pm_tasks DROP COLUMN IF EXISTS window_end;
ALTER TABLE pm_tasks DROP COLUMN IF EXISTS window_start;
ALTER TABLE pm_tasks DROP COLUMN IF EXISTS granularities;
ALTER TABLE pm_tasks DROP COLUMN IF EXISTS metric_paths;
ALTER TABLE pm_tasks DROP COLUMN IF EXISTS cron_expr;
ALTER TABLE pm_tasks DROP COLUMN IF EXISTS mode;
ALTER TABLE pm_tasks DROP COLUMN IF EXISTS task_subtype;
-- +goose StatementEnd

-- ============================================================
-- 000161_create_pm_aggregation_tables (Down)
-- ============================================================
-- +goose StatementBegin
SELECT remove_retention_policy('pm_metrics_hourly', if_exists => true);
SELECT remove_compression_policy('pm_metrics_hourly', if_exists => true);
SELECT remove_retention_policy('pm_group_metrics_hourly', if_exists => true);
SELECT remove_compression_policy('pm_group_metrics_hourly', if_exists => true);

DROP TABLE IF EXISTS pm_group_metrics_monthly CASCADE;
DROP TABLE IF EXISTS pm_group_metrics_weekly CASCADE;
DROP TABLE IF EXISTS pm_group_metrics_daily CASCADE;
DROP TABLE IF EXISTS pm_group_metrics_hourly CASCADE;

DROP TABLE IF EXISTS pm_metrics_monthly CASCADE;
DROP TABLE IF EXISTS pm_metrics_weekly CASCADE;
DROP TABLE IF EXISTS pm_metrics_daily CASCADE;
DROP TABLE IF EXISTS pm_metrics_hourly CASCADE;
-- +goose StatementEnd

-- ============================================================
-- 000160_create_pm_metrics_and_drop_legacy (Down)
-- ============================================================
-- +goose StatementBegin
-- 兜底：删 pm_metrics + 还原空 schema（旧表 0 行数据，无需 restore 业务数据）
-- 完整旧 schema 见 /tmp/pm-kpi-pre-G3-backup-20260523.sql
SELECT remove_retention_policy('pm_metrics', if_exists => true);
SELECT remove_compression_policy('pm_metrics', if_exists => true);
DROP TABLE IF EXISTS pm_metrics CASCADE;

-- 旧 pm_counters 空 schema 兜底
CREATE TABLE IF NOT EXISTS pm_counters (
    "time"         TIMESTAMPTZ NOT NULL,
    device_id      UUID NOT NULL,
    cell_id        VARCHAR(32) NOT NULL DEFAULT '',
    counter_group  VARCHAR(32) NOT NULL,
    counter_name   VARCHAR(64) NOT NULL,
    counter_value  DOUBLE PRECISION NOT NULL,
    granularity    SMALLINT NOT NULL DEFAULT 15
);
CREATE INDEX IF NOT EXISTS idx_pm_counters_device_time ON pm_counters (device_id, "time" DESC);
CREATE INDEX IF NOT EXISTS idx_pm_counters_group_name  ON pm_counters (counter_group, counter_name, "time" DESC);
CREATE INDEX IF NOT EXISTS pm_counters_time_idx        ON pm_counters ("time" DESC);

-- 旧 kpi_values 空 schema 兜底
CREATE TABLE IF NOT EXISTS kpi_values (
    "time"      TIMESTAMPTZ NOT NULL,
    device_id   UUID NOT NULL,
    cell_id     VARCHAR(32) NOT NULL DEFAULT '',
    kpi_name    VARCHAR(64) NOT NULL,
    kpi_value   DOUBLE PRECISION NOT NULL,
    carrier     VARCHAR(4) NOT NULL,
    technology  VARCHAR(3) NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_kpi_values_device_time ON kpi_values (device_id, "time" DESC);
CREATE INDEX IF NOT EXISTS idx_kpi_values_name        ON kpi_values (kpi_name, "time" DESC);
CREATE INDEX IF NOT EXISTS kpi_values_time_idx        ON kpi_values ("time" DESC);
-- +goose StatementEnd

-- ============================================================
-- 000159_async_jobs (Down)
-- ============================================================
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_async_jobs_updated_at ON async_jobs;
DROP INDEX IF EXISTS idx_async_jobs_type_created;
DROP INDEX IF EXISTS idx_async_jobs_zombie_check;
DROP INDEX IF EXISTS idx_async_jobs_pending_pickup;
DROP TABLE IF EXISTS async_jobs CASCADE;
-- +goose StatementEnd

