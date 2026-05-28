-- +goose Up
-- ============================================================
-- 2026-05-27 License 兜底 — 模糊匹配
--
-- 背景:
--   seed/000205 / 000206 用精确 OR (name='License' OR name='许可证管理' OR
--   permission_key='system_license' OR permission_key='license' OR
--   route_path='/license') 应该已覆盖所有已知 License row,但用户反馈
--   "DB 里有一行 License,但 show_status 仍是 show"。
--
--   说明该 row 的字段值与上面所有精确候选都不匹配 — 可能来源:
--     · 客户环境运维通过菜单管理 UI 手改过 name / permission_key / route
--     · 某 patch 脚本 / hot fix 临时塞入的 License 节点
--     · seed/000123 之前 + 之后某种部分混合状态
--
-- 策略:
--   用 ILIKE '%license%' / LIKE '%许可%' 模糊匹配 name + permission_key
--   + route_path + name_i18n JSONB,覆盖任何带 license / 许可 字面的 menu / directory。
--   type 限定 menu / directory(button 子节点本身就不需要单独 hide — 父节点 hide
--   后菜单树渲染时不会展开)。
-- ============================================================

-- +goose StatementBegin
UPDATE menus
   SET show_status = 'hide',
       updated_at  = NOW()
 WHERE (
        name           ILIKE '%license%'
     OR name           LIKE  '%许可证%'
     OR permission_key ILIKE '%license%'
     OR route_path     ILIKE '%license%'
     OR (name_i18n IS NOT NULL AND name_i18n::text ILIKE '%license%')
   )
   AND type IN ('menu', 'directory')
   AND show_status IS DISTINCT FROM 'hide';
-- +goose StatementEnd


-- +goose Down
-- 不做反向 — 模糊匹配的范围不确定,统一恢复 show 风险大;
-- 真要回滚走 000205/000206 的 Down,然后单独 UPDATE 这里被改的 row。
SELECT 1;
