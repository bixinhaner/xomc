-- +goose Up
-- KPI 指标库:按"平台"存储一条可编辑描述(2026-06-02 用户决策:一级列表改为"一个平台一条")。
-- SummaryTab 一级列表每行 = (制式, 平台);一个 XML 文件即一个平台,描述按 (tech, platform) 维度。
-- 与 perf_indicators_<tech> / 公式表解耦:平台对应的 XML 被删/重载不强约束此表(描述为运维注记,
-- 留存即可;无对应平台的孤儿描述无害,后续可由运维清理)。
CREATE TABLE IF NOT EXISTS indicator_file_descriptions (
    tech        TEXT NOT NULL,
    platform    TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tech, platform)
);

-- +goose Down
DROP TABLE IF EXISTS indicator_file_descriptions;
