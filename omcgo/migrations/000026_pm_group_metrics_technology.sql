-- +goose Up
-- 自定义聚合-设备组维度 × 制式过滤治本（B 方案：设备组快表按「组 × 制式」拆行）。
--
-- 四张设备组快表 pm_group_metrics_{hourly,daily,weekly,monthly}：
--   1. 加列 technology（与 devices.technology 同口径：lte/nr/gsm，varchar(3) NOT NULL）。
--   2. 唯一键 / 主键尾部追加 technology——否则同组同指标不同制式两行互相 UPSERT 覆盖只剩一行。
--        - 日/周/月表：重建同名主键，尾部追加 technology；
--        - 小时表：主键 (id, time) 不动（hypertable 要求含分区列），重建自然唯一索引 uq_pm_group_metrics_hourly_natural，尾部追加 technology。
--
-- 不管老数据（设计 §9）：快表是运行时由后台聚合重建的，先 TRUNCATE 清一次再加 technology 列，不做 backfill。
-- 这四张表无被 FK 引用，可独立 TRUNCATE。
-- ⚠ pm_group_metrics_hourly 是启用了 columnstore（压缩）的 hypertable，TimescaleDB 禁止给它直接加
--   「NOT NULL 无默认值」列（SQLSTATE 0A000）。故 technology 改用「先加可空列 → SET NOT NULL」两步
--   （空表瞬间完成，与既有 000020 在 columnstore 表 pm_metrics_hourly 上 SET NOT NULL 同范式）；
--   daily/weekly/monthly 是普通表，同样两步保持一致。

-- ── 1. 清空四张快表（不 backfill 老数据）─────────────────────────────────────
TRUNCATE public.pm_group_metrics_hourly;
TRUNCATE public.pm_group_metrics_daily;
TRUNCATE public.pm_group_metrics_weekly;
TRUNCATE public.pm_group_metrics_monthly;

-- ── 2. 加 technology 列（可空 → SET NOT NULL 两步，绕开 columnstore hypertable 限制；幂等 IF NOT EXISTS）──
ALTER TABLE public.pm_group_metrics_hourly  ADD COLUMN IF NOT EXISTS technology varchar(3);
ALTER TABLE public.pm_group_metrics_daily   ADD COLUMN IF NOT EXISTS technology varchar(3);
ALTER TABLE public.pm_group_metrics_weekly  ADD COLUMN IF NOT EXISTS technology varchar(3);
ALTER TABLE public.pm_group_metrics_monthly ADD COLUMN IF NOT EXISTS technology varchar(3);

ALTER TABLE public.pm_group_metrics_hourly  ALTER COLUMN technology SET NOT NULL;
ALTER TABLE public.pm_group_metrics_daily   ALTER COLUMN technology SET NOT NULL;
ALTER TABLE public.pm_group_metrics_weekly  ALTER COLUMN technology SET NOT NULL;
ALTER TABLE public.pm_group_metrics_monthly ALTER COLUMN technology SET NOT NULL;

-- ── 3. 日/周/月表：重建主键，尾部追加 technology ───────────────────────────────
ALTER TABLE public.pm_group_metrics_daily   DROP CONSTRAINT IF EXISTS pm_group_metrics_daily_pkey;
ALTER TABLE public.pm_group_metrics_daily
    ADD CONSTRAINT pm_group_metrics_daily_pkey
    PRIMARY KEY (device_group_id, metric_path, granularity, end_time, technology);

ALTER TABLE public.pm_group_metrics_weekly  DROP CONSTRAINT IF EXISTS pm_group_metrics_weekly_pkey;
ALTER TABLE public.pm_group_metrics_weekly
    ADD CONSTRAINT pm_group_metrics_weekly_pkey
    PRIMARY KEY (device_group_id, metric_path, granularity, end_time, technology);

ALTER TABLE public.pm_group_metrics_monthly DROP CONSTRAINT IF EXISTS pm_group_metrics_monthly_pkey;
ALTER TABLE public.pm_group_metrics_monthly
    ADD CONSTRAINT pm_group_metrics_monthly_pkey
    PRIMARY KEY (device_group_id, metric_path, granularity, end_time, technology);

-- ── 4. 小时表：主键 (id, time) 不动；重建自然唯一索引，尾部追加 technology ────────
DROP INDEX IF EXISTS uq_pm_group_metrics_hourly_natural;
CREATE UNIQUE INDEX uq_pm_group_metrics_hourly_natural
    ON public.pm_group_metrics_hourly
    USING btree (device_group_id, metric_path, granularity, end_time, "time", technology);

-- +goose Down
-- 完整反向回退：恢复不含 technology 的原键 + DROP COLUMN。回退前同样 TRUNCATE（new 行带 technology，回退到旧键会冲突）。

TRUNCATE public.pm_group_metrics_hourly;
TRUNCATE public.pm_group_metrics_daily;
TRUNCATE public.pm_group_metrics_weekly;
TRUNCATE public.pm_group_metrics_monthly;

-- ── 4'. 小时表：恢复不含 technology 的自然唯一索引 ────────────────────────────
DROP INDEX IF EXISTS uq_pm_group_metrics_hourly_natural;
CREATE UNIQUE INDEX uq_pm_group_metrics_hourly_natural
    ON public.pm_group_metrics_hourly
    USING btree (device_group_id, metric_path, granularity, end_time, "time");

-- ── 3'. 日/周/月表：恢复不含 technology 的原主键 ──────────────────────────────
ALTER TABLE public.pm_group_metrics_monthly DROP CONSTRAINT IF EXISTS pm_group_metrics_monthly_pkey;
ALTER TABLE public.pm_group_metrics_monthly
    ADD CONSTRAINT pm_group_metrics_monthly_pkey
    PRIMARY KEY (device_group_id, metric_path, granularity, end_time);

ALTER TABLE public.pm_group_metrics_weekly  DROP CONSTRAINT IF EXISTS pm_group_metrics_weekly_pkey;
ALTER TABLE public.pm_group_metrics_weekly
    ADD CONSTRAINT pm_group_metrics_weekly_pkey
    PRIMARY KEY (device_group_id, metric_path, granularity, end_time);

ALTER TABLE public.pm_group_metrics_daily   DROP CONSTRAINT IF EXISTS pm_group_metrics_daily_pkey;
ALTER TABLE public.pm_group_metrics_daily
    ADD CONSTRAINT pm_group_metrics_daily_pkey
    PRIMARY KEY (device_group_id, metric_path, granularity, end_time);

-- ── 2'. DROP technology 列 ───────────────────────────────────────────────────
ALTER TABLE public.pm_group_metrics_hourly  DROP COLUMN IF EXISTS technology;
ALTER TABLE public.pm_group_metrics_daily   DROP COLUMN IF EXISTS technology;
ALTER TABLE public.pm_group_metrics_weekly  DROP COLUMN IF EXISTS technology;
ALTER TABLE public.pm_group_metrics_monthly DROP COLUMN IF EXISTS technology;
