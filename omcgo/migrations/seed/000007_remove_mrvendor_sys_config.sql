-- +goose Up
-- #453：移除系统配置-基本设置页的「运营商名称」(mrVendor) 死字段的存量数据。
--
-- 该字段是无人消费的半成品/死字段：后端整库无 Go 代码读取 mrVendor，仅在
-- sys_configs (category='basic', key='mrVendor') 按需存储，存了也不被任何业务流程
-- 使用（不进报表抬头、不进页面标题、不参与运营商适配）。运营商适配由
-- internal/core/carrier/ 独立机制驱动，与此字段无关。前端三皮肤已删 v1 表单项与
-- i18n 文案；此处清理可能存在的存量行（v2/v3 数据驱动 KV 编辑器会把残留行渲染为一行）。
--
-- seed baseline 无该字段初始化记录，仅历史上在 v1 保存过才会有此行。
-- 单条 DELETE 即可，幂等（行不存在则无操作）。
DELETE FROM public.sys_configs WHERE category = 'basic' AND key = 'mrVendor';

-- +goose Down
-- 不可逆：删除的是用户历史输入的运营商名称值，无从恢复原值（且该字段已彻底废弃，
-- 不应重新引入）。Down 段为 no-op。
SELECT 1;
