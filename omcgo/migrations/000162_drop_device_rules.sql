-- 000162_drop_device_rules.sql
-- 历史：原 000161，与并行合入的 000161_config_snapshots.sql 撞号
-- （commits c0123994 ↔ 3af260e3），按 CLAUDE.md §5.5 "后合并的重命名为
-- 更大版本号" 顺延为 162。
--
-- 彻底下线设备规则模块（device_rules）。
--
-- 历史：
--   - 设备规则引擎（DeviceRuleService + cron 调度 + device.registered 订阅）已下线，
--     自动归组改由 GroupMatchEngine（消费 L2 分组自带的匹配规则）接管。
--   - 上一阶段仅注释 modules.go 的 Start() 调用，table/handler/前端页面/菜单仍保留。
--   - 本迁移完成彻底清理：删 table、列、菜单、按钮、role_menus、api_endpoints。
--
-- 影响：
--   - 表 device_rules / device_rule_tasks：DROP
--   - 列 device_groups.bound_rule_id：DROP（FK 指向 device_rules）
--   - 列 device_group_members.source_rule_id：DROP（无 FK 约束，仅业务关联）
--   - 索引 idx_device_group_members_source：DROP（含 source_rule_id 列）
--   - 菜单"设备规则"(aaaa0010-1000-0000-0000-000000000002)及子按钮：DELETE
--     menus_parent_id_fkey ON DELETE CASCADE 自动清子按钮
--     role_menus_menu_id_fkey ON DELETE CASCADE 自动清角色绑定
--   - api_endpoints 中 /api/v1/device-rules* 的 11 条记录：DELETE
--
-- 回滚：留空（删表数据无法回滚；如需恢复请 restore 备份）。

-- +goose Up

-- 1. device_groups: 删除 bound_rule_id 列（含 FK 约束）
-- DROP COLUMN 会级联删除该列上的所有 FK / 索引，无需先 DROP CONSTRAINT
ALTER TABLE device_groups DROP COLUMN IF EXISTS bound_rule_id;

-- 2. device_group_members: 先删依赖列的复合索引，再删列
DROP INDEX IF EXISTS idx_device_group_members_source;
ALTER TABLE device_group_members DROP COLUMN IF EXISTS source_rule_id;

-- 3. 删表（device_rule_tasks 在前，避免 FK 错误；CASCADE 兜底）
DROP TABLE IF EXISTS device_rule_tasks CASCADE;
DROP TABLE IF EXISTS device_rules CASCADE;

-- 4. 删菜单（CASCADE 同时清掉子按钮 + role_menus 绑定）
DELETE FROM menus WHERE id = 'aaaa0010-1000-0000-0000-000000000002';

-- 5. 删 api_endpoints 中已下线的 /api/v1/device-rules* 端点
DELETE FROM api_endpoints WHERE path LIKE '/api/v1/device-rules%';

-- +goose Down

-- 删表/删列后无法逆向重建数据，回滚留空。
-- 若必须回滚：恢复完整 DB 备份后重新切到 000160 之前。
SELECT 1;
