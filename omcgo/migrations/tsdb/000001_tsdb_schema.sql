-- +goose Up
-- =====================================================================================
-- TimescaleDB 时序库 schema baseline（KPI/时序库物理分离，goose 版本表 goose_db_version_tsdb）
--
-- 本文件建立第二个 TimescaleDB 实例（TsPool / postgres-tsdb）的全部时序对象，承载从主库
-- 迁出的表（§契约 1A，含 pm_files / mr_files 两张文件元数据表）+ 7 张影子维度表（§契约 1B，
-- 从主库同步供本库 JOIN，含 mr_customize_task_dim）+ 1 个告警效率物化视图（原在 seed，建在
-- alarms_history 上）。注：mr_files 随「MR 也记录到时序库」由主库迁来，与 mr_records 同库。
--
-- 15 张表的「最终形态」= 主库 000001 原始定义 叠加 这些增量的净效果：
--   - pm_metrics：去掉随机 uuid 主键 pm_metrics_pkey（000044）+ 去掉自然键唯一索引
--     uq_pm_metrics_natural（000042）+ 去掉 idx_pm_metrics_ingest_time / idx_pm_metrics_object_ldn
--     （000043）+ 带 insert-triggered autovacuum reloptions（000044）；chunk 间隔 4 小时（000045，修 B0）。
--   - pm_metrics_{hourly,daily,weekly,monthly}：object_ldn 收紧 NOT NULL DEFAULT ''、键尾追加
--     object_ldn（000020）。
--   - pm_group_metrics_{hourly,daily,weekly,monthly}：带 technology NOT NULL 列、键尾追加 technology（000026）。
--   - pm_adhoc_aggregation_results：带可空 product_id 列（000005）+ 业务去重唯一索引
--     uq_pm_adhoc_results_business（000018）。
--
-- 超表 chunk 间隔 / 压缩 / 保留参数还原自 seed 的 _timescaledb_catalog（dimension.interval_length /
-- bgw_job policy_compression/policy_retention / compression_settings），µs→人类可读换算见各处注释。
--
-- 2026-07-20 consolidated baseline：合并原 000002（alarms_history retention 固定为每天
-- 01:08 Asia/Shanghai）+ 000003（删除与 Go 侧 internal/pm/aggregator 功能重复、全代码库无
-- 查询引用的废弃 pm_metrics_hourly_cagg 连续聚合视图，压测实测其刷新与 autovacuum 抢 IO/Buffer
-- 是「数据进得来、算不出来」的根因）。此后本 baseline 不再建 pm_metrics_hourly_cagg。
-- =====================================================================================

CREATE EXTENSION IF NOT EXISTS timescaledb;

-- =====================================================================================
-- 1. 15 张时序/PM 表（最终形态）
-- =====================================================================================

-- ── alarms_history（超表：time 7d chunk，retention 365d，无压缩）─────────────────────
CREATE TABLE public.alarms_history (
    "time" timestamp with time zone NOT NULL,
    alarm_id uuid NOT NULL,
    device_id uuid NOT NULL,
    device_sn character varying(64) NOT NULL,
    carrier character varying(4) NOT NULL,
    severity smallint NOT NULL,
    alarm_type character varying(64),
    alarm_identifier character varying(64) NOT NULL,
    description text,
    status character varying(16) NOT NULL,
    raised_at timestamp with time zone NOT NULL,
    acknowledged_at timestamp with time zone,
    cleared_at timestamp with time zone,
    device_name character varying(128),
    technology character varying(16),
    alarm_source character varying(64),
    event_type character varying(64),
    network_location text,
    explicit_cause text,
    ack_count integer DEFAULT 0 NOT NULL,
    acknowledged_by character varying(128),
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    ack_note text DEFAULT ''::text,
    cleared_by character varying(128),
    clear_note text DEFAULT ''::text,
    probable_cause text DEFAULT ''::text NOT NULL,
    additional_info jsonb DEFAULT '{}'::jsonb
);

-- ── mr_records（超表：time 1d chunk，compress 7d，retention 90d）──────────────────────
CREATE TABLE public.mr_records (
    "time" timestamp with time zone NOT NULL,
    file_id uuid NOT NULL,
    device_id uuid NOT NULL,
    cell_id character varying(32) DEFAULT ''::character varying NOT NULL,
    mr_type character varying(8) NOT NULL,
    measurement_data jsonb DEFAULT '{}'::jsonb NOT NULL
);

-- ── trace_messages（超表：captured_at 1d chunk，retention 3d，无压缩）──────────────────
CREATE TABLE public.trace_messages (
    captured_at timestamp with time zone NOT NULL,
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_id uuid NOT NULL,
    device_sn character varying(64) NOT NULL,
    direction character varying(8) NOT NULL,
    rpc_method character varying(64),
    cwmp_id character varying(64),
    session_id character varying(64),
    http_status smallint,
    payload_size_bytes integer DEFAULT 0 NOT NULL,
    payload_inline text,
    payload_object_key character varying(512),
    CONSTRAINT trace_messages_direction_check CHECK (((direction)::text = ANY ((ARRAY['in'::character varying, 'out'::character varying])::text[])))
);

-- ── pm_files（普通表；pm_metrics 同库保住 copy_ingest 单事务原子性）────────────────────
-- raw_compressed：原始 PM XML 是否实际以 gzip 存储；worker 入库后一次性压缩成功或已 gzip 时置真。
CREATE TABLE public.pm_files (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_id uuid NOT NULL,
    device_sn character varying(128) NOT NULL,
    carrier character varying(16) NOT NULL,
    technology character varying(16) NOT NULL,
    file_name character varying(512) NOT NULL,
    file_size bigint DEFAULT 0,
    collect_time timestamp with time zone,
	measurement_start timestamp with time zone,
	measurement_end timestamp with time zone,
    minio_path character varying(1024) NOT NULL,
    content_sha256 bytea,
    parsed boolean DEFAULT false,
    parsed_at timestamp with time zone,
    counter_count integer DEFAULT 0,
    created_at timestamp with time zone DEFAULT now(),
    raw_compressed boolean DEFAULT false NOT NULL,
    CONSTRAINT pm_files_pkey PRIMARY KEY (id),
    CONSTRAINT uq_pm_files_device_filename UNIQUE (device_sn, file_name),
    CONSTRAINT uq_pm_files_device_content UNIQUE (device_sn, content_sha256)
);
CREATE INDEX idx_pm_files_created ON public.pm_files USING btree (created_at DESC);
CREATE INDEX idx_pm_files_device ON public.pm_files USING btree (device_id);
CREATE INDEX idx_pm_files_measurement_slot
    ON public.pm_files (measurement_end, technology, carrier, device_id)
    WHERE measurement_end IS NOT NULL AND parsed = true;
-- 部分索引：只索引未压行，保留给 deprecated Sweeper/离线工具按 raw_compressed=false AND
-- created_at<cutoff 查询。
CREATE INDEX idx_pm_files_uncompressed ON public.pm_files USING btree (created_at) WHERE (raw_compressed = false);

CREATE TABLE public.pm_slot_health (
    slot_start timestamptz NOT NULL,
    slot_end timestamptz NOT NULL,
    technology varchar(16) NOT NULL,
    carrier varchar(16) NOT NULL,
    expected_devices bigint NOT NULL,
    received_devices bigint NOT NULL,
    coverage_ratio double precision NOT NULL,
    expected_snapshot_version text NOT NULL DEFAULT '',
    evaluated_at timestamptz NOT NULL,
    status varchar(32) NOT NULL,
    PRIMARY KEY (slot_end, technology, carrier),
    CONSTRAINT chk_pm_slot_health_window CHECK (slot_end > slot_start),
    CONSTRAINT chk_pm_slot_health_counts CHECK (
        expected_devices >= 0 AND received_devices >= 0
    ),
    CONSTRAINT chk_pm_slot_health_coverage CHECK (
        coverage_ratio >= 0 AND coverage_ratio <= 1
    ),
    CONSTRAINT chk_pm_slot_health_status CHECK (
        status IN ('complete', 'partial', 'missing', 'bootstrap_ignored')
    )
);
CREATE INDEX idx_pm_slot_health_latest
    ON public.pm_slot_health (technology, carrier, slot_end DESC);

-- ── mr_files（普通表；与 mr_records 同库 —— MR 文件元数据落时序库，不再放主库）────────────
-- 原在主库（PgPool），随「MR 也记录到时序库」迁来 TsPool：与 pm_files / mr_records 一致。
-- 无外键（device 维度经 device_dim 影子表 JOIN，订阅状态经 mr_customize_task_dim 影子表）。
-- raw_compressed 语义同 pm_files：对象实际以 gzip 存储时置真。
CREATE TABLE public.mr_files (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_id uuid NOT NULL,
    device_sn character varying(64) NOT NULL,
    carrier character varying(4) NOT NULL,
    mr_type character varying(8) NOT NULL,
    file_name character varying(256) NOT NULL,
    file_size bigint DEFAULT 0 NOT NULL,
    collect_time timestamp with time zone NOT NULL,
    minio_path character varying(512) NOT NULL,
    parsed boolean DEFAULT false NOT NULL,
    parsed_at timestamp with time zone,
    record_count integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    raw_compressed boolean DEFAULT false NOT NULL,
    CONSTRAINT mr_files_pkey PRIMARY KEY (id)
);
CREATE UNIQUE INDEX uq_mr_files_sn_filename ON public.mr_files USING btree (device_sn, file_name);
CREATE INDEX idx_mr_files_carrier ON public.mr_files USING btree (carrier);
CREATE INDEX idx_mr_files_device ON public.mr_files USING btree (device_id, collect_time DESC);
CREATE INDEX idx_mr_files_type ON public.mr_files USING btree (mr_type);
CREATE INDEX idx_mr_files_uncompressed ON public.mr_files USING btree (created_at) WHERE (raw_compressed = false);

-- ── PM 稀疏值平面 + 采集语义平面 ───────────────────────────────────────────
CREATE TABLE public.pm_metric_dictionary (
    metric_id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    metric_path text NOT NULL UNIQUE,
    report_key text,
    metric_name text,
    unit text,
    metric_type text NOT NULL DEFAULT 'counter',
    statis_type text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pm_metric_dictionary_type_check CHECK (metric_type IN ('counter','kpi')),
    CONSTRAINT pm_metric_dictionary_statis_check CHECK (statis_type IS NULL OR statis_type IN ('sum','avg','max','min','pct'))
);

CREATE TABLE public.pm_metric_sets (
    metric_set_id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    product_key text NOT NULL,
    counter_group text NOT NULL,
    content_hash bytea NOT NULL,
    metric_ids bigint[] NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_pm_metric_sets_content UNIQUE (product_key, counter_group, content_hash)
);

CREATE TABLE public.pm_ingest_batches (
    ingest_batch_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_file_id uuid NOT NULL UNIQUE,
    status text NOT NULL DEFAULT 'building',
    started_at timestamptz NOT NULL DEFAULT now(),
    committed_at timestamptz,
    CONSTRAINT pm_ingest_batches_status_check CHECK (status IN ('building','committed','failed'))
);

CREATE TABLE public.pm_measurement_anchors (
    "time" timestamptz NOT NULL,
    anchor_id bigint GENERATED BY DEFAULT AS IDENTITY NOT NULL,
    device_dim_id uuid NOT NULL,
    object_type smallint NOT NULL DEFAULT 0,
    object_ldn text NOT NULL DEFAULT '',
    counter_group text NOT NULL,
    metric_set_id bigint NOT NULL,
    source_file_id uuid,
    granularity text NOT NULL,
    ingest_batch_id uuid,
    start_time timestamptz NOT NULL,
    end_time timestamptz NOT NULL,
    PRIMARY KEY ("time", anchor_id)
);
CREATE INDEX idx_pm_anchors_device_time ON public.pm_measurement_anchors (device_dim_id, "time" DESC, anchor_id);
CREATE INDEX idx_pm_anchors_object_time ON public.pm_measurement_anchors (object_ldn, "time" DESC);
-- 设备性能查看默认 15min 查询在稀疏模型中先按设备与时间收窄锚点；
-- metric_path / metric_type 随后通过指标字典和 metric_set 解析，不能再索引兼容视图 pm_metrics。
CREATE INDEX idx_pm_anchors_15min_device_time_default
    ON public.pm_measurement_anchors (device_dim_id, "time" DESC, anchor_id)
    WHERE granularity = '15min';

CREATE TABLE public.pm_metric_values (
    "time" timestamptz NOT NULL,
    anchor_id bigint NOT NULL,
    metric_id bigint NOT NULL,
    metric_value double precision NOT NULL,
    CONSTRAINT uq_pm_metric_values UNIQUE ("time", anchor_id, metric_id)
);
CREATE INDEX idx_pm_metric_values_metric_time ON public.pm_metric_values (metric_id, "time" DESC, anchor_id);

