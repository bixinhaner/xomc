-- 旧 /alarm/library 已在 T-0098-P5-06 下线，治理入口迁到 /product/alarm-library。
-- 000088 已尝试按固定 UUID 隐藏旧菜单；当前库里该节点仍为 show，
-- 说明存在历史数据漂移或后续数据回填覆盖。这里按稳定业务键再次收口。

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
SELECT '000089_force_hide_legacy_alarm_library_menu', id, show_status
FROM menus
WHERE status = 'normal'
  AND (
    route_path = '/alarm/library'
    OR permission_key = 'alarm:library'
    OR permission_key LIKE 'alarm:library:%'
  )
ON CONFLICT (migration_key, menu_id) DO UPDATE
SET previous_show_status = EXCLUDED.previous_show_status,
    captured_at = NOW();

UPDATE menus
SET show_status = 'hide', updated_at = NOW()
WHERE status = 'normal'
  AND (
    route_path = '/alarm/library'
    OR permission_key = 'alarm:library'
    OR permission_key LIKE 'alarm:library:%'
  );

-- +goose Down
UPDATE menus AS m
SET show_status = b.previous_show_status, updated_at = NOW()
FROM seed_menu_show_status_backups AS b
WHERE b.migration_key = '000089_force_hide_legacy_alarm_library_menu'
  AND m.id = b.menu_id;

DELETE FROM seed_menu_show_status_backups
WHERE migration_key = '000089_force_hide_legacy_alarm_library_menu';

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