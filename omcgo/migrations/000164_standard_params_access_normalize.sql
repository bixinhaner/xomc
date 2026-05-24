-- 000164_standard_params_access_normalize.sql
--
-- 归一化 standard_params.access 字段值域。
--
-- 背景：
--   standard_params.access 当前混杂 4 种取值：
--     READ_WRITE (1618) / READ_ONLY (383) / RW (67) / RO (100)
--   而真实链路（XML loader / handler / 前端 AccessTypeTag）期望
--   READ_WRITE / READ_ONLY 完整字面值：
--     - data/param-mappings/*.xml 用 access="READ_WRITE" / "READ_ONLY"
--     - parser.go 定义 AccessReadOnly="READ_ONLY", AccessReadWrite="READ_WRITE"
--     - admin_repository.go::ListEnrichedByCommand SQL：
--         COALESCE(sp.access, 'READ_ONLY') AS access_type
--     - 前端 webcode/src/pages/mml/Console/components/AccessTypeTag.tsx 与
--       RightPanel.tsx / SubFieldChecklist.tsx / SubFieldInputList.tsx 全部
--       硬编 'READ_ONLY' / 'READ_WRITE' 字面值比较
--
--   历史 167 条 RW/RO 短值的来源是早期 seed 迁移（migrations/seed/000126 等）
--   或手动 UPSERT。这些值前端拿到后无法匹配 'READ_ONLY' 比较，落到 fallback
--   分支 — 即"既不只读也不读写"，UI 行为不可预期（实测 MML 控制台 v2 测试
--   报告 §3 类 C：M2 命令测试脚本按 ('RW','WO') 过滤拿不到字段 → 400 empty values）。
--
-- 本迁移：把 RW → READ_WRITE，RO → READ_ONLY，与 XML/Loader/handler/前端
-- 期望值域对齐。
--
-- 范围：仅触 standard_params 表。不动 mml_command_sub_fields.access_type
-- （该列实际未被 API 消费，handler 优先用 sp.access，列存值不影响输出 —
-- 后续可独立 cleanup PR 删除该列）。
--
-- 安全：本迁移幂等。已是 READ_WRITE/READ_ONLY 的行不会被改。

-- +goose Up

UPDATE standard_params
   SET access = 'READ_WRITE',
       updated_at = NOW()
 WHERE access = 'RW';

UPDATE standard_params
   SET access = 'READ_ONLY',
       updated_at = NOW()
 WHERE access = 'RO';

-- +goose Down

-- 回滚：恢复短值（仅为对称，无业务意义；回滚后前端会再次失配 → 视觉异常）。
UPDATE standard_params
   SET access = 'RW',
       updated_at = NOW()
 WHERE access = 'READ_WRITE';

UPDATE standard_params
   SET access = 'RO',
       updated_at = NOW()
 WHERE access = 'READ_ONLY';