-- 小时桶按 building/active/superseded/failed 整桶发布。
CREATE TABLE public.pm_hourly_bucket_versions (
    bucket_version bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    bucket_start timestamptz NOT NULL,
    bucket_end timestamptz NOT NULL,
    status text NOT NULL DEFAULT 'building',
    dirty boolean NOT NULL DEFAULT false,
    expected_batches integer NOT NULL DEFAULT 0,
    completed_batches integer NOT NULL DEFAULT 0,
    anchor_count bigint NOT NULL DEFAULT 0,
    value_count bigint NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,
    CONSTRAINT pm_hourly_version_status_check CHECK (status IN ('building','active','superseded','failed'))
);
CREATE UNIQUE INDEX uq_pm_hourly_active_bucket ON public.pm_hourly_bucket_versions (bucket_start) WHERE status = 'active';

CREATE TABLE public.pm_hourly_rollup_batches (
    bucket_version bigint NOT NULL,
    batch_no integer NOT NULL,
    status text NOT NULL DEFAULT 'building',
    device_count integer NOT NULL DEFAULT 0,
    anchor_count bigint NOT NULL DEFAULT 0,
    value_count bigint NOT NULL DEFAULT 0,
    started_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    PRIMARY KEY (bucket_version, batch_no),
    CONSTRAINT pm_hourly_batch_status_check CHECK (status IN ('building','completed','failed'))
);

CREATE TABLE public.pm_hourly_anchors (
    "time" timestamptz NOT NULL,
    anchor_id bigint GENERATED BY DEFAULT AS IDENTITY NOT NULL,
    bucket_version bigint NOT NULL,
    device_dim_id uuid NOT NULL,
    object_type smallint NOT NULL DEFAULT 0,
    object_ldn text NOT NULL DEFAULT '',
    counter_group text NOT NULL,
    metric_set_id bigint NOT NULL,
    granularity text NOT NULL DEFAULT 'hourly',
    start_time timestamptz NOT NULL,
    end_time timestamptz NOT NULL,
    PRIMARY KEY ("time", anchor_id)
);
CREATE INDEX idx_pm_hourly_anchors_version_device ON public.pm_hourly_anchors (bucket_version, device_dim_id, "time", anchor_id);

CREATE TABLE public.pm_hourly_values (
    "time" timestamptz NOT NULL,
    bucket_version bigint NOT NULL,
    anchor_id bigint NOT NULL,
    metric_id bigint NOT NULL,
    metric_value double precision NOT NULL,
    CONSTRAINT uq_pm_hourly_values UNIQUE ("time", bucket_version, anchor_id, metric_id)
);
CREATE INDEX idx_pm_hourly_values_version_metric_time ON public.pm_hourly_values (bucket_version, metric_id, "time" DESC, anchor_id);

-- ── pm_metrics_daily（普通表；最终形态：object_ldn NOT NULL DEFAULT ''，pkey 尾含 object_ldn）──
CREATE TABLE public.pm_metrics_daily (
    device_oui text NOT NULL,
    device_sn text NOT NULL,
    metric_path text NOT NULL,
    metric_type text NOT NULL,
    metric_value double precision,
    statis_type text,
    granularity text NOT NULL,
    "time" timestamp with time zone NOT NULL,
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone NOT NULL,
    ingest_time timestamp with time zone DEFAULT now() NOT NULL,
    object_ldn text DEFAULT ''::text NOT NULL,
    extra jsonb,
    CONSTRAINT pm_metrics_daily_pkey PRIMARY KEY (device_oui, device_sn, metric_path, granularity, end_time, object_ldn),
    CONSTRAINT pm_metrics_daily_granularity_check CHECK ((granularity = ANY (ARRAY['15min'::text, 'hourly'::text, 'daily'::text, 'weekly'::text, 'monthly'::text]))),
    CONSTRAINT pm_metrics_daily_metric_type_check CHECK ((metric_type = ANY (ARRAY['counter'::text, 'kpi'::text]))),
    CONSTRAINT pm_metrics_daily_statis_type_check CHECK (((statis_type IS NULL) OR (statis_type = ANY (ARRAY['sum'::text, 'avg'::text, 'max'::text, 'min'::text, 'pct'::text]))))
);
CREATE INDEX idx_pm_metrics_daily_device_time ON public.pm_metrics_daily USING btree (device_oui, device_sn, end_time DESC);
CREATE INDEX idx_pm_metrics_daily_object_ldn ON public.pm_metrics_daily USING btree (object_ldn) WHERE (object_ldn IS NOT NULL);
CREATE INDEX idx_pm_metrics_daily_path_time ON public.pm_metrics_daily USING btree (metric_path, end_time DESC);

-- ── pm_metrics_weekly（普通表；同 daily 最终形态）────────────────────────────────────
CREATE TABLE public.pm_metrics_weekly (
    device_oui text NOT NULL,
    device_sn text NOT NULL,
    metric_path text NOT NULL,
    metric_type text NOT NULL,
    metric_value double precision,
    statis_type text,
    granularity text NOT NULL,
    "time" timestamp with time zone NOT NULL,
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone NOT NULL,
    ingest_time timestamp with time zone DEFAULT now() NOT NULL,
    object_ldn text DEFAULT ''::text NOT NULL,
    extra jsonb,
    CONSTRAINT pm_metrics_weekly_pkey PRIMARY KEY (device_oui, device_sn, metric_path, granularity, end_time, object_ldn),
    CONSTRAINT pm_metrics_weekly_granularity_check CHECK ((granularity = ANY (ARRAY['15min'::text, 'hourly'::text, 'daily'::text, 'weekly'::text, 'monthly'::text]))),
    CONSTRAINT pm_metrics_weekly_metric_type_check CHECK ((metric_type = ANY (ARRAY['counter'::text, 'kpi'::text]))),
    CONSTRAINT pm_metrics_weekly_statis_type_check CHECK (((statis_type IS NULL) OR (statis_type = ANY (ARRAY['sum'::text, 'avg'::text, 'max'::text, 'min'::text, 'pct'::text]))))
);
CREATE INDEX idx_pm_metrics_weekly_device_time ON public.pm_metrics_weekly USING btree (device_oui, device_sn, end_time DESC);
CREATE INDEX idx_pm_metrics_weekly_object_ldn ON public.pm_metrics_weekly USING btree (object_ldn) WHERE (object_ldn IS NOT NULL);
CREATE INDEX idx_pm_metrics_weekly_path_time ON public.pm_metrics_weekly USING btree (metric_path, end_time DESC);

-- ── pm_metrics_monthly（普通表；同 daily 最终形态）───────────────────────────────────
CREATE TABLE public.pm_metrics_monthly (
    device_oui text NOT NULL,
    device_sn text NOT NULL,
    metric_path text NOT NULL,
    metric_type text NOT NULL,
    metric_value double precision,
    statis_type text,
    granularity text NOT NULL,
    "time" timestamp with time zone NOT NULL,
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone NOT NULL,
    ingest_time timestamp with time zone DEFAULT now() NOT NULL,
    object_ldn text DEFAULT ''::text NOT NULL,
    extra jsonb,
    CONSTRAINT pm_metrics_monthly_pkey PRIMARY KEY (device_oui, device_sn, metric_path, granularity, end_time, object_ldn),
    CONSTRAINT pm_metrics_monthly_granularity_check CHECK ((granularity = ANY (ARRAY['15min'::text, 'hourly'::text, 'daily'::text, 'weekly'::text, 'monthly'::text]))),
    CONSTRAINT pm_metrics_monthly_metric_type_check CHECK ((metric_type = ANY (ARRAY['counter'::text, 'kpi'::text]))),
    CONSTRAINT pm_metrics_monthly_statis_type_check CHECK (((statis_type IS NULL) OR (statis_type = ANY (ARRAY['sum'::text, 'avg'::text, 'max'::text, 'min'::text, 'pct'::text]))))
);
CREATE INDEX idx_pm_metrics_monthly_device_time ON public.pm_metrics_monthly USING btree (device_oui, device_sn, end_time DESC);
CREATE INDEX idx_pm_metrics_monthly_object_ldn ON public.pm_metrics_monthly USING btree (object_ldn) WHERE (object_ldn IS NOT NULL);
CREATE INDEX idx_pm_metrics_monthly_path_time ON public.pm_metrics_monthly USING btree (metric_path, end_time DESC);

-- ── pm_group_metrics_hourly（超表：time 7d chunk，compress 14d，retention 180d；pkey(id,time)）──
-- 最终形态：带 technology varchar(3) NOT NULL；自然唯一索引尾部含 technology（000026）。
CREATE TABLE public.pm_group_metrics_hourly (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_group_id uuid NOT NULL,
    metric_path text NOT NULL,
    metric_type text NOT NULL,
    metric_value double precision,
    statis_type text,
    granularity text NOT NULL,
    "time" timestamp with time zone NOT NULL,
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone NOT NULL,
    ingest_time timestamp with time zone DEFAULT now() NOT NULL,
    extra jsonb,
    technology varchar(3) NOT NULL,
    CONSTRAINT pm_group_metrics_hourly_pkey PRIMARY KEY (id, "time"),
    CONSTRAINT pm_group_metrics_hourly_granularity_check CHECK ((granularity = ANY (ARRAY['15min'::text, 'hourly'::text, 'daily'::text, 'weekly'::text, 'monthly'::text]))),
    CONSTRAINT pm_group_metrics_hourly_metric_type_check CHECK ((metric_type = ANY (ARRAY['counter'::text, 'kpi'::text]))),
    CONSTRAINT pm_group_metrics_hourly_statis_type_check CHECK (((statis_type IS NULL) OR (statis_type = ANY (ARRAY['sum'::text, 'avg'::text, 'max'::text, 'min'::text, 'pct'::text]))))
);
CREATE INDEX idx_pm_group_metrics_hourly_group_time ON public.pm_group_metrics_hourly USING btree (device_group_id, "time" DESC);
CREATE INDEX idx_pm_group_metrics_hourly_path_time ON public.pm_group_metrics_hourly USING btree (metric_path, "time" DESC);
CREATE INDEX pm_group_metrics_hourly_time_idx ON public.pm_group_metrics_hourly USING btree ("time" DESC);
CREATE UNIQUE INDEX uq_pm_group_metrics_hourly_natural ON public.pm_group_metrics_hourly USING btree (device_group_id, metric_path, granularity, end_time, "time", technology);

-- ── pm_group_metrics_daily（普通表；最终形态：technology NOT NULL，pkey 尾含 technology）──
CREATE TABLE public.pm_group_metrics_daily (
    device_group_id uuid NOT NULL,
    metric_path text NOT NULL,
    metric_type text NOT NULL,
    metric_value double precision,
    statis_type text,
    granularity text NOT NULL,
    "time" timestamp with time zone NOT NULL,
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone NOT NULL,
    ingest_time timestamp with time zone DEFAULT now() NOT NULL,
    extra jsonb,
    technology varchar(3) NOT NULL,
    CONSTRAINT pm_group_metrics_daily_pkey PRIMARY KEY (device_group_id, metric_path, granularity, end_time, technology),
    CONSTRAINT pm_group_metrics_daily_granularity_check CHECK ((granularity = ANY (ARRAY['15min'::text, 'hourly'::text, 'daily'::text, 'weekly'::text, 'monthly'::text]))),
    CONSTRAINT pm_group_metrics_daily_metric_type_check CHECK ((metric_type = ANY (ARRAY['counter'::text, 'kpi'::text]))),
    CONSTRAINT pm_group_metrics_daily_statis_type_check CHECK (((statis_type IS NULL) OR (statis_type = ANY (ARRAY['sum'::text, 'avg'::text, 'max'::text, 'min'::text, 'pct'::text]))))
);
CREATE INDEX idx_pm_group_metrics_daily_group_time ON public.pm_group_metrics_daily USING btree (device_group_id, end_time DESC);
CREATE INDEX idx_pm_group_metrics_daily_path_time ON public.pm_group_metrics_daily USING btree (metric_path, end_time DESC);

-- ── pm_group_metrics_weekly（普通表；同 daily 最终形态）──────────────────────────────
CREATE TABLE public.pm_group_metrics_weekly (
    device_group_id uuid NOT NULL,
    metric_path text NOT NULL,
    metric_type text NOT NULL,
    metric_value double precision,
    statis_type text,
    granularity text NOT NULL,
    "time" timestamp with time zone NOT NULL,
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone NOT NULL,
    ingest_time timestamp with time zone DEFAULT now() NOT NULL,
    extra jsonb,
    technology varchar(3) NOT NULL,
    CONSTRAINT pm_group_metrics_weekly_pkey PRIMARY KEY (device_group_id, metric_path, granularity, end_time, technology),
    CONSTRAINT pm_group_metrics_weekly_granularity_check CHECK ((granularity = ANY (ARRAY['15min'::text, 'hourly'::text, 'daily'::text, 'weekly'::text, 'monthly'::text]))),
    CONSTRAINT pm_group_metrics_weekly_metric_type_check CHECK ((metric_type = ANY (ARRAY['counter'::text, 'kpi'::text]))),
    CONSTRAINT pm_group_metrics_weekly_statis_type_check CHECK (((statis_type IS NULL) OR (statis_type = ANY (ARRAY['sum'::text, 'avg'::text, 'max'::text, 'min'::text, 'pct'::text]))))
);
CREATE INDEX idx_pm_group_metrics_weekly_group_time ON public.pm_group_metrics_weekly USING btree (device_group_id, end_time DESC);
CREATE INDEX idx_pm_group_metrics_weekly_path_time ON public.pm_group_metrics_weekly USING btree (metric_path, end_time DESC);

