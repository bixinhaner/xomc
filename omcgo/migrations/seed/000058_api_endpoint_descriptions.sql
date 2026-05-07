-- +goose Up
-- 批量为 api_endpoints.description 补全（按 path+method+group 启发式推断）。
-- 仅覆盖当前 description 为空的行；手工填入的描述不被覆盖。
-- 数据由 scripts/gen_api_desc 派生（path+method 精确匹配），新增路由后再次同步即可。
UPDATE api_endpoints SET description = CASE
    WHEN method = 'GET' AND path = '/api/v1/alarms/:id' THEN '告警 - 详情'
    WHEN method = 'POST' AND path = '/api/v1/alarms/:id/acknowledge' THEN '告警 - 确认'
    WHEN method = 'POST' AND path = '/api/v1/alarms/:id/clear' THEN '告警 - 清除'
    WHEN method = 'GET' AND path = '/api/v1/alarms/active' THEN '告警 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/alarms/active/:id/read' THEN '告警 - 标记已读'
    WHEN method = 'POST' AND path = '/api/v1/alarms/active/batch/acknowledge' THEN '告警 - 批量确认'
    WHEN method = 'POST' AND path = '/api/v1/alarms/active/batch/clear' THEN '告警 - 批量清除'
    WHEN method = 'POST' AND path = '/api/v1/alarms/active/batch/unacknowledge' THEN '告警 - 批量取消确认'
    WHEN method = 'GET' AND path = '/api/v1/alarms/alarm-filters' THEN '告警 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/alarms/alarm-filters' THEN '告警 - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/alarms/alarm-filters/:id' THEN '告警 - 删除'
    WHEN method = 'GET' AND path = '/api/v1/alarms/alarm-filters/:id' THEN '告警 - 详情'
    WHEN method = 'PUT' AND path = '/api/v1/alarms/alarm-filters/:id' THEN '告警 - 更新'
    WHEN method = 'POST' AND path = '/api/v1/alarms/alarm-filters/:id/toggle' THEN '告警 - 切换启用状态'
    WHEN method = 'GET' AND path = '/api/v1/alarms/alarm-libraries' THEN '告警 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/alarms/alarm-libraries' THEN '告警 - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/alarms/alarm-libraries/:id' THEN '告警 - 删除'
    WHEN method = 'GET' AND path = '/api/v1/alarms/alarm-libraries/:id' THEN '告警 - 详情'
    WHEN method = 'PUT' AND path = '/api/v1/alarms/alarm-libraries/:id' THEN '告警 - 更新'
    WHEN method = 'GET' AND path = '/api/v1/alarms/alarm-libraries/:id/i18n' THEN '告警 - 国际化文案'
    WHEN method = 'POST' AND path = '/api/v1/alarms/alarm-libraries/:id/i18n' THEN '告警 - 国际化文案'
    WHEN method = 'DELETE' AND path = '/api/v1/alarms/alarm-libraries/:id/i18n/:i18nId' THEN '告警 - 国际化文案删除'
    WHEN method = 'GET' AND path = '/api/v1/alarms/history' THEN '告警 - 历史记录'
    WHEN method = 'POST' AND path = '/api/v1/alarms/history/batch/acknowledge' THEN '告警 - 批量确认'
    WHEN method = 'POST' AND path = '/api/v1/alarms/history/batch/delete' THEN '告警 - 批量删除'
    WHEN method = 'POST' AND path = '/api/v1/alarms/history/batch/unacknowledge' THEN '告警 - 批量取消确认'
    WHEN method = 'GET' AND path = '/api/v1/alarms/history/statistics' THEN '告警 - 历史统计'
    WHEN method = 'GET' AND path = '/api/v1/alarms/statistics' THEN '告警 - 统计'
    WHEN method = 'POST' AND path = '/api/v1/alarms/sync/:device_sn' THEN '告警 - 触发同步'
    WHEN method = 'GET' AND path = '/api/v1/admin/api-endpoints' THEN 'API 端点 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/admin/api-endpoints' THEN 'API 端点 - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/admin/api-endpoints/:id' THEN 'API 端点 - 删除'
    WHEN method = 'PUT' AND path = '/api/v1/admin/api-endpoints/:id' THEN 'API 端点 - 更新'
    WHEN method = 'DELETE' AND path = '/api/v1/admin/api-endpoints/batch' THEN 'API 端点 - 批量删除'
    WHEN method = 'GET' AND path = '/api/v1/admin/api-endpoints/groups' THEN 'API 端点 - 分组'
    WHEN method = 'POST' AND path = '/api/v1/admin/api-endpoints/sync' THEN 'API 端点 - 触发同步'
    WHEN method = 'GET' AND path = '/api/v1/api-keys' THEN 'API 密钥 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/api-keys' THEN 'API 密钥 - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/api-keys/:id' THEN 'API 密钥 - 删除'
    WHEN method = 'GET' AND path = '/api/v1/admin/audit-logs' THEN '审计日志 - 列表'
    WHEN method = 'GET' AND path = '/api/v1/auth/captcha' THEN '认证 - 获取验证码'
    WHEN method = 'POST' AND path = '/api/v1/auth/change-password' THEN '认证 - 修改密码'
    WHEN method = 'POST' AND path = '/api/v1/auth/login' THEN '认证 - 登录'
    WHEN method = 'GET' AND path = '/api/v1/auth/me' THEN '认证 - 当前用户信息'
    WHEN method = 'GET' AND path = '/api/v1/auth/menus' THEN '认证 - menus'
    WHEN method = 'POST' AND path = '/api/v1/auth/refresh' THEN '认证 - 刷新令牌'
    WHEN method = 'POST' AND path = '/api/v1/auth/switch-role' THEN '认证 - 切换角色'
    WHEN method = 'GET' AND path = '/api/v1/backup/ftp-configs' THEN '备份 - FTP 配置'
    WHEN method = 'POST' AND path = '/api/v1/backup/ftp-configs' THEN '备份 - FTP 配置'
    WHEN method = 'DELETE' AND path = '/api/v1/backup/ftp-configs/:id' THEN '备份 - FTP 配置删除'
    WHEN method = 'PUT' AND path = '/api/v1/backup/ftp-configs/:id' THEN '备份 - FTP 配置更新'
    WHEN method = 'POST' AND path = '/api/v1/backup/ftp-configs/:id/test' THEN '备份 - 测试'
    WHEN method = 'GET' AND path = '/api/v1/backup/policy' THEN '备份 - 策略'
    WHEN method = 'PUT' AND path = '/api/v1/backup/policy' THEN '备份 - 策略'
    WHEN method = 'POST' AND path = '/api/v1/backup/restore' THEN '备份 - 恢复'
    WHEN method = 'GET' AND path = '/api/v1/backup/restore-tasks' THEN '备份 - 恢复任务'
    WHEN method = 'GET' AND path = '/api/v1/backup/restore-tasks/:id' THEN '备份 - 恢复任务详情'
    WHEN method = 'POST' AND path = '/api/v1/backup/restore/by-task-id' THEN '备份 - 创建'
    WHEN method = 'GET' AND path = '/api/v1/backup/schedules' THEN '备份 - 调度计划'
    WHEN method = 'POST' AND path = '/api/v1/backup/schedules' THEN '备份 - 调度计划'
    WHEN method = 'DELETE' AND path = '/api/v1/backup/schedules/:id' THEN '备份 - 调度计划删除'
    WHEN method = 'PUT' AND path = '/api/v1/backup/schedules/:id' THEN '备份 - 调度计划更新'
    WHEN method = 'GET' AND path = '/api/v1/backup/tasks' THEN '备份 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/backup/tasks' THEN '备份 - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/backup/tasks/:id' THEN '备份 - 删除'
    WHEN method = 'GET' AND path = '/api/v1/backup/tasks/:id' THEN '备份 - 详情'
    WHEN method = 'POST' AND path = '/api/v1/backup/tasks/:id/cancel' THEN '备份 - 取消'
    WHEN method = 'POST' AND path = '/api/v1/cell/perfmgmt/kpimanage/disableIndicator' THEN '小区 - KPI 管理禁用指标'
    WHEN method = 'POST' AND path = '/api/v1/cell/perfmgmt/kpimanage/enableIndicator' THEN '小区 - KPI 管理启用指标'
    WHEN method = 'GET' AND path = '/api/v1/cell/perfmgmt/kpimanage/isIndicatorInTemplate' THEN '小区 - KPI 管理是否在模板中'
    WHEN method = 'GET' AND path = '/api/v1/column-configs/:pageKey' THEN '列配置 - 详情'
    WHEN method = 'PUT' AND path = '/api/v1/column-configs/:pageKey' THEN '列配置 - 更新'
    WHEN method = 'GET' AND path = '/api/v1/config/baselines' THEN '配置 - 基线'
    WHEN method = 'POST' AND path = '/api/v1/config/baselines' THEN '配置 - 基线'
    WHEN method = 'DELETE' AND path = '/api/v1/config/baselines/:id' THEN '配置 - 基线删除'
    WHEN method = 'GET' AND path = '/api/v1/config/baselines/:id' THEN '配置 - 基线详情'
    WHEN method = 'PUT' AND path = '/api/v1/config/baselines/:id' THEN '配置 - 基线更新'
    WHEN method = 'GET' AND path = '/api/v1/config/neighbors' THEN '配置 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/config/sync/pull/:deviceId' THEN '配置 - 拉取'
    WHEN method = 'POST' AND path = '/api/v1/config/sync/push/:deviceId' THEN '配置 - 推送'
    WHEN method = 'GET' AND path = '/api/v1/config/sync/status/:deviceId' THEN '配置 - 详情'
    WHEN method = 'GET' AND path = '/api/v1/config/tasks' THEN '配置 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/config/tasks' THEN '配置 - 创建'
    WHEN method = 'GET' AND path = '/api/v1/dashboard/alarm-trend' THEN '仪表盘 - 告警趋势'
    WHEN method = 'GET' AND path = '/api/v1/dashboard/alarm-type-pie' THEN '仪表盘 - 告警类型分布'
    WHEN method = 'GET' AND path = '/api/v1/dashboard/device-status' THEN '仪表盘 - 设备状态分布'
    WHEN method = 'GET' AND path = '/api/v1/dashboard/kpi-time-series' THEN '仪表盘 - KPI 时间序列'
    WHEN method = 'GET' AND path = '/api/v1/dashboard/kpi-trend' THEN '仪表盘 - KPI 趋势'
    WHEN method = 'GET' AND path = '/api/v1/dashboard/region-stats' THEN '仪表盘 - 区域统计'
    WHEN method = 'GET' AND path = '/api/v1/dashboard/summary' THEN '仪表盘 - 汇总'
    WHEN method = 'GET' AND path = '/api/v1/dashboard/widgets' THEN '仪表盘 - 组件'
    WHEN method = 'PUT' AND path = '/api/v1/dashboard/widgets' THEN '仪表盘 - 组件'
    WHEN method = 'GET' AND path = '/api/v1/datamodels' THEN '数据模型 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/datamodels' THEN '数据模型 - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/datamodels/:id' THEN '数据模型 - 删除'
    WHEN method = 'GET' AND path = '/api/v1/datamodels/:id' THEN '数据模型 - 详情'
    WHEN method = 'PUT' AND path = '/api/v1/datamodels/:id' THEN '数据模型 - 更新'
    WHEN method = 'POST' AND path = '/api/v1/datamodels/:id/activate' THEN '数据模型 - 激活'
    WHEN method = 'POST' AND path = '/api/v1/datamodels/:id/deprecate' THEN '数据模型 - deprecate'
    WHEN method = 'GET' AND path = '/api/v1/datamodels/:id/export' THEN '数据模型 - 导出'
    WHEN method = 'POST' AND path = '/api/v1/datamodels/cache/refresh' THEN '数据模型 - 刷新令牌'
    WHEN method = 'POST' AND path = '/api/v1/datamodels/import' THEN '数据模型 - 创建'
    WHEN method = 'POST' AND path = '/api/v1/datamodels/import-xml' THEN '数据模型 - 创建'
    WHEN method = 'GET' AND path = '/api/v1/datamodels/resolve' THEN '数据模型 - 列表'
    WHEN method = 'GET' AND path = '/api/v1/datamodels/statistics' THEN '数据模型 - 统计'
    WHEN method = 'GET' AND path = '/api/v1/admin/dead-letters' THEN '死信队列 - 列表'
    WHEN method = 'DELETE' AND path = '/api/v1/admin/dead-letters/:id' THEN '死信队列 - 删除'
    WHEN method = 'GET' AND path = '/api/v1/admin/dead-letters/:id' THEN '死信队列 - 详情'
    WHEN method = 'POST' AND path = '/api/v1/admin/dead-letters/:id/replay' THEN '死信队列 - 重放'
    WHEN method = 'POST' AND path = '/api/v1/device-groups' THEN '设备分组 - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/device-groups/:id' THEN '设备分组 - 删除'
    WHEN method = 'GET' AND path = '/api/v1/device-groups/:id' THEN '设备分组 - 详情'
    WHEN method = 'PUT' AND path = '/api/v1/device-groups/:id' THEN '设备分组 - 更新'
    WHEN method = 'GET' AND path = '/api/v1/device-groups/:id/check-delete' THEN '设备分组 - 校验是否可删除'
    WHEN method = 'DELETE' AND path = '/api/v1/device-groups/:id/devices' THEN '设备分组 - 设备列表'
    WHEN method = 'GET' AND path = '/api/v1/device-groups/:id/devices' THEN '设备分组 - 设备列表'
    WHEN method = 'POST' AND path = '/api/v1/device-groups/:id/devices' THEN '设备分组 - 设备列表'
    WHEN method = 'POST' AND path = '/api/v1/device-groups/move-devices' THEN '设备分组 - 移动设备'
    WHEN method = 'PUT' AND path = '/api/v1/device-groups/sort' THEN '设备分组 - 排序'
    WHEN method = 'GET' AND path = '/api/v1/device-groups/stats' THEN '设备分组 - 统计'
    WHEN method = 'GET' AND path = '/api/v1/device-groups/tree' THEN '设备分组 - 树形结构'
    WHEN method = 'GET' AND path = '/api/v1/device-registrations' THEN '设备预登记 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/device-registrations' THEN '设备预登记 - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/device-registrations/:id' THEN '设备预登记 - 删除'
    WHEN method = 'GET' AND path = '/api/v1/device-rules' THEN '设备规则 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/device-rules' THEN '设备规则 - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/device-rules/:id' THEN '设备规则 - 删除'
    WHEN method = 'GET' AND path = '/api/v1/device-rules/:id' THEN '设备规则 - 详情'
    WHEN method = 'PUT' AND path = '/api/v1/device-rules/:id' THEN '设备规则 - 更新'
    WHEN method = 'POST' AND path = '/api/v1/device-rules/:id/apply' THEN '设备规则 - 应用'
    WHEN method = 'GET' AND path = '/api/v1/device-rules/:id/tasks' THEN '设备规则 - tasks'
    WHEN method = 'GET' AND path = '/api/v1/device-rules/:id/tasks/:taskId' THEN '设备规则 - 详情'
    WHEN method = 'PATCH' AND path = '/api/v1/device-rules/:id/toggle' THEN '设备规则 - 切换启用状态'
    WHEN method = 'PUT' AND path = '/api/v1/device-rules/batch-sort' THEN '设备规则 - 批量更新'
    WHEN method = 'GET' AND path = '/api/v1/device-rules/next-priority' THEN '设备规则 - 下一可用优先级'
    WHEN method = 'GET' AND path = '/api/v1/devices' THEN '设备 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/devices' THEN '设备 - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/devices/:id' THEN '设备 - 删除'
    WHEN method = 'GET' AND path = '/api/v1/devices/:id' THEN '设备 - 详情'
    WHEN method = 'PUT' AND path = '/api/v1/devices/:id' THEN '设备 - 更新'
    WHEN method = 'PUT' AND path = '/api/v1/devices/:id/activate' THEN '设备 - 激活'
    WHEN method = 'POST' AND path = '/api/v1/devices/:id/config-file/sync' THEN '设备 - 触发同步'
    WHEN method = 'PUT' AND path = '/api/v1/devices/:id/deactivate' THEN '设备 - 停用'
    WHEN method = 'GET' AND path = '/api/v1/devices/:id/detail' THEN '设备 - 详情'
    WHEN method = 'GET' AND path = '/api/v1/devices/:id/info' THEN '设备 - 信息'
    WHEN method = 'PUT' AND path = '/api/v1/devices/:id/info' THEN '设备 - 信息'
    WHEN method = 'POST' AND path = '/api/v1/devices/:id/objects/add' THEN '设备 - 多实例对象新增'
    WHEN method = 'POST' AND path = '/api/v1/devices/:id/objects/delete' THEN '设备 - 多实例对象删除'
    WHEN method = 'POST' AND path = '/api/v1/devices/:id/param-sync' THEN '设备 - 参数同步'
    WHEN method = 'GET' AND path = '/api/v1/devices/:id/parameters' THEN '设备 - 参数'
    WHEN method = 'PUT' AND path = '/api/v1/devices/:id/parameters' THEN '设备 - 参数'
    WHEN method = 'GET' AND path = '/api/v1/devices/:id/parameters/children' THEN '设备 - 参数子节点'
    WHEN method = 'POST' AND path = '/api/v1/devices/:id/parameters/discover' THEN '设备 - 参数探测'
    WHEN method = 'GET' AND path = '/api/v1/devices/:id/parameters/schema' THEN '设备 - 参数架构 schema'
    WHEN method = 'GET' AND path = '/api/v1/devices/:id/parameters/search' THEN '设备 - 参数搜索'
    WHEN method = 'POST' AND path = '/api/v1/devices/:id/parameters/sync' THEN '设备 - 参数触发同步'
    WHEN method = 'GET' AND path = '/api/v1/devices/:id/parameters/sync-status' THEN '设备 - 参数同步状态'
    WHEN method = 'GET' AND path = '/api/v1/devices/:id/parameters/tree' THEN '设备 - 参数树形结构'
    WHEN method = 'POST' AND path = '/api/v1/devices/:id/reboot' THEN '设备 - 重启'
    WHEN method = 'PUT' AND path = '/api/v1/devices/:id/rf-switch' THEN '设备 - 射频开关'
    WHEN method = 'DELETE' AND path = '/api/v1/devices/batch' THEN '设备 - 批量删除'
    WHEN method = 'POST' AND path = '/api/v1/devices/batch-reboot' THEN '设备 - 批量重启'
    WHEN method = 'GET' AND path = '/api/v1/devices/enums' THEN '设备 - 枚举字典'
    WHEN method = 'GET' AND path = '/api/v1/devices/export' THEN '设备 - 导出'
    WHEN method = 'GET' AND path = '/api/v1/devices/geo' THEN '设备 - 地理位置数据'
    WHEN method = 'GET' AND path = '/api/v1/devices/geo/stats' THEN '设备 - 统计'
    WHEN method = 'GET' AND path = '/api/v1/devices/product-classes' THEN '设备 - 产品型号'
    WHEN method = 'GET' AND path = '/api/v1/devices/recycle' THEN '设备 - 回收站'
    WHEN method = 'DELETE' AND path = '/api/v1/devices/recycle/permanent' THEN '设备 - 回收站彻底删除'
    WHEN method = 'PATCH' AND path = '/api/v1/devices/recycle/restore' THEN '设备 - 回收站恢复'
    WHEN method = 'GET' AND path = '/api/v1/devices/search' THEN '设备 - 搜索'
    WHEN method = 'GET' AND path = '/api/v1/devices/stats' THEN '设备 - 统计'
    WHEN method = 'GET' AND path = '/api/v1/devices/tasks' THEN '设备 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/devices/tasks' THEN '设备 - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/devices/tasks/:task_id' THEN '设备 - 删除'
    WHEN method = 'GET' AND path = '/api/v1/devices/tasks/:task_id' THEN '设备 - 详情'
    WHEN method = 'POST' AND path = '/api/v1/devices/tasks/:task_id/retry' THEN '设备 - 重试'
    WHEN method = 'POST' AND path = '/api/v1/devices/tasks/batch' THEN '设备 - 创建'
    WHEN method = 'GET' AND path = '/api/v1/devices/tasks/pending' THEN '设备 - 列表'
    WHEN method = 'GET' AND path = '/api/v1/devices/tasks/stats' THEN '设备 - 统计'
    WHEN method = 'GET' AND path = '/api/v1/events/stream' THEN '事件 - SSE 推送流'
    WHEN method = 'GET' AND path = '/api/v1/files' THEN '文件 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/files' THEN '文件 - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/files/:id' THEN '文件 - 删除'
    WHEN method = 'GET' AND path = '/api/v1/files/:id' THEN '文件 - 详情'
    WHEN method = 'POST' AND path = '/api/v1/files/:id/distribute' THEN '文件 - distribute'
    WHEN method = 'GET' AND path = '/api/v1/files/:id/download' THEN '文件 - 下载'
    WHEN method = 'GET' AND path = '/api/v1/firmware' THEN '固件 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/firmware' THEN '固件 - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/firmware/:id' THEN '固件 - 删除'
    WHEN method = 'GET' AND path = '/api/v1/firmware/:id' THEN '固件 - 详情'
    WHEN method = 'PUT' AND path = '/api/v1/firmware/:id' THEN '固件 - 更新'
    WHEN method = 'GET' AND path = '/api/v1/firmware/:id/download' THEN '固件 - 下载'
    WHEN method = 'PUT' AND path = '/api/v1/firmware/:id/recommend' THEN '固件 - 设为推荐'
    WHEN method = 'POST' AND path = '/api/v1/gnb/pm/indicatormg/addIndicatorGroup' THEN 'gNB - 指标管理新增指标分组'
    WHEN method = 'POST' AND path = '/api/v1/gnb/pm/indicatormg/addOrModifyIndicator' THEN 'gNB - 指标管理新增或修改指标'
    WHEN method = 'POST' AND path = '/api/v1/gnb/pm/indicatormg/delIndicator' THEN 'gNB - 指标管理删除指标'
    WHEN method = 'POST' AND path = '/api/v1/gnb/pm/indicatormg/delIndicatorGroup' THEN 'gNB - 指标管理删除指标分组'
    WHEN method = 'POST' AND path = '/api/v1/gnb/pm/indicatormg/exportAllIndicator' THEN 'gNB - 指标管理导出全部指标'
    WHEN method = 'POST' AND path = '/api/v1/gnb/pm/indicatormg/getIndicatorGroupInfo' THEN 'gNB - 指标管理查询指标分组信息'
    WHEN method = 'POST' AND path = '/api/v1/gnb/pm/indicatormg/getIndicatorGroupTree' THEN 'gNB - 指标管理查询指标分组树'
    WHEN method = 'POST' AND path = '/api/v1/gnb/pm/indicatormg/getIndicatorInfo' THEN 'gNB - 指标管理查询指标信息'
    WHEN method = 'POST' AND path = '/api/v1/gnb/pm/indicatormg/getIndicatorListByPage' THEN 'gNB - 指标管理查询指标分页列表'
    WHEN method = 'POST' AND path = '/api/v1/gnb/pm/indicatormg/modifyIndicatorGroup' THEN 'gNB - 指标管理修改指标分组'
    WHEN method = 'POST' AND path = '/api/v1/gnb/pm/indicatormg/updateBaseKpiCustName' THEN 'gNB - 指标管理更新基础 KPI 自定义名'
    WHEN method = 'POST' AND path = '/api/v1/gnb/pm/indicatormg/updateGnbIndicatorsName' THEN 'gNB - 指标管理更新gNB 指标名'
    WHEN method = 'GET' AND path = '/api/v1/admin/groups' THEN '用户组 - 列表'
    WHEN method = 'GET' AND path = '/api/v1/admin/groups/:id' THEN '用户组 - 详情'
    WHEN method = 'GET' AND path = '/api/v1/groups' THEN '用户组 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/groups' THEN '用户组 - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/groups/:id' THEN '用户组 - 删除'
    WHEN method = 'GET' AND path = '/api/v1/groups/:id' THEN '用户组 - 详情'
    WHEN method = 'PUT' AND path = '/api/v1/groups/:id' THEN '用户组 - 更新'
    WHEN method = 'GET' AND path = '/api/v1/groups/:id/devices' THEN '用户组 - 设备列表'
    WHEN method = 'POST' AND path = '/api/v1/groups/:id/devices' THEN '用户组 - 设备列表'
    WHEN method = 'DELETE' AND path = '/api/v1/groups/:id/devices/:deviceId' THEN '用户组 - 设备列表删除'
    WHEN method = 'GET' AND path = '/healthz' THEN '健康检查'
    WHEN method = 'POST' AND path = '/api/v1/interop/run' THEN '互操作 - 执行'
    WHEN method = 'POST' AND path = '/api/v1/interop/run/:category' THEN '互操作 - 执行'
    WHEN method = 'GET' AND path = '/api/v1/interop/test-cases' THEN '互操作 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/interop/validate/:deviceId' THEN '互操作 - 校验'
    WHEN method = 'GET' AND path = '/api/v1/licenses' THEN '许可证 - 列表'
    WHEN method = 'GET' AND path = '/api/v1/licenses/:id' THEN '许可证 - 详情'
    WHEN method = 'POST' AND path = '/api/v1/licenses/:id/revoke' THEN '许可证 - 撤销'
    WHEN method = 'POST' AND path = '/api/v1/licenses/activate' THEN '许可证 - 激活'
    WHEN method = 'POST' AND path = '/api/v1/licenses/import' THEN '许可证 - 创建'
    WHEN method = 'GET' AND path = '/api/v1/licenses/quota' THEN '许可证 - 列表'
    WHEN method = 'GET' AND path = '/api/v1/licenses/summary' THEN '许可证 - 汇总'
    WHEN method = 'GET' AND path = '/api/v1/admin/logs/login' THEN '日志 - 登录'
    WHEN method = 'GET' AND path = '/api/v1/admin/logs/operation' THEN '日志 - 列表'
    WHEN method = 'GET' AND path = '/api/v1/admin/logs/task' THEN '日志 - 列表'
    WHEN method = 'GET' AND path = '/api/v1/logs/ne-messages' THEN '日志 - 列表'
    WHEN method = 'GET' AND path = '/api/v1/logs/system' THEN '日志 - 列表'
    WHEN method = 'DELETE' AND path = '/api/v1/admin/menus' THEN '菜单 - 批量删除'
    WHEN method = 'GET' AND path = '/api/v1/admin/menus' THEN '菜单 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/admin/menus' THEN '菜单 - 创建'
    WHEN method = 'GET' AND path = '/api/v1/admin/menus/:id' THEN '菜单 - 详情'
    WHEN method = 'PUT' AND path = '/api/v1/admin/menus/:id' THEN '菜单 - 更新'
    WHEN method = 'GET' AND path = '/api/v1/admin/menus/tree' THEN '菜单 - 树形结构'
    WHEN method = 'GET' AND path = '/api/v1/admin/menus/user-tree' THEN '菜单 - 列表'
    WHEN method = 'GET' AND path = '/api/v1/mml/commands' THEN 'MML - 命令'
    WHEN method = 'GET' AND path = '/api/v1/mml/commands/:id' THEN 'MML - 命令详情'
    WHEN method = 'GET' AND path = '/api/v1/mml/commands/:id/param-paths' THEN 'MML - 参数路径'
    WHEN method = 'GET' AND path = '/api/v1/mml/dangerous-check' THEN 'MML - 危险命令检查'
    WHEN method = 'POST' AND path = '/api/v1/mml/execute' THEN 'MML - 执行'
    WHEN method = 'GET' AND path = '/api/v1/mml/param-versions' THEN 'MML - 参数版本'
    WHEN method = 'GET' AND path = '/api/v1/mml/param-versions/:version/groups' THEN 'MML - 分组'
    WHEN method = 'GET' AND path = '/api/v1/mml/param-versions/:version/groups/:groupId/params' THEN 'MML - 参数'
    WHEN method = 'GET' AND path = '/api/v1/mml/param-versions/:version/params' THEN 'MML - 参数'
    WHEN method = 'GET' AND path = '/api/v1/mml/scripts' THEN 'MML - 脚本'
    WHEN method = 'POST' AND path = '/api/v1/mml/scripts' THEN 'MML - 脚本'
    WHEN method = 'DELETE' AND path = '/api/v1/mml/scripts/:id' THEN 'MML - 脚本删除'
    WHEN method = 'GET' AND path = '/api/v1/mml/scripts/:id' THEN 'MML - 脚本详情'
    WHEN method = 'PUT' AND path = '/api/v1/mml/scripts/:id' THEN 'MML - 脚本更新'
    WHEN method = 'POST' AND path = '/api/v1/mml/scripts/:id/cancel' THEN 'MML - 取消'
    WHEN method = 'POST' AND path = '/api/v1/mml/scripts/:id/pause' THEN 'MML - 暂停'
    WHEN method = 'GET' AND path = '/api/v1/mml/scripts/:id/runs' THEN 'MML - 执行历史'
    WHEN method = 'POST' AND path = '/api/v1/mml/scripts/:id/start' THEN 'MML - 启动'
    WHEN method = 'GET' AND path = '/api/v1/mml/tasks' THEN 'MML - 列表'
    WHEN method = 'POST' AND path = '/api/v1/mml/tasks' THEN 'MML - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/mml/tasks/:id' THEN 'MML - 删除'
    WHEN method = 'GET' AND path = '/api/v1/mml/tasks/:id' THEN 'MML - 详情'
    WHEN method = 'POST' AND path = '/api/v1/mml/tasks/:id/cancel' THEN 'MML - 取消'
    WHEN method = 'POST' AND path = '/api/v1/mml/tasks/:id/pause' THEN 'MML - 暂停'
    WHEN method = 'GET' AND path = '/api/v1/mml/tasks/:id/results' THEN 'MML - 结果'
    WHEN method = 'POST' AND path = '/api/v1/mml/tasks/:id/start' THEN 'MML - 启动'
    WHEN method = 'GET' AND path = '/api/v1/mml/templates' THEN 'MML - 模板'
    WHEN method = 'POST' AND path = '/api/v1/mml/templates' THEN 'MML - 模板'
    WHEN method = 'DELETE' AND path = '/api/v1/mml/templates/:id' THEN 'MML - 模板删除'
    WHEN method = 'GET' AND path = '/api/v1/mml/templates/:id' THEN 'MML - 模板详情'
    WHEN method = 'PUT' AND path = '/api/v1/mml/templates/:id' THEN 'MML - 模板更新'
    WHEN method = 'POST' AND path = '/api/v1/mml/templates/:id/clone' THEN 'MML - 克隆'
    WHEN method = 'GET' AND path = '/api/v1/mr/data' THEN '测量报告 MR - 列表'
    WHEN method = 'POST' AND path = '/api/v1/mr/export' THEN '测量报告 MR - 导出'
    WHEN method = 'GET' AND path = '/api/v1/mr/files' THEN '测量报告 MR - 列表'
    WHEN method = 'GET' AND path = '/api/v1/mr/files/:id/download' THEN '测量报告 MR - 下载'
    WHEN method = 'GET' AND path = '/api/v1/mr/indicators' THEN '测量报告 MR - 指标'
    WHEN method = 'GET' AND path = '/api/v1/mr/indicators/:code/stats' THEN '测量报告 MR - 统计'
    WHEN method = 'GET' AND path = '/api/v1/mr/indicators/all' THEN '测量报告 MR - 指标全量'
    WHEN method = 'GET' AND path = '/api/v1/mr/mappings' THEN '测量报告 MR - 映射'
    WHEN method = 'PUT' AND path = '/api/v1/mr/mappings/:id' THEN '测量报告 MR - 映射更新'
    WHEN method = 'PUT' AND path = '/api/v1/mr/mappings/:id/toggle' THEN '测量报告 MR - 切换启用状态'
    WHEN method = 'POST' AND path = '/api/v1/northbound/export/alarms' THEN '北向接口 - 导出告警'
    WHEN method = 'GET' AND path = '/api/v1/northbound/export/config/:deviceId' THEN '北向接口 - 导出配置详情'
    WHEN method = 'POST' AND path = '/api/v1/northbound/export/pm' THEN '北向接口 - 导出PM'
    WHEN method = 'GET' AND path = '/api/v1/northbound/push/deadletter' THEN '北向接口 - 推送死信'
    WHEN method = 'POST' AND path = '/api/v1/northbound/push/deadletter/:id/replay' THEN '北向接口 - 重放'
    WHEN method = 'GET' AND path = '/api/v1/northbound/push/targets' THEN '北向接口 - 推送推送目标'
    WHEN method = 'POST' AND path = '/api/v1/northbound/push/targets' THEN '北向接口 - 推送推送目标'
    WHEN method = 'DELETE' AND path = '/api/v1/northbound/push/targets/:id' THEN '北向接口 - 推送推送目标删除'
    WHEN method = 'GET' AND path = '/api/v1/northbound/push/targets/:id/circuit' THEN '北向接口 - 熔断器状态'
    WHEN method = 'POST' AND path = '/api/v1/northbound/push/targets/:id/circuit/reset' THEN '北向接口 - 重置'
    WHEN method = 'GET' AND path = '/api/v1/northbound/sync/full' THEN '北向接口 - 全量同步'
    WHEN method = 'GET' AND path = '/api/v1/northbound/sync/incremental' THEN '北向接口 - 增量同步'
    WHEN method = 'GET' AND path = '/api/v1/notifications' THEN '通知 - 列表'
    WHEN method = 'DELETE' AND path = '/api/v1/notifications/:id' THEN '通知 - 删除'
    WHEN method = 'PUT' AND path = '/api/v1/notifications/:id/read' THEN '通知 - 标记已读'
    WHEN method = 'GET' AND path = '/api/v1/notifications/history' THEN '通知 - 历史记录'
    WHEN method = 'GET' AND path = '/api/v1/notifications/history/:id' THEN '通知 - 历史记录详情'
    WHEN method = 'PUT' AND path = '/api/v1/notifications/read-all' THEN '通知 - 批量更新'
    WHEN method = 'GET' AND path = '/api/v1/notifications/templates' THEN '通知 - 模板'
    WHEN method = 'POST' AND path = '/api/v1/notifications/templates' THEN '通知 - 模板'
    WHEN method = 'DELETE' AND path = '/api/v1/notifications/templates/:id' THEN '通知 - 模板删除'
    WHEN method = 'GET' AND path = '/api/v1/notifications/templates/:id' THEN '通知 - 模板详情'
    WHEN method = 'PUT' AND path = '/api/v1/notifications/templates/:id' THEN '通知 - 模板更新'
    WHEN method = 'GET' AND path = '/api/v1/notifications/unread-count' THEN '通知 - 列表'
    WHEN method = 'GET' AND path = '/api/v1/ops/command-records' THEN '运维任务 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/ops/command-records' THEN '运维任务 - 创建'
    WHEN method = 'GET' AND path = '/api/v1/ops/tasks' THEN '运维任务 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/ops/tasks' THEN '运维任务 - 创建'
    WHEN method = 'GET' AND path = '/api/v1/ops/tasks/:id' THEN '运维任务 - 详情'
    WHEN method = 'POST' AND path = '/api/v1/ops/tasks/:id/cancel' THEN '运维任务 - 取消'
    WHEN method = 'POST' AND path = '/api/v1/ops/tasks/:id/pause' THEN '运维任务 - 暂停'
    WHEN method = 'POST' AND path = '/api/v1/ops/tasks/:id/resume' THEN '运维任务 - 恢复'
    WHEN method = 'GET' AND path = '/api/v1/ops/templates' THEN '运维任务 - 模板'
    WHEN method = 'POST' AND path = '/api/v1/ops/templates' THEN '运维任务 - 模板'
    WHEN method = 'DELETE' AND path = '/api/v1/ops/templates/:id' THEN '运维任务 - 模板删除'
    WHEN method = 'GET' AND path = '/api/v1/ops/templates/:id' THEN '运维任务 - 模板详情'
    WHEN method = 'PUT' AND path = '/api/v1/ops/templates/:id' THEN '运维任务 - 模板更新'
    WHEN method = 'GET' AND path = '/api/v1/oui' THEN 'OUI - 列表'
    WHEN method = 'POST' AND path = '/api/v1/oui' THEN 'OUI - 创建'
    WHEN method = 'GET' AND path = '/api/v1/admin/permissions' THEN '权限 - 列表'
    WHEN method = 'GET' AND path = '/api/v1/pm/counters' THEN '性能 PM - 计数器'
    WHEN method = 'GET' AND path = '/api/v1/pm/counters/aggregated' THEN '性能 PM - 聚合'
    WHEN method = 'GET' AND path = '/api/v1/pm/files' THEN '性能 PM - 列表'
    WHEN method = 'GET' AND path = '/api/v1/pm/files/:id/download' THEN '性能 PM - 下载'
    WHEN method = 'POST' AND path = '/api/v1/pm/indicatormg/addIndicatorGroup' THEN '性能 PM - 指标管理新增指标分组'
    WHEN method = 'POST' AND path = '/api/v1/pm/indicatormg/addOrModifyIndicator' THEN '性能 PM - 指标管理新增或修改指标'
    WHEN method = 'POST' AND path = '/api/v1/pm/indicatormg/delIndicator' THEN '性能 PM - 指标管理删除指标'
    WHEN method = 'POST' AND path = '/api/v1/pm/indicatormg/delIndicatorGroup' THEN '性能 PM - 指标管理删除指标分组'
    WHEN method = 'POST' AND path = '/api/v1/pm/indicatormg/exportAllIndicator' THEN '性能 PM - 指标管理导出全部指标'
    WHEN method = 'POST' AND path = '/api/v1/pm/indicatormg/getEffectiveIndicators' THEN '性能 PM - 指标管理查询EffectiveIndicators'
    WHEN method = 'POST' AND path = '/api/v1/pm/indicatormg/getIndicatorGroupInfo' THEN '性能 PM - 指标管理查询指标分组信息'
    WHEN method = 'POST' AND path = '/api/v1/pm/indicatormg/getIndicatorGroupList' THEN '性能 PM - 指标管理查询指标GroupList'
    WHEN method = 'POST' AND path = '/api/v1/pm/indicatormg/getIndicatorGroupTree' THEN '性能 PM - 指标管理查询指标分组树'
    WHEN method = 'POST' AND path = '/api/v1/pm/indicatormg/getIndicatorInfo' THEN '性能 PM - 指标管理查询指标信息'
    WHEN method = 'POST' AND path = '/api/v1/pm/indicatormg/getIndicatorListByPage' THEN '性能 PM - 指标管理查询指标分页列表'
    WHEN method = 'POST' AND path = '/api/v1/pm/indicatormg/getIndicatorTypes' THEN '性能 PM - 指标管理查询指标Types'
    WHEN method = 'GET' AND path = '/api/v1/pm/indicatormg/getIndicatorUnitList' THEN '性能 PM - 指标管理查询指标UnitList'
    WHEN method = 'POST' AND path = '/api/v1/pm/indicatormg/modifyIndicatorGroup' THEN '性能 PM - 指标管理修改指标分组'
    WHEN method = 'POST' AND path = '/api/v1/pm/indicatormg/updateBaseKpiCustName' THEN '性能 PM - 指标管理更新基础 KPI 自定义名'
    WHEN method = 'POST' AND path = '/api/v1/pm/indicatormg/updateEnbIndicatorsName' THEN '性能 PM - 指标管理更新EnbIndicatorsName'
    WHEN method = 'POST' AND path = '/api/v1/pm/indicatormg/updateGsmIndicatorsName' THEN '性能 PM - 指标管理更新GsmIndicatorsName'
    WHEN method = 'GET' AND path = '/api/v1/pm/kpi' THEN '性能 PM - KPI'
    WHEN method = 'POST' AND path = '/api/v1/pm/kpi/calculate' THEN '性能 PM - 计算'
    WHEN method = 'GET' AND path = '/api/v1/pm/kpi/definitions' THEN '性能 PM - 定义'
    WHEN method = 'GET' AND path = '/api/v1/pm/tasks' THEN '性能 PM - 列表'
    WHEN method = 'POST' AND path = '/api/v1/pm/tasks' THEN '性能 PM - 创建'
    WHEN method = 'GET' AND path = '/api/v1/pm/thresholds' THEN '性能 PM - 阈值'
    WHEN method = 'POST' AND path = '/api/v1/pm/thresholds' THEN '性能 PM - 阈值'
    WHEN method = 'DELETE' AND path = '/api/v1/pm/thresholds/:id' THEN '性能 PM - 阈值删除'
    WHEN method = 'GET' AND path = '/api/v1/pm/thresholds/:id' THEN '性能 PM - 阈值详情'
    WHEN method = 'PUT' AND path = '/api/v1/pm/thresholds/:id' THEN '性能 PM - 阈值更新'
    WHEN method = 'GET' AND path = '/api/v1/provisioning/tasks' THEN '自动开站 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/provisioning/tasks' THEN '自动开站 - 创建'
    WHEN method = 'GET' AND path = '/api/v1/provisioning/tasks/:id' THEN '自动开站 - 详情'
    WHEN method = 'POST' AND path = '/api/v1/provisioning/tasks/:id/retry' THEN '自动开站 - 重试'
    WHEN method = 'GET' AND path = '/readyz' THEN '就绪检查'
    WHEN method = 'GET' AND path = '/api/v1/reports/definitions' THEN '报表 - 定义'
    WHEN method = 'POST' AND path = '/api/v1/reports/definitions' THEN '报表 - 定义'
    WHEN method = 'DELETE' AND path = '/api/v1/reports/definitions/:id' THEN '报表 - 定义删除'
    WHEN method = 'GET' AND path = '/api/v1/reports/definitions/:id' THEN '报表 - 定义详情'
    WHEN method = 'PUT' AND path = '/api/v1/reports/definitions/:id' THEN '报表 - 定义更新'
    WHEN method = 'POST' AND path = '/api/v1/reports/generate' THEN '报表 - 创建'
    WHEN method = 'GET' AND path = '/api/v1/reports/records' THEN '报表 - 记录'
    WHEN method = 'GET' AND path = '/api/v1/reports/records/:id/download' THEN '报表 - 下载'
    WHEN method = 'GET' AND path = '/api/v1/reports/sample-data' THEN '报表 - 列表'
    WHEN method = 'GET' AND path = '/api/v1/admin/roles' THEN '角色 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/admin/roles' THEN '角色 - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/admin/roles/:id' THEN '角色 - 删除'
    WHEN method = 'GET' AND path = '/api/v1/admin/roles/:id' THEN '角色 - 详情'
    WHEN method = 'PUT' AND path = '/api/v1/admin/roles/:id' THEN '角色 - 更新'
    WHEN method = 'GET' AND path = '/api/v1/admin/roles/:id/api-permissions' THEN '角色 - api-permissions'
    WHEN method = 'PUT' AND path = '/api/v1/admin/roles/:id/api-permissions' THEN '角色 - api-permissions'
    WHEN method = 'POST' AND path = '/api/v1/admin/roles/:id/copy' THEN '角色 - 复制'
    WHEN method = 'GET' AND path = '/api/v1/admin/roles/:id/device-groups' THEN '角色 - device-groups'
    WHEN method = 'PUT' AND path = '/api/v1/admin/roles/:id/device-groups' THEN '角色 - device-groups'
    WHEN method = 'GET' AND path = '/api/v1/admin/roles/:id/menus' THEN '角色 - menus'
    WHEN method = 'PUT' AND path = '/api/v1/admin/roles/:id/menus' THEN '角色 - menus'
    WHEN method = 'GET' AND path = '/api/v1/admin/roles/:id/users' THEN '角色 - users'
    WHEN method = 'GET' AND path = '/api/v1/admin/roles/all' THEN '角色 - 全量'
    WHEN method = 'GET' AND path = '/api/v1/sites' THEN '站点 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/sites' THEN '站点 - 创建'
    WHEN method = 'GET' AND path = '/api/v1/sites/:id' THEN '站点 - 详情'
    WHEN method = 'GET' AND path = '/api/v1/admin/sysConfig' THEN '系统配置 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/admin/sysConfig' THEN '系统配置 - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/admin/sysConfig/:id' THEN '系统配置 - 删除'
    WHEN method = 'GET' AND path = '/api/v1/admin/sysConfig/:id' THEN '系统配置 - 详情'
    WHEN method = 'PUT' AND path = '/api/v1/admin/sysConfig/:id' THEN '系统配置 - 更新'
    WHEN method = 'POST' AND path = '/api/v1/admin/sysDictionary/createSysDictionary' THEN '字典 - 创建字典'
    WHEN method = 'DELETE' AND path = '/api/v1/admin/sysDictionary/deleteSysDictionary' THEN '字典 - 删除字典'
    WHEN method = 'GET' AND path = '/api/v1/admin/sysDictionary/findSysDictionary' THEN '字典 - 查询字典'
    WHEN method = 'GET' AND path = '/api/v1/admin/sysDictionary/getSysDictionaryList' THEN '字典 - 查询SysDictionaryList'
    WHEN method = 'PUT' AND path = '/api/v1/admin/sysDictionary/updateSysDictionary' THEN '字典 - 更新字典'
    WHEN method = 'POST' AND path = '/api/v1/admin/sysDictionaryDetail/createSysDictionaryDetail' THEN '字典明细 - 创建字典明细'
    WHEN method = 'DELETE' AND path = '/api/v1/admin/sysDictionaryDetail/deleteSysDictionaryDetail' THEN '字典明细 - 删除字典明细'
    WHEN method = 'GET' AND path = '/api/v1/admin/sysDictionaryDetail/findSysDictionaryDetail' THEN '字典明细 - 查询字典明细'
    WHEN method = 'GET' AND path = '/api/v1/admin/sysDictionaryDetail/getSysDictionaryDetailList' THEN '字典明细 - 查询SysDictionaryDetailList'
    WHEN method = 'PUT' AND path = '/api/v1/admin/sysDictionaryDetail/updateSysDictionaryDetail' THEN '字典明细 - 更新字典明细'
    WHEN method = 'GET' AND path = '/api/v1/system/info' THEN '系统运行信息'
    WHEN method = 'POST' AND path = '/api/v1/tasks/purge' THEN '任务 - 清理'
    WHEN method = 'GET' AND path = '/api/v1/templates' THEN '模板 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/templates' THEN '模板 - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/templates/:id' THEN '模板 - 删除'
    WHEN method = 'GET' AND path = '/api/v1/templates/:id' THEN '模板 - 详情'
    WHEN method = 'PUT' AND path = '/api/v1/templates/:id' THEN '模板 - 更新'
    WHEN method = 'GET' AND path = '/api/v1/topology/edges' THEN '拓扑 - 列表'
    WHEN method = 'GET' AND path = '/api/v1/topology/geo' THEN '拓扑 - 地理位置数据'
    WHEN method = 'GET' AND path = '/api/v1/topology/graph' THEN '拓扑 - 列表'
    WHEN method = 'GET' AND path = '/api/v1/topology/nodes' THEN '拓扑 - 列表'
    WHEN method = 'GET' AND path = '/api/v1/upgrade-sub-tasks' THEN '升级子任务 - 列表'
    WHEN method = 'GET' AND path = '/api/v1/upgrade-sub-tasks/:id' THEN '升级子任务 - 详情'
    WHEN method = 'GET' AND path = '/api/v1/upgrade-tasks' THEN '升级任务 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/upgrade-tasks' THEN '升级任务 - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/upgrade-tasks/:id' THEN '升级任务 - 删除'
    WHEN method = 'GET' AND path = '/api/v1/upgrade-tasks/:id' THEN '升级任务 - 详情'
    WHEN method = 'POST' AND path = '/api/v1/upgrade-tasks/:id/abort-canary' THEN '升级任务 - 中止灰度'
    WHEN method = 'POST' AND path = '/api/v1/upgrade-tasks/:id/advance' THEN '升级任务 - 推进灰度阶段'
    WHEN method = 'POST' AND path = '/api/v1/upgrade-tasks/:id/pause-canary' THEN '升级任务 - 暂停灰度'
    WHEN method = 'PUT' AND path = '/api/v1/upgrade-tasks/:id/resume' THEN '升级任务 - 恢复'
    WHEN method = 'POST' AND path = '/api/v1/upgrade-tasks/:id/resume-canary' THEN '升级任务 - 恢复灰度'
    WHEN method = 'POST' AND path = '/api/v1/upgrade-tasks/:id/retry' THEN '升级任务 - 重试'
    WHEN method = 'PUT' AND path = '/api/v1/upgrade-tasks/:id/suspend' THEN '升级任务 - 暂停'
    WHEN method = 'GET' AND path = '/api/v1/upgrade-tasks/:id/tasks' THEN '升级任务 - tasks'
    WHEN method = 'PUT' AND path = '/api/v1/upgrade-tasks/:id/terminate' THEN '升级任务 - 终止'
    WHEN method = 'POST' AND path = '/api/v1/upgrade-tasks/rollback' THEN '升级任务 - 回滚'
    WHEN method = 'GET' AND path = '/api/v1/admin/users' THEN '用户 - 列表'
    WHEN method = 'POST' AND path = '/api/v1/admin/users' THEN '用户 - 创建'
    WHEN method = 'DELETE' AND path = '/api/v1/admin/users/:id' THEN '用户 - 删除'
    WHEN method = 'GET' AND path = '/api/v1/admin/users/:id' THEN '用户 - 详情'
    WHEN method = 'PUT' AND path = '/api/v1/admin/users/:id' THEN '用户 - 更新'
    WHEN method = 'POST' AND path = '/api/v1/admin/users/:id/copy' THEN '用户 - 复制'
    WHEN method = 'POST' AND path = '/api/v1/admin/users/:id/lock' THEN '用户 - 锁定'
    WHEN method = 'POST' AND path = '/api/v1/admin/users/:id/reset-password' THEN '用户 - 重置密码'
    WHEN method = 'POST' AND path = '/api/v1/admin/users/:id/roles' THEN '用户 - roles'
    WHEN method = 'DELETE' AND path = '/api/v1/admin/users/:id/roles/:roleId' THEN '用户 - 删除'
    WHEN method = 'POST' AND path = '/api/v1/admin/users/:id/unlock' THEN '用户 - 解锁'
    WHEN method = 'POST' AND path = '/api/v1/admin/users/assign-roles' THEN '用户 - 创建'
    WHEN method = 'POST' AND path = '/api/v1/admin/users/force-logout' THEN '用户 - 强制下线'
    WHEN method = 'POST' AND path = '/api/v1/admin/users/import' THEN '用户 - 创建'
    WHEN method = 'GET' AND path = '/api/v1/admin/users/import/template' THEN '用户 - 模板'
    ELSE description
