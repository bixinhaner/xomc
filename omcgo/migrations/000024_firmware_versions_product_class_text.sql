-- +goose Up
-- 放宽 firmware_versions.product_class 至 TEXT。
--
-- 根因：固件支持「一个固件适配多个产品类型」——前端多选(mode="multiple")后用逗号
-- 拼成 CSV 存入单列（如 "gNB-100,gNB-200,FAP-LTE-300,..."），但该列仍是早期单值时代的
-- varchar(64)。选到 7 个机型(~79 字符)即超长，UPDATE 报
-- `value too long for type character varying(64)`（SQLSTATE 22001），编辑升级文件失败。
-- 放宽为 TEXT，避免机型数增多后再次溢出。放宽不重写数据、不丢数据。
ALTER TABLE firmware_versions ALTER COLUMN product_class TYPE text;

-- +goose Down
-- 回退到 varchar(64)；若存量数据已有 >64 字符的 CSV 会失败（属预期，避免静默截断）。
ALTER TABLE firmware_versions ALTER COLUMN product_class TYPE varchar(64);