-- ── pm_group_metrics_monthly（普通表；同 daily 最终形态）─────────────────────────────
CREATE TABLE public.pm_group_metrics_monthly (
    device_group_id uuid NOT NULL,
    metric_path text NOT NULL,
    metric_type text NOT NULL,
    metric_value double precision,
    statis_type text,
    granularity text NOT NULL,
    "time" timestamp with time zone NOT NULL,
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone NOT NULL,
    ingest_time timestamp with time zone DEFAULT now() NOT NULL,
    extra jsonb,
    technology varchar(3) NOT NULL,
    CONSTRAINT pm_group_metrics_monthly_pkey PRIMARY KEY (device_group_id, metric_path, granularity, end_time, technology),
    CONSTRAINT pm_group_metrics_monthly_granularity_check CHECK ((granularity = ANY (ARRAY['15min'::text, 'hourly'::text, 'daily'::text, 'weekly'::text, 'monthly'::text]))),
    CONSTRAINT pm_group_metrics_monthly_metric_type_check CHECK ((metric_type = ANY (ARRAY['counter'::text, 'kpi'::text]))),
    CONSTRAINT pm_group_metrics_monthly_statis_type_check CHECK (((statis_type IS NULL) OR (statis_type = ANY (ARRAY['sum'::text, 'avg'::text, 'max'::text, 'min'::text, 'pct'::text]))))
);
CREATE INDEX idx_pm_group_metrics_monthly_group_time ON public.pm_group_metrics_monthly USING btree (device_group_id, end_time DESC);
CREATE INDEX idx_pm_group_metrics_monthly_path_time ON public.pm_group_metrics_monthly USING btree (metric_path, end_time DESC);

-- ── pm_adhoc_aggregation_results（超表：time 30d chunk，compress 90d，retention 365d；pkey(id,time)）──
-- 最终形态：带可空 product_id（000005）+ 业务去重唯一索引 uq_pm_adhoc_results_business（000018）。
CREATE TABLE public.pm_adhoc_aggregation_results (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_id uuid NOT NULL,
    device_oui text NOT NULL,
    device_sn text NOT NULL,
    metric_path text NOT NULL,
    metric_type text NOT NULL,
    metric_value double precision,
    statis_type text,
    granularity text NOT NULL,
    "time" timestamp with time zone NOT NULL,
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone NOT NULL,
    ingest_time timestamp with time zone DEFAULT now() NOT NULL,
    object_ldn text,
    extra jsonb,
    product_id uuid,
    CONSTRAINT pm_adhoc_aggregation_results_pkey PRIMARY KEY (id, "time"),
    CONSTRAINT pm_adhoc_aggregation_results_granularity_check CHECK ((granularity = ANY (ARRAY['15min'::text, 'hourly'::text, 'daily'::text, 'weekly'::text, 'monthly'::text]))),
    CONSTRAINT pm_adhoc_aggregation_results_metric_type_check CHECK ((metric_type = ANY (ARRAY['counter'::text, 'kpi'::text]))),
    CONSTRAINT pm_adhoc_aggregation_results_statis_type_check CHECK (((statis_type IS NULL) OR (statis_type = ANY (ARRAY['sum'::text, 'avg'::text, 'max'::text, 'min'::text, 'pct'::text]))))
);
CREATE INDEX idx_pm_adhoc_results_device ON public.pm_adhoc_aggregation_results USING btree (device_oui, device_sn, "time" DESC);
CREATE INDEX idx_pm_adhoc_results_task_time ON public.pm_adhoc_aggregation_results USING btree (task_id, "time" DESC);
CREATE INDEX pm_adhoc_aggregation_results_time_idx ON public.pm_adhoc_aggregation_results USING btree ("time" DESC);
-- 业务去重唯一索引（表达式文本须与 InsertResults 的 ON CONFLICT 目标逐字一致，见 internal/pm/adhoc）。
CREATE UNIQUE INDEX uq_pm_adhoc_results_business
    ON public.pm_adhoc_aggregation_results (
        task_id, granularity, metric_path,
        COALESCE(device_oui, ''), COALESCE(device_sn, ''),
        COALESCE(product_id::text, ''), COALESCE(object_ldn, ''),
        "time"
    );

-- 稀疏值表 insert-triggered autovacuum reloptions。
ALTER TABLE public.pm_metric_values SET (
    autovacuum_vacuum_insert_scale_factor = 0,
    autovacuum_vacuum_insert_threshold = 5000000,
    autovacuum_analyze_scale_factor = 0,
    autovacuum_analyze_threshold = 5000000
);

-- =====================================================================================
-- 2. 超表化（7 张）+ 压缩（5 张）+ 保留（7 张）
--    chunk 间隔 / compress_after / drop_after 还原自 seed 的 _timescaledb_catalog。
-- =====================================================================================

-- ── create_hypertable（dimension.interval_length µs → 人类可读）────────────────────
-- alarms_history: 604800000000 µs = 7 days; col "time"
SELECT create_hypertable('public.alarms_history', by_range('time', INTERVAL '7 days'), if_not_exists => TRUE, migrate_data => TRUE);
-- mr_records: 86400000000 µs = 1 day; col "time"
SELECT create_hypertable('public.mr_records', by_range('time', INTERVAL '1 day'), if_not_exists => TRUE, migrate_data => TRUE);
-- trace_messages: 86400000000 µs = 1 day; col captured_at
SELECT create_hypertable('public.trace_messages', by_range('captured_at', INTERVAL '1 day'), if_not_exists => TRUE, migrate_data => TRUE);
SELECT create_hypertable('public.pm_measurement_anchors', by_range('time', INTERVAL '4 hours'), if_not_exists => TRUE, migrate_data => TRUE);
SELECT create_hypertable('public.pm_metric_values', by_range('time', INTERVAL '4 hours'), if_not_exists => TRUE, migrate_data => TRUE);
SELECT create_hypertable('public.pm_hourly_anchors', by_range('time', INTERVAL '7 days'), if_not_exists => TRUE, migrate_data => TRUE);
SELECT create_hypertable('public.pm_hourly_values', by_range('time', INTERVAL '7 days'), if_not_exists => TRUE, migrate_data => TRUE);
-- pm_group_metrics_hourly: 604800000000 µs = 7 days; col "time"
SELECT create_hypertable('public.pm_group_metrics_hourly', by_range('time', INTERVAL '7 days'), if_not_exists => TRUE, migrate_data => TRUE);
-- pm_adhoc_aggregation_results: 2592000000000 µs = 30 days; col "time"
SELECT create_hypertable('public.pm_adhoc_aggregation_results', by_range('time', INTERVAL '30 days'), if_not_exists => TRUE, migrate_data => TRUE);

-- ── 压缩（columnstore）：原 5 张（mr_records / pm_metrics / pm_metrics_hourly /
--    pm_group_metrics_hourly / pm_adhoc_aggregation_results）。
--    segmentby 还原自 seed compression_settings；orderby 默认走 time 列（DESC）。
--    ⚠ 压缩 SET 必须先于 add_compression_policy（CLAUDE.md §4.6 TimescaleDB 压缩顺序）。
ALTER TABLE public.mr_records SET (timescaledb.compress, timescaledb.compress_segmentby = 'device_id,file_id,mr_type');
ALTER TABLE public.pm_measurement_anchors SET (timescaledb.compress, timescaledb.compress_segmentby = 'device_dim_id,granularity');
ALTER TABLE public.pm_metric_values SET (timescaledb.compress, timescaledb.compress_segmentby = 'metric_id');
ALTER TABLE public.pm_hourly_anchors SET (timescaledb.compress, timescaledb.compress_segmentby = 'bucket_version,device_dim_id');
ALTER TABLE public.pm_hourly_values SET (timescaledb.compress, timescaledb.compress_segmentby = 'bucket_version,metric_id');
ALTER TABLE public.pm_group_metrics_hourly SET (timescaledb.compress, timescaledb.compress_segmentby = 'device_group_id,metric_type,granularity');
ALTER TABLE public.pm_adhoc_aggregation_results SET (timescaledb.compress, timescaledb.compress_segmentby = 'task_id,device_oui,device_sn,metric_type');

-- compress_after 还原自 seed bgw_job policy_compression：
--   mr_records 7d / pm_metrics 7d / pm_metrics_hourly 14d / pm_group_metrics_hourly 14d / pm_adhoc 90d
SELECT add_compression_policy('public.mr_records', INTERVAL '7 days');
SELECT add_compression_policy('public.pm_measurement_anchors', INTERVAL '7 days');
SELECT add_compression_policy('public.pm_metric_values', INTERVAL '7 days');
SELECT add_compression_policy('public.pm_group_metrics_hourly', INTERVAL '14 days');
SELECT add_compression_policy('public.pm_adhoc_aggregation_results', INTERVAL '90 days');

-- ── 保留：原 7 张。drop_after 还原自 seed bgw_job policy_retention：
--   alarms_history 365d / mr_records 90d / trace_messages 3d / pm_metrics 30d /
--   pm_metrics_hourly 180d / pm_group_metrics_hourly 180d / pm_adhoc 365d
SELECT add_retention_policy('public.alarms_history', INTERVAL '365 days');

-- alarms_history retention 固定为每天 01:08 Asia/Shanghai 执行（原 000002 增量）。
SELECT alter_job(
    j.job_id,
    schedule_interval => INTERVAL '1 day',
    fixed_schedule => TRUE,
    initial_start => TIMESTAMPTZ '2000-01-01 01:08:00+08',
    timezone => 'Asia/Shanghai'
)
  FROM timescaledb_information.jobs j
 WHERE j.proc_name = 'policy_retention'
   AND j.hypertable_schema = 'public'
   AND j.hypertable_name = 'alarms_history';

SELECT add_retention_policy('public.mr_records', INTERVAL '90 days');
SELECT add_retention_policy('public.trace_messages', INTERVAL '3 days');
-- 原始 PM 明细默认保留 30 天；app 启动和 pm.retention 配置保存时会
-- 把两张稀疏表原子更新为 raw_15min_days 的当前值。
SELECT add_retention_policy('public.pm_measurement_anchors', INTERVAL '30 days');
SELECT add_retention_policy('public.pm_metric_values', INTERVAL '30 days');
SELECT add_retention_policy('public.pm_hourly_anchors', INTERVAL '180 days');
SELECT add_retention_policy('public.pm_hourly_values', INTERVAL '180 days');
SELECT add_retention_policy('public.pm_group_metrics_hourly', INTERVAL '180 days');
SELECT add_retention_policy('public.pm_adhoc_aggregation_results', INTERVAL '365 days');

-- =====================================================================================
-- 3. 影子维度表（§契约 1B；从主库同步，供本库 JOIN 替代跨库 JOIN）
--    带主键供 UPSERT；JOIN 键加索引。worker 的 tsdbsync 同步任务定期刷入（见 internal/tsdbsync）。
--    说明：device_dim 是「主库 devices 的列子集」（§1B 指定列）；cell_band_dim 是【派生表】
--    （主库无 cell_band 表，原 band 查询用 CTE，现由同步任务从 device_parameters 派生 device_id/cell_id/band）；
--    其余 4 张（product_dim/device_group_dim/device_group_member_dim/alarm_definition_dim）是源表全列镜像。
-- =====================================================================================