END WHERE COALESCE(description, '') = '';

-- +goose Down
-- 回滚：仅清空本次脚本覆盖过的 path+method 行（避免影响手工记录）。
UPDATE api_endpoints SET description = '' WHERE (method, path) IN (
    ('GET', '/api/v1/alarms/:id'),
    ('POST', '/api/v1/alarms/:id/acknowledge'),
    ('POST', '/api/v1/alarms/:id/clear'),
    ('GET', '/api/v1/alarms/active'),
    ('POST', '/api/v1/alarms/active/:id/read'),
    ('POST', '/api/v1/alarms/active/batch/acknowledge'),
    ('POST', '/api/v1/alarms/active/batch/clear'),
    ('POST', '/api/v1/alarms/active/batch/unacknowledge'),
    ('GET', '/api/v1/alarms/alarm-filters'),
    ('POST', '/api/v1/alarms/alarm-filters'),
    ('DELETE', '/api/v1/alarms/alarm-filters/:id'),
    ('GET', '/api/v1/alarms/alarm-filters/:id'),
    ('PUT', '/api/v1/alarms/alarm-filters/:id'),
    ('POST', '/api/v1/alarms/alarm-filters/:id/toggle'),
    ('GET', '/api/v1/alarms/alarm-libraries'),
    ('POST', '/api/v1/alarms/alarm-libraries'),
    ('DELETE', '/api/v1/alarms/alarm-libraries/:id'),
    ('GET', '/api/v1/alarms/alarm-libraries/:id'),
    ('PUT', '/api/v1/alarms/alarm-libraries/:id'),
    ('GET', '/api/v1/alarms/alarm-libraries/:id/i18n'),
    ('POST', '/api/v1/alarms/alarm-libraries/:id/i18n'),
    ('DELETE', '/api/v1/alarms/alarm-libraries/:id/i18n/:i18nId'),
    ('GET', '/api/v1/alarms/history'),
    ('POST', '/api/v1/alarms/history/batch/acknowledge'),
    ('POST', '/api/v1/alarms/history/batch/delete'),
    ('POST', '/api/v1/alarms/history/batch/unacknowledge'),
    ('GET', '/api/v1/alarms/history/statistics'),
    ('GET', '/api/v1/alarms/statistics'),
    ('POST', '/api/v1/alarms/sync/:device_sn'),
    ('GET', '/api/v1/admin/api-endpoints'),
    ('POST', '/api/v1/admin/api-endpoints'),
    ('DELETE', '/api/v1/admin/api-endpoints/:id'),
    ('PUT', '/api/v1/admin/api-endpoints/:id'),
    ('DELETE', '/api/v1/admin/api-endpoints/batch'),
    ('GET', '/api/v1/admin/api-endpoints/groups'),
    ('POST', '/api/v1/admin/api-endpoints/sync'),
    ('GET', '/api/v1/api-keys'),
    ('POST', '/api/v1/api-keys'),
    ('DELETE', '/api/v1/api-keys/:id'),
    ('GET', '/api/v1/admin/audit-logs'),
    ('GET', '/api/v1/auth/captcha'),
    ('POST', '/api/v1/auth/change-password'),
    ('POST', '/api/v1/auth/login'),
    ('GET', '/api/v1/auth/me'),
    ('GET', '/api/v1/auth/menus'),
    ('POST', '/api/v1/auth/refresh'),
    ('POST', '/api/v1/auth/switch-role'),
    ('GET', '/api/v1/backup/ftp-configs'),
    ('POST', '/api/v1/backup/ftp-configs'),
    ('DELETE', '/api/v1/backup/ftp-configs/:id'),
    ('PUT', '/api/v1/backup/ftp-configs/:id'),
    ('POST', '/api/v1/backup/ftp-configs/:id/test'),
    ('GET', '/api/v1/backup/policy'),
    ('PUT', '/api/v1/backup/policy'),
    ('POST', '/api/v1/backup/restore'),
    ('GET', '/api/v1/backup/restore-tasks'),
    ('GET', '/api/v1/backup/restore-tasks/:id'),
    ('POST', '/api/v1/backup/restore/by-task-id'),
    ('GET', '/api/v1/backup/schedules'),
    ('POST', '/api/v1/backup/schedules'),
    ('DELETE', '/api/v1/backup/schedules/:id'),
    ('PUT', '/api/v1/backup/schedules/:id'),
    ('GET', '/api/v1/backup/tasks'),
    ('POST', '/api/v1/backup/tasks'),
    ('DELETE', '/api/v1/backup/tasks/:id'),
    ('GET', '/api/v1/backup/tasks/:id'),
    ('POST', '/api/v1/backup/tasks/:id/cancel'),
    ('POST', '/api/v1/cell/perfmgmt/kpimanage/disableIndicator'),
    ('POST', '/api/v1/cell/perfmgmt/kpimanage/enableIndicator'),
    ('GET', '/api/v1/cell/perfmgmt/kpimanage/isIndicatorInTemplate'),
    ('GET', '/api/v1/column-configs/:pageKey'),
    ('PUT', '/api/v1/column-configs/:pageKey'),
    ('GET', '/api/v1/config/baselines'),
    ('POST', '/api/v1/config/baselines'),
    ('DELETE', '/api/v1/config/baselines/:id'),
    ('GET', '/api/v1/config/baselines/:id'),
    ('PUT', '/api/v1/config/baselines/:id'),
    ('GET', '/api/v1/config/neighbors'),
    ('POST', '/api/v1/config/sync/pull/:deviceId'),
    ('POST', '/api/v1/config/sync/push/:deviceId'),
    ('GET', '/api/v1/config/sync/status/:deviceId'),
    ('GET', '/api/v1/config/tasks'),
    ('POST', '/api/v1/config/tasks'),
    ('GET', '/api/v1/dashboard/alarm-trend'),
    ('GET', '/api/v1/dashboard/alarm-type-pie'),
    ('GET', '/api/v1/dashboard/device-status'),
    ('GET', '/api/v1/dashboard/kpi-time-series'),
    ('GET', '/api/v1/dashboard/kpi-trend'),
    ('GET', '/api/v1/dashboard/region-stats'),
    ('GET', '/api/v1/dashboard/summary'),
    ('GET', '/api/v1/dashboard/widgets'),
    ('PUT', '/api/v1/dashboard/widgets'),
    ('GET', '/api/v1/datamodels'),
    ('POST', '/api/v1/datamodels'),
    ('DELETE', '/api/v1/datamodels/:id'),
    ('GET', '/api/v1/datamodels/:id'),
    ('PUT', '/api/v1/datamodels/:id'),
    ('POST', '/api/v1/datamodels/:id/activate'),
    ('POST', '/api/v1/datamodels/:id/deprecate'),
    ('GET', '/api/v1/datamodels/:id/export'),
    ('POST', '/api/v1/datamodels/cache/refresh'),
    ('POST', '/api/v1/datamodels/import'),
    ('POST', '/api/v1/datamodels/import-xml'),
    ('GET', '/api/v1/datamodels/resolve'),
    ('GET', '/api/v1/datamodels/statistics'),
    ('GET', '/api/v1/admin/dead-letters'),
    ('DELETE', '/api/v1/admin/dead-letters/:id'),
    ('GET', '/api/v1/admin/dead-letters/:id'),
    ('POST', '/api/v1/admin/dead-letters/:id/replay'),
    ('POST', '/api/v1/device-groups'),
    ('DELETE', '/api/v1/device-groups/:id'),
    ('GET', '/api/v1/device-groups/:id'),
    ('PUT', '/api/v1/device-groups/:id'),
    ('GET', '/api/v1/device-groups/:id/check-delete'),
    ('DELETE', '/api/v1/device-groups/:id/devices'),
    ('GET', '/api/v1/device-groups/:id/devices'),
    ('POST', '/api/v1/device-groups/:id/devices'),
    ('POST', '/api/v1/device-groups/move-devices'),
    ('PUT', '/api/v1/device-groups/sort'),
    ('GET', '/api/v1/device-groups/stats'),
    ('GET', '/api/v1/device-groups/tree'),
    ('GET', '/api/v1/device-registrations'),
    ('POST', '/api/v1/device-registrations'),
    ('DELETE', '/api/v1/device-registrations/:id'),
    ('GET', '/api/v1/device-rules'),
    ('POST', '/api/v1/device-rules'),
    ('DELETE', '/api/v1/device-rules/:id'),
    ('GET', '/api/v1/device-rules/:id'),
    ('PUT', '/api/v1/device-rules/:id'),
    ('POST', '/api/v1/device-rules/:id/apply'),
    ('GET', '/api/v1/device-rules/:id/tasks'),
    ('GET', '/api/v1/device-rules/:id/tasks/:taskId'),
    ('PATCH', '/api/v1/device-rules/:id/toggle'),
    ('PUT', '/api/v1/device-rules/batch-sort'),
    ('GET', '/api/v1/device-rules/next-priority'),
    ('GET', '/api/v1/devices'),
    ('POST', '/api/v1/devices'),
    ('DELETE', '/api/v1/devices/:id'),
    ('GET', '/api/v1/devices/:id'),
    ('PUT', '/api/v1/devices/:id'),
    ('PUT', '/api/v1/devices/:id/activate'),
    ('POST', '/api/v1/devices/:id/config-file/sync'),
    ('PUT', '/api/v1/devices/:id/deactivate'),
    ('GET', '/api/v1/devices/:id/detail'),
    ('GET', '/api/v1/devices/:id/info'),
    ('PUT', '/api/v1/devices/:id/info'),
    ('POST', '/api/v1/devices/:id/objects/add'),
    ('POST', '/api/v1/devices/:id/objects/delete'),
    ('POST', '/api/v1/devices/:id/param-sync'),
    ('GET', '/api/v1/devices/:id/parameters'),
    ('PUT', '/api/v1/devices/:id/parameters'),
    ('GET', '/api/v1/devices/:id/parameters/children'),
    ('POST', '/api/v1/devices/:id/parameters/discover'),
    ('GET', '/api/v1/devices/:id/parameters/schema'),
    ('GET', '/api/v1/devices/:id/parameters/search'),
    ('POST', '/api/v1/devices/:id/parameters/sync'),
    ('GET', '/api/v1/devices/:id/parameters/sync-status'),
    ('GET', '/api/v1/devices/:id/parameters/tree'),
    ('POST', '/api/v1/devices/:id/reboot'),
    ('PUT', '/api/v1/devices/:id/rf-switch'),
    ('DELETE', '/api/v1/devices/batch'),
    ('POST', '/api/v1/devices/batch-reboot'),
    ('GET', '/api/v1/devices/enums'),
    ('GET', '/api/v1/devices/export'),
    ('GET', '/api/v1/devices/geo'),
    ('GET', '/api/v1/devices/geo/stats'),
    ('GET', '/api/v1/devices/product-classes'),
    ('GET', '/api/v1/devices/recycle'),
    ('DELETE', '/api/v1/devices/recycle/permanent'),
    ('PATCH', '/api/v1/devices/recycle/restore'),
    ('GET', '/api/v1/devices/search'),
    ('GET', '/api/v1/devices/stats'),
    ('GET', '/api/v1/devices/tasks'),
    ('POST', '/api/v1/devices/tasks'),
    ('DELETE', '/api/v1/devices/tasks/:task_id'),
    ('GET', '/api/v1/devices/tasks/:task_id'),
    ('POST', '/api/v1/devices/tasks/:task_id/retry'),
    ('POST', '/api/v1/devices/tasks/batch'),
    ('GET', '/api/v1/devices/tasks/pending'),
    ('GET', '/api/v1/devices/tasks/stats'),
    ('GET', '/api/v1/events/stream'),
    ('GET', '/api/v1/files'),
    ('POST', '/api/v1/files'),
    ('DELETE', '/api/v1/files/:id'),
    ('GET', '/api/v1/files/:id'),
    ('POST', '/api/v1/files/:id/distribute'),
    ('GET', '/api/v1/files/:id/download'),
    ('GET', '/api/v1/firmware'),
    ('POST', '/api/v1/firmware'),
    ('DELETE', '/api/v1/firmware/:id'),
    ('GET', '/api/v1/firmware/:id'),
    ('PUT', '/api/v1/firmware/:id'),
    ('GET', '/api/v1/firmware/:id/download'),
    ('PUT', '/api/v1/firmware/:id/recommend'),
    ('POST', '/api/v1/gnb/pm/indicatormg/addIndicatorGroup'),
    ('POST', '/api/v1/gnb/pm/indicatormg/addOrModifyIndicator'),
    ('POST', '/api/v1/gnb/pm/indicatormg/delIndicator'),
    ('POST', '/api/v1/gnb/pm/indicatormg/delIndicatorGroup'),
    ('POST', '/api/v1/gnb/pm/indicatormg/exportAllIndicator'),
    ('POST', '/api/v1/gnb/pm/indicatormg/getIndicatorGroupInfo'),
    ('POST', '/api/v1/gnb/pm/indicatormg/getIndicatorGroupTree'),
    ('POST', '/api/v1/gnb/pm/indicatormg/getIndicatorInfo'),
    ('POST', '/api/v1/gnb/pm/indicatormg/getIndicatorListByPage'),
    ('POST', '/api/v1/gnb/pm/indicatormg/modifyIndicatorGroup'),
    ('POST', '/api/v1/gnb/pm/indicatormg/updateBaseKpiCustName'),
    ('POST', '/api/v1/gnb/pm/indicatormg/updateGnbIndicatorsName'),
    ('GET', '/api/v1/admin/groups'),
    ('GET', '/api/v1/admin/groups/:id'),
    ('GET', '/api/v1/groups'),
    ('POST', '/api/v1/groups'),
    ('DELETE', '/api/v1/groups/:id'),
    ('GET', '/api/v1/groups/:id'),
    ('PUT', '/api/v1/groups/:id'),
    ('GET', '/api/v1/groups/:id/devices'),
    ('POST', '/api/v1/groups/:id/devices'),
    ('DELETE', '/api/v1/groups/:id/devices/:deviceId'),
    ('GET', '/healthz'),
    ('POST', '/api/v1/interop/run'),
    ('POST', '/api/v1/interop/run/:category'),
    ('GET', '/api/v1/interop/test-cases'),
    ('POST', '/api/v1/interop/validate/:deviceId'),
    ('GET', '/api/v1/licenses'),
    ('GET', '/api/v1/licenses/:id'),
    ('POST', '/api/v1/licenses/:id/revoke'),
    ('POST', '/api/v1/licenses/activate'),
    ('POST', '/api/v1/licenses/import'),
    ('GET', '/api/v1/licenses/quota'),
    ('GET', '/api/v1/licenses/summary'),
    ('GET', '/api/v1/admin/logs/login'),
    ('GET', '/api/v1/admin/logs/operation'),
    ('GET', '/api/v1/admin/logs/task'),
    ('GET', '/api/v1/logs/ne-messages'),
    ('GET', '/api/v1/logs/system'),
    ('DELETE', '/api/v1/admin/menus'),
    ('GET', '/api/v1/admin/menus'),
    ('POST', '/api/v1/admin/menus'),
    ('GET', '/api/v1/admin/menus/:id'),
    ('PUT', '/api/v1/admin/menus/:id'),
    ('GET', '/api/v1/admin/menus/tree'),
    ('GET', '/api/v1/admin/menus/user-tree'),
    ('GET', '/api/v1/mml/commands'),
    ('GET', '/api/v1/mml/commands/:id'),
    ('GET', '/api/v1/mml/commands/:id/param-paths'),
    ('GET', '/api/v1/mml/dangerous-check'),
    ('POST', '/api/v1/mml/execute'),
    ('GET', '/api/v1/mml/param-versions'),
    ('GET', '/api/v1/mml/param-versions/:version/groups'),
    ('GET', '/api/v1/mml/param-versions/:version/groups/:groupId/params'),
    ('GET', '/api/v1/mml/param-versions/:version/params'),
    ('GET', '/api/v1/mml/scripts'),
    ('POST', '/api/v1/mml/scripts'),
    ('DELETE', '/api/v1/mml/scripts/:id'),
    ('GET', '/api/v1/mml/scripts/:id'),
    ('PUT', '/api/v1/mml/scripts/:id'),
    ('POST', '/api/v1/mml/scripts/:id/cancel'),
    ('POST', '/api/v1/mml/scripts/:id/pause'),
    ('GET', '/api/v1/mml/scripts/:id/runs'),
    ('POST', '/api/v1/mml/scripts/:id/start'),
    ('GET', '/api/v1/mml/tasks'),
    ('POST', '/api/v1/mml/tasks'),
    ('DELETE', '/api/v1/mml/tasks/:id'),
    ('GET', '/api/v1/mml/tasks/:id'),
    ('POST', '/api/v1/mml/tasks/:id/cancel'),
    ('POST', '/api/v1/mml/tasks/:id/pause'),
    ('GET', '/api/v1/mml/tasks/:id/results'),
    ('POST', '/api/v1/mml/tasks/:id/start'),
    ('GET', '/api/v1/mml/templates'),
    ('POST', '/api/v1/mml/templates'),
    ('DELETE', '/api/v1/mml/templates/:id'),
    ('GET', '/api/v1/mml/templates/:id'),
    ('PUT', '/api/v1/mml/templates/:id'),
    ('POST', '/api/v1/mml/templates/:id/clone'),
    ('GET', '/api/v1/mr/data'),
    ('POST', '/api/v1/mr/export'),
    ('GET', '/api/v1/mr/files'),
    ('GET', '/api/v1/mr/files/:id/download'),
    ('GET', '/api/v1/mr/indicators'),
    ('GET', '/api/v1/mr/indicators/:code/stats'),
    ('GET', '/api/v1/mr/indicators/all'),
    ('GET', '/api/v1/mr/mappings'),
    ('PUT', '/api/v1/mr/mappings/:id'),
    ('PUT', '/api/v1/mr/mappings/:id/toggle'),
    ('POST', '/api/v1/northbound/export/alarms'),
    ('GET', '/api/v1/northbound/export/config/:deviceId'),
    ('POST', '/api/v1/northbound/export/pm'),
    ('GET', '/api/v1/northbound/push/deadletter'),
    ('POST', '/api/v1/northbound/push/deadletter/:id/replay'),
    ('GET', '/api/v1/northbound/push/targets'),
    ('POST', '/api/v1/northbound/push/targets'),
    ('DELETE', '/api/v1/northbound/push/targets/:id'),
    ('GET', '/api/v1/northbound/push/targets/:id/circuit'),
    ('POST', '/api/v1/northbound/push/targets/:id/circuit/reset'),
    ('GET', '/api/v1/northbound/sync/full'),
    ('GET', '/api/v1/northbound/sync/incremental'),
    ('GET', '/api/v1/notifications'),
    ('DELETE', '/api/v1/notifications/:id'),
    ('PUT', '/api/v1/notifications/:id/read'),
    ('GET', '/api/v1/notifications/history'),
    ('GET', '/api/v1/notifications/history/:id'),
    ('PUT', '/api/v1/notifications/read-all'),
    ('GET', '/api/v1/notifications/templates'),
    ('POST', '/api/v1/notifications/templates'),
    ('DELETE', '/api/v1/notifications/templates/:id'),
    ('GET', '/api/v1/notifications/templates/:id'),
    ('PUT', '/api/v1/notifications/templates/:id'),
    ('GET', '/api/v1/notifications/unread-count'),
    ('GET', '/api/v1/ops/command-records'),
    ('POST', '/api/v1/ops/command-records'),
    ('GET', '/api/v1/ops/tasks'),
    ('POST', '/api/v1/ops/tasks'),
    ('GET', '/api/v1/ops/tasks/:id'),
    ('POST', '/api/v1/ops/tasks/:id/cancel'),
    ('POST', '/api/v1/ops/tasks/:id/pause'),
    ('POST', '/api/v1/ops/tasks/:id/resume'),
    ('GET', '/api/v1/ops/templates'),
    ('POST', '/api/v1/ops/templates'),
    ('DELETE', '/api/v1/ops/templates/:id'),
    ('GET', '/api/v1/ops/templates/:id'),
    ('PUT', '/api/v1/ops/templates/:id'),
    ('GET', '/api/v1/oui'),
    ('POST', '/api/v1/oui'),
    ('GET', '/api/v1/admin/permissions'),
    ('GET', '/api/v1/pm/counters'),
    ('GET', '/api/v1/pm/counters/aggregated'),
    ('GET', '/api/v1/pm/files'),
    ('GET', '/api/v1/pm/files/:id/download'),
    ('POST', '/api/v1/pm/indicatormg/addIndicatorGroup'),
    ('POST', '/api/v1/pm/indicatormg/addOrModifyIndicator'),
    ('POST', '/api/v1/pm/indicatormg/delIndicator'),
    ('POST', '/api/v1/pm/indicatormg/delIndicatorGroup'),
    ('POST', '/api/v1/pm/indicatormg/exportAllIndicator'),
    ('POST', '/api/v1/pm/indicatormg/getEffectiveIndicators'),
    ('POST', '/api/v1/pm/indicatormg/getIndicatorGroupInfo'),
    ('POST', '/api/v1/pm/indicatormg/getIndicatorGroupList'),
    ('POST', '/api/v1/pm/indicatormg/getIndicatorGroupTree'),
    ('POST', '/api/v1/pm/indicatormg/getIndicatorInfo'),
    ('POST', '/api/v1/pm/indicatormg/getIndicatorListByPage'),
    ('POST', '/api/v1/pm/indicatormg/getIndicatorTypes'),
    ('GET', '/api/v1/pm/indicatormg/getIndicatorUnitList'),
    ('POST', '/api/v1/pm/indicatormg/modifyIndicatorGroup'),
    ('POST', '/api/v1/pm/indicatormg/updateBaseKpiCustName'),
    ('POST', '/api/v1/pm/indicatormg/updateEnbIndicatorsName'),
    ('POST', '/api/v1/pm/indicatormg/updateGsmIndicatorsName'),
    ('GET', '/api/v1/pm/kpi'),
    ('POST', '/api/v1/pm/kpi/calculate'),
    ('GET', '/api/v1/pm/kpi/definitions'),
    ('GET', '/api/v1/pm/tasks'),
    ('POST', '/api/v1/pm/tasks'),
    ('GET', '/api/v1/pm/thresholds'),
    ('POST', '/api/v1/pm/thresholds'),
    ('DELETE', '/api/v1/pm/thresholds/:id'),
    ('GET', '/api/v1/pm/thresholds/:id'),
    ('PUT', '/api/v1/pm/thresholds/:id'),
    ('GET', '/api/v1/provisioning/tasks'),
    ('POST', '/api/v1/provisioning/tasks'),
    ('GET', '/api/v1/provisioning/tasks/:id'),
    ('POST', '/api/v1/provisioning/tasks/:id/retry'),
    ('GET', '/readyz'),
    ('GET', '/api/v1/reports/definitions'),
    ('POST', '/api/v1/reports/definitions'),
    ('DELETE', '/api/v1/reports/definitions/:id'),
    ('GET', '/api/v1/reports/definitions/:id'),
    ('PUT', '/api/v1/reports/definitions/:id'),
    ('POST', '/api/v1/reports/generate'),
    ('GET', '/api/v1/reports/records'),
    ('GET', '/api/v1/reports/records/:id/download'),
    ('GET', '/api/v1/reports/sample-data'),
    ('GET', '/api/v1/admin/roles'),
    ('POST', '/api/v1/admin/roles'),
    ('DELETE', '/api/v1/admin/roles/:id'),
    ('GET', '/api/v1/admin/roles/:id'),
    ('PUT', '/api/v1/admin/roles/:id'),
    ('GET', '/api/v1/admin/roles/:id/api-permissions'),
    ('PUT', '/api/v1/admin/roles/:id/api-permissions'),
    ('POST', '/api/v1/admin/roles/:id/copy'),
    ('GET', '/api/v1/admin/roles/:id/device-groups'),
    ('PUT', '/api/v1/admin/roles/:id/device-groups'),
    ('GET', '/api/v1/admin/roles/:id/menus'),
    ('PUT', '/api/v1/admin/roles/:id/menus'),
    ('GET', '/api/v1/admin/roles/:id/users'),
    ('GET', '/api/v1/admin/roles/all'),
    ('GET', '/api/v1/sites'),
    ('POST', '/api/v1/sites'),
    ('GET', '/api/v1/sites/:id'),
    ('GET', '/api/v1/admin/sysConfig'),
    ('POST', '/api/v1/admin/sysConfig'),
    ('DELETE', '/api/v1/admin/sysConfig/:id'),
    ('GET', '/api/v1/admin/sysConfig/:id'),
    ('PUT', '/api/v1/admin/sysConfig/:id'),
    ('POST', '/api/v1/admin/sysDictionary/createSysDictionary'),
    ('DELETE', '/api/v1/admin/sysDictionary/deleteSysDictionary'),
    ('GET', '/api/v1/admin/sysDictionary/findSysDictionary'),
    ('GET', '/api/v1/admin/sysDictionary/getSysDictionaryList'),
    ('PUT', '/api/v1/admin/sysDictionary/updateSysDictionary'),
    ('POST', '/api/v1/admin/sysDictionaryDetail/createSysDictionaryDetail'),
    ('DELETE', '/api/v1/admin/sysDictionaryDetail/deleteSysDictionaryDetail'),
    ('GET', '/api/v1/admin/sysDictionaryDetail/findSysDictionaryDetail'),
    ('GET', '/api/v1/admin/sysDictionaryDetail/getSysDictionaryDetailList'),
    ('PUT', '/api/v1/admin/sysDictionaryDetail/updateSysDictionaryDetail'),
    ('GET', '/api/v1/system/info'),
    ('POST', '/api/v1/tasks/purge'),
    ('GET', '/api/v1/templates'),
    ('POST', '/api/v1/templates'),
    ('DELETE', '/api/v1/templates/:id'),
    ('GET', '/api/v1/templates/:id'),
    ('PUT', '/api/v1/templates/:id'),
    ('GET', '/api/v1/topology/edges'),
    ('GET', '/api/v1/topology/geo'),
    ('GET', '/api/v1/topology/graph'),
    ('GET', '/api/v1/topology/nodes'),
    ('GET', '/api/v1/upgrade-sub-tasks'),
    ('GET', '/api/v1/upgrade-sub-tasks/:id'),
    ('GET', '/api/v1/upgrade-tasks'),
    ('POST', '/api/v1/upgrade-tasks'),
    ('DELETE', '/api/v1/upgrade-tasks/:id'),
    ('GET', '/api/v1/upgrade-tasks/:id'),
    ('POST', '/api/v1/upgrade-tasks/:id/abort-canary'),
    ('POST', '/api/v1/upgrade-tasks/:id/advance'),
    ('POST', '/api/v1/upgrade-tasks/:id/pause-canary'),
    ('PUT', '/api/v1/upgrade-tasks/:id/resume'),
    ('POST', '/api/v1/upgrade-tasks/:id/resume-canary'),
    ('POST', '/api/v1/upgrade-tasks/:id/retry'),
    ('PUT', '/api/v1/upgrade-tasks/:id/suspend'),
    ('GET', '/api/v1/upgrade-tasks/:id/tasks'),
    ('PUT', '/api/v1/upgrade-tasks/:id/terminate'),
    ('POST', '/api/v1/upgrade-tasks/rollback'),
    ('GET', '/api/v1/admin/users'),
    ('POST', '/api/v1/admin/users'),
    ('DELETE', '/api/v1/admin/users/:id'),
    ('GET', '/api/v1/admin/users/:id'),
    ('PUT', '/api/v1/admin/users/:id'),
    ('POST', '/api/v1/admin/users/:id/copy'),
    ('POST', '/api/v1/admin/users/:id/lock'),
    ('POST', '/api/v1/admin/users/:id/reset-password'),
    ('POST', '/api/v1/admin/users/:id/roles'),
    ('DELETE', '/api/v1/admin/users/:id/roles/:roleId'),
    ('POST', '/api/v1/admin/users/:id/unlock'),
    ('POST', '/api/v1/admin/users/assign-roles'),
    ('POST', '/api/v1/admin/users/force-logout'),
    ('POST', '/api/v1/admin/users/import'),
    ('GET', '/api/v1/admin/users/import/template')
);
