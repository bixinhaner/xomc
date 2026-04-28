-- +goose Up
-- 迁移 000042: top10 慢查询补索引（W3.F.2 / T-0060）
-- 来源：基于代码 grep（squirrel.Where / OrderBy）+ EXPLAIN 推断 + repository 层主查询路径。
-- 详细分析见 docs/perf/slow-queries-top10.md。
-- 本迁移仅添加索引，不修改既有 schema/数据，幂等。
--
-- 设计原则：
--   1. 复合列顺序按选择性从高到低排列（device_sn/status 高选择性 → 在前）。
--   2. 覆盖列含 raised_at/created_at DESC 时显式 DESC，避免反向扫描。
--   3. 半部分索引（partial index）用 WHERE 过滤无效行，缩小索引体积。
--   4. 已有完全等价索引的不重复创建（IF NOT EXISTS 保证幂等）。

-- ============================================================
-- 1. devices: (carrier, deleted_at) 过滤回收站 / 在册设备
-- ============================================================
-- 命中查询：列表接口 deleted_at IS NULL；回收站 deleted_at IS NOT NULL；按 carrier 维度统计。
CREATE INDEX IF NOT EXISTS idx_devices_carrier_alive
    ON devices (carrier, last_inform_at DESC)
    WHERE deleted_at IS NULL;

-- ============================================================
-- 2. alarms_active: (status, severity, raised_at DESC) 列表主排序
-- ============================================================
-- 命中查询：ListActive ORDER BY raised_at DESC，filter 含 status / severity；
-- 现有 idx_alarms_active_severity / status 单列索引选择性差。
CREATE INDEX IF NOT EXISTS idx_alarms_active_status_severity_time
    ON alarms_active (status, severity, raised_at DESC);

-- ============================================================
-- 3. alarms_active: (carrier, severity) 用于 dashboard 聚合统计
-- ============================================================
-- 命中查询：dashboard 大屏按运营商分桶统计活跃告警的严重程度分布。
CREATE INDEX IF NOT EXISTS idx_alarms_active_carrier_severity
    ON alarms_active (carrier, severity);

-- ============================================================
-- 4. alarms_history: (severity, time DESC) 历史告警按严重程度趋势
-- ============================================================
-- 命中查询：报表"近 N 天严重告警次数"——现有 (device_id, time) / (alarm_id, time) 不覆盖 severity 过滤。
CREATE INDEX IF NOT EXISTS idx_alarms_history_severity_time
    ON alarms_history (severity, time DESC);

-- ============================================================
-- 5. device_tasks: (status, expires_at) 过期任务清理 / 重试调度
-- ============================================================
-- 命中查询：reboot closer / expire reaper：SELECT WHERE status IN (..) AND expires_at < now() ORDER BY expires_at；
-- 现有 idx_device_tasks_status 单列索引退化为全表过滤 expires_at。
CREATE INDEX IF NOT EXISTS idx_device_tasks_status_expires
    ON device_tasks (status, expires_at)
    WHERE expires_at IS NOT NULL;

-- ============================================================
-- 6. notifications: (type, priority, created_at DESC) 通知中心列表
-- ============================================================
-- 命中查询：通知中心按 type+priority 过滤后按时间倒序——现有 (type, created_at) 不含 priority。
CREATE INDEX IF NOT EXISTS idx_notifications_type_priority_time
    ON notifications (type, priority, created_at DESC);

-- ============================================================
-- 7. audit_logs: (action, created_at DESC) 操作类型审计追溯
-- ============================================================
-- 命中查询：安全审计"最近 24h 所有 login / config_update 操作"——现有 (user_id, ts) / (resource, resource_id, ts) 都不能命中。
CREATE INDEX IF NOT EXISTS idx_audit_logs_action_time
    ON audit_logs (action, created_at DESC);

-- ============================================================
-- 8. mml_tasks: (creator, created_at DESC) "我的任务"列表
-- ============================================================
-- 命中查询：MML 控制台"我创建的任务"按 creator 过滤后按时间倒序——
-- 现有 idx_mml_tasks_status / created 都不含 creator 维度。
CREATE INDEX IF NOT EXISTS idx_mml_tasks_creator_time
    ON mml_tasks (creator, created_at DESC)
    WHERE creator IS NOT NULL;

-- ============================================================
-- 9. managed_files: (file_type, status, created_at DESC) 文件管理列表
-- ============================================================
-- 命中查询：文件管理界面按 type+status 过滤分页——现有 type / status 单列索引选择性低。
CREATE INDEX IF NOT EXISTS idx_managed_files_type_status_time
    ON managed_files (file_type, status, created_at DESC);

-- ============================================================
-- 10. system_logs: (level, source, created_at DESC) 运维日志检索
-- ============================================================
-- 命中查询：运维查询"近期 ERROR 级别 + 模块 X 的日志"——现有三个单列索引无法走复合扫描。
CREATE INDEX IF NOT EXISTS idx_system_logs_level_source_time
    ON system_logs (level, source, created_at DESC);

-- ============================================================
-- 11. alarms_active: (device_id, raised_at DESC) 设备维度倒序
-- ============================================================
-- 命中查询：设备详情页"近 N 条活跃告警"——现有 idx_alarms_active_device 不含 raised_at 排序键。
CREATE INDEX IF NOT EXISTS idx_alarms_active_device_time
    ON alarms_active (device_id, raised_at DESC);

-- +goose Down
-- 回滚：删除上面新增的所有索引（顺序与创建相反，便于阅读）
DROP INDEX IF EXISTS idx_alarms_active_device_time;
DROP INDEX IF EXISTS idx_system_logs_level_source_time;
DROP INDEX IF EXISTS idx_managed_files_type_status_time;
DROP INDEX IF EXISTS idx_mml_tasks_creator_time;
DROP INDEX IF EXISTS idx_audit_logs_action_time;
DROP INDEX IF EXISTS idx_notifications_type_priority_time;
DROP INDEX IF EXISTS idx_device_tasks_status_expires;
DROP INDEX IF EXISTS idx_alarms_history_severity_time;
DROP INDEX IF EXISTS idx_alarms_active_carrier_severity;
DROP INDEX IF EXISTS idx_alarms_active_status_severity_time;
DROP INDEX IF EXISTS idx_devices_carrier_alive;