-- device_dim ← devices（列子集：id, oui, serial_number, product_id, technology, site_name, product_class, deleted_at）
CREATE TABLE public.device_dim (
    id uuid NOT NULL,
    oui character varying(6),
    serial_number character varying(64),
    product_id uuid,
    technology character varying(3),
    carrier character varying(16),
    site_name character varying(128),
    product_class character varying(64),
    deleted_at timestamp with time zone,
    CONSTRAINT device_dim_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_device_dim_oui_sn ON public.device_dim USING btree (oui, serial_number);
CREATE INDEX idx_device_dim_product ON public.device_dim USING btree (product_id);

-- device_group_member_dim ← device_group_members（全列镜像：group_id, device_id, added_at, source_type）
CREATE TABLE public.device_group_member_dim (
    group_id uuid NOT NULL,
    device_id uuid NOT NULL,
    added_at timestamp with time zone,
    source_type character varying(16),
    CONSTRAINT device_group_member_dim_pkey PRIMARY KEY (group_id, device_id)
);
CREATE INDEX idx_device_group_member_dim_device ON public.device_group_member_dim USING btree (device_id, group_id);

-- cell_band_dim（派生：device_id, cell_id, band）；主键 (device_id, cell_id)，JOIN 键加索引。
CREATE TABLE public.cell_band_dim (
    device_id uuid NOT NULL,
    cell_id text NOT NULL,
    band text,
    CONSTRAINT cell_band_dim_pkey PRIMARY KEY (device_id, cell_id)
);
CREATE INDEX idx_cell_band_dim_device_cell ON public.cell_band_dim USING btree (device_id, cell_id);

-- product_dim ← products（全列镜像）
CREATE TABLE public.product_dim (
    id uuid NOT NULL,
    product_name character varying(128),
    vendor character varying(64),
    tech character varying(8),
    radio_modes character varying(64),
    description text,
    param_model_id uuid,
    indicator_device_type character varying(8),
    indicator_platform character varying(32),
    alarm_ne_type character varying(16),
    enable_filetype11 boolean,
    device_attrs_override jsonb,
    enable_unknown_alarm boolean,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT product_dim_pkey PRIMARY KEY (id)
);

-- device_group_dim ← device_groups（列子集：id, name；§1B 仅 JOIN 取 name）
CREATE TABLE public.device_group_dim (
    id uuid NOT NULL,
    name character varying(128),
    CONSTRAINT device_group_dim_pkey PRIMARY KEY (id)
);

-- alarm_definition_dim ← alarm_definitions（全列镜像，供 JOIN 取 identifier/cn_name/en_name 等）
CREATE TABLE public.alarm_definition_dim (
    id uuid NOT NULL,
    identifier character varying(32),
    ne_type character varying(16),
    cn_name character varying(256),
    en_name character varying(256),
    severity_id uuid,
    event_type integer,
    cn_probable_cause text,
    en_probable_cause text,
    cn_suggestion text,
    en_suggestion text,
    is_show boolean,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    loaded_from character varying(256),
    CONSTRAINT alarm_definition_dim_pkey PRIMARY KEY (id)
);
-- JOIN 键 identifier（alarms_history.alarm_identifier = ad.identifier）加索引。
CREATE INDEX idx_alarm_definition_dim_identifier ON public.alarm_definition_dim USING btree (identifier);

-- mr_customize_task_dim ← mr_customize_task（列子集：task_id, task_status, target_device_sns）
-- mr_files 迁来时序库后，MR 文件设备聚合的「上报中」徽标判定（mr_customize_task task_status='on'
-- AND device_sn = ANY(target_device_sns)）需本库 JOIN —— 故把任务订阅状态镜像成影子表。
-- mr_customize_task 本身（任务配置）仍留主库，此处只读镜像。
CREATE TABLE public.mr_customize_task_dim (
    task_id uuid NOT NULL,
    task_status character varying(16),
    target_device_sns text[],
    CONSTRAINT mr_customize_task_dim_pkey PRIMARY KEY (task_id)
);

-- =====================================================================================
-- 3b. 仅由时序库侧消费的主库表 —— 整张归时序库（非镜像，本库读写）。
--     trace_tasks / trace_export_jobs：trace repo（单池 TsPool）写 trace_messages 同时 UPDATE
--       trace_tasks、管理 trace_export_jobs，三表同库才能单池工作（避免跨库 split-brain）。
--     kpi_definitions：仅 pm/kpi repo（TsPool）读写（ListDefinitions / SyncDefinitions），全仓无其它
--       消费方，故整张归时序库，免第二池/镜像。
--     （主库 000001 仍保留这几张同名空表，无害未用；fresh 部署不追求主库零冗余。）
-- =====================================================================================
CREATE TABLE public.trace_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    operator_code character varying(16) DEFAULT 'default'::character varying NOT NULL,
    status character varying(16) DEFAULT 'running'::character varying NOT NULL,
    start_time timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    stopped_at timestamp with time zone,
    purged_at timestamp with time zone,
    created_by character varying(64) DEFAULT 'system'::character varying NOT NULL,
    message_count integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT trace_tasks_status_check CHECK (((status)::text = ANY ((ARRAY['running'::character varying, 'stopped'::character varying, 'purged'::character varying])::text[]))),
    CONSTRAINT trace_tasks_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_trace_tasks_created ON public.trace_tasks USING btree (created_at DESC);
CREATE INDEX idx_trace_tasks_sn_status ON public.trace_tasks USING btree (device_sn, status);
CREATE INDEX idx_trace_tasks_status_expires ON public.trace_tasks USING btree (status, expires_at) WHERE ((status)::text = 'running'::text);
CREATE UNIQUE INDEX uniq_trace_tasks_running_sn ON public.trace_tasks USING btree (device_sn) WHERE ((status)::text = 'running'::text);

CREATE TABLE public.trace_export_jobs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_id uuid NOT NULL,
    requested_by character varying(64) DEFAULT 'system'::character varying NOT NULL,
    status character varying(16) DEFAULT 'queued'::character varying NOT NULL,
    object_key character varying(512),
    object_bucket character varying(64),
    message_count integer DEFAULT 0 NOT NULL,
    size_bytes bigint DEFAULT 0 NOT NULL,
    error_message text,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT trace_export_jobs_status_check CHECK (((status)::text = ANY ((ARRAY['queued'::character varying, 'running'::character varying, 'done'::character varying, 'failed'::character varying])::text[]))),
    CONSTRAINT trace_export_jobs_pkey PRIMARY KEY (id),
    CONSTRAINT trace_export_jobs_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.trace_tasks(id) ON DELETE CASCADE
);
CREATE INDEX idx_trace_export_jobs_status ON public.trace_export_jobs USING btree (status, created_at DESC);
CREATE INDEX idx_trace_export_jobs_task ON public.trace_export_jobs USING btree (task_id);

CREATE TABLE public.kpi_definitions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(64) NOT NULL,
    display_name character varying(128) NOT NULL,
    formula text NOT NULL,
    unit character varying(16) NOT NULL,
    category character varying(32) NOT NULL,
    carrier character varying(4),
    technology character varying(3),
    counters jsonb DEFAULT '[]'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT kpi_definitions_pkey PRIMARY KEY (id),
    CONSTRAINT kpi_definitions_name_key UNIQUE (name)
);
CREATE INDEX idx_kpi_definitions_counters_gin ON public.kpi_definitions USING gin (counters);

-- 内置 KPI 定义（原在主库 seed/000001_init_seed.sql；kpi_definitions 归时序库后随之迁来，
-- 否则 ListDefinitions（TsPool）读不到内置定义）。ON CONFLICT DO NOTHING 幂等。
INSERT INTO public.kpi_definitions (id, name, display_name, formula, unit, category, carrier, technology, counters, created_at) VALUES
	('30000000-0001-4000-8000-000000000001', 'RRC_CONN_SETUP_SR', 'RRC连接建立成功率', '(rrc_conn_setup_succ / rrc_conn_setup_att) * 100', '%', 'accessibility', NULL, 'lte', '["rrc_conn_setup_succ", "rrc_conn_setup_att"]', '2026-05-31 11:28:45.998778+08'),
	('30000000-0001-4000-8000-000000000002', 'ERAB_SETUP_SR', 'E-RAB建立成功率', '(erab_setup_succ / erab_setup_att) * 100', '%', 'accessibility', NULL, 'lte', '["erab_setup_succ", "erab_setup_att"]', '2026-05-31 11:28:45.998778+08'),
	('30000000-0001-4000-8000-000000000003', 'INTRA_FREQ_HO_SR', '同频切换成功率', '(intra_freq_ho_succ / intra_freq_ho_att) * 100', '%', 'mobility', NULL, 'lte', '["intra_freq_ho_succ", "intra_freq_ho_att"]', '2026-05-31 11:28:45.998778+08'),
	('30000000-0001-4000-8000-000000000004', 'INTER_FREQ_HO_SR', '异频切换成功率', '(inter_freq_ho_succ / inter_freq_ho_att) * 100', '%', 'mobility', NULL, 'lte', '["inter_freq_ho_succ", "inter_freq_ho_att"]', '2026-05-31 11:28:45.998778+08'),
	('30000000-0001-4000-8000-000000000005', 'CALL_DROP_RATE', '掉话率', '(erab_abnormal_release / erab_release_total) * 100', '%', 'retainability', NULL, 'lte', '["erab_abnormal_release", "erab_release_total"]', '2026-05-31 11:28:45.998778+08'),
	('30000000-0001-4000-8000-000000000006', 'DL_PRB_UTIL', '下行PRB利用率', '(dl_prb_used_avg / dl_prb_available) * 100', '%', 'utilization', NULL, 'lte', '["dl_prb_used_avg", "dl_prb_available"]', '2026-05-31 11:28:45.998778+08'),
	('30000000-0002-4000-8000-000000000001', 'NR_RRC_SETUP_SR', 'NR RRC建立成功率', '(nr_rrc_setup_succ / nr_rrc_setup_att) * 100', '%', 'accessibility', NULL, 'nr', '["nr_rrc_setup_succ", "nr_rrc_setup_att"]', '2026-05-31 11:28:45.998778+08'),
	('30000000-0002-4000-8000-000000000002', 'NR_PDCP_RATE_DL', 'NR 下行PDCP速率', 'pdcp_vol_dl / report_period', 'Mbps', 'throughput', NULL, 'nr', '["pdcp_vol_dl", "report_period"]', '2026-05-31 11:28:45.998778+08'),
	('30000000-0002-4000-8000-000000000003', 'NR_SA_HO_SR', 'NR SA切换成功率', '(nr_ho_succ / nr_ho_att) * 100', '%', 'mobility', NULL, 'nr', '["nr_ho_succ", "nr_ho_att"]', '2026-05-31 11:28:45.998778+08'),
	('30000000-0002-4000-8000-000000000004', 'NR_PRB_UTIL_DL', 'NR 下行PRB利用率', '(nr_dl_prb_used / nr_dl_prb_total) * 100', '%', 'utilization', NULL, 'nr', '["nr_dl_prb_used", "nr_dl_prb_total"]', '2026-05-31 11:28:45.998778+08'),
	('30000000-0002-4000-8000-000000000005', 'NR_CQI_AVG', 'NR 平均CQI', 'AVG(cqi_value)', '', 'quality', NULL, 'nr', '["cqi_value"]', '2026-05-31 11:28:45.998778+08'),
	('30000000-0002-4000-8000-000000000006', 'NR_RLC_LOSS_RATE', 'NR RLC丢包率', '(rlc_retx_dl / rlc_tx_dl) * 100', '%', 'retainability', NULL, 'nr', '["rlc_retx_dl", "rlc_tx_dl"]', '2026-05-31 11:28:45.998778+08')
ON CONFLICT DO NOTHING;

-- =====================================================================================
-- 3c. perf_indicators_{enb,gnb,gsm} —— 时序库【同名镜像】（主库为源，由 worker tsdbsync 全量刷入）。
--     pm 的指标名解析查询（aggregator/adhoc/export，跑 TsPool）按真实表名读，故时序库建同名表，
--     SQL 零改动即命中本库镜像。主库保留原表（仍是字典源 + 指标管理消费方）。
-- =====================================================================================
CREATE TABLE public.perf_indicators_enb (
    id character varying(20) NOT NULL,
    report_key character varying(200),
    en_name character varying(200) NOT NULL,
    cn_name character varying(200) NOT NULL,
    en_description text,
    cn_description text,
    group_id character varying(64) NOT NULL,
    operator_code character varying(100),
    data_type character varying(20),
    unit_id character varying(50),
    updator character varying(64),
    is_build_in character(1) DEFAULT '0'::bpchar NOT NULL,
    is_counter character(1) DEFAULT '1'::bpchar NOT NULL,
    arithmetic text,
    statis_type character varying(20),
    calculating_status character varying(20),
    product_types text,
    indicator_level character varying(20),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    loaded_from character varying(256),
    CONSTRAINT perf_indicators_enb_pkey PRIMARY KEY (id)
);
CREATE TABLE public.perf_indicators_gnb (
    id character varying(20) NOT NULL,
    report_key character varying(200),
    en_name character varying(200) NOT NULL,
    cn_name character varying(200) NOT NULL,
    en_description text,
    cn_description text,
    group_id character varying(64) NOT NULL,
    operator_code character varying(100),
    data_type character varying(20),
    unit_id character varying(50),
    updator character varying(64),
    is_build_in character(1) DEFAULT '0'::bpchar NOT NULL,
    is_counter character(1) DEFAULT '1'::bpchar NOT NULL,
    arithmetic text,
    statis_type character varying(20),
    calculating_status character varying(20),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    loaded_from character varying(256),
    CONSTRAINT perf_indicators_gnb_pkey PRIMARY KEY (id)
);
CREATE TABLE public.perf_indicators_gsm (
    id character varying(20) NOT NULL,
    report_key character varying(200),
    en_name character varying(200) NOT NULL,
    cn_name character varying(200) NOT NULL,
    en_description text,
    cn_description text,
    group_id character varying(64) NOT NULL,
    operator_code character varying(100),
    data_type character varying(20),
    unit_id character varying(50),
    updator character varying(64),
    is_build_in character(1) DEFAULT '0'::bpchar NOT NULL,
    is_counter character(1) DEFAULT '1'::bpchar NOT NULL,
    arithmetic text,
    statis_type character varying(20),
    calculating_status character varying(20),
    product_types text,
    indicator_level character varying(20),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    loaded_from character varying(256),
    CONSTRAINT perf_indicators_gsm_pkey PRIMARY KEY (id)
);

-- =====================================================================================
-- 3d. PM 逻辑兼容视图：锚点展开历史指标集，左连接稀疏真实值。
-- =====================================================================================
CREATE VIEW public.pm_metrics AS
SELECT
    md5(a.anchor_id::text || ':' || d.metric_id::text)::uuid AS id,
    COALESCE(dev.oui, '')::text AS device_oui,
    COALESCE(dev.serial_number, f.device_sn)::text AS device_sn,
    d.metric_path, d.metric_type, v.metric_value, d.statis_type, a.granularity,
    a."time", a.start_time, a.end_time,
    COALESCE(b.committed_at, f.created_at, now()) AS ingest_time,
    a.object_ldn,
    jsonb_strip_nulls(jsonb_build_object(
        'device_id', a.device_dim_id::text, 'counter_group', a.counter_group,
        'carrier', COALESCE(dev.carrier, f.carrier),
        'technology', COALESCE(dev.technology, f.technology)
    )) AS extra
