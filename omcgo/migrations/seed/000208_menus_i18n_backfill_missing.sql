-- +goose Up
-- ============================================================
-- 2026-05-28 回填 seed/000205 漏掉的 7 项菜单 name_i18n.en-US。
--
-- 现状(进 main + 207 跑过后):仍有 7 个菜单 en-US 为空,英文环境 fallback 显示中文。
--   - 产品中心          (product 一级)
--   - 配置管理          (config 一级)
--   - 批量参数模板      (config:batch-template)
--   - 许可证查看        (system:license:view)
--   - 审计日志查看      (system:license:audit)
--   - 同步              (system:api-management:sync)
--   - License           (system_license 一级,name 本身英文,补 i18n 显式标注让前端切换语境时不再 fallback)
--
-- 匹配策略:沿用 000205 思路 — name 命中即更新,只在 en-US 缺失 / 空 / 错填为同名中文时
-- 才覆写,已正确填过的不动(幂等,可重跑)。
-- ============================================================

-- +goose StatementBegin
UPDATE menus AS m
   SET name_i18n = COALESCE(m.name_i18n, '{}'::jsonb)
                   || jsonb_build_object('zh-CN', m.name, 'en-US', t.en_us),
       updated_at = NOW()
  FROM (VALUES
      -- 一级
      ('产品中心',          'Product Center'),
      ('配置管理',          'Config Management'),
      ('License',           'License'),

      -- 配置 / API 子项
      ('批量参数模板',      'Batch Param Templates'),
      ('许可证查看',        'License View'),
      ('审计日志查看',      'Audit Log View'),
      ('同步',              'Sync')
  ) AS t(zh_cn, en_us)
 WHERE m.name = t.zh_cn
   AND (
       m.name_i18n IS NULL
       OR (m.name_i18n->>'en-US') IS NULL
       OR (m.name_i18n->>'en-US') = ''
       OR (m.name_i18n->>'en-US') = m.name  -- 错填为中文同名
   );
-- +goose StatementEnd


-- +goose Down
-- 反向:不清 i18n — 回填本就是修正空白,Down 删掉只会让英文环境又退回 fallback 中文。
SELECT 1;
