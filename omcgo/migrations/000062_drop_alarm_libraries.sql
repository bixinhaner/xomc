-- +goose Up
-- ============================================================
-- 000062_drop_alarm_libraries.sql
-- T-0098-P5-06：DROP 旧 alarm_libraries / alarm_library_i18n
--
-- 决议依据：
--   - 设计文档 `docs/design/参数-KPI-告警-整合设计方案.md` D2=B
--   - 新告警字典走 alarm_definitions（migrations/000059）；XML 重载即恢复种子
--   - 历史告警表 alarms_active / alarms_history 不依赖 alarm_libraries（无 FK）
--
-- DROP 范围：
--   - alarm_library_i18n（FK → alarm_libraries.id ON DELETE CASCADE，先 DROP）
--   - alarm_libraries
--
-- 影响面：
--   - cmd/app/provider/alarm.go：移除 alarmLibraryRepo / alarmLibraryService 接线
--   - cmd/app/provider/router.go：移除 /alarms/alarm-libraries 路由组
--   - internal/alarm/library_*.go（5 文件）：全删
--   - migrations/seed/000028_alarm_library_import.sql：失活（保留版本号占位）
--   - frontend-core/src/services/api/alarmApi.ts：移除 4 个 AlarmLibrary 端点
--   - webcode/src/pages/alarm/AlarmSupportLibrary：删除整页 + 路由
-- ============================================================

DROP TABLE IF EXISTS alarm_library_i18n CASCADE;
DROP TABLE IF EXISTS alarm_libraries CASCADE;

-- +goose Down
-- ============================================================
-- 回滚：重建空表（仅恢复 schema 结构，不恢复数据）。
-- 数据恢复路径：重启 omcgo-app，dictloader 自动从 alarm-definitions/ XML 重载到 alarm_definitions
-- 表（不再写 alarm_libraries）。如确需 alarm_libraries 旧形态，需手动从 git history 找回
-- seed/000028_alarm_library_import.sql 内容并执行。
-- ============================================================

CREATE TABLE alarm_libraries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alarm_identifier VARCHAR(128) NOT NULL UNIQUE,
    alarm_source    VARCHAR(64) NOT NULL,
    event_type      VARCHAR(64) NOT NULL,
    severity        SMALLINT NOT NULL,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    probable_cause  TEXT NOT NULL,
    explanation     TEXT,
    additional_info JSONB DEFAULT '{}',
    carrier         VARCHAR(4),
    technology      VARCHAR(16),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alarm_libraries_alarm_source ON alarm_libraries(alarm_source);
CREATE INDEX IF NOT EXISTS idx_alarm_libraries_alarm_identifier ON alarm_libraries(alarm_identifier);
CREATE INDEX IF NOT EXISTS idx_alarm_libraries_severity ON alarm_libraries(severity);
CREATE INDEX IF NOT EXISTS idx_alarm_libraries_carrier ON alarm_libraries(carrier) WHERE carrier IS NOT NULL;

CREATE TABLE alarm_library_i18n (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    library_id      UUID NOT NULL REFERENCES alarm_libraries(id) ON DELETE CASCADE,
    locale          VARCHAR(16) NOT NULL,
    probable_cause  TEXT NOT NULL,
    explanation     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(library_id, locale)
);

CREATE INDEX IF NOT EXISTS idx_alarm_library_i18n_library_id ON alarm_library_i18n(library_id);
CREATE INDEX IF NOT EXISTS idx_alarm_library_i18n_locale ON alarm_library_i18n(locale);
