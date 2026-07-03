-- +goose Up
-- #798：物理清理「存储设置」页「日志设置」卡片 4 个废弃字段在 sys_configs（category='storage'）
-- 下的历史孤儿行——这 4 个 key 从未被任何后端代码读取过（老系统遗留的未完成项，非本项目引入），
-- 保存过一次表单的部署会残留这几行。不删除会导致 v2/v3 通用 KV 编辑器继续展示这几个已失效的
-- 裸配置项（v1 已同步删除对应表单字段，见 StorageSettings.tsx）。
--   - logDataSaveDays：原始文件保留天数——排查无剩余业务范围（PM/MR 均已有独立保留机制覆盖）。
--   - rebootLogDataSaveDays：与 stationlog.retention.max_retention_days（#320，真实生效）重复。
--   - rebootLogSaveCount：全局维度配额，已被本轮新增的
--     stationlog.retention.max_file_count_per_device（#798，设备维度）取代。
--   - sysOperateLogDataSaveDays：与 log.retention.oper_days（真实生效）重复。
-- 全新库（未保存过该表单）本次 DELETE 影响 0 行，幂等安全。

DELETE FROM public.sys_configs
WHERE category = 'storage'
  AND key IN ('logDataSaveDays', 'rebootLogDataSaveDays', 'rebootLogSaveCount', 'sysOperateLogDataSaveDays');

-- +goose Down
-- 不可回滚：这 4 行是运维在旧版「日志设置」表单里保存过的用户输入值（非 seed 内置数据），
-- 原值已随 DELETE 丢失，无法还原到用户实际保存过的值；且对应字段功能已判定废弃、前端表单
-- 已一并删除，回滚也无处再编辑。此 Down 仅满足 goose 双向演练形式要求，不做实质操作。
SELECT 1;