FROM public.pm_measurement_anchors a
JOIN public.pm_metric_sets s ON s.metric_set_id = a.metric_set_id
CROSS JOIN LATERAL unnest(s.metric_ids) AS sid(metric_id)
JOIN public.pm_metric_dictionary d ON d.metric_id = sid.metric_id
LEFT JOIN public.pm_metric_values v
  ON v."time" = a."time" AND v.anchor_id = a.anchor_id AND v.metric_id = d.metric_id
LEFT JOIN public.pm_files f ON f.id = a.source_file_id
LEFT JOIN public.pm_ingest_batches b ON b.ingest_batch_id = a.ingest_batch_id
LEFT JOIN public.device_dim dev ON dev.id = a.device_dim_id;

CREATE VIEW public.pm_metrics_hourly AS
SELECT
    md5(a.bucket_version::text || ':' || a.anchor_id::text || ':' || d.metric_id::text)::uuid AS id,
    COALESCE(dev.oui, '')::text AS device_oui,
    COALESCE(dev.serial_number, '')::text AS device_sn,
    d.metric_path, d.metric_type, v.metric_value, d.statis_type, a.granularity,
    a."time", a.start_time, a.end_time,
    COALESCE(ver.published_at, ver.created_at) AS ingest_time,
    a.object_ldn,
    jsonb_strip_nulls(jsonb_build_object(
        'device_id', a.device_dim_id::text, 'counter_group', a.counter_group,
        'carrier', dev.carrier, 'technology', dev.technology
    )) AS extra
FROM public.pm_hourly_bucket_versions ver
JOIN public.pm_hourly_anchors a ON a.bucket_version = ver.bucket_version
JOIN public.pm_metric_sets s ON s.metric_set_id = a.metric_set_id
CROSS JOIN LATERAL unnest(s.metric_ids) AS sid(metric_id)
JOIN public.pm_metric_dictionary d ON d.metric_id = sid.metric_id
LEFT JOIN public.pm_hourly_values v
  ON v.bucket_version = a.bucket_version
 AND v."time" = a."time" AND v.anchor_id = a.anchor_id AND v.metric_id = d.metric_id
LEFT JOIN public.device_dim dev ON dev.id = a.device_dim_id
WHERE ver.status = 'active';

-- =====================================================================================
-- 3e. 同名视图（安全网）—— 把 6 张影子维度表以【源表原名】暴露在时序库。
--     作用：pm/aggregator 等已把显式 JOIN 改成 *_dim，但 authz 包的 #64 数据权限子查询
--     （ApplyDeviceSNVisibilityFilter / VisibleSNSubquerySQL / ApplyGroupVisibilityFilter）
--     硬编码 FROM devices / device_group_members，且被 TsPool 查询调用。建同名视图后这些
--     未改名引用在时序库自动命中影子表，零代码改动兜底任何遗漏的跨库引用。
-- =====================================================================================
CREATE VIEW public.devices AS SELECT * FROM public.device_dim;
CREATE VIEW public.device_group_members AS SELECT * FROM public.device_group_member_dim;
CREATE VIEW public.products AS SELECT * FROM public.product_dim;
CREATE VIEW public.device_groups AS SELECT * FROM public.device_group_dim;
CREATE VIEW public.cell_band AS SELECT * FROM public.cell_band_dim;
CREATE VIEW public.alarm_definitions AS SELECT * FROM public.alarm_definition_dim;

-- =====================================================================================
-- 4. 告警效率物化视图 alarm_efficiency_metrics（最终形态 = seed 000042，含 MTTR 倒挂过滤）。
--    建在 alarms_history 上；dashboard 用 REFRESH MATERIALIZED VIEW CONCURRENTLY 刷新，
--    故必须建唯一索引（severity 为 GROUP BY 键 → 每行唯一）。原 seed 漏建唯一索引使
--    CONCURRENTLY 刷新会报错；本处补齐（修该潜伏 bug）。
-- =====================================================================================
CREATE MATERIALIZED VIEW public.alarm_efficiency_metrics AS
SELECT
    severity,
    COUNT(*) FILTER (WHERE acknowledged_at IS NOT NULL) as acknowledged_count,
    COUNT(*) FILTER (WHERE cleared_at IS NOT NULL) as cleared_count,
    COUNT(*) as total_count,
    COALESCE(AVG(EXTRACT(EPOCH FROM (acknowledged_at - raised_at)) / 60)
        FILTER (WHERE acknowledged_at IS NOT NULL), 0) as avg_acknowledge_minutes,
    COALESCE(AVG(EXTRACT(EPOCH FROM (cleared_at - raised_at)) / 60)
        FILTER (WHERE cleared_at IS NOT NULL AND cleared_at >= raised_at), 0) as avg_resolve_minutes,
    ROUND(100.0 * COUNT(*) FILTER (WHERE acknowledged_at IS NOT NULL) /
          NULLIF(COUNT(*), 0), 2) as acknowledge_rate,
    ROUND(100.0 * COUNT(*) FILTER (WHERE cleared_at IS NOT NULL) /
          NULLIF(COUNT(*), 0), 2) as clear_rate
FROM public.alarms_history
WHERE cleared_at > NOW() - INTERVAL '30 days'
GROUP BY severity;

-- 唯一索引：支持 REFRESH MATERIALIZED VIEW CONCURRENTLY（dashboard/efficiency.go）。
CREATE UNIQUE INDEX uq_alarm_efficiency_metrics_severity ON public.alarm_efficiency_metrics USING btree (severity);

COMMENT ON MATERIALIZED VIEW public.alarm_efficiency_metrics IS
    '告警处理效率指标物化视图，包含MTTA、MTTR、确认率、清除率等指标（MTTR 已排除时间倒挂脏数据）';

-- 刷新函数（dashboard 也可直接 REFRESH ... CONCURRENTLY；保留函数与原 seed 行为一致）。
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION public.refresh_alarm_efficiency_metrics()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY public.alarm_efficiency_metrics;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

COMMENT ON FUNCTION public.refresh_alarm_efficiency_metrics() IS '刷新告警效率指标物化视图';



-- Consolidated from former incremental migrations: TSDB schema 000002

-- 旧扫描式聚合结果不迁移。
DROP VIEW IF EXISTS public.pm_metrics_hourly;
DROP TABLE IF EXISTS public.pm_adhoc_aggregation_results;
DROP TABLE IF EXISTS public.pm_group_metrics_monthly;
DROP TABLE IF EXISTS public.pm_group_metrics_weekly;
DROP TABLE IF EXISTS public.pm_group_metrics_daily;
DROP TABLE IF EXISTS public.pm_group_metrics_hourly;
DROP TABLE IF EXISTS public.pm_metrics_monthly;
DROP TABLE IF EXISTS public.pm_metrics_weekly;
DROP TABLE IF EXISTS public.pm_metrics_daily;
DROP TABLE IF EXISTS public.pm_hourly_values;
DROP TABLE IF EXISTS public.pm_hourly_anchors;
DROP TABLE IF EXISTS public.pm_hourly_rollup_batches;
DROP TABLE IF EXISTS public.pm_hourly_bucket_versions;

CREATE TABLE public.pm_aggregation_outbox (
    event_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_file_id uuid NOT NULL UNIQUE,
    ingest_batch_id uuid NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,
    publish_attempts integer NOT NULL DEFAULT 0,
    last_error text
);
CREATE INDEX idx_pm_aggregation_outbox_pending
    ON public.pm_aggregation_outbox (created_at, event_id)
    WHERE published_at IS NULL;

CREATE TABLE public.pm_aggregation_counter_rollups (
    event_id uuid PRIMARY KEY,
    task_id uuid NOT NULL,
    task_version_id uuid NOT NULL,
    publication_task_version_id uuid NOT NULL,
    entity_key text NOT NULL,
    granularity varchar(16) NOT NULL,
    window_start timestamptz NOT NULL,
    window_end timestamptz NOT NULL,
    chunk_index integer NOT NULL,
    chunk_count integer NOT NULL,
    complete boolean NOT NULL,
    revision integer NOT NULL DEFAULT 1,
    publication_eligible boolean NOT NULL DEFAULT true,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_pm_aggregation_counter_rollups_granularity
        CHECK (granularity IN ('hourly', 'daily')),
    CONSTRAINT chk_pm_aggregation_counter_rollups_window
        CHECK (window_end > window_start),
    CONSTRAINT chk_pm_aggregation_counter_rollups_chunk
        CHECK (chunk_index >= 0 AND chunk_count > 0 AND chunk_index < chunk_count)
);
CREATE INDEX idx_pm_aggregation_counter_rollups_recovery
    ON public.pm_aggregation_counter_rollups (
        task_version_id, publication_task_version_id, entity_key,
        granularity, window_start, revision, chunk_index
    );
CREATE INDEX idx_pm_aggregation_counter_rollups_retention
    ON public.pm_aggregation_counter_rollups (granularity, window_start);
CREATE INDEX idx_pm_counter_rollups_period_rebuild_page
    ON public.pm_aggregation_counter_rollups (
        task_version_id, granularity, window_start, entity_key, chunk_index,
        publication_task_version_id, revision, event_id
    );
CREATE INDEX idx_pm_counter_rollups_revision_cleanup
    ON public.pm_aggregation_counter_rollups (
        publication_task_version_id, entity_key, granularity, window_start,
        revision, event_id
    );

CREATE TABLE public.pm_aggregation_rollup_outbox (
    event_id uuid PRIMARY KEY,
    subject text NOT NULL,
    task_version_id uuid NOT NULL,
    publication_task_version_id uuid NOT NULL,
    entity_key text NOT NULL,
    granularity varchar(16) NOT NULL,
    window_start timestamptz NOT NULL,
    revision integer NOT NULL DEFAULT 1,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,
    publish_attempts integer NOT NULL DEFAULT 0,
    last_error text,
    CONSTRAINT chk_pm_aggregation_rollup_outbox_granularity
        CHECK (granularity IN ('hourly', 'daily'))
);
CREATE INDEX idx_pm_aggregation_rollup_outbox_pending
    ON public.pm_aggregation_rollup_outbox (created_at, event_id)
    WHERE published_at IS NULL;
CREATE INDEX idx_pm_rollup_outbox_revision_cleanup
    ON public.pm_aggregation_rollup_outbox (
        publication_task_version_id, entity_key, granularity, window_start,
        revision, event_id
    );
CREATE INDEX idx_pm_aggregation_rollup_outbox_retention
    ON public.pm_aggregation_rollup_outbox (published_at)
    WHERE published_at IS NOT NULL;

CREATE TABLE public.pm_aggregation_windows (
    task_id uuid NOT NULL,
    task_version_id uuid NOT NULL,
    entity_key text NOT NULL,
    granularity varchar(16) NOT NULL,
    window_start timestamptz NOT NULL,
    window_end timestamptz NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'open',
    expected_slots bigint NOT NULL,
    received_slots bigint NOT NULL DEFAULT 0,
    source_expected_slots bigint NOT NULL DEFAULT 0,
    source_received_slots bigint NOT NULL DEFAULT 0,
    source_incomplete_slots bigint NOT NULL DEFAULT 0,
    missing_slots bigint NOT NULL DEFAULT 0,
    children_complete boolean NOT NULL DEFAULT false,
    data_complete boolean NOT NULL DEFAULT false,
    close_reason varchar(16),
    result_count integer NOT NULL DEFAULT 0,
    opened_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now(),
    last_error text,
    PRIMARY KEY (task_version_id, entity_key, granularity, window_start),
    CONSTRAINT chk_pm_aggregation_windows_granularity
        CHECK (granularity IN ('hourly', 'daily', 'weekly', 'monthly')),
    CONSTRAINT chk_pm_aggregation_windows_status
        CHECK (status IN ('open', 'finalizing', 'published', 'failed')),
    CONSTRAINT chk_pm_aggregation_windows_close_reason
        CHECK (close_reason IS NULL OR close_reason IN ('complete', 'timeout')),
    CONSTRAINT chk_pm_aggregation_windows_time CHECK (window_end > window_start)
);
CREATE INDEX idx_pm_aggregation_windows_due
    ON public.pm_aggregation_windows (window_end, status)
    WHERE status IN ('open', 'failed', 'finalizing');
CREATE INDEX idx_pm_aggregation_windows_task_time
    ON public.pm_aggregation_windows (task_id, window_start DESC);

