-- +goose Up
-- 主库 consolidated baseline（2026-07-20；纯 PostgreSQL 16；时序对象位于 migrations/tsdb）。
--
-- PostgreSQL database dump
--


-- Dumped from database version 16.14
-- Dumped by pg_dump version 16.14

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: ltree; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS ltree WITH SCHEMA public;


--
-- Name: EXTENSION ltree; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION ltree IS 'data type for hierarchical tree-like structures';


--
-- Name: pg_trgm; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;


--
-- Name: EXTENSION pg_trgm; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pg_trgm IS 'text similarity measurement and index searching based on trigrams';


--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;


--
-- Name: EXTENSION "uuid-ossp"; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';


--
-- Name: build_v_type_string(character varying, jsonb); Type: FUNCTION; Schema: public; Owner: -
--

-- +goose StatementBegin
CREATE FUNCTION public.build_v_type_string(v_type character varying, v_constraint jsonb) RETURNS text
    LANGUAGE plpgsql IMMUTABLE
    AS $$
DECLARE
    result TEXT;
    labels TEXT[];
    values TEXT[];
    min_val INT;
    max_val INT;
    min_len INT;
    max_len INT;
BEGIN
    IF v_constraint IS NULL THEN
        RETURN v_type;
    END IF;

    -- enum 类型
    IF v_type = 'enum' THEN
        labels := v_constraint->>'labels';
        values := v_constraint->>'values';
        IF labels IS NOT NULL AND values IS NOT NULL THEN
            result := 'enum-{' || array_to_string(labels, ',') || '}-{' || array_to_string(values, ',') || '}';
            RETURN result;
        END IF;
    END IF;

    -- 带范围的数值类型
    IF v_type IN ('unsignedInt', 'int', 'uniqueInt') THEN
        min_val := (v_constraint->>'min')::INT;
        max_val := (v_constraint->>'max')::INT;
        result := v_type || '-[';
        IF min_val IS NOT NULL THEN
            result := result || min_val;
        END IF;
        result := result || ':';
        IF max_val IS NOT NULL THEN
            result := result || max_val;
        END IF;
        result := result || ']';
        RETURN result;
    END IF;

    -- string 带长度
    IF v_type = 'string' THEN
        min_len := (v_constraint->>'min_length')::INT;
        max_len := (v_constraint->>'max_length')::INT;
        result := 'string-[';
        IF min_len IS NOT NULL THEN
            result := result || min_len;
        END IF;
        result := result || ':';
        IF max_len IS NOT NULL THEN
            result := result || max_len;
        END IF;
        result := result || ']';
        RETURN result;
    END IF;

    -- 默认返回类型
    RETURN v_type;
END;
$$;
-- +goose StatementEnd


--
-- Name: nedirect_sessions_updated_at(); Type: FUNCTION; Schema: public; Owner: -
--

-- +goose StatementBegin
CREATE FUNCTION public.nedirect_sessions_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;
-- +goose StatementEnd


--
-- Name: parse_v_type(text); Type: FUNCTION; Schema: public; Owner: -
--

-- +goose StatementBegin
CREATE FUNCTION public.parse_v_type(v_type_str text) RETURNS TABLE(value_type character varying, value_constraint jsonb)
    LANGUAGE plpgsql IMMUTABLE
    AS $_$
DECLARE
    result_type VARCHAR(50);
    result_constraint JSONB;
    parts TEXT[];
    main_part TEXT;
    range_part TEXT;
    enum_labels TEXT[];
    enum_values TEXT[];
    min_val TEXT;
    max_val TEXT;
    max_len INT;
BEGIN
    IF v_type_str IS NULL OR v_type_str = '' THEN
        RETURN QUERY SELECT 'string'::VARCHAR(50), '{"type": "string"}'::JSONB;
        RETURN;
    END IF;

    -- 解析 enum 类型: enum-{label1,label2}-{value1,value2}
    IF v_type_str LIKE 'enum-%' THEN
        parts := regexp_match(v_type_str, '^enum-\{([^}]*)\}-\{([^}]*)\}$');
        IF parts IS NOT NULL THEN
            enum_labels := string_to_array(parts[1], ',');
            enum_values := string_to_array(parts[2], ',');
            result_type := 'enum';
            result_constraint := jsonb_build_object(
                'type', 'enum',
                'labels', enum_labels,
                'values', enum_values
            );
            RETURN QUERY SELECT result_type, result_constraint;
            RETURN;
        END IF;
    END IF;

    -- 解析带范围的类型: type-[min:max]
    IF v_type_str LIKE '%-[%' THEN
        parts := regexp_match(v_type_str, '^([^-]+)-\[([^:]*):([^\]]*)\]$');
        IF parts IS NOT NULL THEN
            main_part := parts[1];
            min_val := parts[2];
            max_val := parts[3];

            result_type := main_part;
            result_constraint := jsonb_build_object('type', main_part);

            IF min_val != '' THEN
                result_constraint := result_constraint || jsonb_build_object('min', min_val::INT);
            END IF;
            IF max_val != '' THEN
                result_constraint := result_constraint || jsonb_build_object('max', max_val::INT);
            END IF;

            RETURN QUERY SELECT result_type, result_constraint;
            RETURN;
        END IF;
    END IF;

    -- 解析 string 带长度: string-[0:256]
    IF v_type_str LIKE 'string-%' THEN
        parts := regexp_match(v_type_str, '^string-\[([^:]*):([^\]]*)\]$');
        IF parts IS NOT NULL THEN
            min_val := parts[1];
            max_val := parts[2];

            result_type := 'string';
            result_constraint := jsonb_build_object('type', 'string');

            IF min_val != '' THEN
                result_constraint := result_constraint || jsonb_build_object('min_length', min_val::INT);
            END IF;
            IF max_val != '' THEN
                result_constraint := result_constraint || jsonb_build_object('max_length', max_val::INT);
            END IF;

            RETURN QUERY SELECT result_type, result_constraint;
            RETURN;
        END IF;
    END IF;

    -- 简单类型(无约束)
    result_type := v_type_str;
    result_constraint := jsonb_build_object('type', v_type_str);
    RETURN QUERY SELECT result_type, result_constraint;
END;
$_$;
-- +goose StatementEnd


--
-- Name: refresh_mml_command_target_paths(uuid); Type: FUNCTION; Schema: public; Owner: -
--

-- +goose StatementBegin
CREATE FUNCTION public.refresh_mml_command_target_paths(p_command_id uuid) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE mml_commands c
    SET target_paths = COALESCE((
            SELECT jsonb_agg(sp.standard_path ORDER BY csf.sort_order)
            FROM mml_command_sub_fields csf
            JOIN standard_params sp ON sp.id = csf.standard_path_id
            WHERE csf.command_id = p_command_id
        ), '[]'::jsonb),
        updated_at = NOW()
    WHERE c.id = p_command_id;
END;
$$;
-- +goose StatementEnd


--
-- Name: trg_mml_sub_fields_refresh_paths(); Type: FUNCTION; Schema: public; Owner: -
--

-- +goose StatementBegin
CREATE FUNCTION public.trg_mml_sub_fields_refresh_paths() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM refresh_mml_command_target_paths(OLD.command_id);
        RETURN OLD;
    ELSE
        PERFORM refresh_mml_command_target_paths(NEW.command_id);
        RETURN NEW;
    END IF;
END;
$$;
-- +goose StatementEnd


--
-- Name: update_updated_at_column(); Type: FUNCTION; Schema: public; Owner: -
--

-- +goose StatementBegin
CREATE FUNCTION public.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;
-- +goose StatementEnd


SET default_table_access_method = heap;

--
-- Name: alarm_definitions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.alarm_definitions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    identifier character varying(32) NOT NULL,
    ne_type character varying(16) NOT NULL,
    cn_name character varying(256),
    en_name character varying(256),
    severity_id uuid NOT NULL,
    event_type integer,
    cn_probable_cause text,
    en_probable_cause text,
    is_show boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    loaded_from character varying(256),
    description text
);


--
-- Name: TABLE alarm_definitions; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.alarm_definitions IS 'T-0098 告警定义主表（设计 §3.2.2）；442 行典型规模，由 8 个 ne_type XML 文件载入';


--
-- Name: COLUMN alarm_definitions.identifier; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.alarm_definitions.identifier IS '告警全局唯一标识，跨 ne_type 唯一（设计 §3.4 加载流程校验）';


--
-- Name: COLUMN alarm_definitions.ne_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.alarm_definitions.ne_type IS '网元类型：ENB / GSM / GNB / OMC / EPC / EGW / CPE / UPS';


--
-- Name: COLUMN alarm_definitions.severity_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.alarm_definitions.severity_id IS 'ON DELETE RESTRICT：不允许误删严重级致告警定义悬挂';


--
-- Name: COLUMN alarm_definitions.loaded_from; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.alarm_definitions.loaded_from IS 'Loader 来源 XML 相对路径(含目录前缀,如 "alarm-definitions/ENB.xml" 或 "alarm-definitions-custom/MY.xml");NULL/空 表示运维手工新增(无 XML 来源)。ClassifySource 据前缀判定 builtin/custom。';


--
-- Name: alarm_filters; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.alarm_filters (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(255) NOT NULL,
    filter_type character varying(32) NOT NULL,
    alarm_sources character varying(64)[] DEFAULT '{}'::character varying[],
    alarm_identifiers character varying(128)[] DEFAULT '{}'::character varying[],
    device_ids uuid[] DEFAULT '{}'::uuid[],
    device_group_ids uuid[] DEFAULT '{}'::uuid[],
    action character varying(32) NOT NULL,
    acknowledge_desc text,
    priority integer DEFAULT 0 NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    created_by character varying(128),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by character varying(128),
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    webhook_url text,
    webhook_secret text,
    email_recipients text[],
    CONSTRAINT chk_alarm_filters_webhook_url_required CHECK ((((action)::text <> 'notify_webhook'::text) OR ((webhook_url IS NOT NULL) AND (webhook_url <> ''::text))))
);


--
-- Name: alarm_rules; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.alarm_rules (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    alarm_identifier character varying(128),
    severity integer DEFAULT 4 NOT NULL,
    condition_type character varying(64) NOT NULL,
    condition_config jsonb DEFAULT '{}'::jsonb NOT NULL,
    action_type character varying(64) NOT NULL,
    action_config jsonb DEFAULT '{}'::jsonb,
    carrier character varying(16),
    technology character varying(16),
    enabled boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: alarm_severity_levels; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.alarm_severity_levels (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    code integer NOT NULL,
    name character varying(16) NOT NULL,
    display_order integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE alarm_severity_levels; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.alarm_severity_levels IS 'T-0098 告警严重级参考表（设计 §3.2.1）；4 行种子（Critical/Major/Minor/Warning）';


--
-- Name: COLUMN alarm_severity_levels.code; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.alarm_severity_levels.code IS '运营商规范固定码 31001-31004，跨系统稳定';


--
-- Name: alarm_webhook_dead_letters; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.alarm_webhook_dead_letters (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    filter_id uuid NOT NULL,
    alarm_id uuid NOT NULL,
    payload jsonb NOT NULL,
    last_error text NOT NULL,
    retry_count integer DEFAULT 0 NOT NULL,
    failed_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: alarms_active; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.alarms_active (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_id uuid NOT NULL,
    device_sn character varying(64) NOT NULL,
    carrier character varying(4) NOT NULL,
    severity smallint NOT NULL,
    alarm_type character varying(64) DEFAULT ''::character varying NOT NULL,
    alarm_identifier character varying(64) NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    status character varying(16) DEFAULT 'active'::character varying NOT NULL,
    raised_at timestamp with time zone NOT NULL,
    acknowledged_at timestamp with time zone,
    acknowledged_by character varying(128),
    additional_info jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    device_name character varying(128),
    technology character varying(16),
    alarm_source character varying(64),
    event_type character varying(64),
    probable_cause text DEFAULT ''::text NOT NULL,
    network_location text,
    explicit_cause text,
    is_read boolean DEFAULT false NOT NULL,
    ack_count integer DEFAULT 0 NOT NULL,
    first_raised_at timestamp with time zone DEFAULT now() NOT NULL,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    ack_note text DEFAULT ''::text,
    is_unknown boolean DEFAULT false NOT NULL
);


--
-- Name: COLUMN alarms_active.is_unknown; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.alarms_active.is_unknown IS 'T-0098 fallback 标记：identifier 不在告警库时 product.enable_unknown_alarm=true 路径写入；治理闭环过滤依据';


--
-- Name: api_endpoints; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.api_endpoints (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    path character varying(256) NOT NULL,
    method character varying(16) NOT NULL,
    name character varying(128) DEFAULT ''::character varying,
    description text DEFAULT ''::text,
    api_group character varying(64) DEFAULT ''::character varying,
    is_auto boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    is_user_modified boolean DEFAULT false NOT NULL
);


--
-- Name: TABLE api_endpoints; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.api_endpoints IS 'API 端点注册表';


--
-- Name: COLUMN api_endpoints.path; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.api_endpoints.path IS 'API 路径';


--
-- Name: COLUMN api_endpoints.method; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.api_endpoints.method IS 'HTTP 方法 (GET/POST/PUT/DELETE)';


--
-- Name: COLUMN api_endpoints.name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.api_endpoints.name IS 'API 名称/简介';


--
-- Name: COLUMN api_endpoints.description; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.api_endpoints.description IS 'API 详细描述';


--
-- Name: COLUMN api_endpoints.api_group; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.api_endpoints.api_group IS 'API 分组名称';


--
-- Name: COLUMN api_endpoints.is_auto; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.api_endpoints.is_auto IS '是否自动扫描生成';


--
-- Name: COLUMN api_endpoints.is_user_modified; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.api_endpoints.is_user_modified IS 'true=name/api_group 被用户在 UI 手工改过；Sync 扫描不再用自动推断值覆盖。Update 改 name/api_group 时置 true。';


--
-- Name: api_keys; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.api_keys (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    key_prefix character varying(8) NOT NULL,
    key_hash character varying(255) NOT NULL,
    scopes text[] DEFAULT '{}'::text[] NOT NULL,
    expires_at timestamp with time zone,
    last_used_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    revoked_at timestamp with time zone
);


--
-- Name: async_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.async_jobs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    job_type text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    schedule_expr text,
    scheduled_at timestamp with time zone NOT NULL,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    heartbeat_at timestamp with time zone,
    lock_owner text,
    attempt integer DEFAULT 1 NOT NULL,
    max_attempts integer DEFAULT 3 NOT NULL,
    recovery_count integer DEFAULT 0 NOT NULL,
    last_recovered_at timestamp with time zone,
    payload jsonb,
    result jsonb,
    error_message text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    bucket_start timestamp with time zone,
    bucket_end timestamp with time zone,
    CONSTRAINT async_jobs_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'running'::text, 'succeeded'::text, 'failed'::text, 'zombie'::text, 'canceled'::text])))
);


--
-- Name: TABLE async_jobs; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.async_jobs IS 'T-0164-P8 / G8 通用异步任务总线（承载 G5 cron + 未来批量计算）';


--
-- Name: COLUMN async_jobs.status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.async_jobs.status IS 'pending/running/succeeded/failed/zombie/canceled';


--
-- Name: COLUMN async_jobs.heartbeat_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.async_jobs.heartbeat_at IS 'running 期间每 30s 更新；sweeper 用 heartbeat_at < now()-5min 识别僵尸';


--
-- Name: COLUMN async_jobs.lock_owner; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.async_jobs.lock_owner IS 'worker 进程标识 hostname-pid，便于排查"哪个 worker 抢到任务"';


--
-- Name: COLUMN async_jobs.bucket_start; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.async_jobs.bucket_start IS 'Bucket job source window start, for idempotent enqueue.';


--
-- Name: COLUMN async_jobs.bucket_end; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.async_jobs.bucket_end IS 'Bucket job source window end, for idempotent enqueue.';


--
-- Name: async_jobs_cron_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.async_jobs_cron_state (
    job_type text NOT NULL,
    cron_expr text NOT NULL,
    last_triggered_at timestamp with time zone NOT NULL,
    last_bucket_end timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: audit_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.audit_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid,
    username character varying(64) NOT NULL,
    action character varying(64) NOT NULL,
    resource character varying(64),
    resource_id character varying(128),
    details jsonb,
    ip_address inet,
    user_agent character varying(256),
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: backup_policies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.backup_policies (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    retention_days integer DEFAULT 30 NOT NULL,
    max_backup_count integer DEFAULT 100 NOT NULL,
    min_backup_count integer DEFAULT 3 NOT NULL,
    auto_cleanup boolean DEFAULT true NOT NULL,
    cleanup_time character varying(8) DEFAULT '03:00'::character varying NOT NULL,
    cleanup_day_of_week integer DEFAULT '-1'::integer NOT NULL,
    keep_last_n integer DEFAULT 5 NOT NULL,
    enable_compression boolean DEFAULT true NOT NULL,
    compression_level integer DEFAULT 6 NOT NULL,
    compression_format character varying(16) DEFAULT 'gzip'::character varying NOT NULL,
    storage_backend character varying(16) DEFAULT 'local'::character varying NOT NULL,
    ftp_config_id uuid,
    local_path text DEFAULT '/var/backup/omc'::text NOT NULL,
    max_storage_gb integer DEFAULT 500 NOT NULL,
    enable_encryption boolean DEFAULT false NOT NULL,
    encryption_algorithm character varying(32) DEFAULT 'AES-256-GCM'::character varying NOT NULL,
    alert_on_failure boolean DEFAULT true NOT NULL,
    alert_email character varying(256) DEFAULT ''::character varying NOT NULL,
    alert_threshold_percent integer DEFAULT 80 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    alert_severity character varying(16) DEFAULT 'major'::character varying NOT NULL,
    CONSTRAINT backup_policies_alert_severity_check CHECK (((alert_severity)::text = ANY (ARRAY[('warning'::character varying)::text, ('major'::character varying)::text, ('critical'::character varying)::text]))),
    CONSTRAINT backup_policies_alert_threshold_percent_check CHECK (((alert_threshold_percent >= 50) AND (alert_threshold_percent <= 95))),
    CONSTRAINT backup_policies_cleanup_day_of_week_check CHECK (((cleanup_day_of_week >= '-1'::integer) AND (cleanup_day_of_week <= 6))),
    CONSTRAINT backup_policies_compression_format_check CHECK (((compression_format)::text = ANY (ARRAY[('gzip'::character varying)::text, ('bzip2'::character varying)::text, ('lz4'::character varying)::text, ('zstd'::character varying)::text]))),
    CONSTRAINT backup_policies_compression_level_check CHECK (((compression_level >= 1) AND (compression_level <= 9))),
    CONSTRAINT backup_policies_keep_last_n_check CHECK ((keep_last_n >= 1)),
    CONSTRAINT backup_policies_max_backup_count_check CHECK ((max_backup_count >= 1)),
    CONSTRAINT backup_policies_max_storage_gb_check CHECK ((max_storage_gb >= 1)),
    CONSTRAINT backup_policies_min_backup_count_check CHECK ((min_backup_count >= 1)),
    CONSTRAINT backup_policies_retention_days_check CHECK (((retention_days >= 1) AND (retention_days <= 3650))),
    CONSTRAINT backup_policies_storage_backend_check CHECK (((storage_backend)::text = ANY (ARRAY[('local'::character varying)::text, ('ftp'::character varying)::text, ('sftp'::character varying)::text, ('nfs'::character varying)::text])))
);


--
-- Name: backup_restore_file; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.backup_restore_file (
    id bigint NOT NULL,
    serial_number character varying(64) NOT NULL,
    file_name text NOT NULL,
    object_path text NOT NULL,
    md5 character varying(64),
    file_size bigint DEFAULT 0 NOT NULL,
    operator_code character varying(8),
    update_time timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    task_id uuid,
    is_deleted boolean DEFAULT false NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: backup_restore_file_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.backup_restore_file_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: backup_restore_file_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.backup_restore_file_id_seq OWNED BY public.backup_restore_file.id;


--
-- Name: backup_schedules; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.backup_schedules (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(200) NOT NULL,
    cron_expr character varying(100) NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    task_type character varying(20) DEFAULT 'full'::character varying NOT NULL,
    target_type character varying(20),
    target_ids jsonb DEFAULT '[]'::jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    is_enable smallint GENERATED ALWAYS AS (
CASE
    WHEN enabled THEN 1
    ELSE 0
END) STORED
);


--
-- Name: backup_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.backup_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_type character varying(20) DEFAULT 'full'::character varying NOT NULL,
    target_type character varying(20) DEFAULT 'device'::character varying NOT NULL,
    target_ids jsonb DEFAULT '[]'::jsonb NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    progress integer DEFAULT 0 NOT NULL,
    file_path text,
    error_message text,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    task_seq bigint NOT NULL,
    task_name character varying(200),
    task_result smallint,
    operator_code character varying(8),
    create_user character varying(64),
    CONSTRAINT backup_tasks_task_result_check CHECK (((task_result IS NULL) OR (task_result = ANY (ARRAY[1, 2]))))
);


--
-- Name: backup_tasks_task_seq_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.backup_tasks_task_seq_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: backup_tasks_task_seq_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.backup_tasks_task_seq_seq OWNED BY public.backup_tasks.task_seq;


--
-- Name: config_backup_sub_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.config_backup_sub_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_id uuid NOT NULL,
    device_id uuid NOT NULL,
    firmware_id uuid,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    error_message text,
    retry_count integer DEFAULT 0 NOT NULL,
    max_retries integer DEFAULT 3 NOT NULL,
    device_sn character varying(64),
    ori_version character varying(64),
    dest_version character varying(64),
    command_key character varying(256),
    failure_reason text,
    pre_suspend_status character varying(20),
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_upgrade_sub_tasks_status CHECK (((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('downloading'::character varying)::text, ('uploading'::character varying)::text, ('rebooting'::character varying)::text, ('verifying'::character varying)::text, ('completed'::character varying)::text, ('failed'::character varying)::text, ('suspended'::character varying)::text, ('terminated'::character varying)::text])))
);


--
-- Name: config_backup_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.config_backup_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    firmware_id uuid,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    error_message text,
    retry_count integer DEFAULT 0 NOT NULL,
    max_retries integer DEFAULT 3 NOT NULL,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    task_name character varying(256),
    task_type smallint DEFAULT 1 NOT NULL,
    file_name character varying(256),
    file_md5 character varying(64),
    result character varying(20),
    product_class character varying(64),
    is_keep_config boolean DEFAULT true,
    create_status character varying(16) DEFAULT 'active'::character varying NOT NULL,
    create_user character varying(64) DEFAULT 'system'::character varying NOT NULL,
    total_count integer DEFAULT 0 NOT NULL,
    success_count integer DEFAULT 0 NOT NULL,
    fail_count integer DEFAULT 0 NOT NULL,
    max_concurrent integer DEFAULT 5,
    ended_at timestamp with time zone,
    strategy character varying(16) DEFAULT 'full'::character varying NOT NULL,
    canary_stages jsonb,
    current_stage integer DEFAULT 0 NOT NULL,
    stage_status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    stage_history jsonb DEFAULT '[]'::jsonb NOT NULL,
    auto_advance boolean DEFAULT false NOT NULL,
    auto_advance_minutes integer DEFAULT 0 NOT NULL,
    rollback_reason text,
    rollback_source character varying(32) DEFAULT 'manual'::character varying NOT NULL,
    rollback_target_firmware_id uuid,
    rollback_on_failure boolean DEFAULT false NOT NULL,
    download_file_type text DEFAULT ''::text NOT NULL,
    scheduled_at timestamp with time zone,
    CONSTRAINT chk_upgrade_tasks_create_status CHECK (((create_status)::text = ANY (ARRAY[('active'::character varying)::text, ('suspend'::character varying)::text, ('timing'::character varying)::text]))),
    CONSTRAINT chk_upgrade_tasks_result CHECK (((result IS NULL) OR ((result)::text = ANY (ARRAY[('success'::character varying)::text, ('partial'::character varying)::text, ('failed'::character varying)::text, ('terminated'::character varying)::text])))),
    CONSTRAINT chk_upgrade_tasks_status CHECK (((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('in_progress'::character varying)::text, ('suspended'::character varying)::text, ('ended'::character varying)::text]))),
    CONSTRAINT chk_upgrade_tasks_task_type CHECK ((task_type = ANY (ARRAY[1, 2, 4, 6, 8, 10]))),
    CONSTRAINT upgrade_tasks_auto_advance_minutes_check CHECK (((auto_advance_minutes >= 0) AND (auto_advance_minutes <= 1440))),
    CONSTRAINT upgrade_tasks_current_stage_check CHECK (((current_stage >= 0) AND (current_stage <= 100))),
    CONSTRAINT upgrade_tasks_rollback_source_check CHECK (((rollback_source)::text = ANY (ARRAY[('manual'::character varying)::text, ('canary_failure'::character varying)::text, ('compatibility'::character varying)::text, ('scheduled'::character varying)::text]))),
    CONSTRAINT upgrade_tasks_stage_status_check CHECK (((stage_status)::text = ANY (ARRAY[('pending'::character varying)::text, ('running'::character varying)::text, ('paused'::character varying)::text, ('aborted'::character varying)::text, ('completed'::character varying)::text]))),
    CONSTRAINT upgrade_tasks_strategy_check CHECK (((strategy)::text = ANY (ARRAY[('full'::character varying)::text, ('canary'::character varying)::text])))
);


--
-- Name: config_baselines; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.config_baselines (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    baseline_name character varying(200) NOT NULL,
    description text,
    device_type character varying(50),
    version character varying(50),
    params jsonb DEFAULT '[]'::jsonb NOT NULL,
    creator character varying(100),
    status character varying(20) DEFAULT 'draft'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: config_neighbors; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.config_neighbors (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_cell_id character varying(64) NOT NULL,
    source_cell_name character varying(200),
    target_cell_id character varying(64) NOT NULL,
    target_cell_name character varying(200),
    neighbor_type character varying(20) DEFAULT 'intra-freq'::character varying NOT NULL,
    params jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: config_restore_sub_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.config_restore_sub_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_id uuid NOT NULL,
    device_id uuid NOT NULL,
    firmware_id uuid,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    error_message text,
    retry_count integer DEFAULT 0 NOT NULL,
    max_retries integer DEFAULT 3 NOT NULL,
    device_sn character varying(64),
    ori_version character varying(64),
    dest_version character varying(64),
    command_key character varying(256),
    failure_reason text,
    pre_suspend_status character varying(20),
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_upgrade_sub_tasks_status CHECK (((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('downloading'::character varying)::text, ('uploading'::character varying)::text, ('rebooting'::character varying)::text, ('verifying'::character varying)::text, ('completed'::character varying)::text, ('failed'::character varying)::text, ('suspended'::character varying)::text, ('terminated'::character varying)::text])))
);


--
-- Name: config_restore_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.config_restore_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    firmware_id uuid,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    error_message text,
    retry_count integer DEFAULT 0 NOT NULL,
    max_retries integer DEFAULT 3 NOT NULL,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    task_name character varying(256),
    task_type smallint DEFAULT 1 NOT NULL,
    file_name character varying(256),
    file_md5 character varying(64),
    result character varying(20),
    product_class character varying(64),
    is_keep_config boolean DEFAULT true,
    create_status character varying(16) DEFAULT 'active'::character varying NOT NULL,
    create_user character varying(64) DEFAULT 'system'::character varying NOT NULL,
    total_count integer DEFAULT 0 NOT NULL,
    success_count integer DEFAULT 0 NOT NULL,
    fail_count integer DEFAULT 0 NOT NULL,
    max_concurrent integer DEFAULT 5,
    ended_at timestamp with time zone,
    strategy character varying(16) DEFAULT 'full'::character varying NOT NULL,
    canary_stages jsonb,
    current_stage integer DEFAULT 0 NOT NULL,
    stage_status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    stage_history jsonb DEFAULT '[]'::jsonb NOT NULL,
    auto_advance boolean DEFAULT false NOT NULL,
    auto_advance_minutes integer DEFAULT 0 NOT NULL,
    rollback_reason text,
    rollback_source character varying(32) DEFAULT 'manual'::character varying NOT NULL,
    rollback_target_firmware_id uuid,
    rollback_on_failure boolean DEFAULT false NOT NULL,
    download_file_type text DEFAULT ''::text NOT NULL,
    scheduled_at timestamp with time zone,
    CONSTRAINT chk_upgrade_tasks_create_status CHECK (((create_status)::text = ANY (ARRAY[('active'::character varying)::text, ('suspend'::character varying)::text, ('timing'::character varying)::text]))),
    CONSTRAINT chk_upgrade_tasks_result CHECK (((result IS NULL) OR ((result)::text = ANY (ARRAY[('success'::character varying)::text, ('partial'::character varying)::text, ('failed'::character varying)::text, ('terminated'::character varying)::text])))),
    CONSTRAINT chk_upgrade_tasks_status CHECK (((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('in_progress'::character varying)::text, ('suspended'::character varying)::text, ('ended'::character varying)::text]))),
    CONSTRAINT chk_upgrade_tasks_task_type CHECK ((task_type = ANY (ARRAY[1, 2, 4, 6, 8, 10]))),
    CONSTRAINT upgrade_tasks_auto_advance_minutes_check CHECK (((auto_advance_minutes >= 0) AND (auto_advance_minutes <= 1440))),
    CONSTRAINT upgrade_tasks_current_stage_check CHECK (((current_stage >= 0) AND (current_stage <= 100))),
    CONSTRAINT upgrade_tasks_rollback_source_check CHECK (((rollback_source)::text = ANY (ARRAY[('manual'::character varying)::text, ('canary_failure'::character varying)::text, ('compatibility'::character varying)::text, ('scheduled'::character varying)::text]))),
    CONSTRAINT upgrade_tasks_stage_status_check CHECK (((stage_status)::text = ANY (ARRAY[('pending'::character varying)::text, ('running'::character varying)::text, ('paused'::character varying)::text, ('aborted'::character varying)::text, ('completed'::character varying)::text]))),
    CONSTRAINT upgrade_tasks_strategy_check CHECK (((strategy)::text = ANY (ARRAY[('full'::character varying)::text, ('canary'::character varying)::text])))
);


--
-- Name: config_snapshots; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.config_snapshots (
    serial_number character varying(64) NOT NULL,
    enb_name character varying(128),
    product_type character varying(64),
    file_name text NOT NULL,
    file_ext character varying(8) NOT NULL,
    object_bucket character varying(64) NOT NULL,
    object_path text NOT NULL,
    md5 character varying(64),
    file_size bigint DEFAULT 0 NOT NULL,
    source character varying(16) NOT NULL,
    source_task_id uuid,
    update_by character varying(64),
    update_time timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    source_version character varying(64),
    CONSTRAINT config_snapshots_file_ext_check CHECK (((file_ext)::text = ANY (ARRAY[('xml'::character varying)::text, ('nv'::character varying)::text]))),
    CONSTRAINT config_snapshots_source_check CHECK (((source)::text = ANY (ARRAY[('backup'::character varying)::text, ('manual_upload'::character varying)::text])))
);


--
-- Name: config_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.config_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_name character varying(200) NOT NULL,
    task_type character varying(30) NOT NULL,
    device_sns jsonb DEFAULT '[]'::jsonb NOT NULL,
    template_id uuid,
    baseline_id uuid,
    params jsonb,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    progress integer DEFAULT 0 NOT NULL,
    success_count integer DEFAULT 0 NOT NULL,
    fail_count integer DEFAULT 0 NOT NULL,
    total_count integer DEFAULT 0 NOT NULL,
    creator character varying(100),
    message text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: config_templates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.config_templates (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(128) NOT NULL,
    carrier character varying(4) NOT NULL,
    technology character varying(3) NOT NULL,
    product_class character varying(64),
    template_type character varying(32) NOT NULL,
    parameters jsonb DEFAULT '{}'::jsonb NOT NULL,
    priority integer DEFAULT 0 NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    auto_dispatch boolean DEFAULT false NOT NULL,
    CONSTRAINT config_templates_template_type_check CHECK (((template_type)::text = ANY (ARRAY[('provisioning'::character varying)::text, ('batch_config'::character varying)::text, ('firmware_upgrade'::character varying)::text])))
);


--
-- Name: COLUMN config_templates.auto_dispatch; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.config_templates.auto_dispatch IS 'T-0120-b opt-in 自动下发：device.registered 事件触发时若模板匹配且此列 true → 强制走 Path A，默认 FALSE 保留手动 dispatch 语义';


--
-- Name: dashboard_kpi_layouts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.dashboard_kpi_layouts (
    tech text NOT NULL,
    layout jsonb DEFAULT '{"panels": []}'::jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by uuid,
    CONSTRAINT dashboard_kpi_layouts_tech_check CHECK ((tech = ANY (ARRAY['lte'::text, 'nr'::text, 'gsm'::text])))
);


--
-- Name: TABLE dashboard_kpi_layouts; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.dashboard_kpi_layouts IS 'issue #213：Dashboard 首页 KPI 折线图区全局布局，按制式各一行（lte/nr/gsm），全局单套所有用户共享。';


--
-- Name: COLUMN dashboard_kpi_layouts.tech; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.dashboard_kpi_layouts.tech IS '制式主键：lte / nr / gsm。';


--
-- Name: COLUMN dashboard_kpi_layouts.layout; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.dashboard_kpi_layouts.layout IS '布局体 JSONB：panels 数组，每图含 title / metrics(symbolic key 列表) / x,y(网格位置) / w,h(网格大小) / chartType(预留，恒 line)。';


--
-- Name: COLUMN dashboard_kpi_layouts.updated_by; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.dashboard_kpi_layouts.updated_by IS '最近一次保存的管理员用户 ID（nullable：seed 灌入的初始行无来源用户）。';


--
-- Name: dashboard_widgets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.dashboard_widgets (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    layout jsonb DEFAULT '[]'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: dead_letters; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.dead_letters (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_module character varying(64) NOT NULL,
    source_subject character varying(128) NOT NULL,
    payload bytea NOT NULL,
    error text NOT NULL,
    retry_count integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    last_attempt_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT dead_letters_retry_count_check CHECK (((retry_count >= 0) AND (retry_count <= 100)))
);


--
-- Name: device_active_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_active_tasks (
    device_id uuid NOT NULL,
    sub_task_id uuid NOT NULL,
    business_type character varying(32) NOT NULL,
    sub_task_table character varying(64) NOT NULL,
    acquired_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT device_active_tasks_business_type_check CHECK (((business_type)::text = ANY (ARRAY[('upgrade'::character varying)::text, ('rollback'::character varying)::text, ('config_backup'::character varying)::text, ('config_restore'::character varying)::text, ('runtime_log_collect'::character varying)::text, ('fault_log_collect'::character varying)::text])))
);


--
-- Name: device_group_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_group_members (
    group_id uuid NOT NULL,
    device_id uuid NOT NULL,
    added_at timestamp with time zone DEFAULT now() NOT NULL,
    source_type character varying(16) DEFAULT 'manual'::character varying NOT NULL,
    CONSTRAINT chk_device_group_members_source_type CHECK (((source_type)::text = ANY (ARRAY[('manual'::character varying)::text, ('rule'::character varying)::text])))
);


--
-- Name: device_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_groups (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(128) NOT NULL,
    parent_id uuid,
    carrier character varying(16),
    description text,
    sort_order integer DEFAULT 0 NOT NULL,
    status character varying(16) DEFAULT 'active'::character varying NOT NULL,
    remark text,
    is_default boolean DEFAULT false NOT NULL,
    level smallint DEFAULT 1 NOT NULL,
    created_by character varying(64),
    updated_by character varying(64),
    matching_mode character varying(16),
    name_rule_list jsonb,
    lac_list integer[],
    tac_list integer[],
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    serial_number_list text[],
    name_i18n jsonb DEFAULT '{}'::jsonb NOT NULL,
    description_i18n jsonb DEFAULT '{}'::jsonb NOT NULL,
    remark_i18n jsonb DEFAULT '{}'::jsonb NOT NULL,
    source_group_id uuid,
    CONSTRAINT chk_dg_level CHECK ((level = ANY (ARRAY[1, 2]))),
    CONSTRAINT chk_dg_level_parent CHECK ((((level = 1) AND (parent_id IS NULL)) OR ((level = 2) AND (parent_id IS NOT NULL)))),
    CONSTRAINT chk_dg_matching_mode CHECK (((matching_mode IS NULL) OR ((matching_mode)::text = ANY (ARRAY[('deviceName'::character varying)::text, ('lac'::character varying)::text, ('tac'::character varying)::text, ('serialNumber'::character varying)::text])))),
    CONSTRAINT device_groups_rule_source_not_self CHECK (((source_group_id IS NULL) OR (source_group_id <> id)))
);


--
-- Name: COLUMN device_groups.source_group_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_groups.source_group_id IS 'Source L2 group for automatic matching; NULL disables legacy rules until configured.';


--
-- Name: device_info; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_info (
    device_id uuid NOT NULL,
    device_name character varying(128),
    address character varying(256),
    remark text,
    project_status character varying(20),
    height numeric(10,2),
    eci character varying(64),
    pci character varying(64),
    cell_id character varying(64),
    freq_point character varying(32),
    bandwidth numeric(8,2),
    transmit_power numeric(8,2),
    plmn character varying(40),
    rf_status character varying(64),
    cell_status character varying(20),
    mme_status character varying(20),
    sync_status character varying(32),
    kpi_status character varying(20),
    num_of_cells integer DEFAULT 1,
    gps_status character varying(20),
    alarm_severity character varying(20),
    license_status character varying(20),
    mac character varying(64),
    hardware_version character varying(64),
    first_online_time timestamp with time zone,
    last_online_time timestamp with time zone,
    last_offline_time timestamp with time zone,
    run_time bigint DEFAULT 0,
    creator character varying(64),
    updater character varying(64),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    tac character varying(16),
    band character varying(16),
    ul_earfcn character varying(32),
    subframe_assignment character varying(8),
    special_subframe character varying(8),
    root_index character varying(16),
    gps_satellites integer,
    gps_height numeric(10,2),
    lock_status character varying(16),
    enb_id character varying(32),
    network_model character varying(16),
    lac character varying(16),
    cumulative_online_duration bigint DEFAULT 0 NOT NULL,
    op_state character varying(8) DEFAULT '0'::character varying,
    admin_state character varying(16),
    ipsec_addr character varying(64),
    bsc_select character varying(8),
    oml_remote_ip character varying(45),
    oml_remote_ip_bak character varying(45),
    ipa_unit_id character varying(32),
    ue_count integer DEFAULT 0,
    active_alarm_count integer DEFAULT 0 NOT NULL,
    name_sync_pending boolean DEFAULT false NOT NULL,
    lmt_device_name character varying(255),
    highest_alarm_severity smallint,
    highest_severity_alarm_count smallint DEFAULT 0
)
WITH (autovacuum_vacuum_scale_factor='0.02', autovacuum_vacuum_threshold='200', autovacuum_analyze_scale_factor='0.02', autovacuum_analyze_threshold='200');


--
-- Name: TABLE device_info; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.device_info IS '设备运维扩展信息表（与 devices 1:1）。承载 device_name / 位置 / 小区配置 / 运维状态聚合 / 生命周期时间戳 / 累计时长等"快速查询"列;扩展字段由 InfoSyncer 从 device_parameters 投影。';


--
-- Name: COLUMN device_info.device_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.device_id IS '外键 → devices.id（1:1 关系,与 devices 主键同分布）。';


--
-- Name: COLUMN device_info.device_name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.device_name IS '设备显示名（站点名 / 别名）;运营商规划或人工设置,区别于 serial_number。';


--
-- Name: COLUMN device_info.address; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.address IS '设备安装地址（文本）,人工录入,与 latitude / longitude 互补。';


--
-- Name: COLUMN device_info.remark; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.remark IS '运维备注文本,人工录入。';


--
-- Name: COLUMN device_info.project_status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.project_status IS '工程阶段状态:planned / installed / commissioned / decommissioned,与 devices.lifecycle_state 业务进度互补。';


--
-- Name: COLUMN device_info.height; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.height IS '天线安装挂高（米）。';


--
-- Name: COLUMN device_info.eci; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.eci IS 'E-UTRAN Cell Identifier (28 bit),LTE 小区全局标识 = eNodeB ID (20 bit) << 8 | Cell ID (8 bit)。';


--
-- Name: COLUMN device_info.pci; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.pci IS '物理小区标识 (Physical Cell ID,0-503),对应 TR-181 Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PhyCellID。';


--
-- Name: COLUMN device_info.cell_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.cell_id IS '小区 ID (8 bit),ECI 的低 8 位。';


--
-- Name: COLUMN device_info.freq_point; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.freq_point IS '频点号（LTE EARFCN / NR NRARFCN）。';


--
-- Name: COLUMN device_info.bandwidth; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.bandwidth IS '工作带宽（MHz,小数支持 1.4 / 3 / 5 / 10 / 15 / 20 等）。';


--
-- Name: COLUMN device_info.transmit_power; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.transmit_power IS '发射功率（dBm），口径=参考信号功率（与 LMT 一致），来源 TR-181 FAPService.{i}.CellConfig.LTE.RAN.RF.ReferenceSignalPower（经 cmcc/ctcc carrier adapter 映射）。注：非硬件最大能力上限 MaxTxPower。';


--
-- Name: COLUMN device_info.plmn; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.plmn IS '公共陆地移动网络号（MCC+MNC,如 46000 表示 CMCC LTE）。';


--
-- Name: COLUMN device_info.rf_status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.rf_status IS 'RF 状态聚合:on/off,多小区时可逗号分隔;支持最多 9 个小区状态;由 InfoSyncer.CalcRFStatus 从参数派生。';


--
-- Name: COLUMN device_info.cell_status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.cell_status IS '小区状态聚合:active / inactive / locked;由 InfoSyncer.CalcCellStatus 派生。';


--
-- Name: COLUMN device_info.mme_status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.mme_status IS 'MME (4G) / AMF (5G) 连接状态:connected / disconnected;由 InfoSyncer.CalcMMEStatus 派生。';


--
-- Name: COLUMN device_info.sync_status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.sync_status IS '时钟同步状态:synchronized / gps / 1588 / rem / not_synchronized;由 InfoSyncer.CalcSyncStatus 派生。';


--
-- Name: COLUMN device_info.kpi_status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.kpi_status IS 'KPI 采集状态:reporting / silent,反映 PM 文件上报是否正常。';


--
-- Name: COLUMN device_info.num_of_cells; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.num_of_cells IS '小区数（多小区设备）,由 InfoSyncer.CalcNumOfCells 计数派生。';


--
-- Name: COLUMN device_info.gps_status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.gps_status IS 'GPS 锁定状态:locked / searching / disabled;由 InfoSyncer.CalcGPSStatus 派生。';


--
-- Name: COLUMN device_info.alarm_severity; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.alarm_severity IS '当前最严重告警级别冗余字段:critical / major / minor / warning / none;由告警模块维护。';


--
-- Name: COLUMN device_info.license_status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.license_status IS '设备 License 状态:valid / expired / missing;由 InfoSyncer.CalcLicenseStatus 派生。';


--
-- Name: COLUMN device_info.mac; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.mac IS '设备 MAC 地址（HEX,无分隔符）,对应 TR-181 Device.Ethernet.Interface.{i}.MACAddress。';


--
-- Name: COLUMN device_info.hardware_version; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.hardware_version IS '硬件版本号,对应 TR-181 Device.DeviceInfo.HardwareVersion。';


--
-- Name: COLUMN device_info.first_online_time; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.first_online_time IS '设备首次成功 Inform 注册的时刻;一旦写入永不变更;由 InfoSyncer.RecordOnline 在 first_online_time IS NULL 时填充。';


--
-- Name: COLUMN device_info.last_online_time; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.last_online_time IS '本次上线时刻（offline→online 边沿）。由 DeviceService.UpdateFromInform 在检测到状态翻转时,或 InfoSyncer.RecordOnline 调用时写入。前端"当前在线时长" = is_online ? NOW - last_online_time : 0。';


--
-- Name: COLUMN device_info.last_offline_time; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.last_offline_time IS '本次离线时刻（online→offline 边沿）。由 DeviceStatusReconciler.markOffline 事务性写入,同时累加 cumulative_online_duration += NOW - last_online_time。';


--
-- Name: COLUMN device_info.run_time; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.run_time IS '设备自报运行时长（秒）,对应 TR-181 Device.DeviceInfo.UpTime;由 InfoSyncer.SyncFromParameters 在每次参数同步时刷新。注意:这是"设备从上次本地重启算起的时长",与 OMC 视角的累计在线时长（cumulative_online_duration）含义不同。';


--
-- Name: COLUMN device_info.creator; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.creator IS '记录创建者用户标识。';


--
-- Name: COLUMN device_info.updater; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.updater IS '记录最近更新者用户标识。';


--
-- Name: COLUMN device_info.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.created_at IS '记录入库时间（PG 自动填充）。';


--
-- Name: COLUMN device_info.updated_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.updated_at IS '记录最近更新时间（PG 触发器 trigger_device_info_updated_at 自动维护）。';


--
-- Name: COLUMN device_info.tac; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.tac IS 'LTE 跟踪区码（Tracking Area Code），由 ACS Inform 解析 Device.Services.FAPService.{i}.CellConfig.LTE.EPC.TAC 写入。可空 —— GSM-only 设备无此值。';


--
-- Name: COLUMN device_info.band; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.band IS 'LTE Band, TR-069 RAN.RF.FreqBandIndicator';


--
-- Name: COLUMN device_info.ul_earfcn; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.ul_earfcn IS 'Uplink EARFCN, TR-069 RAN.RF.EARFCNUL';


--
-- Name: COLUMN device_info.subframe_assignment; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.subframe_assignment IS 'TDD frame, RAN.PHY.TDDFrame.SubFrameAssignment';


--
-- Name: COLUMN device_info.special_subframe; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.special_subframe IS 'TDD special subframe pattern';


--
-- Name: COLUMN device_info.root_index; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.root_index IS 'PRACH ZeroCorrelationZoneConfig';


--
-- Name: COLUMN device_info.gps_satellites; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.gps_satellites IS 'GPS satellite count, FAP.GPS.NumberOfSatellites';


--
-- Name: COLUMN device_info.gps_height; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.gps_height IS 'GPS height (meters), FAP.GPS.altidute';


--
-- Name: COLUMN device_info.lock_status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.lock_status IS 'Cell AdminState (true=unlocked / false=locked)';


--
-- Name: COLUMN device_info.enb_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.enb_id IS 'eNodeB ID, derived from ECI >> 8 (LTE 28-bit ECI = 20-bit eNB + 8-bit Cell)';


--
-- Name: COLUMN device_info.network_model; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.network_model IS 'TDD or FDD, derived from frame structure';


--
-- Name: COLUMN device_info.lac; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.lac IS 'GSM 位置区码（Location Area Code），由 ACS Inform 解析 Device.DeviceInfo.BTS.CurrentLac 写入。可空 —— LTE-only 设备无此值。';


--
-- Name: COLUMN device_info.cumulative_online_duration; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.cumulative_online_duration IS '累计在线总时长（秒）。online→offline 转换时由 DeviceStatusReconciler 事务性累加 (NOW - last_online_time)。新字段从启用日开始累计,存量设备初值为 0;查询"总在线时长" 应用 = cumulative_online_duration + (is_online ? NOW - last_online_time : 0)。';


--
-- Name: COLUMN device_info.op_state; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.op_state IS '激活状态:1=激活/0=未激活;由 InfoSyncer.CalcOpState 派生(等价 cell_status: 任一 cell active → "1")。与 first_online_time 派生口径解耦。';


--
-- Name: COLUMN device_info.admin_state; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.admin_state IS 'NR FAPControl AdminState ("1"=Locked, "2"=Unlocked, "3"=ShuttingDown), source Device.Services.FAPService.1.FAPControl.NR.RAN.Common.AdminState. NULL for LTE devices (use lock_status instead).';


--
-- Name: COLUMN device_info.ipsec_addr; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.ipsec_addr IS 'IPSec serving unit 1 tunnel address, source Device.DeviceInfo.SERVING_UNIT1_IPSEC_Address. "0.0.0.0" means tunnel not established.';


--
-- Name: COLUMN device_info.bsc_select; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.bsc_select IS 'GSM BSC primary/backup role from DeviceGSM.BscSelect ("0"=Master, "1"=Backup). BTS-only; LTE/NR stay NULL.';


--
-- Name: COLUMN device_info.oml_remote_ip; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.oml_remote_ip IS 'Abis OML primary BSC IP from DeviceGSM.OmlRemoteIp. BTS-only.';


--
-- Name: COLUMN device_info.oml_remote_ip_bak; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.oml_remote_ip_bak IS 'Abis OML backup BSC IP from DeviceGSM.OmlRemoteIpBak. BTS-only.';


--
-- Name: COLUMN device_info.ipa_unit_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.ipa_unit_id IS 'IPA unit ID from DeviceGSM.IpaUnitId, for example "9227-2". BTS-only.';


--
-- Name: COLUMN device_info.name_sync_pending; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.name_sync_pending IS '设备名称同步待处理标记（true=需人工确认，前端显示小红点）。Path B 同步检测到 LMT 名称与网管不一致且 prompt=true 时置 true；用户确认或下次同步名称一致时自动清 false。';


--
-- Name: COLUMN device_info.lmt_device_name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_info.lmt_device_name IS '从 LMT 读取的设备名称（HNBName / gNBName）缓存。供前端在名称冲突时对比展示"LMT 名称"与"网管名称"。';


--
-- Name: device_licenses; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_licenses (
    serial_number character varying(64) NOT NULL,
    enb_name character varying(128),
    product_type character varying(64),
    file_name text NOT NULL,
    file_ext character varying(8) NOT NULL,
    object_bucket character varying(64) NOT NULL,
    object_path text NOT NULL,
    md5 character varying(64),
    file_size bigint DEFAULT 0 NOT NULL,
    source character varying(16) DEFAULT 'manual_upload'::character varying NOT NULL,
    description text,
    auto_dispatch_pending boolean DEFAULT true NOT NULL,
    update_by character varying(64),
    update_time timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT device_licenses_file_ext_check CHECK (((file_ext)::text = 'lic'::text)),
    CONSTRAINT device_licenses_source_check CHECK (((source)::text = 'manual_upload'::text))
);


--
-- Name: device_location_observations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_location_observations (
    device_id uuid NOT NULL,
    latitude double precision NOT NULL,
    longitude double precision NOT NULL,
    gps_height double precision,
    observed_at timestamp with time zone NOT NULL,
    version bigint DEFAULT 1 NOT NULL,
    source_path text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT device_location_observations_latitude_check CHECK (((latitude >= ('-90'::integer)::double precision) AND (latitude <= (90)::double precision))),
    CONSTRAINT device_location_observations_longitude_check CHECK (((longitude >= ('-180'::integer)::double precision) AND (longitude <= (180)::double precision))),
    CONSTRAINT device_location_observations_version_check CHECK ((version > 0))
);


--
-- Name: TABLE device_location_observations; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.device_location_observations IS '设备最新有效 GPS 观测值；不等同于网管已接受坐标';


--
-- Name: device_parameters; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
)
PARTITION BY HASH (device_id);


--
-- Name: device_parameters_p00; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p00 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p01; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p01 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p02; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p02 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p03; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p03 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p04; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p04 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p05; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p05 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p06; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p06 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p07; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p07 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p08; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p08 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p09; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p09 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p10; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p10 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p11; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p11 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p12; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p12 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p13; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p13 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p14; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p14 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p15; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p15 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p16; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p16 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p17; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p17 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p18; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p18 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p19; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p19 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p20; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p20 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p21; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p21 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p22; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p22 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p23; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p23 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p24; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p24 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p25; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p25 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p26; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p26 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p27; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p27 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p28; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p28 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p29; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p29 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p30; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p30 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_parameters_p31; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_parameters_p31 (
    device_id uuid NOT NULL,
    parameter_path character varying(512) NOT NULL,
    parameter_value text,
    parameter_type character varying(20),
    writable boolean DEFAULT false,
    last_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fap_instance smallint DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL
);


--
-- Name: device_antenna_sector_plans; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_antenna_sector_plans (
    device_id uuid NOT NULL,
    sector_no smallint NOT NULL,
    azimuth_deg numeric(6,2),
    antenna_height_m numeric(8,2),
    mechanical_downtilt_deg numeric(6,2),
    horizontal_beamwidth_deg numeric(6,2),
    vertical_beamwidth_deg numeric(6,2),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT device_antenna_sector_plans_pkey PRIMARY KEY (device_id, sector_no)
);


COMMENT ON TABLE public.device_antenna_sector_plans IS 'OMC 天线扇区规划参数；保存后用于 GIS 覆盖示意，不向设备下发。';


--
-- Name: device_registrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_registrations (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    serial_number character varying(64) NOT NULL,
    group_id uuid NOT NULL,
    device_id uuid,
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    site_name character varying(128),
    device_name character varying(128),
    longitude double precision,
    latitude double precision,
    height numeric(10,2),
    azimuth smallint,
    tilt_angle smallint,
    beam_width smallint,
    remark text,
    created_by character varying(64),
    import_batch_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    carrier character varying(4) DEFAULT 'cmcc'::character varying NOT NULL
);


--
-- Name: device_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    method character varying(64) NOT NULL,
    params jsonb,
    priority integer DEFAULT 10,
    command_key character varying(128),
    cwmp_id character varying(256),
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    retry_count integer DEFAULT 0,
    max_retries integer DEFAULT 3,
    retry_interval_seconds integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_at timestamp with time zone,
    completed_at timestamp with time zone,
    expires_at timestamp with time zone,
    next_attempt_at timestamp with time zone,
    result jsonb,
    error_code integer,
    error_message text,
    source character varying(32) DEFAULT 'api'::character varying,
    creator_id character varying(64),
    description text,
    source_id uuid,
    command_index integer DEFAULT 0,
    device_index integer DEFAULT 0,
    has_path_translation_miss boolean DEFAULT false NOT NULL,
    path_translation_miss_count integer DEFAULT 0 NOT NULL,
    path_translation_source character varying(32)
)
PARTITION BY HASH (device_sn);


--
-- Name: COLUMN device_tasks.retry_interval_seconds; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_tasks.retry_interval_seconds IS '自动重试间隔秒数；MML failed_retry_interval 下传后用于控制失败重试间隔。0 表示立即可重试。';


--
-- Name: COLUMN device_tasks.next_attempt_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_tasks.next_attempt_at IS '下一次允许出队时间；用于 MML 失败重试延迟，未到时间的 pending 任务不会被 Redis 队列弹出。';


--
-- Name: COLUMN device_tasks.has_path_translation_miss; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_tasks.has_path_translation_miss IS 'MML fanout 翻译标记：true=本任务中至少一个 standardPath 未找到 device 对应的 privatePath，已 fallback 用 standardPath 下发；前端任务详情页据此显示警告。';


--
-- Name: COLUMN device_tasks.path_translation_miss_count; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_tasks.path_translation_miss_count IS 'MML fanout 翻译未命中的 path 数量；与 task.params.names 长度对照可定位具体 standardPath 直透条目。0 = 全部命中或本任务非 MML 来源（默认值）。';


--
-- Name: COLUMN device_tasks.path_translation_source; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.device_tasks.path_translation_source IS 'T-0168: per-device 翻译来源（同 mml_tasks.path_translation_source 枚举）。首版由 fanout 从 task 直接复制（R-8.4 保证 task 内 product_class 一致）；D27 弹性保留：未来若引入 per-device swVersion 差异化翻译，本列由 per-device translate 重写。';


--
-- Name: device_tasks_p00; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_tasks_p00 (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    method character varying(64) NOT NULL,
    params jsonb,
    priority integer DEFAULT 10,
    command_key character varying(128),
    cwmp_id character varying(256),
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    retry_count integer DEFAULT 0,
    max_retries integer DEFAULT 3,
    retry_interval_seconds integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_at timestamp with time zone,
    completed_at timestamp with time zone,
    expires_at timestamp with time zone,
    next_attempt_at timestamp with time zone,
    result jsonb,
    error_code integer,
    error_message text,
    source character varying(32) DEFAULT 'api'::character varying,
    creator_id character varying(64),
    description text,
    source_id uuid,
    command_index integer DEFAULT 0,
    device_index integer DEFAULT 0,
    has_path_translation_miss boolean DEFAULT false NOT NULL,
    path_translation_miss_count integer DEFAULT 0 NOT NULL,
    path_translation_source character varying(32)
);


--
-- Name: device_tasks_p01; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_tasks_p01 (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    method character varying(64) NOT NULL,
    params jsonb,
    priority integer DEFAULT 10,
    command_key character varying(128),
    cwmp_id character varying(256),
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    retry_count integer DEFAULT 0,
    max_retries integer DEFAULT 3,
    retry_interval_seconds integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_at timestamp with time zone,
    completed_at timestamp with time zone,
    expires_at timestamp with time zone,
    next_attempt_at timestamp with time zone,
    result jsonb,
    error_code integer,
    error_message text,
    source character varying(32) DEFAULT 'api'::character varying,
    creator_id character varying(64),
    description text,
    source_id uuid,
    command_index integer DEFAULT 0,
    device_index integer DEFAULT 0,
    has_path_translation_miss boolean DEFAULT false NOT NULL,
    path_translation_miss_count integer DEFAULT 0 NOT NULL,
    path_translation_source character varying(32)
);


--
-- Name: device_tasks_p02; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_tasks_p02 (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    method character varying(64) NOT NULL,
    params jsonb,
    priority integer DEFAULT 10,
    command_key character varying(128),
    cwmp_id character varying(256),
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    retry_count integer DEFAULT 0,
    max_retries integer DEFAULT 3,
    retry_interval_seconds integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_at timestamp with time zone,
    completed_at timestamp with time zone,
    expires_at timestamp with time zone,
    next_attempt_at timestamp with time zone,
    result jsonb,
    error_code integer,
    error_message text,
    source character varying(32) DEFAULT 'api'::character varying,
    creator_id character varying(64),
    description text,
    source_id uuid,
    command_index integer DEFAULT 0,
    device_index integer DEFAULT 0,
    has_path_translation_miss boolean DEFAULT false NOT NULL,
    path_translation_miss_count integer DEFAULT 0 NOT NULL,
    path_translation_source character varying(32)
);


--
-- Name: device_tasks_p03; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_tasks_p03 (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    method character varying(64) NOT NULL,
    params jsonb,
    priority integer DEFAULT 10,
    command_key character varying(128),
    cwmp_id character varying(256),
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    retry_count integer DEFAULT 0,
    max_retries integer DEFAULT 3,
    retry_interval_seconds integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_at timestamp with time zone,
    completed_at timestamp with time zone,
    expires_at timestamp with time zone,
    next_attempt_at timestamp with time zone,
    result jsonb,
    error_code integer,
    error_message text,
    source character varying(32) DEFAULT 'api'::character varying,
    creator_id character varying(64),
    description text,
    source_id uuid,
    command_index integer DEFAULT 0,
    device_index integer DEFAULT 0,
    has_path_translation_miss boolean DEFAULT false NOT NULL,
    path_translation_miss_count integer DEFAULT 0 NOT NULL,
    path_translation_source character varying(32)
);


--
-- Name: device_tasks_p04; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_tasks_p04 (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    method character varying(64) NOT NULL,
    params jsonb,
    priority integer DEFAULT 10,
    command_key character varying(128),
    cwmp_id character varying(256),
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    retry_count integer DEFAULT 0,
    max_retries integer DEFAULT 3,
    retry_interval_seconds integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_at timestamp with time zone,
    completed_at timestamp with time zone,
    expires_at timestamp with time zone,
    next_attempt_at timestamp with time zone,
    result jsonb,
    error_code integer,
    error_message text,
    source character varying(32) DEFAULT 'api'::character varying,
    creator_id character varying(64),
    description text,
    source_id uuid,
    command_index integer DEFAULT 0,
    device_index integer DEFAULT 0,
    has_path_translation_miss boolean DEFAULT false NOT NULL,
    path_translation_miss_count integer DEFAULT 0 NOT NULL,
    path_translation_source character varying(32)
);


--
-- Name: device_tasks_p05; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_tasks_p05 (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    method character varying(64) NOT NULL,
    params jsonb,
    priority integer DEFAULT 10,
    command_key character varying(128),
    cwmp_id character varying(256),
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    retry_count integer DEFAULT 0,
    max_retries integer DEFAULT 3,
    retry_interval_seconds integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_at timestamp with time zone,
    completed_at timestamp with time zone,
    expires_at timestamp with time zone,
    next_attempt_at timestamp with time zone,
    result jsonb,
    error_code integer,
    error_message text,
    source character varying(32) DEFAULT 'api'::character varying,
    creator_id character varying(64),
    description text,
    source_id uuid,
    command_index integer DEFAULT 0,
    device_index integer DEFAULT 0,
    has_path_translation_miss boolean DEFAULT false NOT NULL,
    path_translation_miss_count integer DEFAULT 0 NOT NULL,
    path_translation_source character varying(32)
);


--
-- Name: device_tasks_p06; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_tasks_p06 (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    method character varying(64) NOT NULL,
    params jsonb,
    priority integer DEFAULT 10,
    command_key character varying(128),
    cwmp_id character varying(256),
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    retry_count integer DEFAULT 0,
    max_retries integer DEFAULT 3,
    retry_interval_seconds integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_at timestamp with time zone,
    completed_at timestamp with time zone,
    expires_at timestamp with time zone,
    next_attempt_at timestamp with time zone,
    result jsonb,
    error_code integer,
    error_message text,
    source character varying(32) DEFAULT 'api'::character varying,
    creator_id character varying(64),
    description text,
    source_id uuid,
    command_index integer DEFAULT 0,
    device_index integer DEFAULT 0,
    has_path_translation_miss boolean DEFAULT false NOT NULL,
    path_translation_miss_count integer DEFAULT 0 NOT NULL,
    path_translation_source character varying(32)
);


--
-- Name: device_tasks_p07; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_tasks_p07 (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    method character varying(64) NOT NULL,
    params jsonb,
    priority integer DEFAULT 10,
    command_key character varying(128),
    cwmp_id character varying(256),
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    retry_count integer DEFAULT 0,
    max_retries integer DEFAULT 3,
    retry_interval_seconds integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_at timestamp with time zone,
    completed_at timestamp with time zone,
    expires_at timestamp with time zone,
    next_attempt_at timestamp with time zone,
    result jsonb,
    error_code integer,
    error_message text,
    source character varying(32) DEFAULT 'api'::character varying,
    creator_id character varying(64),
    description text,
    source_id uuid,
    command_index integer DEFAULT 0,
    device_index integer DEFAULT 0,
    has_path_translation_miss boolean DEFAULT false NOT NULL,
    path_translation_miss_count integer DEFAULT 0 NOT NULL,
    path_translation_source character varying(32)
);


--
-- Name: device_tasks_p08; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_tasks_p08 (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    method character varying(64) NOT NULL,
    params jsonb,
    priority integer DEFAULT 10,
    command_key character varying(128),
    cwmp_id character varying(256),
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    retry_count integer DEFAULT 0,
    max_retries integer DEFAULT 3,
    retry_interval_seconds integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_at timestamp with time zone,
    completed_at timestamp with time zone,
    expires_at timestamp with time zone,
    next_attempt_at timestamp with time zone,
    result jsonb,
    error_code integer,
    error_message text,
    source character varying(32) DEFAULT 'api'::character varying,
    creator_id character varying(64),
    description text,
    source_id uuid,
    command_index integer DEFAULT 0,
    device_index integer DEFAULT 0,
    has_path_translation_miss boolean DEFAULT false NOT NULL,
    path_translation_miss_count integer DEFAULT 0 NOT NULL,
    path_translation_source character varying(32)
);


--
-- Name: device_tasks_p09; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_tasks_p09 (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    method character varying(64) NOT NULL,
    params jsonb,
    priority integer DEFAULT 10,
    command_key character varying(128),
    cwmp_id character varying(256),
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    retry_count integer DEFAULT 0,
    max_retries integer DEFAULT 3,
    retry_interval_seconds integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_at timestamp with time zone,
    completed_at timestamp with time zone,
    expires_at timestamp with time zone,
    next_attempt_at timestamp with time zone,
    result jsonb,
    error_code integer,
    error_message text,
    source character varying(32) DEFAULT 'api'::character varying,
    creator_id character varying(64),
    description text,
    source_id uuid,
    command_index integer DEFAULT 0,
    device_index integer DEFAULT 0,
    has_path_translation_miss boolean DEFAULT false NOT NULL,
    path_translation_miss_count integer DEFAULT 0 NOT NULL,
    path_translation_source character varying(32)
);


--
-- Name: device_tasks_p10; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_tasks_p10 (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    method character varying(64) NOT NULL,
    params jsonb,
    priority integer DEFAULT 10,
    command_key character varying(128),
    cwmp_id character varying(256),
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    retry_count integer DEFAULT 0,
    max_retries integer DEFAULT 3,
    retry_interval_seconds integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_at timestamp with time zone,
    completed_at timestamp with time zone,
    expires_at timestamp with time zone,
    next_attempt_at timestamp with time zone,
    result jsonb,
    error_code integer,
    error_message text,
    source character varying(32) DEFAULT 'api'::character varying,
    creator_id character varying(64),
    description text,
    source_id uuid,
    command_index integer DEFAULT 0,
    device_index integer DEFAULT 0,
    has_path_translation_miss boolean DEFAULT false NOT NULL,
    path_translation_miss_count integer DEFAULT 0 NOT NULL,
    path_translation_source character varying(32)
);


--
-- Name: device_tasks_p11; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_tasks_p11 (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    method character varying(64) NOT NULL,
    params jsonb,
    priority integer DEFAULT 10,
    command_key character varying(128),
    cwmp_id character varying(256),
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    retry_count integer DEFAULT 0,
    max_retries integer DEFAULT 3,
    retry_interval_seconds integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_at timestamp with time zone,
    completed_at timestamp with time zone,
    expires_at timestamp with time zone,
    next_attempt_at timestamp with time zone,
    result jsonb,
    error_code integer,
    error_message text,
    source character varying(32) DEFAULT 'api'::character varying,
    creator_id character varying(64),
    description text,
    source_id uuid,
    command_index integer DEFAULT 0,
    device_index integer DEFAULT 0,
    has_path_translation_miss boolean DEFAULT false NOT NULL,
    path_translation_miss_count integer DEFAULT 0 NOT NULL,
    path_translation_source character varying(32)
);


--
-- Name: device_tasks_p12; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_tasks_p12 (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    method character varying(64) NOT NULL,
    params jsonb,
    priority integer DEFAULT 10,
    command_key character varying(128),
    cwmp_id character varying(256),
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    retry_count integer DEFAULT 0,
    max_retries integer DEFAULT 3,
    retry_interval_seconds integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_at timestamp with time zone,
    completed_at timestamp with time zone,
    expires_at timestamp with time zone,
    next_attempt_at timestamp with time zone,
    result jsonb,
    error_code integer,
    error_message text,
    source character varying(32) DEFAULT 'api'::character varying,
    creator_id character varying(64),
    description text,
    source_id uuid,
    command_index integer DEFAULT 0,
    device_index integer DEFAULT 0,
    has_path_translation_miss boolean DEFAULT false NOT NULL,
    path_translation_miss_count integer DEFAULT 0 NOT NULL,
    path_translation_source character varying(32)
);


--
-- Name: device_tasks_p13; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_tasks_p13 (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    method character varying(64) NOT NULL,
    params jsonb,
    priority integer DEFAULT 10,
    command_key character varying(128),
    cwmp_id character varying(256),
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    retry_count integer DEFAULT 0,
    max_retries integer DEFAULT 3,
    retry_interval_seconds integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_at timestamp with time zone,
    completed_at timestamp with time zone,
    expires_at timestamp with time zone,
    next_attempt_at timestamp with time zone,
    result jsonb,
    error_code integer,
    error_message text,
    source character varying(32) DEFAULT 'api'::character varying,
    creator_id character varying(64),
    description text,
    source_id uuid,
    command_index integer DEFAULT 0,
    device_index integer DEFAULT 0,
    has_path_translation_miss boolean DEFAULT false NOT NULL,
    path_translation_miss_count integer DEFAULT 0 NOT NULL,
    path_translation_source character varying(32)
);


--
-- Name: device_tasks_p14; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_tasks_p14 (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    method character varying(64) NOT NULL,
    params jsonb,
    priority integer DEFAULT 10,
    command_key character varying(128),
    cwmp_id character varying(256),
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    retry_count integer DEFAULT 0,
    max_retries integer DEFAULT 3,
    retry_interval_seconds integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_at timestamp with time zone,
    completed_at timestamp with time zone,
    expires_at timestamp with time zone,
    next_attempt_at timestamp with time zone,
    result jsonb,
    error_code integer,
    error_message text,
    source character varying(32) DEFAULT 'api'::character varying,
    creator_id character varying(64),
    description text,
    source_id uuid,
    command_index integer DEFAULT 0,
    device_index integer DEFAULT 0,
    has_path_translation_miss boolean DEFAULT false NOT NULL,
    path_translation_miss_count integer DEFAULT 0 NOT NULL,
    path_translation_source character varying(32)
);


--
-- Name: device_tasks_p15; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_tasks_p15 (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    method character varying(64) NOT NULL,
    params jsonb,
    priority integer DEFAULT 10,
    command_key character varying(128),
    cwmp_id character varying(256),
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    retry_count integer DEFAULT 0,
    max_retries integer DEFAULT 3,
    retry_interval_seconds integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_at timestamp with time zone,
    completed_at timestamp with time zone,
    expires_at timestamp with time zone,
    next_attempt_at timestamp with time zone,
    result jsonb,
    error_code integer,
    error_message text,
    source character varying(32) DEFAULT 'api'::character varying,
    creator_id character varying(64),
    description text,
    source_id uuid,
    command_index integer DEFAULT 0,
    device_index integer DEFAULT 0,
    has_path_translation_miss boolean DEFAULT false NOT NULL,
    path_translation_miss_count integer DEFAULT 0 NOT NULL,
    path_translation_source character varying(32)
);


--
-- Name: devices; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.devices (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    serial_number character varying(64) NOT NULL,
    oui character varying(6) NOT NULL,
    product_class character varying(64),
    manufacturer character varying(128),
    model_name character varying(128),
    carrier character varying(16) NOT NULL,
    technology character varying(3) NOT NULL,
    firmware_version character varying(64),
    ip_address inet,
    connection_request_url character varying(256),
    nat_detected boolean DEFAULT false NOT NULL,
    udp_connection_request_address character varying(64),
    last_inform_at timestamp with time zone,
    last_inform_events jsonb,
    inform_interval integer DEFAULT 300,
    site_name character varying(128),
    site_id character varying(64),
    latitude double precision,
    longitude double precision,
    location_source_mode character varying(16) DEFAULT 'tr069'::character varying NOT NULL,
    extension_data jsonb,
    deleted_at timestamp with time zone,
    deleted_by text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    last_boot_at timestamp with time zone,
    boot_count integer DEFAULT 0 NOT NULL,
    product_id uuid,
    param_model_id uuid,
    last_param_sync_at timestamp with time zone,
    lifecycle_state character varying(20) DEFAULT 'registered'::character varying NOT NULL,
    is_online boolean DEFAULT false NOT NULL,
    last_param_sync_failed_at timestamp with time zone,
    last_param_sync_error text,
    last_offline_reason character varying(32),
    recycle_type character varying(16) DEFAULT ''::character varying NOT NULL,
    recycle_executor character varying(128) DEFAULT ''::character varying NOT NULL,
    CONSTRAINT devices_location_source_mode_check CHECK (((location_source_mode)::text = ANY ((ARRAY['tr069'::character varying, 'external'::character varying])::text[]))),
    CONSTRAINT chk_devices_lifecycle_state CHECK (((lifecycle_state)::text = ANY (ARRAY[('discovered'::character varying)::text, ('registered'::character varying)::text, ('provisioning'::character varying)::text, ('commissioned'::character varying)::text, ('maintenance'::character varying)::text, ('decommissioned'::character varying)::text]))),
    CONSTRAINT devices_recycle_type_check CHECK (((recycle_type)::text = ANY ((ARRAY[''::character varying, 'manual'::character varying, 'auto'::character varying])::text[])))
)
PARTITION BY LIST (carrier);


--
-- Name: TABLE devices; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.devices IS '设备核心表（按 carrier 分区）。承载 TR-069 设备身份、ACS 通信通道、实时在线状态、生命周期状态。device_info 表为 1:1 扩展，承载运维属性。';


--
-- Name: COLUMN devices.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.id IS '设备主键 UUID，由 PG gen_random_uuid() 生成。';


--
-- Name: COLUMN devices.serial_number; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.serial_number IS '设备唯一序列号 SN，对应 TR-069 Inform DeviceID.SerialNumber。';


--
-- Name: COLUMN devices.oui; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.oui IS '厂商组织唯一标识符（6 位十六进制），对应 TR-069 DeviceID.OUI。';


--
-- Name: COLUMN devices.product_class; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.product_class IS '产品分类字符串，对应 TR-069 DeviceID.ProductClass；配合 OUI 路由到具体产品定义。';


--
-- Name: COLUMN devices.manufacturer; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.manufacturer IS '厂商名称，对应 TR-069 DeviceID.Manufacturer。';


--
-- Name: COLUMN devices.model_name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.model_name IS '设备型号名，对应 TR-181 Device.DeviceInfo.ModelName；由 ProductRegistry 按 productClass 回填。';


--
-- Name: COLUMN devices.carrier; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.carrier IS '运营商代码：cmcc / ctcc / cucc / other；本表的 LIST 分区键，不可 ALTER。';


--
-- Name: COLUMN devices.technology; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.technology IS '制式：lte / nr / gsm；决定 KPI 计算分支与 RPC 参数路径选择。';


--
-- Name: COLUMN devices.firmware_version; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.firmware_version IS '当前运行固件版本号，对应 TR-181 Device.DeviceInfo.SoftwareVersion；变化时触发 device.firmware.changed 事件。';


--
-- Name: COLUMN devices.ip_address; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.ip_address IS '设备公网 / 局域网 IP（INET 类型），来自 ACS HTTP RemoteAddr 或 Inform 携带的 UDPConnectionRequestAddress。';


--
-- Name: COLUMN devices.connection_request_url; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.connection_request_url IS 'ACS 主动触发 Connection Request 的回调 URL，对应 TR-181 Device.ManagementServer.ConnectionRequestURL。';


--
-- Name: COLUMN devices.nat_detected; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.nat_detected IS '是否检测到设备处于 NAT 之后；为 true 时 ConnReq 走 STUN/UDP 通道而非 HTTP。';


--
-- Name: COLUMN devices.udp_connection_request_address; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.udp_connection_request_address IS 'STUN 协商出的 UDP 直达地址（host:port），NAT 穿透场景使用。';


--
-- Name: COLUMN devices.last_inform_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.last_inform_at IS '最近一次成功接收 Inform 的时间（PG 持久化，重启不丢）；DeviceStatusReconciler 据此判定离线。';


--
-- Name: COLUMN devices.last_inform_events; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.last_inform_events IS '最近一次 Inform 携带的事件码列表（JSONB，如 ["2 PERIODIC"]），用于排障审计。';


--
-- Name: COLUMN devices.inform_interval; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.inform_interval IS '心跳间隔（秒），对应 TR-181 Device.ManagementServer.PeriodicInformInterval；Reconciler 据此动态计算离线阈值 = max(2×inform_interval, 600s)。';


--
-- Name: COLUMN devices.site_name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.site_name IS '所属站点名称（人工录入或运营商规划下发）。';


--
-- Name: COLUMN devices.site_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.site_id IS '所属站点 ID（运营商规划标识符）。';


--
-- Name: COLUMN devices.latitude; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.latitude IS '设备安装位置纬度，对应 TR-181 Device.FAP.GPS.LocationLatitude。';


--
-- Name: COLUMN devices.longitude; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.longitude IS '设备安装位置经度，对应 TR-181 Device.FAP.GPS.LocationLongitude。';


--
-- Name: COLUMN devices.extension_data; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.extension_data IS '扩展字段 JSONB，用于承载未规范化的设备属性或厂商私有数据。';


--
-- Name: COLUMN devices.deleted_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.deleted_at IS '软删除时间戳，NULL 表示未删除；分区索引 idx_devices_deleted_at WHERE deleted_at IS NOT NULL 加速回收站查询。';


--
-- Name: COLUMN devices.deleted_by; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.deleted_by IS '执行软删除操作的用户标识（用户名或用户 ID 字符串）。';


--
-- Name: COLUMN devices.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.created_at IS '记录入库时间（PG 自动填充）。';


--
-- Name: COLUMN devices.updated_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.updated_at IS '记录最近更新时间（PG 触发器 trigger_devices_updated_at 自动维护）。';


--
-- Name: COLUMN devices.last_boot_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.last_boot_at IS '设备最近一次重启完成时间（由 TR-069 Inform 1 BOOT / M Reboot 驱动）';


--
-- Name: COLUMN devices.boot_count; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.boot_count IS '设备累计重启次数';


--
-- Name: COLUMN devices.product_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.product_id IS 'T-0098 由 ProductRegistry 路由 productClass 后写入；分区表无 FK 约束（与 data_model_id 既有模式一致）';


--
-- Name: COLUMN devices.param_model_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.param_model_id IS 'T-0098 参数模型 UUID 软引用；分区表无 FK（与 data_model_id / product_id 模式一致）';


--
-- Name: COLUMN devices.lifecycle_state; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.lifecycle_state IS '设备业务生命周期阶段（T-0162 解耦后单字段）。取值：discovered / registered / provisioning / commissioned / maintenance / decommissioned。与 is_online 正交：commissioned + is_online=false 表示"已入网但当前掉线"。';


--
-- Name: COLUMN devices.is_online; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.is_online IS '实时在线状态（T-0162 解耦后单字段）。true=最近一次 Inform 在心跳阈值内。写入路径：ACS Inform 接收时置 true；DeviceStatusReconciler 扫描超时设备置 false。';


--
-- Name: COLUMN devices.last_offline_reason; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.last_offline_reason IS '最近一次被标记离线的原因。取值：heartbeat_timeout（Reconciler 扫描判定）/ manual（管理面手工操作）/ reboot（重启过程中短暂离线）/ NULL（从未离线或已恢复在线）。诊断字段，不参与业务决策。';


--
-- Name: COLUMN devices.recycle_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.recycle_type IS '移入回收站方式：manual 或 auto';


--
-- Name: COLUMN devices.recycle_executor; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.devices.recycle_executor IS '实际执行软删除的用户或系统任务标识';


--
-- Name: devices_cmcc; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.devices_cmcc (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    serial_number character varying(64) NOT NULL,
    oui character varying(6) NOT NULL,
    product_class character varying(64),
    manufacturer character varying(128),
    model_name character varying(128),
    carrier character varying(16) NOT NULL,
    technology character varying(3) NOT NULL,
    firmware_version character varying(64),
    ip_address inet,
    connection_request_url character varying(256),
    nat_detected boolean DEFAULT false NOT NULL,
    udp_connection_request_address character varying(64),
    last_inform_at timestamp with time zone,
    last_inform_events jsonb,
    inform_interval integer DEFAULT 300,
    site_name character varying(128),
    site_id character varying(64),
    latitude double precision,
    longitude double precision,
    location_source_mode character varying(16) DEFAULT 'tr069'::character varying NOT NULL,
    extension_data jsonb,
    deleted_at timestamp with time zone,
    deleted_by text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    last_boot_at timestamp with time zone,
    boot_count integer DEFAULT 0 NOT NULL,
    product_id uuid,
    param_model_id uuid,
    last_param_sync_at timestamp with time zone,
    lifecycle_state character varying(20) DEFAULT 'registered'::character varying NOT NULL,
    is_online boolean DEFAULT false NOT NULL,
    last_param_sync_failed_at timestamp with time zone,
    last_param_sync_error text,
    last_offline_reason character varying(32),
    recycle_type character varying(16) DEFAULT ''::character varying NOT NULL,
    recycle_executor character varying(128) DEFAULT ''::character varying NOT NULL,
    CONSTRAINT devices_location_source_mode_check CHECK (((location_source_mode)::text = ANY ((ARRAY['tr069'::character varying, 'external'::character varying])::text[]))),
    CONSTRAINT chk_devices_lifecycle_state CHECK (((lifecycle_state)::text = ANY (ARRAY[('discovered'::character varying)::text, ('registered'::character varying)::text, ('provisioning'::character varying)::text, ('commissioned'::character varying)::text, ('maintenance'::character varying)::text, ('decommissioned'::character varying)::text]))),
    CONSTRAINT devices_recycle_type_check CHECK (((recycle_type)::text = ANY ((ARRAY[''::character varying, 'manual'::character varying, 'auto'::character varying])::text[])))
)
WITH (autovacuum_vacuum_scale_factor='0.02', autovacuum_vacuum_threshold='200', autovacuum_analyze_scale_factor='0.02', autovacuum_analyze_threshold='200');


--
-- Name: devices_ctcc; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.devices_ctcc (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    serial_number character varying(64) NOT NULL,
    oui character varying(6) NOT NULL,
    product_class character varying(64),
    manufacturer character varying(128),
    model_name character varying(128),
    carrier character varying(16) NOT NULL,
    technology character varying(3) NOT NULL,
    firmware_version character varying(64),
    ip_address inet,
    connection_request_url character varying(256),
    nat_detected boolean DEFAULT false NOT NULL,
    udp_connection_request_address character varying(64),
    last_inform_at timestamp with time zone,
    last_inform_events jsonb,
    inform_interval integer DEFAULT 300,
    site_name character varying(128),
    site_id character varying(64),
    latitude double precision,
    longitude double precision,
    location_source_mode character varying(16) DEFAULT 'tr069'::character varying NOT NULL,
    extension_data jsonb,
    deleted_at timestamp with time zone,
    deleted_by text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    last_boot_at timestamp with time zone,
    boot_count integer DEFAULT 0 NOT NULL,
    product_id uuid,
    param_model_id uuid,
    last_param_sync_at timestamp with time zone,
    lifecycle_state character varying(20) DEFAULT 'registered'::character varying NOT NULL,
    is_online boolean DEFAULT false NOT NULL,
    last_param_sync_failed_at timestamp with time zone,
    last_param_sync_error text,
    last_offline_reason character varying(32),
    recycle_type character varying(16) DEFAULT ''::character varying NOT NULL,
    recycle_executor character varying(128) DEFAULT ''::character varying NOT NULL,
    CONSTRAINT devices_location_source_mode_check CHECK (((location_source_mode)::text = ANY ((ARRAY['tr069'::character varying, 'external'::character varying])::text[]))),
    CONSTRAINT chk_devices_lifecycle_state CHECK (((lifecycle_state)::text = ANY (ARRAY[('discovered'::character varying)::text, ('registered'::character varying)::text, ('provisioning'::character varying)::text, ('commissioned'::character varying)::text, ('maintenance'::character varying)::text, ('decommissioned'::character varying)::text]))),
    CONSTRAINT devices_recycle_type_check CHECK (((recycle_type)::text = ANY ((ARRAY[''::character varying, 'manual'::character varying, 'auto'::character varying])::text[])))
);


--
-- Name: devices_cucc; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.devices_cucc (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    serial_number character varying(64) NOT NULL,
    oui character varying(6) NOT NULL,
    product_class character varying(64),
    manufacturer character varying(128),
    model_name character varying(128),
    carrier character varying(16) NOT NULL,
    technology character varying(3) NOT NULL,
    firmware_version character varying(64),
    ip_address inet,
    connection_request_url character varying(256),
    nat_detected boolean DEFAULT false NOT NULL,
    udp_connection_request_address character varying(64),
    last_inform_at timestamp with time zone,
    last_inform_events jsonb,
    inform_interval integer DEFAULT 300,
    site_name character varying(128),
    site_id character varying(64),
    latitude double precision,
    longitude double precision,
    location_source_mode character varying(16) DEFAULT 'tr069'::character varying NOT NULL,
    extension_data jsonb,
    deleted_at timestamp with time zone,
    deleted_by text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    last_boot_at timestamp with time zone,
    boot_count integer DEFAULT 0 NOT NULL,
    product_id uuid,
    param_model_id uuid,
    last_param_sync_at timestamp with time zone,
    lifecycle_state character varying(20) DEFAULT 'registered'::character varying NOT NULL,
    is_online boolean DEFAULT false NOT NULL,
    last_param_sync_failed_at timestamp with time zone,
    last_param_sync_error text,
    last_offline_reason character varying(32),
    recycle_type character varying(16) DEFAULT ''::character varying NOT NULL,
    recycle_executor character varying(128) DEFAULT ''::character varying NOT NULL,
    CONSTRAINT devices_location_source_mode_check CHECK (((location_source_mode)::text = ANY ((ARRAY['tr069'::character varying, 'external'::character varying])::text[]))),
    CONSTRAINT chk_devices_lifecycle_state CHECK (((lifecycle_state)::text = ANY (ARRAY[('discovered'::character varying)::text, ('registered'::character varying)::text, ('provisioning'::character varying)::text, ('commissioned'::character varying)::text, ('maintenance'::character varying)::text, ('decommissioned'::character varying)::text]))),
    CONSTRAINT devices_recycle_type_check CHECK (((recycle_type)::text = ANY ((ARRAY[''::character varying, 'manual'::character varying, 'auto'::character varying])::text[])))
);


--
-- Name: devices_other; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.devices_other (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    serial_number character varying(64) NOT NULL,
    oui character varying(6) NOT NULL,
    product_class character varying(64),
    manufacturer character varying(128),
    model_name character varying(128),
    carrier character varying(16) NOT NULL,
    technology character varying(3) NOT NULL,
    firmware_version character varying(64),
    ip_address inet,
    connection_request_url character varying(256),
    nat_detected boolean DEFAULT false NOT NULL,
    udp_connection_request_address character varying(64),
    last_inform_at timestamp with time zone,
    last_inform_events jsonb,
    inform_interval integer DEFAULT 300,
    site_name character varying(128),
    site_id character varying(64),
    latitude double precision,
    longitude double precision,
    location_source_mode character varying(16) DEFAULT 'tr069'::character varying NOT NULL,
    extension_data jsonb,
    deleted_at timestamp with time zone,
    deleted_by text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    last_boot_at timestamp with time zone,
    boot_count integer DEFAULT 0 NOT NULL,
    product_id uuid,
    param_model_id uuid,
    last_param_sync_at timestamp with time zone,
    lifecycle_state character varying(20) DEFAULT 'registered'::character varying NOT NULL,
    is_online boolean DEFAULT false NOT NULL,
    last_param_sync_failed_at timestamp with time zone,
    last_param_sync_error text,
    last_offline_reason character varying(32),
    recycle_type character varying(16) DEFAULT ''::character varying NOT NULL,
    recycle_executor character varying(128) DEFAULT ''::character varying NOT NULL,
    CONSTRAINT devices_location_source_mode_check CHECK (((location_source_mode)::text = ANY ((ARRAY['tr069'::character varying, 'external'::character varying])::text[]))),
    CONSTRAINT chk_devices_lifecycle_state CHECK (((lifecycle_state)::text = ANY (ARRAY[('discovered'::character varying)::text, ('registered'::character varying)::text, ('provisioning'::character varying)::text, ('commissioned'::character varying)::text, ('maintenance'::character varying)::text, ('decommissioned'::character varying)::text]))),
    CONSTRAINT devices_recycle_type_check CHECK (((recycle_type)::text = ANY ((ARRAY[''::character varying, 'manual'::character varying, 'auto'::character varying])::text[])))
);


--
-- Name: TABLE devices_other; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.devices_other IS '设备表 - 国际运营商分区（非 cmcc/ctcc/cucc，如赞比亚 ZED 等；carrier=''intl''）';


--
-- Name: discovered_param_mappings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.discovered_param_mappings (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid NOT NULL,
    software_version character varying(64) NOT NULL,
    standard_path text NOT NULL,
    private_path text NOT NULL,
    entry_type character varying(16) NOT NULL,
    access character varying(16),
    data_type character varying(16),
    change_applies character varying(16),
    min_value bigint,
    max_value bigint,
    is_storable boolean DEFAULT true NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    is_supported boolean DEFAULT true NOT NULL,
    enum_values text,
    enum_labels text,
    mirror_with character varying(256),
    CONSTRAINT discovered_param_mappings_entry_type_check CHECK (((entry_type)::text = ANY (ARRAY[('object'::character varying)::text, ('parameter'::character varying)::text])))
);


--
-- Name: TABLE discovered_param_mappings; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.discovered_param_mappings IS 'T-0098 设备发现映射（设计 §1.2.3）；按 product_id + swVersion 隔离，FileType=11 交集落地';


--
-- Name: COLUMN discovered_param_mappings.product_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.discovered_param_mappings.product_id IS 'CASCADE：product 删除时一并清理本产品全部 swVersion 的交集数据';


--
-- Name: COLUMN discovered_param_mappings.is_storable; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.discovered_param_mappings.is_storable IS '从默认映射继承；不参与 device_attrs_override 覆盖（设计 §1.2.3）';


--
-- Name: COLUMN discovered_param_mappings.is_supported; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.discovered_param_mappings.is_supported IS 'T-0103 从 param_mappings 继承（intersect 时复制），默认 TRUE';


--
-- Name: enabled_pm_indicators_enb; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.enabled_pm_indicators_enb (
    operator_code character varying(100) NOT NULL,
    indicator_id character varying(20) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: enabled_pm_indicators_gnb; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.enabled_pm_indicators_gnb (
    operator_code character varying(100) NOT NULL,
    indicator_id character varying(20) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: enabled_pm_indicators_gsm; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.enabled_pm_indicators_gsm (
    operator_code character varying(100) NOT NULL,
    indicator_id character varying(20) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: event_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.event_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_id uuid,
    device_sn text NOT NULL,
    device_name text,
    device_type text,
    is_gnb boolean DEFAULT false NOT NULL,
    operate_ip text,
    software_version text,
    event_type text NOT NULL,
    event_reason text,
    event_level text DEFAULT 'info'::text NOT NULL,
    event_data jsonb,
    occurred_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT event_logs_event_level_check CHECK ((event_level = ANY (ARRAY['info'::text, 'warning'::text, 'error'::text, 'success'::text])))
);


--
-- Name: fault_log_collect_sub_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.fault_log_collect_sub_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_id uuid NOT NULL,
    device_id uuid NOT NULL,
    firmware_id uuid,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    error_message text,
    retry_count integer DEFAULT 0 NOT NULL,
    max_retries integer DEFAULT 3 NOT NULL,
    device_sn character varying(64),
    ori_version character varying(64),
    dest_version character varying(64),
    command_key character varying(256),
    failure_reason text,
    pre_suspend_status character varying(20),
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_upgrade_sub_tasks_status CHECK (((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('downloading'::character varying)::text, ('uploading'::character varying)::text, ('rebooting'::character varying)::text, ('verifying'::character varying)::text, ('completed'::character varying)::text, ('failed'::character varying)::text, ('suspended'::character varying)::text, ('terminated'::character varying)::text])))
);


--
-- Name: fault_log_collect_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.fault_log_collect_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    firmware_id uuid,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    error_message text,
    retry_count integer DEFAULT 0 NOT NULL,
    max_retries integer DEFAULT 3 NOT NULL,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    task_name character varying(256),
    task_type smallint DEFAULT 1 NOT NULL,
    file_name character varying(256),
    file_md5 character varying(64),
    result character varying(20),
    product_class character varying(64),
    is_keep_config boolean DEFAULT true,
    create_status character varying(16) DEFAULT 'active'::character varying NOT NULL,
    create_user character varying(64) DEFAULT 'system'::character varying NOT NULL,
    total_count integer DEFAULT 0 NOT NULL,
    success_count integer DEFAULT 0 NOT NULL,
    fail_count integer DEFAULT 0 NOT NULL,
    max_concurrent integer DEFAULT 5,
    ended_at timestamp with time zone,
    strategy character varying(16) DEFAULT 'full'::character varying NOT NULL,
    canary_stages jsonb,
    current_stage integer DEFAULT 0 NOT NULL,
    stage_status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    stage_history jsonb DEFAULT '[]'::jsonb NOT NULL,
    auto_advance boolean DEFAULT false NOT NULL,
    auto_advance_minutes integer DEFAULT 0 NOT NULL,
    rollback_reason text,
    rollback_source character varying(32) DEFAULT 'manual'::character varying NOT NULL,
    rollback_target_firmware_id uuid,
    rollback_on_failure boolean DEFAULT false NOT NULL,
    download_file_type text DEFAULT ''::text NOT NULL,
    scheduled_at timestamp with time zone,
    CONSTRAINT chk_upgrade_tasks_create_status CHECK (((create_status)::text = ANY (ARRAY[('active'::character varying)::text, ('suspend'::character varying)::text, ('timing'::character varying)::text]))),
    CONSTRAINT chk_upgrade_tasks_result CHECK (((result IS NULL) OR ((result)::text = ANY (ARRAY[('success'::character varying)::text, ('partial'::character varying)::text, ('failed'::character varying)::text, ('terminated'::character varying)::text])))),
    CONSTRAINT chk_upgrade_tasks_status CHECK (((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('in_progress'::character varying)::text, ('suspended'::character varying)::text, ('ended'::character varying)::text]))),
    CONSTRAINT chk_upgrade_tasks_task_type CHECK ((task_type = ANY (ARRAY[1, 2, 4, 6, 8, 10]))),
    CONSTRAINT upgrade_tasks_auto_advance_minutes_check CHECK (((auto_advance_minutes >= 0) AND (auto_advance_minutes <= 1440))),
    CONSTRAINT upgrade_tasks_current_stage_check CHECK (((current_stage >= 0) AND (current_stage <= 100))),
    CONSTRAINT upgrade_tasks_rollback_source_check CHECK (((rollback_source)::text = ANY (ARRAY[('manual'::character varying)::text, ('canary_failure'::character varying)::text, ('compatibility'::character varying)::text, ('scheduled'::character varying)::text]))),
    CONSTRAINT upgrade_tasks_stage_status_check CHECK (((stage_status)::text = ANY (ARRAY[('pending'::character varying)::text, ('running'::character varying)::text, ('paused'::character varying)::text, ('aborted'::character varying)::text, ('completed'::character varying)::text]))),
    CONSTRAINT upgrade_tasks_strategy_check CHECK (((strategy)::text = ANY (ARRAY[('full'::character varying)::text, ('canary'::character varying)::text])))
);


--
-- Name: firmware_versions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.firmware_versions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_class text,
    version character varying(64) NOT NULL,
    file_name character varying(256) NOT NULL,
    file_size bigint,
    minio_path character varying(512) NOT NULL,
    compatible_oui jsonb DEFAULT '[]'::jsonb,
    release_notes text,
    status character varying(16) DEFAULT 'active'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    file_type smallint DEFAULT 0 NOT NULL,
    md5_val character varying(64),
    recommend boolean DEFAULT false,
    uploader character varying(64),
    manufacturer character varying(128),
    description text,
    sha256_val character varying(64),
    signature text,
    signature_alg character varying(32),
    public_key_id character varying(128),
    product_id uuid,
    product_ids uuid[] DEFAULT '{}'::uuid[] NOT NULL,
    CONSTRAINT chk_firmware_versions_file_type CHECK ((file_type = ANY (ARRAY[0, 1, 6]))),
    CONSTRAINT chk_firmware_versions_status CHECK (((status)::text = ANY (ARRAY[('active'::character varying)::text, ('deprecated'::character varying)::text, ('archived'::character varying)::text])))
);


--
-- Name: COLUMN firmware_versions.product_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.firmware_versions.product_id IS '#492 固件所属产品（products.id）。上传选产品名 → 存此列；升级/库列表按产品名展示与过滤。';


--
-- Name: COLUMN firmware_versions.product_ids; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.firmware_versions.product_ids IS '#638 固件适用的多个产品 ID 列表（products.id）。上传支持多选；product_id 保留为兼容主产品=列表首项。列表/任务按产品过滤命中 ANY(product_ids)。';


--
-- Name: ftp_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ftp_configs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    config_name character varying(200) NOT NULL,
    host character varying(255) NOT NULL,
    port integer DEFAULT 21 NOT NULL,
    username character varying(100) NOT NULL,
    password_encrypted text,
    protocol character varying(10) DEFAULT 'FTP'::character varying NOT NULL,
    remote_path character varying(512) DEFAULT '/'::character varying NOT NULL,
    passive boolean DEFAULT true NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: indicator_file_descriptions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.indicator_file_descriptions (
    tech text NOT NULL,
    platform text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: indicator_group_enb; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.indicator_group_enb (
    id character varying(64) NOT NULL,
    en_name character varying(200),
    operator_code character varying(100),
    is_build_in character(1) DEFAULT '0'::bpchar NOT NULL,
    description text,
    parent_id character varying(64) NOT NULL,
    cn_name character varying(200),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: indicator_group_gnb; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.indicator_group_gnb (
    id character varying(64) NOT NULL,
    en_name character varying(200),
    operator_code character varying(100),
    is_build_in character(1) DEFAULT '0'::bpchar NOT NULL,
    description text,
    parent_id character varying(64) NOT NULL,
    cn_name character varying(200),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: indicator_group_gsm; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.indicator_group_gsm (
    id character varying(64) NOT NULL,
    en_name character varying(200),
    operator_code character varying(100),
    is_build_in character(1) DEFAULT '0'::bpchar NOT NULL,
    description text,
    parent_id character varying(64) NOT NULL,
    cn_name character varying(200),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: indicator_threshold; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.indicator_threshold (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    indicator_id character varying(20),
    threshold_period character varying(10),
    threshold_color character varying(10),
    threshold_low character varying(20),
    threshold_high character varying(20),
    threshold_level character varying(10),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: kpi_definitions; Type: TABLE; Schema: public; Owner: -
--

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
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: kpi_thresholds; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.kpi_thresholds (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    kpi_name character varying(128) NOT NULL,
    carrier character varying(16),
    technology character varying(16),
    warning_threshold double precision,
    minor_threshold double precision,
    major_threshold double precision,
    critical_threshold double precision,
    comparison character varying(16) DEFAULT 'gt'::character varying NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: managed_files; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.managed_files (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    file_name character varying(500) NOT NULL,
    file_type character varying(20) DEFAULT 'other'::character varying NOT NULL,
    file_size bigint DEFAULT 0 NOT NULL,
    minio_path text NOT NULL,
    content_type character varying(200) DEFAULT 'application/octet-stream'::character varying,
    uploader character varying(100),
    device_sn character varying(64),
    status character varying(20) DEFAULT 'uploaded'::character varying NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: menus; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.menus (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(64) NOT NULL,
    type character varying(16) NOT NULL,
    permission_key character varying(128) NOT NULL,
    parent_id uuid,
    sort_order integer DEFAULT 0 NOT NULL,
    route_path character varying(256),
    component_path character varying(256),
    icon character varying(64),
    show_status character varying(16) DEFAULT 'show'::character varying NOT NULL,
    status character varying(16) DEFAULT 'normal'::character varying NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by uuid,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    name_i18n jsonb,
    feature_code text[],
    CONSTRAINT chk_menu_type CHECK (((type)::text = ANY (ARRAY[('directory'::character varying)::text, ('menu'::character varying)::text, ('button'::character varying)::text]))),
    CONSTRAINT chk_show_status CHECK (((show_status)::text = ANY (ARRAY[('show'::character varying)::text, ('hide'::character varying)::text]))),
    CONSTRAINT chk_status CHECK (((status)::text = ANY (ARRAY[('normal'::character varying)::text, ('disabled'::character varying)::text])))
);


--
-- Name: TABLE menus; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.menus IS '菜单表：通过菜单统一管理权限，菜单即权限';


--
-- Name: COLUMN menus.type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.menus.type IS '菜单类型: directory=一级目录, menu=二级菜单, button=三级按钮';


--
-- Name: COLUMN menus.permission_key; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.menus.permission_key IS '权限标识，格式: {module}:{page}[:{action}]，如 device:list:query';


--
-- Name: COLUMN menus.name_i18n; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.menus.name_i18n IS '多语言译文 JSONB，键为 locale code（如 zh-CN/en-US），值为对应译文；NULL 表示未配置多语言，前端回退到 name 字段';


--
-- Name: mml_audit_log; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mml_audit_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_id uuid,
    command_code character varying(255),
    operation_type character varying(20),
    device_sn character varying(64),
    parameters jsonb,
    param_paths jsonb,
    result_status character varying(20),
    result_message text,
    creator character varying(100),
    duration_ms numeric(12,3),
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE mml_audit_log; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.mml_audit_log IS 'MML 命令执行审计日志。每次 ExecuteCommand 对每个 device+command 组合写一条记录，
     记录下发时的参数、最终结果状态与耗时，供合规审计与运维回溯。';


--
-- Name: mml_catalog_link_health; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mml_catalog_link_health (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    spec_version character varying(64) NOT NULL,
    standard_path character varying(512) NOT NULL,
    group_code_object character varying(512) NOT NULL,
    operation_type character varying(8),
    failure_reason character varying(8) NOT NULL,
    notes text,
    decision text,
    owner character varying(64),
    detected_at timestamp with time zone DEFAULT now() NOT NULL,
    resolved_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_mml_catalog_link_health_reason CHECK (((failure_reason)::text = ANY (ARRAY[('A'::character varying)::text, ('B'::character varying)::text, ('C'::character varying)::text, ('D'::character varying)::text])))
);


--
-- Name: TABLE mml_catalog_link_health; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.mml_catalog_link_health IS '§R-2.5.2: catalog Loader 解析 spec MD 时把"无法在 standard_params 命中"的 path 写入。每次 Loader 启动按 (spec_version, standard_path, group_code_object) 幂等 UPSERT；admin API GET /api/v1/mml/catalog/link-health 列出 resolved_at IS NULL 的项；维护者修正 spec / standard_params 后，下次 Loader 启动如果该 path 命中标准参数树，自动把对应记录的 resolved_at 写为 NOW()';


--
-- Name: COLUMN mml_catalog_link_health.failure_reason; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_catalog_link_health.failure_reason IS 'A=标准参数树未覆盖（spec 合法 path 但 standard_params 未导入）/ B=spec 写错（拼写/大小写/{i} 嵌套不符 TR-181）/ C=厂商私有路径泄漏到 standardPath 列（应移到 privatePath 或 X_VENDOR_*）/ D=命令已废弃（新协议版本已移除）';


--
-- Name: mml_catalog_orphan_paths_audit_t0171; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mml_catalog_orphan_paths_audit_t0171 (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    standard_path text NOT NULL,
    standard_param_id uuid NOT NULL,
    object_prefix text NOT NULL,
    access character varying(16),
    data_type character varying(16),
    disposition character varying(32),
    suggested_group_code text,
    suggested_command_code text,
    reviewer character varying(100),
    reviewed_at timestamp with time zone,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE mml_catalog_orphan_paths_audit_t0171; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.mml_catalog_orphan_paths_audit_t0171 IS 'T-0171 审计：standard_params 中未被任何 mml_command_sub_fields 引用的孤儿 path 快照（1544 行 BAICELLS 私有扩展 + 5G/GSM/SAS/IPsec 子树）。业务方 review 后填 disposition 决定处置；本表不污染 mml_commands / sub_fields 业务表。';


--
-- Name: COLUMN mml_catalog_orphan_paths_audit_t0171.disposition; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_catalog_orphan_paths_audit_t0171.disposition IS '业务方决策：ignore=忽略仅作字典存档 / private_catalog=进独立私有 catalog（用 omcctl mml import-spec-md 生成 seed） / manual_subfield=admin UI 手工补关联到 现有命令。NULL=待 review。';


--
-- Name: mml_command_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mml_command_groups (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    group_code character varying(255) NOT NULL,
    group_name_zh character varying(500) NOT NULL,
    group_name_en character varying(500),
    path public.ltree NOT NULL,
    param_version character varying(50) NOT NULL,
    display_order integer DEFAULT 0 NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_by character varying(100),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    name_i18n jsonb DEFAULT '{}'::jsonb NOT NULL,
    source character varying(20) DEFAULT 'admin'::character varying NOT NULL,
    catalog_protected boolean DEFAULT false NOT NULL,
    object_path_template character varying(256),
    chapter_code character varying(8),
    instance_arity smallint DEFAULT 0 NOT NULL,
    instance_levels text[],
    deprecated_at timestamp with time zone,
    family_code character varying(64) DEFAULT ''::character varying NOT NULL,
    family_name_zh character varying(128) DEFAULT ''::character varying NOT NULL
);


--
-- Name: COLUMN mml_command_groups.source; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_command_groups.source IS '来源：standard / admin';


--
-- Name: COLUMN mml_command_groups.catalog_protected; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_command_groups.catalog_protected IS 'true=admin UI 不可删除关键 group';


--
-- Name: COLUMN mml_command_groups.object_path_template; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_command_groups.object_path_template IS 'v2.3 catalog: object 维度归一化 TR-181 路径模板（如 Device.DeviceInfo.*），R-1 标识';


--
-- Name: COLUMN mml_command_groups.chapter_code; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_command_groups.chapter_code IS 'v2.3 catalog: spec 章节代码（SA…SR），仅用于排序元数据，不渲染';


--
-- Name: COLUMN mml_command_groups.instance_arity; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_command_groups.instance_arity IS 'v2.3 catalog: {i} 占位符层级数；0=无实例；多层如 SR 段达 5';


--
-- Name: COLUMN mml_command_groups.instance_levels; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_command_groups.instance_levels IS 'v2.3 catalog: 多层 {i} 的层级语义名（["MU","Slot","EU","RU","RFChannel"]）';


--
-- Name: mml_command_sub_field_overrides; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mml_command_sub_field_overrides (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    sub_field_id uuid NOT NULL,
    owner_user_id uuid NOT NULL,
    default_selected_override boolean,
    label_i18n_override jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE mml_command_sub_field_overrides; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.mml_command_sub_field_overrides IS 'R-6/D29: per-user 保护用户调过的 default_selected / label_i18n，Loader UPSERT 不覆盖。运行期读取走 LEFT JOIN + COALESCE';


--
-- Name: mml_command_sub_fields; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mml_command_sub_fields (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    command_id uuid NOT NULL,
    mml_code character varying(100) NOT NULL,
    label_i18n jsonb DEFAULT '{}'::jsonb NOT NULL,
    default_selected boolean DEFAULT true NOT NULL,
    is_required boolean DEFAULT false NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    standard_path_id uuid NOT NULL,
    access_type character varying(4),
    deprecated_at timestamp with time zone,
    is_supported boolean DEFAULT true NOT NULL,
    CONSTRAINT chk_sub_field_access_type CHECK (((access_type IS NULL) OR ((access_type)::text = ANY (ARRAY[('RO'::character varying)::text, ('RW'::character varying)::text]))))
);


--
-- Name: TABLE mml_command_sub_fields; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.mml_command_sub_fields IS 'MML 命令 → sub-field 多对多关系（替代 000090 DROP 的 mml_command_params_rel）；T-0123 老交互恢复方案核心数据结构';


--
-- Name: COLUMN mml_command_sub_fields.mml_code; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_command_sub_fields.mml_code IS '老系统 MML 字符串内部 code（如 LTE_GSM_MODEL_NAME）；命令上下文相关';


--
-- Name: COLUMN mml_command_sub_fields.label_i18n; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_command_sub_fields.label_i18n IS '命令上下文的 sub-field 显示标签（覆盖 mml_params.name_i18n 全局默认）';


--
-- Name: COLUMN mml_command_sub_fields.default_selected; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_command_sub_fields.default_selected IS 'LST 命令：UI 是否默认勾选；老系统所有 sub-field 默认全勾选';


--
-- Name: COLUMN mml_command_sub_fields.is_required; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_command_sub_fields.is_required IS 'MOD/ADD 命令：sub-field 是否必填（前端校验）';


--
-- Name: COLUMN mml_command_sub_fields.standard_path_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_command_sub_fields.standard_path_id IS 'FK → standard_params(id)。MML 命令的子字段直接引用系统级标准 path 字典，由 product/param-model 前端统一管理。运行时 ACS 通过 ParamModel Translator 把 standard_path 翻译为厂商私有 path 下发（暂未启用，当前直接用 standard_path）。';


--
-- Name: COLUMN mml_command_sub_fields.access_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_command_sub_fields.access_type IS 'R-4: RO/RW 冗余列加速前端按 op 过滤（MOD 隐藏 RO）；Loader 写入时按 standardPath 元属性拷贝';


--
-- Name: COLUMN mml_command_sub_fields.is_supported; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_command_sub_fields.is_supported IS 'CPE 是否支持该 path。true (默认) = supported;false = 测试验证不支持(GPV fault 9005)。MML executor 构造 GPV 时按此列过滤,catalog loader 的 ON CONFLICT UPDATE 不动此列,保留学习状态。';


--
-- Name: mml_commands; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mml_commands (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    command_name character varying(500) NOT NULL,
    command_code character varying(255) NOT NULL,
    category character varying(50),
    description text,
    rpc_method character varying(50) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    target_paths jsonb DEFAULT '[]'::jsonb NOT NULL,
    target_object character varying(500),
    group_id uuid,
    command_name_i18n jsonb DEFAULT '{}'::jsonb NOT NULL,
    require_confirm boolean DEFAULT false NOT NULL,
    confirm_msg_i18n jsonb DEFAULT '{}'::jsonb NOT NULL,
    operation_type character varying(20) DEFAULT 'LST'::character varying NOT NULL,
    logical_name_i18n jsonb DEFAULT '{}'::jsonb NOT NULL,
    source character varying(20) DEFAULT 'admin'::character varying NOT NULL,
    catalog_protected boolean DEFAULT false NOT NULL,
    platform_tags jsonb DEFAULT '{}'::jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deprecated_at timestamp with time zone,
    tree_node_refs jsonb DEFAULT '[]'::jsonb NOT NULL,
    instance_range_meta jsonb DEFAULT '[]'::jsonb NOT NULL,
    help_doc text DEFAULT ''::text,
    notes text DEFAULT ''::text
);


--
-- Name: COLUMN mml_commands.logical_name_i18n; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_commands.logical_name_i18n IS '逻辑命令显示名（JSONB），如 {"en-US":"Device info","zh-CN":"设备信息"}；命令树叶子 label 前缀来源';


--
-- Name: COLUMN mml_commands.source; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_commands.source IS '来源：standard（XML import）/ admin（手动创建）';


--
-- Name: COLUMN mml_commands.catalog_protected; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_commands.catalog_protected IS 'true=admin UI 不可删除（Q2=C 决议）；不可 PATCH';


--
-- Name: COLUMN mml_commands.platform_tags; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_commands.platform_tags IS '平台支持标记（JSONB），形如 {"mobile":true,"broadband":true,"platform_codes":["1"]}；由老 OMC small_cell_param_group 的 mobile_support/broadband_support/platform_support 三列合并而来。前端按当前皮肤/设备类型过滤命令树叶子。';


--
-- Name: COLUMN mml_commands.updated_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_commands.updated_at IS '最后修改时间，由 update_updated_at_column 触发器自动维护；也由 refresh_mml_command_target_paths()（000095）在 sub_fields 变更时回填。';


--
-- Name: COLUMN mml_commands.deprecated_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_commands.deprecated_at IS 'v2.3 catalog: Loader 软删标记；列表查询默认过滤非空';


--
-- Name: COLUMN mml_commands.tree_node_refs; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_commands.tree_node_refs IS '§R-2.5: 引用标准参数树(standard_params.standardPath) 的列表（JSONB array of strings）。运行时 JOIN standard_params 拉 path 类型/范围/权限/中文描述。替代旧 target_paths 字符串数组（兼容窗口期，target_paths 仍由 v1 Loader 写入；P2 v2 Loader 同时写两列，P3 切流量到 tree_node_refs 后再 DROP target_paths）';


--
-- Name: COLUMN mml_commands.instance_range_meta; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_commands.instance_range_meta IS '§R-4.1.1: 每层 {i} 占位符的取值范围元数据，按 layer 顺序排列。每个元素含 layer/rangeExpr/rangeMin/rangeMax/dynamic/nSource/description；前端 LST Control Panel 渲染 InstancePicker 时用于校验：静态范围 [min,max] 闭区间硬校验，动态范围（dynamic=true，nSource 指向某 NumberOfEntries 参数）由设备 LST 缓存上限 + 后端兜底';


--
-- Name: COLUMN mml_commands.help_doc; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_commands.help_doc IS '命令的帮助文档（前端 tooltip / detail panel 使用）';


--
-- Name: COLUMN mml_commands.notes; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_commands.notes IS '命令的备注信息（admin 后台维护）';


--
-- Name: mml_custom_command; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mml_custom_command (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    command_name character varying(200) NOT NULL,
    command_code character varying(100) NOT NULL,
    operation_type character varying(20) NOT NULL,
    command_scope character varying(20) DEFAULT 'private'::character varying NOT NULL,
    parameters jsonb DEFAULT '{}'::jsonb NOT NULL,
    param_paths jsonb DEFAULT '[]'::jsonb NOT NULL,
    description text,
    creator character varying(100) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    category_group character varying(50),
    owner_user_id uuid,
    CONSTRAINT chk_mml_custom_command_op CHECK (((operation_type)::text = ANY (ARRAY[('LST'::character varying)::text, ('MOD'::character varying)::text, ('ADD'::character varying)::text, ('RMV'::character varying)::text]))),
    CONSTRAINT chk_mml_custom_command_scope CHECK (((command_scope)::text = ANY (ARRAY[('private'::character varying)::text, ('public'::character varying)::text])))
);


--
-- Name: COLUMN mml_custom_command.owner_user_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_custom_command.owner_user_id IS 'UUID FK → users.id；正在替代历史 creator(varchar) 作为所有权字段。Phase 1 仅 dual-write，read 路径仍按 creator；Phase 2 切读后 creator 进入弃用期。';


--
-- Name: mml_custom_command_paths; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mml_custom_command_paths (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    command_id uuid NOT NULL,
    standard_path_id uuid NOT NULL,
    default_selected boolean DEFAULT true NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE mml_custom_command_paths; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.mml_custom_command_paths IS 'issue #115 调整3：自定义命令↔标准路径精瘦关联表。仅存关联+排序+默认勾选；元数据与是否支持读时 JOIN standard_params/param_mappings，不重复落库。standard_path_id NOT NULL = path 仅来自字典。';


--
-- Name: mml_param_versions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mml_param_versions (
    version_code character varying(50) NOT NULL,
    version_name character varying(200) NOT NULL,
    description text,
    is_active boolean DEFAULT true NOT NULL,
    is_deprecated boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source character varying(20) DEFAULT 'standard'::character varying NOT NULL,
    content_hash character varying(64)
);


--
-- Name: COLUMN mml_param_versions.content_hash; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_param_versions.content_hash IS 'Sprint B 增量重载：标准 XML 文件 sha256 hex。Loader 通过比对此值跳过未变更 reload';


--
-- Name: mml_scripts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mml_scripts (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    script_name character varying(200) NOT NULL,
    description text,
    content text NOT NULL,
    creator character varying(100),
    tags jsonb DEFAULT '[]'::jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    status character varying(20) DEFAULT 'active'::character varying NOT NULL,
    last_run_status character varying(20),
    last_run_at timestamp with time zone,
    start_time timestamp with time zone,
    end_time timestamp with time zone,
    type character varying(20) DEFAULT 'manual'::character varying NOT NULL,
    progress numeric(5,2) DEFAULT 0 NOT NULL,
    result jsonb DEFAULT '{}'::jsonb,
    import_session_id uuid DEFAULT gen_random_uuid() NOT NULL,
    original_filename text DEFAULT ''::text NOT NULL,
    content_sha256 text DEFAULT ''::text NOT NULL,
    validation_version text DEFAULT ''::text NOT NULL,
    validated_at timestamp with time zone,
    plan_items jsonb DEFAULT '[]'::jsonb NOT NULL,
    validation_summary jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: COLUMN mml_scripts.last_run_status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_scripts.last_run_status IS 'MML 脚本最近一次关联 mml_task 的终态：completed/failed/partial。每次执行详情查 mml_tasks。';


--
-- Name: COLUMN mml_scripts.last_run_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_scripts.last_run_at IS 'MML 脚本最近一次执行完成时刻（与 last_run_status 对应）。';


--
-- Name: mml_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mml_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_name character varying(200),
    script_id uuid,
    device_sns jsonb NOT NULL,
    commands jsonb DEFAULT '[]'::jsonb NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    results jsonb DEFAULT '[]'::jsonb,
    creator character varying(100),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    executor character varying(100),
    next_trigger_at timestamp with time zone,
    parent_task_id uuid,
    execute_type character varying(20) DEFAULT 'immediate'::character varying NOT NULL,
    period_start timestamp with time zone,
    period_end timestamp with time zone,
    period_time character varying(20) DEFAULT ''::character varying NOT NULL,
    offline_retry boolean DEFAULT false NOT NULL,
    offline_retry_wait integer DEFAULT 0 NOT NULL,
    failed_retry boolean DEFAULT false NOT NULL,
    failed_retry_count integer DEFAULT 0 NOT NULL,
    failed_retry_interval integer DEFAULT 0 NOT NULL,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    total_devices integer DEFAULT 0 NOT NULL,
    success_count integer DEFAULT 0 NOT NULL,
    failed_count integer DEFAULT 0 NOT NULL,
    result text,
    product_resolved boolean DEFAULT true NOT NULL,
    matched_product_id uuid,
    matched_product_class character varying(64),
    path_translation_source character varying(32),
    scheduled_at timestamp with time zone,
    export_object text,
    device_export_objects jsonb DEFAULT '{}'::jsonb NOT NULL,
    export_generated_at timestamp with time zone,
    execute_mode text DEFAULT 'common'::text NOT NULL,
    plan_items jsonb DEFAULT '[]'::jsonb NOT NULL,
    script_content_sha256 text DEFAULT ''::text NOT NULL,
    script_validation_version text DEFAULT ''::text NOT NULL,
    request_id text,
    CONSTRAINT mml_tasks_execute_mode_check CHECK ((execute_mode = ANY (ARRAY['common'::text, 'device_bound'::text])))
);


--
-- Name: COLUMN mml_tasks.next_trigger_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_tasks.next_trigger_at IS 'MML 任务下一次触发时刻：scheduled 为一次性值；periodic 为模板行的下次 period_time 命中时刻（由 Scheduler 滚动更新）';


--
-- Name: COLUMN mml_tasks.parent_task_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_tasks.parent_task_id IS 'periodic 模板行生成的子实例指向模板 mml_task.id；非 periodic 子实例或模板本身为 NULL';


--
-- Name: COLUMN mml_tasks.product_resolved; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_tasks.product_resolved IS 'T-0168: 设备 product_class 是否通过 ProductRegistry.MatchProductClass 命中 product。false=orphan（激进路线下 path 走 orphan_passthrough 原路径下发，触发 Prometheus 告警 mml_path_translation_orphan_total）。历史数据默认 true（不回填，假设旧任务非 orphan）。';


--
-- Name: COLUMN mml_tasks.matched_product_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_tasks.matched_product_id IS 'T-0168: 命中的 product.id（UUID）。NULL=product_resolved=false 或非 MML 翻译路径。便于审计查询 join products 表拿厂商/参数模型信息。';


--
-- Name: COLUMN mml_tasks.matched_product_class; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_tasks.matched_product_class IS 'T-0168: 翻译时使用的设备 product_class 字符串（取 device_sns[0].product_class）。冗余存储避免 join devices 表；R-8.4 保证 task 内一致。';


--
-- Name: COLUMN mml_tasks.path_translation_source; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_tasks.path_translation_source IS 'T-0168: 任务级翻译来源汇总。枚举值：discovered（全部走 discovered_param_mappings）/ default（全部走 param_mappings 默认表）/ passthrough（mapping 缺失，原路径下发）/ orphan_passthrough（product 未识别，激进路线下发）/ mixed（任务内多种来源混合）。NULL=非 MML 翻译路径或 PathTranslator 未注入。';


--
-- Name: COLUMN mml_tasks.scheduled_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_tasks.scheduled_at IS 'MML 任务计划执行时刻：execute_type=scheduled 时为一次性执行时间；execute_type=immediate / periodic / 立即派发场景为 NULL。Scheduler 与 next_trigger_at 配合使用（next_trigger_at 是滚动触发时间，scheduled_at 是原始计划时间，便于审计 / 列表展示）。';


--
-- Name: COLUMN mml_tasks.execute_mode; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_tasks.execute_mode IS 'MML task execution mode: common broadcasts commands to selected devices; device_bound uses plan_items as the execution source.';


--
-- Name: COLUMN mml_tasks.plan_items; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_tasks.plan_items IS 'Device-bound execution plan rows. Each row binds source line, device SN, per-device order, raw line, and normalized command JSON.';


--
-- Name: COLUMN mml_tasks.request_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mml_tasks.request_id IS 'Client-generated idempotency key for task creation requests.';


--
-- Name: model_upload_intents; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.model_upload_intents (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_id uuid NOT NULL,
    upload_task_id uuid NOT NULL,
    discovery_log_id uuid,
    source_event_id character varying(128) NOT NULL,
    model_version character varying(128),
    model_hash character varying(128),
    status character varying(24) DEFAULT 'requested'::character varying NOT NULL,
    failure_code character varying(64),
    failure_message text,
    attempts integer DEFAULT 0 NOT NULL,
    next_attempt_at timestamp with time zone DEFAULT now() NOT NULL,
    last_error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT model_upload_intents_status_chk CHECK (((status)::text = ANY ((ARRAY['requested'::character varying, 'uploaded'::character varying, 'not_supported'::character varying, 'failed'::character varying, 'sync_queued'::character varying, 'sync_submitted'::character varying, 'manual_review'::character varying])::text[])))
);


--
-- Name: mr_customize_task; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mr_customize_task (
    task_id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_name character varying(128) NOT NULL,
    mr_type character varying(64) DEFAULT 'MRS,MRE,MRO'::character varying NOT NULL,
    statis_period character varying(16) DEFAULT '5120'::character varying NOT NULL,
    report_period character varying(8) DEFAULT '15'::character varying NOT NULL,
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone,
    task_status character varying(16) DEFAULT 'waitting'::character varying NOT NULL,
    task_result character varying(16),
    creator character varying(64) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    target_device_sns text[],
    CONSTRAINT chk_mr_task_report_period CHECK (((report_period)::text = ANY (ARRAY[('15'::character varying)::text, ('30'::character varying)::text, ('60'::character varying)::text]))),
    CONSTRAINT chk_mr_task_status CHECK (((task_status)::text = ANY (ARRAY[('waitting'::character varying)::text, ('on'::character varying)::text, ('off'::character varying)::text, ('suspend'::character varying)::text, ('termination'::character varying)::text]))),
    CONSTRAINT chk_mr_task_time_order CHECK (((end_time IS NULL) OR (end_time > start_time)))
);


--
-- Name: TABLE mr_customize_task; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.mr_customize_task IS 'MR 测量任务主表（F05），由用户创建，worker scheduler 按 start_time/end_time 自动下发开启/关闭 SPV';


--
-- Name: COLUMN mr_customize_task.mr_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mr_customize_task.mr_type IS '测量类型组合，逗号分隔，强制含 MRS,MRE,MRO';


--
-- Name: COLUMN mr_customize_task.statis_period; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mr_customize_task.statis_period IS 'PeriodicReportInterval 下发原值';


--
-- Name: COLUMN mr_customize_task.report_period; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mr_customize_task.report_period IS '上报周期（分钟），下发为 UploadPeriod = report_period × 60 秒';


--
-- Name: COLUMN mr_customize_task.task_status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mr_customize_task.task_status IS 'waitting=待执行/on=执行中/off=已关闭/suspend=已挂起/termination=终止中';


--
-- Name: COLUMN mr_customize_task.target_device_sns; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mr_customize_task.target_device_sns IS '用户选定的目标设备 SN 列表；scheduler 开启任务时 JOIN mr_device_mappings 展开为 cells';


--
-- Name: mr_customize_task_progress; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mr_customize_task_progress (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_id uuid NOT NULL,
    small_cell_code character varying(64) NOT NULL,
    serial_number character varying(64) NOT NULL,
    host_name character varying(128),
    progress_status character varying(32) DEFAULT 'pending'::character varying NOT NULL,
    health_status character varying(16) DEFAULT 'unknown'::character varying NOT NULL,
    fault_code character varying(32),
    last_heartbeat timestamp with time zone,
    missed_heartbeat integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_mr_progress_health CHECK (((health_status)::text = ANY (ARRAY[('normal'::character varying)::text, ('abnormal'::character varying)::text, ('unknown'::character varying)::text]))),
    CONSTRAINT chk_mr_progress_status CHECK (((progress_status)::text = ANY (ARRAY[('pending'::character varying)::text, ('openSuccess'::character varying)::text, ('openFailure'::character varying)::text, ('closeSuccess'::character varying)::text, ('closeFailure'::character varying)::text, ('unsupport'::character varying)::text, ('timeOut'::character varying)::text, ('noPermission'::character varying)::text])))
);


--
-- Name: TABLE mr_customize_task_progress; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.mr_customize_task_progress IS 'MR 任务 — 小站级进度，含下发结果与上报心跳健康度';


--
-- Name: COLUMN mr_customize_task_progress.progress_status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mr_customize_task_progress.progress_status IS 'pending/openSuccess/openFailure/closeSuccess/closeFailure/unsupport/timeOut/noPermission';


--
-- Name: COLUMN mr_customize_task_progress.health_status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mr_customize_task_progress.health_status IS 'normal/abnormal/unknown — Redis MRFileReport TTL 巡检结果';


--
-- Name: COLUMN mr_customize_task_progress.missed_heartbeat; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mr_customize_task_progress.missed_heartbeat IS '连续未命中心跳次数；scheduler 阈值 (默认 2) 时标记 abnormal';


--
-- Name: mr_device_mappings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mr_device_mappings (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    device_name character varying(200),
    cell_id character varying(64) NOT NULL,
    cell_name character varying(200),
    enabled boolean DEFAULT true NOT NULL,
    sampling_interval integer DEFAULT 15 NOT NULL,
    last_collect_time timestamp with time zone,
    total_records bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: mr_indicators; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mr_indicators (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    indicator_name character varying(200) NOT NULL,
    indicator_code character varying(100) NOT NULL,
    description text,
    unit character varying(50),
    category character varying(50),
    value_range_min double precision,
    value_range_max double precision,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: ne_message_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ne_message_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(128) NOT NULL,
    device_id uuid,
    message_type character varying(64) NOT NULL,
    direction character varying(8) NOT NULL,
    content text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: nedirect_commands; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.nedirect_commands (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    session_id uuid NOT NULL,
    device_sn character varying(128) NOT NULL,
    command text DEFAULT ''::text NOT NULL,
    status character varying(32) DEFAULT 'pending'::character varying NOT NULL,
    response text DEFAULT ''::text NOT NULL,
    error_msg text DEFAULT ''::text NOT NULL,
    sent_at timestamp with time zone DEFAULT now() NOT NULL,
    respond_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: nedirect_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.nedirect_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_id uuid NOT NULL,
    device_sn character varying(128) NOT NULL,
    user_id character varying(128) NOT NULL,
    username character varying(128) DEFAULT ''::character varying NOT NULL,
    status character varying(32) DEFAULT 'active'::character varying NOT NULL,
    ip_address character varying(64) DEFAULT ''::character varying NOT NULL,
    connected_at timestamp with time zone DEFAULT now() NOT NULL,
    last_active_at timestamp with time zone DEFAULT now() NOT NULL,
    disconnect_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: northbound_endpoints; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.northbound_endpoints (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(128) NOT NULL,
    protocol character varying(16) NOT NULL,
    purpose character varying(64) DEFAULT ''::character varying NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    host character varying(255) DEFAULT ''::character varying NOT NULL,
    port integer,
    username character varying(128) DEFAULT ''::character varying NOT NULL,
    credential_ref character varying(255) DEFAULT ''::character varying NOT NULL,
    auth_mode character varying(32) DEFAULT 'PASSWORD'::character varying NOT NULL,
    remote_root character varying(512) DEFAULT ''::character varying NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    status character varying(32) DEFAULT 'terminated'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT northbound_endpoints_auth_mode_check CHECK (((auth_mode)::text = ANY (ARRAY[('PASSWORD'::character varying)::text, ('PRIVATE_KEY'::character varying)::text, ('NONE'::character varying)::text]))),
    CONSTRAINT northbound_endpoints_port_check CHECK (((port IS NULL) OR ((port > 0) AND (port <= 65535)))),
    CONSTRAINT northbound_endpoints_protocol_check CHECK (((protocol)::text = ANY (ARRAY[('FTP'::character varying)::text, ('SFTP'::character varying)::text, ('SNMP'::character varying)::text, ('SOCKET'::character varying)::text, ('API'::character varying)::text]))),
    CONSTRAINT northbound_endpoints_status_check CHECK (((status)::text = ANY (ARRAY[('normal'::character varying)::text, ('terminated'::character varying)::text])))
);


--
-- Name: northbound_field_mappings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.northbound_field_mappings (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    profile_kind character varying(32) NOT NULL,
    profile_code character varying(64) NOT NULL,
    domain character varying(32) NOT NULL,
    object_code character varying(64) NOT NULL,
    field_key text NOT NULL,
    output_alias text NOT NULL,
    system_field text NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    support_status character varying(32) DEFAULT 'supported'::character varying NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT northbound_field_mappings_domain_check CHECK (((domain)::text = ANY (ARRAY[('CM'::character varying)::text, ('PM'::character varying)::text, ('MR'::character varying)::text, ('LOG'::character varying)::text, ('INVENTORY'::character varying)::text]))),
    CONSTRAINT northbound_field_mappings_profile_kind_check CHECK (((profile_kind)::text = ANY (ARRAY[('file'::character varying)::text, ('inventory'::character varying)::text, ('socket'::character varying)::text, ('snmp'::character varying)::text, ('api'::character varying)::text]))),
    CONSTRAINT northbound_field_mappings_support_status_check CHECK (((support_status)::text = ANY (ARRAY[('supported'::character varying)::text, ('partial'::character varying)::text, ('unsupported'::character varying)::text])))
);


--
-- Name: northbound_file_profiles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.northbound_file_profiles (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    code character varying(32) NOT NULL,
    name character varying(255) NOT NULL,
    vendor character varying(128) DEFAULT 'Baicells'::character varying NOT NULL,
    scenario_name character varying(128) DEFAULT ''::character varying NOT NULL,
    scenario_name_en character varying(128) DEFAULT ''::character varying NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    flags jsonb DEFAULT '[]'::jsonb NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    status character varying(32) DEFAULT 'terminated'::character varying NOT NULL,
    groups jsonb DEFAULT '[]'::jsonb NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT northbound_file_profiles_code_check CHECK (((code)::text ~ '^S[0-9]{4}$'::text)),
    CONSTRAINT northbound_file_profiles_status_check CHECK (((status)::text = ANY (ARRAY[('normal'::character varying)::text, ('terminated'::character varying)::text])))
);


--
-- Name: northbound_inventory_profiles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.northbound_inventory_profiles (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    code character varying(32) NOT NULL,
    name character varying(255) NOT NULL,
    object_code character varying(32) NOT NULL,
    tech character varying(32) DEFAULT ''::character varying NOT NULL,
    period character varying(16) DEFAULT '24H'::character varying NOT NULL,
    start_minute integer DEFAULT 5 NOT NULL,
    path_template text NOT NULL,
    file_name_template text NOT NULL,
    compression_enabled boolean DEFAULT true NOT NULL,
    compression_format character varying(16) DEFAULT 'zip'::character varying NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    status character varying(32) DEFAULT 'terminated'::character varying NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT northbound_inventory_profiles_code_check CHECK (((code)::text = ANY (ARRAY[('ENB'::character varying)::text, ('GNB'::character varying)::text, ('GSM'::character varying)::text, ('OMC'::character varying)::text]))),
    CONSTRAINT northbound_inventory_profiles_compression_format_check CHECK (((compression_format)::text = ANY (ARRAY[('zip'::character varying)::text, ('gz'::character varying)::text]))),
    CONSTRAINT northbound_inventory_profiles_period_check CHECK (((period)::text = ANY (ARRAY[('15M'::character varying)::text, ('60M'::character varying)::text, ('24H'::character varying)::text, ('7D'::character varying)::text, ('1MO'::character varying)::text]))),
    CONSTRAINT northbound_inventory_profiles_start_minute_check CHECK (((start_minute >= 0) AND (start_minute <= 59))),
    CONSTRAINT northbound_inventory_profiles_status_check CHECK (((status)::text = ANY (ARRAY[('normal'::character varying)::text, ('terminated'::character varying)::text])))
);


--
-- Name: northbound_file_runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.northbound_file_runs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    profile_kind character varying(32) NOT NULL,
    profile_code character varying(32) NOT NULL,
    group_id character varying(128) DEFAULT ''::character varying NOT NULL,
    domain character varying(32) NOT NULL,
    object_code character varying(32) DEFAULT ''::character varying NOT NULL,
    status character varying(32) DEFAULT 'running'::character varying NOT NULL,
    window_start timestamp with time zone,
    window_end timestamp with time zone,
    artifact_path text DEFAULT ''::text NOT NULL,
    artifact_name text DEFAULT ''::text NOT NULL,
    artifact_content text DEFAULT ''::text NOT NULL,
    artifact_size bigint DEFAULT 0 NOT NULL,
    row_count integer DEFAULT 0 NOT NULL,
    compression_enabled boolean DEFAULT false NOT NULL,
    compression_format character varying(16),
    error_message text DEFAULT ''::text NOT NULL,
    summary jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT northbound_file_runs_artifact_size_check CHECK ((artifact_size >= 0)),
    CONSTRAINT northbound_file_runs_profile_kind_check CHECK (((profile_kind)::text = ANY (ARRAY[('file'::character varying)::text, ('inventory'::character varying)::text]))),
    CONSTRAINT northbound_file_runs_row_count_check CHECK ((row_count >= 0)),
    CONSTRAINT northbound_file_runs_status_check CHECK (((status)::text = ANY (ARRAY[('running'::character varying)::text, ('success'::character varying)::text, ('failed'::character varying)::text, ('terminated'::character varying)::text])))
);


--
-- Name: northbound_delivery_targets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.northbound_delivery_targets (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    scope character varying(32) NOT NULL,
    owner_code character varying(64) DEFAULT ''::character varying NOT NULL,
    target_key character varying(128) NOT NULL,
    name character varying(200) NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    protocol character varying(8) DEFAULT 'FTP'::character varying NOT NULL,
    host character varying(255) DEFAULT ''::character varying NOT NULL,
    port integer DEFAULT 21 NOT NULL,
    username character varying(128) DEFAULT ''::character varying NOT NULL,
    credential_secret text DEFAULT ''::text NOT NULL,
    auth_mode character varying(32) DEFAULT 'PASSWORD'::character varying NOT NULL,
    remote_root text DEFAULT '/northupload'::text NOT NULL,
    retry_times integer DEFAULT 3 NOT NULL,
    timeout_seconds integer DEFAULT 30 NOT NULL,
    passive_mode boolean DEFAULT true NOT NULL,
    host_key_policy character varying(32) DEFAULT 'INSECURE'::character varying NOT NULL,
    host_key_fingerprint text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT northbound_delivery_targets_auth_mode_check CHECK (((auth_mode)::text = ANY (ARRAY[('PASSWORD'::character varying)::text, ('PRIVATE_KEY'::character varying)::text]))),
    CONSTRAINT northbound_delivery_targets_host_key_policy_check CHECK (((host_key_policy)::text = ANY (ARRAY[('INSECURE'::character varying)::text, ('FINGERPRINT'::character varying)::text]))),
    CONSTRAINT northbound_delivery_targets_port_check CHECK (((port >= 1) AND (port <= 65535))),
    CONSTRAINT northbound_delivery_targets_protocol_check CHECK (((protocol)::text = ANY (ARRAY[('FTP'::character varying)::text, ('SFTP'::character varying)::text]))),
    CONSTRAINT northbound_delivery_targets_retry_times_check CHECK (((retry_times >= 0) AND (retry_times <= 20))),
    CONSTRAINT northbound_delivery_targets_scope_check CHECK (((scope)::text = ANY (ARRAY[('file'::character varying)::text, ('inventory'::character varying)::text, ('socket'::character varying)::text]))),
    CONSTRAINT northbound_delivery_targets_timeout_seconds_check CHECK (((timeout_seconds >= 1) AND (timeout_seconds <= 300)))
);


--
-- Name: northbound_snmp_alarm_targets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.northbound_snmp_alarm_targets (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    target_key character varying(128) NOT NULL,
    name character varying(200) NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    version character varying(8) DEFAULT 'v2'::character varying NOT NULL,
    notification_type character varying(16) DEFAULT 'Trap'::character varying NOT NULL,
    listen_ip character varying(64) DEFAULT '0.0.0.0'::character varying NOT NULL,
    listen_port integer DEFAULT 161 NOT NULL,
    target_host character varying(255) DEFAULT ''::character varying NOT NULL,
    target_port integer DEFAULT 162 NOT NULL,
    community_secret text DEFAULT ''::text NOT NULL,
    security_name character varying(128) DEFAULT ''::character varying NOT NULL,
    auth_protocol character varying(16) DEFAULT ''::character varying NOT NULL,
    auth_secret text DEFAULT ''::text NOT NULL,
    priv_protocol character varying(16) DEFAULT ''::character varying NOT NULL,
    priv_secret text DEFAULT ''::text NOT NULL,
    clear_severity_policy character varying(64) DEFAULT '保留原级别'::character varying NOT NULL,
    mib_query_enabled boolean DEFAULT true NOT NULL,
    timeout_seconds integer DEFAULT 5 NOT NULL,
    retries integer DEFAULT 1 NOT NULL,
    mib_fields jsonb DEFAULT '[]'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT northbound_snmp_alarm_targets_notification_type_check CHECK (((notification_type)::text = ANY (ARRAY[('Trap'::character varying)::text, ('Inform'::character varying)::text]))),
    CONSTRAINT northbound_snmp_alarm_targets_port_check CHECK (((listen_port >= 1) AND (listen_port <= 65535) AND (target_port >= 1) AND (target_port <= 65535))),
    CONSTRAINT northbound_snmp_alarm_targets_retries_check CHECK (((retries >= 0) AND (retries <= 20))),
    CONSTRAINT northbound_snmp_alarm_targets_timeout_seconds_check CHECK (((timeout_seconds >= 1) AND (timeout_seconds <= 300))),
    CONSTRAINT northbound_snmp_alarm_targets_version_check CHECK (((version)::text = ANY (ARRAY[('v2'::character varying)::text, ('v3'::character varying)::text])))
);


--
-- Name: northbound_socket_alarm_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.northbound_socket_alarm_configs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    config_key character varying(128) NOT NULL,
    name character varying(200) NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    profile character varying(32) NOT NULL,
    mode character varying(32) DEFAULT 'server'::character varying NOT NULL,
    listen_ip character varying(64) DEFAULT '0.0.0.0'::character varying NOT NULL,
    listen_port integer DEFAULT 31232 NOT NULL,
    max_clients integer DEFAULT 20 NOT NULL,
    realtime_push_enabled boolean DEFAULT true NOT NULL,
    client_sync_enabled boolean DEFAULT true NOT NULL,
    heartbeat_seconds integer DEFAULT 60 NOT NULL,
    heartbeat_times integer DEFAULT 3 NOT NULL,
    idle_timeout_seconds integer DEFAULT 180 NOT NULL,
    accounts jsonb DEFAULT '[]'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT northbound_socket_alarm_configs_heartbeat_seconds_check CHECK (((heartbeat_seconds >= 5) AND (heartbeat_seconds <= 3600))),
    CONSTRAINT northbound_socket_alarm_configs_heartbeat_times_check CHECK (((heartbeat_times >= 1) AND (heartbeat_times <= 100))),
    CONSTRAINT northbound_socket_alarm_configs_idle_timeout_seconds_check CHECK (((idle_timeout_seconds >= 10) AND (idle_timeout_seconds <= 86400))),
    CONSTRAINT northbound_socket_alarm_configs_listen_port_check CHECK (((listen_port >= 1) AND (listen_port <= 65535))),
    CONSTRAINT northbound_socket_alarm_configs_max_clients_check CHECK (((max_clients >= 1) AND (max_clients <= 10000))),
    CONSTRAINT northbound_socket_alarm_configs_mode_check CHECK (((mode)::text = 'server'::text)),
    CONSTRAINT northbound_socket_alarm_configs_profile_check CHECK (((profile)::text = ANY (ARRAY[('CTCC'::character varying)::text, ('CUCC'::character varying)::text])))
);


--
-- Name: northbound_api_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.northbound_api_configs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    api_key character varying(128) NOT NULL,
    name character varying(200) NOT NULL,
    method character varying(16) NOT NULL,
    path character varying(256) NOT NULL,
    kind character varying(64) DEFAULT '业务复用'::character varying NOT NULL,
    data_type character varying(64) DEFAULT ''::character varying NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    old_system_supported boolean DEFAULT true NOT NULL,
    current_supported boolean DEFAULT true NOT NULL,
    source character varying(128) DEFAULT ''::character varying NOT NULL,
    response_contract jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT northbound_api_configs_method_check CHECK (((method)::text = ANY (ARRAY[('GET'::character varying)::text, ('POST'::character varying)::text, ('PUT'::character varying)::text, ('DELETE'::character varying)::text])))
);


--
-- Name: northbound_api_clients; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.northbound_api_clients (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    client_key character varying(128) NOT NULL,
    name character varying(200) NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    token_secret text DEFAULT ''::text NOT NULL,
    allowed_api_keys jsonb DEFAULT '[]'::jsonb NOT NULL,
    ip_whitelist jsonb DEFAULT '[]'::jsonb NOT NULL,
    expires_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: northbound_page_config_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.northbound_page_config_events (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    capability character varying(32) NOT NULL,
    owner_code character varying(128) DEFAULT ''::character varying NOT NULL,
    target_key character varying(128) DEFAULT ''::character varying NOT NULL,
    event_type character varying(64) DEFAULT 'status'::character varying NOT NULL,
    status character varying(32) DEFAULT 'success'::character varying NOT NULL,
    artifact_type character varying(32) DEFAULT 'message'::character varying NOT NULL,
    artifact_name text DEFAULT ''::text NOT NULL,
    artifact_path text DEFAULT ''::text NOT NULL,
    payload text DEFAULT ''::text NOT NULL,
    payload_content_type character varying(128) DEFAULT 'text/plain; charset=utf-8'::character varying NOT NULL,
    error_message text DEFAULT ''::text NOT NULL,
    summary jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT northbound_page_config_events_artifact_type_check CHECK (((artifact_type)::text = ANY (ARRAY[('file'::character varying)::text, ('message'::character varying)::text, ('json'::character varying)::text]))),
    CONSTRAINT northbound_page_config_events_capability_check CHECK (((capability)::text = ANY (ARRAY[('file'::character varying)::text, ('inventory'::character varying)::text, ('delivery'::character varying)::text, ('snmp'::character varying)::text, ('socket'::character varying)::text, ('api'::character varying)::text]))),
    CONSTRAINT northbound_page_config_events_status_check CHECK (((status)::text = ANY (ARRAY[('running'::character varying)::text, ('success'::character varying)::text, ('failed'::character varying)::text, ('terminated'::character varying)::text])))
);


--
-- Name: northbound_outbox; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.northbound_outbox (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    event_id text NOT NULL,
    subject text NOT NULL,
    payload jsonb NOT NULL,
    target_id text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    max_attempts integer DEFAULT 3 NOT NULL,
    last_error text,
    next_retry_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: northbound_servers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.northbound_servers (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    role character varying(16) NOT NULL,
    host character varying(255) NOT NULL,
    port integer NOT NULL,
    description character varying(255) DEFAULT ''::character varying NOT NULL,
    is_active boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT northbound_servers_port_check CHECK (((port > 0) AND (port <= 65535))),
    CONSTRAINT northbound_servers_role_check CHECK (((role)::text = ANY (ARRAY[('primary'::character varying)::text, ('standby'::character varying)::text])))
);


--
-- Name: notification_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notification_history (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    template_id uuid,
    channel character varying(32) NOT NULL,
    recipients text[] NOT NULL,
    subject text NOT NULL,
    body text NOT NULL,
    status character varying(32) NOT NULL,
    error_message text,
    alarm_id uuid,
    retry_count integer DEFAULT 0 NOT NULL,
    sent_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT notification_history_channel_check CHECK (((channel)::text = ANY (ARRAY[('email'::character varying)::text, ('sms'::character varying)::text, ('webhook'::character varying)::text]))),
    CONSTRAINT notification_history_status_check CHECK (((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('sent'::character varying)::text, ('failed'::character varying)::text, ('dead_letter'::character varying)::text])))
);


--
-- Name: notification_templates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notification_templates (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(128) NOT NULL,
    channel character varying(32) NOT NULL,
    language character varying(16) DEFAULT 'zh-CN'::character varying NOT NULL,
    subject text NOT NULL,
    body text NOT NULL,
    variables text[] DEFAULT '{}'::text[] NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT notification_templates_channel_check CHECK (((channel)::text = ANY (ARRAY[('email'::character varying)::text, ('sms'::character varying)::text, ('webhook'::character varying)::text])))
);


--
-- Name: notifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notifications (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id character varying(100) NOT NULL,
    type character varying(20) NOT NULL,
    priority character varying(10) DEFAULT 'normal'::character varying NOT NULL,
    title character varying(200) NOT NULL,
    content text,
    link character varying(500),
    sender character varying(100) DEFAULT 'system'::character varying,
    is_read boolean DEFAULT false NOT NULL,
    read_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    status character varying(20) DEFAULT 'completed'::character varying NOT NULL,
    dedup_key character varying(64)
);


--
-- Name: ops_audit_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ops_audit_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    op_type character varying(32) NOT NULL,
    target_type character varying(64) NOT NULL,
    target_id character varying(128) NOT NULL,
    operator_user_id uuid,
    operator_name character varying(128) DEFAULT ''::character varying NOT NULL,
    risk_level character varying(16) DEFAULT 'safe'::character varying NOT NULL,
    input jsonb,
    output_summary text,
    result character varying(16) DEFAULT 'success'::character varying NOT NULL,
    approver_user_id uuid,
    break_glass boolean DEFAULT false NOT NULL,
    client_ip inet,
    user_agent text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: ops_command_records; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ops_command_records (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    command_text text NOT NULL,
    device_sn character varying(64) NOT NULL,
    device_name character varying(200),
    operator character varying(100),
    execute_time timestamp with time zone DEFAULT now() NOT NULL,
    duration integer DEFAULT 0 NOT NULL,
    success boolean DEFAULT false NOT NULL,
    output text,
    error_message text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: ops_diagnostics; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ops_diagnostics (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64),
    diag_type character varying(32) NOT NULL,
    initiator character varying(16) DEFAULT 'omc'::character varying NOT NULL,
    request jsonb DEFAULT '{}'::jsonb NOT NULL,
    result jsonb,
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    started_at timestamp with time zone DEFAULT now() NOT NULL,
    completed_at timestamp with time zone,
    duration_ms integer,
    operator character varying(128),
    task_id uuid,
    file_path character varying(500),
    error_message text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT ops_diagnostics_initiator_check CHECK (((initiator)::text = ANY (ARRAY[('device'::character varying)::text, ('omc'::character varying)::text]))),
    CONSTRAINT ops_diagnostics_status_check CHECK (((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('running'::character varying)::text, ('complete'::character varying)::text, ('failed'::character varying)::text, ('timeout'::character varying)::text])))
);


--
-- Name: ops_downloads; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ops_downloads (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_sn character varying(64) NOT NULL,
    content_type character varying(32) NOT NULL,
    file_path character varying(500) DEFAULT ''::character varying NOT NULL,
    file_size bigint DEFAULT 0 NOT NULL,
    checksum character varying(64),
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    operator character varying(128),
    task_id uuid,
    expires_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT ops_downloads_status_check CHECK (((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('uploading'::character varying)::text, ('complete'::character varying)::text, ('failed'::character varying)::text, ('expired'::character varying)::text])))
);


--
-- Name: ops_maintenance_windows; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ops_maintenance_windows (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(200) NOT NULL,
    scope_type character varying(16) DEFAULT 'device'::character varying NOT NULL,
    scope_ids jsonb DEFAULT '[]'::jsonb NOT NULL,
    start_at timestamp with time zone NOT NULL,
    end_at timestamp with time zone NOT NULL,
    suppress_alarms boolean DEFAULT true NOT NULL,
    pause_provision boolean DEFAULT true NOT NULL,
    allow_dangerous boolean DEFAULT true NOT NULL,
    reason text,
    creator_user_id uuid,
    approver_user_id uuid,
    status character varying(16) DEFAULT 'planned'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT ops_maintenance_windows_scope_type_check CHECK (((scope_type)::text = ANY (ARRAY[('device'::character varying)::text, ('group'::character varying)::text, ('all'::character varying)::text]))),
    CONSTRAINT ops_maintenance_windows_status_check CHECK (((status)::text = ANY (ARRAY[('planned'::character varying)::text, ('approved'::character varying)::text, ('active'::character varying)::text, ('ended'::character varying)::text, ('cancelled'::character varying)::text])))
);


--
-- Name: ops_playbooks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ops_playbooks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    alarm_pattern jsonb DEFAULT '{}'::jsonb NOT NULL,
    recommended_templates jsonb DEFAULT '[]'::jsonb NOT NULL,
    title character varying(200) NOT NULL,
    docs text,
    tags jsonb DEFAULT '[]'::jsonb NOT NULL,
    success_rate numeric(5,2),
    use_count integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: ops_task_executions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ops_task_executions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_id uuid NOT NULL,
    device_sn character varying(64) NOT NULL,
    step_index integer NOT NULL,
    step_name character varying(200) NOT NULL,
    step_type character varying(32) NOT NULL,
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    duration_ms integer,
    request jsonb,
    response jsonb,
    error_message text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: ops_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ops_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_name character varying(200) NOT NULL,
    template_id uuid,
    device_sns jsonb DEFAULT '[]'::jsonb NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    current_step integer DEFAULT 0 NOT NULL,
    total_steps integer DEFAULT 0 NOT NULL,
    progress integer DEFAULT 0 NOT NULL,
    success_count integer DEFAULT 0 NOT NULL,
    fail_count integer DEFAULT 0 NOT NULL,
    total_count integer DEFAULT 0 NOT NULL,
    creator character varying(100),
    message text,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    template_snapshot jsonb,
    risk_level character varying(16) DEFAULT 'safe'::character varying NOT NULL,
    approval_state character varying(16) DEFAULT 'not_required'::character varying NOT NULL,
    approver_user_id uuid,
    approved_at timestamp with time zone,
    batch_config jsonb,
    failure_policy character varying(16) DEFAULT 'continue'::character varying NOT NULL,
    CONSTRAINT ops_tasks_approval_state_check CHECK (((approval_state)::text = ANY (ARRAY[('not_required'::character varying)::text, ('pending'::character varying)::text, ('approved'::character varying)::text, ('rejected'::character varying)::text]))),
    CONSTRAINT ops_tasks_failure_policy_check CHECK (((failure_policy)::text = ANY (ARRAY[('continue'::character varying)::text, ('abort'::character varying)::text, ('retry'::character varying)::text, ('rollback'::character varying)::text]))),
    CONSTRAINT ops_tasks_risk_level_check CHECK (((risk_level)::text = ANY (ARRAY[('safe'::character varying)::text, ('cautious'::character varying)::text, ('dangerous'::character varying)::text])))
);


--
-- Name: ops_templates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ops_templates (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    template_name character varying(200) NOT NULL,
    description text,
    category character varying(50),
    target_device_types jsonb DEFAULT '[]'::jsonb NOT NULL,
    steps jsonb DEFAULT '[]'::jsonb NOT NULL,
    estimated_duration integer DEFAULT 0 NOT NULL,
    creator character varying(100),
    use_count integer DEFAULT 0 NOT NULL,
    tags jsonb DEFAULT '[]'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    risk_level character varying(16) DEFAULT 'safe'::character varying NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    change_log jsonb DEFAULT '[]'::jsonb NOT NULL,
    owner_user_id uuid,
    rollback_steps jsonb,
    target_carriers jsonb DEFAULT '[]'::jsonb NOT NULL,
    CONSTRAINT ops_templates_risk_level_check CHECK (((risk_level)::text = ANY (ARRAY[('safe'::character varying)::text, ('cautious'::character varying)::text, ('dangerous'::character varying)::text])))
);


--
-- Name: param_mappings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.param_mappings (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    param_model_id uuid NOT NULL,
    standard_path text NOT NULL,
    private_path text NOT NULL,
    entry_type character varying(16) NOT NULL,
    access character varying(16),
    data_type character varying(16),
    change_applies character varying(16),
    min_value bigint,
    max_value bigint,
    is_storable boolean DEFAULT true NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    is_supported boolean DEFAULT true NOT NULL,
    enum_values text,
    enum_labels text,
    mirror_with character varying(256),
    source character varying(16) DEFAULT 'builtin'::character varying NOT NULL,
    CONSTRAINT param_mappings_entry_type_check CHECK (((entry_type)::text = ANY (ARRAY[('object'::character varying)::text, ('parameter'::character varying)::text]))),
    CONSTRAINT param_mappings_source_check CHECK (((source)::text = ANY (ARRAY[('builtin'::character varying)::text, ('custom'::character varying)::text])))
);


--
-- Name: TABLE param_mappings; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.param_mappings IS 'T-0098 默认映射（设计 §1.2.2）；~4781 行典型规模，每条 standardPath ↔ privatePath 双向映射';


--
-- Name: COLUMN param_mappings.entry_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.param_mappings.entry_type IS 'object（容器对象）或 parameter（叶子参数）';


--
-- Name: COLUMN param_mappings.is_storable; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.param_mappings.is_storable IS '是否纳入 OMC 自动同步；XML store="false" → false，缺省 true（Path B sync 据此过滤）';


--
-- Name: COLUMN param_mappings.is_supported; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.param_mappings.is_supported IS 'T-0103 XML supported="false" → false；path-b sync 据此过滤，默认 TRUE';


--
-- Name: param_models; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.param_models (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(64) NOT NULL,
    total_entries integer DEFAULT 0 NOT NULL,
    total_objects integer DEFAULT 0 NOT NULL,
    total_params integer DEFAULT 0 NOT NULL,
    description text,
    is_active boolean DEFAULT true NOT NULL,
    loaded_from character varying(256),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE param_models; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.param_models IS 'T-0098 参数模型主表（设计 §1.2.1）；9 行典型规模，由 9 个 XML 文件载入';


--
-- Name: COLUMN param_models.name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.param_models.name IS '模型名（唯一），如 "BLQ" / "MLN" / "BaiBNQ"';


--
-- Name: COLUMN param_models.loaded_from; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.param_models.loaded_from IS 'XML 源文件名（如 BLQ.xml），溯源用';


--
-- Name: parameter_discovery_log; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.parameter_discovery_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_id uuid NOT NULL,
    device_sn text NOT NULL,
    oui text NOT NULL,
    product_class text,
    firmware_version text,
    parameter_count integer DEFAULT 0 NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    error_message text,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    param_model_id uuid,
    CONSTRAINT parameter_discovery_log_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'discovering'::text, 'syncing'::text, 'completed'::text, 'failed'::text])))
);


--
-- Name: COLUMN parameter_discovery_log.param_model_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.parameter_discovery_log.param_model_id IS 'T-0098 参数模型解析结果；与 data_model_id 共存（旧通路）直至 P5 清理';


--
-- Name: parameter_sync_admission_reservations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.parameter_sync_admission_reservations (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    request_id uuid NOT NULL,
    admission_class character varying(32) NOT NULL,
    bucket_id smallint NOT NULL,
    reserved_runs integer DEFAULT 0 NOT NULL,
    reserved_tasks integer DEFAULT 0 NOT NULL,
    status character varying(16) DEFAULT 'reserved'::character varying NOT NULL,
    lease_until timestamp with time zone DEFAULT now() NOT NULL,
    released_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT parameter_sync_admission_reservation_counts_chk CHECK (((reserved_runs >= 0) AND (reserved_tasks >= 0))),
    CONSTRAINT parameter_sync_admission_reservation_status_chk CHECK (((status)::text = ANY ((ARRAY['reserved'::character varying, 'released'::character varying, 'expired'::character varying])::text[])))
);


--
-- Name: parameter_sync_admission_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.parameter_sync_admission_state (
    admission_class character varying(32) NOT NULL,
    bucket_id smallint NOT NULL,
    active_run_limit integer DEFAULT 0 NOT NULL,
    active_task_limit integer DEFAULT 0 NOT NULL,
    missing_result_limit integer DEFAULT 0 NOT NULL,
    create_rate_per_minute integer DEFAULT 0 NOT NULL,
    reserved_runs integer DEFAULT 0 NOT NULL,
    reserved_tasks integer DEFAULT 0 NOT NULL,
    version bigint DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT parameter_sync_admission_bucket_chk CHECK ((bucket_id >= 0)),
    CONSTRAINT parameter_sync_admission_class_chk CHECK (((admission_class)::text = ANY ((ARRAY['global'::character varying, 'model_upload'::character varying, 'periodic'::character varying, 'manual'::character varying])::text[]))),
    CONSTRAINT parameter_sync_admission_counts_chk CHECK (((active_run_limit >= 0) AND (active_task_limit >= 0) AND (missing_result_limit >= 0) AND (create_rate_per_minute >= 0) AND (reserved_runs >= 0) AND (reserved_tasks >= 0)))
);


--
-- Name: parameter_sync_device_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.parameter_sync_device_state (
    device_id uuid NOT NULL,
    consecutive_failures integer DEFAULT 0 NOT NULL,
    last_attempt_at timestamp with time zone,
    last_success_at timestamp with time zone,
    last_failure_at timestamp with time zone,
    next_auto_sync_at timestamp with time zone,
    last_error text,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT parameter_sync_device_state_consecutive_failures_check CHECK ((consecutive_failures >= 0))
);


--
-- Name: parameter_sync_event_failures; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.parameter_sync_event_failures (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    subject character varying(255) NOT NULL,
    event_id character varying(128) NOT NULL,
    device_id uuid,
    device_sn character varying(64),
    request_id uuid,
    run_id uuid,
    task_id uuid,
    raw_payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    delivery_count integer DEFAULT 0 NOT NULL,
    status character varying(24) DEFAULT 'pending'::character varying NOT NULL,
    last_error text,
    next_retry_at timestamp with time zone,
    replayed_at timestamp with time zone,
    recovered_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT parameter_sync_event_failure_status_chk CHECK (((status)::text = ANY ((ARRAY['pending'::character varying, 'replayed'::character varying, 'recovered'::character varying, 'manual_review'::character varying])::text[])))
);


--
-- Name: parameter_sync_outbox; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.parameter_sync_outbox (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    event_type character varying(64) NOT NULL,
    aggregate_type character varying(32) NOT NULL,
    aggregate_id uuid NOT NULL,
    dedupe_key character varying(192) NOT NULL,
    payload jsonb NOT NULL,
    status character varying(24) DEFAULT 'pending'::character varying NOT NULL,
    attempt_count integer DEFAULT 0 NOT NULL,
    next_attempt_at timestamp with time zone DEFAULT now() NOT NULL,
    last_error text,
    delivered_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT parameter_sync_outbox_attempt_chk CHECK ((attempt_count >= 0)),
    CONSTRAINT parameter_sync_outbox_status_chk CHECK (((status)::text = ANY ((ARRAY['pending'::character varying, 'delivering'::character varying, 'delivered'::character varying, 'failed'::character varying, 'dead'::character varying])::text[])))
);


--
-- Name: parameter_sync_recovery_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.parameter_sync_recovery_state (
    run_id uuid NOT NULL,
    task_id uuid NOT NULL,
    status character varying(24) DEFAULT 'pending'::character varying NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    next_retry_at timestamp with time zone DEFAULT now() NOT NULL,
    lease_token uuid,
    lease_until timestamp with time zone,
    last_error text,
    claimed_at timestamp with time zone,
    processed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT parameter_sync_recovery_status_chk CHECK (((status)::text = ANY ((ARRAY['pending'::character varying, 'processing'::character varying, 'processed'::character varying, 'failed'::character varying, 'manual_review'::character varying])::text[])))
);


--
-- Name: parameter_sync_request_bindings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.parameter_sync_request_bindings (
    request_id uuid NOT NULL,
    run_id uuid NOT NULL,
    provisioning_task_id uuid NOT NULL,
    status character varying(24) DEFAULT 'waiting'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    completed_at timestamp with time zone,
    CONSTRAINT parameter_sync_bindings_status_chk CHECK (((status)::text = ANY ((ARRAY['waiting'::character varying, 'completed'::character varying, 'failed'::character varying, 'cancelled'::character varying])::text[])))
);


--
-- Name: parameter_sync_requests; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.parameter_sync_requests (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_id uuid NOT NULL,
    device_sn character varying(64) NOT NULL,
    caller_type character varying(32) DEFAULT 'system'::character varying NOT NULL,
    trigger_reason character varying(32) NOT NULL,
    sync_scope character varying(24) NOT NULL,
    requested_paths jsonb DEFAULT '[]'::jsonb NOT NULL,
    status character varying(24) DEFAULT 'accepted'::character varying NOT NULL,
    run_id uuid,
    active_run_id uuid,
    priority integer DEFAULT 10 NOT NULL,
    next_attempt_at timestamp with time zone DEFAULT now() NOT NULL,
    deadline_at timestamp with time zone,
    idempotency_key character varying(160),
    result_code character varying(64),
    result_summary jsonb,
    error_message text,
    campaign_id uuid,
    source_event_id character varying(128),
    origin_event_type character varying(64),
    model_upload_intent_id uuid,
    model_upload_status character varying(24),
    admission_class character varying(32),
    admission_reason text,
    admission_snapshot jsonb,
    admission_queued_at timestamp with time zone,
    deduplicated_to_request_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT parameter_sync_requests_scope_chk CHECK (((sync_scope)::text = ANY ((ARRAY['full'::character varying, 'partial'::character varying, 'readback'::character varying, 'policy_probe'::character varying])::text[]))),
    CONSTRAINT parameter_sync_requests_status_chk CHECK (((status)::text = ANY ((ARRAY['accepted'::character varying, 'queued'::character varying, 'running'::character varying, 'succeeded'::character varying, 'failed'::character varying, 'timed_out'::character varying, 'cancelled'::character varying, 'deduplicated'::character varying, 'rejected'::character varying])::text[]))),
    CONSTRAINT parameter_sync_requests_trigger_reason_chk CHECK (((trigger_reason)::text = ANY ((ARRAY['bootstrap'::character varying, 'model_upload'::character varying, 'device_online'::character varying, 'firmware_changed'::character varying, 'periodic'::character varying, 'manual'::character varying, 'config_pull'::character varying, 'license'::character varying, 'spv_readback'::character varying, 'add_object_readback'::character varying, 'inform_period_probe'::character varying])::text[])))
);


--
-- Name: parameter_sync_runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.parameter_sync_runs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    request_id uuid NOT NULL,
    device_id uuid NOT NULL,
    device_sn character varying(64) NOT NULL,
    trigger_reason character varying(32) NOT NULL,
    sync_scope character varying(24) NOT NULL,
    mapping_source character varying(128),
    mapping_version character varying(128),
    coverage jsonb DEFAULT '[]'::jsonb NOT NULL,
    status character varying(24) DEFAULT 'planning'::character varying NOT NULL,
    expected_task_count integer DEFAULT 0 NOT NULL,
    terminal_task_count integer DEFAULT 0 NOT NULL,
    processed_task_count integer DEFAULT 0 NOT NULL,
    failed_task_count integer DEFAULT 0 NOT NULL,
    error_message text,
    started_at timestamp with time zone DEFAULT now() NOT NULL,
    completed_at timestamp with time zone,
    version bigint DEFAULT 0 NOT NULL,
    projection_status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    projection_attempts integer DEFAULT 0 NOT NULL,
    projection_error text,
    projection_completed_at timestamp with time zone,
    projection_lease_token uuid,
    projection_lease_until timestamp with time zone,
    projection_next_attempt_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT parameter_sync_runs_counts_chk CHECK (((expected_task_count >= 0) AND (terminal_task_count >= 0) AND (processed_task_count >= 0) AND (failed_task_count >= 0) AND (terminal_task_count <= expected_task_count) AND (processed_task_count <= terminal_task_count) AND (failed_task_count <= processed_task_count))),
    CONSTRAINT parameter_sync_runs_scope_chk CHECK (((sync_scope)::text = ANY ((ARRAY['full'::character varying, 'partial'::character varying, 'readback'::character varying, 'policy_probe'::character varying])::text[]))),
    CONSTRAINT parameter_sync_runs_status_chk CHECK (((status)::text = ANY ((ARRAY['planning'::character varying, 'enqueuing'::character varying, 'waiting_device'::character varying, 'executing'::character varying, 'processing'::character varying, 'cancelling'::character varying, 'succeeded'::character varying, 'failed'::character varying, 'cancelled'::character varying])::text[]))),
    CONSTRAINT parameter_sync_runs_trigger_reason_chk CHECK (((trigger_reason)::text = ANY ((ARRAY['bootstrap'::character varying, 'model_upload'::character varying, 'device_online'::character varying, 'firmware_changed'::character varying, 'periodic'::character varying, 'manual'::character varying, 'config_pull'::character varying, 'license'::character varying, 'spv_readback'::character varying, 'add_object_readback'::character varying, 'inform_period_probe'::character varying])::text[])))
);


--
-- Name: parameter_sync_staging_values; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.parameter_sync_staging_values (
    run_id uuid NOT NULL,
    parameter_path text NOT NULL,
    private_path text NOT NULL,
    value jsonb,
    value_type character varying(64),
    writable boolean DEFAULT false NOT NULL,
    fap_instance integer DEFAULT 0 NOT NULL,
    param_group character varying(32) DEFAULT 'other'::character varying NOT NULL,
    coverage_scope text,
    task_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: parameter_sync_task_results; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.parameter_sync_task_results (
    run_id uuid NOT NULL,
    task_id uuid NOT NULL,
    event_id character varying(128) NOT NULL,
    success boolean NOT NULL,
    result_ref text,
    status character varying(24) DEFAULT 'received'::character varying NOT NULL,
    error_code character varying(64),
    error_message text,
    processed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT parameter_sync_task_results_status_chk CHECK (((status)::text = ANY ((ARRAY['received'::character varying, 'processed'::character varying, 'failed'::character varying])::text[])))
);


--
-- Name: perf_alarm_threshold; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.perf_alarm_threshold (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    temp_id character varying(32),
    indicator_id character varying(20),
    comparison character varying(8),
    threshold_value character varying(20),
    comparison2 character varying(8),
    threshold_value2 character varying(20),
    operation character varying(50),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: perf_cust_name; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.perf_cust_name (
    operator_code character varying(100) NOT NULL,
    perf_id character varying(20) NOT NULL,
    cust_name character varying(200),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: perf_indicators_enb; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.perf_indicators_enb (
    id character varying(20) NOT NULL,
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
    report_key character varying(256)
);


--
-- Name: COLUMN perf_indicators_enb.loaded_from; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.perf_indicators_enb.loaded_from IS 'T-0180 Loader 来源 XML 相对路径(含前缀,如 "indicator-library/enb/ALL.xml" 或 "indicator-library-custom/enb/MY.xml");NULL 表示历史数据未回填,Reload 后会自动写入';


--
-- Name: perf_indicators_gnb; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.perf_indicators_gnb (
    id character varying(20) NOT NULL,
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
    report_key character varying(256)
);


--
-- Name: COLUMN perf_indicators_gnb.loaded_from; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.perf_indicators_gnb.loaded_from IS 'T-0180 Loader 来源 XML 相对路径(含前缀);NULL 表示历史数据未回填,Reload 后会自动写入';


--
-- Name: perf_indicators_gsm; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.perf_indicators_gsm (
    id character varying(20) NOT NULL,
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
    report_key character varying(256)
);


--
-- Name: COLUMN perf_indicators_gsm.loaded_from; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.perf_indicators_gsm.loaded_from IS 'T-0180 Loader 来源 XML 相对路径(含前缀);NULL 表示历史数据未回填,Reload 后会自动写入';


--
-- Name: perf_template_rel_arithmetic; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.perf_template_rel_arithmetic (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    temp_id character varying(32) NOT NULL,
    indicator_id character varying(20) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: pm_adhoc_task_runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pm_adhoc_task_runs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_id uuid NOT NULL,
    run_seq integer NOT NULL,
    granularity text DEFAULT ''::text NOT NULL,
    dimension text DEFAULT ''::text NOT NULL,
    window_start timestamp with time zone,
    window_end timestamp with time zone,
    status text NOT NULL,
    queued_at timestamp with time zone,
    started_at timestamp with time zone DEFAULT now() NOT NULL,
    finished_at timestamp with time zone,
    error text DEFAULT ''::text NOT NULL,
    rows_total integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: pm_completion_watermarks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pm_completion_watermarks (
    granularity text NOT NULL,
    level text NOT NULL,
    completed_bucket_start timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT pm_completion_watermarks_granularity_check CHECK ((granularity = ANY (ARRAY['hourly'::text, 'daily'::text, 'weekly'::text, 'monthly'::text]))),
    CONSTRAINT pm_completion_watermarks_level_check CHECK ((level = ANY (ARRAY['device'::text, 'group'::text])))
);


--
-- Name: TABLE pm_completion_watermarks; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.pm_completion_watermarks IS '#528 PM 持续聚合完成水位：按 (粒度, 层级) 记录上游卷数据已成功处理完到哪一格起点（语义为「已处理」非「有数据」，空格也推进；只进不退）。';


--
-- Name: COLUMN pm_completion_watermarks.granularity; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.pm_completion_watermarks.granularity IS '聚合粒度：hourly/daily/weekly/monthly。';


--
-- Name: COLUMN pm_completion_watermarks.level; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.pm_completion_watermarks.level IS '完成层级：device（设备级）/ group（设备组级，链式产出，完成更晚）。';


--
-- Name: COLUMN pm_completion_watermarks.completed_bucket_start; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.pm_completion_watermarks.completed_bucket_start IS '已处理完成到的格起点（含），桶头时间戳；下游取「≤ 此值」的下一格。';


--
-- Name: pm_dashboards; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pm_dashboards (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    description text,
    owner_id uuid NOT NULL,
    shared_with uuid[] DEFAULT ARRAY[]::uuid[] NOT NULL,
    parent_dashboard_id uuid,
    technology text NOT NULL,
    layout jsonb DEFAULT '{"panels": []}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    is_builtin boolean DEFAULT false NOT NULL,
    CONSTRAINT pm_dashboards_technology_check CHECK ((technology = ANY (ARRAY['lte'::text, 'nr'::text, 'gsm'::text])))
);


--
-- Name: pm_kpi_export_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pm_kpi_export_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_name text DEFAULT ''::text NOT NULL,
    source_type text NOT NULL,
    params jsonb DEFAULT '{}'::jsonb NOT NULL,
    format text DEFAULT 'csv'::text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    row_count bigint DEFAULT 0 NOT NULL,
    bucket text DEFAULT ''::text NOT NULL,
    file_path text DEFAULT ''::text NOT NULL,
    file_size bigint DEFAULT 0 NOT NULL,
    error text DEFAULT ''::text NOT NULL,
    create_user text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    expire_at timestamp with time zone,
    CONSTRAINT pm_kpi_export_tasks_source_type_chk CHECK ((source_type = ANY (ARRAY['dashboard'::text, 'device_view'::text, 'kpi_query'::text, 'pm_dashboard'::text, 'adhoc_result'::text]))),
    CONSTRAINT pm_kpi_export_tasks_status_chk CHECK ((status = ANY (ARRAY['pending'::text, 'running'::text, 'succeeded'::text, 'failed'::text])))
);


--
-- Name: pm_panels; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pm_panels (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    dashboard_id uuid NOT NULL,
    panel_type text NOT NULL,
    title text NOT NULL,
    metric_paths text[] NOT NULL,
    dimension text NOT NULL,
    device_sns text[],
    device_group_ids uuid[],
    time_range jsonb DEFAULT '{}'::jsonb NOT NULL,
    compare_mode text,
    adhoc_task_id uuid,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    granularities text[] NOT NULL,
    CONSTRAINT pm_panels_compare_mode_check CHECK (((compare_mode IS NULL) OR (compare_mode = ANY (ARRAY['same_window_other_devices'::text, 'previous_window'::text])))),
    CONSTRAINT pm_panels_dimension_check CHECK ((dimension = ANY (ARRAY['device'::text, 'device_group'::text]))),
    CONSTRAINT pm_panels_granularities_check CHECK (((granularities <@ ARRAY['15min'::text, 'hourly'::text, 'daily'::text, 'weekly'::text, 'monthly'::text]) AND (array_length(granularities, 1) >= 1))),
    CONSTRAINT pm_panels_panel_type_check CHECK ((panel_type = ANY (ARRAY['kpi_card'::text, 'line_chart'::text, 'bar_chart'::text, 'table'::text, 'gauge'::text, 'topn'::text, 'big_number'::text, 'adhoc_result'::text])))
);


--
-- Name: pm_query_templates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pm_query_templates (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(128) NOT NULL,
    visibility character varying(16) NOT NULL,
    creator_id uuid NOT NULL,
    description text,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT pm_query_templates_visibility_check CHECK (((visibility)::text = ANY (ARRAY[('public'::character varying)::text, ('private'::character varying)::text])))
);


--
-- Name: TABLE pm_query_templates; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.pm_query_templates IS 'T-0174 指标查询页可复用查询模板；payload 为前端表单完整序列化状态';


--
-- Name: COLUMN pm_query_templates.visibility; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.pm_query_templates.visibility IS 'public（公共模板，super_admin 可写） / private（私有模板，仅 creator 可写）';


--
-- Name: COLUMN pm_query_templates.creator_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.pm_query_templates.creator_id IS '逻辑关联 admin_users.id；不建 FK（admin_users 分区表）';


--
-- Name: COLUMN pm_query_templates.payload; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.pm_query_templates.payload IS 'JSONB 表单序列化：{ device_sns, metric_paths, granularity, time_range_preset, custom_start, custom_end, ... }';


--
-- Name: pm_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pm_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_name character varying(200) NOT NULL,
    task_type character varying(30) DEFAULT 'extraction'::character varying NOT NULL,
    device_sns jsonb DEFAULT '[]'::jsonb NOT NULL,
    kpi_codes jsonb DEFAULT '[]'::jsonb NOT NULL,
    granularity character varying(10) DEFAULT '15min'::character varying NOT NULL,
    time_range jsonb,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    progress integer DEFAULT 0 NOT NULL,
    creator character varying(100),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    task_subtype text,
    mode text,
    cron_expr text,
    metric_paths text[],
    granularities text[],
    window_start timestamp with time zone,
    window_end timestamp with time zone,
    last_fire_at timestamp with time zone,
    dimension text DEFAULT 'device'::text NOT NULL,
    technology text,
    is_builtin boolean DEFAULT false NOT NULL,
    expire_days integer DEFAULT 60 NOT NULL,
    object_ldns text[],
    visibility character varying(16) DEFAULT 'private'::character varying NOT NULL,
    CONSTRAINT chk_pm_tasks_continuous_cron CHECK (((mode IS DISTINCT FROM 'continuous'::text) OR (cron_expr IS NOT NULL))),
    CONSTRAINT chk_pm_tasks_dimension CHECK ((dimension = ANY (ARRAY['device'::text, 'aggregate_group'::text, 'device_group'::text, 'product'::text, 'band'::text, 'network'::text]))),
    CONSTRAINT chk_pm_tasks_mode CHECK (((mode IS NULL) OR (mode = ANY (ARRAY['oneshot'::text, 'continuous'::text])))),
    CONSTRAINT chk_pm_tasks_technology CHECK (((technology IS NULL) OR (technology = ANY (ARRAY['lte'::text, 'nr'::text, 'gsm'::text])))),
    CONSTRAINT chk_pm_tasks_visibility CHECK (((visibility)::text = ANY (ARRAY[('private'::character varying)::text, ('public'::character varying)::text])))
);


--
-- Name: COLUMN pm_tasks.last_fire_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.pm_tasks.last_fire_at IS 'G7 continuous 任务上次 cron 触发时刻；ContinuousScheduler 写入，worker 不动。';


--
-- Name: COLUMN pm_tasks.visibility; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.pm_tasks.visibility IS 'PM adhoc 自定义聚合任务可见性：private=仅创建者/超管可见可操作；public=登录用户可见可操作。旧任务默认 private。';


--
-- Name: pm_user_dashboard_preferences; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pm_user_dashboard_preferences (
    user_id uuid NOT NULL,
    kpi_card_layout jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    technology text NOT NULL,
    current_dashboard_id uuid,
    shared_filters jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT pm_user_dashboard_preferences_technology_check CHECK ((technology = ANY (ARRAY['lte'::text, 'nr'::text, 'gsm'::text])))
);


--
-- Name: product_class_patterns; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.product_class_patterns (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid NOT NULL,
    product_class text NOT NULL,
    sort_order integer NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    source character varying(16) DEFAULT 'builtin'::character varying NOT NULL,
    CONSTRAINT product_class_patterns_source_check CHECK (((source)::text = ANY (ARRAY[('builtin'::character varying)::text, ('custom'::character varying)::text])))
);


--
-- Name: TABLE product_class_patterns; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.product_class_patterns IS 'productClass → product 路由正则；运行时按 sort_order 全局升序匹配，首命中返回 product_id';


--
-- Name: COLUMN product_class_patterns.sort_order; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.product_class_patterns.sort_order IS '全局唯一 sort_order；FAP 兜底正则必须排在最末';


--
-- Name: product_unsupported_paths; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.product_unsupported_paths (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid NOT NULL,
    standard_path character varying(512) NOT NULL,
    read_unsupported boolean DEFAULT false NOT NULL,
    write_unsupported boolean DEFAULT false NOT NULL,
    last_fault_code integer,
    last_device_sn character varying(64),
    hit_count integer DEFAULT 1 NOT NULL,
    first_seen_at timestamp with time zone DEFAULT now() NOT NULL,
    last_seen_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    firmware_version character varying(128) DEFAULT ''::character varying NOT NULL
);


--
-- Name: products; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.products (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_name character varying(128) NOT NULL,
    vendor character varying(64),
    tech character varying(8),
    radio_modes character varying(64),
    description text,
    param_model_id uuid,
    indicator_device_type character varying(8) NOT NULL,
    indicator_platform character varying(32) NOT NULL,
    alarm_ne_type character varying(16) NOT NULL,
    enable_filetype11 boolean DEFAULT true NOT NULL,
    device_attrs_override jsonb DEFAULT '{}'::jsonb NOT NULL,
    enable_unknown_alarm boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    is_builtin boolean DEFAULT false NOT NULL,
    CONSTRAINT chk_products_indicator_device_type CHECK (((indicator_device_type)::text = ANY (ARRAY[('enb'::character varying)::text, ('gsm'::character varying)::text, ('gnb'::character varying)::text])))
);


--
-- Name: TABLE products; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.products IS 'T-0098 产品装配件主表（设计 §4.2.1）；15 行典型规模，三字典软引用 + 三策略字段';


--
-- Name: COLUMN products.param_model_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.products.param_model_id IS '参数模型软引用；硬 FK 由 P1-03 创建 param_models 后追加';


--
-- Name: COLUMN products.indicator_platform; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.products.indicator_platform IS 'KPI 平台名软引用，handler 校验存在于 platform_indicator_formulas_{deviceType}';


--
-- Name: COLUMN products.alarm_ne_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.products.alarm_ne_type IS '告警 ne_type 软引用，handler 校验存在于 alarm_definitions.ne_type';


--
-- Name: COLUMN products.enable_filetype11; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.products.enable_filetype11 IS '该产品 Bootstrap 时是否下发 Upload(FileType=11)';


--
-- Name: COLUMN products.device_attrs_override; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.products.device_attrs_override IS '交集时哪些元属性以设备上传值覆盖默认；data_type=true 业务禁止';


--
-- Name: COLUMN products.enable_unknown_alarm; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.products.enable_unknown_alarm IS 'identifier 不在告警库时是否 fallback 写活动告警表（is_unknown=true）';


--
-- Name: COLUMN products.is_builtin; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.products.is_builtin IS 'true=products.xml 装配加载的内置产品（禁止删除）；false=UI 新建的自定义产品。loader UPSERT 置 true，Create 默认 false。';


--
-- Name: provisioning_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.provisioning_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_id uuid NOT NULL,
    template_id uuid,
    status character varying(20) DEFAULT 'discovered'::character varying NOT NULL,
    current_step integer DEFAULT 0 NOT NULL,
    total_steps integer DEFAULT 0 NOT NULL,
    error_message text,
    retry_count integer DEFAULT 0 NOT NULL,
    max_retries integer DEFAULT 3 NOT NULL,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT provisioning_tasks_status_check CHECK (((status)::text = ANY (ARRAY[('discovered'::character varying)::text, ('identifying'::character varying)::text, ('matching'::character varying)::text, ('configuring'::character varying)::text, ('verifying'::character varying)::text, ('discovering'::character varying)::text, ('syncing'::character varying)::text, ('completed'::character varying)::text, ('failed'::character varying)::text])))
);


--
-- Name: rela_platform_indicator_formula_enb; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.rela_platform_indicator_formula_enb (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    platform_name character varying(64) NOT NULL,
    indicator_id character varying(20) NOT NULL,
    formula text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    loaded_from text
);


--
-- Name: rela_platform_indicator_formula_gnb; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.rela_platform_indicator_formula_gnb (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    platform_name character varying(64) NOT NULL,
    indicator_id character varying(20) NOT NULL,
    formula text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    loaded_from text
);


--
-- Name: rela_platform_indicator_formula_gsm; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.rela_platform_indicator_formula_gsm (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    platform_name character varying(64) NOT NULL,
    indicator_id character varying(20) NOT NULL,
    formula text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    loaded_from text
);


--
-- Name: report_definitions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.report_definitions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    report_name character varying(200) NOT NULL,
    report_type character varying(20) NOT NULL,
    description text,
    format jsonb DEFAULT '["pdf"]'::jsonb NOT NULL,
    period character varying(20) DEFAULT 'daily'::character varying NOT NULL,
    kpi_codes jsonb DEFAULT '[]'::jsonb,
    device_groups jsonb DEFAULT '[]'::jsonb,
    auto_generate boolean DEFAULT false NOT NULL,
    cron_expression character varying(100),
    status character varying(20) DEFAULT 'draft'::character varying NOT NULL,
    creator character varying(100),
    last_gen_time timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: report_records; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.report_records (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    report_definition_id uuid NOT NULL,
    report_name character varying(200) NOT NULL,
    period character varying(100),
    generate_time timestamp with time zone DEFAULT now() NOT NULL,
    file_size bigint DEFAULT 0 NOT NULL,
    download_url text,
    format character varying(10) DEFAULT 'pdf'::character varying NOT NULL,
    status character varying(20) DEFAULT 'generating'::character varying NOT NULL,
    minio_path text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: restore_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.restore_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_bucket character varying(64) NOT NULL,
    source_object_path text NOT NULL,
    target_device_sns jsonb DEFAULT '[]'::jsonb NOT NULL,
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    progress integer DEFAULT 0 NOT NULL,
    error_message text,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by character varying(64),
    task_seq bigint NOT NULL,
    task_name character varying(200),
    task_result smallint,
    operator_code character varying(8),
    create_user character varying(64),
    expected_hash character varying(128),
    hash_algo character varying(16),
    verified_hash character varying(128),
    verification_method character varying(24),
    verified_at timestamp with time zone,
    downloaded_at timestamp with time zone,
    source_version character varying(64),
    CONSTRAINT restore_tasks_progress_check CHECK (((progress >= 0) AND (progress <= 100))),
    CONSTRAINT restore_tasks_status_check CHECK (((status)::text = ANY (ARRAY['pending'::text, 'running'::text, 'downloaded'::text, 'completed'::text, 'failed'::text, 'cancelled'::text]))),
    CONSTRAINT restore_tasks_task_result_check CHECK (((task_result IS NULL) OR (task_result = ANY (ARRAY[1, 2]))))
);


--
-- Name: restore_tasks_task_seq_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.restore_tasks_task_seq_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: restore_tasks_task_seq_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.restore_tasks_task_seq_seq OWNED BY public.restore_tasks.task_seq;


--
-- Name: role_api_permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.role_api_permissions (
    role_id uuid NOT NULL,
    endpoint_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE role_api_permissions; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.role_api_permissions IS '角色-API端点权限关联表';


--
-- Name: COLUMN role_api_permissions.role_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.role_api_permissions.role_id IS '角色 ID';


--
-- Name: COLUMN role_api_permissions.endpoint_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.role_api_permissions.endpoint_id IS 'API 端点 ID';


--
-- Name: role_device_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.role_device_groups (
    role_id uuid NOT NULL,
    group_id uuid NOT NULL,
    network_types text[] DEFAULT '{}'::text[] NOT NULL
);


--
-- Name: TABLE role_device_groups; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.role_device_groups IS '角色数据权限：角色可见的设备组';


--
-- Name: COLUMN role_device_groups.network_types; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.role_device_groups.network_types IS 'v0.6 决议 B：DB 是每分组独立数组；当前前端 UI 简化为角色级统一字段（同步 SetRoleDeviceGroups 时所有行写入相同值）';


--
-- Name: role_inheritance; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.role_inheritance (
    parent_role_id uuid NOT NULL,
    child_role_id uuid NOT NULL,
    domain character varying(16) DEFAULT 'system'::character varying NOT NULL
);


--
-- Name: role_menus; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.role_menus (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    role_id uuid NOT NULL,
    menu_id uuid NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE role_menus; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.role_menus IS '角色-菜单关联：角色拥有哪些菜单，就拥有哪些权限';


--
-- Name: roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.roles (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(64) NOT NULL,
    description text,
    is_system boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    code character varying(64),
    created_by uuid,
    updated_by uuid
);


--
-- Name: COLUMN roles.code; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.roles.code IS '角色编码（程序化引用用，可选）';


--
-- Name: COLUMN roles.created_by; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.roles.created_by IS '创建者用户 ID；NULL 表示由 seed 写入（内置角色）';


--
-- Name: COLUMN roles.updated_by; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.roles.updated_by IS '最近一次修改者用户 ID；NULL 表示由 seed 写入或从未被人工修改过';


--
-- Name: runtime_log_collect_sub_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.runtime_log_collect_sub_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_id uuid NOT NULL,
    device_id uuid NOT NULL,
    firmware_id uuid,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    error_message text,
    retry_count integer DEFAULT 0 NOT NULL,
    max_retries integer DEFAULT 3 NOT NULL,
    device_sn character varying(64),
    ori_version character varying(64),
    dest_version character varying(64),
    command_key character varying(256),
    failure_reason text,
    pre_suspend_status character varying(20),
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_upgrade_sub_tasks_status CHECK (((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('downloading'::character varying)::text, ('uploading'::character varying)::text, ('rebooting'::character varying)::text, ('verifying'::character varying)::text, ('completed'::character varying)::text, ('failed'::character varying)::text, ('suspended'::character varying)::text, ('terminated'::character varying)::text])))
);


--
-- Name: runtime_log_collect_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.runtime_log_collect_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    firmware_id uuid,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    error_message text,
    retry_count integer DEFAULT 0 NOT NULL,
    max_retries integer DEFAULT 3 NOT NULL,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    task_name character varying(256),
    task_type smallint DEFAULT 1 NOT NULL,
    file_name character varying(256),
    file_md5 character varying(64),
    result character varying(20),
    product_class character varying(64),
    is_keep_config boolean DEFAULT true,
    create_status character varying(16) DEFAULT 'active'::character varying NOT NULL,
    create_user character varying(64) DEFAULT 'system'::character varying NOT NULL,
    total_count integer DEFAULT 0 NOT NULL,
    success_count integer DEFAULT 0 NOT NULL,
    fail_count integer DEFAULT 0 NOT NULL,
    max_concurrent integer DEFAULT 5,
    ended_at timestamp with time zone,
    strategy character varying(16) DEFAULT 'full'::character varying NOT NULL,
    canary_stages jsonb,
    current_stage integer DEFAULT 0 NOT NULL,
    stage_status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    stage_history jsonb DEFAULT '[]'::jsonb NOT NULL,
    auto_advance boolean DEFAULT false NOT NULL,
    auto_advance_minutes integer DEFAULT 0 NOT NULL,
    rollback_reason text,
    rollback_source character varying(32) DEFAULT 'manual'::character varying NOT NULL,
    rollback_target_firmware_id uuid,
    rollback_on_failure boolean DEFAULT false NOT NULL,
    download_file_type text DEFAULT ''::text NOT NULL,
    scheduled_at timestamp with time zone,
    CONSTRAINT chk_upgrade_tasks_create_status CHECK (((create_status)::text = ANY (ARRAY[('active'::character varying)::text, ('suspend'::character varying)::text, ('timing'::character varying)::text]))),
    CONSTRAINT chk_upgrade_tasks_result CHECK (((result IS NULL) OR ((result)::text = ANY (ARRAY[('success'::character varying)::text, ('partial'::character varying)::text, ('failed'::character varying)::text, ('terminated'::character varying)::text])))),
    CONSTRAINT chk_upgrade_tasks_status CHECK (((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('in_progress'::character varying)::text, ('suspended'::character varying)::text, ('ended'::character varying)::text]))),
    CONSTRAINT chk_upgrade_tasks_task_type CHECK ((task_type = ANY (ARRAY[1, 2, 4, 6, 8, 10]))),
    CONSTRAINT upgrade_tasks_auto_advance_minutes_check CHECK (((auto_advance_minutes >= 0) AND (auto_advance_minutes <= 1440))),
    CONSTRAINT upgrade_tasks_current_stage_check CHECK (((current_stage >= 0) AND (current_stage <= 100))),
    CONSTRAINT upgrade_tasks_rollback_source_check CHECK (((rollback_source)::text = ANY (ARRAY[('manual'::character varying)::text, ('canary_failure'::character varying)::text, ('compatibility'::character varying)::text, ('scheduled'::character varying)::text]))),
    CONSTRAINT upgrade_tasks_stage_status_check CHECK (((stage_status)::text = ANY (ARRAY[('pending'::character varying)::text, ('running'::character varying)::text, ('paused'::character varying)::text, ('aborted'::character varying)::text, ('completed'::character varying)::text]))),
    CONSTRAINT upgrade_tasks_strategy_check CHECK (((strategy)::text = ANY (ARRAY[('full'::character varying)::text, ('canary'::character varying)::text])))
);


--
-- Name: seed_menu_show_status_backups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.seed_menu_show_status_backups (
    migration_key character varying(128) NOT NULL,
    menu_id uuid NOT NULL,
    previous_show_status character varying(16) NOT NULL,
    captured_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_seed_menu_show_status_backups_status CHECK (((previous_show_status)::text = ANY (ARRAY[('show'::character varying)::text, ('hide'::character varying)::text])))
);


--
-- Name: sites; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sites (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(200) NOT NULL,
    domain_id uuid,
    address text,
    longitude double precision,
    latitude double precision,
    device_count integer DEFAULT 0 NOT NULL,
    status character varying(20) DEFAULT 'active'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: standard_commands; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.standard_commands (
    id bigint NOT NULL,
    version_code character varying(50) NOT NULL,
    group_code text NOT NULL,
    group_name text NOT NULL,
    object_path text NOT NULL,
    command_name text NOT NULL,
    has_rw_params boolean DEFAULT false NOT NULL,
    has_instance boolean DEFAULT false NOT NULL,
    is_creatable boolean DEFAULT true NOT NULL,
    param_count integer DEFAULT 0 NOT NULL,
    rw_param_count integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE standard_commands; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.standard_commands IS '南向数据模型标准命令权威表，按 version_code 存储 MML 命令树的命令元数据';


--
-- Name: COLUMN standard_commands.has_rw_params; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.standard_commands.has_rw_params IS '为 true 时才生成 MOD 操作叶子';


--
-- Name: COLUMN standard_commands.is_creatable; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.standard_commands.is_creatable IS '为 false 表示在非可创建白名单中，不生成 ADD/RMV';


--
-- Name: standard_commands_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.standard_commands_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: standard_commands_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.standard_commands_id_seq OWNED BY public.standard_commands.id;


--
-- Name: standard_params; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.standard_params (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    standard_path text NOT NULL,
    entry_type character varying(16) NOT NULL,
    access character varying(16),
    data_type character varying(16),
    change_applies character varying(16),
    min_value bigint,
    max_value bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    updated_fields text[] DEFAULT '{}'::text[] NOT NULL,
    CONSTRAINT standard_params_entry_type_check CHECK (((entry_type)::text = ANY (ARRAY[('object'::character varying)::text, ('parameter'::character varying)::text])))
);


--
-- Name: TABLE standard_params; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.standard_params IS 'T-0098 标准参数树（设计 §1.2.4）；2001 行典型规模，OMC 统一标准 path 字典';


--
-- Name: COLUMN standard_params.description; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.standard_params.description IS 'TR-181 path 的中文含义说明（来自规范 JSON seed 的 params[].name 字段；前端 MML 控制台 path 行 tooltip / 行内提示用）';


--
-- Name: COLUMN standard_params.updated_fields; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.standard_params.updated_fields IS '最近一次人工编辑发生变化的字段名；新增及尚未人工编辑的记录为空数组';


--
-- Name: station_fault_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.station_fault_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_id uuid,
    device_sn text NOT NULL,
    file_name text,
    object_path text,
    bucket text,
    file_size bigint DEFAULT 0 NOT NULL,
    fault_reason text,
    fault_detail text,
    task_id uuid,
    is_deleted boolean DEFAULT false NOT NULL,
    collected_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    device_name text,
    device_type text,
    is_gnb boolean DEFAULT false NOT NULL,
    operate_ip text,
    software_version text,
    runtime_before_reboot bigint,
    record_status text DEFAULT 'detected'::text NOT NULL,
    collection_fail_reason text,
    manual_collection_status text DEFAULT '0'::text NOT NULL,
    CONSTRAINT station_fault_logs_manual_collection_status_check CHECK ((manual_collection_status = ANY (ARRAY['0'::text, '1'::text, '2'::text]))),
    CONSTRAINT station_fault_logs_record_status_check CHECK ((record_status = ANY (ARRAY['detected'::text, 'file_received'::text, 'collection_failed'::text])))
);


--
-- Name: station_running_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.station_running_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_id uuid,
    device_sn text NOT NULL,
    file_name text NOT NULL,
    object_path text NOT NULL,
    bucket text NOT NULL,
    file_size bigint DEFAULT 0 NOT NULL,
    task_id uuid,
    is_deleted boolean DEFAULT false NOT NULL,
    collected_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: sys_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sys_configs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    category character varying(64) NOT NULL,
    key character varying(128) NOT NULL,
    value text DEFAULT ''::text NOT NULL,
    value_type character varying(16) DEFAULT 'string'::character varying NOT NULL,
    description character varying(255) DEFAULT ''::character varying NOT NULL,
    is_public boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    description_i18n jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT chk_value_type CHECK (((value_type)::text = ANY (ARRAY[('string'::character varying)::text, ('int'::character varying)::text, ('float'::character varying)::text, ('bool'::character varying)::text, ('json'::character varying)::text])))
);


--
-- Name: TABLE sys_configs; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.sys_configs IS '系统配置参数表';


--
-- Name: config_apply_versions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.config_apply_versions (
    category character varying(64) NOT NULL,
    config_version bigint DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: config_apply_batches; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.config_apply_batches (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    category character varying(64) NOT NULL,
    config_version bigint NOT NULL,
    status character varying(16) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_config_apply_batch_status CHECK (((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('applying'::character varying)::text, ('applied'::character varying)::text, ('failed'::character varying)::text])))
);


--
-- Name: config_apply_targets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.config_apply_targets (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    batch_id uuid NOT NULL,
    category character varying(64) NOT NULL,
    target character varying(128) NOT NULL,
    status character varying(16) NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    applied_at timestamp with time zone,
    last_error text DEFAULT ''::text NOT NULL,
    expected_value jsonb DEFAULT '{}'::jsonb NOT NULL,
    actual_value jsonb DEFAULT '{}'::jsonb NOT NULL,
    lease_token uuid,
    lease_expires_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_config_apply_target_status CHECK (((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('applying'::character varying)::text, ('applied'::character varying)::text, ('failed'::character varying)::text])))
);


--
-- Name: sys_dictionaries; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sys_dictionaries (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    type character varying(255) NOT NULL,
    status boolean DEFAULT true NOT NULL,
    description character varying(255) DEFAULT ''::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    source_table character varying(64),
    source_label_field character varying(64),
    source_value_field character varying(64),
    source_filter jsonb,
    last_refresh_at timestamp with time zone,
    last_refresh_status character varying(16),
    last_refresh_error text,
    last_refresh_count integer,
    name_i18n jsonb DEFAULT '{}'::jsonb NOT NULL,
    description_i18n jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: TABLE sys_dictionaries; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.sys_dictionaries IS '字典主表：统一管理枚举值分类';


--
-- Name: COLUMN sys_dictionaries.type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.sys_dictionaries.type IS '字典类型（英文标识），全局唯一，如 gender、status';


--
-- Name: COLUMN sys_dictionaries.source_table; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.sys_dictionaries.source_table IS '数据源表(白名单业务名);NULL=手工字典';


--
-- Name: COLUMN sys_dictionaries.source_label_field; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.sys_dictionaries.source_label_field IS '源表中作为字典项 label 的字段(白名单内)';


--
-- Name: COLUMN sys_dictionaries.source_value_field; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.sys_dictionaries.source_value_field IS '源表中作为字典项 value 的字段(白名单内,可与 label 同字段)';


--
-- Name: COLUMN sys_dictionaries.source_filter; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.sys_dictionaries.source_filter IS 'v1 预留,JSONB 过滤条件;v1 不读不写';


--
-- Name: COLUMN sys_dictionaries.last_refresh_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.sys_dictionaries.last_refresh_at IS '上次同步成功/失败的时间';


--
-- Name: COLUMN sys_dictionaries.last_refresh_status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.sys_dictionaries.last_refresh_status IS '上次同步状态 ok|failed|running|timeout';


--
-- Name: COLUMN sys_dictionaries.last_refresh_error; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.sys_dictionaries.last_refresh_error IS '上次同步失败摘要(<=500 chars)';


--
-- Name: COLUMN sys_dictionaries.last_refresh_count; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.sys_dictionaries.last_refresh_count IS '上次同步完成后 origin=auto 项总数';


--
-- Name: sys_dictionaries_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.sys_dictionaries_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: sys_dictionaries_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.sys_dictionaries_id_seq OWNED BY public.sys_dictionaries.id;


--
-- Name: sys_dictionary_details; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sys_dictionary_details (
    id bigint NOT NULL,
    label character varying(255) NOT NULL,
    value character varying(255) NOT NULL,
    extend character varying(255) DEFAULT ''::character varying NOT NULL,
    status boolean DEFAULT true NOT NULL,
    sort integer DEFAULT 0 NOT NULL,
    sys_dictionary_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    parent_id bigint,
    level integer DEFAULT 0 NOT NULL,
    origin character varying(16) DEFAULT 'manual'::character varying NOT NULL,
    label_i18n jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: TABLE sys_dictionary_details; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.sys_dictionary_details IS '字典详情表：存储字典的具体选项值';


--
-- Name: COLUMN sys_dictionary_details.parent_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.sys_dictionary_details.parent_id IS '父明细 ID；NULL = 顶层；非 NULL = 子项（自引用 FK，删父级联子）';


--
-- Name: COLUMN sys_dictionary_details.level; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.sys_dictionary_details.level IS '层级冗余：0=顶层 / 1=一级子 / 2=二级子，最大深度 3 层（应用层校验）';


--
-- Name: COLUMN sys_dictionary_details.origin; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.sys_dictionary_details.origin IS '来源 manual=手工 / auto=数据源同步;同步任务只动 auto 行,manual 行保留';


--
-- Name: sys_dictionary_details_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.sys_dictionary_details_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: sys_dictionary_details_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.sys_dictionary_details_id_seq OWNED BY public.sys_dictionary_details.id;


--
-- Name: sys_login_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sys_login_logs (
    id bigint NOT NULL,
    user_id uuid,
    username character varying(64) NOT NULL,
    ip_address character varying(64) DEFAULT ''::character varying NOT NULL,
    location character varying(128) DEFAULT ''::character varying NOT NULL,
    browser character varying(128) DEFAULT ''::character varying NOT NULL,
    os character varying(128) DEFAULT ''::character varying NOT NULL,
    status boolean DEFAULT true NOT NULL,
    message character varying(255) DEFAULT ''::character varying NOT NULL,
    login_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE sys_login_logs; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.sys_login_logs IS '登录日志：记录用户登录/登出事件';


--
-- Name: sys_login_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.sys_login_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: sys_login_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.sys_login_logs_id_seq OWNED BY public.sys_login_logs.id;


--
-- Name: sys_oper_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sys_oper_logs (
    id bigint NOT NULL,
    user_id uuid,
    username character varying(64) DEFAULT ''::character varying NOT NULL,
    action character varying(64) NOT NULL,
    module character varying(64) DEFAULT ''::character varying NOT NULL,
    target character varying(255) DEFAULT ''::character varying NOT NULL,
    detail text DEFAULT ''::text NOT NULL,
    ip_address character varying(64) DEFAULT ''::character varying NOT NULL,
    user_agent character varying(512) DEFAULT ''::character varying NOT NULL,
    status boolean DEFAULT true NOT NULL,
    error_msg character varying(512) DEFAULT ''::character varying NOT NULL,
    cost_ms integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE sys_oper_logs; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.sys_oper_logs IS '操作日志：记录用户关键操作';


--
-- Name: sys_oper_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.sys_oper_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: sys_oper_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.sys_oper_logs_id_seq OWNED BY public.sys_oper_logs.id;


--
-- Name: sys_task_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sys_task_logs (
    id bigint NOT NULL,
    task_type character varying(64) NOT NULL,
    task_id character varying(128) DEFAULT ''::character varying NOT NULL,
    status character varying(16) DEFAULT 'running'::character varying NOT NULL,
    operator_id uuid,
    operator character varying(64) DEFAULT ''::character varying NOT NULL,
    target character varying(255) DEFAULT ''::character varying NOT NULL,
    detail text DEFAULT ''::text NOT NULL,
    error_msg text DEFAULT ''::text NOT NULL,
    started_at timestamp with time zone DEFAULT now() NOT NULL,
    finished_at timestamp with time zone,
    cost_ms integer DEFAULT 0 NOT NULL
);


--
-- Name: TABLE sys_task_logs; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.sys_task_logs IS '任务日志：记录后台任务执行结果';


--
-- Name: sys_task_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.sys_task_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: sys_task_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.sys_task_logs_id_seq OWNED BY public.sys_task_logs.id;


--
-- Name: system_license; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.system_license (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    license_id character varying(100) NOT NULL,
    license_type character varying(50) NOT NULL,
    issuer character varying(200),
    licensee character varying(200),
    issued_at timestamp with time zone NOT NULL,
    expiry_date timestamp with time zone,
    devices_support jsonb DEFAULT '{}'::jsonb NOT NULL,
    feature_list jsonb DEFAULT '{}'::jsonb NOT NULL,
    raw_content text NOT NULL,
    signature text,
    signature_key_id character varying(128),
    signature_status character varying(20) NOT NULL,
    uploaded_at timestamp with time zone DEFAULT now() NOT NULL,
    uploaded_by_user_id uuid,
    is_current boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_license_type CHECK (((license_type)::text = ANY (ARRAY[('Commercial'::character varying)::text, ('Trial'::character varying)::text, ('Evaluation'::character varying)::text, ('Internal'::character varying)::text]))),
    CONSTRAINT chk_signature_status CHECK (((signature_status)::text = ANY (ARRAY[('verified'::character varying)::text, ('unverified'::character varying)::text, ('invalid'::character varying)::text])))
);

-- system_license_usage：累计使用时长 + 时间回拨检测的 singleton 状态（复刻旧项目 check_au_info）。
-- 单行（id 固定为 1）；enforcer 在过期检查时 compute-on-read：按 now - last_visited_time 累加，
-- 若 now < last_visited_time 判定系统时间回拨 → license 失效。
CREATE TABLE public.system_license_usage (
    id integer DEFAULT 1 NOT NULL,
    use_duration_hours double precision DEFAULT 0 NOT NULL,
    last_visited_time timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT system_license_usage_pkey PRIMARY KEY (id),
    CONSTRAINT system_license_usage_singleton CHECK (id = 1)
);


--
-- Name: TABLE system_license; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.system_license IS 'Singleton system-wide license; 最多 1 行 is_current=true，由 partial unique 约束';


--
-- Name: COLUMN system_license.license_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.system_license.license_id IS '厂商签发的全局唯一 ID，例如 NO2022-03-14002';


--
-- Name: COLUMN system_license.devices_support; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.system_license.devices_support IS '设备类型 → 配额 map，例如 {"eNB":10000,"gNB":10000}';


--
-- Name: COLUMN system_license.feature_list; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.system_license.feature_list IS '三级嵌套功能矩阵，详见 PRD §4';


--
-- Name: COLUMN system_license.raw_content; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.system_license.raw_content IS '原始 license 文件全文，用于审计与重新验签';


--
-- Name: system_license_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.system_license_history (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    license_id character varying(100) NOT NULL,
    license_type character varying(50) NOT NULL,
    issuer character varying(200),
    licensee character varying(200),
    issued_at timestamp with time zone NOT NULL,
    expiry_date timestamp with time zone,
    devices_support jsonb NOT NULL,
    feature_list jsonb NOT NULL,
    raw_content text NOT NULL,
    signature text,
    signature_key_id character varying(128),
    signature_status character varying(20) NOT NULL,
    uploaded_at timestamp with time zone NOT NULL,
    uploaded_by_user_id uuid,
    replaced_at timestamp with time zone DEFAULT now() NOT NULL,
    replaced_by_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE system_license_history; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.system_license_history IS '每次 Update 把旧的 current license 拷贝进本表，留作合规审计';


--
-- Name: system_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.system_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    level character varying(16) NOT NULL,
    source character varying(128),
    message text NOT NULL,
    details jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: topo_edges; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.topo_edges (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_id uuid NOT NULL,
    target_id uuid NOT NULL,
    label character varying(100),
    status character varying(20) DEFAULT 'active'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: topo_nodes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.topo_nodes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    label character varying(200) NOT NULL,
    node_type character varying(20) NOT NULL,
    x double precision DEFAULT 0 NOT NULL,
    y double precision DEFAULT 0 NOT NULL,
    status character varying(20) DEFAULT 'online'::character varying NOT NULL,
    device_sn character varying(64),
    site_id uuid,
    domain_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: trace_export_jobs; Type: TABLE; Schema: public; Owner: -
--

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
    CONSTRAINT trace_export_jobs_status_check CHECK (((status)::text = ANY (ARRAY[('queued'::character varying)::text, ('running'::character varying)::text, ('done'::character varying)::text, ('failed'::character varying)::text])))
);


--
-- Name: trace_tasks; Type: TABLE; Schema: public; Owner: -
--

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
    CONSTRAINT trace_tasks_status_check CHECK (((status)::text = ANY (ARRAY[('running'::character varying)::text, ('stopped'::character varying)::text, ('purged'::character varying)::text])))
);


--
-- Name: ufte_task_types; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ufte_task_types (
    type_code text NOT NULL,
    category text NOT NULL,
    category_label text NOT NULL,
    display_name text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    rpc_type text NOT NULL,
    built_in boolean DEFAULT false NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    step_chain jsonb DEFAULT '[]'::jsonb NOT NULL,
    post_tc_event_code text DEFAULT ''::text NOT NULL,
    permission_code text NOT NULL,
    platform_scope jsonb DEFAULT '[]'::jsonb NOT NULL,
    file_type text DEFAULT ''::text NOT NULL,
    file_type_label text DEFAULT ''::text NOT NULL,
    file_type_editable boolean DEFAULT true NOT NULL,
    url_template text DEFAULT ''::text NOT NULL,
    target_file_name_template text DEFAULT ''::text NOT NULL,
    file_name_template text DEFAULT ''::text NOT NULL,
    file_size_field text DEFAULT ''::text NOT NULL,
    checksum_field text DEFAULT ''::text NOT NULL,
    raw_mode text DEFAULT ''::text NOT NULL,
    delay_seconds integer DEFAULT 0 NOT NULL,
    transport_path text DEFAULT ''::text NOT NULL,
    last_editor text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    firmware_file_type integer,
    sort_order integer DEFAULT 100 NOT NULL,
    product_scope jsonb DEFAULT '[]'::jsonb NOT NULL,
    CONSTRAINT ufte_task_types_rpc_type_check CHECK ((rpc_type = ANY (ARRAY['DOWNLOAD'::text, 'UPLOAD'::text, 'SET_PARAM_VALUES'::text])))
);


--
-- Name: COLUMN ufte_task_types.product_scope; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.ufte_task_types.product_scope IS '#492 适用产品：产品英文名列表（引用 products.product_name）。非空时设备匹配按产品名精确匹配，空则回退 platform_scope。';


--
-- Name: upgrade_sub_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.upgrade_sub_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_id uuid NOT NULL,
    device_id uuid NOT NULL,
    firmware_id uuid,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    error_message text,
    retry_count integer DEFAULT 0 NOT NULL,
    max_retries integer DEFAULT 3 NOT NULL,
    device_sn character varying(64),
    ori_version character varying(64),
    dest_version character varying(64),
    command_key character varying(256),
    failure_reason text,
    pre_suspend_status character varying(20),
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_upgrade_sub_tasks_status CHECK (((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('downloading'::character varying)::text, ('uploading'::character varying)::text, ('rebooting'::character varying)::text, ('verifying'::character varying)::text, ('completed'::character varying)::text, ('failed'::character varying)::text, ('suspended'::character varying)::text, ('terminated'::character varying)::text])))
);


--
-- Name: upgrade_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.upgrade_tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    firmware_id uuid,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    error_message text,
    retry_count integer DEFAULT 0 NOT NULL,
    max_retries integer DEFAULT 3 NOT NULL,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    task_name character varying(256),
    task_type smallint DEFAULT 1 NOT NULL,
    file_name character varying(256),
    file_md5 character varying(64),
    result character varying(20),
    product_class character varying(64),
    is_keep_config boolean DEFAULT true,
    create_status character varying(16) DEFAULT 'active'::character varying NOT NULL,
    create_user character varying(64) DEFAULT 'system'::character varying NOT NULL,
    total_count integer DEFAULT 0 NOT NULL,
    success_count integer DEFAULT 0 NOT NULL,
    fail_count integer DEFAULT 0 NOT NULL,
    max_concurrent integer DEFAULT 5,
    ended_at timestamp with time zone,
    strategy character varying(16) DEFAULT 'full'::character varying NOT NULL,
    canary_stages jsonb,
    current_stage integer DEFAULT 0 NOT NULL,
    stage_status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    stage_history jsonb DEFAULT '[]'::jsonb NOT NULL,
    auto_advance boolean DEFAULT false NOT NULL,
    auto_advance_minutes integer DEFAULT 0 NOT NULL,
    rollback_reason text,
    rollback_source character varying(32) DEFAULT 'manual'::character varying NOT NULL,
    rollback_target_firmware_id uuid,
    rollback_on_failure boolean DEFAULT false NOT NULL,
    download_file_type text DEFAULT ''::text NOT NULL,
    scheduled_at timestamp with time zone,
    CONSTRAINT chk_upgrade_tasks_create_status CHECK (((create_status)::text = ANY (ARRAY[('active'::character varying)::text, ('suspend'::character varying)::text, ('timing'::character varying)::text]))),
    CONSTRAINT chk_upgrade_tasks_result CHECK (((result IS NULL) OR ((result)::text = ANY (ARRAY[('success'::character varying)::text, ('partial'::character varying)::text, ('failed'::character varying)::text, ('terminated'::character varying)::text])))),
    CONSTRAINT chk_upgrade_tasks_status CHECK (((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('in_progress'::character varying)::text, ('suspended'::character varying)::text, ('ended'::character varying)::text]))),
    CONSTRAINT chk_upgrade_tasks_task_type CHECK ((task_type = ANY (ARRAY[1, 2, 4, 6, 8, 10]))),
    CONSTRAINT upgrade_tasks_auto_advance_minutes_check CHECK (((auto_advance_minutes >= 0) AND (auto_advance_minutes <= 1440))),
    CONSTRAINT upgrade_tasks_current_stage_check CHECK (((current_stage >= 0) AND (current_stage <= 100))),
    CONSTRAINT upgrade_tasks_rollback_source_check CHECK (((rollback_source)::text = ANY (ARRAY[('manual'::character varying)::text, ('canary_failure'::character varying)::text, ('compatibility'::character varying)::text, ('scheduled'::character varying)::text]))),
    CONSTRAINT upgrade_tasks_stage_status_check CHECK (((stage_status)::text = ANY (ARRAY[('pending'::character varying)::text, ('running'::character varying)::text, ('paused'::character varying)::text, ('aborted'::character varying)::text, ('completed'::character varying)::text]))),
    CONSTRAINT upgrade_tasks_strategy_check CHECK (((strategy)::text = ANY (ARRAY[('full'::character varying)::text, ('canary'::character varying)::text])))
);


--
-- Name: user_column_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_column_configs (
    user_id uuid NOT NULL,
    page_key character varying(64) NOT NULL,
    columns jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_roles (
    user_id uuid NOT NULL,
    role_id uuid NOT NULL,
    is_default boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    username character varying(64) NOT NULL,
    password_hash character varying(256) NOT NULL,
    display_name character varying(128),
    email character varying(256),
    status character varying(16) DEFAULT 'active'::character varying NOT NULL,
    last_login_at timestamp with time zone,
    failed_login_attempts integer DEFAULT 0 NOT NULL,
    locked_until timestamp with time zone,
    last_failed_login_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    phone character varying(32),
    source character varying(16) DEFAULT 'admin'::character varying NOT NULL,
    description text,
    expire_at timestamp with time zone,
    created_by uuid,
    updated_by uuid,
    must_change_password boolean DEFAULT false NOT NULL,
    password_changed_at timestamp with time zone,
    CONSTRAINT users_source_check CHECK (((source)::text = ANY (ARRAY[('builtIn'::character varying)::text, ('admin'::character varying)::text, ('LDAP'::character varying)::text])))
);


--
-- Name: COLUMN users.must_change_password; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.users.must_change_password IS 'P1 ①：true 表示用户下次登录必须先修改密码（FE 跳改密页）。CreateUser/ResetPassword 时由策略决定；ChangePassword 后清零';


--
-- Name: COLUMN users.password_changed_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.users.password_changed_at IS 'P1 ④：最近一次密码修改时间戳；NULL 视为初始未改。Login 时与 sys_configs.security.validPeriod 比较判断过期';


--
-- Name: device_parameters_p00; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p00 FOR VALUES WITH (modulus 32, remainder 0);


--
-- Name: device_parameters_p01; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p01 FOR VALUES WITH (modulus 32, remainder 1);


--
-- Name: device_parameters_p02; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p02 FOR VALUES WITH (modulus 32, remainder 2);


--
-- Name: device_parameters_p03; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p03 FOR VALUES WITH (modulus 32, remainder 3);


--
-- Name: device_parameters_p04; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p04 FOR VALUES WITH (modulus 32, remainder 4);


--
-- Name: device_parameters_p05; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p05 FOR VALUES WITH (modulus 32, remainder 5);


--
-- Name: device_parameters_p06; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p06 FOR VALUES WITH (modulus 32, remainder 6);


--
-- Name: device_parameters_p07; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p07 FOR VALUES WITH (modulus 32, remainder 7);


--
-- Name: device_parameters_p08; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p08 FOR VALUES WITH (modulus 32, remainder 8);


--
-- Name: device_parameters_p09; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p09 FOR VALUES WITH (modulus 32, remainder 9);


--
-- Name: device_parameters_p10; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p10 FOR VALUES WITH (modulus 32, remainder 10);


--
-- Name: device_parameters_p11; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p11 FOR VALUES WITH (modulus 32, remainder 11);


--
-- Name: device_parameters_p12; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p12 FOR VALUES WITH (modulus 32, remainder 12);


--
-- Name: device_parameters_p13; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p13 FOR VALUES WITH (modulus 32, remainder 13);


--
-- Name: device_parameters_p14; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p14 FOR VALUES WITH (modulus 32, remainder 14);


--
-- Name: device_parameters_p15; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p15 FOR VALUES WITH (modulus 32, remainder 15);


--
-- Name: device_parameters_p16; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p16 FOR VALUES WITH (modulus 32, remainder 16);


--
-- Name: device_parameters_p17; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p17 FOR VALUES WITH (modulus 32, remainder 17);


--
-- Name: device_parameters_p18; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p18 FOR VALUES WITH (modulus 32, remainder 18);


--
-- Name: device_parameters_p19; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p19 FOR VALUES WITH (modulus 32, remainder 19);


--
-- Name: device_parameters_p20; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p20 FOR VALUES WITH (modulus 32, remainder 20);


--
-- Name: device_parameters_p21; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p21 FOR VALUES WITH (modulus 32, remainder 21);


--
-- Name: device_parameters_p22; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p22 FOR VALUES WITH (modulus 32, remainder 22);


--
-- Name: device_parameters_p23; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p23 FOR VALUES WITH (modulus 32, remainder 23);


--
-- Name: device_parameters_p24; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p24 FOR VALUES WITH (modulus 32, remainder 24);


--
-- Name: device_parameters_p25; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p25 FOR VALUES WITH (modulus 32, remainder 25);


--
-- Name: device_parameters_p26; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p26 FOR VALUES WITH (modulus 32, remainder 26);


--
-- Name: device_parameters_p27; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p27 FOR VALUES WITH (modulus 32, remainder 27);


--
-- Name: device_parameters_p28; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p28 FOR VALUES WITH (modulus 32, remainder 28);


--
-- Name: device_parameters_p29; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p29 FOR VALUES WITH (modulus 32, remainder 29);


--
-- Name: device_parameters_p30; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p30 FOR VALUES WITH (modulus 32, remainder 30);


--
-- Name: device_parameters_p31; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters ATTACH PARTITION public.device_parameters_p31 FOR VALUES WITH (modulus 32, remainder 31);


--
-- Name: device_tasks_p00; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks ATTACH PARTITION public.device_tasks_p00 FOR VALUES WITH (modulus 16, remainder 0);


--
-- Name: device_tasks_p01; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks ATTACH PARTITION public.device_tasks_p01 FOR VALUES WITH (modulus 16, remainder 1);


--
-- Name: device_tasks_p02; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks ATTACH PARTITION public.device_tasks_p02 FOR VALUES WITH (modulus 16, remainder 2);


--
-- Name: device_tasks_p03; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks ATTACH PARTITION public.device_tasks_p03 FOR VALUES WITH (modulus 16, remainder 3);


--
-- Name: device_tasks_p04; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks ATTACH PARTITION public.device_tasks_p04 FOR VALUES WITH (modulus 16, remainder 4);


--
-- Name: device_tasks_p05; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks ATTACH PARTITION public.device_tasks_p05 FOR VALUES WITH (modulus 16, remainder 5);


--
-- Name: device_tasks_p06; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks ATTACH PARTITION public.device_tasks_p06 FOR VALUES WITH (modulus 16, remainder 6);


--
-- Name: device_tasks_p07; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks ATTACH PARTITION public.device_tasks_p07 FOR VALUES WITH (modulus 16, remainder 7);


--
-- Name: device_tasks_p08; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks ATTACH PARTITION public.device_tasks_p08 FOR VALUES WITH (modulus 16, remainder 8);


--
-- Name: device_tasks_p09; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks ATTACH PARTITION public.device_tasks_p09 FOR VALUES WITH (modulus 16, remainder 9);


--
-- Name: device_tasks_p10; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks ATTACH PARTITION public.device_tasks_p10 FOR VALUES WITH (modulus 16, remainder 10);


--
-- Name: device_tasks_p11; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks ATTACH PARTITION public.device_tasks_p11 FOR VALUES WITH (modulus 16, remainder 11);


--
-- Name: device_tasks_p12; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks ATTACH PARTITION public.device_tasks_p12 FOR VALUES WITH (modulus 16, remainder 12);


--
-- Name: device_tasks_p13; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks ATTACH PARTITION public.device_tasks_p13 FOR VALUES WITH (modulus 16, remainder 13);


--
-- Name: device_tasks_p14; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks ATTACH PARTITION public.device_tasks_p14 FOR VALUES WITH (modulus 16, remainder 14);


--
-- Name: device_tasks_p15; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks ATTACH PARTITION public.device_tasks_p15 FOR VALUES WITH (modulus 16, remainder 15);


--
-- Name: devices_cmcc; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.devices ATTACH PARTITION public.devices_cmcc FOR VALUES IN ('cmcc');


--
-- Name: devices_ctcc; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.devices ATTACH PARTITION public.devices_ctcc FOR VALUES IN ('ctcc');


--
-- Name: devices_cucc; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.devices ATTACH PARTITION public.devices_cucc FOR VALUES IN ('cucc');


--
-- Name: devices_other; Type: TABLE ATTACH; Schema: public; Owner: -
--

ALTER TABLE ONLY public.devices ATTACH PARTITION public.devices_other FOR VALUES IN ('intl');


--
-- Name: backup_restore_file id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.backup_restore_file ALTER COLUMN id SET DEFAULT nextval('public.backup_restore_file_id_seq'::regclass);


--
-- Name: backup_tasks task_seq; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.backup_tasks ALTER COLUMN task_seq SET DEFAULT nextval('public.backup_tasks_task_seq_seq'::regclass);


--
-- Name: restore_tasks task_seq; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.restore_tasks ALTER COLUMN task_seq SET DEFAULT nextval('public.restore_tasks_task_seq_seq'::regclass);


--
-- Name: standard_commands id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.standard_commands ALTER COLUMN id SET DEFAULT nextval('public.standard_commands_id_seq'::regclass);


--
-- Name: sys_dictionaries id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sys_dictionaries ALTER COLUMN id SET DEFAULT nextval('public.sys_dictionaries_id_seq'::regclass);


--
-- Name: sys_dictionary_details id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sys_dictionary_details ALTER COLUMN id SET DEFAULT nextval('public.sys_dictionary_details_id_seq'::regclass);


--
-- Name: sys_login_logs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sys_login_logs ALTER COLUMN id SET DEFAULT nextval('public.sys_login_logs_id_seq'::regclass);


--
-- Name: sys_oper_logs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sys_oper_logs ALTER COLUMN id SET DEFAULT nextval('public.sys_oper_logs_id_seq'::regclass);


--
-- Name: sys_task_logs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sys_task_logs ALTER COLUMN id SET DEFAULT nextval('public.sys_task_logs_id_seq'::regclass);


--
-- Name: alarm_definitions alarm_definitions_identifier_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alarm_definitions
    ADD CONSTRAINT alarm_definitions_identifier_key UNIQUE (identifier);


--
-- Name: alarm_definitions alarm_definitions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alarm_definitions
    ADD CONSTRAINT alarm_definitions_pkey PRIMARY KEY (id);


--
-- Name: alarm_filters alarm_filters_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alarm_filters
    ADD CONSTRAINT alarm_filters_pkey PRIMARY KEY (id);


--
-- Name: alarm_rules alarm_rules_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alarm_rules
    ADD CONSTRAINT alarm_rules_pkey PRIMARY KEY (id);


--
-- Name: alarm_severity_levels alarm_severity_levels_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alarm_severity_levels
    ADD CONSTRAINT alarm_severity_levels_code_key UNIQUE (code);


--
-- Name: alarm_severity_levels alarm_severity_levels_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alarm_severity_levels
    ADD CONSTRAINT alarm_severity_levels_name_key UNIQUE (name);


--
-- Name: alarm_severity_levels alarm_severity_levels_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alarm_severity_levels
    ADD CONSTRAINT alarm_severity_levels_pkey PRIMARY KEY (id);


--
-- Name: alarm_webhook_dead_letters alarm_webhook_dead_letters_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alarm_webhook_dead_letters
    ADD CONSTRAINT alarm_webhook_dead_letters_pkey PRIMARY KEY (id);


--
-- Name: alarms_active alarms_active_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alarms_active
    ADD CONSTRAINT alarms_active_pkey PRIMARY KEY (id);


--
-- Name: api_endpoints api_endpoints_path_method_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_endpoints
    ADD CONSTRAINT api_endpoints_path_method_key UNIQUE (path, method);


--
-- Name: api_endpoints api_endpoints_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_endpoints
    ADD CONSTRAINT api_endpoints_pkey PRIMARY KEY (id);


--
-- Name: api_keys api_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT api_keys_pkey PRIMARY KEY (id);


--
-- Name: async_jobs_cron_state async_jobs_cron_state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.async_jobs_cron_state
    ADD CONSTRAINT async_jobs_cron_state_pkey PRIMARY KEY (job_type);


--
-- Name: async_jobs async_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.async_jobs
    ADD CONSTRAINT async_jobs_pkey PRIMARY KEY (id);


--
-- Name: audit_logs audit_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_logs
    ADD CONSTRAINT audit_logs_pkey PRIMARY KEY (id);


--
-- Name: backup_policies backup_policies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.backup_policies
    ADD CONSTRAINT backup_policies_pkey PRIMARY KEY (id);


--
-- Name: backup_restore_file backup_restore_file_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.backup_restore_file
    ADD CONSTRAINT backup_restore_file_pkey PRIMARY KEY (id);


--
-- Name: backup_restore_file backup_restore_file_sn_task_file_uk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.backup_restore_file
    ADD CONSTRAINT backup_restore_file_sn_task_file_uk UNIQUE (serial_number, task_id, file_name);


--
-- Name: backup_schedules backup_schedules_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.backup_schedules
    ADD CONSTRAINT backup_schedules_pkey PRIMARY KEY (id);


--
-- Name: backup_tasks backup_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.backup_tasks
    ADD CONSTRAINT backup_tasks_pkey PRIMARY KEY (id);


--
-- Name: backup_tasks backup_tasks_task_seq_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.backup_tasks
    ADD CONSTRAINT backup_tasks_task_seq_key UNIQUE (task_seq);


--
-- Name: config_backup_sub_tasks config_backup_sub_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_backup_sub_tasks
    ADD CONSTRAINT config_backup_sub_tasks_pkey PRIMARY KEY (id);


--
-- Name: config_backup_tasks config_backup_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_backup_tasks
    ADD CONSTRAINT config_backup_tasks_pkey PRIMARY KEY (id);


--
-- Name: config_baselines config_baselines_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_baselines
    ADD CONSTRAINT config_baselines_pkey PRIMARY KEY (id);


--
-- Name: config_neighbors config_neighbors_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_neighbors
    ADD CONSTRAINT config_neighbors_pkey PRIMARY KEY (id);


--
-- Name: config_restore_sub_tasks config_restore_sub_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_restore_sub_tasks
    ADD CONSTRAINT config_restore_sub_tasks_pkey PRIMARY KEY (id);


--
-- Name: config_restore_tasks config_restore_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_restore_tasks
    ADD CONSTRAINT config_restore_tasks_pkey PRIMARY KEY (id);


--
-- Name: config_snapshots config_snapshots_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_snapshots
    ADD CONSTRAINT config_snapshots_pkey PRIMARY KEY (serial_number);


--
-- Name: config_tasks config_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_tasks
    ADD CONSTRAINT config_tasks_pkey PRIMARY KEY (id);


--
-- Name: config_templates config_templates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_templates
    ADD CONSTRAINT config_templates_pkey PRIMARY KEY (id);


--
-- Name: dashboard_kpi_layouts dashboard_kpi_layouts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dashboard_kpi_layouts
    ADD CONSTRAINT dashboard_kpi_layouts_pkey PRIMARY KEY (tech);


--
-- Name: dashboard_widgets dashboard_widgets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dashboard_widgets
    ADD CONSTRAINT dashboard_widgets_pkey PRIMARY KEY (id);


--
-- Name: dashboard_widgets dashboard_widgets_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dashboard_widgets
    ADD CONSTRAINT dashboard_widgets_user_id_key UNIQUE (user_id);


--
-- Name: dead_letters dead_letters_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dead_letters
    ADD CONSTRAINT dead_letters_pkey PRIMARY KEY (id);


--
-- Name: device_active_tasks device_active_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_active_tasks
    ADD CONSTRAINT device_active_tasks_pkey PRIMARY KEY (device_id);


--
-- Name: device_group_members device_group_members_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_group_members
    ADD CONSTRAINT device_group_members_pkey PRIMARY KEY (group_id, device_id);


--
-- Name: device_groups device_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_groups
    ADD CONSTRAINT device_groups_pkey PRIMARY KEY (id);


--
-- Name: device_info device_info_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_info
    ADD CONSTRAINT device_info_pkey PRIMARY KEY (device_id);


--
-- Name: device_licenses device_licenses_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_licenses
    ADD CONSTRAINT device_licenses_pkey PRIMARY KEY (serial_number);


--
-- Name: device_location_observations device_location_observations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_location_observations
    ADD CONSTRAINT device_location_observations_pkey PRIMARY KEY (device_id);


--
-- Name: device_parameters device_parameters_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters
    ADD CONSTRAINT device_parameters_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p00 device_parameters_p00_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p00
    ADD CONSTRAINT device_parameters_p00_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p01 device_parameters_p01_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p01
    ADD CONSTRAINT device_parameters_p01_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p02 device_parameters_p02_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p02
    ADD CONSTRAINT device_parameters_p02_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p03 device_parameters_p03_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p03
    ADD CONSTRAINT device_parameters_p03_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p04 device_parameters_p04_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p04
    ADD CONSTRAINT device_parameters_p04_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p05 device_parameters_p05_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p05
    ADD CONSTRAINT device_parameters_p05_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p06 device_parameters_p06_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p06
    ADD CONSTRAINT device_parameters_p06_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p07 device_parameters_p07_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p07
    ADD CONSTRAINT device_parameters_p07_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p08 device_parameters_p08_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p08
    ADD CONSTRAINT device_parameters_p08_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p09 device_parameters_p09_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p09
    ADD CONSTRAINT device_parameters_p09_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p10 device_parameters_p10_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p10
    ADD CONSTRAINT device_parameters_p10_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p11 device_parameters_p11_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p11
    ADD CONSTRAINT device_parameters_p11_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p12 device_parameters_p12_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p12
    ADD CONSTRAINT device_parameters_p12_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p13 device_parameters_p13_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p13
    ADD CONSTRAINT device_parameters_p13_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p14 device_parameters_p14_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p14
    ADD CONSTRAINT device_parameters_p14_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p15 device_parameters_p15_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p15
    ADD CONSTRAINT device_parameters_p15_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p16 device_parameters_p16_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p16
    ADD CONSTRAINT device_parameters_p16_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p17 device_parameters_p17_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p17
    ADD CONSTRAINT device_parameters_p17_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p18 device_parameters_p18_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p18
    ADD CONSTRAINT device_parameters_p18_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p19 device_parameters_p19_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p19
    ADD CONSTRAINT device_parameters_p19_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p20 device_parameters_p20_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p20
    ADD CONSTRAINT device_parameters_p20_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p21 device_parameters_p21_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p21
    ADD CONSTRAINT device_parameters_p21_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p22 device_parameters_p22_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p22
    ADD CONSTRAINT device_parameters_p22_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p23 device_parameters_p23_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p23
    ADD CONSTRAINT device_parameters_p23_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p24 device_parameters_p24_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p24
    ADD CONSTRAINT device_parameters_p24_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p25 device_parameters_p25_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p25
    ADD CONSTRAINT device_parameters_p25_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p26 device_parameters_p26_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p26
    ADD CONSTRAINT device_parameters_p26_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p27 device_parameters_p27_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p27
    ADD CONSTRAINT device_parameters_p27_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p28 device_parameters_p28_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p28
    ADD CONSTRAINT device_parameters_p28_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p29 device_parameters_p29_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p29
    ADD CONSTRAINT device_parameters_p29_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p30 device_parameters_p30_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p30
    ADD CONSTRAINT device_parameters_p30_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_parameters_p31 device_parameters_p31_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_parameters_p31
    ADD CONSTRAINT device_parameters_p31_pkey PRIMARY KEY (device_id, parameter_path);


--
-- Name: device_registrations device_registrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_registrations
    ADD CONSTRAINT device_registrations_pkey PRIMARY KEY (id);


--
-- Name: device_tasks device_tasks_pkey1; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks
    ADD CONSTRAINT device_tasks_pkey1 PRIMARY KEY (id, device_sn);


--
-- Name: device_tasks_p00 device_tasks_p00_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks_p00
    ADD CONSTRAINT device_tasks_p00_pkey PRIMARY KEY (id, device_sn);


--
-- Name: device_tasks_p01 device_tasks_p01_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks_p01
    ADD CONSTRAINT device_tasks_p01_pkey PRIMARY KEY (id, device_sn);


--
-- Name: device_tasks_p02 device_tasks_p02_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks_p02
    ADD CONSTRAINT device_tasks_p02_pkey PRIMARY KEY (id, device_sn);


--
-- Name: device_tasks_p03 device_tasks_p03_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks_p03
    ADD CONSTRAINT device_tasks_p03_pkey PRIMARY KEY (id, device_sn);


--
-- Name: device_tasks_p04 device_tasks_p04_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks_p04
    ADD CONSTRAINT device_tasks_p04_pkey PRIMARY KEY (id, device_sn);


--
-- Name: device_tasks_p05 device_tasks_p05_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks_p05
    ADD CONSTRAINT device_tasks_p05_pkey PRIMARY KEY (id, device_sn);


--
-- Name: device_tasks_p06 device_tasks_p06_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks_p06
    ADD CONSTRAINT device_tasks_p06_pkey PRIMARY KEY (id, device_sn);


--
-- Name: device_tasks_p07 device_tasks_p07_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks_p07
    ADD CONSTRAINT device_tasks_p07_pkey PRIMARY KEY (id, device_sn);


--
-- Name: device_tasks_p08 device_tasks_p08_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks_p08
    ADD CONSTRAINT device_tasks_p08_pkey PRIMARY KEY (id, device_sn);


--
-- Name: device_tasks_p09 device_tasks_p09_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks_p09
    ADD CONSTRAINT device_tasks_p09_pkey PRIMARY KEY (id, device_sn);


--
-- Name: device_tasks_p10 device_tasks_p10_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks_p10
    ADD CONSTRAINT device_tasks_p10_pkey PRIMARY KEY (id, device_sn);


--
-- Name: device_tasks_p11 device_tasks_p11_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks_p11
    ADD CONSTRAINT device_tasks_p11_pkey PRIMARY KEY (id, device_sn);


--
-- Name: device_tasks_p12 device_tasks_p12_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks_p12
    ADD CONSTRAINT device_tasks_p12_pkey PRIMARY KEY (id, device_sn);


--
-- Name: device_tasks_p13 device_tasks_p13_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks_p13
    ADD CONSTRAINT device_tasks_p13_pkey PRIMARY KEY (id, device_sn);


--
-- Name: device_tasks_p14 device_tasks_p14_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks_p14
    ADD CONSTRAINT device_tasks_p14_pkey PRIMARY KEY (id, device_sn);


--
-- Name: device_tasks_p15 device_tasks_p15_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tasks_p15
    ADD CONSTRAINT device_tasks_p15_pkey PRIMARY KEY (id, device_sn);


--
-- Name: devices devices_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.devices
    ADD CONSTRAINT devices_pkey PRIMARY KEY (id, carrier);


--
-- Name: devices_cmcc devices_cmcc_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.devices_cmcc
    ADD CONSTRAINT devices_cmcc_pkey PRIMARY KEY (id, carrier);


--
-- Name: devices_ctcc devices_ctcc_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.devices_ctcc
    ADD CONSTRAINT devices_ctcc_pkey PRIMARY KEY (id, carrier);


--
-- Name: devices_cucc devices_cucc_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.devices_cucc
    ADD CONSTRAINT devices_cucc_pkey PRIMARY KEY (id, carrier);


--
-- Name: devices_other devices_other_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.devices_other
    ADD CONSTRAINT devices_other_pkey PRIMARY KEY (id, carrier);


--
-- Name: discovered_param_mappings discovered_param_mappings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discovered_param_mappings
    ADD CONSTRAINT discovered_param_mappings_pkey PRIMARY KEY (id);


--
-- Name: event_logs event_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_logs
    ADD CONSTRAINT event_logs_pkey PRIMARY KEY (id);


--
-- Name: fault_log_collect_sub_tasks fault_log_collect_sub_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.fault_log_collect_sub_tasks
    ADD CONSTRAINT fault_log_collect_sub_tasks_pkey PRIMARY KEY (id);


--
-- Name: fault_log_collect_tasks fault_log_collect_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.fault_log_collect_tasks
    ADD CONSTRAINT fault_log_collect_tasks_pkey PRIMARY KEY (id);


--
-- Name: firmware_versions firmware_versions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.firmware_versions
    ADD CONSTRAINT firmware_versions_pkey PRIMARY KEY (id);


--
-- Name: ftp_configs ftp_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ftp_configs
    ADD CONSTRAINT ftp_configs_pkey PRIMARY KEY (id);


--
-- Name: indicator_file_descriptions indicator_file_descriptions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.indicator_file_descriptions
    ADD CONSTRAINT indicator_file_descriptions_pkey PRIMARY KEY (tech, platform);


--
-- Name: kpi_definitions kpi_definitions_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kpi_definitions
    ADD CONSTRAINT kpi_definitions_name_key UNIQUE (name);


--
-- Name: kpi_definitions kpi_definitions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kpi_definitions
    ADD CONSTRAINT kpi_definitions_pkey PRIMARY KEY (id);


--
-- Name: kpi_thresholds kpi_thresholds_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kpi_thresholds
    ADD CONSTRAINT kpi_thresholds_pkey PRIMARY KEY (id);


--
-- Name: managed_files managed_files_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.managed_files
    ADD CONSTRAINT managed_files_pkey PRIMARY KEY (id);


--
-- Name: menus menus_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.menus
    ADD CONSTRAINT menus_pkey PRIMARY KEY (id);


--
-- Name: mml_audit_log mml_audit_log_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_audit_log
    ADD CONSTRAINT mml_audit_log_pkey PRIMARY KEY (id);


--
-- Name: mml_catalog_link_health mml_catalog_link_health_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_catalog_link_health
    ADD CONSTRAINT mml_catalog_link_health_pkey PRIMARY KEY (id);


--
-- Name: mml_catalog_orphan_paths_audit_t0171 mml_catalog_orphan_paths_audit_t0171_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_catalog_orphan_paths_audit_t0171
    ADD CONSTRAINT mml_catalog_orphan_paths_audit_t0171_pkey PRIMARY KEY (id);


--
-- Name: mml_catalog_orphan_paths_audit_t0171 mml_catalog_orphan_paths_audit_t0171_standard_path_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_catalog_orphan_paths_audit_t0171
    ADD CONSTRAINT mml_catalog_orphan_paths_audit_t0171_standard_path_key UNIQUE (standard_path);


--
-- Name: mml_command_groups mml_command_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_command_groups
    ADD CONSTRAINT mml_command_groups_pkey PRIMARY KEY (id);


--
-- Name: mml_command_sub_field_overrides mml_command_sub_field_overrides_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_command_sub_field_overrides
    ADD CONSTRAINT mml_command_sub_field_overrides_pkey PRIMARY KEY (id);


--
-- Name: mml_command_sub_fields mml_command_sub_fields_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_command_sub_fields
    ADD CONSTRAINT mml_command_sub_fields_pkey PRIMARY KEY (id);


--
-- Name: mml_commands mml_commands_command_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_commands
    ADD CONSTRAINT mml_commands_command_code_key UNIQUE (command_code);


--
-- Name: mml_commands mml_commands_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_commands
    ADD CONSTRAINT mml_commands_pkey PRIMARY KEY (id);


--
-- Name: mml_custom_command_paths mml_custom_command_paths_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_custom_command_paths
    ADD CONSTRAINT mml_custom_command_paths_pkey PRIMARY KEY (id);


--
-- Name: mml_param_versions mml_param_versions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_param_versions
    ADD CONSTRAINT mml_param_versions_pkey PRIMARY KEY (id);


--
-- Name: mml_scripts mml_scripts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_scripts
    ADD CONSTRAINT mml_scripts_pkey PRIMARY KEY (id);


--
-- Name: mml_tasks mml_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_tasks
    ADD CONSTRAINT mml_tasks_pkey PRIMARY KEY (id);


--
-- Name: mml_custom_command mml_templates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_custom_command
    ADD CONSTRAINT mml_templates_pkey PRIMARY KEY (id);


--
-- Name: model_upload_intents model_upload_intents_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.model_upload_intents
    ADD CONSTRAINT model_upload_intents_pkey PRIMARY KEY (id);


--
-- Name: mr_customize_task mr_customize_task_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mr_customize_task
    ADD CONSTRAINT mr_customize_task_pkey PRIMARY KEY (task_id);


--
-- Name: mr_customize_task_progress mr_customize_task_progress_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mr_customize_task_progress
    ADD CONSTRAINT mr_customize_task_progress_pkey PRIMARY KEY (id);


--
-- Name: mr_device_mappings mr_device_mappings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mr_device_mappings
    ADD CONSTRAINT mr_device_mappings_pkey PRIMARY KEY (id);


--
-- Name: mr_indicators mr_indicators_indicator_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mr_indicators
    ADD CONSTRAINT mr_indicators_indicator_code_key UNIQUE (indicator_code);


--
-- Name: mr_indicators mr_indicators_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mr_indicators
    ADD CONSTRAINT mr_indicators_pkey PRIMARY KEY (id);


--
-- Name: ne_message_logs ne_message_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ne_message_logs
    ADD CONSTRAINT ne_message_logs_pkey PRIMARY KEY (id);


--
-- Name: nedirect_commands nedirect_commands_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.nedirect_commands
    ADD CONSTRAINT nedirect_commands_pkey PRIMARY KEY (id);


--
-- Name: nedirect_sessions nedirect_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.nedirect_sessions
    ADD CONSTRAINT nedirect_sessions_pkey PRIMARY KEY (id);


--
-- Name: northbound_endpoints northbound_endpoints_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_endpoints
    ADD CONSTRAINT northbound_endpoints_pkey PRIMARY KEY (id);


--
-- Name: northbound_field_mappings northbound_field_mappings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_field_mappings
    ADD CONSTRAINT northbound_field_mappings_pkey PRIMARY KEY (id);


--
-- Name: northbound_file_profiles northbound_file_profiles_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_file_profiles
    ADD CONSTRAINT northbound_file_profiles_code_key UNIQUE (code);


--
-- Name: northbound_file_profiles northbound_file_profiles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_file_profiles
    ADD CONSTRAINT northbound_file_profiles_pkey PRIMARY KEY (id);


--
-- Name: northbound_file_runs northbound_file_runs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_file_runs
    ADD CONSTRAINT northbound_file_runs_pkey PRIMARY KEY (id);


--
-- Name: northbound_delivery_targets northbound_delivery_targets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_delivery_targets
    ADD CONSTRAINT northbound_delivery_targets_pkey PRIMARY KEY (id);


--
-- Name: northbound_delivery_targets northbound_delivery_targets_scope_owner_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_delivery_targets
    ADD CONSTRAINT northbound_delivery_targets_scope_owner_key_key UNIQUE (scope, owner_code, target_key);


--
-- Name: northbound_snmp_alarm_targets northbound_snmp_alarm_targets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_snmp_alarm_targets
    ADD CONSTRAINT northbound_snmp_alarm_targets_pkey PRIMARY KEY (id);


--
-- Name: northbound_snmp_alarm_targets northbound_snmp_alarm_targets_target_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_snmp_alarm_targets
    ADD CONSTRAINT northbound_snmp_alarm_targets_target_key_key UNIQUE (target_key);


--
-- Name: northbound_socket_alarm_configs northbound_socket_alarm_configs_config_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_socket_alarm_configs
    ADD CONSTRAINT northbound_socket_alarm_configs_config_key_key UNIQUE (config_key);


--
-- Name: northbound_socket_alarm_configs northbound_socket_alarm_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_socket_alarm_configs
    ADD CONSTRAINT northbound_socket_alarm_configs_pkey PRIMARY KEY (id);


--
-- Name: northbound_api_configs northbound_api_configs_api_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_api_configs
    ADD CONSTRAINT northbound_api_configs_api_key_key UNIQUE (api_key);


--
-- Name: northbound_api_configs northbound_api_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_api_configs
    ADD CONSTRAINT northbound_api_configs_pkey PRIMARY KEY (id);


--
-- Name: northbound_api_clients northbound_api_clients_client_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_api_clients
    ADD CONSTRAINT northbound_api_clients_client_key_key UNIQUE (client_key);


--
-- Name: northbound_api_clients northbound_api_clients_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_api_clients
    ADD CONSTRAINT northbound_api_clients_pkey PRIMARY KEY (id);


--
-- Name: northbound_page_config_events northbound_page_config_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_page_config_events
    ADD CONSTRAINT northbound_page_config_events_pkey PRIMARY KEY (id);


--
-- Name: northbound_inventory_profiles northbound_inventory_profiles_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_inventory_profiles
    ADD CONSTRAINT northbound_inventory_profiles_code_key UNIQUE (code);


--
-- Name: northbound_inventory_profiles northbound_inventory_profiles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_inventory_profiles
    ADD CONSTRAINT northbound_inventory_profiles_pkey PRIMARY KEY (id);


--
-- Name: northbound_outbox northbound_outbox_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_outbox
    ADD CONSTRAINT northbound_outbox_pkey PRIMARY KEY (id);


--
-- Name: northbound_servers northbound_servers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_servers
    ADD CONSTRAINT northbound_servers_pkey PRIMARY KEY (id);


--
-- Name: northbound_servers northbound_servers_role_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.northbound_servers
    ADD CONSTRAINT northbound_servers_role_key UNIQUE (role);


--
-- Name: notification_history notification_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification_history
    ADD CONSTRAINT notification_history_pkey PRIMARY KEY (id);


--
-- Name: notification_templates notification_templates_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification_templates
    ADD CONSTRAINT notification_templates_name_key UNIQUE (name);


--
-- Name: notification_templates notification_templates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification_templates
    ADD CONSTRAINT notification_templates_pkey PRIMARY KEY (id);


--
-- Name: notifications notifications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_pkey PRIMARY KEY (id);


--
-- Name: ops_audit_logs ops_audit_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ops_audit_logs
    ADD CONSTRAINT ops_audit_logs_pkey PRIMARY KEY (id);


--
-- Name: ops_command_records ops_command_records_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ops_command_records
    ADD CONSTRAINT ops_command_records_pkey PRIMARY KEY (id);


--
-- Name: ops_diagnostics ops_diagnostics_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ops_diagnostics
    ADD CONSTRAINT ops_diagnostics_pkey PRIMARY KEY (id);


--
-- Name: ops_downloads ops_downloads_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ops_downloads
    ADD CONSTRAINT ops_downloads_pkey PRIMARY KEY (id);


--
-- Name: ops_maintenance_windows ops_maintenance_windows_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ops_maintenance_windows
    ADD CONSTRAINT ops_maintenance_windows_pkey PRIMARY KEY (id);


--
-- Name: ops_playbooks ops_playbooks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ops_playbooks
    ADD CONSTRAINT ops_playbooks_pkey PRIMARY KEY (id);


--
-- Name: ops_task_executions ops_task_executions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ops_task_executions
    ADD CONSTRAINT ops_task_executions_pkey PRIMARY KEY (id);


--
-- Name: ops_tasks ops_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ops_tasks
    ADD CONSTRAINT ops_tasks_pkey PRIMARY KEY (id);


--
-- Name: ops_templates ops_templates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ops_templates
    ADD CONSTRAINT ops_templates_pkey PRIMARY KEY (id);


--
-- Name: param_mappings param_mappings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.param_mappings
    ADD CONSTRAINT param_mappings_pkey PRIMARY KEY (id);


--
-- Name: param_models param_models_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.param_models
    ADD CONSTRAINT param_models_name_key UNIQUE (name);


--
-- Name: param_models param_models_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.param_models
    ADD CONSTRAINT param_models_pkey PRIMARY KEY (id);


--
-- Name: parameter_discovery_log parameter_discovery_log_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_discovery_log
    ADD CONSTRAINT parameter_discovery_log_pkey PRIMARY KEY (id);


--
-- Name: parameter_sync_admission_reservations parameter_sync_admission_reservations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_admission_reservations
    ADD CONSTRAINT parameter_sync_admission_reservations_pkey PRIMARY KEY (id);


--
-- Name: parameter_sync_admission_state parameter_sync_admission_state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_admission_state
    ADD CONSTRAINT parameter_sync_admission_state_pkey PRIMARY KEY (admission_class, bucket_id);


--
-- Name: parameter_sync_device_state parameter_sync_device_state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_device_state
    ADD CONSTRAINT parameter_sync_device_state_pkey PRIMARY KEY (device_id);


--
-- Name: parameter_sync_event_failures parameter_sync_event_failures_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_event_failures
    ADD CONSTRAINT parameter_sync_event_failures_pkey PRIMARY KEY (id);


--
-- Name: parameter_sync_outbox parameter_sync_outbox_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_outbox
    ADD CONSTRAINT parameter_sync_outbox_pkey PRIMARY KEY (id);


--
-- Name: parameter_sync_recovery_state parameter_sync_recovery_state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_recovery_state
    ADD CONSTRAINT parameter_sync_recovery_state_pkey PRIMARY KEY (run_id, task_id);


--
-- Name: parameter_sync_request_bindings parameter_sync_request_bindings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_request_bindings
    ADD CONSTRAINT parameter_sync_request_bindings_pkey PRIMARY KEY (request_id, provisioning_task_id);


--
-- Name: parameter_sync_requests parameter_sync_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_requests
    ADD CONSTRAINT parameter_sync_requests_pkey PRIMARY KEY (id);


--
-- Name: parameter_sync_runs parameter_sync_runs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_runs
    ADD CONSTRAINT parameter_sync_runs_pkey PRIMARY KEY (id);


--
-- Name: parameter_sync_staging_values parameter_sync_staging_values_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_staging_values
    ADD CONSTRAINT parameter_sync_staging_values_pkey PRIMARY KEY (run_id, parameter_path);


--
-- Name: parameter_sync_task_results parameter_sync_task_results_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_task_results
    ADD CONSTRAINT parameter_sync_task_results_pkey PRIMARY KEY (run_id, task_id);


--
-- Name: enabled_pm_indicators_enb pk_enabled_pm_indicators_enb; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.enabled_pm_indicators_enb
    ADD CONSTRAINT pk_enabled_pm_indicators_enb PRIMARY KEY (operator_code, indicator_id);


--
-- Name: enabled_pm_indicators_gnb pk_enabled_pm_indicators_gnb; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.enabled_pm_indicators_gnb
    ADD CONSTRAINT pk_enabled_pm_indicators_gnb PRIMARY KEY (operator_code, indicator_id);


--
-- Name: enabled_pm_indicators_gsm pk_enabled_pm_indicators_gsm; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.enabled_pm_indicators_gsm
    ADD CONSTRAINT pk_enabled_pm_indicators_gsm PRIMARY KEY (operator_code, indicator_id);


--
-- Name: indicator_group_enb pk_indicator_group_enb; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.indicator_group_enb
    ADD CONSTRAINT pk_indicator_group_enb PRIMARY KEY (id);


--
-- Name: indicator_group_gnb pk_indicator_group_gnb; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.indicator_group_gnb
    ADD CONSTRAINT pk_indicator_group_gnb PRIMARY KEY (id);


--
-- Name: indicator_group_gsm pk_indicator_group_gsm; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.indicator_group_gsm
    ADD CONSTRAINT pk_indicator_group_gsm PRIMARY KEY (id);


--
-- Name: indicator_threshold pk_indicator_threshold; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.indicator_threshold
    ADD CONSTRAINT pk_indicator_threshold PRIMARY KEY (id);


--
-- Name: perf_alarm_threshold pk_perf_alarm_threshold; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.perf_alarm_threshold
    ADD CONSTRAINT pk_perf_alarm_threshold PRIMARY KEY (id);


--
-- Name: perf_cust_name pk_perf_cust_name; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.perf_cust_name
    ADD CONSTRAINT pk_perf_cust_name PRIMARY KEY (operator_code, perf_id);


--
-- Name: perf_indicators_enb pk_perf_indicators_enb; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.perf_indicators_enb
    ADD CONSTRAINT pk_perf_indicators_enb PRIMARY KEY (id);


--
-- Name: perf_indicators_gnb pk_perf_indicators_gnb; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.perf_indicators_gnb
    ADD CONSTRAINT pk_perf_indicators_gnb PRIMARY KEY (id);


--
-- Name: perf_indicators_gsm pk_perf_indicators_gsm; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.perf_indicators_gsm
    ADD CONSTRAINT pk_perf_indicators_gsm PRIMARY KEY (id);


--
-- Name: perf_template_rel_arithmetic pk_perf_template_rel_arithmetic; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.perf_template_rel_arithmetic
    ADD CONSTRAINT pk_perf_template_rel_arithmetic PRIMARY KEY (id);


--
-- Name: rela_platform_indicator_formula_enb pk_rela_platform_indicator_formula_enb; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rela_platform_indicator_formula_enb
    ADD CONSTRAINT pk_rela_platform_indicator_formula_enb PRIMARY KEY (id);


--
-- Name: rela_platform_indicator_formula_gnb pk_rela_platform_indicator_formula_gnb; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rela_platform_indicator_formula_gnb
    ADD CONSTRAINT pk_rela_platform_indicator_formula_gnb PRIMARY KEY (id);


--
-- Name: rela_platform_indicator_formula_gsm pk_rela_platform_indicator_formula_gsm; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rela_platform_indicator_formula_gsm
    ADD CONSTRAINT pk_rela_platform_indicator_formula_gsm PRIMARY KEY (id);


--
-- Name: pm_adhoc_task_runs pm_adhoc_task_runs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pm_adhoc_task_runs
    ADD CONSTRAINT pm_adhoc_task_runs_pkey PRIMARY KEY (id);


--
-- Name: pm_completion_watermarks pm_completion_watermarks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pm_completion_watermarks
    ADD CONSTRAINT pm_completion_watermarks_pkey PRIMARY KEY (granularity, level);


--
-- Name: pm_dashboards pm_dashboards_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pm_dashboards
    ADD CONSTRAINT pm_dashboards_pkey PRIMARY KEY (id);


--
-- Name: pm_kpi_export_tasks pm_kpi_export_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pm_kpi_export_tasks
    ADD CONSTRAINT pm_kpi_export_tasks_pkey PRIMARY KEY (id);


--
-- Name: pm_panels pm_panels_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pm_panels
    ADD CONSTRAINT pm_panels_pkey PRIMARY KEY (id);


--
-- Name: pm_query_templates pm_query_templates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pm_query_templates
    ADD CONSTRAINT pm_query_templates_pkey PRIMARY KEY (id);


--
-- Name: pm_tasks pm_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pm_tasks
    ADD CONSTRAINT pm_tasks_pkey PRIMARY KEY (id);


--
-- Name: pm_user_dashboard_preferences pm_user_dashboard_preferences_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pm_user_dashboard_preferences
    ADD CONSTRAINT pm_user_dashboard_preferences_pkey PRIMARY KEY (user_id, technology);


--
-- Name: product_class_patterns product_class_patterns_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_class_patterns
    ADD CONSTRAINT product_class_patterns_pkey PRIMARY KEY (id);


--
-- Name: product_unsupported_paths product_unsupported_paths_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_unsupported_paths
    ADD CONSTRAINT product_unsupported_paths_pkey PRIMARY KEY (id);


--
-- Name: products products_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.products
    ADD CONSTRAINT products_pkey PRIMARY KEY (id);


--
-- Name: products products_product_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.products
    ADD CONSTRAINT products_product_name_key UNIQUE (product_name);


--
-- Name: provisioning_tasks provisioning_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.provisioning_tasks
    ADD CONSTRAINT provisioning_tasks_pkey PRIMARY KEY (id);


--
-- Name: report_definitions report_definitions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.report_definitions
    ADD CONSTRAINT report_definitions_pkey PRIMARY KEY (id);


--
-- Name: report_records report_records_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.report_records
    ADD CONSTRAINT report_records_pkey PRIMARY KEY (id);


--
-- Name: restore_tasks restore_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.restore_tasks
    ADD CONSTRAINT restore_tasks_pkey PRIMARY KEY (id);


--
-- Name: restore_tasks restore_tasks_task_seq_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.restore_tasks
    ADD CONSTRAINT restore_tasks_task_seq_key UNIQUE (task_seq);


--
-- Name: role_api_permissions role_api_permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_api_permissions
    ADD CONSTRAINT role_api_permissions_pkey PRIMARY KEY (role_id, endpoint_id);


--
-- Name: role_device_groups role_device_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_device_groups
    ADD CONSTRAINT role_device_groups_pkey PRIMARY KEY (role_id, group_id);


--
-- Name: role_inheritance role_inheritance_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_inheritance
    ADD CONSTRAINT role_inheritance_pkey PRIMARY KEY (parent_role_id, child_role_id, domain);


--
-- Name: role_menus role_menus_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_menus
    ADD CONSTRAINT role_menus_pkey PRIMARY KEY (id);


--
-- Name: roles roles_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_name_key UNIQUE (name);


--
-- Name: roles roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_pkey PRIMARY KEY (id);


--
-- Name: config_apply_batches config_apply_batches_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_apply_batches
    ADD CONSTRAINT config_apply_batches_pkey PRIMARY KEY (id);


--
-- Name: config_apply_batches uq_config_apply_batch_version; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_apply_batches
    ADD CONSTRAINT uq_config_apply_batch_version UNIQUE (category, config_version);


--
-- Name: config_apply_targets config_apply_targets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_apply_targets
    ADD CONSTRAINT config_apply_targets_pkey PRIMARY KEY (id);


--
-- Name: config_apply_targets uq_config_apply_target; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_apply_targets
    ADD CONSTRAINT uq_config_apply_target UNIQUE (batch_id, target);


--
-- Name: config_apply_versions config_apply_versions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_apply_versions
    ADD CONSTRAINT config_apply_versions_pkey PRIMARY KEY (category);


--
-- Name: runtime_log_collect_sub_tasks runtime_log_collect_sub_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.runtime_log_collect_sub_tasks
    ADD CONSTRAINT runtime_log_collect_sub_tasks_pkey PRIMARY KEY (id);


--
-- Name: runtime_log_collect_tasks runtime_log_collect_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.runtime_log_collect_tasks
    ADD CONSTRAINT runtime_log_collect_tasks_pkey PRIMARY KEY (id);


--
-- Name: seed_menu_show_status_backups seed_menu_show_status_backups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.seed_menu_show_status_backups
    ADD CONSTRAINT seed_menu_show_status_backups_pkey PRIMARY KEY (migration_key, menu_id);


--
-- Name: sites sites_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sites
    ADD CONSTRAINT sites_pkey PRIMARY KEY (id);


--
-- Name: standard_commands standard_commands_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.standard_commands
    ADD CONSTRAINT standard_commands_pkey PRIMARY KEY (id);


--
-- Name: standard_commands standard_commands_version_code_object_path_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.standard_commands
    ADD CONSTRAINT standard_commands_version_code_object_path_key UNIQUE (version_code, object_path);


--
-- Name: standard_params standard_params_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.standard_params
    ADD CONSTRAINT standard_params_pkey PRIMARY KEY (id);


--
-- Name: standard_params standard_params_standard_path_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.standard_params
    ADD CONSTRAINT standard_params_standard_path_key UNIQUE (standard_path);


--
-- Name: station_fault_logs station_fault_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.station_fault_logs
    ADD CONSTRAINT station_fault_logs_pkey PRIMARY KEY (id);


--
-- Name: station_running_logs station_running_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.station_running_logs
    ADD CONSTRAINT station_running_logs_pkey PRIMARY KEY (id);


--
-- Name: sys_configs sys_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sys_configs
    ADD CONSTRAINT sys_configs_pkey PRIMARY KEY (id);


--
-- Name: sys_dictionaries sys_dictionaries_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sys_dictionaries
    ADD CONSTRAINT sys_dictionaries_pkey PRIMARY KEY (id);


--
-- Name: sys_dictionary_details sys_dictionary_details_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sys_dictionary_details
    ADD CONSTRAINT sys_dictionary_details_pkey PRIMARY KEY (id);


--
-- Name: sys_login_logs sys_login_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sys_login_logs
    ADD CONSTRAINT sys_login_logs_pkey PRIMARY KEY (id);


--
-- Name: sys_oper_logs sys_oper_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sys_oper_logs
    ADD CONSTRAINT sys_oper_logs_pkey PRIMARY KEY (id);


--
-- Name: sys_task_logs sys_task_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sys_task_logs
    ADD CONSTRAINT sys_task_logs_pkey PRIMARY KEY (id);


--
-- Name: system_license_history system_license_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_license_history
    ADD CONSTRAINT system_license_history_pkey PRIMARY KEY (id);


--
-- Name: system_license system_license_license_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_license
    ADD CONSTRAINT system_license_license_id_key UNIQUE (license_id);


--
-- Name: system_license system_license_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_license
    ADD CONSTRAINT system_license_pkey PRIMARY KEY (id);


--
-- Name: system_logs system_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_logs
    ADD CONSTRAINT system_logs_pkey PRIMARY KEY (id);


--
-- Name: topo_edges topo_edges_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.topo_edges
    ADD CONSTRAINT topo_edges_pkey PRIMARY KEY (id);


--
-- Name: topo_nodes topo_nodes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.topo_nodes
    ADD CONSTRAINT topo_nodes_pkey PRIMARY KEY (id);


--
-- Name: trace_export_jobs trace_export_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trace_export_jobs
    ADD CONSTRAINT trace_export_jobs_pkey PRIMARY KEY (id);


--
-- Name: trace_tasks trace_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trace_tasks
    ADD CONSTRAINT trace_tasks_pkey PRIMARY KEY (id);


--
-- Name: ufte_task_types ufte_task_types_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ufte_task_types
    ADD CONSTRAINT ufte_task_types_pkey PRIMARY KEY (type_code);


--
-- Name: sys_configs uniq_config_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sys_configs
    ADD CONSTRAINT uniq_config_key UNIQUE (category, key);


--
-- Name: role_menus uniq_role_menu; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_menus
    ADD CONSTRAINT uniq_role_menu UNIQUE (role_id, menu_id);


--
-- Name: upgrade_sub_tasks upgrade_sub_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.upgrade_sub_tasks
    ADD CONSTRAINT upgrade_sub_tasks_pkey PRIMARY KEY (id);


--
-- Name: upgrade_tasks upgrade_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.upgrade_tasks
    ADD CONSTRAINT upgrade_tasks_pkey PRIMARY KEY (id);


--
-- Name: mml_custom_command_paths uq_ccp_command_path; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_custom_command_paths
    ADD CONSTRAINT uq_ccp_command_path UNIQUE (command_id, standard_path_id);


--
-- Name: mml_command_sub_fields uq_command_mml_code; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_command_sub_fields
    ADD CONSTRAINT uq_command_mml_code UNIQUE (command_id, mml_code);


--
-- Name: mml_command_sub_fields uq_command_standard_path; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_command_sub_fields
    ADD CONSTRAINT uq_command_standard_path UNIQUE (command_id, standard_path_id);


--
-- Name: device_group_members uq_dgm_device; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_group_members
    ADD CONSTRAINT uq_dgm_device UNIQUE (device_id);


--
-- Name: mml_command_groups uq_group_version_code; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_command_groups
    ADD CONSTRAINT uq_group_version_code UNIQUE (param_version, group_code);


--
-- Name: mml_catalog_link_health uq_mml_catalog_link_health_path; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_catalog_link_health
    ADD CONSTRAINT uq_mml_catalog_link_health_path UNIQUE (spec_version, standard_path, group_code_object);


--
-- Name: mml_param_versions uq_mml_param_versions_code; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_param_versions
    ADD CONSTRAINT uq_mml_param_versions_code UNIQUE (version_code);


--
-- Name: model_upload_intents uq_model_upload_intents_device_task; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.model_upload_intents
    ADD CONSTRAINT uq_model_upload_intents_device_task UNIQUE (device_id, upload_task_id);


--
-- Name: model_upload_intents uq_model_upload_intents_source_event; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.model_upload_intents
    ADD CONSTRAINT uq_model_upload_intents_source_event UNIQUE (source_event_id);


--
-- Name: mr_customize_task_progress uq_mr_progress_task_cell; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mr_customize_task_progress
    ADD CONSTRAINT uq_mr_progress_task_cell UNIQUE (task_id, small_cell_code);


--
-- Name: parameter_sync_admission_reservations uq_parameter_sync_admission_reservation_request; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_admission_reservations
    ADD CONSTRAINT uq_parameter_sync_admission_reservation_request UNIQUE (request_id, admission_class);


--
-- Name: parameter_sync_event_failures uq_parameter_sync_event_failure_event; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_event_failures
    ADD CONSTRAINT uq_parameter_sync_event_failure_event UNIQUE (subject, event_id);


--
-- Name: parameter_sync_outbox uq_parameter_sync_outbox_dedupe; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_outbox
    ADD CONSTRAINT uq_parameter_sync_outbox_dedupe UNIQUE (dedupe_key);


--
-- Name: product_unsupported_paths uq_product_unsupported_path; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_unsupported_paths
    ADD CONSTRAINT uq_product_unsupported_path UNIQUE (product_id, firmware_version, standard_path);


--
-- Name: mml_command_sub_field_overrides uq_subfield_owner; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_command_sub_field_overrides
    ADD CONSTRAINT uq_subfield_owner UNIQUE (sub_field_id, owner_user_id);


--
-- Name: user_column_configs user_column_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_column_configs
    ADD CONSTRAINT user_column_configs_pkey PRIMARY KEY (user_id, page_key);


--
-- Name: user_roles user_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_pkey PRIMARY KEY (user_id, role_id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: users users_username_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_username_key UNIQUE (username);


--
-- Name: idx_device_params_device; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_params_device ON ONLY public.device_parameters USING btree (device_id);


--
-- Name: device_parameters_p00_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p00_device_id_idx ON public.device_parameters_p00 USING btree (device_id);


--
-- Name: idx_device_params_path_prefix; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_params_path_prefix ON ONLY public.device_parameters USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p00_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p00_device_id_parameter_path_idx ON public.device_parameters_p00 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: idx_device_params_swver; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_params_swver ON ONLY public.device_parameters USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);

-- cell_band_dim 每分钟只读取这六类参数；部分覆盖索引避免对全部参数分区做后缀全扫。
CREATE INDEX idx_device_params_cell_band_dim ON public.device_parameters USING btree (device_id, fap_instance, parameter_path) INCLUDE (parameter_value)
WHERE parameter_value IS NOT NULL
  AND parameter_value <> ''
  AND (parameter_path LIKE '%.CellConfig.LTE.RAN.Common.CellIdentity'
       OR parameter_path LIKE '%.NrcellIdentity'
       OR parameter_path LIKE '%IpaUnitId'
       OR parameter_path LIKE '%.CellConfig.LTE.RAN.RF.FreqBandIndicator'
       OR parameter_path LIKE '%.FreqBandIndicatorNR'
       OR parameter_path LIKE 'DeviceGSM.Bts.%.Band');


--
-- Name: device_parameters_p00_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p00_parameter_value_device_id_idx ON public.device_parameters_p00 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p01_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p01_device_id_idx ON public.device_parameters_p01 USING btree (device_id);


--
-- Name: device_parameters_p01_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p01_device_id_parameter_path_idx ON public.device_parameters_p01 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p01_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p01_parameter_value_device_id_idx ON public.device_parameters_p01 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p02_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p02_device_id_idx ON public.device_parameters_p02 USING btree (device_id);


--
-- Name: device_parameters_p02_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p02_device_id_parameter_path_idx ON public.device_parameters_p02 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p02_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p02_parameter_value_device_id_idx ON public.device_parameters_p02 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p03_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p03_device_id_idx ON public.device_parameters_p03 USING btree (device_id);


--
-- Name: device_parameters_p03_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p03_device_id_parameter_path_idx ON public.device_parameters_p03 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p03_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p03_parameter_value_device_id_idx ON public.device_parameters_p03 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p04_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p04_device_id_idx ON public.device_parameters_p04 USING btree (device_id);


--
-- Name: device_parameters_p04_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p04_device_id_parameter_path_idx ON public.device_parameters_p04 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p04_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p04_parameter_value_device_id_idx ON public.device_parameters_p04 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p05_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p05_device_id_idx ON public.device_parameters_p05 USING btree (device_id);


--
-- Name: device_parameters_p05_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p05_device_id_parameter_path_idx ON public.device_parameters_p05 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p05_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p05_parameter_value_device_id_idx ON public.device_parameters_p05 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p06_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p06_device_id_idx ON public.device_parameters_p06 USING btree (device_id);


--
-- Name: device_parameters_p06_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p06_device_id_parameter_path_idx ON public.device_parameters_p06 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p06_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p06_parameter_value_device_id_idx ON public.device_parameters_p06 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p07_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p07_device_id_idx ON public.device_parameters_p07 USING btree (device_id);


--
-- Name: device_parameters_p07_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p07_device_id_parameter_path_idx ON public.device_parameters_p07 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p07_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p07_parameter_value_device_id_idx ON public.device_parameters_p07 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p08_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p08_device_id_idx ON public.device_parameters_p08 USING btree (device_id);


--
-- Name: device_parameters_p08_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p08_device_id_parameter_path_idx ON public.device_parameters_p08 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p08_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p08_parameter_value_device_id_idx ON public.device_parameters_p08 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p09_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p09_device_id_idx ON public.device_parameters_p09 USING btree (device_id);


--
-- Name: device_parameters_p09_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p09_device_id_parameter_path_idx ON public.device_parameters_p09 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p09_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p09_parameter_value_device_id_idx ON public.device_parameters_p09 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p10_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p10_device_id_idx ON public.device_parameters_p10 USING btree (device_id);


--
-- Name: device_parameters_p10_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p10_device_id_parameter_path_idx ON public.device_parameters_p10 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p10_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p10_parameter_value_device_id_idx ON public.device_parameters_p10 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p11_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p11_device_id_idx ON public.device_parameters_p11 USING btree (device_id);


--
-- Name: device_parameters_p11_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p11_device_id_parameter_path_idx ON public.device_parameters_p11 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p11_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p11_parameter_value_device_id_idx ON public.device_parameters_p11 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p12_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p12_device_id_idx ON public.device_parameters_p12 USING btree (device_id);


--
-- Name: device_parameters_p12_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p12_device_id_parameter_path_idx ON public.device_parameters_p12 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p12_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p12_parameter_value_device_id_idx ON public.device_parameters_p12 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p13_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p13_device_id_idx ON public.device_parameters_p13 USING btree (device_id);


--
-- Name: device_parameters_p13_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p13_device_id_parameter_path_idx ON public.device_parameters_p13 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p13_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p13_parameter_value_device_id_idx ON public.device_parameters_p13 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p14_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p14_device_id_idx ON public.device_parameters_p14 USING btree (device_id);


--
-- Name: device_parameters_p14_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p14_device_id_parameter_path_idx ON public.device_parameters_p14 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p14_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p14_parameter_value_device_id_idx ON public.device_parameters_p14 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p15_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p15_device_id_idx ON public.device_parameters_p15 USING btree (device_id);


--
-- Name: device_parameters_p15_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p15_device_id_parameter_path_idx ON public.device_parameters_p15 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p15_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p15_parameter_value_device_id_idx ON public.device_parameters_p15 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p16_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p16_device_id_idx ON public.device_parameters_p16 USING btree (device_id);


--
-- Name: device_parameters_p16_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p16_device_id_parameter_path_idx ON public.device_parameters_p16 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p16_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p16_parameter_value_device_id_idx ON public.device_parameters_p16 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p17_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p17_device_id_idx ON public.device_parameters_p17 USING btree (device_id);


--
-- Name: device_parameters_p17_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p17_device_id_parameter_path_idx ON public.device_parameters_p17 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p17_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p17_parameter_value_device_id_idx ON public.device_parameters_p17 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p18_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p18_device_id_idx ON public.device_parameters_p18 USING btree (device_id);


--
-- Name: device_parameters_p18_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p18_device_id_parameter_path_idx ON public.device_parameters_p18 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p18_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p18_parameter_value_device_id_idx ON public.device_parameters_p18 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p19_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p19_device_id_idx ON public.device_parameters_p19 USING btree (device_id);


--
-- Name: device_parameters_p19_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p19_device_id_parameter_path_idx ON public.device_parameters_p19 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p19_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p19_parameter_value_device_id_idx ON public.device_parameters_p19 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p20_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p20_device_id_idx ON public.device_parameters_p20 USING btree (device_id);


--
-- Name: device_parameters_p20_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p20_device_id_parameter_path_idx ON public.device_parameters_p20 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p20_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p20_parameter_value_device_id_idx ON public.device_parameters_p20 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p21_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p21_device_id_idx ON public.device_parameters_p21 USING btree (device_id);


--
-- Name: device_parameters_p21_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p21_device_id_parameter_path_idx ON public.device_parameters_p21 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p21_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p21_parameter_value_device_id_idx ON public.device_parameters_p21 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p22_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p22_device_id_idx ON public.device_parameters_p22 USING btree (device_id);


--
-- Name: device_parameters_p22_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p22_device_id_parameter_path_idx ON public.device_parameters_p22 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p22_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p22_parameter_value_device_id_idx ON public.device_parameters_p22 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p23_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p23_device_id_idx ON public.device_parameters_p23 USING btree (device_id);


--
-- Name: device_parameters_p23_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p23_device_id_parameter_path_idx ON public.device_parameters_p23 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p23_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p23_parameter_value_device_id_idx ON public.device_parameters_p23 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p24_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p24_device_id_idx ON public.device_parameters_p24 USING btree (device_id);


--
-- Name: device_parameters_p24_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p24_device_id_parameter_path_idx ON public.device_parameters_p24 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p24_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p24_parameter_value_device_id_idx ON public.device_parameters_p24 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p25_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p25_device_id_idx ON public.device_parameters_p25 USING btree (device_id);


--
-- Name: device_parameters_p25_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p25_device_id_parameter_path_idx ON public.device_parameters_p25 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p25_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p25_parameter_value_device_id_idx ON public.device_parameters_p25 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p26_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p26_device_id_idx ON public.device_parameters_p26 USING btree (device_id);


--
-- Name: device_parameters_p26_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p26_device_id_parameter_path_idx ON public.device_parameters_p26 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p26_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p26_parameter_value_device_id_idx ON public.device_parameters_p26 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p27_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p27_device_id_idx ON public.device_parameters_p27 USING btree (device_id);


--
-- Name: device_parameters_p27_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p27_device_id_parameter_path_idx ON public.device_parameters_p27 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p27_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p27_parameter_value_device_id_idx ON public.device_parameters_p27 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p28_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p28_device_id_idx ON public.device_parameters_p28 USING btree (device_id);


--
-- Name: device_parameters_p28_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p28_device_id_parameter_path_idx ON public.device_parameters_p28 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p28_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p28_parameter_value_device_id_idx ON public.device_parameters_p28 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p29_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p29_device_id_idx ON public.device_parameters_p29 USING btree (device_id);


--
-- Name: device_parameters_p29_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p29_device_id_parameter_path_idx ON public.device_parameters_p29 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p29_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p29_parameter_value_device_id_idx ON public.device_parameters_p29 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p30_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p30_device_id_idx ON public.device_parameters_p30 USING btree (device_id);


--
-- Name: device_parameters_p30_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p30_device_id_parameter_path_idx ON public.device_parameters_p30 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p30_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p30_parameter_value_device_id_idx ON public.device_parameters_p30 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: device_parameters_p31_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p31_device_id_idx ON public.device_parameters_p31 USING btree (device_id);


--
-- Name: device_parameters_p31_device_id_parameter_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p31_device_id_parameter_path_idx ON public.device_parameters_p31 USING btree (device_id, parameter_path varchar_pattern_ops);


--
-- Name: device_parameters_p31_parameter_value_device_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_parameters_p31_parameter_value_device_id_idx ON public.device_parameters_p31 USING btree (parameter_value, device_id) WHERE ((parameter_path)::text = 'Device.DeviceInfo.SoftwareVersion'::text);


--
-- Name: idx_device_tasks_pending_created_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_tasks_pending_created_id ON ONLY public.device_tasks USING btree (created_at, id) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p00_created_at_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p00_created_at_id_idx ON public.device_tasks_p00 USING btree (created_at, id) WHERE ((status)::text = 'pending'::text);


--
-- Name: idx_device_tasks_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_tasks_created_at ON ONLY public.device_tasks USING btree (created_at);


--
-- Name: device_tasks_p00_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p00_created_at_idx ON public.device_tasks_p00 USING btree (created_at);


--
-- Name: idx_device_tasks_has_path_translation_miss; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_tasks_has_path_translation_miss ON ONLY public.device_tasks USING btree (created_at DESC) WHERE (has_path_translation_miss = true);


--
-- Name: device_tasks_p00_created_at_idx1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p00_created_at_idx1 ON public.device_tasks_p00 USING btree (created_at DESC) WHERE (has_path_translation_miss = true);


--
-- Name: idx_device_tasks_cwmp_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_tasks_cwmp_id ON ONLY public.device_tasks USING btree (cwmp_id) WHERE (cwmp_id IS NOT NULL);


--
-- Name: device_tasks_p00_cwmp_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p00_cwmp_id_idx ON public.device_tasks_p00 USING btree (cwmp_id) WHERE (cwmp_id IS NOT NULL);


--
-- Name: idx_device_tasks_pending; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_tasks_pending ON ONLY public.device_tasks USING btree (device_sn, status, priority, created_at) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p00_device_sn_status_priority_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p00_device_sn_status_priority_created_at_idx ON public.device_tasks_p00 USING btree (device_sn, status, priority, created_at) WHERE ((status)::text = 'pending'::text);


--
-- Name: idx_device_tasks_params_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_tasks_params_gin ON ONLY public.device_tasks USING gin (params jsonb_path_ops);


--
-- Name: device_tasks_p00_params_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p00_params_idx ON public.device_tasks_p00 USING gin (params jsonb_path_ops);


--
-- Name: idx_device_tasks_source_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_tasks_source_id ON ONLY public.device_tasks USING btree (source_id) WHERE (source_id IS NOT NULL);


--
-- Name: device_tasks_p00_parent_task_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p00_parent_task_id_idx ON public.device_tasks_p00 USING btree (source_id) WHERE (source_id IS NOT NULL);


--
-- Name: idx_device_tasks_result_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_tasks_result_gin ON ONLY public.device_tasks USING gin (result jsonb_path_ops);


--
-- Name: device_tasks_p00_result_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p00_result_idx ON public.device_tasks_p00 USING gin (result jsonb_path_ops);


--
-- Name: idx_device_tasks_status_expires; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_tasks_status_expires ON ONLY public.device_tasks USING btree (status, expires_at) WHERE (expires_at IS NOT NULL);


--
-- Name: device_tasks_p00_status_expires_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p00_status_expires_at_idx ON public.device_tasks_p00 USING btree (status, expires_at) WHERE (expires_at IS NOT NULL);


--
-- Name: idx_device_tasks_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_tasks_status ON ONLY public.device_tasks USING btree (status);


--
-- Name: device_tasks_p00_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p00_status_idx ON public.device_tasks_p00 USING btree (status);


--
-- Name: device_tasks_p01_created_at_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p01_created_at_id_idx ON public.device_tasks_p01 USING btree (created_at, id) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p01_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p01_created_at_idx ON public.device_tasks_p01 USING btree (created_at);


--
-- Name: device_tasks_p01_created_at_idx1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p01_created_at_idx1 ON public.device_tasks_p01 USING btree (created_at DESC) WHERE (has_path_translation_miss = true);


--
-- Name: device_tasks_p01_cwmp_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p01_cwmp_id_idx ON public.device_tasks_p01 USING btree (cwmp_id) WHERE (cwmp_id IS NOT NULL);


--
-- Name: device_tasks_p01_device_sn_status_priority_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p01_device_sn_status_priority_created_at_idx ON public.device_tasks_p01 USING btree (device_sn, status, priority, created_at) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p01_params_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p01_params_idx ON public.device_tasks_p01 USING gin (params jsonb_path_ops);


--
-- Name: device_tasks_p01_parent_task_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p01_parent_task_id_idx ON public.device_tasks_p01 USING btree (source_id) WHERE (source_id IS NOT NULL);


--
-- Name: device_tasks_p01_result_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p01_result_idx ON public.device_tasks_p01 USING gin (result jsonb_path_ops);


--
-- Name: device_tasks_p01_status_expires_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p01_status_expires_at_idx ON public.device_tasks_p01 USING btree (status, expires_at) WHERE (expires_at IS NOT NULL);


--
-- Name: device_tasks_p01_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p01_status_idx ON public.device_tasks_p01 USING btree (status);


--
-- Name: device_tasks_p02_created_at_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p02_created_at_id_idx ON public.device_tasks_p02 USING btree (created_at, id) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p02_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p02_created_at_idx ON public.device_tasks_p02 USING btree (created_at);


--
-- Name: device_tasks_p02_created_at_idx1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p02_created_at_idx1 ON public.device_tasks_p02 USING btree (created_at DESC) WHERE (has_path_translation_miss = true);


--
-- Name: device_tasks_p02_cwmp_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p02_cwmp_id_idx ON public.device_tasks_p02 USING btree (cwmp_id) WHERE (cwmp_id IS NOT NULL);


--
-- Name: device_tasks_p02_device_sn_status_priority_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p02_device_sn_status_priority_created_at_idx ON public.device_tasks_p02 USING btree (device_sn, status, priority, created_at) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p02_params_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p02_params_idx ON public.device_tasks_p02 USING gin (params jsonb_path_ops);


--
-- Name: device_tasks_p02_parent_task_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p02_parent_task_id_idx ON public.device_tasks_p02 USING btree (source_id) WHERE (source_id IS NOT NULL);


--
-- Name: device_tasks_p02_result_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p02_result_idx ON public.device_tasks_p02 USING gin (result jsonb_path_ops);


--
-- Name: device_tasks_p02_status_expires_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p02_status_expires_at_idx ON public.device_tasks_p02 USING btree (status, expires_at) WHERE (expires_at IS NOT NULL);


--
-- Name: device_tasks_p02_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p02_status_idx ON public.device_tasks_p02 USING btree (status);


--
-- Name: device_tasks_p03_created_at_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p03_created_at_id_idx ON public.device_tasks_p03 USING btree (created_at, id) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p03_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p03_created_at_idx ON public.device_tasks_p03 USING btree (created_at);


--
-- Name: device_tasks_p03_created_at_idx1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p03_created_at_idx1 ON public.device_tasks_p03 USING btree (created_at DESC) WHERE (has_path_translation_miss = true);


--
-- Name: device_tasks_p03_cwmp_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p03_cwmp_id_idx ON public.device_tasks_p03 USING btree (cwmp_id) WHERE (cwmp_id IS NOT NULL);


--
-- Name: device_tasks_p03_device_sn_status_priority_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p03_device_sn_status_priority_created_at_idx ON public.device_tasks_p03 USING btree (device_sn, status, priority, created_at) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p03_params_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p03_params_idx ON public.device_tasks_p03 USING gin (params jsonb_path_ops);


--
-- Name: device_tasks_p03_parent_task_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p03_parent_task_id_idx ON public.device_tasks_p03 USING btree (source_id) WHERE (source_id IS NOT NULL);


--
-- Name: device_tasks_p03_result_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p03_result_idx ON public.device_tasks_p03 USING gin (result jsonb_path_ops);


--
-- Name: device_tasks_p03_status_expires_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p03_status_expires_at_idx ON public.device_tasks_p03 USING btree (status, expires_at) WHERE (expires_at IS NOT NULL);


--
-- Name: device_tasks_p03_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p03_status_idx ON public.device_tasks_p03 USING btree (status);


--
-- Name: device_tasks_p04_created_at_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p04_created_at_id_idx ON public.device_tasks_p04 USING btree (created_at, id) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p04_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p04_created_at_idx ON public.device_tasks_p04 USING btree (created_at);


--
-- Name: device_tasks_p04_created_at_idx1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p04_created_at_idx1 ON public.device_tasks_p04 USING btree (created_at DESC) WHERE (has_path_translation_miss = true);


--
-- Name: device_tasks_p04_cwmp_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p04_cwmp_id_idx ON public.device_tasks_p04 USING btree (cwmp_id) WHERE (cwmp_id IS NOT NULL);


--
-- Name: device_tasks_p04_device_sn_status_priority_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p04_device_sn_status_priority_created_at_idx ON public.device_tasks_p04 USING btree (device_sn, status, priority, created_at) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p04_params_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p04_params_idx ON public.device_tasks_p04 USING gin (params jsonb_path_ops);


--
-- Name: device_tasks_p04_parent_task_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p04_parent_task_id_idx ON public.device_tasks_p04 USING btree (source_id) WHERE (source_id IS NOT NULL);


--
-- Name: device_tasks_p04_result_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p04_result_idx ON public.device_tasks_p04 USING gin (result jsonb_path_ops);


--
-- Name: device_tasks_p04_status_expires_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p04_status_expires_at_idx ON public.device_tasks_p04 USING btree (status, expires_at) WHERE (expires_at IS NOT NULL);


--
-- Name: device_tasks_p04_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p04_status_idx ON public.device_tasks_p04 USING btree (status);


--
-- Name: device_tasks_p05_created_at_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p05_created_at_id_idx ON public.device_tasks_p05 USING btree (created_at, id) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p05_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p05_created_at_idx ON public.device_tasks_p05 USING btree (created_at);


--
-- Name: device_tasks_p05_created_at_idx1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p05_created_at_idx1 ON public.device_tasks_p05 USING btree (created_at DESC) WHERE (has_path_translation_miss = true);


--
-- Name: device_tasks_p05_cwmp_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p05_cwmp_id_idx ON public.device_tasks_p05 USING btree (cwmp_id) WHERE (cwmp_id IS NOT NULL);


--
-- Name: device_tasks_p05_device_sn_status_priority_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p05_device_sn_status_priority_created_at_idx ON public.device_tasks_p05 USING btree (device_sn, status, priority, created_at) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p05_params_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p05_params_idx ON public.device_tasks_p05 USING gin (params jsonb_path_ops);


--
-- Name: device_tasks_p05_parent_task_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p05_parent_task_id_idx ON public.device_tasks_p05 USING btree (source_id) WHERE (source_id IS NOT NULL);


--
-- Name: device_tasks_p05_result_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p05_result_idx ON public.device_tasks_p05 USING gin (result jsonb_path_ops);


--
-- Name: device_tasks_p05_status_expires_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p05_status_expires_at_idx ON public.device_tasks_p05 USING btree (status, expires_at) WHERE (expires_at IS NOT NULL);


--
-- Name: device_tasks_p05_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p05_status_idx ON public.device_tasks_p05 USING btree (status);


--
-- Name: device_tasks_p06_created_at_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p06_created_at_id_idx ON public.device_tasks_p06 USING btree (created_at, id) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p06_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p06_created_at_idx ON public.device_tasks_p06 USING btree (created_at);


--
-- Name: device_tasks_p06_created_at_idx1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p06_created_at_idx1 ON public.device_tasks_p06 USING btree (created_at DESC) WHERE (has_path_translation_miss = true);


--
-- Name: device_tasks_p06_cwmp_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p06_cwmp_id_idx ON public.device_tasks_p06 USING btree (cwmp_id) WHERE (cwmp_id IS NOT NULL);


--
-- Name: device_tasks_p06_device_sn_status_priority_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p06_device_sn_status_priority_created_at_idx ON public.device_tasks_p06 USING btree (device_sn, status, priority, created_at) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p06_params_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p06_params_idx ON public.device_tasks_p06 USING gin (params jsonb_path_ops);


--
-- Name: device_tasks_p06_parent_task_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p06_parent_task_id_idx ON public.device_tasks_p06 USING btree (source_id) WHERE (source_id IS NOT NULL);


--
-- Name: device_tasks_p06_result_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p06_result_idx ON public.device_tasks_p06 USING gin (result jsonb_path_ops);


--
-- Name: device_tasks_p06_status_expires_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p06_status_expires_at_idx ON public.device_tasks_p06 USING btree (status, expires_at) WHERE (expires_at IS NOT NULL);


--
-- Name: device_tasks_p06_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p06_status_idx ON public.device_tasks_p06 USING btree (status);


--
-- Name: device_tasks_p07_created_at_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p07_created_at_id_idx ON public.device_tasks_p07 USING btree (created_at, id) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p07_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p07_created_at_idx ON public.device_tasks_p07 USING btree (created_at);


--
-- Name: device_tasks_p07_created_at_idx1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p07_created_at_idx1 ON public.device_tasks_p07 USING btree (created_at DESC) WHERE (has_path_translation_miss = true);


--
-- Name: device_tasks_p07_cwmp_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p07_cwmp_id_idx ON public.device_tasks_p07 USING btree (cwmp_id) WHERE (cwmp_id IS NOT NULL);


--
-- Name: device_tasks_p07_device_sn_status_priority_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p07_device_sn_status_priority_created_at_idx ON public.device_tasks_p07 USING btree (device_sn, status, priority, created_at) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p07_params_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p07_params_idx ON public.device_tasks_p07 USING gin (params jsonb_path_ops);


--
-- Name: device_tasks_p07_parent_task_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p07_parent_task_id_idx ON public.device_tasks_p07 USING btree (source_id) WHERE (source_id IS NOT NULL);


--
-- Name: device_tasks_p07_result_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p07_result_idx ON public.device_tasks_p07 USING gin (result jsonb_path_ops);


--
-- Name: device_tasks_p07_status_expires_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p07_status_expires_at_idx ON public.device_tasks_p07 USING btree (status, expires_at) WHERE (expires_at IS NOT NULL);


--
-- Name: device_tasks_p07_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p07_status_idx ON public.device_tasks_p07 USING btree (status);


--
-- Name: device_tasks_p08_created_at_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p08_created_at_id_idx ON public.device_tasks_p08 USING btree (created_at, id) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p08_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p08_created_at_idx ON public.device_tasks_p08 USING btree (created_at);


--
-- Name: device_tasks_p08_created_at_idx1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p08_created_at_idx1 ON public.device_tasks_p08 USING btree (created_at DESC) WHERE (has_path_translation_miss = true);


--
-- Name: device_tasks_p08_cwmp_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p08_cwmp_id_idx ON public.device_tasks_p08 USING btree (cwmp_id) WHERE (cwmp_id IS NOT NULL);


--
-- Name: device_tasks_p08_device_sn_status_priority_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p08_device_sn_status_priority_created_at_idx ON public.device_tasks_p08 USING btree (device_sn, status, priority, created_at) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p08_params_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p08_params_idx ON public.device_tasks_p08 USING gin (params jsonb_path_ops);


--
-- Name: device_tasks_p08_parent_task_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p08_parent_task_id_idx ON public.device_tasks_p08 USING btree (source_id) WHERE (source_id IS NOT NULL);


--
-- Name: device_tasks_p08_result_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p08_result_idx ON public.device_tasks_p08 USING gin (result jsonb_path_ops);


--
-- Name: device_tasks_p08_status_expires_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p08_status_expires_at_idx ON public.device_tasks_p08 USING btree (status, expires_at) WHERE (expires_at IS NOT NULL);


--
-- Name: device_tasks_p08_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p08_status_idx ON public.device_tasks_p08 USING btree (status);


--
-- Name: device_tasks_p09_created_at_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p09_created_at_id_idx ON public.device_tasks_p09 USING btree (created_at, id) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p09_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p09_created_at_idx ON public.device_tasks_p09 USING btree (created_at);


--
-- Name: device_tasks_p09_created_at_idx1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p09_created_at_idx1 ON public.device_tasks_p09 USING btree (created_at DESC) WHERE (has_path_translation_miss = true);


--
-- Name: device_tasks_p09_cwmp_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p09_cwmp_id_idx ON public.device_tasks_p09 USING btree (cwmp_id) WHERE (cwmp_id IS NOT NULL);


--
-- Name: device_tasks_p09_device_sn_status_priority_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p09_device_sn_status_priority_created_at_idx ON public.device_tasks_p09 USING btree (device_sn, status, priority, created_at) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p09_params_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p09_params_idx ON public.device_tasks_p09 USING gin (params jsonb_path_ops);


--
-- Name: device_tasks_p09_parent_task_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p09_parent_task_id_idx ON public.device_tasks_p09 USING btree (source_id) WHERE (source_id IS NOT NULL);


--
-- Name: device_tasks_p09_result_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p09_result_idx ON public.device_tasks_p09 USING gin (result jsonb_path_ops);


--
-- Name: device_tasks_p09_status_expires_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p09_status_expires_at_idx ON public.device_tasks_p09 USING btree (status, expires_at) WHERE (expires_at IS NOT NULL);


--
-- Name: device_tasks_p09_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p09_status_idx ON public.device_tasks_p09 USING btree (status);


--
-- Name: device_tasks_p10_created_at_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p10_created_at_id_idx ON public.device_tasks_p10 USING btree (created_at, id) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p10_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p10_created_at_idx ON public.device_tasks_p10 USING btree (created_at);


--
-- Name: device_tasks_p10_created_at_idx1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p10_created_at_idx1 ON public.device_tasks_p10 USING btree (created_at DESC) WHERE (has_path_translation_miss = true);


--
-- Name: device_tasks_p10_cwmp_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p10_cwmp_id_idx ON public.device_tasks_p10 USING btree (cwmp_id) WHERE (cwmp_id IS NOT NULL);


--
-- Name: device_tasks_p10_device_sn_status_priority_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p10_device_sn_status_priority_created_at_idx ON public.device_tasks_p10 USING btree (device_sn, status, priority, created_at) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p10_params_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p10_params_idx ON public.device_tasks_p10 USING gin (params jsonb_path_ops);


--
-- Name: device_tasks_p10_parent_task_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p10_parent_task_id_idx ON public.device_tasks_p10 USING btree (source_id) WHERE (source_id IS NOT NULL);


--
-- Name: device_tasks_p10_result_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p10_result_idx ON public.device_tasks_p10 USING gin (result jsonb_path_ops);


--
-- Name: device_tasks_p10_status_expires_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p10_status_expires_at_idx ON public.device_tasks_p10 USING btree (status, expires_at) WHERE (expires_at IS NOT NULL);


--
-- Name: device_tasks_p10_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p10_status_idx ON public.device_tasks_p10 USING btree (status);


--
-- Name: device_tasks_p11_created_at_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p11_created_at_id_idx ON public.device_tasks_p11 USING btree (created_at, id) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p11_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p11_created_at_idx ON public.device_tasks_p11 USING btree (created_at);


--
-- Name: device_tasks_p11_created_at_idx1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p11_created_at_idx1 ON public.device_tasks_p11 USING btree (created_at DESC) WHERE (has_path_translation_miss = true);


--
-- Name: device_tasks_p11_cwmp_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p11_cwmp_id_idx ON public.device_tasks_p11 USING btree (cwmp_id) WHERE (cwmp_id IS NOT NULL);


--
-- Name: device_tasks_p11_device_sn_status_priority_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p11_device_sn_status_priority_created_at_idx ON public.device_tasks_p11 USING btree (device_sn, status, priority, created_at) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p11_params_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p11_params_idx ON public.device_tasks_p11 USING gin (params jsonb_path_ops);


--
-- Name: device_tasks_p11_parent_task_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p11_parent_task_id_idx ON public.device_tasks_p11 USING btree (source_id) WHERE (source_id IS NOT NULL);


--
-- Name: device_tasks_p11_result_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p11_result_idx ON public.device_tasks_p11 USING gin (result jsonb_path_ops);


--
-- Name: device_tasks_p11_status_expires_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p11_status_expires_at_idx ON public.device_tasks_p11 USING btree (status, expires_at) WHERE (expires_at IS NOT NULL);


--
-- Name: device_tasks_p11_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p11_status_idx ON public.device_tasks_p11 USING btree (status);


--
-- Name: device_tasks_p12_created_at_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p12_created_at_id_idx ON public.device_tasks_p12 USING btree (created_at, id) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p12_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p12_created_at_idx ON public.device_tasks_p12 USING btree (created_at);


--
-- Name: device_tasks_p12_created_at_idx1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p12_created_at_idx1 ON public.device_tasks_p12 USING btree (created_at DESC) WHERE (has_path_translation_miss = true);


--
-- Name: device_tasks_p12_cwmp_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p12_cwmp_id_idx ON public.device_tasks_p12 USING btree (cwmp_id) WHERE (cwmp_id IS NOT NULL);


--
-- Name: device_tasks_p12_device_sn_status_priority_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p12_device_sn_status_priority_created_at_idx ON public.device_tasks_p12 USING btree (device_sn, status, priority, created_at) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p12_params_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p12_params_idx ON public.device_tasks_p12 USING gin (params jsonb_path_ops);


--
-- Name: device_tasks_p12_parent_task_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p12_parent_task_id_idx ON public.device_tasks_p12 USING btree (source_id) WHERE (source_id IS NOT NULL);


--
-- Name: device_tasks_p12_result_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p12_result_idx ON public.device_tasks_p12 USING gin (result jsonb_path_ops);


--
-- Name: device_tasks_p12_status_expires_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p12_status_expires_at_idx ON public.device_tasks_p12 USING btree (status, expires_at) WHERE (expires_at IS NOT NULL);


--
-- Name: device_tasks_p12_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p12_status_idx ON public.device_tasks_p12 USING btree (status);


--
-- Name: device_tasks_p13_created_at_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p13_created_at_id_idx ON public.device_tasks_p13 USING btree (created_at, id) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p13_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p13_created_at_idx ON public.device_tasks_p13 USING btree (created_at);


--
-- Name: device_tasks_p13_created_at_idx1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p13_created_at_idx1 ON public.device_tasks_p13 USING btree (created_at DESC) WHERE (has_path_translation_miss = true);


--
-- Name: device_tasks_p13_cwmp_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p13_cwmp_id_idx ON public.device_tasks_p13 USING btree (cwmp_id) WHERE (cwmp_id IS NOT NULL);


--
-- Name: device_tasks_p13_device_sn_status_priority_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p13_device_sn_status_priority_created_at_idx ON public.device_tasks_p13 USING btree (device_sn, status, priority, created_at) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p13_params_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p13_params_idx ON public.device_tasks_p13 USING gin (params jsonb_path_ops);


--
-- Name: device_tasks_p13_parent_task_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p13_parent_task_id_idx ON public.device_tasks_p13 USING btree (source_id) WHERE (source_id IS NOT NULL);


--
-- Name: device_tasks_p13_result_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p13_result_idx ON public.device_tasks_p13 USING gin (result jsonb_path_ops);


--
-- Name: device_tasks_p13_status_expires_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p13_status_expires_at_idx ON public.device_tasks_p13 USING btree (status, expires_at) WHERE (expires_at IS NOT NULL);


--
-- Name: device_tasks_p13_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p13_status_idx ON public.device_tasks_p13 USING btree (status);


--
-- Name: device_tasks_p14_created_at_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p14_created_at_id_idx ON public.device_tasks_p14 USING btree (created_at, id) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p14_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p14_created_at_idx ON public.device_tasks_p14 USING btree (created_at);


--
-- Name: device_tasks_p14_created_at_idx1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p14_created_at_idx1 ON public.device_tasks_p14 USING btree (created_at DESC) WHERE (has_path_translation_miss = true);


--
-- Name: device_tasks_p14_cwmp_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p14_cwmp_id_idx ON public.device_tasks_p14 USING btree (cwmp_id) WHERE (cwmp_id IS NOT NULL);


--
-- Name: device_tasks_p14_device_sn_status_priority_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p14_device_sn_status_priority_created_at_idx ON public.device_tasks_p14 USING btree (device_sn, status, priority, created_at) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p14_params_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p14_params_idx ON public.device_tasks_p14 USING gin (params jsonb_path_ops);


--
-- Name: device_tasks_p14_parent_task_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p14_parent_task_id_idx ON public.device_tasks_p14 USING btree (source_id) WHERE (source_id IS NOT NULL);


--
-- Name: device_tasks_p14_result_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p14_result_idx ON public.device_tasks_p14 USING gin (result jsonb_path_ops);


--
-- Name: device_tasks_p14_status_expires_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p14_status_expires_at_idx ON public.device_tasks_p14 USING btree (status, expires_at) WHERE (expires_at IS NOT NULL);


--
-- Name: device_tasks_p14_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p14_status_idx ON public.device_tasks_p14 USING btree (status);


--
-- Name: device_tasks_p15_created_at_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p15_created_at_id_idx ON public.device_tasks_p15 USING btree (created_at, id) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p15_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p15_created_at_idx ON public.device_tasks_p15 USING btree (created_at);


--
-- Name: device_tasks_p15_created_at_idx1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p15_created_at_idx1 ON public.device_tasks_p15 USING btree (created_at DESC) WHERE (has_path_translation_miss = true);


--
-- Name: device_tasks_p15_cwmp_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p15_cwmp_id_idx ON public.device_tasks_p15 USING btree (cwmp_id) WHERE (cwmp_id IS NOT NULL);


--
-- Name: device_tasks_p15_device_sn_status_priority_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p15_device_sn_status_priority_created_at_idx ON public.device_tasks_p15 USING btree (device_sn, status, priority, created_at) WHERE ((status)::text = 'pending'::text);


--
-- Name: device_tasks_p15_params_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p15_params_idx ON public.device_tasks_p15 USING gin (params jsonb_path_ops);


--
-- Name: device_tasks_p15_parent_task_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p15_parent_task_id_idx ON public.device_tasks_p15 USING btree (source_id) WHERE (source_id IS NOT NULL);


--
-- Name: device_tasks_p15_result_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p15_result_idx ON public.device_tasks_p15 USING gin (result jsonb_path_ops);


--
-- Name: device_tasks_p15_status_expires_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p15_status_expires_at_idx ON public.device_tasks_p15 USING btree (status, expires_at) WHERE (expires_at IS NOT NULL);


--
-- Name: device_tasks_p15_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tasks_p15_status_idx ON public.device_tasks_p15 USING btree (status);


--
-- Name: idx_devices_carrier_alive; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_carrier_alive ON ONLY public.devices USING btree (carrier, last_inform_at DESC) WHERE (deleted_at IS NULL);


--
-- Name: devices_cmcc_carrier_last_inform_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cmcc_carrier_last_inform_at_idx ON public.devices_cmcc USING btree (carrier, last_inform_at DESC) WHERE (deleted_at IS NULL);


--
-- Name: idx_devices_carrier_lifecycle; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_carrier_lifecycle ON ONLY public.devices USING btree (carrier, lifecycle_state);


--
-- Name: devices_cmcc_carrier_lifecycle_state_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cmcc_carrier_lifecycle_state_idx ON public.devices_cmcc USING btree (carrier, lifecycle_state);


--
-- Name: idx_devices_carrier_tech; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_carrier_tech ON ONLY public.devices USING btree (carrier, technology);


--
-- Name: devices_cmcc_carrier_technology_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cmcc_carrier_technology_idx ON public.devices_cmcc USING btree (carrier, technology);


--
-- Name: idx_devices_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_deleted_at ON ONLY public.devices USING btree (deleted_at) WHERE (deleted_at IS NOT NULL);


--
-- Name: devices_cmcc_deleted_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cmcc_deleted_at_idx ON public.devices_cmcc USING btree (deleted_at) WHERE (deleted_at IS NOT NULL);


--
-- Name: idx_devices_extension_data_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_extension_data_gin ON ONLY public.devices USING gin (extension_data jsonb_path_ops);


--
-- Name: devices_cmcc_extension_data_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cmcc_extension_data_idx ON public.devices_cmcc USING gin (extension_data jsonb_path_ops);


--
-- Name: idx_devices_ip_address; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_ip_address ON ONLY public.devices USING btree (ip_address) WHERE (ip_address IS NOT NULL);


--
-- Name: devices_cmcc_ip_address_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cmcc_ip_address_idx ON public.devices_cmcc USING btree (ip_address) WHERE (ip_address IS NOT NULL);


--
-- Name: idx_devices_is_online; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_is_online ON ONLY public.devices USING btree (is_online) WHERE (is_online = true);


--
-- Name: devices_cmcc_is_online_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cmcc_is_online_idx ON public.devices_cmcc USING btree (is_online) WHERE (is_online = true);


--
-- Name: idx_devices_last_boot_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_last_boot_at ON ONLY public.devices USING btree (last_boot_at) WHERE (last_boot_at IS NOT NULL);


--
-- Name: devices_cmcc_last_boot_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cmcc_last_boot_at_idx ON public.devices_cmcc USING btree (last_boot_at) WHERE (last_boot_at IS NOT NULL);


--
-- Name: idx_devices_last_inform; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_last_inform ON ONLY public.devices USING btree (last_inform_at);


--
-- Name: devices_cmcc_last_inform_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cmcc_last_inform_at_idx ON public.devices_cmcc USING btree (last_inform_at);


--
-- Name: idx_devices_last_inform_events_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_last_inform_events_gin ON ONLY public.devices USING gin (last_inform_events jsonb_path_ops);


--
-- Name: devices_cmcc_last_inform_events_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cmcc_last_inform_events_idx ON public.devices_cmcc USING gin (last_inform_events jsonb_path_ops);


--
-- Name: idx_devices_lifecycle_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_lifecycle_state ON ONLY public.devices USING btree (lifecycle_state);


--
-- Name: devices_cmcc_lifecycle_state_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cmcc_lifecycle_state_idx ON public.devices_cmcc USING btree (lifecycle_state);


--
-- Name: idx_devices_oui; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_oui ON ONLY public.devices USING btree (oui);


--
-- Name: devices_cmcc_oui_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cmcc_oui_idx ON public.devices_cmcc USING btree (oui);


--
-- Name: idx_devices_serial_number; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_devices_serial_number ON ONLY public.devices USING btree (serial_number, carrier) WHERE (deleted_at IS NULL);


--
-- Name: devices_cmcc_serial_number_carrier_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX devices_cmcc_serial_number_carrier_idx ON public.devices_cmcc USING btree (serial_number, carrier) WHERE (deleted_at IS NULL);


--
-- Name: devices_ctcc_carrier_last_inform_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_ctcc_carrier_last_inform_at_idx ON public.devices_ctcc USING btree (carrier, last_inform_at DESC) WHERE (deleted_at IS NULL);


--
-- Name: devices_ctcc_carrier_lifecycle_state_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_ctcc_carrier_lifecycle_state_idx ON public.devices_ctcc USING btree (carrier, lifecycle_state);


--
-- Name: devices_ctcc_carrier_technology_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_ctcc_carrier_technology_idx ON public.devices_ctcc USING btree (carrier, technology);


--
-- Name: devices_ctcc_deleted_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_ctcc_deleted_at_idx ON public.devices_ctcc USING btree (deleted_at) WHERE (deleted_at IS NOT NULL);


--
-- Name: devices_ctcc_extension_data_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_ctcc_extension_data_idx ON public.devices_ctcc USING gin (extension_data jsonb_path_ops);


--
-- Name: devices_ctcc_ip_address_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_ctcc_ip_address_idx ON public.devices_ctcc USING btree (ip_address) WHERE (ip_address IS NOT NULL);


--
-- Name: devices_ctcc_is_online_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_ctcc_is_online_idx ON public.devices_ctcc USING btree (is_online) WHERE (is_online = true);


--
-- Name: devices_ctcc_last_boot_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_ctcc_last_boot_at_idx ON public.devices_ctcc USING btree (last_boot_at) WHERE (last_boot_at IS NOT NULL);


--
-- Name: devices_ctcc_last_inform_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_ctcc_last_inform_at_idx ON public.devices_ctcc USING btree (last_inform_at);


--
-- Name: devices_ctcc_last_inform_events_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_ctcc_last_inform_events_idx ON public.devices_ctcc USING gin (last_inform_events jsonb_path_ops);


--
-- Name: devices_ctcc_lifecycle_state_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_ctcc_lifecycle_state_idx ON public.devices_ctcc USING btree (lifecycle_state);


--
-- Name: devices_ctcc_oui_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_ctcc_oui_idx ON public.devices_ctcc USING btree (oui);


--
-- Name: devices_ctcc_serial_number_carrier_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX devices_ctcc_serial_number_carrier_idx ON public.devices_ctcc USING btree (serial_number, carrier) WHERE (deleted_at IS NULL);


--
-- Name: devices_cucc_carrier_last_inform_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cucc_carrier_last_inform_at_idx ON public.devices_cucc USING btree (carrier, last_inform_at DESC) WHERE (deleted_at IS NULL);


--
-- Name: devices_cucc_carrier_lifecycle_state_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cucc_carrier_lifecycle_state_idx ON public.devices_cucc USING btree (carrier, lifecycle_state);


--
-- Name: devices_cucc_carrier_technology_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cucc_carrier_technology_idx ON public.devices_cucc USING btree (carrier, technology);


--
-- Name: devices_cucc_deleted_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cucc_deleted_at_idx ON public.devices_cucc USING btree (deleted_at) WHERE (deleted_at IS NOT NULL);


--
-- Name: devices_cucc_extension_data_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cucc_extension_data_idx ON public.devices_cucc USING gin (extension_data jsonb_path_ops);


--
-- Name: devices_cucc_ip_address_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cucc_ip_address_idx ON public.devices_cucc USING btree (ip_address) WHERE (ip_address IS NOT NULL);


--
-- Name: devices_cucc_is_online_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cucc_is_online_idx ON public.devices_cucc USING btree (is_online) WHERE (is_online = true);


--
-- Name: devices_cucc_last_boot_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cucc_last_boot_at_idx ON public.devices_cucc USING btree (last_boot_at) WHERE (last_boot_at IS NOT NULL);


--
-- Name: devices_cucc_last_inform_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cucc_last_inform_at_idx ON public.devices_cucc USING btree (last_inform_at);


--
-- Name: devices_cucc_last_inform_events_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cucc_last_inform_events_idx ON public.devices_cucc USING gin (last_inform_events jsonb_path_ops);


--
-- Name: devices_cucc_lifecycle_state_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cucc_lifecycle_state_idx ON public.devices_cucc USING btree (lifecycle_state);


--
-- Name: devices_cucc_oui_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_cucc_oui_idx ON public.devices_cucc USING btree (oui);


--
-- Name: devices_cucc_serial_number_carrier_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX devices_cucc_serial_number_carrier_idx ON public.devices_cucc USING btree (serial_number, carrier) WHERE (deleted_at IS NULL);


--
-- Name: devices_other_carrier_last_inform_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_other_carrier_last_inform_at_idx ON public.devices_other USING btree (carrier, last_inform_at DESC) WHERE (deleted_at IS NULL);


--
-- Name: devices_other_carrier_lifecycle_state_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_other_carrier_lifecycle_state_idx ON public.devices_other USING btree (carrier, lifecycle_state);


--
-- Name: devices_other_carrier_technology_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_other_carrier_technology_idx ON public.devices_other USING btree (carrier, technology);


--
-- Name: devices_other_deleted_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_other_deleted_at_idx ON public.devices_other USING btree (deleted_at) WHERE (deleted_at IS NOT NULL);


--
-- Name: devices_other_extension_data_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_other_extension_data_idx ON public.devices_other USING gin (extension_data jsonb_path_ops);


--
-- Name: devices_other_ip_address_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_other_ip_address_idx ON public.devices_other USING btree (ip_address) WHERE (ip_address IS NOT NULL);


--
-- Name: devices_other_is_online_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_other_is_online_idx ON public.devices_other USING btree (is_online) WHERE (is_online = true);


--
-- Name: devices_other_last_boot_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_other_last_boot_at_idx ON public.devices_other USING btree (last_boot_at) WHERE (last_boot_at IS NOT NULL);


--
-- Name: devices_other_last_inform_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_other_last_inform_at_idx ON public.devices_other USING btree (last_inform_at);


--
-- Name: devices_other_last_inform_events_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_other_last_inform_events_idx ON public.devices_other USING gin (last_inform_events jsonb_path_ops);


--
-- Name: devices_other_lifecycle_state_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_other_lifecycle_state_idx ON public.devices_other USING btree (lifecycle_state);


--
-- Name: devices_other_oui_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_other_oui_idx ON public.devices_other USING btree (oui);


--
-- Name: devices_other_serial_number_carrier_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX devices_other_serial_number_carrier_idx ON public.devices_other USING btree (serial_number, carrier) WHERE (deleted_at IS NULL);


--
-- Name: idx_alarm_definitions_ne_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarm_definitions_ne_type ON public.alarm_definitions USING btree (ne_type);


--
-- Name: idx_alarm_definitions_severity_show; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarm_definitions_severity_show ON public.alarm_definitions USING btree (severity_id, is_show);


--
-- Name: idx_alarm_filters_action; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarm_filters_action ON public.alarm_filters USING btree (action);


--
-- Name: idx_alarm_filters_alarm_identifiers; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarm_filters_alarm_identifiers ON public.alarm_filters USING gin (alarm_identifiers);


--
-- Name: idx_alarm_filters_alarm_sources; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarm_filters_alarm_sources ON public.alarm_filters USING gin (alarm_sources);


--
-- Name: idx_alarm_filters_device_group_ids; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarm_filters_device_group_ids ON public.alarm_filters USING gin (device_group_ids);


--
-- Name: idx_alarm_filters_device_ids; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarm_filters_device_ids ON public.alarm_filters USING gin (device_ids);


--
-- Name: idx_alarm_filters_enabled_priority; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarm_filters_enabled_priority ON public.alarm_filters USING btree (enabled, priority);


--
-- Name: idx_alarm_rules_action_config_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarm_rules_action_config_gin ON public.alarm_rules USING gin (action_config jsonb_path_ops);


--
-- Name: idx_alarm_rules_alarm_identifier; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarm_rules_alarm_identifier ON public.alarm_rules USING btree (alarm_identifier);


--
-- Name: idx_alarm_rules_carrier; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarm_rules_carrier ON public.alarm_rules USING btree (carrier);


--
-- Name: idx_alarm_rules_condition_config_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarm_rules_condition_config_gin ON public.alarm_rules USING gin (condition_config jsonb_path_ops);


--
-- Name: idx_alarm_rules_enabled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarm_rules_enabled ON public.alarm_rules USING btree (enabled);


--
-- Name: idx_alarm_webhook_dead_letters_failed_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarm_webhook_dead_letters_failed_at ON public.alarm_webhook_dead_letters USING btree (failed_at DESC);


--
-- Name: idx_alarm_webhook_dead_letters_filter_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarm_webhook_dead_letters_filter_id ON public.alarm_webhook_dead_letters USING btree (filter_id);


--
-- Name: idx_alarms_active_additional_info_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarms_active_additional_info_gin ON public.alarms_active USING gin (additional_info jsonb_path_ops);


--
-- Name: idx_alarms_active_alarm_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarms_active_alarm_type ON public.alarms_active USING btree (alarm_type);


--
-- Name: idx_alarms_active_carrier_severity; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarms_active_carrier_severity ON public.alarms_active USING btree (carrier, severity);


--
-- Name: idx_alarms_active_device; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarms_active_device ON public.alarms_active USING btree (device_id);


--
-- Name: idx_alarms_active_device_identifier; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarms_active_device_identifier ON public.alarms_active USING btree (device_sn, alarm_identifier);


--
-- Name: idx_alarms_active_device_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarms_active_device_name ON public.alarms_active USING btree (device_name);


--
-- Name: idx_alarms_active_device_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarms_active_device_time ON public.alarms_active USING btree (device_id, raised_at DESC);


--
-- Name: idx_alarms_active_is_read; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarms_active_is_read ON public.alarms_active USING btree (is_read);


--
-- Name: idx_alarms_active_is_unknown; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarms_active_is_unknown ON public.alarms_active USING btree (is_unknown) WHERE is_unknown;


--
-- Name: idx_alarms_active_raised_at_brin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarms_active_raised_at_brin ON public.alarms_active USING brin (raised_at) WITH (pages_per_range='128');


--
-- Name: idx_alarms_active_severity; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarms_active_severity ON public.alarms_active USING btree (severity);


--
-- Name: idx_alarms_active_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarms_active_status ON public.alarms_active USING btree (status);


--
-- Name: idx_alarms_active_status_severity_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alarms_active_status_severity_time ON public.alarms_active USING btree (status, severity, raised_at DESC);


--
-- Name: idx_api_endpoints_group; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_api_endpoints_group ON public.api_endpoints USING btree (api_group);


--
-- Name: idx_api_endpoints_method; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_api_endpoints_method ON public.api_endpoints USING btree (method);


--
-- Name: idx_api_keys_key_prefix; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_api_keys_key_prefix ON public.api_keys USING btree (key_prefix) WHERE (revoked_at IS NULL);


--
-- Name: idx_api_keys_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_api_keys_user_id ON public.api_keys USING btree (user_id);


--
-- Name: idx_async_jobs_pending_pickup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_async_jobs_pending_pickup ON public.async_jobs USING btree (job_type, status, scheduled_at) WHERE (status = 'pending'::text);


--
-- Name: idx_async_jobs_type_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_async_jobs_type_created ON public.async_jobs USING btree (job_type, created_at DESC);


--
-- Name: idx_async_jobs_zombie_check; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_async_jobs_zombie_check ON public.async_jobs USING btree (status, heartbeat_at) WHERE (status = 'running'::text);


--
-- Name: idx_async_jobs_hourly_failed_recovery; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_async_jobs_hourly_failed_recovery ON public.async_jobs USING btree (bucket_start, recovery_count, last_recovered_at) WHERE ((job_type = 'pm_aggregate_hourly'::text) AND (status = 'failed'::text) AND (bucket_start IS NOT NULL) AND (bucket_end IS NOT NULL));


--
-- Name: idx_audit_logs_action_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_logs_action_time ON public.audit_logs USING btree (action, created_at DESC);


--
-- Name: idx_audit_logs_details_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_logs_details_gin ON public.audit_logs USING gin (details jsonb_path_ops);


--
-- Name: idx_audit_logs_resource; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_logs_resource ON public.audit_logs USING btree (resource, resource_id, created_at DESC);


--
-- Name: idx_audit_logs_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_logs_time ON public.audit_logs USING btree (created_at DESC);


--
-- Name: idx_audit_logs_user_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_logs_user_time ON public.audit_logs USING btree (user_id, created_at DESC);


--
-- Name: idx_backup_restore_file_active_fault_logs; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_backup_restore_file_active_fault_logs ON public.backup_restore_file USING btree (serial_number, update_time, id) WHERE ((is_deleted = false) AND ((object_path ~~ '%/fault/%'::text) OR (object_path ~~ 'fault/%'::text)));


--
-- Name: idx_backup_restore_file_active_station_logs_retention; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_backup_restore_file_active_station_logs_retention ON public.backup_restore_file USING btree (update_time, id) WHERE ((is_deleted = false) AND ((object_path ~~ '%/running/%'::text) OR (object_path ~~ 'running/%'::text) OR (object_path ~~ '%/fault/%'::text) OR (object_path ~~ 'fault/%'::text)));


--
-- Name: idx_backup_restore_file_operator; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_backup_restore_file_operator ON public.backup_restore_file USING btree (operator_code);


--
-- Name: idx_backup_restore_file_sn; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_backup_restore_file_sn ON public.backup_restore_file USING btree (serial_number);


--
-- Name: idx_backup_restore_file_task_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_backup_restore_file_task_id ON public.backup_restore_file USING btree (task_id) WHERE (task_id IS NOT NULL);


--
-- Name: idx_backup_restore_file_update_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_backup_restore_file_update_time ON public.backup_restore_file USING btree (update_time DESC);


--
-- Name: idx_backup_schedules_enabled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_backup_schedules_enabled ON public.backup_schedules USING btree (enabled);


--
-- Name: idx_backup_schedules_target_ids_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_backup_schedules_target_ids_gin ON public.backup_schedules USING gin (target_ids);


--
-- Name: idx_backup_tasks_create_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_backup_tasks_create_user ON public.backup_tasks USING btree (create_user);


--
-- Name: idx_backup_tasks_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_backup_tasks_created ON public.backup_tasks USING btree (created_at DESC);


--
-- Name: idx_backup_tasks_operator_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_backup_tasks_operator_code ON public.backup_tasks USING btree (operator_code);


--
-- Name: idx_backup_tasks_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_backup_tasks_status ON public.backup_tasks USING btree (status);


--
-- Name: idx_backup_tasks_target_ids_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_backup_tasks_target_ids_gin ON public.backup_tasks USING gin (target_ids);


--
-- Name: idx_backup_tasks_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_backup_tasks_type ON public.backup_tasks USING btree (task_type);


--
-- Name: idx_ccp_command_sort; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ccp_command_sort ON public.mml_custom_command_paths USING btree (command_id, sort_order);


--
-- Name: idx_config_backup_sub_tasks_command_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_backup_sub_tasks_command_key ON public.config_backup_sub_tasks USING btree (command_key) WHERE (command_key IS NOT NULL);


--
-- Name: idx_config_backup_sub_tasks_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_backup_sub_tasks_created_at ON public.config_backup_sub_tasks USING btree (created_at DESC);


--
-- Name: idx_config_backup_sub_tasks_device_sn; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_backup_sub_tasks_device_sn ON public.config_backup_sub_tasks USING btree (device_sn);


--
-- Name: idx_config_backup_sub_tasks_task_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_backup_sub_tasks_task_status ON public.config_backup_sub_tasks USING btree (task_id, status);


--
-- Name: idx_config_backup_tasks_due_scheduled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_backup_tasks_due_scheduled ON public.config_backup_tasks USING btree (scheduled_at) WHERE (((create_status)::text = 'timing'::text) AND ((status)::text = 'pending'::text));


--
-- Name: idx_config_baselines_device_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_baselines_device_type ON public.config_baselines USING btree (device_type);


--
-- Name: idx_config_baselines_params_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_baselines_params_gin ON public.config_baselines USING gin (params);


--
-- Name: idx_config_baselines_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_baselines_status ON public.config_baselines USING btree (status);


--
-- Name: idx_config_neighbors_params_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_neighbors_params_gin ON public.config_neighbors USING gin (params);


--
-- Name: idx_config_neighbors_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_neighbors_source ON public.config_neighbors USING btree (source_cell_id);


--
-- Name: idx_config_neighbors_target; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_neighbors_target ON public.config_neighbors USING btree (target_cell_id);


--
-- Name: idx_config_restore_sub_tasks_command_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_restore_sub_tasks_command_key ON public.config_restore_sub_tasks USING btree (command_key) WHERE (command_key IS NOT NULL);


--
-- Name: idx_config_restore_sub_tasks_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_restore_sub_tasks_created_at ON public.config_restore_sub_tasks USING btree (created_at DESC);


--
-- Name: idx_config_restore_sub_tasks_device_sn; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_restore_sub_tasks_device_sn ON public.config_restore_sub_tasks USING btree (device_sn);


--
-- Name: idx_config_restore_sub_tasks_task_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_restore_sub_tasks_task_status ON public.config_restore_sub_tasks USING btree (task_id, status);


--
-- Name: idx_config_restore_tasks_due_scheduled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_restore_tasks_due_scheduled ON public.config_restore_tasks USING btree (scheduled_at) WHERE (((create_status)::text = 'timing'::text) AND ((status)::text = 'pending'::text));


--
-- Name: idx_config_snapshots_enb_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_snapshots_enb_name ON public.config_snapshots USING btree (enb_name);


--
-- Name: idx_config_snapshots_product_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_snapshots_product_type ON public.config_snapshots USING btree (product_type);


--
-- Name: idx_config_snapshots_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_snapshots_source ON public.config_snapshots USING btree (source);


--
-- Name: idx_config_snapshots_update_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_snapshots_update_time ON public.config_snapshots USING btree (update_time DESC);


--
-- Name: idx_config_tasks_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_tasks_created ON public.config_tasks USING btree (created_at DESC);


--
-- Name: idx_config_tasks_device_sns_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_tasks_device_sns_gin ON public.config_tasks USING gin (device_sns);


--
-- Name: idx_config_tasks_params_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_tasks_params_gin ON public.config_tasks USING gin (params);


--
-- Name: idx_config_tasks_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_tasks_status ON public.config_tasks USING btree (status);


--
-- Name: idx_ct_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ct_active ON public.config_templates USING btree (active) WHERE (active = true);


--
-- Name: idx_ct_carrier_tech; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ct_carrier_tech ON public.config_templates USING btree (carrier, technology);


--
-- Name: idx_ct_match; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ct_match ON public.config_templates USING btree (carrier, technology, product_class, template_type) WHERE (active = true);


--
-- Name: idx_ct_parameters_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ct_parameters_gin ON public.config_templates USING gin (parameters);


--
-- Name: idx_ct_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ct_type ON public.config_templates USING btree (template_type);


--
-- Name: idx_dead_letters_module_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dead_letters_module_created ON public.dead_letters USING btree (source_module, created_at DESC);


--
-- Name: idx_dead_letters_subject; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dead_letters_subject ON public.dead_letters USING btree (source_subject);


--
-- Name: idx_device_active_tasks_business; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_active_tasks_business ON public.device_active_tasks USING btree (business_type);


--
-- Name: idx_device_active_tasks_sub_task; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_active_tasks_sub_task ON public.device_active_tasks USING btree (sub_task_id);


--
-- Name: idx_device_groups_source_group_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_groups_source_group_id ON public.device_groups USING btree (source_group_id) WHERE (source_group_id IS NOT NULL);


--
-- Name: idx_device_info_active_alarm_count; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_info_active_alarm_count ON public.device_info USING btree (active_alarm_count) WHERE (active_alarm_count > 0);


--
-- Name: idx_device_info_alarm_severity; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_info_alarm_severity ON public.device_info USING btree (alarm_severity);


--
-- Name: idx_device_info_cell_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_info_cell_status ON public.device_info USING btree (cell_status);


--
-- Name: idx_device_info_device_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_info_device_name ON public.device_info USING btree (device_name) WHERE ((device_name IS NOT NULL) AND ((device_name)::text <> ''::text));


--
-- Name: idx_device_info_device_name_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_info_device_name_trgm ON public.device_info USING gin (device_name public.gin_trgm_ops) WHERE ((device_name IS NOT NULL) AND ((device_name)::text <> ''::text));


--
-- Name: idx_device_info_gps_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_info_gps_status ON public.device_info USING btree (gps_status);


--
-- Name: idx_device_info_last_online_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_info_last_online_time ON public.device_info USING btree (last_online_time);


--
-- Name: idx_device_info_license_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_info_license_status ON public.device_info USING btree (license_status);


--
-- Name: idx_device_info_mac; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_info_mac ON public.device_info USING btree (mac) WHERE ((mac IS NOT NULL) AND ((mac)::text <> ''::text));


--
-- Name: idx_device_info_mac_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_info_mac_trgm ON public.device_info USING gin (mac public.gin_trgm_ops) WHERE ((mac IS NOT NULL) AND ((mac)::text <> ''::text));


--
-- Name: idx_device_info_name_sync_pending; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_info_name_sync_pending ON public.device_info USING btree (device_id) WHERE (name_sync_pending = true);


--
-- Name: idx_device_info_op_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_info_op_state ON public.device_info USING btree (op_state);


--
-- Name: idx_device_info_pci; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_info_pci ON public.device_info USING btree (pci) WHERE ((pci IS NOT NULL) AND ((pci)::text <> ''::text));


--
-- Name: idx_device_info_pci_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_info_pci_trgm ON public.device_info USING gin (pci public.gin_trgm_ops) WHERE ((pci IS NOT NULL) AND ((pci)::text <> ''::text));


--
-- Name: idx_device_info_project_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_info_project_status ON public.device_info USING btree (project_status);


--
-- Name: idx_device_info_rf_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_info_rf_status ON public.device_info USING btree (rf_status);


--
-- Name: idx_device_info_ue_count; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_info_ue_count ON public.device_info USING btree (ue_count);


--
-- Name: idx_device_licenses_enb_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_licenses_enb_name ON public.device_licenses USING btree (enb_name);


--
-- Name: idx_device_licenses_product_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_licenses_product_type ON public.device_licenses USING btree (product_type);


--
-- Name: idx_device_licenses_update_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_licenses_update_time ON public.device_licenses USING btree (update_time DESC);


--
-- Name: idx_device_location_observations_observed_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_location_observations_observed_at ON public.device_location_observations USING btree (device_id, observed_at DESC);


--
-- Name: idx_devices_other_carrier_lifecycle; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_other_carrier_lifecycle ON public.devices_other USING btree (carrier, lifecycle_state);


--
-- Name: idx_devices_other_carrier_tech; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_other_carrier_tech ON public.devices_other USING btree (carrier, technology);


--
-- Name: idx_devices_other_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_other_deleted_at ON public.devices_other USING btree (deleted_at) WHERE (deleted_at IS NOT NULL);


--
-- Name: idx_devices_other_is_online; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_other_is_online ON public.devices_other USING btree (is_online) WHERE (is_online = true);


--
-- Name: idx_devices_other_last_inform; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_other_last_inform ON public.devices_other USING btree (last_inform_at);


--
-- Name: idx_devices_other_lifecycle_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_other_lifecycle_state ON public.devices_other USING btree (lifecycle_state);


--
-- Name: idx_devices_other_oui; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_other_oui ON public.devices_other USING btree (oui);


--
-- Name: idx_devices_other_serial_number; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_devices_other_serial_number ON public.devices_other USING btree (serial_number);


--
-- Name: idx_dg_carrier; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dg_carrier ON public.device_groups USING btree (carrier) WHERE (carrier IS NOT NULL);


--
-- Name: idx_dg_matching_mode; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dg_matching_mode ON public.device_groups USING btree (matching_mode) WHERE (matching_mode IS NOT NULL);


--
-- Name: idx_dg_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dg_name ON public.device_groups USING btree (name);


--
-- Name: idx_dg_name_parent; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_dg_name_parent ON public.device_groups USING btree (name, COALESCE(parent_id, '00000000-0000-0000-0000-000000000000'::uuid));


--
-- Name: idx_dg_parent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dg_parent ON public.device_groups USING btree (parent_id);


--
-- Name: idx_dg_serial_number_list; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dg_serial_number_list ON public.device_groups USING gin (serial_number_list) WHERE (serial_number_list IS NOT NULL);


--
-- Name: idx_dgm_device; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dgm_device ON public.device_group_members USING btree (device_id);


--
-- Name: idx_dgm_group; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dgm_group ON public.device_group_members USING btree (group_id);


--
-- Name: idx_dict_detail_parent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dict_detail_parent ON public.sys_dictionary_details USING btree (parent_id) WHERE (parent_id IS NOT NULL);


--
-- Name: idx_discovered_product_swver_private_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_discovered_product_swver_private_active ON public.discovered_param_mappings USING btree (product_id, software_version, private_path) WHERE is_active;


--
-- Name: idx_dr_batch; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dr_batch ON public.device_registrations USING btree (import_batch_id) WHERE (import_batch_id IS NOT NULL);


--
-- Name: idx_dr_group; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dr_group ON public.device_registrations USING btree (group_id);


--
-- Name: idx_dr_sn_pending; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_dr_sn_pending ON public.device_registrations USING btree (serial_number) WHERE ((status)::text = 'pending'::text);


--
-- Name: idx_dr_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dr_status ON public.device_registrations USING btree (status);


--
-- Name: idx_event_logs_device_sn_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_event_logs_device_sn_time ON public.event_logs USING btree (device_sn, occurred_at DESC);


--
-- Name: idx_event_logs_event_type_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_event_logs_event_type_time ON public.event_logs USING btree (event_type, occurred_at DESC);


--
-- Name: idx_event_logs_occurred_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_event_logs_occurred_at ON public.event_logs USING btree (occurred_at DESC);


--
-- Name: idx_fault_log_collect_sub_tasks_command_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_fault_log_collect_sub_tasks_command_key ON public.fault_log_collect_sub_tasks USING btree (command_key) WHERE (command_key IS NOT NULL);


--
-- Name: idx_fault_log_collect_sub_tasks_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_fault_log_collect_sub_tasks_created_at ON public.fault_log_collect_sub_tasks USING btree (created_at DESC);


--
-- Name: idx_fault_log_collect_sub_tasks_device_sn; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_fault_log_collect_sub_tasks_device_sn ON public.fault_log_collect_sub_tasks USING btree (device_sn);


--
-- Name: idx_fault_log_collect_sub_tasks_task_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_fault_log_collect_sub_tasks_task_status ON public.fault_log_collect_sub_tasks USING btree (task_id, status);


--
-- Name: idx_fault_log_collect_tasks_due_scheduled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_fault_log_collect_tasks_due_scheduled ON public.fault_log_collect_tasks USING btree (scheduled_at) WHERE (((create_status)::text = 'timing'::text) AND ((status)::text = 'pending'::text));


--
-- Name: idx_firmware_file_type_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_firmware_file_type_status ON public.firmware_versions USING btree (file_type, status);


--
-- Name: idx_firmware_unique_version; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_firmware_unique_version ON public.firmware_versions USING btree (product_id, version, file_type);


--
-- Name: idx_firmware_versions_compatible_oui_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_firmware_versions_compatible_oui_gin ON public.firmware_versions USING gin (compatible_oui);


--
-- Name: idx_firmware_versions_product_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_firmware_versions_product_id ON public.firmware_versions USING btree (product_id);


--
-- Name: idx_firmware_versions_product_ids_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_firmware_versions_product_ids_gin ON public.firmware_versions USING gin (product_ids);


--
-- Name: idx_ftp_configs_enabled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ftp_configs_enabled ON public.ftp_configs USING btree (enabled);


--
-- Name: idx_indicator_group_enb_parent_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_indicator_group_enb_parent_id ON public.indicator_group_enb USING btree (parent_id);


--
-- Name: idx_indicator_group_gnb_parent_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_indicator_group_gnb_parent_id ON public.indicator_group_gnb USING btree (parent_id);


--
-- Name: idx_indicator_group_gsm_parent_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_indicator_group_gsm_parent_id ON public.indicator_group_gsm USING btree (parent_id);


--
-- Name: idx_indicator_threshold_indicator_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_indicator_threshold_indicator_id ON public.indicator_threshold USING btree (indicator_id);


--
-- Name: idx_kpi_definitions_counters_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_kpi_definitions_counters_gin ON public.kpi_definitions USING gin (counters);


--
-- Name: idx_kpi_thresholds_carrier; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_kpi_thresholds_carrier ON public.kpi_thresholds USING btree (carrier);


--
-- Name: idx_kpi_thresholds_enabled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_kpi_thresholds_enabled ON public.kpi_thresholds USING btree (enabled);


--
-- Name: idx_kpi_thresholds_kpi_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_kpi_thresholds_kpi_name ON public.kpi_thresholds USING btree (kpi_name);


--
-- Name: idx_login_logs_login_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_login_logs_login_at ON public.sys_login_logs USING btree (login_at);


--
-- Name: idx_login_logs_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_login_logs_user_id ON public.sys_login_logs USING btree (user_id) WHERE (user_id IS NOT NULL);


--
-- Name: idx_managed_files_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_managed_files_created ON public.managed_files USING btree (created_at DESC);


--
-- Name: idx_managed_files_device; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_managed_files_device ON public.managed_files USING btree (device_sn);


--
-- Name: idx_managed_files_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_managed_files_status ON public.managed_files USING btree (status);


--
-- Name: idx_managed_files_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_managed_files_type ON public.managed_files USING btree (file_type);


--
-- Name: idx_managed_files_type_status_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_managed_files_type_status_time ON public.managed_files USING btree (file_type, status, created_at DESC);


--
-- Name: idx_menus_parent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_menus_parent ON public.menus USING btree (parent_id) WHERE (parent_id IS NOT NULL);


--
-- Name: idx_menus_permission_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_menus_permission_key ON public.menus USING btree (permission_key);


--
-- Name: idx_menus_sort; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_menus_sort ON public.menus USING btree (parent_id NULLS FIRST, sort_order);


--
-- Name: idx_menus_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_menus_status ON public.menus USING btree (status);


--
-- Name: idx_menus_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_menus_type ON public.menus USING btree (type);


--
-- Name: idx_mml_audit_log_command_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_audit_log_command_code ON public.mml_audit_log USING btree (command_code) WHERE (command_code IS NOT NULL);


--
-- Name: idx_mml_audit_log_device_sn_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_audit_log_device_sn_created ON public.mml_audit_log USING btree (device_sn, created_at DESC);


--
-- Name: idx_mml_audit_log_task_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_audit_log_task_id ON public.mml_audit_log USING btree (task_id) WHERE (task_id IS NOT NULL);


--
-- Name: idx_mml_catalog_link_health_unresolved; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_catalog_link_health_unresolved ON public.mml_catalog_link_health USING btree (spec_version, detected_at) WHERE (resolved_at IS NULL);


--
-- Name: idx_mml_command_sub_fields_command; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_command_sub_fields_command ON public.mml_command_sub_fields USING btree (command_id, sort_order);


--
-- Name: idx_mml_command_sub_fields_std_path; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_command_sub_fields_std_path ON public.mml_command_sub_fields USING btree (standard_path_id);


--
-- Name: idx_mml_commands_category; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_commands_category ON public.mml_commands USING btree (category);


--
-- Name: idx_mml_commands_deprecated; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_commands_deprecated ON public.mml_commands USING btree (deprecated_at) WHERE (deprecated_at IS NOT NULL);


--
-- Name: idx_mml_commands_group_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_commands_group_id ON public.mml_commands USING btree (group_id);


--
-- Name: idx_mml_commands_operation_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_commands_operation_type ON public.mml_commands USING btree (operation_type);


--
-- Name: idx_mml_commands_platform_tags_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_commands_platform_tags_gin ON public.mml_commands USING gin (platform_tags);


--
-- Name: idx_mml_commands_target_paths_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_commands_target_paths_gin ON public.mml_commands USING gin (target_paths);


--
-- Name: idx_mml_commands_tree_node_refs_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_commands_tree_node_refs_gin ON public.mml_commands USING gin (tree_node_refs);


--
-- Name: idx_mml_custom_command_command_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_custom_command_command_code ON public.mml_custom_command USING btree (command_code);


--
-- Name: idx_mml_custom_command_owner; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_custom_command_owner ON public.mml_custom_command USING btree (owner_user_id) WHERE (owner_user_id IS NOT NULL);


--
-- Name: idx_mml_custom_command_parameters_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_custom_command_parameters_gin ON public.mml_custom_command USING gin (parameters);


--
-- Name: idx_mml_custom_command_scope_creator; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_custom_command_scope_creator ON public.mml_custom_command USING btree (command_scope, creator);


--
-- Name: idx_mml_param_groups_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_param_groups_active ON public.mml_command_groups USING btree (param_version, is_active) WHERE (is_active = true);


--
-- Name: idx_mml_param_groups_chapter; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_param_groups_chapter ON public.mml_command_groups USING btree (chapter_code, display_order) WHERE (chapter_code IS NOT NULL);


--
-- Name: idx_mml_param_groups_deleted; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_param_groups_deleted ON public.mml_command_groups USING btree (deleted_at) WHERE (deleted_at IS NOT NULL);


--
-- Name: idx_mml_param_groups_deprecated; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_param_groups_deprecated ON public.mml_command_groups USING btree (deprecated_at) WHERE (deprecated_at IS NOT NULL);


--
-- Name: idx_mml_param_groups_family_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_param_groups_family_code ON public.mml_command_groups USING btree (family_code) WHERE (((family_code)::text <> ''::text) AND (deleted_at IS NULL));


--
-- Name: idx_mml_param_groups_path; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_param_groups_path ON public.mml_command_groups USING gist (path);


--
-- Name: idx_mml_param_groups_version; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_param_groups_version ON public.mml_command_groups USING btree (param_version);


--
-- Name: idx_mml_scripts_content_sha256; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_scripts_content_sha256 ON public.mml_scripts USING btree (content_sha256);


--
-- Name: idx_mml_scripts_last_run_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_scripts_last_run_at ON public.mml_scripts USING btree (last_run_at DESC) WHERE (last_run_at IS NOT NULL);


--
-- Name: idx_mml_scripts_plan_items_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_scripts_plan_items_gin ON public.mml_scripts USING gin (plan_items jsonb_path_ops);


--
-- Name: idx_mml_scripts_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_scripts_status ON public.mml_scripts USING btree (status);


--
-- Name: idx_mml_scripts_tags_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_scripts_tags_gin ON public.mml_scripts USING gin (tags);


--
-- Name: idx_mml_scripts_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_scripts_type ON public.mml_scripts USING btree (type);


--
-- Name: idx_mml_sub_fields_supported; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_sub_fields_supported ON public.mml_command_sub_fields USING btree (command_id, sort_order) WHERE (is_supported = true);


--
-- Name: idx_mml_tasks_commands_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_tasks_commands_gin ON public.mml_tasks USING gin (commands jsonb_path_ops);


--
-- Name: idx_mml_tasks_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_tasks_created ON public.mml_tasks USING btree (created_at DESC);


--
-- Name: idx_mml_tasks_creator_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_tasks_creator_time ON public.mml_tasks USING btree (creator, created_at DESC) WHERE (creator IS NOT NULL);


--
-- Name: idx_mml_tasks_device_sns_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_tasks_device_sns_gin ON public.mml_tasks USING gin (device_sns);


--
-- Name: idx_mml_tasks_next_trigger; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_tasks_next_trigger ON public.mml_tasks USING btree (next_trigger_at) WHERE ((next_trigger_at IS NOT NULL) AND ((status)::text = 'pending'::text));


--
-- Name: idx_mml_tasks_parent_task; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_tasks_parent_task ON public.mml_tasks USING btree (parent_task_id) WHERE (parent_task_id IS NOT NULL);


--
-- Name: idx_mml_tasks_plan_items_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_tasks_plan_items_gin ON public.mml_tasks USING gin (plan_items jsonb_path_ops);


--
-- Name: idx_mml_tasks_results_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_tasks_results_gin ON public.mml_tasks USING gin (results jsonb_path_ops);


--
-- Name: idx_mml_tasks_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_tasks_status ON public.mml_tasks USING btree (status);


--
-- Name: idx_mml_templates_scope_group; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mml_templates_scope_group ON public.mml_custom_command USING btree (command_scope, category_group);


--
-- Name: idx_model_upload_intents_dispatch; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_model_upload_intents_dispatch ON public.model_upload_intents USING btree (status, next_attempt_at, created_at);


--
-- Name: idx_mr_indicators_category; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mr_indicators_category ON public.mr_indicators USING btree (category);


--
-- Name: idx_mr_indicators_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mr_indicators_code ON public.mr_indicators USING btree (indicator_code);


--
-- Name: idx_mr_mappings_device; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mr_mappings_device ON public.mr_device_mappings USING btree (device_sn);


--
-- Name: idx_mr_mappings_enabled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mr_mappings_enabled ON public.mr_device_mappings USING btree (enabled);


--
-- Name: idx_mr_progress_cell; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mr_progress_cell ON public.mr_customize_task_progress USING btree (small_cell_code);


--
-- Name: idx_mr_progress_health; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mr_progress_health ON public.mr_customize_task_progress USING btree (health_status) WHERE ((progress_status)::text = 'openSuccess'::text);


--
-- Name: idx_mr_progress_task_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mr_progress_task_status ON public.mr_customize_task_progress USING btree (task_id, progress_status);


--
-- Name: idx_mr_task_status_endtime; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mr_task_status_endtime ON public.mr_customize_task USING btree (task_status, end_time) WHERE (end_time IS NOT NULL);


--
-- Name: idx_mr_task_status_starttime; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mr_task_status_starttime ON public.mr_customize_task USING btree (task_status, start_time);


--
-- Name: idx_ne_message_logs_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ne_message_logs_created_at ON public.ne_message_logs USING btree (created_at DESC);


--
-- Name: idx_ne_message_logs_created_at_brin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ne_message_logs_created_at_brin ON public.ne_message_logs USING brin (created_at) WITH (pages_per_range='128');


--
-- Name: idx_ne_message_logs_device_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ne_message_logs_device_id ON public.ne_message_logs USING btree (device_id);


--
-- Name: idx_ne_message_logs_device_sn; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ne_message_logs_device_sn ON public.ne_message_logs USING btree (device_sn);


--
-- Name: idx_nedirect_commands_device_sn; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_nedirect_commands_device_sn ON public.nedirect_commands USING btree (device_sn);


--
-- Name: idx_nedirect_commands_session_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_nedirect_commands_session_id ON public.nedirect_commands USING btree (session_id);


--
-- Name: idx_nedirect_commands_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_nedirect_commands_status ON public.nedirect_commands USING btree (status);


--
-- Name: idx_nedirect_sessions_device_sn; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_nedirect_sessions_device_sn ON public.nedirect_sessions USING btree (device_sn);


--
-- Name: idx_nedirect_sessions_device_user_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_nedirect_sessions_device_user_status ON public.nedirect_sessions USING btree (device_sn, user_id, status);


--
-- Name: idx_nedirect_sessions_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_nedirect_sessions_status ON public.nedirect_sessions USING btree (status);


--
-- Name: idx_nedirect_sessions_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_nedirect_sessions_user_id ON public.nedirect_sessions USING btree (user_id);


--
-- Name: idx_northbound_endpoints_protocol_purpose; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_northbound_endpoints_protocol_purpose ON public.northbound_endpoints USING btree (protocol, purpose);


--
-- Name: idx_northbound_endpoints_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_northbound_endpoints_status ON public.northbound_endpoints USING btree (status, enabled);


--
-- Name: idx_northbound_field_mappings_profile; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_northbound_field_mappings_profile ON public.northbound_field_mappings USING btree (profile_kind, profile_code, domain, object_code, sort_order);


--
-- Name: idx_northbound_file_profiles_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_northbound_file_profiles_status ON public.northbound_file_profiles USING btree (status, enabled);


--
-- Name: idx_northbound_file_runs_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_northbound_file_runs_created_at ON public.northbound_file_runs USING btree (created_at DESC);


--
-- Name: idx_northbound_file_runs_profile; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_northbound_file_runs_profile ON public.northbound_file_runs USING btree (profile_kind, profile_code, created_at DESC);


--
-- Name: idx_northbound_file_runs_schedule; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_northbound_file_runs_schedule ON public.northbound_file_runs USING btree (profile_kind, profile_code, group_id, window_end DESC, status);


--
-- Name: idx_northbound_file_runs_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_northbound_file_runs_status ON public.northbound_file_runs USING btree (status, created_at DESC);


--
-- Name: idx_northbound_delivery_targets_scope; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_northbound_delivery_targets_scope ON public.northbound_delivery_targets USING btree (scope, owner_code, enabled);


--
-- Name: idx_northbound_snmp_alarm_targets_enabled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_northbound_snmp_alarm_targets_enabled ON public.northbound_snmp_alarm_targets USING btree (enabled, version);


--
-- Name: idx_northbound_socket_alarm_configs_enabled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_northbound_socket_alarm_configs_enabled ON public.northbound_socket_alarm_configs USING btree (enabled, profile);


--
-- Name: idx_northbound_api_configs_path; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_northbound_api_configs_path ON public.northbound_api_configs USING btree (method, path, enabled);


--
-- Name: idx_northbound_api_clients_enabled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_northbound_api_clients_enabled ON public.northbound_api_clients USING btree (enabled, client_key);


--
-- Name: idx_northbound_page_config_events_lookup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_northbound_page_config_events_lookup ON public.northbound_page_config_events USING btree (capability, owner_code, target_key, event_type, created_at DESC);


--
-- Name: idx_northbound_page_config_events_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_northbound_page_config_events_status ON public.northbound_page_config_events USING btree (status, created_at DESC);


--
-- Name: idx_northbound_inventory_profiles_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_northbound_inventory_profiles_status ON public.northbound_inventory_profiles USING btree (status, enabled);


--
-- Name: idx_northbound_outbox_dead; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_northbound_outbox_dead ON public.northbound_outbox USING btree (created_at DESC) WHERE (status = 'dead'::text);


--
-- Name: idx_northbound_outbox_event_target; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_northbound_outbox_event_target ON public.northbound_outbox USING btree (event_id, target_id);


--
-- Name: idx_northbound_outbox_status_retry; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_northbound_outbox_status_retry ON public.northbound_outbox USING btree (status, next_retry_at) WHERE (status = ANY (ARRAY['pending'::text, 'processing'::text]));


--
-- Name: idx_notification_history_alarm_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_history_alarm_id ON public.notification_history USING btree (alarm_id);


--
-- Name: idx_notification_history_channel_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_history_channel_created ON public.notification_history USING btree (channel, created_at DESC);


--
-- Name: idx_notification_history_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_history_status ON public.notification_history USING btree (status);


--
-- Name: idx_notification_history_template_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_history_template_id ON public.notification_history USING btree (template_id);


--
-- Name: idx_notification_templates_channel; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_templates_channel ON public.notification_templates USING btree (channel);


--
-- Name: idx_notification_templates_enabled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_templates_enabled ON public.notification_templates USING btree (enabled);


--
-- Name: idx_notifications_dedup_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notifications_dedup_key ON public.notifications USING btree (dedup_key) WHERE (dedup_key IS NOT NULL);


--
-- Name: idx_notifications_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notifications_type ON public.notifications USING btree (type, created_at DESC);


--
-- Name: idx_notifications_type_priority_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notifications_type_priority_time ON public.notifications USING btree (type, priority, created_at DESC);


--
-- Name: idx_notifications_user_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notifications_user_status ON public.notifications USING btree (user_id, status, created_at DESC);


--
-- Name: idx_notifications_user_unread; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notifications_user_unread ON public.notifications USING btree (user_id, is_read, created_at DESC);


--
-- Name: idx_oper_logs_action; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oper_logs_action ON public.sys_oper_logs USING btree (action);


--
-- Name: idx_oper_logs_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oper_logs_created_at ON public.sys_oper_logs USING btree (created_at);


--
-- Name: idx_oper_logs_module; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oper_logs_module ON public.sys_oper_logs USING btree (module);


--
-- Name: idx_oper_logs_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oper_logs_user_id ON public.sys_oper_logs USING btree (user_id) WHERE (user_id IS NOT NULL);


--
-- Name: idx_ops_audit_bg; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_audit_bg ON public.ops_audit_logs USING btree (break_glass) WHERE (break_glass = true);


--
-- Name: idx_ops_audit_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_audit_created ON public.ops_audit_logs USING btree (created_at DESC);


--
-- Name: idx_ops_audit_operator; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_audit_operator ON public.ops_audit_logs USING btree (operator_user_id);


--
-- Name: idx_ops_audit_target; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_audit_target ON public.ops_audit_logs USING btree (target_type, target_id);


--
-- Name: idx_ops_cmd_records_device; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_cmd_records_device ON public.ops_command_records USING btree (device_sn);


--
-- Name: idx_ops_cmd_records_operator; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_cmd_records_operator ON public.ops_command_records USING btree (operator);


--
-- Name: idx_ops_cmd_records_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_cmd_records_time ON public.ops_command_records USING btree (execute_time DESC);


--
-- Name: idx_ops_diag_device; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_diag_device ON public.ops_diagnostics USING btree (device_sn);


--
-- Name: idx_ops_diag_started; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_diag_started ON public.ops_diagnostics USING btree (started_at DESC);


--
-- Name: idx_ops_diag_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_diag_type ON public.ops_diagnostics USING btree (diag_type);


--
-- Name: idx_ops_dl_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_dl_created ON public.ops_downloads USING btree (created_at DESC);


--
-- Name: idx_ops_dl_device; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_dl_device ON public.ops_downloads USING btree (device_sn);


--
-- Name: idx_ops_dl_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_dl_type ON public.ops_downloads USING btree (content_type);


--
-- Name: idx_ops_mw_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_mw_active ON public.ops_maintenance_windows USING btree (start_at, end_at) WHERE ((status)::text = 'active'::text);


--
-- Name: idx_ops_mw_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_mw_status ON public.ops_maintenance_windows USING btree (status);


--
-- Name: idx_ops_pb_use_count; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_pb_use_count ON public.ops_playbooks USING btree (use_count DESC);


--
-- Name: idx_ops_task_exec_device; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_task_exec_device ON public.ops_task_executions USING btree (device_sn);


--
-- Name: idx_ops_task_exec_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_task_exec_status ON public.ops_task_executions USING btree (status);


--
-- Name: idx_ops_task_exec_task; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_task_exec_task ON public.ops_task_executions USING btree (task_id);


--
-- Name: idx_ops_tasks_approval_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_tasks_approval_state ON public.ops_tasks USING btree (approval_state) WHERE ((approval_state)::text = 'pending'::text);


--
-- Name: idx_ops_tasks_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_tasks_created ON public.ops_tasks USING btree (created_at DESC);


--
-- Name: idx_ops_tasks_device_sns_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_tasks_device_sns_gin ON public.ops_tasks USING gin (device_sns);


--
-- Name: idx_ops_tasks_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_tasks_status ON public.ops_tasks USING btree (status);


--
-- Name: idx_ops_tasks_template; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_tasks_template ON public.ops_tasks USING btree (template_id);


--
-- Name: idx_ops_templates_category; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_templates_category ON public.ops_templates USING btree (category);


--
-- Name: idx_ops_templates_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_templates_created ON public.ops_templates USING btree (created_at DESC);


--
-- Name: idx_ops_templates_risk_level; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_templates_risk_level ON public.ops_templates USING btree (risk_level);


--
-- Name: idx_ops_templates_steps_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_templates_steps_gin ON public.ops_templates USING gin (steps jsonb_path_ops);


--
-- Name: idx_ops_templates_tags_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_templates_tags_gin ON public.ops_templates USING gin (tags);


--
-- Name: idx_ops_templates_target_device_types_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ops_templates_target_device_types_gin ON public.ops_templates USING gin (target_device_types);


--
-- Name: idx_orphan_paths_disposition_null; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_orphan_paths_disposition_null ON public.mml_catalog_orphan_paths_audit_t0171 USING btree (disposition) WHERE (disposition IS NULL);


--
-- Name: idx_orphan_paths_object_prefix; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_orphan_paths_object_prefix ON public.mml_catalog_orphan_paths_audit_t0171 USING btree (object_prefix);


--
-- Name: idx_param_mappings_model_private_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_param_mappings_model_private_active ON public.param_mappings USING btree (param_model_id, private_path) WHERE is_active;


--
-- Name: idx_param_mappings_model_standard; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_param_mappings_model_standard ON public.param_mappings USING btree (param_model_id, standard_path);


--
-- Name: idx_param_mappings_storable; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_param_mappings_storable ON public.param_mappings USING btree (param_model_id) WHERE is_storable;


--
-- Name: idx_param_models_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_param_models_active ON public.param_models USING btree (is_active);


--
-- Name: idx_parameter_sync_admission_reservations_bucket; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_admission_reservations_bucket ON public.parameter_sync_admission_reservations USING btree (admission_class, bucket_id, status);


--
-- Name: idx_parameter_sync_admission_reservations_due; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_admission_reservations_due ON public.parameter_sync_admission_reservations USING btree (status, lease_until, updated_at);


--
-- Name: idx_parameter_sync_bindings_run; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_bindings_run ON public.parameter_sync_request_bindings USING btree (run_id, status);


--
-- Name: idx_parameter_sync_device_state_due; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_device_state_due ON public.parameter_sync_device_state USING btree (next_auto_sync_at) WHERE (next_auto_sync_at IS NOT NULL);


--
-- Name: idx_parameter_sync_event_failures_replay; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_event_failures_replay ON public.parameter_sync_event_failures USING btree (status, next_retry_at, created_at);


--
-- Name: idx_parameter_sync_event_failures_run_task; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_event_failures_run_task ON public.parameter_sync_event_failures USING btree (run_id, task_id) WHERE ((run_id IS NOT NULL) OR (task_id IS NOT NULL));


--
-- Name: idx_parameter_sync_outbox_dispatch; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_outbox_dispatch ON public.parameter_sync_outbox USING btree (status, next_attempt_at, created_at) WHERE ((status)::text = ANY ((ARRAY['pending'::character varying, 'failed'::character varying])::text[]));


--
-- Name: idx_parameter_sync_outbox_ready_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_outbox_ready_created ON public.parameter_sync_outbox USING btree (created_at, id) WHERE ((status)::text = ANY ((ARRAY['pending'::character varying, 'failed'::character varying])::text[]));

CREATE INDEX idx_parameter_sync_outbox_delivering_updated ON public.parameter_sync_outbox USING btree (updated_at) WHERE ((status)::text = 'delivering'::text);

CREATE INDEX idx_parameter_sync_outbox_terminal_status ON public.parameter_sync_outbox USING btree (status) WHERE ((status)::text = ANY ((ARRAY['delivered'::character varying, 'dead'::character varying])::text[]));


--
-- Name: idx_parameter_sync_recovery_claim; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_recovery_claim ON public.parameter_sync_recovery_state USING btree (status, next_retry_at, lease_until, updated_at);


--
-- Name: idx_parameter_sync_requests_device_history; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_requests_device_history ON public.parameter_sync_requests USING btree (device_id, created_at DESC);


--
-- Name: idx_parameter_sync_requests_schedule; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_requests_schedule ON public.parameter_sync_requests USING btree (status, priority, next_attempt_at, created_at);

CREATE INDEX idx_parameter_sync_requests_auto_backpressure ON public.parameter_sync_requests USING btree (admission_queued_at) WHERE (((status)::text = 'queued'::text) AND ((result_code)::text = 'AUTOMATIC_BACKPRESSURE'::text));


--
-- Name: idx_parameter_sync_requests_source_event; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_requests_source_event ON public.parameter_sync_requests USING btree (source_event_id) WHERE (source_event_id IS NOT NULL);


--
-- Name: idx_parameter_sync_runs_device_history; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_runs_device_history ON public.parameter_sync_runs USING btree (device_id, started_at DESC);


--
-- Name: idx_parameter_sync_runs_device_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_runs_device_status ON public.parameter_sync_runs USING btree (device_id, status, started_at DESC);


--
-- Name: idx_parameter_sync_runs_active_convergence; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_runs_active_convergence ON public.parameter_sync_runs USING btree (started_at, id) WHERE ((status)::text = ANY ((ARRAY['planning'::character varying, 'enqueuing'::character varying, 'waiting_device'::character varying, 'executing'::character varying, 'processing'::character varying, 'cancelling'::character varying])::text[]));


--
-- Name: idx_parameter_sync_runs_projection_due; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_runs_projection_due ON public.parameter_sync_runs USING btree (projection_next_attempt_at, completed_at, id) WHERE (((status)::text = 'succeeded'::text) AND ((sync_scope)::text = 'full'::text) AND ((projection_status)::text <> 'completed'::text));


--
-- Name: idx_parameter_sync_runs_projection_pending; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_runs_projection_pending ON public.parameter_sync_runs USING btree (completed_at, id) WHERE (((status)::text = 'succeeded'::text) AND ((sync_scope)::text = 'full'::text) AND ((projection_status)::text <> 'completed'::text));


--
-- Name: idx_parameter_sync_runs_request; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_runs_request ON public.parameter_sync_runs USING btree (request_id);


--
-- Name: idx_parameter_sync_task_results_event; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_task_results_event ON public.parameter_sync_task_results USING btree (event_id);


--
-- Name: idx_parameter_sync_task_results_status_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_parameter_sync_task_results_status_time ON public.parameter_sync_task_results USING btree (status, created_at, processed_at);


--
-- Name: idx_pdl_device_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pdl_device_id ON public.parameter_discovery_log USING btree (device_id);


--
-- Name: idx_pdl_device_sn; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pdl_device_sn ON public.parameter_discovery_log USING btree (device_sn);


--
-- Name: idx_pdl_param_model; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pdl_param_model ON public.parameter_discovery_log USING btree (param_model_id) WHERE (param_model_id IS NOT NULL);


--
-- Name: idx_pdl_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pdl_status ON public.parameter_discovery_log USING btree (status);


--
-- Name: idx_perf_alarm_threshold_temp_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_perf_alarm_threshold_temp_id ON public.perf_alarm_threshold USING btree (temp_id);


--
-- Name: idx_perf_indicators_enb_group_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_perf_indicators_enb_group_id ON public.perf_indicators_enb USING btree (group_id);


--
-- Name: idx_perf_indicators_enb_is_build_in; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_perf_indicators_enb_is_build_in ON public.perf_indicators_enb USING btree (is_build_in);


--
-- Name: idx_perf_indicators_enb_loaded_from; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_perf_indicators_enb_loaded_from ON public.perf_indicators_enb USING btree (loaded_from);


--
-- Name: idx_perf_indicators_gnb_group_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_perf_indicators_gnb_group_id ON public.perf_indicators_gnb USING btree (group_id);


--
-- Name: idx_perf_indicators_gnb_is_build_in; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_perf_indicators_gnb_is_build_in ON public.perf_indicators_gnb USING btree (is_build_in);


--
-- Name: idx_perf_indicators_gnb_loaded_from; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_perf_indicators_gnb_loaded_from ON public.perf_indicators_gnb USING btree (loaded_from);


--
-- Name: idx_perf_indicators_gsm_group_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_perf_indicators_gsm_group_id ON public.perf_indicators_gsm USING btree (group_id);


--
-- Name: idx_perf_indicators_gsm_is_build_in; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_perf_indicators_gsm_is_build_in ON public.perf_indicators_gsm USING btree (is_build_in);


--
-- Name: idx_perf_indicators_gsm_loaded_from; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_perf_indicators_gsm_loaded_from ON public.perf_indicators_gsm USING btree (loaded_from);


--
-- Name: idx_perf_template_rel_arithmetic_temp_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_perf_template_rel_arithmetic_temp_id ON public.perf_template_rel_arithmetic USING btree (temp_id);


--
-- Name: idx_pm_adhoc_task_runs_task_started; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pm_adhoc_task_runs_task_started ON public.pm_adhoc_task_runs USING btree (task_id, started_at DESC);


--
-- Name: idx_pm_dashboards_is_builtin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pm_dashboards_is_builtin ON public.pm_dashboards USING btree (is_builtin) WHERE (is_builtin = true);


--
-- Name: idx_pm_dashboards_owner; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pm_dashboards_owner ON public.pm_dashboards USING btree (owner_id);


--
-- Name: idx_pm_dashboards_parent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pm_dashboards_parent ON public.pm_dashboards USING btree (parent_dashboard_id) WHERE (parent_dashboard_id IS NOT NULL);


--
-- Name: idx_pm_dashboards_shared; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pm_dashboards_shared ON public.pm_dashboards USING gin (shared_with);


--
-- Name: idx_pm_kpi_export_tasks_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pm_kpi_export_tasks_created_at ON public.pm_kpi_export_tasks USING btree (created_at DESC);


--
-- Name: idx_pm_kpi_export_tasks_files; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pm_kpi_export_tasks_files ON public.pm_kpi_export_tasks USING btree (created_at DESC) WHERE ((status = 'succeeded'::text) AND (file_path <> ''::text));


--
-- Name: idx_pm_panels_adhoc_task; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pm_panels_adhoc_task ON public.pm_panels USING btree (adhoc_task_id) WHERE (adhoc_task_id IS NOT NULL);


--
-- Name: idx_pm_panels_dashboard; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pm_panels_dashboard ON public.pm_panels USING btree (dashboard_id);


--
-- Name: idx_pm_query_templates_visibility_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pm_query_templates_visibility_created ON public.pm_query_templates USING btree (visibility, created_at DESC);


--
-- Name: idx_pm_tasks_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pm_tasks_created ON public.pm_tasks USING btree (created_at DESC);


--
-- Name: idx_pm_tasks_device_sns_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pm_tasks_device_sns_gin ON public.pm_tasks USING gin (device_sns);


--
-- Name: idx_pm_tasks_kpi_codes_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pm_tasks_kpi_codes_gin ON public.pm_tasks USING gin (kpi_codes);


--
-- Name: idx_pm_tasks_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pm_tasks_status ON public.pm_tasks USING btree (status);


--
-- Name: idx_pm_tasks_subtype_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pm_tasks_subtype_status ON public.pm_tasks USING btree (task_subtype, status) WHERE (task_subtype IS NOT NULL);


--
-- Name: idx_pm_tasks_adhoc_visibility_creator; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pm_tasks_adhoc_visibility_creator ON public.pm_tasks USING btree (is_builtin, visibility, creator, task_name) WHERE (task_subtype = 'adhoc_aggregation'::text);


--
-- Name: idx_product_class_patterns_product; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_product_class_patterns_product ON public.product_class_patterns USING btree (product_id);


--
-- Name: idx_product_class_patterns_sort; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_product_class_patterns_sort ON public.product_class_patterns USING btree (sort_order) WHERE is_active;


--
-- Name: idx_product_unsupported_paths_product; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_product_unsupported_paths_product ON public.product_unsupported_paths USING btree (product_id);


--
-- Name: idx_product_unsupported_paths_product_firmware; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_product_unsupported_paths_product_firmware ON public.product_unsupported_paths USING btree (product_id, firmware_version) WHERE (read_unsupported = true);


--
-- Name: idx_products_alarm_ne_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_products_alarm_ne_type ON public.products USING btree (alarm_ne_type);


--
-- Name: idx_products_indicator; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_products_indicator ON public.products USING btree (indicator_device_type, indicator_platform);


--
-- Name: idx_products_param_model; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_products_param_model ON public.products USING btree (param_model_id) WHERE (param_model_id IS NOT NULL);


--
-- Name: idx_pt_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pt_created ON public.provisioning_tasks USING btree (created_at DESC);


--
-- Name: idx_pt_device; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pt_device ON public.provisioning_tasks USING btree (device_id);


--
-- Name: idx_pt_device_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pt_device_status ON public.provisioning_tasks USING btree (device_id, status);


--
-- Name: idx_pt_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pt_status ON public.provisioning_tasks USING btree (status);


--
-- Name: idx_rdg_group; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rdg_group ON public.role_device_groups USING btree (group_id);


--
-- Name: idx_rdg_role; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rdg_role ON public.role_device_groups USING btree (role_id);


--
-- Name: idx_rela_platform_indicator_formula_enb_platform_indicator; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rela_platform_indicator_formula_enb_platform_indicator ON public.rela_platform_indicator_formula_enb USING btree (platform_name, indicator_id);


--
-- Name: idx_rela_platform_indicator_formula_gnb_platform_indicator; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rela_platform_indicator_formula_gnb_platform_indicator ON public.rela_platform_indicator_formula_gnb USING btree (platform_name, indicator_id);


--
-- Name: idx_rela_platform_indicator_formula_gsm_platform_indicator; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rela_platform_indicator_formula_gsm_platform_indicator ON public.rela_platform_indicator_formula_gsm USING btree (platform_name, indicator_id);


--
-- Name: idx_report_defs_device_groups_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_report_defs_device_groups_gin ON public.report_definitions USING gin (device_groups);


--
-- Name: idx_report_defs_kpi_codes_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_report_defs_kpi_codes_gin ON public.report_definitions USING gin (kpi_codes);


--
-- Name: idx_report_defs_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_report_defs_status ON public.report_definitions USING btree (status);


--
-- Name: idx_report_defs_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_report_defs_type ON public.report_definitions USING btree (report_type);


--
-- Name: idx_report_records_def; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_report_records_def ON public.report_records USING btree (report_definition_id);


--
-- Name: idx_report_records_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_report_records_status ON public.report_records USING btree (status);


--
-- Name: idx_report_records_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_report_records_time ON public.report_records USING btree (generate_time DESC);


--
-- Name: idx_restore_tasks_create_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_restore_tasks_create_user ON public.restore_tasks USING btree (create_user);


--
-- Name: idx_restore_tasks_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_restore_tasks_created_at ON public.restore_tasks USING btree (created_at DESC);


--
-- Name: idx_restore_tasks_operator_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_restore_tasks_operator_code ON public.restore_tasks USING btree (operator_code);


--
-- Name: idx_restore_tasks_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_restore_tasks_status ON public.restore_tasks USING btree (status);


--
-- Name: idx_role_api_permissions_endpoint; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_role_api_permissions_endpoint ON public.role_api_permissions USING btree (endpoint_id);


--
-- Name: idx_role_api_permissions_role; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_role_api_permissions_role ON public.role_api_permissions USING btree (role_id);


--
-- Name: idx_role_menus_menu; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_role_menus_menu ON public.role_menus USING btree (menu_id);


--
-- Name: idx_role_menus_role; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_role_menus_role ON public.role_menus USING btree (role_id);


--
-- Name: idx_runtime_log_collect_sub_tasks_command_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_runtime_log_collect_sub_tasks_command_key ON public.runtime_log_collect_sub_tasks USING btree (command_key) WHERE (command_key IS NOT NULL);


--
-- Name: idx_runtime_log_collect_sub_tasks_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_runtime_log_collect_sub_tasks_created_at ON public.runtime_log_collect_sub_tasks USING btree (created_at DESC);


--
-- Name: idx_runtime_log_collect_sub_tasks_device_sn; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_runtime_log_collect_sub_tasks_device_sn ON public.runtime_log_collect_sub_tasks USING btree (device_sn);


--
-- Name: idx_runtime_log_collect_sub_tasks_task_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_runtime_log_collect_sub_tasks_task_status ON public.runtime_log_collect_sub_tasks USING btree (task_id, status);


--
-- Name: idx_runtime_log_collect_tasks_due_scheduled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_runtime_log_collect_tasks_due_scheduled ON public.runtime_log_collect_tasks USING btree (scheduled_at) WHERE (((create_status)::text = 'timing'::text) AND ((status)::text = 'pending'::text));


--
-- Name: idx_sites_domain; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sites_domain ON public.sites USING btree (domain_id);


--
-- Name: idx_sites_geo; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sites_geo ON public.sites USING btree (longitude, latitude);


--
-- Name: idx_sites_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sites_status ON public.sites USING btree (status);


--
-- Name: idx_standard_commands_group; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_standard_commands_group ON public.standard_commands USING btree (version_code, group_code);


--
-- Name: idx_standard_params_entry_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_standard_params_entry_type ON public.standard_params USING btree (entry_type);


--
-- Name: idx_station_fault_logs_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_station_fault_logs_active ON public.station_fault_logs USING btree (collected_at DESC) WHERE (is_deleted = false);


--
-- Name: idx_station_fault_logs_device_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_station_fault_logs_device_id ON public.station_fault_logs USING btree (device_id);


--
-- Name: idx_station_fault_logs_device_sn; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_station_fault_logs_device_sn ON public.station_fault_logs USING btree (device_sn);


--
-- Name: idx_station_fault_logs_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_station_fault_logs_status ON public.station_fault_logs USING btree (record_status, collected_at DESC);


--
-- Name: idx_config_apply_targets_pending; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_apply_targets_pending ON public.config_apply_targets USING btree (status, updated_at) WHERE ((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('failed'::character varying)::text]));


--
-- Name: idx_config_apply_targets_recovering; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_apply_targets_recovering ON public.config_apply_targets USING btree (lease_expires_at) WHERE ((status)::text = 'applying'::text);


--
-- Name: uq_config_apply_target_running; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_config_apply_target_running ON public.config_apply_targets USING btree (category, target) WHERE ((status)::text = 'applying'::text);


--
-- Name: idx_station_running_logs_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_station_running_logs_active ON public.station_running_logs USING btree (collected_at DESC) WHERE (is_deleted = false);


--
-- Name: idx_station_running_logs_device_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_station_running_logs_device_id ON public.station_running_logs USING btree (device_id);


--
-- Name: idx_station_running_logs_device_sn; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_station_running_logs_device_sn ON public.station_running_logs USING btree (device_sn);


--
-- Name: idx_sub_field_overrides_owner; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sub_field_overrides_owner ON public.mml_command_sub_field_overrides USING btree (owner_user_id);


--
-- Name: idx_sys_configs_category; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sys_configs_category ON public.sys_configs USING btree (category);


--
-- Name: idx_sys_configs_public; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sys_configs_public ON public.sys_configs USING btree (is_public) WHERE (is_public = true);


--
-- Name: idx_sys_dict_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sys_dict_deleted_at ON public.sys_dictionaries USING btree (deleted_at) WHERE (deleted_at IS NOT NULL);


--
-- Name: idx_sys_dict_detail_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sys_dict_detail_deleted_at ON public.sys_dictionary_details USING btree (deleted_at) WHERE (deleted_at IS NOT NULL);


--
-- Name: idx_sys_dict_detail_dict_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sys_dict_detail_dict_id ON public.sys_dictionary_details USING btree (sys_dictionary_id);


--
-- Name: idx_sys_dict_detail_dictid_origin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sys_dict_detail_dictid_origin ON public.sys_dictionary_details USING btree (sys_dictionary_id, origin);


--
-- Name: idx_sys_dict_source_table; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sys_dict_source_table ON public.sys_dictionaries USING btree (source_table) WHERE (source_table IS NOT NULL);


--
-- Name: idx_system_license_expiry; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_system_license_expiry ON public.system_license USING btree (expiry_date) WHERE (is_current = true);


--
-- Name: idx_system_license_history_license_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_system_license_history_license_id ON public.system_license_history USING btree (license_id);


--
-- Name: idx_system_license_history_replaced; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_system_license_history_replaced ON public.system_license_history USING btree (replaced_at DESC);


--
-- Name: idx_system_license_history_uploaded; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_system_license_history_uploaded ON public.system_license_history USING btree (uploaded_at DESC);


--
-- Name: idx_system_license_uploaded; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_system_license_uploaded ON public.system_license USING btree (uploaded_at DESC);


--
-- Name: idx_system_logs_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_system_logs_created_at ON public.system_logs USING btree (created_at DESC);


--
-- Name: idx_system_logs_created_at_brin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_system_logs_created_at_brin ON public.system_logs USING brin (created_at) WITH (pages_per_range='128');


--
-- Name: idx_system_logs_details_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_system_logs_details_gin ON public.system_logs USING gin (details jsonb_path_ops);


--
-- Name: idx_system_logs_level; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_system_logs_level ON public.system_logs USING btree (level);


--
-- Name: idx_system_logs_level_source_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_system_logs_level_source_time ON public.system_logs USING btree (level, source, created_at DESC);


--
-- Name: idx_system_logs_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_system_logs_source ON public.system_logs USING btree (source);


--
-- Name: idx_task_logs_operator_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_logs_operator_id ON public.sys_task_logs USING btree (operator_id) WHERE (operator_id IS NOT NULL);


--
-- Name: idx_task_logs_started_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_logs_started_at ON public.sys_task_logs USING btree (started_at);


--
-- Name: idx_task_logs_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_logs_status ON public.sys_task_logs USING btree (status);


--
-- Name: idx_task_logs_task_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_logs_task_type ON public.sys_task_logs USING btree (task_type);


--
-- Name: idx_topo_edges_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_topo_edges_source ON public.topo_edges USING btree (source_id);


--
-- Name: idx_topo_edges_source_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_topo_edges_source_id ON public.topo_edges USING btree (source_id) WHERE ((status)::text = 'active'::text);


--
-- Name: idx_topo_edges_target; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_topo_edges_target ON public.topo_edges USING btree (target_id);


--
-- Name: idx_topo_edges_target_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_topo_edges_target_id ON public.topo_edges USING btree (target_id) WHERE ((status)::text = 'active'::text);


--
-- Name: idx_topo_nodes_device; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_topo_nodes_device ON public.topo_nodes USING btree (device_sn);


--
-- Name: idx_topo_nodes_domain; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_topo_nodes_domain ON public.topo_nodes USING btree (domain_id);


--
-- Name: idx_topo_nodes_domain_type_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_topo_nodes_domain_type_status ON public.topo_nodes USING btree (domain_id, node_type, status);


--
-- Name: idx_topo_nodes_label_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_topo_nodes_label_active ON public.topo_nodes USING btree (label);


--
-- Name: idx_topo_nodes_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_topo_nodes_type ON public.topo_nodes USING btree (node_type);


--
-- Name: idx_trace_export_jobs_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_trace_export_jobs_status ON public.trace_export_jobs USING btree (status, created_at DESC);


--
-- Name: idx_trace_export_jobs_task; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_trace_export_jobs_task ON public.trace_export_jobs USING btree (task_id);


--
-- Name: idx_trace_tasks_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_trace_tasks_created ON public.trace_tasks USING btree (created_at DESC);


--
-- Name: idx_trace_tasks_sn_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_trace_tasks_sn_status ON public.trace_tasks USING btree (device_sn, status);


--
-- Name: idx_trace_tasks_status_expires; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_trace_tasks_status_expires ON public.trace_tasks USING btree (status, expires_at) WHERE ((status)::text = 'running'::text);


--
-- Name: idx_ufte_task_types_built_in; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ufte_task_types_built_in ON public.ufte_task_types USING btree (built_in);


--
-- Name: idx_ufte_task_types_category; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ufte_task_types_category ON public.ufte_task_types USING btree (category);


--
-- Name: idx_ufte_task_types_enabled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ufte_task_types_enabled ON public.ufte_task_types USING btree (enabled);


--
-- Name: idx_ufte_task_types_firmware_file_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ufte_task_types_firmware_file_type ON public.ufte_task_types USING btree (firmware_file_type);


--
-- Name: idx_ufte_task_types_sort; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ufte_task_types_sort ON public.ufte_task_types USING btree (category, sort_order, display_name);


--
-- Name: idx_upgrade_sub_tasks_command_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_upgrade_sub_tasks_command_key ON public.upgrade_sub_tasks USING btree (command_key) WHERE (command_key IS NOT NULL);


--
-- Name: idx_upgrade_sub_tasks_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_upgrade_sub_tasks_created_at ON public.upgrade_sub_tasks USING btree (created_at DESC);


--
-- Name: idx_upgrade_sub_tasks_device_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_upgrade_sub_tasks_device_active ON public.upgrade_sub_tasks USING btree (device_id, status);


--
-- Name: idx_upgrade_sub_tasks_device_active_uniq; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_upgrade_sub_tasks_device_active_uniq ON public.upgrade_sub_tasks USING btree (device_id) WHERE ((status)::text <> ALL (ARRAY[('completed'::character varying)::text, ('failed'::character varying)::text, ('terminated'::character varying)::text]));


--
-- Name: idx_upgrade_sub_tasks_device_sn; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_upgrade_sub_tasks_device_sn ON public.upgrade_sub_tasks USING btree (device_sn);


--
-- Name: idx_upgrade_sub_tasks_task_id_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_upgrade_sub_tasks_task_id_status ON public.upgrade_sub_tasks USING btree (task_id, status);


--
-- Name: idx_upgrade_tasks_canary_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_upgrade_tasks_canary_active ON public.upgrade_tasks USING btree (strategy, stage_status) WHERE (((strategy)::text = 'canary'::text) AND ((stage_status)::text = ANY (ARRAY[('running'::character varying)::text, ('paused'::character varying)::text])));


--
-- Name: idx_upgrade_tasks_due_scheduled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_upgrade_tasks_due_scheduled ON public.upgrade_tasks USING btree (scheduled_at) WHERE (((create_status)::text = 'timing'::text) AND ((status)::text = 'pending'::text));


--
-- Name: idx_upgrade_tasks_firmware; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_upgrade_tasks_firmware ON public.upgrade_tasks USING btree (firmware_id);


--
-- Name: idx_upgrade_tasks_rollback_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_upgrade_tasks_rollback_source ON public.upgrade_tasks USING btree (rollback_source) WHERE (task_type = 2);


--
-- Name: idx_upgrade_tasks_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_upgrade_tasks_status ON public.upgrade_tasks USING btree (status);


--
-- Name: idx_upgrade_tasks_task_create_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_upgrade_tasks_task_create_user ON public.upgrade_tasks USING btree (create_user);


--
-- Name: idx_upgrade_tasks_task_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_upgrade_tasks_task_created_at ON public.upgrade_tasks USING btree (created_at DESC);


--
-- Name: idx_upgrade_tasks_task_product_class; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_upgrade_tasks_task_product_class ON public.upgrade_tasks USING btree (product_class);


--
-- Name: idx_upgrade_tasks_task_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_upgrade_tasks_task_status ON public.upgrade_tasks USING btree (status);


--
-- Name: idx_users_expire_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_expire_at ON public.users USING btree (expire_at) WHERE (expire_at IS NOT NULL);


--
-- Name: idx_users_must_change_password; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_must_change_password ON public.users USING btree (must_change_password) WHERE (must_change_password = true);


--
-- Name: idx_users_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_source ON public.users USING btree (source);


--
-- Name: idx_users_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_status ON public.users USING btree (status);


--
-- Name: idx_users_username; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_username ON public.users USING btree (username);


--
-- Name: uniq_dict_detail_child_value; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uniq_dict_detail_child_value ON public.sys_dictionary_details USING btree (sys_dictionary_id, parent_id, value) WHERE ((parent_id IS NOT NULL) AND (deleted_at IS NULL));


--
-- Name: uniq_dict_detail_top_value; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uniq_dict_detail_top_value ON public.sys_dictionary_details USING btree (sys_dictionary_id, value) WHERE ((parent_id IS NULL) AND (deleted_at IS NULL));


--
-- Name: uniq_dict_type_active; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uniq_dict_type_active ON public.sys_dictionaries USING btree (type) WHERE (deleted_at IS NULL);


--
-- Name: uniq_discovered_product_swver_standard; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uniq_discovered_product_swver_standard ON public.discovered_param_mappings USING btree (product_id, software_version, standard_path);


--
-- Name: uniq_menu_name_per_parent; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uniq_menu_name_per_parent ON public.menus USING btree (parent_id NULLS FIRST, name) WHERE ((status)::text = 'normal'::text);


--
-- Name: uniq_mml_commands_command_name_active; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uniq_mml_commands_command_name_active ON public.mml_commands USING btree (command_name) WHERE (deprecated_at IS NULL);


--
-- Name: uniq_mml_param_groups_object_path; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uniq_mml_param_groups_object_path ON public.mml_command_groups USING btree (object_path_template) WHERE ((object_path_template IS NOT NULL) AND (deleted_at IS NULL));


--
-- Name: uniq_northbound_servers_active; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uniq_northbound_servers_active ON public.northbound_servers USING btree (is_active) WHERE (is_active = true);


--
-- Name: uniq_notifications_user_dedup; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uniq_notifications_user_dedup ON public.notifications USING btree (user_id, dedup_key) WHERE (dedup_key IS NOT NULL);


--
-- Name: uniq_param_mappings_model_private; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uniq_param_mappings_model_private ON public.param_mappings USING btree (param_model_id, private_path);


--
-- Name: uniq_pm_query_templates_creator_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uniq_pm_query_templates_creator_name ON public.pm_query_templates USING btree (creator_id, name);


--
-- Name: uniq_product_class_patterns_global_order; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uniq_product_class_patterns_global_order ON public.product_class_patterns USING btree (sort_order) WHERE is_active;


--
-- Name: uniq_roles_code; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uniq_roles_code ON public.roles USING btree (code) WHERE (code IS NOT NULL);


--
-- Name: uniq_trace_tasks_running_sn; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uniq_trace_tasks_running_sn ON public.trace_tasks USING btree (device_sn) WHERE ((status)::text = 'running'::text);


--
-- Name: uniq_user_default_role; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uniq_user_default_role ON public.user_roles USING btree (user_id) WHERE (is_default = true);


--
-- Name: uq_async_jobs_bucket; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_async_jobs_bucket ON public.async_jobs USING btree (job_type, bucket_start, bucket_end) WHERE ((bucket_start IS NOT NULL) AND (bucket_end IS NOT NULL));


--
-- Name: uq_mml_custom_command_private_name_per_owner; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_mml_custom_command_private_name_per_owner ON public.mml_custom_command USING btree (owner_user_id, command_name) WHERE (((command_scope)::text = 'private'::text) AND (owner_user_id IS NOT NULL));


--
-- Name: INDEX uq_mml_custom_command_private_name_per_owner; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON INDEX public.uq_mml_custom_command_private_name_per_owner IS '用户级唯一性：每个用户的私有模板 command_name 不重复。public 跨用户允许同名；owner_user_id NULL 的历史脏数据豁免（待 000134 backfill 覆盖）。';


--
-- Name: uq_mml_scripts_creator_name_ci; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_mml_scripts_creator_name_ci ON public.mml_scripts USING btree (COALESCE(creator, ''::character varying), lower(btrim((script_name)::text)));


--
-- Name: uq_mml_scripts_import_session_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_mml_scripts_import_session_id ON public.mml_scripts USING btree (import_session_id);


--
-- Name: uq_mml_tasks_active_root_script; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_mml_tasks_active_root_script ON public.mml_tasks USING btree (script_id) WHERE ((script_id IS NOT NULL) AND (parent_task_id IS NULL) AND ((status)::text = ANY ((ARRAY['pending'::character varying, 'running'::character varying, 'paused'::character varying])::text[])));


--
-- Name: INDEX uq_mml_tasks_active_root_script; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON INDEX public.uq_mml_tasks_active_root_script IS 'At most one unfinished top-level MML script task per script. Periodic child runs keep parent_task_id and are not constrained here.';


--
-- Name: uq_mml_tasks_creator_request_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_mml_tasks_creator_request_id ON public.mml_tasks USING btree (COALESCE(creator, ''::character varying), request_id) WHERE ((request_id IS NOT NULL) AND (btrim(request_id) <> ''::text));


--
-- Name: uq_parameter_sync_requests_idempotency; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_parameter_sync_requests_idempotency ON public.parameter_sync_requests USING btree (caller_type, idempotency_key) WHERE (idempotency_key IS NOT NULL);


--
-- Name: uq_parameter_sync_requests_model_upload_intent; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_parameter_sync_requests_model_upload_intent ON public.parameter_sync_requests USING btree (model_upload_intent_id) WHERE (model_upload_intent_id IS NOT NULL);


--
-- Name: uq_parameter_sync_runs_active_device; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_parameter_sync_runs_active_device ON public.parameter_sync_runs USING btree (device_id) WHERE ((status)::text = ANY ((ARRAY['planning'::character varying, 'enqueuing'::character varying, 'waiting_device'::character varying, 'executing'::character varying, 'processing'::character varying, 'cancelling'::character varying])::text[]));


--
-- Name: uq_system_license_current; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_system_license_current ON public.system_license USING btree (is_current) WHERE (is_current = true);


--
-- Name: device_parameters_p00_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p00_device_id_idx;


--
-- Name: device_parameters_p00_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p00_device_id_parameter_path_idx;


--
-- Name: device_parameters_p00_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p00_parameter_value_device_id_idx;


--
-- Name: device_parameters_p00_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p00_pkey;


--
-- Name: device_parameters_p01_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p01_device_id_idx;


--
-- Name: device_parameters_p01_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p01_device_id_parameter_path_idx;


--
-- Name: device_parameters_p01_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p01_parameter_value_device_id_idx;


--
-- Name: device_parameters_p01_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p01_pkey;


--
-- Name: device_parameters_p02_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p02_device_id_idx;


--
-- Name: device_parameters_p02_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p02_device_id_parameter_path_idx;


--
-- Name: device_parameters_p02_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p02_parameter_value_device_id_idx;


--
-- Name: device_parameters_p02_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p02_pkey;


--
-- Name: device_parameters_p03_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p03_device_id_idx;


--
-- Name: device_parameters_p03_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p03_device_id_parameter_path_idx;


--
-- Name: device_parameters_p03_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p03_parameter_value_device_id_idx;


--
-- Name: device_parameters_p03_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p03_pkey;


--
-- Name: device_parameters_p04_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p04_device_id_idx;


--
-- Name: device_parameters_p04_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p04_device_id_parameter_path_idx;


--
-- Name: device_parameters_p04_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p04_parameter_value_device_id_idx;


--
-- Name: device_parameters_p04_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p04_pkey;


--
-- Name: device_parameters_p05_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p05_device_id_idx;


--
-- Name: device_parameters_p05_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p05_device_id_parameter_path_idx;


--
-- Name: device_parameters_p05_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p05_parameter_value_device_id_idx;


--
-- Name: device_parameters_p05_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p05_pkey;


--
-- Name: device_parameters_p06_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p06_device_id_idx;


--
-- Name: device_parameters_p06_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p06_device_id_parameter_path_idx;


--
-- Name: device_parameters_p06_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p06_parameter_value_device_id_idx;


--
-- Name: device_parameters_p06_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p06_pkey;


--
-- Name: device_parameters_p07_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p07_device_id_idx;


--
-- Name: device_parameters_p07_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p07_device_id_parameter_path_idx;


--
-- Name: device_parameters_p07_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p07_parameter_value_device_id_idx;


--
-- Name: device_parameters_p07_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p07_pkey;


--
-- Name: device_parameters_p08_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p08_device_id_idx;


--
-- Name: device_parameters_p08_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p08_device_id_parameter_path_idx;


--
-- Name: device_parameters_p08_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p08_parameter_value_device_id_idx;


--
-- Name: device_parameters_p08_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p08_pkey;


--
-- Name: device_parameters_p09_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p09_device_id_idx;


--
-- Name: device_parameters_p09_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p09_device_id_parameter_path_idx;


--
-- Name: device_parameters_p09_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p09_parameter_value_device_id_idx;


--
-- Name: device_parameters_p09_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p09_pkey;


--
-- Name: device_parameters_p10_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p10_device_id_idx;


--
-- Name: device_parameters_p10_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p10_device_id_parameter_path_idx;


--
-- Name: device_parameters_p10_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p10_parameter_value_device_id_idx;


--
-- Name: device_parameters_p10_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p10_pkey;


--
-- Name: device_parameters_p11_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p11_device_id_idx;


--
-- Name: device_parameters_p11_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p11_device_id_parameter_path_idx;


--
-- Name: device_parameters_p11_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p11_parameter_value_device_id_idx;


--
-- Name: device_parameters_p11_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p11_pkey;


--
-- Name: device_parameters_p12_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p12_device_id_idx;


--
-- Name: device_parameters_p12_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p12_device_id_parameter_path_idx;


--
-- Name: device_parameters_p12_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p12_parameter_value_device_id_idx;


--
-- Name: device_parameters_p12_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p12_pkey;


--
-- Name: device_parameters_p13_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p13_device_id_idx;


--
-- Name: device_parameters_p13_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p13_device_id_parameter_path_idx;


--
-- Name: device_parameters_p13_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p13_parameter_value_device_id_idx;


--
-- Name: device_parameters_p13_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p13_pkey;


--
-- Name: device_parameters_p14_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p14_device_id_idx;


--
-- Name: device_parameters_p14_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p14_device_id_parameter_path_idx;


--
-- Name: device_parameters_p14_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p14_parameter_value_device_id_idx;


--
-- Name: device_parameters_p14_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p14_pkey;


--
-- Name: device_parameters_p15_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p15_device_id_idx;


--
-- Name: device_parameters_p15_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p15_device_id_parameter_path_idx;


--
-- Name: device_parameters_p15_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p15_parameter_value_device_id_idx;


--
-- Name: device_parameters_p15_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p15_pkey;


--
-- Name: device_parameters_p16_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p16_device_id_idx;


--
-- Name: device_parameters_p16_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p16_device_id_parameter_path_idx;


--
-- Name: device_parameters_p16_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p16_parameter_value_device_id_idx;


--
-- Name: device_parameters_p16_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p16_pkey;


--
-- Name: device_parameters_p17_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p17_device_id_idx;


--
-- Name: device_parameters_p17_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p17_device_id_parameter_path_idx;


--
-- Name: device_parameters_p17_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p17_parameter_value_device_id_idx;


--
-- Name: device_parameters_p17_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p17_pkey;


--
-- Name: device_parameters_p18_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p18_device_id_idx;


--
-- Name: device_parameters_p18_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p18_device_id_parameter_path_idx;


--
-- Name: device_parameters_p18_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p18_parameter_value_device_id_idx;


--
-- Name: device_parameters_p18_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p18_pkey;


--
-- Name: device_parameters_p19_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p19_device_id_idx;


--
-- Name: device_parameters_p19_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p19_device_id_parameter_path_idx;


--
-- Name: device_parameters_p19_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p19_parameter_value_device_id_idx;


--
-- Name: device_parameters_p19_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p19_pkey;


--
-- Name: device_parameters_p20_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p20_device_id_idx;


--
-- Name: device_parameters_p20_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p20_device_id_parameter_path_idx;


--
-- Name: device_parameters_p20_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p20_parameter_value_device_id_idx;


--
-- Name: device_parameters_p20_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p20_pkey;


--
-- Name: device_parameters_p21_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p21_device_id_idx;


--
-- Name: device_parameters_p21_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p21_device_id_parameter_path_idx;


--
-- Name: device_parameters_p21_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p21_parameter_value_device_id_idx;


--
-- Name: device_parameters_p21_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p21_pkey;


--
-- Name: device_parameters_p22_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p22_device_id_idx;


--
-- Name: device_parameters_p22_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p22_device_id_parameter_path_idx;


--
-- Name: device_parameters_p22_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p22_parameter_value_device_id_idx;


--
-- Name: device_parameters_p22_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p22_pkey;


--
-- Name: device_parameters_p23_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p23_device_id_idx;


--
-- Name: device_parameters_p23_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p23_device_id_parameter_path_idx;


--
-- Name: device_parameters_p23_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p23_parameter_value_device_id_idx;


--
-- Name: device_parameters_p23_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p23_pkey;


--
-- Name: device_parameters_p24_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p24_device_id_idx;


--
-- Name: device_parameters_p24_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p24_device_id_parameter_path_idx;


--
-- Name: device_parameters_p24_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p24_parameter_value_device_id_idx;


--
-- Name: device_parameters_p24_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p24_pkey;


--
-- Name: device_parameters_p25_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p25_device_id_idx;


--
-- Name: device_parameters_p25_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p25_device_id_parameter_path_idx;


--
-- Name: device_parameters_p25_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p25_parameter_value_device_id_idx;


--
-- Name: device_parameters_p25_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p25_pkey;


--
-- Name: device_parameters_p26_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p26_device_id_idx;


--
-- Name: device_parameters_p26_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p26_device_id_parameter_path_idx;


--
-- Name: device_parameters_p26_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p26_parameter_value_device_id_idx;


--
-- Name: device_parameters_p26_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p26_pkey;


--
-- Name: device_parameters_p27_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p27_device_id_idx;


--
-- Name: device_parameters_p27_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p27_device_id_parameter_path_idx;


--
-- Name: device_parameters_p27_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p27_parameter_value_device_id_idx;


--
-- Name: device_parameters_p27_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p27_pkey;


--
-- Name: device_parameters_p28_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p28_device_id_idx;


--
-- Name: device_parameters_p28_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p28_device_id_parameter_path_idx;


--
-- Name: device_parameters_p28_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p28_parameter_value_device_id_idx;


--
-- Name: device_parameters_p28_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p28_pkey;


--
-- Name: device_parameters_p29_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p29_device_id_idx;


--
-- Name: device_parameters_p29_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p29_device_id_parameter_path_idx;


--
-- Name: device_parameters_p29_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p29_parameter_value_device_id_idx;


--
-- Name: device_parameters_p29_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p29_pkey;


--
-- Name: device_parameters_p30_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p30_device_id_idx;


--
-- Name: device_parameters_p30_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p30_device_id_parameter_path_idx;


--
-- Name: device_parameters_p30_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p30_parameter_value_device_id_idx;


--
-- Name: device_parameters_p30_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p30_pkey;


--
-- Name: device_parameters_p31_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_device ATTACH PARTITION public.device_parameters_p31_device_id_idx;


--
-- Name: device_parameters_p31_device_id_parameter_path_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_path_prefix ATTACH PARTITION public.device_parameters_p31_device_id_parameter_path_idx;


--
-- Name: device_parameters_p31_parameter_value_device_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_params_swver ATTACH PARTITION public.device_parameters_p31_parameter_value_device_id_idx;


--
-- Name: device_parameters_p31_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_parameters_pkey ATTACH PARTITION public.device_parameters_p31_pkey;


--
-- Name: device_tasks_p00_created_at_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending_created_id ATTACH PARTITION public.device_tasks_p00_created_at_id_idx;


--
-- Name: device_tasks_p00_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_created_at ATTACH PARTITION public.device_tasks_p00_created_at_idx;


--
-- Name: device_tasks_p00_created_at_idx1; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_has_path_translation_miss ATTACH PARTITION public.device_tasks_p00_created_at_idx1;


--
-- Name: device_tasks_p00_cwmp_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_cwmp_id ATTACH PARTITION public.device_tasks_p00_cwmp_id_idx;


--
-- Name: device_tasks_p00_device_sn_status_priority_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending ATTACH PARTITION public.device_tasks_p00_device_sn_status_priority_created_at_idx;


--
-- Name: device_tasks_p00_params_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_params_gin ATTACH PARTITION public.device_tasks_p00_params_idx;


--
-- Name: device_tasks_p00_parent_task_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_source_id ATTACH PARTITION public.device_tasks_p00_parent_task_id_idx;


--
-- Name: device_tasks_p00_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_tasks_pkey1 ATTACH PARTITION public.device_tasks_p00_pkey;


--
-- Name: device_tasks_p00_result_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_result_gin ATTACH PARTITION public.device_tasks_p00_result_idx;


--
-- Name: device_tasks_p00_status_expires_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status_expires ATTACH PARTITION public.device_tasks_p00_status_expires_at_idx;


--
-- Name: device_tasks_p00_status_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status ATTACH PARTITION public.device_tasks_p00_status_idx;


--
-- Name: device_tasks_p01_created_at_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending_created_id ATTACH PARTITION public.device_tasks_p01_created_at_id_idx;


--
-- Name: device_tasks_p01_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_created_at ATTACH PARTITION public.device_tasks_p01_created_at_idx;


--
-- Name: device_tasks_p01_created_at_idx1; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_has_path_translation_miss ATTACH PARTITION public.device_tasks_p01_created_at_idx1;


--
-- Name: device_tasks_p01_cwmp_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_cwmp_id ATTACH PARTITION public.device_tasks_p01_cwmp_id_idx;


--
-- Name: device_tasks_p01_device_sn_status_priority_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending ATTACH PARTITION public.device_tasks_p01_device_sn_status_priority_created_at_idx;


--
-- Name: device_tasks_p01_params_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_params_gin ATTACH PARTITION public.device_tasks_p01_params_idx;


--
-- Name: device_tasks_p01_parent_task_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_source_id ATTACH PARTITION public.device_tasks_p01_parent_task_id_idx;


--
-- Name: device_tasks_p01_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_tasks_pkey1 ATTACH PARTITION public.device_tasks_p01_pkey;


--
-- Name: device_tasks_p01_result_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_result_gin ATTACH PARTITION public.device_tasks_p01_result_idx;


--
-- Name: device_tasks_p01_status_expires_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status_expires ATTACH PARTITION public.device_tasks_p01_status_expires_at_idx;


--
-- Name: device_tasks_p01_status_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status ATTACH PARTITION public.device_tasks_p01_status_idx;


--
-- Name: device_tasks_p02_created_at_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending_created_id ATTACH PARTITION public.device_tasks_p02_created_at_id_idx;


--
-- Name: device_tasks_p02_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_created_at ATTACH PARTITION public.device_tasks_p02_created_at_idx;


--
-- Name: device_tasks_p02_created_at_idx1; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_has_path_translation_miss ATTACH PARTITION public.device_tasks_p02_created_at_idx1;


--
-- Name: device_tasks_p02_cwmp_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_cwmp_id ATTACH PARTITION public.device_tasks_p02_cwmp_id_idx;


--
-- Name: device_tasks_p02_device_sn_status_priority_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending ATTACH PARTITION public.device_tasks_p02_device_sn_status_priority_created_at_idx;


--
-- Name: device_tasks_p02_params_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_params_gin ATTACH PARTITION public.device_tasks_p02_params_idx;


--
-- Name: device_tasks_p02_parent_task_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_source_id ATTACH PARTITION public.device_tasks_p02_parent_task_id_idx;


--
-- Name: device_tasks_p02_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_tasks_pkey1 ATTACH PARTITION public.device_tasks_p02_pkey;


--
-- Name: device_tasks_p02_result_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_result_gin ATTACH PARTITION public.device_tasks_p02_result_idx;


--
-- Name: device_tasks_p02_status_expires_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status_expires ATTACH PARTITION public.device_tasks_p02_status_expires_at_idx;


--
-- Name: device_tasks_p02_status_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status ATTACH PARTITION public.device_tasks_p02_status_idx;


--
-- Name: device_tasks_p03_created_at_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending_created_id ATTACH PARTITION public.device_tasks_p03_created_at_id_idx;


--
-- Name: device_tasks_p03_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_created_at ATTACH PARTITION public.device_tasks_p03_created_at_idx;


--
-- Name: device_tasks_p03_created_at_idx1; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_has_path_translation_miss ATTACH PARTITION public.device_tasks_p03_created_at_idx1;


--
-- Name: device_tasks_p03_cwmp_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_cwmp_id ATTACH PARTITION public.device_tasks_p03_cwmp_id_idx;


--
-- Name: device_tasks_p03_device_sn_status_priority_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending ATTACH PARTITION public.device_tasks_p03_device_sn_status_priority_created_at_idx;


--
-- Name: device_tasks_p03_params_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_params_gin ATTACH PARTITION public.device_tasks_p03_params_idx;


--
-- Name: device_tasks_p03_parent_task_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_source_id ATTACH PARTITION public.device_tasks_p03_parent_task_id_idx;


--
-- Name: device_tasks_p03_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_tasks_pkey1 ATTACH PARTITION public.device_tasks_p03_pkey;


--
-- Name: device_tasks_p03_result_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_result_gin ATTACH PARTITION public.device_tasks_p03_result_idx;


--
-- Name: device_tasks_p03_status_expires_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status_expires ATTACH PARTITION public.device_tasks_p03_status_expires_at_idx;


--
-- Name: device_tasks_p03_status_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status ATTACH PARTITION public.device_tasks_p03_status_idx;


--
-- Name: device_tasks_p04_created_at_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending_created_id ATTACH PARTITION public.device_tasks_p04_created_at_id_idx;


--
-- Name: device_tasks_p04_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_created_at ATTACH PARTITION public.device_tasks_p04_created_at_idx;


--
-- Name: device_tasks_p04_created_at_idx1; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_has_path_translation_miss ATTACH PARTITION public.device_tasks_p04_created_at_idx1;


--
-- Name: device_tasks_p04_cwmp_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_cwmp_id ATTACH PARTITION public.device_tasks_p04_cwmp_id_idx;


--
-- Name: device_tasks_p04_device_sn_status_priority_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending ATTACH PARTITION public.device_tasks_p04_device_sn_status_priority_created_at_idx;


--
-- Name: device_tasks_p04_params_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_params_gin ATTACH PARTITION public.device_tasks_p04_params_idx;


--
-- Name: device_tasks_p04_parent_task_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_source_id ATTACH PARTITION public.device_tasks_p04_parent_task_id_idx;


--
-- Name: device_tasks_p04_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_tasks_pkey1 ATTACH PARTITION public.device_tasks_p04_pkey;


--
-- Name: device_tasks_p04_result_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_result_gin ATTACH PARTITION public.device_tasks_p04_result_idx;


--
-- Name: device_tasks_p04_status_expires_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status_expires ATTACH PARTITION public.device_tasks_p04_status_expires_at_idx;


--
-- Name: device_tasks_p04_status_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status ATTACH PARTITION public.device_tasks_p04_status_idx;


--
-- Name: device_tasks_p05_created_at_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending_created_id ATTACH PARTITION public.device_tasks_p05_created_at_id_idx;


--
-- Name: device_tasks_p05_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_created_at ATTACH PARTITION public.device_tasks_p05_created_at_idx;


--
-- Name: device_tasks_p05_created_at_idx1; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_has_path_translation_miss ATTACH PARTITION public.device_tasks_p05_created_at_idx1;


--
-- Name: device_tasks_p05_cwmp_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_cwmp_id ATTACH PARTITION public.device_tasks_p05_cwmp_id_idx;


--
-- Name: device_tasks_p05_device_sn_status_priority_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending ATTACH PARTITION public.device_tasks_p05_device_sn_status_priority_created_at_idx;


--
-- Name: device_tasks_p05_params_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_params_gin ATTACH PARTITION public.device_tasks_p05_params_idx;


--
-- Name: device_tasks_p05_parent_task_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_source_id ATTACH PARTITION public.device_tasks_p05_parent_task_id_idx;


--
-- Name: device_tasks_p05_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_tasks_pkey1 ATTACH PARTITION public.device_tasks_p05_pkey;


--
-- Name: device_tasks_p05_result_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_result_gin ATTACH PARTITION public.device_tasks_p05_result_idx;


--
-- Name: device_tasks_p05_status_expires_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status_expires ATTACH PARTITION public.device_tasks_p05_status_expires_at_idx;


--
-- Name: device_tasks_p05_status_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status ATTACH PARTITION public.device_tasks_p05_status_idx;


--
-- Name: device_tasks_p06_created_at_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending_created_id ATTACH PARTITION public.device_tasks_p06_created_at_id_idx;


--
-- Name: device_tasks_p06_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_created_at ATTACH PARTITION public.device_tasks_p06_created_at_idx;


--
-- Name: device_tasks_p06_created_at_idx1; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_has_path_translation_miss ATTACH PARTITION public.device_tasks_p06_created_at_idx1;


--
-- Name: device_tasks_p06_cwmp_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_cwmp_id ATTACH PARTITION public.device_tasks_p06_cwmp_id_idx;


--
-- Name: device_tasks_p06_device_sn_status_priority_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending ATTACH PARTITION public.device_tasks_p06_device_sn_status_priority_created_at_idx;


--
-- Name: device_tasks_p06_params_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_params_gin ATTACH PARTITION public.device_tasks_p06_params_idx;


--
-- Name: device_tasks_p06_parent_task_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_source_id ATTACH PARTITION public.device_tasks_p06_parent_task_id_idx;


--
-- Name: device_tasks_p06_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_tasks_pkey1 ATTACH PARTITION public.device_tasks_p06_pkey;


--
-- Name: device_tasks_p06_result_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_result_gin ATTACH PARTITION public.device_tasks_p06_result_idx;


--
-- Name: device_tasks_p06_status_expires_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status_expires ATTACH PARTITION public.device_tasks_p06_status_expires_at_idx;


--
-- Name: device_tasks_p06_status_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status ATTACH PARTITION public.device_tasks_p06_status_idx;


--
-- Name: device_tasks_p07_created_at_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending_created_id ATTACH PARTITION public.device_tasks_p07_created_at_id_idx;


--
-- Name: device_tasks_p07_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_created_at ATTACH PARTITION public.device_tasks_p07_created_at_idx;


--
-- Name: device_tasks_p07_created_at_idx1; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_has_path_translation_miss ATTACH PARTITION public.device_tasks_p07_created_at_idx1;


--
-- Name: device_tasks_p07_cwmp_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_cwmp_id ATTACH PARTITION public.device_tasks_p07_cwmp_id_idx;


--
-- Name: device_tasks_p07_device_sn_status_priority_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending ATTACH PARTITION public.device_tasks_p07_device_sn_status_priority_created_at_idx;


--
-- Name: device_tasks_p07_params_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_params_gin ATTACH PARTITION public.device_tasks_p07_params_idx;


--
-- Name: device_tasks_p07_parent_task_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_source_id ATTACH PARTITION public.device_tasks_p07_parent_task_id_idx;


--
-- Name: device_tasks_p07_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_tasks_pkey1 ATTACH PARTITION public.device_tasks_p07_pkey;


--
-- Name: device_tasks_p07_result_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_result_gin ATTACH PARTITION public.device_tasks_p07_result_idx;


--
-- Name: device_tasks_p07_status_expires_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status_expires ATTACH PARTITION public.device_tasks_p07_status_expires_at_idx;


--
-- Name: device_tasks_p07_status_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status ATTACH PARTITION public.device_tasks_p07_status_idx;


--
-- Name: device_tasks_p08_created_at_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending_created_id ATTACH PARTITION public.device_tasks_p08_created_at_id_idx;


--
-- Name: device_tasks_p08_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_created_at ATTACH PARTITION public.device_tasks_p08_created_at_idx;


--
-- Name: device_tasks_p08_created_at_idx1; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_has_path_translation_miss ATTACH PARTITION public.device_tasks_p08_created_at_idx1;


--
-- Name: device_tasks_p08_cwmp_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_cwmp_id ATTACH PARTITION public.device_tasks_p08_cwmp_id_idx;


--
-- Name: device_tasks_p08_device_sn_status_priority_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending ATTACH PARTITION public.device_tasks_p08_device_sn_status_priority_created_at_idx;


--
-- Name: device_tasks_p08_params_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_params_gin ATTACH PARTITION public.device_tasks_p08_params_idx;


--
-- Name: device_tasks_p08_parent_task_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_source_id ATTACH PARTITION public.device_tasks_p08_parent_task_id_idx;


--
-- Name: device_tasks_p08_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_tasks_pkey1 ATTACH PARTITION public.device_tasks_p08_pkey;


--
-- Name: device_tasks_p08_result_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_result_gin ATTACH PARTITION public.device_tasks_p08_result_idx;


--
-- Name: device_tasks_p08_status_expires_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status_expires ATTACH PARTITION public.device_tasks_p08_status_expires_at_idx;


--
-- Name: device_tasks_p08_status_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status ATTACH PARTITION public.device_tasks_p08_status_idx;


--
-- Name: device_tasks_p09_created_at_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending_created_id ATTACH PARTITION public.device_tasks_p09_created_at_id_idx;


--
-- Name: device_tasks_p09_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_created_at ATTACH PARTITION public.device_tasks_p09_created_at_idx;


--
-- Name: device_tasks_p09_created_at_idx1; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_has_path_translation_miss ATTACH PARTITION public.device_tasks_p09_created_at_idx1;


--
-- Name: device_tasks_p09_cwmp_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_cwmp_id ATTACH PARTITION public.device_tasks_p09_cwmp_id_idx;


--
-- Name: device_tasks_p09_device_sn_status_priority_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending ATTACH PARTITION public.device_tasks_p09_device_sn_status_priority_created_at_idx;


--
-- Name: device_tasks_p09_params_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_params_gin ATTACH PARTITION public.device_tasks_p09_params_idx;


--
-- Name: device_tasks_p09_parent_task_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_source_id ATTACH PARTITION public.device_tasks_p09_parent_task_id_idx;


--
-- Name: device_tasks_p09_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_tasks_pkey1 ATTACH PARTITION public.device_tasks_p09_pkey;


--
-- Name: device_tasks_p09_result_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_result_gin ATTACH PARTITION public.device_tasks_p09_result_idx;


--
-- Name: device_tasks_p09_status_expires_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status_expires ATTACH PARTITION public.device_tasks_p09_status_expires_at_idx;


--
-- Name: device_tasks_p09_status_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status ATTACH PARTITION public.device_tasks_p09_status_idx;


--
-- Name: device_tasks_p10_created_at_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending_created_id ATTACH PARTITION public.device_tasks_p10_created_at_id_idx;


--
-- Name: device_tasks_p10_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_created_at ATTACH PARTITION public.device_tasks_p10_created_at_idx;


--
-- Name: device_tasks_p10_created_at_idx1; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_has_path_translation_miss ATTACH PARTITION public.device_tasks_p10_created_at_idx1;


--
-- Name: device_tasks_p10_cwmp_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_cwmp_id ATTACH PARTITION public.device_tasks_p10_cwmp_id_idx;


--
-- Name: device_tasks_p10_device_sn_status_priority_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending ATTACH PARTITION public.device_tasks_p10_device_sn_status_priority_created_at_idx;


--
-- Name: device_tasks_p10_params_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_params_gin ATTACH PARTITION public.device_tasks_p10_params_idx;


--
-- Name: device_tasks_p10_parent_task_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_source_id ATTACH PARTITION public.device_tasks_p10_parent_task_id_idx;


--
-- Name: device_tasks_p10_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_tasks_pkey1 ATTACH PARTITION public.device_tasks_p10_pkey;


--
-- Name: device_tasks_p10_result_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_result_gin ATTACH PARTITION public.device_tasks_p10_result_idx;


--
-- Name: device_tasks_p10_status_expires_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status_expires ATTACH PARTITION public.device_tasks_p10_status_expires_at_idx;


--
-- Name: device_tasks_p10_status_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status ATTACH PARTITION public.device_tasks_p10_status_idx;


--
-- Name: device_tasks_p11_created_at_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending_created_id ATTACH PARTITION public.device_tasks_p11_created_at_id_idx;


--
-- Name: device_tasks_p11_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_created_at ATTACH PARTITION public.device_tasks_p11_created_at_idx;


--
-- Name: device_tasks_p11_created_at_idx1; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_has_path_translation_miss ATTACH PARTITION public.device_tasks_p11_created_at_idx1;


--
-- Name: device_tasks_p11_cwmp_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_cwmp_id ATTACH PARTITION public.device_tasks_p11_cwmp_id_idx;


--
-- Name: device_tasks_p11_device_sn_status_priority_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending ATTACH PARTITION public.device_tasks_p11_device_sn_status_priority_created_at_idx;


--
-- Name: device_tasks_p11_params_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_params_gin ATTACH PARTITION public.device_tasks_p11_params_idx;


--
-- Name: device_tasks_p11_parent_task_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_source_id ATTACH PARTITION public.device_tasks_p11_parent_task_id_idx;


--
-- Name: device_tasks_p11_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_tasks_pkey1 ATTACH PARTITION public.device_tasks_p11_pkey;


--
-- Name: device_tasks_p11_result_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_result_gin ATTACH PARTITION public.device_tasks_p11_result_idx;


--
-- Name: device_tasks_p11_status_expires_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status_expires ATTACH PARTITION public.device_tasks_p11_status_expires_at_idx;


--
-- Name: device_tasks_p11_status_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status ATTACH PARTITION public.device_tasks_p11_status_idx;


--
-- Name: device_tasks_p12_created_at_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending_created_id ATTACH PARTITION public.device_tasks_p12_created_at_id_idx;


--
-- Name: device_tasks_p12_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_created_at ATTACH PARTITION public.device_tasks_p12_created_at_idx;


--
-- Name: device_tasks_p12_created_at_idx1; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_has_path_translation_miss ATTACH PARTITION public.device_tasks_p12_created_at_idx1;


--
-- Name: device_tasks_p12_cwmp_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_cwmp_id ATTACH PARTITION public.device_tasks_p12_cwmp_id_idx;


--
-- Name: device_tasks_p12_device_sn_status_priority_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending ATTACH PARTITION public.device_tasks_p12_device_sn_status_priority_created_at_idx;


--
-- Name: device_tasks_p12_params_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_params_gin ATTACH PARTITION public.device_tasks_p12_params_idx;


--
-- Name: device_tasks_p12_parent_task_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_source_id ATTACH PARTITION public.device_tasks_p12_parent_task_id_idx;


--
-- Name: device_tasks_p12_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_tasks_pkey1 ATTACH PARTITION public.device_tasks_p12_pkey;


--
-- Name: device_tasks_p12_result_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_result_gin ATTACH PARTITION public.device_tasks_p12_result_idx;


--
-- Name: device_tasks_p12_status_expires_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status_expires ATTACH PARTITION public.device_tasks_p12_status_expires_at_idx;


--
-- Name: device_tasks_p12_status_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status ATTACH PARTITION public.device_tasks_p12_status_idx;


--
-- Name: device_tasks_p13_created_at_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending_created_id ATTACH PARTITION public.device_tasks_p13_created_at_id_idx;


--
-- Name: device_tasks_p13_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_created_at ATTACH PARTITION public.device_tasks_p13_created_at_idx;


--
-- Name: device_tasks_p13_created_at_idx1; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_has_path_translation_miss ATTACH PARTITION public.device_tasks_p13_created_at_idx1;


--
-- Name: device_tasks_p13_cwmp_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_cwmp_id ATTACH PARTITION public.device_tasks_p13_cwmp_id_idx;


--
-- Name: device_tasks_p13_device_sn_status_priority_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending ATTACH PARTITION public.device_tasks_p13_device_sn_status_priority_created_at_idx;


--
-- Name: device_tasks_p13_params_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_params_gin ATTACH PARTITION public.device_tasks_p13_params_idx;


--
-- Name: device_tasks_p13_parent_task_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_source_id ATTACH PARTITION public.device_tasks_p13_parent_task_id_idx;


--
-- Name: device_tasks_p13_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_tasks_pkey1 ATTACH PARTITION public.device_tasks_p13_pkey;


--
-- Name: device_tasks_p13_result_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_result_gin ATTACH PARTITION public.device_tasks_p13_result_idx;


--
-- Name: device_tasks_p13_status_expires_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status_expires ATTACH PARTITION public.device_tasks_p13_status_expires_at_idx;


--
-- Name: device_tasks_p13_status_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status ATTACH PARTITION public.device_tasks_p13_status_idx;


--
-- Name: device_tasks_p14_created_at_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending_created_id ATTACH PARTITION public.device_tasks_p14_created_at_id_idx;


--
-- Name: device_tasks_p14_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_created_at ATTACH PARTITION public.device_tasks_p14_created_at_idx;


--
-- Name: device_tasks_p14_created_at_idx1; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_has_path_translation_miss ATTACH PARTITION public.device_tasks_p14_created_at_idx1;


--
-- Name: device_tasks_p14_cwmp_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_cwmp_id ATTACH PARTITION public.device_tasks_p14_cwmp_id_idx;


--
-- Name: device_tasks_p14_device_sn_status_priority_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending ATTACH PARTITION public.device_tasks_p14_device_sn_status_priority_created_at_idx;


--
-- Name: device_tasks_p14_params_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_params_gin ATTACH PARTITION public.device_tasks_p14_params_idx;


--
-- Name: device_tasks_p14_parent_task_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_source_id ATTACH PARTITION public.device_tasks_p14_parent_task_id_idx;


--
-- Name: device_tasks_p14_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_tasks_pkey1 ATTACH PARTITION public.device_tasks_p14_pkey;


--
-- Name: device_tasks_p14_result_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_result_gin ATTACH PARTITION public.device_tasks_p14_result_idx;


--
-- Name: device_tasks_p14_status_expires_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status_expires ATTACH PARTITION public.device_tasks_p14_status_expires_at_idx;


--
-- Name: device_tasks_p14_status_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status ATTACH PARTITION public.device_tasks_p14_status_idx;


--
-- Name: device_tasks_p15_created_at_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending_created_id ATTACH PARTITION public.device_tasks_p15_created_at_id_idx;


--
-- Name: device_tasks_p15_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_created_at ATTACH PARTITION public.device_tasks_p15_created_at_idx;


--
-- Name: device_tasks_p15_created_at_idx1; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_has_path_translation_miss ATTACH PARTITION public.device_tasks_p15_created_at_idx1;


--
-- Name: device_tasks_p15_cwmp_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_cwmp_id ATTACH PARTITION public.device_tasks_p15_cwmp_id_idx;


--
-- Name: device_tasks_p15_device_sn_status_priority_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_pending ATTACH PARTITION public.device_tasks_p15_device_sn_status_priority_created_at_idx;


--
-- Name: device_tasks_p15_params_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_params_gin ATTACH PARTITION public.device_tasks_p15_params_idx;


--
-- Name: device_tasks_p15_parent_task_id_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_source_id ATTACH PARTITION public.device_tasks_p15_parent_task_id_idx;


--
-- Name: device_tasks_p15_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.device_tasks_pkey1 ATTACH PARTITION public.device_tasks_p15_pkey;


--
-- Name: device_tasks_p15_result_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_result_gin ATTACH PARTITION public.device_tasks_p15_result_idx;


--
-- Name: device_tasks_p15_status_expires_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status_expires ATTACH PARTITION public.device_tasks_p15_status_expires_at_idx;


--
-- Name: device_tasks_p15_status_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_device_tasks_status ATTACH PARTITION public.device_tasks_p15_status_idx;


--
-- Name: devices_cmcc_carrier_last_inform_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_carrier_alive ATTACH PARTITION public.devices_cmcc_carrier_last_inform_at_idx;


--
-- Name: devices_cmcc_carrier_lifecycle_state_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_carrier_lifecycle ATTACH PARTITION public.devices_cmcc_carrier_lifecycle_state_idx;


--
-- Name: devices_cmcc_carrier_technology_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_carrier_tech ATTACH PARTITION public.devices_cmcc_carrier_technology_idx;


--
-- Name: devices_cmcc_deleted_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_deleted_at ATTACH PARTITION public.devices_cmcc_deleted_at_idx;


--
-- Name: devices_cmcc_extension_data_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_extension_data_gin ATTACH PARTITION public.devices_cmcc_extension_data_idx;


--
-- Name: devices_cmcc_ip_address_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_ip_address ATTACH PARTITION public.devices_cmcc_ip_address_idx;


--
-- Name: devices_cmcc_is_online_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_is_online ATTACH PARTITION public.devices_cmcc_is_online_idx;


--
-- Name: devices_cmcc_last_boot_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_last_boot_at ATTACH PARTITION public.devices_cmcc_last_boot_at_idx;


--
-- Name: devices_cmcc_last_inform_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_last_inform ATTACH PARTITION public.devices_cmcc_last_inform_at_idx;


--
-- Name: devices_cmcc_last_inform_events_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_last_inform_events_gin ATTACH PARTITION public.devices_cmcc_last_inform_events_idx;


--
-- Name: devices_cmcc_lifecycle_state_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_lifecycle_state ATTACH PARTITION public.devices_cmcc_lifecycle_state_idx;


--
-- Name: devices_cmcc_oui_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_oui ATTACH PARTITION public.devices_cmcc_oui_idx;


--
-- Name: devices_cmcc_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.devices_pkey ATTACH PARTITION public.devices_cmcc_pkey;


--
-- Name: devices_cmcc_serial_number_carrier_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_serial_number ATTACH PARTITION public.devices_cmcc_serial_number_carrier_idx;


--
-- Name: devices_ctcc_carrier_last_inform_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_carrier_alive ATTACH PARTITION public.devices_ctcc_carrier_last_inform_at_idx;


--
-- Name: devices_ctcc_carrier_lifecycle_state_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_carrier_lifecycle ATTACH PARTITION public.devices_ctcc_carrier_lifecycle_state_idx;


--
-- Name: devices_ctcc_carrier_technology_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_carrier_tech ATTACH PARTITION public.devices_ctcc_carrier_technology_idx;


--
-- Name: devices_ctcc_deleted_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_deleted_at ATTACH PARTITION public.devices_ctcc_deleted_at_idx;


--
-- Name: devices_ctcc_extension_data_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_extension_data_gin ATTACH PARTITION public.devices_ctcc_extension_data_idx;


--
-- Name: devices_ctcc_ip_address_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_ip_address ATTACH PARTITION public.devices_ctcc_ip_address_idx;


--
-- Name: devices_ctcc_is_online_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_is_online ATTACH PARTITION public.devices_ctcc_is_online_idx;


--
-- Name: devices_ctcc_last_boot_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_last_boot_at ATTACH PARTITION public.devices_ctcc_last_boot_at_idx;


--
-- Name: devices_ctcc_last_inform_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_last_inform ATTACH PARTITION public.devices_ctcc_last_inform_at_idx;


--
-- Name: devices_ctcc_last_inform_events_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_last_inform_events_gin ATTACH PARTITION public.devices_ctcc_last_inform_events_idx;


--
-- Name: devices_ctcc_lifecycle_state_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_lifecycle_state ATTACH PARTITION public.devices_ctcc_lifecycle_state_idx;


--
-- Name: devices_ctcc_oui_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_oui ATTACH PARTITION public.devices_ctcc_oui_idx;


--
-- Name: devices_ctcc_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.devices_pkey ATTACH PARTITION public.devices_ctcc_pkey;


--
-- Name: devices_ctcc_serial_number_carrier_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_serial_number ATTACH PARTITION public.devices_ctcc_serial_number_carrier_idx;


--
-- Name: devices_cucc_carrier_last_inform_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_carrier_alive ATTACH PARTITION public.devices_cucc_carrier_last_inform_at_idx;


--
-- Name: devices_cucc_carrier_lifecycle_state_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_carrier_lifecycle ATTACH PARTITION public.devices_cucc_carrier_lifecycle_state_idx;


--
-- Name: devices_cucc_carrier_technology_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_carrier_tech ATTACH PARTITION public.devices_cucc_carrier_technology_idx;


--
-- Name: devices_cucc_deleted_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_deleted_at ATTACH PARTITION public.devices_cucc_deleted_at_idx;


--
-- Name: devices_cucc_extension_data_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_extension_data_gin ATTACH PARTITION public.devices_cucc_extension_data_idx;


--
-- Name: devices_cucc_ip_address_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_ip_address ATTACH PARTITION public.devices_cucc_ip_address_idx;


--
-- Name: devices_cucc_is_online_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_is_online ATTACH PARTITION public.devices_cucc_is_online_idx;


--
-- Name: devices_cucc_last_boot_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_last_boot_at ATTACH PARTITION public.devices_cucc_last_boot_at_idx;


--
-- Name: devices_cucc_last_inform_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_last_inform ATTACH PARTITION public.devices_cucc_last_inform_at_idx;


--
-- Name: devices_cucc_last_inform_events_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_last_inform_events_gin ATTACH PARTITION public.devices_cucc_last_inform_events_idx;


--
-- Name: devices_cucc_lifecycle_state_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_lifecycle_state ATTACH PARTITION public.devices_cucc_lifecycle_state_idx;


--
-- Name: devices_cucc_oui_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_oui ATTACH PARTITION public.devices_cucc_oui_idx;


--
-- Name: devices_cucc_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.devices_pkey ATTACH PARTITION public.devices_cucc_pkey;


--
-- Name: devices_cucc_serial_number_carrier_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_serial_number ATTACH PARTITION public.devices_cucc_serial_number_carrier_idx;


--
-- Name: devices_other_carrier_last_inform_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_carrier_alive ATTACH PARTITION public.devices_other_carrier_last_inform_at_idx;


--
-- Name: devices_other_carrier_lifecycle_state_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_carrier_lifecycle ATTACH PARTITION public.devices_other_carrier_lifecycle_state_idx;


--
-- Name: devices_other_carrier_technology_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_carrier_tech ATTACH PARTITION public.devices_other_carrier_technology_idx;


--
-- Name: devices_other_deleted_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_deleted_at ATTACH PARTITION public.devices_other_deleted_at_idx;


--
-- Name: devices_other_extension_data_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_extension_data_gin ATTACH PARTITION public.devices_other_extension_data_idx;


--
-- Name: devices_other_ip_address_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_ip_address ATTACH PARTITION public.devices_other_ip_address_idx;


--
-- Name: devices_other_is_online_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_is_online ATTACH PARTITION public.devices_other_is_online_idx;


--
-- Name: devices_other_last_boot_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_last_boot_at ATTACH PARTITION public.devices_other_last_boot_at_idx;


--
-- Name: devices_other_last_inform_at_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_last_inform ATTACH PARTITION public.devices_other_last_inform_at_idx;


--
-- Name: devices_other_last_inform_events_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_last_inform_events_gin ATTACH PARTITION public.devices_other_last_inform_events_idx;


--
-- Name: devices_other_lifecycle_state_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_lifecycle_state ATTACH PARTITION public.devices_other_lifecycle_state_idx;


--
-- Name: devices_other_oui_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_oui ATTACH PARTITION public.devices_other_oui_idx;


--
-- Name: devices_other_pkey; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.devices_pkey ATTACH PARTITION public.devices_other_pkey;


--
-- Name: devices_other_serial_number_carrier_idx; Type: INDEX ATTACH; Schema: public; Owner: -
--

ALTER INDEX public.idx_devices_serial_number ATTACH PARTITION public.devices_other_serial_number_carrier_idx;


--
-- Name: async_jobs trg_async_jobs_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_async_jobs_updated_at BEFORE UPDATE ON public.async_jobs FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: mml_command_sub_fields trg_mml_command_sub_fields_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_mml_command_sub_fields_updated_at BEFORE UPDATE ON public.mml_command_sub_fields FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: mml_commands trg_mml_commands_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_mml_commands_updated_at BEFORE UPDATE ON public.mml_commands FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: mml_command_sub_fields trg_mml_sub_fields_target_paths; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_mml_sub_fields_target_paths AFTER INSERT OR DELETE OR UPDATE ON public.mml_command_sub_fields FOR EACH ROW EXECUTE FUNCTION public.trg_mml_sub_fields_refresh_paths();


--
-- Name: mr_customize_task_progress trg_mr_customize_task_progress_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_mr_customize_task_progress_updated_at BEFORE UPDATE ON public.mr_customize_task_progress FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: mr_customize_task trg_mr_customize_task_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_mr_customize_task_updated_at BEFORE UPDATE ON public.mr_customize_task FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: nedirect_sessions trg_nedirect_sessions_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_nedirect_sessions_updated_at BEFORE UPDATE ON public.nedirect_sessions FOR EACH ROW EXECUTE FUNCTION public.nedirect_sessions_updated_at();


--
-- Name: northbound_endpoints trg_northbound_endpoints_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_northbound_endpoints_updated_at BEFORE UPDATE ON public.northbound_endpoints FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: northbound_field_mappings trg_northbound_field_mappings_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_northbound_field_mappings_updated_at BEFORE UPDATE ON public.northbound_field_mappings FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: northbound_file_profiles trg_northbound_file_profiles_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_northbound_file_profiles_updated_at BEFORE UPDATE ON public.northbound_file_profiles FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: northbound_file_runs trg_northbound_file_runs_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_northbound_file_runs_updated_at BEFORE UPDATE ON public.northbound_file_runs FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: northbound_delivery_targets trg_northbound_delivery_targets_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_northbound_delivery_targets_updated_at BEFORE UPDATE ON public.northbound_delivery_targets FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: northbound_snmp_alarm_targets trg_northbound_snmp_alarm_targets_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_northbound_snmp_alarm_targets_updated_at BEFORE UPDATE ON public.northbound_snmp_alarm_targets FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: northbound_socket_alarm_configs trg_northbound_socket_alarm_configs_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_northbound_socket_alarm_configs_updated_at BEFORE UPDATE ON public.northbound_socket_alarm_configs FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: northbound_api_configs trg_northbound_api_configs_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_northbound_api_configs_updated_at BEFORE UPDATE ON public.northbound_api_configs FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: northbound_api_clients trg_northbound_api_clients_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_northbound_api_clients_updated_at BEFORE UPDATE ON public.northbound_api_clients FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: northbound_page_config_events trg_northbound_page_config_events_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_northbound_page_config_events_updated_at BEFORE UPDATE ON public.northbound_page_config_events FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: northbound_inventory_profiles trg_northbound_inventory_profiles_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_northbound_inventory_profiles_updated_at BEFORE UPDATE ON public.northbound_inventory_profiles FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: northbound_servers trg_northbound_servers_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_northbound_servers_updated_at BEFORE UPDATE ON public.northbound_servers FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: restore_tasks trg_restore_tasks_updated; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_restore_tasks_updated BEFORE UPDATE ON public.restore_tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: alarm_definitions trigger_alarm_definitions_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_alarm_definitions_updated_at BEFORE UPDATE ON public.alarm_definitions FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: alarm_rules trigger_alarm_rules_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_alarm_rules_updated_at BEFORE UPDATE ON public.alarm_rules FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: async_jobs_cron_state trigger_async_jobs_cron_state_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_async_jobs_cron_state_updated_at BEFORE UPDATE ON public.async_jobs_cron_state FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: backup_schedules trigger_backup_schedules_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_backup_schedules_updated_at BEFORE UPDATE ON public.backup_schedules FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: backup_tasks trigger_backup_tasks_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_backup_tasks_updated_at BEFORE UPDATE ON public.backup_tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: config_backup_sub_tasks trigger_config_backup_sub_tasks_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_config_backup_sub_tasks_updated_at BEFORE UPDATE ON public.config_backup_sub_tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: config_backup_tasks trigger_config_backup_tasks_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_config_backup_tasks_updated_at BEFORE UPDATE ON public.config_backup_tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: config_baselines trigger_config_baselines_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_config_baselines_updated_at BEFORE UPDATE ON public.config_baselines FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: config_neighbors trigger_config_neighbors_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_config_neighbors_updated_at BEFORE UPDATE ON public.config_neighbors FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: config_restore_sub_tasks trigger_config_restore_sub_tasks_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_config_restore_sub_tasks_updated_at BEFORE UPDATE ON public.config_restore_sub_tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: config_restore_tasks trigger_config_restore_tasks_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_config_restore_tasks_updated_at BEFORE UPDATE ON public.config_restore_tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: config_tasks trigger_config_tasks_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_config_tasks_updated_at BEFORE UPDATE ON public.config_tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: config_templates trigger_ct_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_ct_updated_at BEFORE UPDATE ON public.config_templates FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: dashboard_widgets trigger_dashboard_widgets_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_dashboard_widgets_updated_at BEFORE UPDATE ON public.dashboard_widgets FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: device_info trigger_device_info_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_device_info_updated_at BEFORE UPDATE ON public.device_info FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: devices trigger_devices_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_devices_updated_at BEFORE UPDATE ON public.devices FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: device_groups trigger_dg_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_dg_updated_at BEFORE UPDATE ON public.device_groups FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: discovered_param_mappings trigger_discovered_param_mappings_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_discovered_param_mappings_updated_at BEFORE UPDATE ON public.discovered_param_mappings FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: device_registrations trigger_dr_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_dr_updated_at BEFORE UPDATE ON public.device_registrations FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: enabled_pm_indicators_enb trigger_enabled_pm_indicators_enb_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_enabled_pm_indicators_enb_updated_at BEFORE UPDATE ON public.enabled_pm_indicators_enb FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: enabled_pm_indicators_gnb trigger_enabled_pm_indicators_gnb_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_enabled_pm_indicators_gnb_updated_at BEFORE UPDATE ON public.enabled_pm_indicators_gnb FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: enabled_pm_indicators_gsm trigger_enabled_pm_indicators_gsm_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_enabled_pm_indicators_gsm_updated_at BEFORE UPDATE ON public.enabled_pm_indicators_gsm FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: fault_log_collect_sub_tasks trigger_fault_log_collect_sub_tasks_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_fault_log_collect_sub_tasks_updated_at BEFORE UPDATE ON public.fault_log_collect_sub_tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: fault_log_collect_tasks trigger_fault_log_collect_tasks_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_fault_log_collect_tasks_updated_at BEFORE UPDATE ON public.fault_log_collect_tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: firmware_versions trigger_firmware_versions_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_firmware_versions_updated_at BEFORE UPDATE ON public.firmware_versions FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: ftp_configs trigger_ftp_configs_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_ftp_configs_updated_at BEFORE UPDATE ON public.ftp_configs FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: indicator_group_enb trigger_indicator_group_enb_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_indicator_group_enb_updated_at BEFORE UPDATE ON public.indicator_group_enb FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: indicator_group_gnb trigger_indicator_group_gnb_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_indicator_group_gnb_updated_at BEFORE UPDATE ON public.indicator_group_gnb FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: indicator_group_gsm trigger_indicator_group_gsm_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_indicator_group_gsm_updated_at BEFORE UPDATE ON public.indicator_group_gsm FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: indicator_threshold trigger_indicator_threshold_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_indicator_threshold_updated_at BEFORE UPDATE ON public.indicator_threshold FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: kpi_thresholds trigger_kpi_thresholds_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_kpi_thresholds_updated_at BEFORE UPDATE ON public.kpi_thresholds FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: managed_files trigger_managed_files_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_managed_files_updated_at BEFORE UPDATE ON public.managed_files FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: mml_catalog_link_health trigger_mml_catalog_link_health_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_mml_catalog_link_health_updated_at BEFORE UPDATE ON public.mml_catalog_link_health FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: mml_custom_command trigger_mml_custom_command_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_mml_custom_command_updated_at BEFORE UPDATE ON public.mml_custom_command FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: mml_command_groups trigger_mml_param_groups_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_mml_param_groups_updated_at BEFORE UPDATE ON public.mml_command_groups FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: mml_scripts trigger_mml_scripts_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_mml_scripts_updated_at BEFORE UPDATE ON public.mml_scripts FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: mml_tasks trigger_mml_tasks_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_mml_tasks_updated_at BEFORE UPDATE ON public.mml_tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: mr_device_mappings trigger_mr_device_mappings_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_mr_device_mappings_updated_at BEFORE UPDATE ON public.mr_device_mappings FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: ops_maintenance_windows trigger_ops_mw_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_ops_mw_updated_at BEFORE UPDATE ON public.ops_maintenance_windows FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: ops_playbooks trigger_ops_pb_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_ops_pb_updated_at BEFORE UPDATE ON public.ops_playbooks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: ops_tasks trigger_ops_tasks_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_ops_tasks_updated_at BEFORE UPDATE ON public.ops_tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: ops_templates trigger_ops_templates_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_ops_templates_updated_at BEFORE UPDATE ON public.ops_templates FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: mml_catalog_orphan_paths_audit_t0171 trigger_orphan_paths_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_orphan_paths_updated_at BEFORE UPDATE ON public.mml_catalog_orphan_paths_audit_t0171 FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: param_mappings trigger_param_mappings_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_param_mappings_updated_at BEFORE UPDATE ON public.param_mappings FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: param_models trigger_param_models_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_param_models_updated_at BEFORE UPDATE ON public.param_models FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: perf_alarm_threshold trigger_perf_alarm_threshold_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_perf_alarm_threshold_updated_at BEFORE UPDATE ON public.perf_alarm_threshold FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: perf_cust_name trigger_perf_cust_name_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_perf_cust_name_updated_at BEFORE UPDATE ON public.perf_cust_name FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: perf_indicators_enb trigger_perf_indicators_enb_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_perf_indicators_enb_updated_at BEFORE UPDATE ON public.perf_indicators_enb FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: perf_indicators_gnb trigger_perf_indicators_gnb_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_perf_indicators_gnb_updated_at BEFORE UPDATE ON public.perf_indicators_gnb FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: perf_indicators_gsm trigger_perf_indicators_gsm_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_perf_indicators_gsm_updated_at BEFORE UPDATE ON public.perf_indicators_gsm FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: perf_template_rel_arithmetic trigger_perf_template_rel_arithmetic_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_perf_template_rel_arithmetic_updated_at BEFORE UPDATE ON public.perf_template_rel_arithmetic FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: pm_dashboards trigger_pm_dashboards_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_pm_dashboards_updated_at BEFORE UPDATE ON public.pm_dashboards FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: pm_panels trigger_pm_panels_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_pm_panels_updated_at BEFORE UPDATE ON public.pm_panels FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: pm_tasks trigger_pm_tasks_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_pm_tasks_updated_at BEFORE UPDATE ON public.pm_tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: pm_user_dashboard_preferences trigger_pm_user_dashboard_prefs_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_pm_user_dashboard_prefs_updated_at BEFORE UPDATE ON public.pm_user_dashboard_preferences FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: product_class_patterns trigger_product_class_patterns_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_product_class_patterns_updated_at BEFORE UPDATE ON public.product_class_patterns FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: products trigger_products_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_products_updated_at BEFORE UPDATE ON public.products FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: provisioning_tasks trigger_pt_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_pt_updated_at BEFORE UPDATE ON public.provisioning_tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: rela_platform_indicator_formula_enb trigger_rela_platform_indicator_formula_enb_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_rela_platform_indicator_formula_enb_updated_at BEFORE UPDATE ON public.rela_platform_indicator_formula_enb FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: rela_platform_indicator_formula_gnb trigger_rela_platform_indicator_formula_gnb_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_rela_platform_indicator_formula_gnb_updated_at BEFORE UPDATE ON public.rela_platform_indicator_formula_gnb FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: rela_platform_indicator_formula_gsm trigger_rela_platform_indicator_formula_gsm_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_rela_platform_indicator_formula_gsm_updated_at BEFORE UPDATE ON public.rela_platform_indicator_formula_gsm FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: report_definitions trigger_report_definitions_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_report_definitions_updated_at BEFORE UPDATE ON public.report_definitions FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: roles trigger_roles_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_roles_updated_at BEFORE UPDATE ON public.roles FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: runtime_log_collect_sub_tasks trigger_runtime_log_collect_sub_tasks_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_runtime_log_collect_sub_tasks_updated_at BEFORE UPDATE ON public.runtime_log_collect_sub_tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: runtime_log_collect_tasks trigger_runtime_log_collect_tasks_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_runtime_log_collect_tasks_updated_at BEFORE UPDATE ON public.runtime_log_collect_tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: sites trigger_sites_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_sites_updated_at BEFORE UPDATE ON public.sites FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: standard_params trigger_standard_params_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_standard_params_updated_at BEFORE UPDATE ON public.standard_params FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: mml_command_sub_field_overrides trigger_sub_field_overrides_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_sub_field_overrides_updated_at BEFORE UPDATE ON public.mml_command_sub_field_overrides FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: system_license trigger_system_license_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_system_license_updated_at BEFORE UPDATE ON public.system_license FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: topo_nodes trigger_topo_nodes_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_topo_nodes_updated_at BEFORE UPDATE ON public.topo_nodes FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: trace_export_jobs trigger_trace_export_jobs_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_trace_export_jobs_updated_at BEFORE UPDATE ON public.trace_export_jobs FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: trace_tasks trigger_trace_tasks_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_trace_tasks_updated_at BEFORE UPDATE ON public.trace_tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: user_column_configs trigger_ucc_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_ucc_updated_at BEFORE UPDATE ON public.user_column_configs FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: upgrade_sub_tasks trigger_upgrade_sub_tasks_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_upgrade_sub_tasks_updated_at BEFORE UPDATE ON public.upgrade_sub_tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: upgrade_tasks trigger_upgrade_tasks_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_upgrade_tasks_updated_at BEFORE UPDATE ON public.upgrade_tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: users trigger_users_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_users_updated_at BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: alarm_definitions alarm_definitions_severity_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alarm_definitions
    ADD CONSTRAINT alarm_definitions_severity_id_fkey FOREIGN KEY (severity_id) REFERENCES public.alarm_severity_levels(id) ON DELETE RESTRICT;


--
-- Name: alarm_webhook_dead_letters alarm_webhook_dead_letters_filter_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alarm_webhook_dead_letters
    ADD CONSTRAINT alarm_webhook_dead_letters_filter_id_fkey FOREIGN KEY (filter_id) REFERENCES public.alarm_filters(id) ON DELETE CASCADE;


--
-- Name: api_keys api_keys_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT api_keys_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: audit_logs audit_logs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_logs
    ADD CONSTRAINT audit_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: backup_policies backup_policies_ftp_config_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.backup_policies
    ADD CONSTRAINT backup_policies_ftp_config_id_fkey FOREIGN KEY (ftp_config_id) REFERENCES public.ftp_configs(id) ON DELETE SET NULL;


--
-- Name: config_backup_sub_tasks config_backup_sub_tasks_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_backup_sub_tasks
    ADD CONSTRAINT config_backup_sub_tasks_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.config_backup_tasks(id) ON DELETE CASCADE;


--
-- Name: config_restore_sub_tasks config_restore_sub_tasks_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_restore_sub_tasks
    ADD CONSTRAINT config_restore_sub_tasks_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.config_restore_tasks(id) ON DELETE CASCADE;


--
-- Name: dashboard_widgets dashboard_widgets_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dashboard_widgets
    ADD CONSTRAINT dashboard_widgets_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: device_group_members device_group_members_group_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_group_members
    ADD CONSTRAINT device_group_members_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.device_groups(id) ON DELETE CASCADE;


--
-- Name: device_groups device_groups_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_groups
    ADD CONSTRAINT device_groups_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.device_groups(id) ON DELETE CASCADE;


--
-- Name: device_groups device_groups_source_group_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_groups
    ADD CONSTRAINT device_groups_source_group_id_fkey FOREIGN KEY (source_group_id) REFERENCES public.device_groups(id) ON DELETE SET NULL;


--
-- Name: device_registrations device_registrations_group_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_registrations
    ADD CONSTRAINT device_registrations_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.device_groups(id);


--
-- Name: discovered_param_mappings discovered_param_mappings_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discovered_param_mappings
    ADD CONSTRAINT discovered_param_mappings_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE CASCADE;


--
-- Name: fault_log_collect_sub_tasks fault_log_collect_sub_tasks_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.fault_log_collect_sub_tasks
    ADD CONSTRAINT fault_log_collect_sub_tasks_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.fault_log_collect_tasks(id) ON DELETE CASCADE;


--
-- Name: mml_custom_command_paths fk_ccp_command; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_custom_command_paths
    ADD CONSTRAINT fk_ccp_command FOREIGN KEY (command_id) REFERENCES public.mml_custom_command(id) ON DELETE CASCADE;


--
-- Name: mml_custom_command_paths fk_ccp_standard_path; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_custom_command_paths
    ADD CONSTRAINT fk_ccp_standard_path FOREIGN KEY (standard_path_id) REFERENCES public.standard_params(id) ON DELETE RESTRICT;


--
-- Name: mml_custom_command fk_mml_custom_command_owner; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_custom_command
    ADD CONSTRAINT fk_mml_custom_command_owner FOREIGN KEY (owner_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: parameter_discovery_log fk_pdl_param_model; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_discovery_log
    ADD CONSTRAINT fk_pdl_param_model FOREIGN KEY (param_model_id) REFERENCES public.param_models(id) ON DELETE SET NULL;


--
-- Name: pm_user_dashboard_preferences fk_pm_user_prefs_dashboard; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pm_user_dashboard_preferences
    ADD CONSTRAINT fk_pm_user_prefs_dashboard FOREIGN KEY (current_dashboard_id) REFERENCES public.pm_dashboards(id) ON DELETE SET NULL;


--
-- Name: products fk_products_param_model; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.products
    ADD CONSTRAINT fk_products_param_model FOREIGN KEY (param_model_id) REFERENCES public.param_models(id) ON DELETE SET NULL;


--
-- Name: config_apply_targets config_apply_targets_batch_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_apply_targets
    ADD CONSTRAINT config_apply_targets_batch_id_fkey FOREIGN KEY (batch_id) REFERENCES public.config_apply_batches(id) ON DELETE CASCADE;


--
-- Name: system_license_history fk_replaced_by; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_license_history
    ADD CONSTRAINT fk_replaced_by FOREIGN KEY (replaced_by_id) REFERENCES public.system_license(id) ON DELETE SET NULL;


--
-- Name: menus menus_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.menus
    ADD CONSTRAINT menus_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: menus menus_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.menus
    ADD CONSTRAINT menus_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.menus(id) ON DELETE CASCADE;


--
-- Name: menus menus_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.menus
    ADD CONSTRAINT menus_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: mml_audit_log mml_audit_log_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_audit_log
    ADD CONSTRAINT mml_audit_log_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.mml_tasks(id) ON DELETE SET NULL;


--
-- Name: mml_catalog_orphan_paths_audit_t0171 mml_catalog_orphan_paths_audit_t0171_standard_param_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_catalog_orphan_paths_audit_t0171
    ADD CONSTRAINT mml_catalog_orphan_paths_audit_t0171_standard_param_id_fkey FOREIGN KEY (standard_param_id) REFERENCES public.standard_params(id) ON DELETE CASCADE;


--
-- Name: mml_command_sub_field_overrides mml_command_sub_field_overrides_sub_field_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_command_sub_field_overrides
    ADD CONSTRAINT mml_command_sub_field_overrides_sub_field_id_fkey FOREIGN KEY (sub_field_id) REFERENCES public.mml_command_sub_fields(id) ON DELETE CASCADE;


--
-- Name: mml_command_sub_fields mml_command_sub_fields_command_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_command_sub_fields
    ADD CONSTRAINT mml_command_sub_fields_command_id_fkey FOREIGN KEY (command_id) REFERENCES public.mml_commands(id) ON DELETE CASCADE;


--
-- Name: mml_command_sub_fields mml_command_sub_fields_standard_path_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_command_sub_fields
    ADD CONSTRAINT mml_command_sub_fields_standard_path_id_fkey FOREIGN KEY (standard_path_id) REFERENCES public.standard_params(id) ON DELETE RESTRICT;


--
-- Name: mml_commands mml_commands_group_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_commands
    ADD CONSTRAINT mml_commands_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.mml_command_groups(id) ON DELETE SET NULL;


--
-- Name: mml_command_groups mml_param_groups_param_version_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_command_groups
    ADD CONSTRAINT mml_param_groups_param_version_fkey FOREIGN KEY (param_version) REFERENCES public.mml_param_versions(version_code) ON DELETE RESTRICT;


--
-- Name: mml_tasks mml_tasks_script_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mml_tasks
    ADD CONSTRAINT mml_tasks_script_id_fkey FOREIGN KEY (script_id) REFERENCES public.mml_scripts(id) ON DELETE SET NULL;


--
-- Name: mr_customize_task_progress mr_customize_task_progress_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mr_customize_task_progress
    ADD CONSTRAINT mr_customize_task_progress_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.mr_customize_task(task_id) ON DELETE CASCADE;


--
-- Name: nedirect_commands nedirect_commands_session_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.nedirect_commands
    ADD CONSTRAINT nedirect_commands_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.nedirect_sessions(id) ON DELETE CASCADE;


--
-- Name: notification_history notification_history_template_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification_history
    ADD CONSTRAINT notification_history_template_id_fkey FOREIGN KEY (template_id) REFERENCES public.notification_templates(id) ON DELETE SET NULL;


--
-- Name: ops_audit_logs ops_audit_logs_approver_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ops_audit_logs
    ADD CONSTRAINT ops_audit_logs_approver_user_id_fkey FOREIGN KEY (approver_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: ops_audit_logs ops_audit_logs_operator_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ops_audit_logs
    ADD CONSTRAINT ops_audit_logs_operator_user_id_fkey FOREIGN KEY (operator_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: ops_diagnostics ops_diagnostics_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ops_diagnostics
    ADD CONSTRAINT ops_diagnostics_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.ops_tasks(id) ON DELETE SET NULL;


--
-- Name: ops_downloads ops_downloads_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ops_downloads
    ADD CONSTRAINT ops_downloads_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.ops_tasks(id) ON DELETE SET NULL;


--
-- Name: ops_maintenance_windows ops_maintenance_windows_approver_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ops_maintenance_windows
    ADD CONSTRAINT ops_maintenance_windows_approver_user_id_fkey FOREIGN KEY (approver_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: ops_maintenance_windows ops_maintenance_windows_creator_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ops_maintenance_windows
    ADD CONSTRAINT ops_maintenance_windows_creator_user_id_fkey FOREIGN KEY (creator_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: ops_task_executions ops_task_executions_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ops_task_executions
    ADD CONSTRAINT ops_task_executions_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.ops_tasks(id) ON DELETE CASCADE;


--
-- Name: ops_tasks ops_tasks_approver_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ops_tasks
    ADD CONSTRAINT ops_tasks_approver_user_id_fkey FOREIGN KEY (approver_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: ops_tasks ops_tasks_template_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ops_tasks
    ADD CONSTRAINT ops_tasks_template_id_fkey FOREIGN KEY (template_id) REFERENCES public.ops_templates(id) ON DELETE SET NULL;


--
-- Name: ops_templates ops_templates_owner_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ops_templates
    ADD CONSTRAINT ops_templates_owner_user_id_fkey FOREIGN KEY (owner_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: param_mappings param_mappings_param_model_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.param_mappings
    ADD CONSTRAINT param_mappings_param_model_id_fkey FOREIGN KEY (param_model_id) REFERENCES public.param_models(id) ON DELETE CASCADE;


--
-- Name: parameter_sync_admission_reservations parameter_sync_admission_reservations_request_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_admission_reservations
    ADD CONSTRAINT parameter_sync_admission_reservations_request_id_fkey FOREIGN KEY (request_id) REFERENCES public.parameter_sync_requests(id) ON DELETE CASCADE;


--
-- Name: parameter_sync_request_bindings parameter_sync_request_bindings_provisioning_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_request_bindings
    ADD CONSTRAINT parameter_sync_request_bindings_provisioning_task_id_fkey FOREIGN KEY (provisioning_task_id) REFERENCES public.provisioning_tasks(id) ON DELETE CASCADE;


--
-- Name: parameter_sync_request_bindings parameter_sync_request_bindings_request_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_request_bindings
    ADD CONSTRAINT parameter_sync_request_bindings_request_id_fkey FOREIGN KEY (request_id) REFERENCES public.parameter_sync_requests(id) ON DELETE CASCADE;


--
-- Name: parameter_sync_request_bindings parameter_sync_request_bindings_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_request_bindings
    ADD CONSTRAINT parameter_sync_request_bindings_run_id_fkey FOREIGN KEY (run_id) REFERENCES public.parameter_sync_runs(id) ON DELETE CASCADE;


--
-- Name: parameter_sync_requests parameter_sync_requests_active_run_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_requests
    ADD CONSTRAINT parameter_sync_requests_active_run_fk FOREIGN KEY (active_run_id) REFERENCES public.parameter_sync_runs(id) ON DELETE SET NULL;


--
-- Name: parameter_sync_requests parameter_sync_requests_run_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_requests
    ADD CONSTRAINT parameter_sync_requests_run_fk FOREIGN KEY (run_id) REFERENCES public.parameter_sync_runs(id) ON DELETE SET NULL;


--
-- Name: parameter_sync_runs parameter_sync_runs_request_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_runs
    ADD CONSTRAINT parameter_sync_runs_request_id_fkey FOREIGN KEY (request_id) REFERENCES public.parameter_sync_requests(id) ON DELETE CASCADE;


--
-- Name: parameter_sync_staging_values parameter_sync_staging_values_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_staging_values
    ADD CONSTRAINT parameter_sync_staging_values_run_id_fkey FOREIGN KEY (run_id) REFERENCES public.parameter_sync_runs(id) ON DELETE CASCADE;


--
-- Name: parameter_sync_task_results parameter_sync_task_results_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parameter_sync_task_results
    ADD CONSTRAINT parameter_sync_task_results_run_id_fkey FOREIGN KEY (run_id) REFERENCES public.parameter_sync_runs(id) ON DELETE CASCADE;


--
-- Name: pm_dashboards pm_dashboards_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pm_dashboards
    ADD CONSTRAINT pm_dashboards_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: pm_dashboards pm_dashboards_parent_dashboard_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pm_dashboards
    ADD CONSTRAINT pm_dashboards_parent_dashboard_id_fkey FOREIGN KEY (parent_dashboard_id) REFERENCES public.pm_dashboards(id) ON DELETE SET NULL;


--
-- Name: pm_panels pm_panels_dashboard_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pm_panels
    ADD CONSTRAINT pm_panels_dashboard_id_fkey FOREIGN KEY (dashboard_id) REFERENCES public.pm_dashboards(id) ON DELETE CASCADE;


--
-- Name: pm_user_dashboard_preferences pm_user_dashboard_preferences_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pm_user_dashboard_preferences
    ADD CONSTRAINT pm_user_dashboard_preferences_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: product_class_patterns product_class_patterns_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_class_patterns
    ADD CONSTRAINT product_class_patterns_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE CASCADE;


--
-- Name: provisioning_tasks provisioning_tasks_template_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.provisioning_tasks
    ADD CONSTRAINT provisioning_tasks_template_id_fkey FOREIGN KEY (template_id) REFERENCES public.config_templates(id);


--
-- Name: report_records report_records_report_definition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.report_records
    ADD CONSTRAINT report_records_report_definition_id_fkey FOREIGN KEY (report_definition_id) REFERENCES public.report_definitions(id) ON DELETE CASCADE;


--
-- Name: role_api_permissions role_api_permissions_endpoint_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_api_permissions
    ADD CONSTRAINT role_api_permissions_endpoint_id_fkey FOREIGN KEY (endpoint_id) REFERENCES public.api_endpoints(id) ON DELETE CASCADE;


--
-- Name: role_api_permissions role_api_permissions_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_api_permissions
    ADD CONSTRAINT role_api_permissions_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE;


--
-- Name: role_device_groups role_device_groups_group_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_device_groups
    ADD CONSTRAINT role_device_groups_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.device_groups(id) ON DELETE CASCADE;


--
-- Name: role_device_groups role_device_groups_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_device_groups
    ADD CONSTRAINT role_device_groups_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE;


--
-- Name: role_inheritance role_inheritance_child_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_inheritance
    ADD CONSTRAINT role_inheritance_child_role_id_fkey FOREIGN KEY (child_role_id) REFERENCES public.roles(id) ON DELETE CASCADE;


--
-- Name: role_inheritance role_inheritance_parent_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_inheritance
    ADD CONSTRAINT role_inheritance_parent_role_id_fkey FOREIGN KEY (parent_role_id) REFERENCES public.roles(id) ON DELETE CASCADE;


--
-- Name: role_menus role_menus_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_menus
    ADD CONSTRAINT role_menus_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: role_menus role_menus_menu_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_menus
    ADD CONSTRAINT role_menus_menu_id_fkey FOREIGN KEY (menu_id) REFERENCES public.menus(id) ON DELETE CASCADE;


--
-- Name: role_menus role_menus_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_menus
    ADD CONSTRAINT role_menus_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE;


--
-- Name: roles roles_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: roles roles_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: runtime_log_collect_sub_tasks runtime_log_collect_sub_tasks_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.runtime_log_collect_sub_tasks
    ADD CONSTRAINT runtime_log_collect_sub_tasks_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.runtime_log_collect_tasks(id) ON DELETE CASCADE;


--
-- Name: sites sites_domain_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sites
    ADD CONSTRAINT sites_domain_id_fkey FOREIGN KEY (domain_id) REFERENCES public.device_groups(id) ON DELETE SET NULL;


--
-- Name: standard_commands standard_commands_version_code_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.standard_commands
    ADD CONSTRAINT standard_commands_version_code_fkey FOREIGN KEY (version_code) REFERENCES public.mml_param_versions(version_code) ON DELETE CASCADE;


--
-- Name: station_fault_logs station_fault_logs_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.station_fault_logs
    ADD CONSTRAINT station_fault_logs_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.upgrade_tasks(id) ON DELETE SET NULL;


--
-- Name: station_running_logs station_running_logs_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.station_running_logs
    ADD CONSTRAINT station_running_logs_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.upgrade_tasks(id) ON DELETE SET NULL;


--
-- Name: sys_dictionary_details sys_dictionary_details_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sys_dictionary_details
    ADD CONSTRAINT sys_dictionary_details_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.sys_dictionary_details(id) ON DELETE CASCADE;


--
-- Name: sys_dictionary_details sys_dictionary_details_sys_dictionary_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sys_dictionary_details
    ADD CONSTRAINT sys_dictionary_details_sys_dictionary_id_fkey FOREIGN KEY (sys_dictionary_id) REFERENCES public.sys_dictionaries(id) ON DELETE CASCADE;


--
-- Name: sys_login_logs sys_login_logs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sys_login_logs
    ADD CONSTRAINT sys_login_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: sys_oper_logs sys_oper_logs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sys_oper_logs
    ADD CONSTRAINT sys_oper_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: sys_task_logs sys_task_logs_operator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sys_task_logs
    ADD CONSTRAINT sys_task_logs_operator_id_fkey FOREIGN KEY (operator_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: topo_edges topo_edges_source_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.topo_edges
    ADD CONSTRAINT topo_edges_source_id_fkey FOREIGN KEY (source_id) REFERENCES public.topo_nodes(id) ON DELETE CASCADE;


--
-- Name: topo_edges topo_edges_target_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.topo_edges
    ADD CONSTRAINT topo_edges_target_id_fkey FOREIGN KEY (target_id) REFERENCES public.topo_nodes(id) ON DELETE CASCADE;


--
-- Name: topo_nodes topo_nodes_domain_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.topo_nodes
    ADD CONSTRAINT topo_nodes_domain_id_fkey FOREIGN KEY (domain_id) REFERENCES public.device_groups(id) ON DELETE SET NULL;


--
-- Name: topo_nodes topo_nodes_site_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.topo_nodes
    ADD CONSTRAINT topo_nodes_site_id_fkey FOREIGN KEY (site_id) REFERENCES public.sites(id) ON DELETE SET NULL;


--
-- Name: trace_export_jobs trace_export_jobs_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trace_export_jobs
    ADD CONSTRAINT trace_export_jobs_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.trace_tasks(id) ON DELETE CASCADE;


--
-- Name: upgrade_sub_tasks upgrade_sub_tasks_firmware_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.upgrade_sub_tasks
    ADD CONSTRAINT upgrade_sub_tasks_firmware_id_fkey FOREIGN KEY (firmware_id) REFERENCES public.firmware_versions(id) ON DELETE SET NULL;


--
-- Name: upgrade_sub_tasks upgrade_sub_tasks_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.upgrade_sub_tasks
    ADD CONSTRAINT upgrade_sub_tasks_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.upgrade_tasks(id) ON DELETE CASCADE;


--
-- Name: upgrade_tasks upgrade_tasks_firmware_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.upgrade_tasks
    ADD CONSTRAINT upgrade_tasks_firmware_id_fkey FOREIGN KEY (firmware_id) REFERENCES public.firmware_versions(id) ON DELETE SET NULL;


--
-- Name: upgrade_tasks upgrade_tasks_rollback_target_firmware_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.upgrade_tasks
    ADD CONSTRAINT upgrade_tasks_rollback_target_firmware_id_fkey FOREIGN KEY (rollback_target_firmware_id) REFERENCES public.firmware_versions(id) ON DELETE SET NULL;


--
-- Name: user_column_configs user_column_configs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_column_configs
    ADD CONSTRAINT user_column_configs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_roles user_roles_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE;


--
-- Name: user_roles user_roles_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: users users_created_by_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: users users_updated_by_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_updated_by_fk FOREIGN KEY (updated_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- PostgreSQL database dump complete
--

SELECT pg_catalog.set_config('search_path', 'public', false);

-- Consolidated from former incremental migrations: main schema 000003-000007

ALTER TABLE param_mappings
    ADD COLUMN IF NOT EXISTS default_value text,
    ADD COLUMN IF NOT EXISTS validation_pattern text;

ALTER TABLE discovered_param_mappings
    ADD COLUMN IF NOT EXISTS default_value text,
    ADD COLUMN IF NOT EXISTS validation_pattern text;

COMMENT ON COLUMN param_mappings.default_value IS '参数模型 XML defaultValue；用于 MML 控制台默认填值提示';
COMMENT ON COLUMN param_mappings.validation_pattern IS '参数模型 XML validationPattern；用于 MML 控制台输入校验';
COMMENT ON COLUMN discovered_param_mappings.default_value IS '从 param_mappings 继承的 defaultValue';
COMMENT ON COLUMN discovered_param_mappings.validation_pattern IS '从 param_mappings 继承的 validationPattern';

ALTER TABLE parameter_sync_requests
    DROP CONSTRAINT parameter_sync_requests_trigger_reason_chk,
    ADD CONSTRAINT parameter_sync_requests_trigger_reason_chk
        CHECK (trigger_reason::text = ANY (ARRAY[
            'bootstrap', 'model_upload', 'device_online', 'firmware_changed',
            'periodic', 'manual', 'config_pull', 'license', 'spv_readback',
            'add_object_readback', 'inform_period_probe', 'device_registered',
            'omc_upgrade'
        ]::text[]));

ALTER TABLE parameter_sync_runs
    DROP CONSTRAINT parameter_sync_runs_trigger_reason_chk,
    ADD CONSTRAINT parameter_sync_runs_trigger_reason_chk
        CHECK (trigger_reason::text = ANY (ARRAY[
            'bootstrap', 'model_upload', 'device_online', 'firmware_changed',
            'periodic', 'manual', 'config_pull', 'license', 'spv_readback',
            'add_object_readback', 'inform_period_probe', 'device_registered',
            'omc_upgrade'
        ]::text[]));

-- Bounded recovery metadata for terminal hourly aggregation jobs.
ALTER TABLE async_jobs
    ADD COLUMN IF NOT EXISTS recovery_count integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_recovered_at timestamptz;

CREATE INDEX IF NOT EXISTS idx_async_jobs_hourly_failed_recovery
    ON async_jobs (bucket_start, recovery_count, last_recovered_at)
    WHERE job_type = 'pm_aggregate_hourly'
      AND status = 'failed'
      AND bucket_start IS NOT NULL
      AND bucket_end IS NOT NULL;

-- PM 聚合正式切换：旧任务、运行记录和完成水位不迁移。
DROP TRIGGER IF EXISTS trg_ops_pause_pm_natural_aggregation ON public.async_jobs;
DROP TRIGGER IF EXISTS trg_ops_pause_pm_adhoc_aggregation ON public.pm_tasks;
DROP FUNCTION IF EXISTS public.ops_pause_pm_natural_aggregation();
DROP FUNCTION IF EXISTS public.ops_pause_pm_adhoc_aggregation();

DELETE FROM public.async_jobs
 WHERE job_type IN (
    'pm_aggregate_hourly', 'pm_aggregate_daily', 'pm_aggregate_weekly', 'pm_aggregate_monthly',
    'pm_aggregate_group_hourly', 'pm_aggregate_group_daily',
    'pm_aggregate_group_weekly', 'pm_aggregate_group_monthly'
 );
DELETE FROM public.async_jobs_cron_state
 WHERE job_type IN (
    'pm_aggregate_hourly', 'pm_aggregate_daily', 'pm_aggregate_weekly', 'pm_aggregate_monthly',
    'pm_aggregate_group_hourly', 'pm_aggregate_group_daily',
    'pm_aggregate_group_weekly', 'pm_aggregate_group_monthly'
 );

TRUNCATE TABLE public.pm_adhoc_task_runs;
TRUNCATE TABLE public.pm_tasks;
DROP TABLE IF EXISTS public.pm_completion_watermarks;

CREATE TABLE public.pm_aggregation_tasks (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name varchar(200) NOT NULL,
    enabled boolean NOT NULL DEFAULT true,
    visibility varchar(16) NOT NULL DEFAULT 'private',
    creator varchar(100) NOT NULL,
    current_version_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CONSTRAINT chk_pm_aggregation_tasks_visibility
        CHECK (visibility IN ('private', 'public'))
);

CREATE TABLE public.pm_aggregation_task_versions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id uuid NOT NULL REFERENCES public.pm_aggregation_tasks(id) ON DELETE CASCADE,
    version_no integer NOT NULL,
    enabled boolean NOT NULL,
    effective_from timestamptz NOT NULL,
    effective_to timestamptz,
    technology varchar(16),
    dimension varchar(32) NOT NULL,
    granularities text[] NOT NULL,
    object_ldns text[] NOT NULL DEFAULT '{}',
    created_by varchar(100) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_pm_aggregation_task_versions UNIQUE (task_id, version_no),
    CONSTRAINT chk_pm_aggregation_task_versions_time
        CHECK (effective_to IS NULL OR effective_to >= effective_from),
    CONSTRAINT chk_pm_aggregation_task_versions_technology
        CHECK (technology IS NULL OR technology IN ('lte', 'nr', 'gsm')),
    CONSTRAINT chk_pm_aggregation_task_versions_dimension
        CHECK (dimension IN ('device', 'aggregate_group', 'device_group', 'product', 'band', 'network')),
    CONSTRAINT chk_pm_aggregation_task_versions_granularities
        CHECK (
            cardinality(granularities) = 4
            AND granularities @> ARRAY['hourly', 'daily', 'weekly', 'monthly']::text[]
        )
);

ALTER TABLE public.pm_aggregation_tasks
    ADD CONSTRAINT fk_pm_aggregation_tasks_current_version
    FOREIGN KEY (current_version_id)
    REFERENCES public.pm_aggregation_task_versions(id)
    ON DELETE SET NULL
    DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE public.pm_aggregation_version_metrics (
    task_version_id uuid NOT NULL
        REFERENCES public.pm_aggregation_task_versions(id) ON DELETE CASCADE,
    metric_id text NOT NULL,
    metric_path text NOT NULL,
    metric_type varchar(16) NOT NULL,
    aggregation_op varchar(8) NOT NULL,
    formula text NOT NULL DEFAULT '',
    dependencies text[] NOT NULL DEFAULT '{}',
    PRIMARY KEY (task_version_id, metric_id),
    CONSTRAINT chk_pm_aggregation_version_metrics_type
        CHECK (metric_type IN ('counter', 'kpi')),
    CONSTRAINT chk_pm_aggregation_version_metrics_op
        CHECK (aggregation_op IN ('sum', 'avg', 'min', 'max', 'formula'))
);

CREATE TABLE public.pm_aggregation_version_counters (
    task_version_id uuid NOT NULL
        REFERENCES public.pm_aggregation_task_versions(id) ON DELETE CASCADE,
    metric_path text NOT NULL,
    aggregation_op varchar(8) NOT NULL,
    PRIMARY KEY (task_version_id, metric_path),
    CONSTRAINT chk_pm_aggregation_version_counters_op
        CHECK (aggregation_op IN ('sum', 'avg', 'min', 'max'))
);

CREATE TABLE public.pm_aggregation_version_members (
    task_version_id uuid NOT NULL
        REFERENCES public.pm_aggregation_task_versions(id) ON DELETE CASCADE,
    device_id uuid NOT NULL,
    device_sn varchar(128) NOT NULL,
    dimension_key text NOT NULL,
    dimension_name text NOT NULL DEFAULT '',
    object_ldn text NOT NULL DEFAULT '',
    PRIMARY KEY (task_version_id, device_id, dimension_key, object_ldn)
);

CREATE INDEX idx_pm_aggregation_tasks_active
    ON public.pm_aggregation_tasks (enabled, updated_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_pm_aggregation_tasks_creator
    ON public.pm_aggregation_tasks (creator, visibility, updated_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_pm_aggregation_versions_effective
    ON public.pm_aggregation_task_versions (effective_from, effective_to);
CREATE INDEX idx_pm_aggregation_version_members_device
    ON public.pm_aggregation_version_members (device_id, task_version_id);
CREATE INDEX idx_pm_aggregation_version_metrics_path
    ON public.pm_aggregation_version_metrics (metric_path, task_version_id);

CREATE TRIGGER trigger_pm_aggregation_tasks_updated_at
    BEFORE UPDATE ON public.pm_aggregation_tasks
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

ALTER TABLE public.pm_aggregation_task_versions
    ADD COLUMN content_hash bytea;

CREATE INDEX idx_pm_aggregation_task_versions_content_hash
    ON public.pm_aggregation_task_versions (task_id, content_hash);

-- +omcgo MainReconcileBegin
-- Existing pre-release databases may already record goose version 1 while
-- missing this consolidated additive block. Keep it idempotent so migrate can
-- replay it after the seed baseline without rebuilding the database.
CREATE TABLE IF NOT EXISTS public.plug_and_play_policies (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    name varchar(100) NOT NULL,
    enabled boolean NOT NULL DEFAULT false,
    product_class varchar(128) NOT NULL,
    product_classes varchar(128)[] NOT NULL DEFAULT '{}'::varchar[],
    execute_type varchar(16) NOT NULL DEFAULT 'manual',
    priority integer NOT NULL DEFAULT 100,
    upgrade_enabled boolean NOT NULL DEFAULT false,
    target_version varchar(128),
    license_enabled boolean NOT NULL DEFAULT false,
    self_config_enabled boolean NOT NULL DEFAULT false,
    config jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CONSTRAINT plug_and_play_policies_execute_type_check
        CHECK (execute_type IN ('auto', 'manual'))
);

CREATE INDEX IF NOT EXISTS idx_plug_and_play_policies_match
    ON public.plug_and_play_policies (enabled, product_class, priority, created_at);

CREATE INDEX IF NOT EXISTS idx_plug_and_play_policies_product_classes
    ON public.plug_and_play_policies USING gin (product_classes);

CREATE TABLE IF NOT EXISTS public.provisioning_xml_files (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    policy_id uuid NOT NULL REFERENCES public.plug_and_play_policies(id),
    device_id uuid NOT NULL,
    file_name varchar(255) NOT NULL,
    content text NOT NULL,
    checksum varchar(64) NOT NULL,
    download_token uuid NOT NULL DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_provisioning_xml_files_device
    ON public.provisioning_xml_files (device_id, created_at DESC);

ALTER TABLE public.provisioning_tasks
    ADD COLUMN IF NOT EXISTS policy_id uuid REFERENCES public.plug_and_play_policies(id),
    ADD COLUMN IF NOT EXISTS xml_file_id uuid REFERENCES public.provisioning_xml_files(id),
    ADD COLUMN IF NOT EXISTS device_task_id uuid,
    ADD COLUMN IF NOT EXISTS current_step_name varchar(64);

CREATE INDEX IF NOT EXISTS idx_provisioning_tasks_policy
    ON public.provisioning_tasks (policy_id);
-- +omcgo MainReconcileEnd


-- Consolidated from pre-release baseline-only migrations: main schema 000002-000005

ALTER TABLE public.pm_tasks
    ADD COLUMN IF NOT EXISTS planned_end_at timestamp with time zone;

COMMENT ON COLUMN public.pm_tasks.planned_end_at IS
    'PM adhoc 自建 continuous 任务计划结束时间；NULL 表示老任务或内置任务不设置计划结束。';

ALTER TABLE public.pm_aggregation_tasks
    ADD COLUMN IF NOT EXISTS planned_end_at timestamp with time zone;

COMMENT ON COLUMN public.pm_aggregation_tasks.planned_end_at IS
    '流式聚合任务计划结束时间；NULL 表示不设置计划结束。';

ALTER TABLE public.rela_platform_indicator_formula_enb
    ADD COLUMN IF NOT EXISTS report_key text;
ALTER TABLE public.rela_platform_indicator_formula_gsm
    ADD COLUMN IF NOT EXISTS report_key text;
ALTER TABLE public.rela_platform_indicator_formula_gnb
    ADD COLUMN IF NOT EXISTS report_key text;

CREATE INDEX IF NOT EXISTS idx_device_tasks_open_method_description
    ON public.device_tasks (device_sn, method, description, created_at DESC)
    WHERE status IN ('pending', 'sent');

CREATE INDEX IF NOT EXISTS idx_device_tasks_completed_command_lookup
    ON public.device_tasks (device_sn, command_key, completed_at DESC, created_at DESC)
    WHERE status = 'completed';

CREATE INDEX IF NOT EXISTS idx_device_tasks_active_created_id
    ON public.device_tasks (created_at, id)
    WHERE status IN ('pending', 'sent');

-- Consolidated from pre-release storage protection migrations. The current
-- deployment has one physical filesystem target; logical components share it.
CREATE TABLE IF NOT EXISTS public.storage_protection_policies (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type varchar(32) NOT NULL,
    target_id varchar(128) NOT NULL,
    write_scope varchar(32) NOT NULL,
    enabled boolean NOT NULL DEFAULT false,
    warn_used_percent integer NOT NULL DEFAULT 80,
    block_used_percent integer NOT NULL DEFAULT 90,
    recover_used_percent integer NOT NULL DEFAULT 85,
    check_interval_seconds integer NOT NULL DEFAULT 30,
    unknown_behavior varchar(32) NOT NULL DEFAULT 'allow_with_alarm',
    current_state varchar(16) NOT NULL DEFAULT 'normal',
    state_observations integer NOT NULL DEFAULT 0,
    last_observed_ratio double precision,
    last_observed_at timestamptz,
    last_state_changed_at timestamptz,
    updated_by varchar(64) NOT NULL DEFAULT 'system',
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT storage_protection_thresholds_chk CHECK (
        warn_used_percent >= 0 AND
        warn_used_percent < recover_used_percent AND
        recover_used_percent < block_used_percent AND
        block_used_percent <= 100
    ),
    CONSTRAINT storage_protection_target_type_chk CHECK (
        target_type = 'filesystem' AND target_id = 'root' AND write_scope = 'all'
    ),
    CONSTRAINT storage_protection_unknown_behavior_chk CHECK (
        unknown_behavior IN ('allow_with_alarm', 'block_new_uploads')
    ),
    CONSTRAINT storage_protection_state_chk CHECK (
        current_state IN ('normal', 'warning', 'blocked', 'unknown')
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS storage_protection_policy_target_scope_uq
    ON public.storage_protection_policies (target_type, target_id, write_scope);

CREATE TABLE IF NOT EXISTS public.storage_protection_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id uuid NOT NULL REFERENCES public.storage_protection_policies(id) ON DELETE CASCADE,
    target_type varchar(32) NOT NULL,
    target_id varchar(128) NOT NULL,
    write_scope varchar(32) NOT NULL,
    previous_state varchar(16),
    new_state varchar(16) NOT NULL,
    reason varchar(256) NOT NULL,
    observed_ratio double precision,
    policy_version bigint NOT NULL,
    operator_id varchar(64) NOT NULL DEFAULT 'system',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS storage_protection_events_target_time_idx
    ON public.storage_protection_events (target_type, target_id, write_scope, created_at DESC);
-- Consolidated pre-release geofence Observe schema.

-- +omcgo MainReconcileBegin
ALTER TABLE public.devices
    ADD COLUMN IF NOT EXISTS location_source_mode varchar(16)
        NOT NULL DEFAULT 'tr069';

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_catalog.pg_constraint
        WHERE conrelid = 'public.devices'::regclass
          AND conname = 'devices_location_source_mode_check'
    ) THEN
        ALTER TABLE public.devices
            ADD CONSTRAINT devices_location_source_mode_check
            CHECK (location_source_mode IN ('tr069', 'external'));
    END IF;
END
$$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS public.geofence_carrier_settings (
    carrier varchar(16) PRIMARY KEY,
    mode varchar(16) NOT NULL DEFAULT 'off',
    default_baseline_radius_meters double precision NOT NULL DEFAULT 100,
    updated_by uuid,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT geofence_carrier_settings_mode_check
        CHECK (mode IN ('off', 'observe', 'enforce')),
    CONSTRAINT geofence_carrier_settings_radius_check
        CHECK (default_baseline_radius_meters > 0
            AND default_baseline_radius_meters <= 50000)
);

CREATE TABLE IF NOT EXISTS public.geofence_definitions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name varchar(128) NOT NULL,
    carrier varchar(16) NOT NULL,
    rule_type varchar(32) NOT NULL,
    owner_device_id uuid,
    status varchar(16) NOT NULL DEFAULT 'draft',
    current_version_id uuid,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT geofence_definitions_rule_type_check
        CHECK (rule_type IN ('polygon_allow_zone', 'baseline_radius')),
    CONSTRAINT geofence_definitions_status_check
        CHECK (status IN ('draft', 'enabled', 'disabled', 'archived')),
    CONSTRAINT geofence_definitions_owner_check CHECK (
        (rule_type = 'baseline_radius' AND owner_device_id IS NOT NULL)
        OR
        (rule_type = 'polygon_allow_zone' AND owner_device_id IS NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_geofence_definitions_carrier_name
    ON public.geofence_definitions (carrier, lower(name))
    WHERE status <> 'archived';

CREATE UNIQUE INDEX IF NOT EXISTS uq_geofence_definitions_baseline_owner
    ON public.geofence_definitions (owner_device_id)
    WHERE rule_type = 'baseline_radius' AND status <> 'archived';

CREATE TABLE IF NOT EXISTS public.geofence_versions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    geofence_id uuid NOT NULL
        REFERENCES public.geofence_definitions(id) ON DELETE RESTRICT,
    version bigint NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'draft',
    geometry_json jsonb NOT NULL,
    bbox_min_longitude double precision NOT NULL,
    bbox_min_latitude double precision NOT NULL,
    bbox_max_longitude double precision NOT NULL,
    bbox_max_latitude double precision NOT NULL,
    policy_json jsonb NOT NULL,
    created_by uuid NOT NULL,
    published_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,
    CONSTRAINT geofence_versions_version_check CHECK (version > 0),
    CONSTRAINT geofence_versions_status_check
        CHECK (status IN ('draft', 'published', 'superseded')),
    CONSTRAINT geofence_versions_bbox_check CHECK (
        bbox_min_longitude >= -180 AND bbox_max_longitude <= 180
        AND bbox_min_latitude >= -90 AND bbox_max_latitude <= 90
        AND bbox_min_longitude <= bbox_max_longitude
        AND bbox_min_latitude <= bbox_max_latitude
    ),
    CONSTRAINT uq_geofence_versions_number UNIQUE (geofence_id, version)
);

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_catalog.pg_constraint
        WHERE conrelid = 'public.geofence_definitions'::regclass
          AND conname = 'geofence_definitions_current_version_fk'
    ) THEN
        ALTER TABLE public.geofence_definitions
            ADD CONSTRAINT geofence_definitions_current_version_fk
            FOREIGN KEY (current_version_id)
            REFERENCES public.geofence_versions(id) ON DELETE RESTRICT;
    END IF;
END
$$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS public.device_geofence_bindings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id uuid NOT NULL,
    geofence_id uuid NOT NULL
        REFERENCES public.geofence_definitions(id) ON DELETE RESTRICT,
    rule_type varchar(32) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'pending',
    bind_source varchar(16) NOT NULL,
    bound_by uuid NOT NULL,
    bound_at timestamptz NOT NULL DEFAULT now(),
    removed_by uuid,
    removed_at timestamptz,
    remove_reason text,
    CONSTRAINT device_geofence_bindings_rule_type_check
        CHECK (rule_type IN ('polygon_allow_zone', 'baseline_radius')),
    CONSTRAINT device_geofence_bindings_status_check
        CHECK (status IN ('pending', 'active', 'suspended', 'removed')),
    CONSTRAINT device_geofence_bindings_source_check
        CHECK (bind_source IN ('manual', 'auto'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_device_geofence_active_rule_type
    ON public.device_geofence_bindings (device_id, rule_type)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_device_geofence_bindings_geofence
    ON public.device_geofence_bindings (geofence_id, status);

CREATE TABLE IF NOT EXISTS public.device_geofence_states (
    binding_id uuid PRIMARY KEY
        REFERENCES public.device_geofence_bindings(id) ON DELETE RESTRICT,
    device_id uuid NOT NULL,
    confirmed_state varchar(16) NOT NULL DEFAULT 'unknown',
    candidate_state varchar(16),
    candidate_count integer NOT NULL DEFAULT 0,
    candidate_since timestamptz,
    state_version bigint NOT NULL DEFAULT 1,
    last_geofence_version_id uuid
        REFERENCES public.geofence_versions(id) ON DELETE RESTRICT,
    last_observation_version bigint,
    last_observed_at timestamptz,
    last_distance_to_boundary double precision,
    last_evaluation_id uuid,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT device_geofence_states_confirmed_check
        CHECK (confirmed_state IN ('unknown', 'inside', 'outside')),
    CONSTRAINT device_geofence_states_candidate_check
        CHECK (candidate_state IS NULL OR candidate_state IN ('exit', 'reentry')),
    CONSTRAINT device_geofence_states_count_check CHECK (candidate_count >= 0),
    CONSTRAINT device_geofence_states_version_check CHECK (state_version > 0)
);

CREATE INDEX IF NOT EXISTS idx_device_geofence_states_device
    ON public.device_geofence_states (device_id);

CREATE TABLE IF NOT EXISTS public.device_geofence_effective_states (
    device_id uuid PRIMARY KEY,
    effective_state varchar(16) NOT NULL DEFAULT 'unmanaged',
    required_action_level varchar(16) NOT NULL DEFAULT 'none',
    state_version bigint NOT NULL DEFAULT 1,
    trigger_binding_id uuid
        REFERENCES public.device_geofence_bindings(id) ON DELETE SET NULL,
    last_observation_version bigint,
    evaluation_health varchar(16) NOT NULL DEFAULT 'healthy',
    last_evaluation_error_code varchar(64),
    last_successful_evaluation_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT device_geofence_effective_state_check
        CHECK (effective_state IN ('unmanaged', 'unknown', 'inside', 'outside')),
    CONSTRAINT device_geofence_action_level_check
        CHECK (required_action_level IN ('none', 'notify_only', 'manual_review', 'deactivate')),
    CONSTRAINT device_geofence_health_check
        CHECK (evaluation_health IN ('healthy', 'stale', 'failed')),
    CONSTRAINT device_geofence_effective_version_check CHECK (state_version > 0)
);

CREATE TABLE IF NOT EXISTS public.geofence_control_actions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    action_key varchar(255) NOT NULL UNIQUE,
    parent_action_id uuid
        REFERENCES public.geofence_control_actions(id) ON DELETE RESTRICT,
    device_id uuid NOT NULL,
    device_sn varchar(64) NOT NULL,
    geofence_id uuid
        REFERENCES public.geofence_definitions(id) ON DELETE RESTRICT,
    binding_id uuid
        REFERENCES public.device_geofence_bindings(id) ON DELETE SET NULL,
    effective_state_version bigint NOT NULL,
    action_type varchar(16) NOT NULL,
    status varchar(32) NOT NULL DEFAULT 'pending',
    before_state jsonb NOT NULL DEFAULT '[]'::jsonb,
    requested_state jsonb NOT NULL DEFAULT '[]'::jsonb,
    verified_state jsonb NOT NULL DEFAULT '[]'::jsonb,
    last_error text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    CONSTRAINT geofence_control_actions_state_version_check
        CHECK (effective_state_version >= 0),
    CONSTRAINT geofence_control_actions_type_check
        CHECK (action_type IN ('deactivate', 'activate')),
    CONSTRAINT geofence_control_actions_status_check CHECK (status IN (
        'pending', 'executing', 'verifying', 'verified', 'partial_failed',
        'failed'
    )),
    CONSTRAINT geofence_control_actions_before_state_array_check
        CHECK (jsonb_typeof(before_state) = 'array'),
    CONSTRAINT geofence_control_actions_requested_state_array_check
        CHECK (jsonb_typeof(requested_state) = 'array'),
    CONSTRAINT geofence_control_actions_verified_state_array_check
        CHECK (jsonb_typeof(verified_state) = 'array')
);

CREATE INDEX IF NOT EXISTS idx_geofence_control_actions_device_time
    ON public.geofence_control_actions (device_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_geofence_control_actions_status
    ON public.geofence_control_actions (status, updated_at)
    WHERE status IN ('pending', 'executing', 'verifying');

ALTER TABLE public.device_location_observations
    ADD COLUMN IF NOT EXISTS received_at timestamptz,
    ADD COLUMN IF NOT EXISTS device_reported_at timestamptz,
    ADD COLUMN IF NOT EXISTS gps_lock_status varchar(32),
    ADD COLUMN IF NOT EXISTS satellite_count integer,
    ADD COLUMN IF NOT EXISTS accuracy_meters double precision;

UPDATE public.device_location_observations
SET received_at = observed_at
WHERE received_at IS NULL;

ALTER TABLE public.device_location_observations
    ALTER COLUMN received_at SET DEFAULT now(),
    ALTER COLUMN received_at SET NOT NULL;

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_catalog.pg_constraint
        WHERE conrelid = 'public.device_location_observations'::regclass
          AND conname = 'device_location_observations_satellite_count_check'
    ) THEN
        ALTER TABLE public.device_location_observations
            ADD CONSTRAINT device_location_observations_satellite_count_check
            CHECK (satellite_count IS NULL OR satellite_count >= 0);
    END IF;
    IF NOT EXISTS (
        SELECT 1
        FROM pg_catalog.pg_constraint
        WHERE conrelid = 'public.device_location_observations'::regclass
          AND conname = 'device_location_observations_accuracy_check'
    ) THEN
        ALTER TABLE public.device_location_observations
            ADD CONSTRAINT device_location_observations_accuracy_check
            CHECK (accuracy_meters IS NULL OR accuracy_meters >= 0);
    END IF;
END
$$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS public.event_outbox (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type varchar(64) NOT NULL,
    aggregate_id varchar(160) NOT NULL,
    subject varchar(160) NOT NULL,
    payload jsonb NOT NULL,
    dedupe_key varchar(255) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'pending',
    attempts integer NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    claim_token uuid,
    claim_expires_at timestamptz,
    last_error text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,
    CONSTRAINT event_outbox_dedupe_key_unique UNIQUE (dedupe_key),
    CONSTRAINT event_outbox_status_check
        CHECK (status IN ('pending', 'publishing', 'published', 'failed', 'dead')),
    CONSTRAINT event_outbox_attempts_check CHECK (attempts >= 0),
    CONSTRAINT event_outbox_claim_check CHECK (
        (status = 'publishing' AND claim_token IS NOT NULL AND claim_expires_at IS NOT NULL)
        OR
        (status <> 'publishing' AND claim_token IS NULL AND claim_expires_at IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_event_outbox_claimable
    ON public.event_outbox (status, next_attempt_at, claim_expires_at, created_at)
    WHERE status IN ('pending', 'failed', 'publishing');

CREATE TABLE IF NOT EXISTS public.geofence_evaluations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    binding_id uuid NOT NULL
        REFERENCES public.device_geofence_bindings(id) ON DELETE RESTRICT,
    device_id uuid NOT NULL,
    geofence_id uuid NOT NULL
        REFERENCES public.geofence_definitions(id) ON DELETE RESTRICT,
    geofence_version_id uuid NOT NULL
        REFERENCES public.geofence_versions(id) ON DELETE RESTRICT,
    observation_version bigint NOT NULL,
    latitude double precision NOT NULL,
    longitude double precision NOT NULL,
    gps_height double precision,
    observed_at timestamptz NOT NULL,
    received_at timestamptz NOT NULL,
    device_reported_at timestamptz,
    gps_lock_status varchar(32),
    satellite_count integer,
    accuracy_meters double precision,
    source_path text NOT NULL,
    previous_observation_version bigint,
    movement_distance_meters double precision,
    elapsed_seconds double precision,
    implied_speed_mps double precision,
    rule_type varchar(32) NOT NULL,
    raw_position varchar(16),
    signed_distance_meters double precision,
    previous_confirmed_state varchar(16) NOT NULL,
    confirmed_state varchar(16) NOT NULL,
    candidate_state varchar(16),
    candidate_count integer NOT NULL DEFAULT 0,
    candidate_since timestamptz,
    state_edge boolean NOT NULL DEFAULT false,
    status varchar(16) NOT NULL,
    reason_code varchar(64) NOT NULL,
    failure_stage varchar(64),
    error_code varchar(64),
    error_summary text,
    evaluated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT geofence_evaluations_observation_version_check
        CHECK (observation_version > 0),
    CONSTRAINT geofence_evaluations_rule_type_check
        CHECK (rule_type IN ('polygon_allow_zone', 'baseline_radius')),
    CONSTRAINT geofence_evaluations_raw_position_check
        CHECK (raw_position IS NULL OR raw_position IN ('unknown', 'inside', 'outside', 'boundary')),
    CONSTRAINT geofence_evaluations_previous_state_check
        CHECK (previous_confirmed_state IN ('unknown', 'inside', 'outside')),
    CONSTRAINT geofence_evaluations_confirmed_state_check
        CHECK (confirmed_state IN ('unknown', 'inside', 'outside')),
    CONSTRAINT geofence_evaluations_candidate_state_check
        CHECK (candidate_state IS NULL OR candidate_state IN ('exit', 'reentry')),
    CONSTRAINT geofence_evaluations_candidate_count_check
        CHECK (candidate_count >= 0),
    CONSTRAINT geofence_evaluations_status_check
        CHECK (status IN ('completed', 'failed')),
    CONSTRAINT geofence_evaluations_failure_evidence_check CHECK (
        (status = 'completed'
            AND failure_stage IS NULL
            AND error_code IS NULL
            AND error_summary IS NULL)
        OR
        (status = 'failed'
            AND failure_stage IS NOT NULL
            AND error_code IS NOT NULL)
    ),
    CONSTRAINT uq_geofence_evaluations_identity
        UNIQUE (binding_id, geofence_version_id, observation_version)
);

CREATE INDEX IF NOT EXISTS idx_geofence_evaluations_device_time
    ON public.geofence_evaluations (device_id, evaluated_at DESC);

CREATE INDEX IF NOT EXISTS idx_geofence_evaluations_status_time
    ON public.geofence_evaluations (status, evaluated_at DESC);

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_catalog.pg_constraint
        WHERE conrelid = 'public.device_geofence_states'::regclass
          AND conname = 'device_geofence_states_last_evaluation_fk'
    ) THEN
        ALTER TABLE public.device_geofence_states
            ADD CONSTRAINT device_geofence_states_last_evaluation_fk
            FOREIGN KEY (last_evaluation_id)
            REFERENCES public.geofence_evaluations(id) ON DELETE RESTRICT;
    END IF;
END
$$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS public.geofence_batch_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id uuid NOT NULL
        REFERENCES public.async_jobs(id) ON DELETE RESTRICT,
    geofence_id uuid NOT NULL
        REFERENCES public.geofence_definitions(id) ON DELETE RESTRICT,
    input_key varchar(320) NOT NULL,
    input_kind varchar(16) NOT NULL,
    input_value varchar(255) NOT NULL,
    device_id uuid,
    device_sn_snapshot varchar(255),
    status varchar(16) NOT NULL DEFAULT 'pending',
    reason_code varchar(64),
    error_message text,
    expected_source_binding_id uuid
        REFERENCES public.device_geofence_bindings(id) ON DELETE RESTRICT,
    binding_id uuid
        REFERENCES public.device_geofence_bindings(id) ON DELETE RESTRICT,
    attempt integer NOT NULL DEFAULT 0,
    started_at timestamptz,
    finished_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT geofence_batch_items_input_kind_check
        CHECK (input_kind IN ('device_id', 'device_sn')),
    CONSTRAINT geofence_batch_items_status_check
        CHECK (status IN ('pending', 'succeeded', 'skipped', 'failed')),
    CONSTRAINT geofence_batch_items_attempt_check CHECK (attempt >= 0),
    CONSTRAINT geofence_batch_items_success_binding_check
        CHECK (status <> 'succeeded' OR binding_id IS NOT NULL),
    CONSTRAINT uq_geofence_batch_items_job_input UNIQUE (job_id, input_key)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_geofence_batch_items_job_device
    ON public.geofence_batch_items (job_id, device_id)
    WHERE device_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_geofence_batch_items_job_status
    ON public.geofence_batch_items (job_id, status, created_at, id);

CREATE INDEX IF NOT EXISTS idx_geofence_batch_items_geofence_status
    ON public.geofence_batch_items (geofence_id, status);

CREATE TABLE IF NOT EXISTS public.third_party_location_batches (
    idempotency_key varchar(255) PRIMARY KEY,
    request_hash char(64) NOT NULL,
    status varchar(16) NOT NULL,
    result jsonb,
    completed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT third_party_location_batches_status_check
        CHECK (status IN ('processing', 'completed')),
    CONSTRAINT third_party_location_batches_completed_result_check
        CHECK (status <> 'completed' OR result IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_third_party_location_batches_created_at
    ON public.third_party_location_batches (created_at);

CREATE UNIQUE INDEX IF NOT EXISTS uq_async_jobs_geofence_manual_bind_request
    ON public.async_jobs (
        job_type,
        (payload->>'geofence_id'),
        (payload->>'requested_by'),
        (payload->>'preview_fingerprint')
    )
    WHERE job_type = 'geofence_manual_bind'
      AND status IN ('pending', 'running', 'succeeded');

-- +goose StatementBegin
DO $$
DECLARE
    required_relation text;
BEGIN
    FOREACH required_relation IN ARRAY ARRAY[
        'public.geofence_carrier_settings',
        'public.geofence_definitions',
        'public.geofence_versions',
        'public.device_geofence_bindings',
        'public.device_geofence_states',
        'public.device_geofence_effective_states',
        'public.geofence_control_actions',
        'public.event_outbox',
        'public.geofence_evaluations',
        'public.geofence_batch_items',
        'public.third_party_location_batches'
    ] LOOP
        IF to_regclass(required_relation) IS NULL THEN
            RAISE EXCEPTION 'main baseline reconcile missing relation %', required_relation;
        END IF;
    END LOOP;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'devices'
          AND column_name = 'location_source_mode'
    ) THEN
        RAISE EXCEPTION 'main baseline reconcile missing devices.location_source_mode';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'provisioning_tasks'
          AND column_name = 'policy_id'
    ) OR NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'provisioning_tasks'
          AND column_name = 'current_step_name'
    ) THEN
        RAISE EXCEPTION 'main baseline reconcile missing provisioning task columns';
    END IF;
END
$$;
-- +goose StatementEnd
-- +omcgo MainReconcileEnd


-- +goose Down
-- +goose StatementBegin
CREATE SCHEMA goose_baseline_meta;
ALTER TABLE public.goose_db_version SET SCHEMA goose_baseline_meta;
DROP SCHEMA IF EXISTS public CASCADE;
CREATE SCHEMA public;
ALTER TABLE goose_baseline_meta.goose_db_version SET SCHEMA public;
DROP SCHEMA goose_baseline_meta;
-- +goose StatementEnd
