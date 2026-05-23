-- UFTE CONFIG_RESTORE 模板 file_type 文案修正
--
-- 背景：seed/000102 把 file_type 写成 '3 Vendor Configuration File'，文案误用了
-- 备份上传的类型号。配置恢复（Download）规范要求 FileType = "10 <OUI> Configuration File"，
-- OUI 在派发时由 restore_service.go::buildRestoreFileType 用设备真实 OUI 替换 <OUI>。
--
-- 这里只更新 ufte_task_types.file_type / file_type_label 两个展示字段；运行时实际下发的
-- FileType 由 backup/restore_service.go 直接构造，与本表的 file_type 字段解耦（已写硬编码
-- 拼接，不读模板），所以本 seed 只为 UI / 文档一致性服务，不影响下发行为。

-- +goose Up

UPDATE ufte_task_types
   SET file_type       = '10 <OUI> Configuration File',
       file_type_label = '10 <OUI> Configuration File',
       updated_at      = NOW()
 WHERE type_code = 'CONFIG_RESTORE';


-- +goose Down

UPDATE ufte_task_types
   SET file_type       = '3 Vendor Configuration File',
       file_type_label = '3 Vendor Configuration File',
       updated_at      = NOW()
 WHERE type_code = 'CONFIG_RESTORE';
