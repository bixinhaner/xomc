-- 历史菜单 name_i18n 中英文回填（migration 000083 新增字段）
--
-- 策略：
--   1. 通用按钮（查询/添加/编辑/删除/...）按 name 批量回填（同名按钮在系统里出现几十次）
--   2. directory 与 menu 类型节点按 id 精准回填（命名独立、无重复）
--   3. 仅回填 name_i18n IS NULL 的行，幂等可重跑
--
-- 英文译文由开发顺手译，后续可由产品在菜单管理 UI 内更新（PUT /admin/menus/:id）。
-- i18n_key 留空：方案 C 主路径是 DB JSONB，i18n_key 仅供未来灰度切换。

-- +goose Up

-- ============================================================
-- 1. 通用按钮（按 name 批量；仅作用于 type='button'）
-- ============================================================
UPDATE menus
SET name_i18n = CASE name
    WHEN '查询'        THEN '{"zh-CN":"查询","en-US":"Query"}'::jsonb
    WHEN '添加'        THEN '{"zh-CN":"添加","en-US":"Add"}'::jsonb
    WHEN '新增'        THEN '{"zh-CN":"新增","en-US":"Create"}'::jsonb
    WHEN '新建'        THEN '{"zh-CN":"新建","en-US":"New"}'::jsonb
    WHEN '修改'        THEN '{"zh-CN":"修改","en-US":"Edit"}'::jsonb
    WHEN '编辑'        THEN '{"zh-CN":"编辑","en-US":"Edit"}'::jsonb
    WHEN '删除'        THEN '{"zh-CN":"删除","en-US":"Delete"}'::jsonb
    WHEN '导出'        THEN '{"zh-CN":"导出","en-US":"Export"}'::jsonb
    WHEN '批量操作'    THEN '{"zh-CN":"批量操作","en-US":"Batch Operation"}'::jsonb
    WHEN '重置密码'    THEN '{"zh-CN":"重置密码","en-US":"Reset Password"}'::jsonb
    WHEN '取消'        THEN '{"zh-CN":"取消","en-US":"Cancel"}'::jsonb
    WHEN '暂停/恢复'   THEN '{"zh-CN":"暂停/恢复","en-US":"Pause / Resume"}'::jsonb
    WHEN '执行'        THEN '{"zh-CN":"执行","en-US":"Execute"}'::jsonb
    WHEN '发起'        THEN '{"zh-CN":"发起","en-US":"Run"}'::jsonb
    WHEN '触发'        THEN '{"zh-CN":"触发","en-US":"Trigger"}'::jsonb
    WHEN '审批'        THEN '{"zh-CN":"审批","en-US":"Approve"}'::jsonb
    WHEN '审计查看'    THEN '{"zh-CN":"审计查看","en-US":"Audit View"}'::jsonb
    WHEN '紧急权限'    THEN '{"zh-CN":"紧急权限","en-US":"Break Glass"}'::jsonb
    WHEN '安全命令'    THEN '{"zh-CN":"安全命令","en-US":"Safe Command"}'::jsonb
    WHEN '谨慎命令'    THEN '{"zh-CN":"谨慎命令","en-US":"Cautious Command"}'::jsonb
    WHEN '危险命令'    THEN '{"zh-CN":"危险命令","en-US":"Dangerous Command"}'::jsonb
    WHEN '北向服务器编辑' THEN '{"zh-CN":"北向服务器编辑","en-US":"Edit Northbound Server"}'::jsonb
    WHEN '许可证操作'  THEN '{"zh-CN":"许可证操作","en-US":"License Operations"}'::jsonb
    ELSE name_i18n
END
WHERE type = 'button' AND name_i18n IS NULL;

-- ============================================================
-- 2. directory / menu 节点（按 id 精准回填）
-- ============================================================
UPDATE menus m
SET name_i18n = i.name_i18n::jsonb
FROM (VALUES
    -- 设备管理（migrations/000009）
    ('11111111-1111-1111-1111-111111111101', '{"zh-CN":"设备管理","en-US":"Devices"}'),
    ('11111111-1111-1111-1111-111111111102', '{"zh-CN":"设备列表","en-US":"Device List"}'),
    ('11111111-1111-1111-1111-111111111103', '{"zh-CN":"设备分组","en-US":"Device Groups"}'),
    ('11111111-1111-1111-1111-111111111104', '{"zh-CN":"设备注册","en-US":"Device Registration"}'),

    -- 告警管理
    ('11111111-1111-1111-1111-111111111105', '{"zh-CN":"告警管理","en-US":"Alarms"}'),
    ('11111111-1111-1111-1111-111111111106', '{"zh-CN":"当前告警","en-US":"Active Alarms"}'),
    ('11111111-1111-1111-1111-111111111107', '{"zh-CN":"历史告警","en-US":"Historical Alarms"}'),

    -- 系统管理
    ('11111111-1111-1111-1111-111111111108', '{"zh-CN":"系统管理","en-US":"System"}'),
    ('11111111-1111-1111-1111-111111111109', '{"zh-CN":"用户管理","en-US":"Users"}'),
    ('11111111-1111-1111-1111-111111111110', '{"zh-CN":"角色管理","en-US":"Roles"}'),
    ('11111111-1111-1111-1111-111111111111', '{"zh-CN":"菜单管理","en-US":"Menus"}'),
    ('11111111-1111-1111-1111-111111111112', '{"zh-CN":"操作日志","en-US":"Operation Logs"}'),

    -- 许可证管理（seed/000078）
    ('aaaa0009-0000-0000-0000-000000000001', '{"zh-CN":"许可证管理","en-US":"License"}'),
    ('aaaa0009-1000-0000-0000-000000000001', '{"zh-CN":"许可证列表","en-US":"License List"}'),
    ('aaaa0009-1000-0000-0000-000000000002', '{"zh-CN":"许可证操作","en-US":"License Operations"}'),
    ('aaaa0009-1000-0000-0000-000000000003', '{"zh-CN":"审计日志","en-US":"Audit Logs"}'),

    -- 运维管理（seed/000079）
    ('aaaa000a-0000-0000-0000-000000000001', '{"zh-CN":"运维管理","en-US":"Operations"}'),
    ('aaaa000a-1000-0000-0000-000000000001', '{"zh-CN":"运维模板","en-US":"Templates"}'),
    ('aaaa000a-1000-0000-0000-000000000002', '{"zh-CN":"运维任务","en-US":"Tasks"}'),
    ('aaaa000a-1000-0000-0000-000000000003', '{"zh-CN":"运维命令","en-US":"Commands"}'),
    ('aaaa000a-1000-0000-0000-000000000004', '{"zh-CN":"网络诊断","en-US":"Network Diagnostics"}'),
    ('aaaa000a-1000-0000-0000-000000000005', '{"zh-CN":"运维下载","en-US":"Downloads"}')
) AS i(id, name_i18n)
WHERE m.id = i.id::uuid AND m.name_i18n IS NULL;

-- ============================================================
-- 3. 兜底回填：剩余仍为 NULL 的菜单，至少把 name 灌进 zh-CN，
--    避免前端在 locale='zh-CN' 时拿到 NULL 后走到 name fallback 链路最末端。
--    en-US 暂缺，由前端兜底回退到 zh-CN（参 mapBackendMenu 渲染优先级）。
-- ============================================================
UPDATE menus
SET name_i18n = jsonb_build_object('zh-CN', name)
WHERE name_i18n IS NULL AND name IS NOT NULL AND name <> '';

-- +goose Down
-- 回滚仅清空 i18n 数据；DDL 字段由 000083 的 Down 处理。
UPDATE menus SET name_i18n = NULL;
