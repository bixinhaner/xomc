-- +goose Up
-- T-A: PM 多粒度物化保留「小区/PLMN」维度
--
-- 只动设备级四张汇总表（hourly/daily/weekly/monthly）：
--   1. object_ldn 由可空 text 收紧为 NOT NULL DEFAULT ''（与 15min 原始表 pm_metrics 对齐）；
--      收紧前先 backfill 历史 NULL → '' （§5.5.11 铁律 2）。
--   2. object_ldn 加入唯一键尾部：
--        - 日/周/月表：重建同名主键，尾部追加 object_ldn；
--        - 小时表：主键 (id, time) 不动；重建自然唯一索引 uq_pm_metrics_hourly_natural，尾部追加 object_ldn。
--
-- 设备组四张表 pm_group_metrics_*（决策 #1）与 15min 原始表 pm_metrics（已是 NOT NULL DEFAULT ''）一律不碰。

-- ── 1. backfill 历史 NULL → '' （收紧约束前必须先做）──────────────────────────
UPDATE public.pm_metrics_hourly  SET object_ldn = '' WHERE object_ldn IS NULL;
UPDATE public.pm_metrics_daily   SET object_ldn = '' WHERE object_ldn IS NULL;
UPDATE public.pm_metrics_weekly  SET object_ldn = '' WHERE object_ldn IS NULL;
UPDATE public.pm_metrics_monthly SET object_ldn = '' WHERE object_ldn IS NULL;

-- ── 2. object_ldn 收紧为 NOT NULL DEFAULT '' ─────────────────────────────────
ALTER TABLE public.pm_metrics_hourly  ALTER COLUMN object_ldn SET DEFAULT '';
ALTER TABLE public.pm_metrics_hourly  ALTER COLUMN object_ldn SET NOT NULL;
ALTER TABLE public.pm_metrics_daily   ALTER COLUMN object_ldn SET DEFAULT '';
ALTER TABLE public.pm_metrics_daily   ALTER COLUMN object_ldn SET NOT NULL;
ALTER TABLE public.pm_metrics_weekly  ALTER COLUMN object_ldn SET DEFAULT '';
ALTER TABLE public.pm_metrics_weekly  ALTER COLUMN object_ldn SET NOT NULL;
ALTER TABLE public.pm_metrics_monthly ALTER COLUMN object_ldn SET DEFAULT '';
ALTER TABLE public.pm_metrics_monthly ALTER COLUMN object_ldn SET NOT NULL;

-- ── 3. 日/周/月表：重建主键，尾部追加 object_ldn ───────────────────────────────
ALTER TABLE public.pm_metrics_daily   DROP CONSTRAINT IF EXISTS pm_metrics_daily_pkey;
ALTER TABLE public.pm_metrics_daily
    ADD CONSTRAINT pm_metrics_daily_pkey
    PRIMARY KEY (device_oui, device_sn, metric_path, granularity, end_time, object_ldn);

ALTER TABLE public.pm_metrics_weekly  DROP CONSTRAINT IF EXISTS pm_metrics_weekly_pkey;
ALTER TABLE public.pm_metrics_weekly
    ADD CONSTRAINT pm_metrics_weekly_pkey
    PRIMARY KEY (device_oui, device_sn, metric_path, granularity, end_time, object_ldn);

ALTER TABLE public.pm_metrics_monthly DROP CONSTRAINT IF EXISTS pm_metrics_monthly_pkey;
ALTER TABLE public.pm_metrics_monthly
    ADD CONSTRAINT pm_metrics_monthly_pkey
    PRIMARY KEY (device_oui, device_sn, metric_path, granularity, end_time, object_ldn);

-- ── 4. 小时表：主键 (id, time) 不动；重建自然唯一索引，尾部追加 object_ldn ─────────
DROP INDEX IF EXISTS uq_pm_metrics_hourly_natural;
CREATE UNIQUE INDEX uq_pm_metrics_hourly_natural
    ON public.pm_metrics_hourly
    USING btree (device_oui, device_sn, metric_path, granularity, end_time, "time", object_ldn);

-- +goose Down
-- 完整反向回退：恢复原 PK / 唯一索引为不含 object_ldn 的原样，object_ldn 改回可空、去掉 DEFAULT。

-- ── 4'. 小时表：恢复不含 object_ldn 的自然唯一索引 ─────────────────────────────
DROP INDEX IF EXISTS uq_pm_metrics_hourly_natural;
CREATE UNIQUE INDEX uq_pm_metrics_hourly_natural
    ON public.pm_metrics_hourly
    USING btree (device_oui, device_sn, metric_path, granularity, end_time, "time");

-- ── 3'. 日/周/月表：恢复不含 object_ldn 的原主键 ──────────────────────────────
ALTER TABLE public.pm_metrics_monthly DROP CONSTRAINT IF EXISTS pm_metrics_monthly_pkey;
ALTER TABLE public.pm_metrics_monthly
    ADD CONSTRAINT pm_metrics_monthly_pkey
    PRIMARY KEY (device_oui, device_sn, metric_path, granularity, end_time);

ALTER TABLE public.pm_metrics_weekly  DROP CONSTRAINT IF EXISTS pm_metrics_weekly_pkey;
ALTER TABLE public.pm_metrics_weekly
    ADD CONSTRAINT pm_metrics_weekly_pkey
    PRIMARY KEY (device_oui, device_sn, metric_path, granularity, end_time);

ALTER TABLE public.pm_metrics_daily   DROP CONSTRAINT IF EXISTS pm_metrics_daily_pkey;
ALTER TABLE public.pm_metrics_daily
    ADD CONSTRAINT pm_metrics_daily_pkey
    PRIMARY KEY (device_oui, device_sn, metric_path, granularity, end_time);

-- ── 2'. object_ldn 改回可空、去掉 DEFAULT ────────────────────────────────────
ALTER TABLE public.pm_metrics_monthly ALTER COLUMN object_ldn DROP NOT NULL;
ALTER TABLE public.pm_metrics_monthly ALTER COLUMN object_ldn DROP DEFAULT;
ALTER TABLE public.pm_metrics_weekly  ALTER COLUMN object_ldn DROP NOT NULL;
ALTER TABLE public.pm_metrics_weekly  ALTER COLUMN object_ldn DROP DEFAULT;
ALTER TABLE public.pm_metrics_daily   ALTER COLUMN object_ldn DROP NOT NULL;
ALTER TABLE public.pm_metrics_daily   ALTER COLUMN object_ldn DROP DEFAULT;
ALTER TABLE public.pm_metrics_hourly  ALTER COLUMN object_ldn DROP NOT NULL;
ALTER TABLE public.pm_metrics_hourly  ALTER COLUMN object_ldn DROP DEFAULT;
