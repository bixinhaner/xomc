-- +goose Up
-- ============================================================
-- 000107_seed_menu_software_hide_transfer_reorder.sql
-- 隐藏"软件管理"菜单 + "文件传输"上移到拓扑管理下方
-- ============================================================

-- 1. 隐藏"软件管理"（升级功能已整合到文件传输模块）
UPDATE menus
SET show_status = 'hide'
WHERE id = 'aaaa0006-0000-0000-0000-000000000001';

-- 2. "文件传输"上移到拓扑管理（sort_order=5）下方
UPDATE menus
SET sort_order = 6
WHERE id = 'aaaa000b-0000-0000-0000-000000000001';

-- 3. "备份恢复"顺移（原 sort_order=6 → 7）
UPDATE menus
SET sort_order = 7
WHERE id = 'aaaa0005-0000-0000-0000-000000000001';

-- +goose Down
-- 恢复"软件管理"显示
UPDATE menus
SET show_status = 'show'
WHERE id = 'aaaa0006-0000-0000-0000-000000000001';

-- 恢复"文件传输"原始排序
UPDATE menus
SET sort_order = 12
WHERE id = 'aaaa000b-0000-0000-0000-000000000001';

-- 恢复"备份恢复"原始排序
UPDATE menus
SET sort_order = 6
WHERE id = 'aaaa0005-0000-0000-0000-000000000001';
