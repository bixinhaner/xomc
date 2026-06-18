-- +goose Up
-- #528 P1 水位基建：PM 持续聚合「完成水位」表。
--
-- 背景：下游持续聚合任务要消费「上游各级卷数据已成功处理完到哪一格」，
-- 但当前没有任何地方记录这个完成进度，下游只能靠「当前时刻 − 2 格」的固定时间偏移去猜，
-- 导致数据新鲜度白白落后一整个粒度。本表把「完成水位」这个事实显式记录下来：
-- 上游每级卷数据每**成功处理完一格**后，把该 (粒度, 层级) 的完成水位推进到本格起点。
--
-- 键 = (granularity, level)：
--   - granularity：聚合粒度（hourly / daily / weekly / monthly）
--   - level：完成层级，'device'（设备级）或 'group'（设备组级，链式产出，完成时刻更晚）
--
-- completed_bucket_start 语义是「已处理到哪一格起点（含）」，**不是「有数据」**——
-- 无论该格有无数据行都推进（空格也照样推进），否则下游会卡死在真正空的格上。
-- 水位只进不退（UPSERT 取 max），并发/重复触发幂等。

CREATE TABLE IF NOT EXISTS pm_completion_watermarks (
    granularity            text        NOT NULL,
    level                  text        NOT NULL,
    completed_bucket_start timestamptz NOT NULL,
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pm_completion_watermarks_pkey PRIMARY KEY (granularity, level),
    CONSTRAINT pm_completion_watermarks_granularity_check
        CHECK (granularity = ANY (ARRAY['hourly'::text, 'daily'::text, 'weekly'::text, 'monthly'::text])),
    CONSTRAINT pm_completion_watermarks_level_check
        CHECK (level = ANY (ARRAY['device'::text, 'group'::text]))
);

COMMENT ON TABLE pm_completion_watermarks IS
    '#528 PM 持续聚合完成水位：按 (粒度, 层级) 记录上游卷数据已成功处理完到哪一格起点（语义为「已处理」非「有数据」，空格也推进；只进不退）。';
COMMENT ON COLUMN pm_completion_watermarks.granularity IS '聚合粒度：hourly/daily/weekly/monthly。';
COMMENT ON COLUMN pm_completion_watermarks.level IS '完成层级：device（设备级）/ group（设备组级，链式产出，完成更晚）。';
COMMENT ON COLUMN pm_completion_watermarks.completed_bucket_start IS '已处理完成到的格起点（含），桶头时间戳；下游取「≤ 此值」的下一格。';

-- +goose Down
DROP TABLE IF EXISTS pm_completion_watermarks;
