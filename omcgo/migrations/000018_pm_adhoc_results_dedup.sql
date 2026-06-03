-- +goose Up
-- T-0194 Part A：PM 聚合结果去重根治。
-- pm_adhoc_aggregation_results 原仅 (id, time) 主键 + 随机 UUID id，业务组合无任何唯一兜底，
-- 写入裸 INSERT 无 ON CONFLICT，导致同「任务×指标×粒度×对象×设备×时间桶」被写多份。
-- 本迁移：① 先按 8 列业务键清存量重复行（每组保留一行）；② 建业务唯一表达式索引。
-- 可空列统一 COALESCE 兜空串，否则 PG 的 NULL != NULL 让空值行逃过唯一约束。

-- ① 清存量重复：按 8 列业务键分组（可空列 COALESCE 兜空串），每组保留 ctid 最大（最新）的一行，
--    DELETE 其余。dev 库 6-3 刚重建无存量重复，此处对其为空操作（幂等）。
-- +goose StatementBegin
DO $$
BEGIN
    DELETE FROM pm_adhoc_aggregation_results t
    USING (
        SELECT ctid,
               row_number() OVER (
                   PARTITION BY
                       task_id, granularity, metric_path,
                       COALESCE(device_oui, ''), COALESCE(device_sn, ''),
                       COALESCE(product_id::text, ''), COALESCE(object_ldn, ''),
                       "time"
                   ORDER BY ctid DESC
               ) AS rn
        FROM pm_adhoc_aggregation_results
    ) dup
    WHERE t.ctid = dup.ctid
      AND dup.rn > 1;
END $$;
-- +goose StatementEnd

-- ② 建业务唯一表达式索引。表达式文本必须与 InsertResults 的 ON CONFLICT 目标逐字一致。
CREATE UNIQUE INDEX IF NOT EXISTS uq_pm_adhoc_results_business
    ON pm_adhoc_aggregation_results (
        task_id, granularity, metric_path,
        COALESCE(device_oui, ''), COALESCE(device_sn, ''),
        COALESCE(product_id::text, ''), COALESCE(object_ldn, ''),
        "time"
    );

-- +goose Down
-- 存量 DELETE 不可逆，Down 不回填（符合常规）；仅删唯一索引。
DROP INDEX IF EXISTS uq_pm_adhoc_results_business;