CREATE TABLE public.pm_aggregation_publications (
    task_id uuid NOT NULL,
    task_version_id uuid NOT NULL,
    granularity varchar(16) NOT NULL,
    window_start timestamptz NOT NULL,
    window_end timestamptz NOT NULL,
    revision integer NOT NULL DEFAULT 0,
    preparing_revision integer,
    status varchar(16) NOT NULL DEFAULT 'preparing',
    expected_entities integer NOT NULL DEFAULT 0,
    prepared_entities integer NOT NULL DEFAULT 0,
    dirty_entities integer NOT NULL DEFAULT 0,
    watermark_at timestamptz,
    published_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (task_version_id, granularity, window_start),
    CONSTRAINT chk_pm_aggregation_publications_granularity
        CHECK (granularity IN ('hourly', 'daily', 'weekly', 'monthly')),
    CONSTRAINT chk_pm_aggregation_publications_status
        CHECK (status IN ('preparing', 'published')),
    CONSTRAINT chk_pm_aggregation_publications_time CHECK (window_end > window_start)
);
CREATE INDEX idx_pm_aggregation_publications_due
    ON public.pm_aggregation_publications (window_end, task_version_id, window_start)
    WHERE status = 'preparing' OR preparing_revision IS NOT NULL;

CREATE TABLE public.pm_aggregation_results (
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    window_start timestamptz NOT NULL,
    window_end timestamptz NOT NULL,
    task_id uuid NOT NULL,
    task_version_id uuid NOT NULL,
    granularity varchar(16) NOT NULL,
    dimension varchar(32) NOT NULL,
    dimension_key text NOT NULL,
    dimension_name text NOT NULL DEFAULT '',
    object_ldn text NOT NULL DEFAULT '',
    device_oui text NOT NULL DEFAULT '',
    device_sn text NOT NULL DEFAULT '',
    technology varchar(16) NOT NULL DEFAULT '',
    metric_id text NOT NULL,
    metric_path text NOT NULL,
    metric_type varchar(16) NOT NULL,
    aggregation_op varchar(8) NOT NULL,
    metric_value double precision NOT NULL,
    sample_count bigint NOT NULL,
    complete boolean NOT NULL,
    missing_slots bigint NOT NULL DEFAULT 0,
    revision integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_pm_aggregation_results_granularity
        CHECK (granularity IN ('hourly', 'daily', 'weekly', 'monthly')),
    CONSTRAINT chk_pm_aggregation_results_dimension
        CHECK (dimension IN ('device', 'aggregate_group', 'device_group', 'product', 'band', 'network')),
    CONSTRAINT chk_pm_aggregation_results_metric_type
        CHECK (metric_type IN ('counter', 'kpi')),
    CONSTRAINT chk_pm_aggregation_results_op
        CHECK (aggregation_op IN ('sum', 'avg', 'min', 'max', 'formula')),
    CONSTRAINT chk_pm_aggregation_results_time CHECK (window_end > window_start)
);

SELECT create_hypertable(
    'public.pm_aggregation_results',
    by_range('window_start', INTERVAL '30 days'),
    if_not_exists => TRUE
);

CREATE UNIQUE INDEX uq_pm_aggregation_results_business
    ON public.pm_aggregation_results (
        task_version_id, granularity, window_start,
        dimension_key, object_ldn, technology, metric_id, revision
    );
CREATE INDEX idx_pm_aggregation_results_task_time
    ON public.pm_aggregation_results (task_id, window_start DESC);
CREATE INDEX idx_pm_aggregation_results_dimension_time
    ON public.pm_aggregation_results (dimension, dimension_key, window_start DESC);
CREATE INDEX idx_pm_aggregation_results_metric_time
    ON public.pm_aggregation_results (metric_id, window_start DESC);

ALTER TABLE public.pm_aggregation_results SET (
    timescaledb.compress,
    timescaledb.compress_segmentby =
        'task_id,task_version_id,granularity,dimension,dimension_key,technology,metric_id'
);
SELECT add_compression_policy(
    'public.pm_aggregation_results',
    INTERVAL '90 days',
    if_not_exists => TRUE
);
-- 各粒度保留时间读取主库 sys_configs(pm.retention)，由 worker 分批清理；
-- 同一 hypertable 内存在小时/日/周/月多种期限，不能挂单一 drop_after policy。

-- 现有报表/导出 API 的只读投影。它不保留旧结果，也不触发原始 PM 查询。
CREATE VIEW public.pm_adhoc_aggregation_results AS
SELECT
    r.id,
    r.task_id,
    r.device_oui,
    CASE WHEN r.dimension = 'device' THEN r.device_sn ELSE 'AGGREGATED' END::text AS device_sn,
    CASE WHEN r.dimension = 'product' THEN r.dimension_key::uuid ELSE NULL::uuid END AS product_id,
    r.metric_path,
    r.metric_type,
    r.metric_value,
    r.aggregation_op::text AS statis_type,
    r.granularity::text,
    r.window_start AS "time",
    r.window_start AS start_time,
    r.window_end AS end_time,
    r.created_at AS ingest_time,
    CASE
        WHEN r.dimension = 'device' THEN NULLIF(r.object_ldn, '')
        WHEN r.dimension = 'network' THEN 'Network'
        ELSE r.dimension_key
    END::text AS object_ldn,
    jsonb_build_object(
        'task_version_id', r.task_version_id,
        'complete', r.complete,
        'missing_slots', r.missing_slots,
        'dimension', r.dimension
    ) AS extra
FROM public.pm_aggregation_results r
JOIN public.pm_aggregation_publications published_window
  ON published_window.task_version_id = r.task_version_id
 AND published_window.granularity = r.granularity
 AND published_window.window_start = r.window_start
 AND published_window.status = 'published'
 AND published_window.revision = r.revision;

-- 旧报表查询名称改为统一结果表的只读投影。投影不扫描原始 PM 表；当多个逻辑任务
-- 覆盖同一设备/指标/窗口时，仅暴露最新写入的一份，避免报表重复计数。
CREATE VIEW public.pm_metrics_hourly AS
SELECT DISTINCT ON (r.dimension_key, r.metric_id, r.window_start, r.object_ldn)
    r.id, r.device_oui, r.device_sn, r.metric_path, r.metric_type, r.metric_value,
    r.aggregation_op::text AS statis_type, r.granularity::text,
    r.window_start AS "time", r.window_start AS start_time, r.window_end AS end_time,
    r.created_at AS ingest_time, r.object_ldn,
    jsonb_build_object('task_id', r.task_id, 'task_version_id', r.task_version_id,
                       'complete', r.complete, 'missing_slots', r.missing_slots) AS extra
FROM public.pm_aggregation_results r
JOIN public.pm_aggregation_publications published_window
  ON published_window.task_version_id = r.task_version_id
 AND published_window.granularity = r.granularity
 AND published_window.window_start = r.window_start
 AND published_window.status = 'published'
 AND published_window.revision = r.revision
WHERE r.dimension = 'device' AND r.granularity = 'hourly'
ORDER BY r.dimension_key, r.metric_id, r.window_start, r.object_ldn, r.created_at DESC;

CREATE VIEW public.pm_metrics_daily AS
SELECT DISTINCT ON (r.dimension_key, r.metric_id, r.window_start, r.object_ldn)
    r.device_oui, r.device_sn, r.metric_path, r.metric_type, r.metric_value,
    r.aggregation_op::text AS statis_type, r.granularity::text,
    r.window_start AS "time", r.window_start AS start_time,
    r.window_end AS end_time, r.created_at AS ingest_time, r.object_ldn,
    jsonb_build_object('task_id', r.task_id, 'task_version_id', r.task_version_id,
                       'complete', r.complete, 'missing_slots', r.missing_slots) AS extra
FROM public.pm_aggregation_results r
JOIN public.pm_aggregation_publications published_window
  ON published_window.task_version_id = r.task_version_id
 AND published_window.granularity = r.granularity
 AND published_window.window_start = r.window_start
 AND published_window.status = 'published'
 AND published_window.revision = r.revision
WHERE r.dimension = 'device' AND r.granularity = 'daily'
ORDER BY r.dimension_key, r.metric_id, r.window_start, r.object_ldn, r.created_at DESC;

CREATE VIEW public.pm_metrics_weekly AS
SELECT DISTINCT ON (r.dimension_key, r.metric_id, r.window_start, r.object_ldn)
    r.device_oui, r.device_sn, r.metric_path, r.metric_type, r.metric_value,
    r.aggregation_op::text AS statis_type, r.granularity::text,
    r.window_start AS "time", r.window_start AS start_time,
    r.window_end AS end_time, r.created_at AS ingest_time, r.object_ldn,
    jsonb_build_object('task_id', r.task_id, 'task_version_id', r.task_version_id,
                       'complete', r.complete, 'missing_slots', r.missing_slots) AS extra
FROM public.pm_aggregation_results r
JOIN public.pm_aggregation_publications published_window
  ON published_window.task_version_id = r.task_version_id
 AND published_window.granularity = r.granularity
 AND published_window.window_start = r.window_start
 AND published_window.status = 'published'
 AND published_window.revision = r.revision
WHERE r.dimension = 'device' AND r.granularity = 'weekly'
ORDER BY r.dimension_key, r.metric_id, r.window_start, r.object_ldn, r.created_at DESC;

CREATE VIEW public.pm_metrics_monthly AS
SELECT DISTINCT ON (r.dimension_key, r.metric_id, r.window_start, r.object_ldn)
    r.device_oui, r.device_sn, r.metric_path, r.metric_type, r.metric_value,
    r.aggregation_op::text AS statis_type, r.granularity::text,
    r.window_start AS "time", r.window_start AS start_time,
    r.window_end AS end_time, r.created_at AS ingest_time, r.object_ldn,
    jsonb_build_object('task_id', r.task_id, 'task_version_id', r.task_version_id,
                       'complete', r.complete, 'missing_slots', r.missing_slots) AS extra
FROM public.pm_aggregation_results r
JOIN public.pm_aggregation_publications published_window
  ON published_window.task_version_id = r.task_version_id
 AND published_window.granularity = r.granularity
 AND published_window.window_start = r.window_start
 AND published_window.status = 'published'
 AND published_window.revision = r.revision
WHERE r.dimension = 'device' AND r.granularity = 'monthly'
ORDER BY r.dimension_key, r.metric_id, r.window_start, r.object_ldn, r.created_at DESC;

CREATE VIEW public.pm_group_metrics_hourly AS
SELECT DISTINCT ON (r.dimension_key, r.metric_id, r.window_start, r.technology)
    r.id,
    replace(r.dimension_key, 'DeviceGroup=', '')::uuid AS device_group_id,
    r.metric_path, r.metric_type, r.metric_value, r.aggregation_op::text AS statis_type,
    r.granularity::text, r.window_start AS "time", r.window_start AS start_time,
    r.window_end AS end_time, r.created_at AS ingest_time,
    jsonb_build_object('task_id', r.task_id, 'task_version_id', r.task_version_id,
                       'complete', r.complete, 'missing_slots', r.missing_slots) AS extra,
    r.technology
FROM public.pm_aggregation_results r
JOIN public.pm_aggregation_publications published_window
  ON published_window.task_version_id = r.task_version_id
 AND published_window.granularity = r.granularity
 AND published_window.window_start = r.window_start
 AND published_window.status = 'published'
 AND published_window.revision = r.revision
WHERE r.dimension = 'device_group' AND r.granularity = 'hourly'
ORDER BY r.dimension_key, r.metric_id, r.window_start, r.technology, r.created_at DESC;

CREATE VIEW public.pm_group_metrics_daily AS
SELECT DISTINCT ON (r.dimension_key, r.metric_id, r.window_start, r.technology)
    replace(r.dimension_key, 'DeviceGroup=', '')::uuid AS device_group_id,
    r.metric_path, r.metric_type, r.metric_value, r.aggregation_op::text AS statis_type,
    r.granularity::text, r.window_start AS "time", r.window_start AS start_time,
    r.window_end AS end_time, r.created_at AS ingest_time,
    jsonb_build_object('task_id', r.task_id, 'task_version_id', r.task_version_id,
                       'complete', r.complete, 'missing_slots', r.missing_slots) AS extra,
    r.technology
FROM public.pm_aggregation_results r
JOIN public.pm_aggregation_publications published_window
  ON published_window.task_version_id = r.task_version_id
 AND published_window.granularity = r.granularity
 AND published_window.window_start = r.window_start
 AND published_window.status = 'published'
 AND published_window.revision = r.revision
WHERE r.dimension = 'device_group' AND r.granularity = 'daily'
ORDER BY r.dimension_key, r.metric_id, r.window_start, r.technology, r.created_at DESC;

CREATE VIEW public.pm_group_metrics_weekly AS
SELECT DISTINCT ON (r.dimension_key, r.metric_id, r.window_start, r.technology)
    replace(r.dimension_key, 'DeviceGroup=', '')::uuid AS device_group_id,
    r.metric_path, r.metric_type, r.metric_value, r.aggregation_op::text AS statis_type,
    r.granularity::text, r.window_start AS "time", r.window_start AS start_time,
    r.window_end AS end_time, r.created_at AS ingest_time,
    jsonb_build_object('task_id', r.task_id, 'task_version_id', r.task_version_id,
                       'complete', r.complete, 'missing_slots', r.missing_slots) AS extra,
    r.technology
FROM public.pm_aggregation_results r
JOIN public.pm_aggregation_publications published_window
  ON published_window.task_version_id = r.task_version_id
 AND published_window.granularity = r.granularity
 AND published_window.window_start = r.window_start
 AND published_window.status = 'published'
 AND published_window.revision = r.revision
WHERE r.dimension = 'device_group' AND r.granularity = 'weekly'
ORDER BY r.dimension_key, r.metric_id, r.window_start, r.technology, r.created_at DESC;

