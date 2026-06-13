-- +goose Up
-- 审计/业务日志保留（log.retention）+ 运行期日志文件轮转（log.rotation）的 sys_configs 默认值。
-- 让这些配置出现在前端「系统配置」页可改、并由各模块热加载（worker 清理 cron 读 log.retention；
-- app/acs/worker 的 logger override watcher 读 log.rotation）。代码侧均有同值兜底默认，故本 seed
-- 缺失不影响功能，仅为可见/可配。
-- 幂等：ON CONFLICT (category,key) DO NOTHING —— 已存在的键不覆盖运维改过的值。
INSERT INTO public.sys_configs (category, key, value, value_type, description, is_public) VALUES
  -- 审计/业务日志按时间保留（log.retention）：worker 每日 05:00 cron 批量删过期行。
  -- 安全/审计类久留、高频报文类短留；总开关 enabled。
  ('log.retention', 'enabled', 'true', 'bool', '日志保留清理总开关（关则跳过整轮清理）', false),
  ('log.retention', 'audit_days', '180', 'int', '审计日志(audit_logs)保留天数', false),
  ('log.retention', 'ops_audit_days', '180', 'int', '运维审计日志(ops_audit_logs)保留天数', false),
  ('log.retention', 'login_days', '180', 'int', '登录日志(sys_login_logs)保留天数', false),
  ('log.retention', 'oper_days', '180', 'int', '操作日志(sys_oper_logs)保留天数', false),
  ('log.retention', 'task_days', '90', 'int', '任务日志(sys_task_logs)保留天数', false),
  ('log.retention', 'system_days', '90', 'int', '系统日志(system_logs)保留天数', false),
  ('log.retention', 'ne_message_days', '30', 'int', '网元报文日志(ne_message_logs)保留天数', false),
  ('log.retention', 'event_days', '90', 'int', '设备事件日志(event_logs)保留天数', false),

  -- 运行期日志文件轮转（log.rotation）：app/acs/worker 各自 logger 读取，热加载（≤1 分钟生效）。
  -- max_size_mb 调小即时生效（compactor 检测活动文件超限即切割），调大以各服务 YAML 为硬兜底。
  ('log.rotation', 'max_size_mb', '50', 'int', '单个日志文件触发切割的大小(MB)', false),
  ('log.rotation', 'max_age_days', '30', 'int', '日志归档保留天数(早于此的归档删除)', false),
  ('log.rotation', 'keep_files', '10', 'int', '保持不压缩的最新归档个数(可直接 tail 的近期文件数)', false)
ON CONFLICT (category, key) DO NOTHING;

-- +goose Down
DELETE FROM public.sys_configs
WHERE category IN ('log.retention', 'log.rotation');
