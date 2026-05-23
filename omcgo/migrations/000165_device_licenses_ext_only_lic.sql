-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- 简化 license 文件命名：只允许 <SN>.lic（移除 .bin/.dat）
--
-- 原 000162 的 CHECK 是 file_ext IN ('lic','bin','dat')，新规范只保留 .lic。
-- 已有数据全部按 .lic 入库（开发环境暂无非 lic 行），ALTER 安全。
-- ═══════════════════════════════════════════════════════════════════════════

ALTER TABLE device_licenses
    DROP CONSTRAINT IF EXISTS device_licenses_file_ext_check;

ALTER TABLE device_licenses
    ADD CONSTRAINT device_licenses_file_ext_check CHECK (file_ext = 'lic');

-- +goose Down
ALTER TABLE device_licenses
    DROP CONSTRAINT IF EXISTS device_licenses_file_ext_check;

ALTER TABLE device_licenses
    ADD CONSTRAINT device_licenses_file_ext_check CHECK (file_ext IN ('lic', 'bin', 'dat'));