CREATE VIEW public.pm_group_metrics_monthly AS
SELECT DISTINCT ON (r.dimension_key, r.metric_id, r.window_start, r.technology)
    replace(r.dimension_key, 'DeviceGroup=', '')::uuid AS device_group_id,
    r.metric_path, r.metric_type, r.metric_value, r.aggregation_op::text AS statis_type,
    r.granularity::text, r.window_start AS "time", r.window_start AS start_time,
    r.window_end AS end_time, r.created_at AS ingest_time,
    jsonb_build_object('task_id', r.task_id, 'task_version_id', r.task_version_id,
                       'complete', r.complete, 'missing_slots', r.missing_slots) AS extra,
    r.technology
FROM public.pm_aggregation_results r
JOIN public.pm_aggregation_publications published_window
  ON published_window.task_version_id = r.task_version_id
 AND published_window.granularity = r.granularity
 AND published_window.window_start = r.window_start
 AND published_window.status = 'published'
 AND published_window.revision = r.revision
WHERE r.dimension = 'device_group' AND r.granularity = 'monthly'
ORDER BY r.dimension_key, r.metric_id, r.window_start, r.technology, r.created_at DESC;


-- Consolidated from pre-release baseline-only migrations: TSDB schema 000002-000008

ALTER TABLE public.pm_files
    ADD COLUMN IF NOT EXISTS raw_deleted_at timestamptz,
    ADD COLUMN IF NOT EXISTS raw_delete_attempts integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS raw_delete_next_attempt_at timestamptz,
    ADD COLUMN IF NOT EXISTS raw_delete_last_error varchar(512);

ALTER TABLE public.mr_files
    ADD COLUMN IF NOT EXISTS raw_deleted_at timestamptz,
    ADD COLUMN IF NOT EXISTS raw_delete_attempts integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS raw_delete_next_attempt_at timestamptz,
    ADD COLUMN IF NOT EXISTS raw_delete_last_error varchar(512);

CREATE INDEX IF NOT EXISTS idx_pm_files_raw_cleanup
    ON public.pm_files (collect_time, id)
    WHERE raw_deleted_at IS NULL AND raw_delete_next_attempt_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_pm_files_raw_cleanup_retry
    ON public.pm_files (raw_delete_next_attempt_at, collect_time, id)
    WHERE raw_deleted_at IS NULL AND raw_delete_next_attempt_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_mr_files_created
    ON public.mr_files (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_mr_files_raw_cleanup
    ON public.mr_files (collect_time, id)
    WHERE raw_deleted_at IS NULL AND raw_delete_next_attempt_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_mr_files_raw_cleanup_retry
    ON public.mr_files (raw_delete_next_attempt_at, collect_time, id)
    WHERE raw_deleted_at IS NULL AND raw_delete_next_attempt_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_mr_records_file_time
    ON public.mr_records (file_id, "time" DESC);
CREATE INDEX IF NOT EXISTS idx_pm_anchors_source_file
    ON public.pm_measurement_anchors (source_file_id)
    WHERE source_file_id IS NOT NULL;

TRUNCATE TABLE
    public.pm_aggregation_rollup_outbox,
    public.pm_aggregation_counter_rollups,
    public.pm_aggregation_windows,
    public.pm_aggregation_results;

ALTER TABLE public.pm_aggregation_counter_rollups
    ADD COLUMN IF NOT EXISTS entity_key text;
ALTER TABLE public.pm_aggregation_counter_rollups
    ALTER COLUMN entity_key SET NOT NULL;

DROP INDEX IF EXISTS public.idx_pm_aggregation_counter_rollups_recovery;
CREATE INDEX idx_pm_aggregation_counter_rollups_recovery
    ON public.pm_aggregation_counter_rollups (
        task_version_id, entity_key, granularity, window_start, chunk_index
    );

ALTER TABLE public.pm_aggregation_windows
    ADD COLUMN IF NOT EXISTS entity_key text;
ALTER TABLE public.pm_aggregation_windows
    ALTER COLUMN entity_key SET NOT NULL;
ALTER TABLE public.pm_aggregation_windows
    DROP CONSTRAINT IF EXISTS pm_aggregation_windows_pkey;
ALTER TABLE public.pm_aggregation_windows
    ADD CONSTRAINT pm_aggregation_windows_pkey
    PRIMARY KEY (task_version_id, entity_key, granularity, window_start);

CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

CREATE INDEX IF NOT EXISTS idx_pm_aggregation_results_dashboard_network
    ON public.pm_aggregation_results (
        task_id,
        granularity,
        technology,
        metric_path,
        window_start DESC,
        created_at DESC
    )
    WHERE dimension = 'network';
CREATE INDEX IF NOT EXISTS idx_pm_aggregation_results_dashboard_product
    ON public.pm_aggregation_results (
        task_id,
        granularity,
        dimension_key,
        window_start DESC,
        metric_path,
        task_version_id,
        created_at DESC
    )
    WHERE dimension = 'product';
CREATE INDEX IF NOT EXISTS idx_pm_aggregation_results_dashboard_device_group
    ON public.pm_aggregation_results (
        task_id,
        granularity,
        dimension_key,
        window_start DESC,
        metric_path,
        task_version_id,
        created_at DESC
    )
    WHERE dimension = 'device_group';
CREATE INDEX IF NOT EXISTS idx_pm_aggregation_results_dashboard_band
    ON public.pm_aggregation_results (
        task_id,
        granularity,
        dimension_key,
        window_start DESC,
        metric_path,
        task_version_id,
        created_at DESC
    )
    WHERE dimension = 'band';

CREATE TABLE IF NOT EXISTS public.pm_file_quarantines (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_file_id uuid NOT NULL,
    device_sn text NOT NULL,
    declared_technology varchar(16) NOT NULL,
    detected_technology varchar(16) NOT NULL,
    reason varchar(64) NOT NULL,
    minio_path text NOT NULL,
    evidence jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_pm_file_quarantines_source_reason
        UNIQUE (source_file_id, reason)
);

CREATE INDEX IF NOT EXISTS idx_pm_file_quarantines_created_at
    ON public.pm_file_quarantines (created_at DESC);

ALTER TABLE public.pm_aggregation_outbox
    ADD COLUMN event_window_start timestamptz,
    ADD COLUMN event_window_end timestamptz,
    ADD COLUMN device_id uuid,
    ADD COLUMN consumed_at timestamptz,
    ADD COLUMN barrier_eligible boolean,
    ADD COLUMN claim_token uuid,
    ADD COLUMN claim_expires_at timestamptz,
    ADD COLUMN next_attempt_at timestamptz NOT NULL DEFAULT '-infinity';

UPDATE public.pm_aggregation_outbox
SET event_window_start = (payload->>'window_start')::timestamptz,
    event_window_end = (payload->>'window_end')::timestamptz,
    device_id = (payload->>'device_id')::uuid
WHERE event_window_start IS NULL
   OR event_window_end IS NULL
   OR device_id IS NULL;

UPDATE public.pm_aggregation_outbox SET barrier_eligible = false;

ALTER TABLE public.pm_aggregation_outbox
    ALTER COLUMN event_window_start SET NOT NULL,
    ALTER COLUMN event_window_end SET NOT NULL,
    ALTER COLUMN device_id SET NOT NULL,
    ALTER COLUMN barrier_eligible SET DEFAULT false,
    ALTER COLUMN barrier_eligible SET NOT NULL;

CREATE INDEX idx_pm_aggregation_outbox_consume_barrier
    ON public.pm_aggregation_outbox (event_window_start, created_at)
    WHERE consumed_at IS NULL AND barrier_eligible;
CREATE INDEX idx_pm_aggregation_outbox_claim_due
    ON public.pm_aggregation_outbox (next_attempt_at, created_at, event_id)
    WHERE published_at IS NULL AND consumed_at IS NULL;
CREATE INDEX idx_pm_aggregation_outbox_device_period_replay
    ON public.pm_aggregation_outbox (device_id, event_window_start, event_id)
    WHERE NOT barrier_eligible;

ALTER TABLE public.pm_aggregation_rollup_outbox
    ADD COLUMN consumed_at timestamptz,
    ADD COLUMN barrier_eligible boolean,
    ADD COLUMN claim_token uuid,
    ADD COLUMN claim_expires_at timestamptz,
    ADD COLUMN next_attempt_at timestamptz NOT NULL DEFAULT '-infinity';

UPDATE public.pm_aggregation_rollup_outbox
SET barrier_eligible = false;

ALTER TABLE public.pm_aggregation_rollup_outbox
    ALTER COLUMN barrier_eligible SET DEFAULT false,
    ALTER COLUMN barrier_eligible SET NOT NULL;

CREATE INDEX idx_pm_rollup_outbox_publication
    ON public.pm_aggregation_rollup_outbox (
        publication_task_version_id, granularity, window_start, revision
    ) WHERE NOT barrier_eligible;

CREATE INDEX idx_pm_aggregation_rollup_consume_barrier
    ON public.pm_aggregation_rollup_outbox (subject, window_start)
    WHERE consumed_at IS NULL AND barrier_eligible;
CREATE INDEX idx_pm_aggregation_rollup_claim_due
    ON public.pm_aggregation_rollup_outbox (next_attempt_at, created_at, event_id)
    WHERE published_at IS NULL AND consumed_at IS NULL AND barrier_eligible;

ALTER TABLE public.pm_aggregation_windows
    ADD COLUMN revision integer NOT NULL DEFAULT 1,
    ADD COLUMN published_revision integer NOT NULL DEFAULT 0,
    ADD COLUMN rebuild_requested_at timestamptz,
    ADD COLUMN version_effective_from timestamptz,
    ADD COLUMN version_effective_to timestamptz,
    ADD COLUMN finalize_lease_owner uuid,
    ADD COLUMN finalize_lease_until timestamptz,
    ADD COLUMN finalize_attempts integer NOT NULL DEFAULT 0,
    ADD COLUMN finalize_next_attempt_at timestamptz NOT NULL DEFAULT '-infinity',
    ADD COLUMN version_audit_fingerprint text,
    ADD COLUMN recovery_original_status varchar(16),
    ADD COLUMN recovery_terminal_at timestamptz,
    ADD COLUMN recovery_terminal_reason text,
    ADD COLUMN runtime_cleaned_at timestamptz;

CREATE INDEX IF NOT EXISTS idx_pm_aggregation_windows_active_version_guard
    ON public.pm_aggregation_windows (
        task_id,
        granularity,
        window_start,
        version_effective_from DESC,
        task_version_id
    )
    WHERE status IN ('open', 'finalizing', 'prepared', 'rebuilding', 'failed')
      AND version_effective_from IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_pm_aggregation_windows_dashboard_published
    ON public.pm_aggregation_windows (
        task_id,
        granularity,
        window_start,
        task_version_id,
        entity_key,
        version_effective_from DESC,
        revision DESC,
        published_at DESC,
        updated_at DESC
    )
    WHERE status = 'published';

ALTER TABLE public.pm_aggregation_windows
    DROP CONSTRAINT chk_pm_aggregation_windows_status;
ALTER TABLE public.pm_aggregation_windows
    ADD CONSTRAINT chk_pm_aggregation_windows_status
        CHECK (status IN (
            'open', 'finalizing', 'prepared', 'published', 'failed', 'rebuilding',
            'orphaned', 'retired', 'abandoned'
        ));

CREATE INDEX idx_pm_windows_prepared_publication
    ON public.pm_aggregation_windows (
        task_version_id, granularity, window_start, revision, entity_key
    ) WHERE status = 'prepared';

CREATE INDEX idx_pm_windows_unready_publication
    ON public.pm_aggregation_windows (
        task_version_id, granularity, window_start, status
    ) WHERE status NOT IN ('prepared', 'published');

CREATE INDEX idx_pm_counter_rollups_publication
    ON public.pm_aggregation_counter_rollups (
        task_version_id, granularity, window_start, revision
    ) WHERE NOT publication_eligible;

CREATE INDEX idx_pm_windows_due_claim
    ON public.pm_aggregation_windows (
        granularity, finalize_next_attempt_at, window_end,
        task_version_id, entity_key, window_start
    )
    WHERE status IN ('open', 'failed');

CREATE INDEX idx_pm_windows_oldest_due
    ON public.pm_aggregation_windows (granularity, window_end)
    WHERE status IN ('open', 'failed');

CREATE INDEX idx_pm_windows_recovery_cleanup_pending
    ON public.pm_aggregation_windows (
        recovery_terminal_at, task_version_id, entity_key, granularity, window_start
    )
    WHERE status IN ('orphaned', 'retired')
      AND runtime_cleaned_at IS NULL;

-- The finalizer alternates oldest/newest scans. Keep every stable ordering
-- column in one partial index so PostgreSQL can scan it in either direction
-- without sorting the full due hour again for every claimed batch.
CREATE INDEX idx_pm_windows_due_claim_order
    ON public.pm_aggregation_windows (granularity, window_end, task_version_id, entity_key, window_start)
    INCLUDE (finalize_next_attempt_at, finalize_lease_until)
    WHERE status IN ('open', 'failed');

-- Hourly rule windows must not close before their device-hour source windows
-- have published every durable rollup. This partial index keeps that
-- correlated hierarchy watermark bounded while the 20k-device hour drains.
CREATE INDEX idx_pm_windows_hourly_source_barrier
    ON public.pm_aggregation_windows (granularity, window_start, task_version_id)
    WHERE status IN ('open', 'failed', 'finalizing', 'rebuilding');

CREATE INDEX idx_pm_windows_version_audit
    ON public.pm_aggregation_windows (
        task_version_id, version_audit_fingerprint,
        window_start, entity_key, granularity
    )
    WHERE status = 'published'
      AND granularity IN ('daily', 'weekly', 'monthly');

ALTER TABLE public.pm_aggregation_results
    ADD COLUMN version_effective_from timestamptz,
    ADD COLUMN version_effective_to timestamptz,
    ADD COLUMN received_slots bigint NOT NULL DEFAULT 0,
    ADD COLUMN expected_slots bigint NOT NULL DEFAULT 0,
    ADD COLUMN version_expected_slots bigint NOT NULL DEFAULT 0,
    ADD COLUMN natural_expected_slots bigint NOT NULL DEFAULT 0,
    ADD COLUMN version_slice_complete boolean NOT NULL DEFAULT false,
    ADD COLUMN period_complete boolean NOT NULL DEFAULT false;

CREATE TABLE public.pm_aggregation_rebuilds (
    id bigserial PRIMARY KEY,
    task_id uuid NOT NULL,
    task_version_id uuid NOT NULL,
    entity_key text NOT NULL,
    granularity varchar(16) NOT NULL,
    window_start timestamptz NOT NULL,
    window_end timestamptz NOT NULL,
    source_event_id text NOT NULL,
    request_generation bigint NOT NULL DEFAULT 1,
    status varchar(16) NOT NULL DEFAULT 'pending',
    attempts integer NOT NULL DEFAULT 0,
    last_error text,
    requested_at timestamptz NOT NULL DEFAULT now(),
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    started_at timestamptz,
    lease_expires_at timestamptz,
    lease_owner uuid,
    completed_at timestamptz,
    CONSTRAINT chk_pm_aggregation_rebuilds_granularity
        CHECK (granularity IN ('hourly', 'daily', 'weekly', 'monthly')),
    CONSTRAINT chk_pm_aggregation_rebuilds_status
        CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    UNIQUE (task_version_id, entity_key, granularity, window_start)
);

CREATE INDEX idx_pm_aggregation_rebuilds_pending
    ON public.pm_aggregation_rebuilds (next_attempt_at, requested_at, id)
    WHERE status IN ('pending', 'failed', 'running');

DROP INDEX IF EXISTS public.idx_pm_aggregation_outbox_consume_barrier;
CREATE INDEX idx_pm_aggregation_outbox_consume_barrier
    ON public.pm_aggregation_outbox (event_window_start, created_at)
    WHERE consumed_at IS NULL AND barrier_eligible;
DROP INDEX IF EXISTS public.idx_pm_aggregation_rollup_consume_barrier;
CREATE INDEX idx_pm_aggregation_rollup_consume_barrier
    ON public.pm_aggregation_rollup_outbox (subject, window_start)
    WHERE consumed_at IS NULL AND barrier_eligible;

CREATE TABLE public.pm_aggregation_replay_sources (
    event_window_start timestamptz NOT NULL,
    event_id uuid NOT NULL,
    device_id uuid,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (event_window_start, event_id)
);

UPDATE public.pm_aggregation_replay_sources
SET device_id = (payload->>'device_id')::uuid
WHERE device_id IS NULL;
ALTER TABLE public.pm_aggregation_replay_sources
    ALTER COLUMN device_id SET NOT NULL;

CREATE INDEX idx_pm_replay_sources_device_period
    ON public.pm_aggregation_replay_sources (device_id, event_window_start, event_id);

SELECT create_hypertable(
    'public.pm_aggregation_replay_sources',
    by_range('event_window_start', INTERVAL '1 day'),
    if_not_exists => TRUE,
    migrate_data => TRUE
);

ALTER TABLE public.pm_aggregation_replay_sources SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'device_id',
    timescaledb.compress_orderby = 'event_window_start, event_id'
);
SELECT add_compression_policy(
    'public.pm_aggregation_replay_sources',
    INTERVAL '1 day',
    if_not_exists => TRUE
);

