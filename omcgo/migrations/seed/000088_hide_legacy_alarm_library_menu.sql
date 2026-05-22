-- 旧 /alarm/library 已在 T-0098-P5-06 下线，治理入口迁到 /product/alarm-library。
-- 动态菜单仍会从 menus 表读出历史“告警管理 -> 告警库”节点，导致左侧菜单残留。
-- 本 seed 迁移只隐藏旧菜单与其按钮，不删除历史记录，便于回滚和审计。

-- +goose Up
CREATE TABLE IF NOT EXISTS seed_menu_show_status_backups (
   migration_key VARCHAR(128) NOT NULL,
   menu_id UUID NOT NULL,
   previous_show_status VARCHAR(16) NOT NULL,
   captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
   PRIMARY KEY (migration_key, menu_id),
   CONSTRAINT chk_seed_menu_show_status_backups_status CHECK (previous_show_status IN ('show', 'hide'))
);

INSERT INTO seed_menu_show_status_backups (migration_key, menu_id, previous_show_status)
SELECT '000088_hide_legacy_alarm_library_menu', id, show_status
FROM menus
WHERE id = 'aaaa0011-1000-0000-0000-000000000002'::uuid
   OR parent_id = 'aaaa0011-1000-0000-0000-000000000002'::uuid
ON CONFLICT (migration_key, menu_id) DO UPDATE
SET previous_show_status = EXCLUDED.previous_show_status,
   captured_at = NOW();

UPDATE menus
SET show_status = 'hide', updated_at = NOW()
WHERE id = 'aaaa0011-1000-0000-0000-000000000002'::uuid
   OR parent_id = 'aaaa0011-1000-0000-0000-000000000002'::uuid;

-- +goose Down
UPDATE menus AS m
SET show_status = b.previous_show_status, updated_at = NOW()
FROM seed_menu_show_status_backups AS b
WHERE b.migration_key = '000088_hide_legacy_alarm_library_menu'
  AND m.id = b.menu_id;

DELETE FROM seed_menu_show_status_backups
WHERE migration_key = '000088_hide_legacy_alarm_library_menu';

-- +goose StatementBegin
DO $$
BEGIN
   IF EXISTS (
      SELECT 1
      FROM information_schema.tables
      WHERE table_schema = 'public'
        AND table_name = 'seed_menu_show_status_backups'
   ) AND NOT EXISTS (SELECT 1 FROM seed_menu_show_status_backups) THEN
      DROP TABLE seed_menu_show_status_backups;
   END IF;
END $$;
-- +goose StatementEnd