CREATE OR REPLACE VIEW public.pm_adhoc_aggregation_results AS
SELECT
    r.id, r.task_id, r.device_oui,
    CASE WHEN r.dimension = 'device' THEN r.device_sn ELSE 'AGGREGATED' END::text AS device_sn,
    CASE WHEN r.dimension = 'product' THEN r.dimension_key::uuid ELSE NULL::uuid END AS product_id,
    r.metric_path, r.metric_type, r.metric_value,
    r.aggregation_op::text AS statis_type, r.granularity::text,
    r.window_start AS "time", r.window_start AS start_time, r.window_end AS end_time,
    r.created_at AS ingest_time,
    CASE
        WHEN r.dimension = 'device' THEN NULLIF(r.object_ldn, '')
        WHEN r.dimension = 'network' THEN 'Network'
        ELSE r.dimension_key
    END::text AS object_ldn,
    jsonb_build_object(
        'task_version_id', r.task_version_id,
        'complete', r.period_complete,
        'missing_slots', r.missing_slots,
        'dimension', r.dimension,
        'revision', r.revision,
        'version_effective_from', r.version_effective_from,
        'version_effective_to', r.version_effective_to,
        'received_slots', r.received_slots,
        'expected_slots', r.expected_slots,
        'version_expected_slots', r.version_expected_slots,
        'natural_expected_slots', r.natural_expected_slots,
        'version_slice_complete', r.version_slice_complete,
        'period_complete', r.period_complete,
        'active_version', r.task_version_id = (
            SELECT candidate.task_version_id
            FROM public.pm_aggregation_results candidate
            WHERE candidate.task_id = r.task_id
              AND candidate.granularity = r.granularity
              AND candidate.window_start = r.window_start
            GROUP BY candidate.task_version_id, candidate.version_effective_from
            ORDER BY candidate.version_effective_from DESC NULLS LAST,
                     MAX(candidate.revision) DESC,
                     MAX(candidate.created_at) DESC,
                     candidate.task_version_id DESC
            LIMIT 1
        ) AND NOT EXISTS (
            SELECT 1
            FROM public.pm_aggregation_windows active_window
            WHERE active_window.task_id = r.task_id
              AND active_window.granularity = r.granularity
              AND active_window.window_start = r.window_start
              AND active_window.task_version_id <> r.task_version_id
              AND active_window.status IN ('open', 'finalizing', 'prepared', 'rebuilding', 'failed')
              AND active_window.version_effective_from IS NOT NULL
              AND (
                  r.version_effective_from IS NULL
                  OR active_window.version_effective_from > r.version_effective_from
              )
        ),
        'partial', false
    ) AS extra
FROM public.pm_aggregation_results r
JOIN public.pm_aggregation_publications published_window
  ON published_window.task_version_id = r.task_version_id
 AND published_window.granularity = r.granularity
 AND published_window.window_start = r.window_start
 AND published_window.status = 'published'
 AND published_window.revision = r.revision;

CREATE INDEX IF NOT EXISTS idx_pm_aggregation_outbox_unacknowledged
    ON public.pm_aggregation_outbox (published_at, event_id)
    WHERE consumed_at IS NULL
      AND barrier_eligible
      AND published_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_pm_aggregation_rollup_unacknowledged
    ON public.pm_aggregation_rollup_outbox (published_at, event_id)
    WHERE consumed_at IS NULL
      AND barrier_eligible
      AND published_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_pm_aggregation_outbox_retention_consumed
    ON public.pm_aggregation_outbox (published_at, event_id)
    WHERE consumed_at IS NOT NULL
      AND barrier_eligible;

CREATE INDEX IF NOT EXISTS idx_pm_aggregation_outbox_retention_legacy
    ON public.pm_aggregation_outbox (published_at, event_id)
    WHERE NOT barrier_eligible
      AND published_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_pm_aggregation_rollup_retention_consumed
    ON public.pm_aggregation_rollup_outbox (published_at, event_id)
    WHERE consumed_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_pm_aggregation_windows_published_retention
    ON public.pm_aggregation_windows (
        published_at, window_start, task_version_id, entity_key, granularity
    )
    WHERE status = 'published';

CREATE INDEX IF NOT EXISTS idx_pm_aggregation_windows_recovery_retention
    ON public.pm_aggregation_windows (
        recovery_terminal_at, window_start, task_version_id, entity_key, granularity
    )
    WHERE status IN ('retired', 'orphaned');

CREATE INDEX IF NOT EXISTS idx_pm_aggregation_windows_abandoned_retention
    ON public.pm_aggregation_windows (
        updated_at, window_start, task_version_id, entity_key, granularity
    )
    WHERE status = 'abandoned';


-- +goose Down
-- DROP 全部对象（同名视图 → matview + 函数 → 镜像/归库表 → 影子表 → 15 张时序表；策略随 DROP TABLE 级联消失）。
DROP VIEW IF EXISTS public.alarm_definitions;
DROP VIEW IF EXISTS public.cell_band;
DROP VIEW IF EXISTS public.device_groups;
DROP VIEW IF EXISTS public.products;
DROP VIEW IF EXISTS public.device_group_members;
DROP VIEW IF EXISTS public.devices;
DROP VIEW IF EXISTS public.pm_metrics_hourly;
DROP VIEW IF EXISTS public.pm_metrics_daily;
DROP VIEW IF EXISTS public.pm_metrics_weekly;
DROP VIEW IF EXISTS public.pm_metrics_monthly;
DROP VIEW IF EXISTS public.pm_group_metrics_hourly;
DROP VIEW IF EXISTS public.pm_group_metrics_daily;
DROP VIEW IF EXISTS public.pm_group_metrics_weekly;
DROP VIEW IF EXISTS public.pm_group_metrics_monthly;
DROP VIEW IF EXISTS public.pm_adhoc_aggregation_results;
DROP VIEW IF EXISTS public.pm_metrics;

DROP TABLE IF EXISTS public.perf_indicators_gsm;
DROP TABLE IF EXISTS public.perf_indicators_gnb;
DROP TABLE IF EXISTS public.perf_indicators_enb;
DROP TABLE IF EXISTS public.kpi_definitions;
DROP TABLE IF EXISTS public.trace_export_jobs;
DROP TABLE IF EXISTS public.trace_tasks;

DROP FUNCTION IF EXISTS public.refresh_alarm_efficiency_metrics();
DROP MATERIALIZED VIEW IF EXISTS public.alarm_efficiency_metrics;

DROP TABLE IF EXISTS public.mr_customize_task_dim;
DROP TABLE IF EXISTS public.alarm_definition_dim;
DROP TABLE IF EXISTS public.device_group_dim;
DROP TABLE IF EXISTS public.product_dim;
DROP TABLE IF EXISTS public.cell_band_dim;
DROP TABLE IF EXISTS public.device_group_member_dim;
DROP TABLE IF EXISTS public.device_dim;

DROP TABLE IF EXISTS public.pm_adhoc_aggregation_results;
DROP TABLE IF EXISTS public.pm_group_metrics_monthly;
DROP TABLE IF EXISTS public.pm_group_metrics_weekly;
DROP TABLE IF EXISTS public.pm_group_metrics_daily;
DROP TABLE IF EXISTS public.pm_group_metrics_hourly;
DROP TABLE IF EXISTS public.pm_metrics_monthly;
DROP TABLE IF EXISTS public.pm_metrics_weekly;
DROP TABLE IF EXISTS public.pm_metrics_daily;
DROP TABLE IF EXISTS public.pm_hourly_values;
DROP TABLE IF EXISTS public.pm_hourly_anchors;
DROP TABLE IF EXISTS public.pm_hourly_rollup_batches;
DROP TABLE IF EXISTS public.pm_hourly_bucket_versions;
DROP TABLE IF EXISTS public.pm_aggregation_replay_sources;
DROP TABLE IF EXISTS public.pm_aggregation_rebuilds;
DROP TABLE IF EXISTS public.pm_aggregation_publications;
DROP TABLE IF EXISTS public.pm_file_quarantines;
DROP TABLE IF EXISTS public.pm_aggregation_rollup_outbox;
DROP TABLE IF EXISTS public.pm_aggregation_counter_rollups;
DROP TABLE IF EXISTS public.pm_aggregation_results;
DROP TABLE IF EXISTS public.pm_aggregation_windows;
DROP TABLE IF EXISTS public.pm_aggregation_outbox;
DROP TABLE IF EXISTS public.pm_metric_values;
DROP TABLE IF EXISTS public.pm_measurement_anchors;
DROP TABLE IF EXISTS public.pm_ingest_batches;
DROP TABLE IF EXISTS public.pm_metric_sets;
DROP TABLE IF EXISTS public.pm_metric_dictionary;
DROP TABLE IF EXISTS public.pm_slot_health;
DROP TABLE IF EXISTS public.pm_files;
DROP TABLE IF EXISTS public.mr_files;
DROP TABLE IF EXISTS public.trace_messages;
DROP TABLE IF EXISTS public.mr_records;
DROP TABLE IF EXISTS public.alarms_history;
