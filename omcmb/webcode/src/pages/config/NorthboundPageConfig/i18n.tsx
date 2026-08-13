import {
  Children,
  cloneElement,
  isValidElement,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type ReactElement,
  type ReactNode,
} from 'react';
import { useAppStore } from '@core/store/appStore';
import type { Locale } from '@core/types/common';

const exactEn: Record<string, string> = {
  '北向文件配置': 'Northbound File Config',
  'Inventory 文件': 'Inventory Files',
  'Socket 告警': 'Socket Alarms',
  'SNMP 告警': 'SNMP Alarms',
  'SNMP V2C 告警': 'SNMP V2C Alarms',
  'SNMP V3 告警': 'SNMP V3 Alarms',
  '编辑 SNMP V2C': 'Edit SNMP V2C',
  '编辑 SNMP V3': 'Edit SNMP V3',
  '北向 API': 'Northbound API',
  '北向 API 接口清单': 'Northbound API List',
  '北向 API 用户': 'Northbound API Users',
  '北向 API 总开关': 'Northbound API Master Switch',
  '查看接口清单': 'View API List',
  '刷新北向文件配置': 'Refresh Northbound File Config',
  '新增北向文件配置': 'Add Northbound File Config',
  '编辑北向文件配置': 'Edit Northbound File Config',
  '新增配置': 'Add Config',
  '新增目标': 'Add Target',
  '新增': 'Add',
  '保存': 'Save',
  '刷新': 'Refresh',
  '查看': 'View',
  '编辑': 'Edit',
  '删除': 'Delete',
  '启用': 'Enable',
  '上报': 'Report',
  '开': 'On',
  '关': 'Off',
  '已启用': 'Enabled',
  '已停用': 'Disabled',
  '部分开启': 'Partially Enabled',
  '未启用': 'Disabled',
  '正常': 'Normal',
  '异常': 'Broken',
  '待上报': 'Pending',
  '失败': 'Failed',
  '成功': 'Success',
  '处理中': 'Processing',
  '终止': 'Stopped',
  '操作': 'Actions',
  '状态': 'Status',
  '时间': 'Time',
  '事件': 'Event',
  '摘要': 'Summary',
  '目标/内容': 'Target / Content',
  '目标/地址': 'Target / Address',
  '目标名称': 'Target Name',
  '地址': 'Address',
  '结果摘要': 'Result Summary',
  '说明': 'Description',
  '最近事件': 'Recent Events',
  '最近时间': 'Latest Time',
  '对象上报结果': 'Object Report Results',
  '上报结果': 'Report Results',
  '文件内容预览': 'File Content Preview',
  '上报报文': 'Reported Message',
  '接口检查结果': 'API Check Result',
  '复制结果': 'Copy Result',
  '复制报文': 'Copy Message',
  '下载最新文件': 'Download Latest File',
  '下载': 'Download',
  '接口检查': 'API Check',
  '最近检查结果': 'Latest Check Result',
  '检查接口': 'Check API',
  '接口可用': 'API Available',
  '接口名称': 'API Name',
  '模块': 'Module',
  '业务模块': 'Business Module',
  '方法': 'Method',
  '总开关状态': 'Master Switch Status',
  '认证方式': 'Authentication',
  '返回字段': 'Response Fields',
  '字段数量': 'Field Count',
  '接口 URL': 'API URL',
  '接口说明': 'API Description',
  '返回字段清单': 'Response Field List',
  '请求参数示例': 'Request Example',
  '返回结果示例': 'Response Example',
  '字段': 'Field',
  '值': 'Value',
  '字段数': 'Fields',
  '序号': 'No.',
  '业务域': 'Domain',
  '场景号': 'Scene No.',
  '场景名称': 'Scene Name',
  '输出内容': 'Output Content',
  '调度': 'Schedule',
  '文件规则': 'File Rule',
  '压缩': 'Compression',
  '对象': 'Object',
  '制式': 'Technology',
  '输出别名': 'Output Alias',
  '系统字段/指标': 'System Field / Metric',
  '系统字段 key': 'System Field Key',
  '系统取数字段': 'System Source Field',
  '取数字段': 'Source Field',
  '适用范围': 'Scope',
  '类型/口径': 'Type / Aggregation',
  '单位/渲染': 'Unit / Renderer',
  '输出字段': 'Output Field',
  '字段名称': 'Field Name',
  '业务对象': 'Business Object',
  '字段/指标配置': 'Field / Metric Config',
  '字段配置': 'Field Config',
  '告警字段映射': 'Alarm Field Mapping',
  'omcAlarmEntry 字段映射': 'omcAlarmEntry Field Mapping',
  '字段映射': 'Field Mapping',
  '字段模板': 'Field Template',
  '启停': 'Enable',
  '名称': 'Name',
  '用户名': 'Username',
  '密码': 'Password',
  '创建时间': 'Created At',
  '删除用户': 'Delete User',
  '启用状态': 'Enable Status',
  '配置编号': 'Config Code',
  '配置名称': 'Config Name',
  '描述': 'Description',
  '业务对象配置': 'Business Object Config',
  '文件名': 'File Name',
  '文件大小': 'File Size',
  '大小': 'Size',
  '存储路径': 'Storage Path',
  '文件路径': 'File Path',
  '传输目标': 'Delivery Target',
  '新增传输目标': 'Add Delivery Target',
  '测试连接': 'Test Connection',
  '删除目标': 'Delete Target',
  '主机地址': 'Host',
  '端口': 'Port',
  '协议': 'Protocol',
  '目录': 'Directory',
  '超时秒': 'Timeout (s)',
  '重试': 'Retry',
  '账号用途': 'Account Purpose',
  '类型': 'Type',
  '能力范围': 'Scope',
  '目标主机': 'Target Host',
  '目标端口': 'Target Port',
  '服务能力': 'Service Capability',
  '会话与心跳': 'Session / Heartbeat',
  '账号': 'Account',
  '版本': 'Version',
  '通知': 'Notification',
  'Agent 监听': 'Agent Listen',
  '通知目标': 'Notification Target',
  '安全配置': 'Security Config',
  'MIB/清除策略': 'MIB / Clear Policy',
  '查询/清除策略': 'Query / Clear Policy',
  '告警上报': 'Alarm Reporting',
  '启用告警上报': 'Enable Alarm Reporting',
  '允许 MIB 查询': 'Allow MIB Query',
  '未开放查询': 'Query Disabled',
  '未启用上报': 'Reporting Disabled',
  '实时关闭': 'Realtime Off',
  '无同步': 'No Sync',
  '文件补录': 'File Backfill',
  '生成样例': 'Generate Sample',
  '测试发送': 'Test Send',
  '保留原级别': 'Keep Original Severity',
  '主用目标': 'Primary Target',
  '备用目标': 'Backup Target',
  '电信': 'CTCC',
  '联通': 'CUCC',
  '电信 Socket 告警服务端': 'CTCC Socket Alarm Server',
  '联通 Socket 告警服务端': 'CUCC Socket Alarm Server',
  '安全用户': 'Security User',
  '安全级别': 'Security Level',
  '安全名': 'Security Name',
  '认证并加密': 'Auth + Privacy',
  '仅认证': 'Auth Only',
  '不认证不加密': 'No Auth / No Privacy',
  '认证协议': 'Auth Protocol',
  '认证算法': 'Auth Algorithm',
  '认证密码': 'Auth Password',
  '加密协议': 'Privacy Protocol',
  '加密算法': 'Privacy Algorithm',
  '加密密码': 'Privacy Password',
  '通知类型': 'Notification Type',
  '清除级别': 'Clear Severity',
  'MIB 查询': 'MIB Query',
  'MIB 关闭': 'MIB Disabled',
  '监听地址': 'Listen Address',
  '监听端口': 'Listen Port',
  '最大客户端': 'Max Clients',
  '心跳周期': 'Heartbeat Period',
  '心跳次数': 'Heartbeat Times',
  '实时推送': 'Realtime Push',
  '消息同步': 'Message Sync',
  '文件同步': 'File Sync',
  '文件同步账号': 'File Sync Account',
  '实时/消息同步账号': 'Realtime / Message Sync Account',
  '登录、实时告警、历史消息同步': 'Login, realtime alarms, and historical message sync',
  '登录、告警文件同步请求': 'Login and alarm file sync requests',
  '已加密存储': 'Encrypted',
  '未修改保持原密码': 'Keep existing password if unchanged',
  '未修改保持原 community': 'Keep existing community if unchanged',
  '默认 baicells': 'Default baicells',
  '请先填写 SNMP v3 安全名': 'Enter SNMP v3 security name first',
  '请先填写 SNMP v3 认证密码': 'Enter SNMP v3 auth password first',
  '请先填写 SNMP v3 加密密码': 'Enter SNMP v3 privacy password first',
  'SNMP v3 认证密码至少 8 位': 'SNMP v3 auth password must be at least 8 characters',
  'SNMP v3 加密密码至少 8 位': 'SNMP v3 privacy password must be at least 8 characters',
  '请输入认证密码': 'Enter auth password',
  '请输入加密密码': 'Enter privacy password',
  '至少 8 位认证密码': 'At least 8 characters for auth password',
  '至少 8 位加密密码': 'At least 8 characters for privacy password',
  '请输入密码': 'Please enter password',
  '请输入用户名': 'Please enter username',
  '请输入名称': 'Please enter name',
  '请选择': 'Please select',
  '搜索可新增字段/指标': 'Search fields/metrics to add',
  '搜索可新增字段': 'Search fields to add',
  '新增字段': 'Add Field',
  '暂无数据': 'No Data',
  '无可新增字段': 'No fields available',
  '暂无字段目标，请先新增对象': 'No field target. Add an object first.',
  '未配置北向 API 用户，外部系统不能调用北向 API': 'No northbound API users are configured. External systems cannot call northbound APIs.',
  '北向页面配置已刷新': 'Northbound page config refreshed',
  '北向页面配置加载失败，已保留当前页面数据': 'Failed to load northbound page config. Current page data is kept.',
  '指标目录加载失败': 'Failed to load metric catalog',
  '请先填写主机地址和用户名': 'Fill in host and username first.',
  '请输入配置编号': 'Enter config code',
  '配置编号必须使用 S0000 格式': 'Config code must use S0000 format',
  '请先选择业务域': 'Select a domain first',
  '报文已复制': 'Message copied',
  '报文复制失败': 'Failed to copy message',
  '未读取到最近上报记录，已显示配置预览': 'Failed to read recent report records. Showing config preview.',
  '未读取到完整事件详情，已显示列表摘要': 'Failed to read full event details. Showing list summary.',
  '未读取到最近事件，已显示配置预览': 'Failed to read recent events. Showing config preview.',
  '北向 API 总开关保存失败': 'Failed to save northbound API master switch',
  '北向 API 用户已保存': 'Northbound API users saved',
  '北向 API 用户保存失败': 'Failed to save northbound API users',
  '上报文件下载失败': 'Failed to download report file',
  '基础信息': 'Basic Info',
  '接口类型': 'API Type',
  '数据类型': 'Data Type',
  '旧系统支持': 'Old System Support',
  '当前系统支持': 'Current System Support',
  '支持': 'Supported',
  '不支持': 'Not Supported',
  '正式北向': 'Official Northbound',
  '业务复用': 'Business Reuse',
  '鉴权管理': 'Authentication',
  '日志管理': 'Log Management',
  '标准北向': 'Standard Northbound',
  '旧系统兼容': 'Legacy Compatible',
  '认证管理': 'Authentication Management',
  '认证鉴权': 'Authentication',
  '北向用户管理': 'Northbound User Management',
  '北向接口日志': 'Northbound API Logs',
  '设备管理': 'Device Management',
  '设备组管理': 'Device Group Management',
  '参数配置': 'Parameter Configuration',
  '异步任务': 'Async Tasks',
  '高级任务': 'Advanced Tasks',
  '设备日志收集': 'Device Log Collection',
  '数据同步': 'Data Sync',
  '配置导出': 'Config Export',
  '告警查询': 'Alarm Query',
  '性能管理': 'Performance Management',
  'MR 数据': 'MR Data',
  'Inventory 导出': 'Inventory Export',
  '主备服务器': 'Primary/Standby Servers',
  '北向接口': 'Northbound Interface',
  '北向认证：获取访问 Token': 'Northbound Auth: Get Access Token',
  '北向API用户：查询用户列表': 'Northbound API User: Query User List',
  '北向API用户：新增用户': 'Northbound API User: Create User',
  '北向API用户：更新用户': 'Northbound API User: Update User',
  '北向API用户：删除用户': 'Northbound API User: Delete User',
  '北向调用日志：分页查询': 'Northbound API Log: Paged Query',
  '北向调用日志：导出CSV': 'Northbound API Log: Export CSV',
  '设备日志收集：触发运行/故障日志上传': 'Device Log Collection: Trigger Runtime/Fault Log Upload',
  '完整返回字段': 'Full Response Fields',
  '当前返回字段': 'Current Response Fields',
  '示例返回字段': 'Sample Response Fields',
  '公开接口；只校验北向 API 专用用户': 'Public endpoint; validates northbound API users only',
  '北向 API 用户 token；Authorization: Bearer <access-token> 或 X-Northbound-Token': 'Northbound API user token; Authorization: Bearer <access-token> or X-Northbound-Token',
  '外部系统先调用该接口获取访问 Token，再调用其它北向 API。': 'External systems call this API to obtain an access token before calling other northbound APIs.',
  '用于外部系统维护北向 API 调用账号，包括查询、新增、修改和删除。': 'Used by external systems to manage northbound API accounts, including query, create, update, and delete.',
  '用于外部系统查询或导出北向 API 调用日志。': 'Used by external systems to query or export northbound API invocation logs.',
  '用于外部系统拉取设备、告警等北向同步数据。': 'Used by external systems to pull northbound sync data such as devices and alarms.',
  '用于外部系统查询设备清单、设备详情、设备状态、注册设备或触发设备操作。': 'Used by external systems to query device lists, details, status, registration, or trigger device operations.',
  '用于外部系统同步和维护设备分组，以及维护分组内设备关系。': 'Used by external systems to sync and manage device groups and devices in groups.',
  '用于外部系统查询或设置设备参数，返回结果以当前系统设备参数能力为准。': 'Used by external systems to query or set device parameters. Results depend on current system capabilities.',
  '用于外部系统查询当前告警、历史告警或告警统计信息。': 'Used by external systems to query active alarms, historical alarms, or alarm statistics.',
  '用于外部系统导出性能数据或查询聚合指标。': 'Used by external systems to export PM data or query aggregated metrics.',
  '用于外部系统查询或导出 MR 数据。': 'Used by external systems to query or export MR data.',
  '用于外部系统触发设备日志收集并查询处理结果。': 'Used by external systems to trigger device log collection and query processing results.',
  '用于外部系统触发设备运行/故障日志收集并查询处理结果。': 'Used by external systems to trigger runtime/fault log collection and query processing results.',
  '用于外部系统查询异步任务执行结果。': 'Used by external systems to query asynchronous task results.',
  '用于外部系统查询或创建异步任务，并通过任务 ID 查看执行结果。': 'Used by external systems to query or create async tasks and view results by task ID.',
  '用于外部系统维护 HTTP Push 目标、熔断和死信重放。': 'Used by external systems to manage HTTP Push targets, circuit breakers, and dead-letter replay.',
  '用于外部系统查询或维护北向主备服务器配置。': 'Used by external systems to query or maintain northbound primary/standby server configuration.',
  '用于外部系统调用当前系统已开放的北向业务能力。': 'Used by external systems to call northbound business capabilities exposed by the current system.',
};

const phraseRules: Array<[RegExp, string]> = [
  [/^共 (\d+) 条，显示最近 (\d+) 条$/, 'Total $1, showing latest $2'],
  [/^共 (\d+) 个对象：成功 (\d+) · 失败 (\d+)$/, 'Total $1 objects: success $2 · failed $3'],
  [/^共 (\d+) 项$/, '$1 items'],
  [/^字段 (\d+)$/, '$1 fields'],
  [/^返回字段 (\d+) 项$/, '$1 response fields'],
  [/^(.+) 接口检查结果$/, '$1 API Check Result'],
  [/^(.+) 接口检查已记录$/, '$1 API check recorded'],
  [/^(.+) 接口检查失败$/, '$1 API check failed'],
  [/^(.+) 样例报文已记录$/, '$1 sample message recorded'],
  [/^(.+) 样例报文生成失败$/, '$1 failed to generate sample message'],
  [/^(.+) 测试报文已生成$/, '$1 test message generated'],
  [/^(.+) 测试报文生成失败$/, '$1 failed to generate test message'],
  [/^(.+) 测试报文已生成，但目标配置不完整$/, '$1 test message generated, but target config is incomplete'],
  [/^(.+) 已保存$/, '$1 saved'],
  [/^(.+) 配置保存失败$/, '$1 config save failed'],
  [/^(.+) 启停状态保存失败$/, '$1 enable status save failed'],
  [/^(.+) 手动执行失败$/, '$1 manual run failed'],
  [/^(.+) 已生成上报记录$/, '$1 report record generated'],
  [/^(.+) 已生成 (\d+) 条上报记录$/, '$1 generated $2 report records'],
  [/^(.+) 已复制为草稿$/, '$1 copied as draft'],
  [/^新增配置已保存：(.+)$/, 'New config saved: $1'],
  [/^编辑配置已保存：(.+)$/, 'Config saved: $1'],
  [/^配置保存失败：(.+)$/, 'Config save failed: $1'],
  [/^北向 API 总开关已(.+)$/, 'Northbound API master switch $1'],
  [/^(.+) (FTP|SFTP) 连接测试通过$/, '$1 $2 connection test passed'],
  [/^(.+) (FTP|SFTP) 连接测试未通过$/, '$1 $2 connection test did not pass'],
  [/^(.+) (FTP|SFTP) 连接测试失败$/, '$1 $2 connection test failed'],
  [/^(.+) (SNMP|Socket) 配置保存失败$/, '$1 $2 config save failed'],
];

const replacements: Array<[string, string]> = [
  ['北向页面配置', 'Northbound page config'],
  ['北向页面化配置', 'Northbound page config'],
  ['北向文件配置', 'Northbound file config'],
  ['北向API用户', 'Northbound API user'],
  ['北向调用日志', 'Northbound API log'],
  ['北向 API', 'Northbound API'],
  ['设备日志收集', 'device log collection'],
  ['运行/故障日志上传', 'runtime/fault log upload'],
  ['查询用户列表', 'query user list'],
  ['新增用户', 'create user'],
  ['更新用户', 'update user'],
  ['删除用户', 'delete user'],
  ['导出CSV', 'export CSV'],
  ['分页查询', 'paged query'],
  ['Socket 告警', 'Socket alarms'],
  ['SNMP 告警', 'SNMP alarms'],
  ['Inventory 文件', 'Inventory files'],
  ['文件内容预览', 'File content preview'],
  ['上报报文', 'Reported message'],
  ['接口检查结果', 'API check result'],
  ['接口检查', 'API check'],
  ['实时告警', 'Realtime alarm'],
  ['历史消息同步', 'historical message sync'],
  ['消息同步', 'message sync'],
  ['文件同步', 'file sync'],
  ['样例报文', 'sample message'],
  ['测试报文', 'test message'],
  ['配置预览', 'config preview'],
  ['运行记录', 'run record'],
  ['连接测试', 'connection test'],
  ['文件投递', 'file delivery'],
  ['心跳', 'heartbeat'],
  ['登录', 'login'],
  ['会话重载', 'session reload'],
  ['端口启动', 'port started'],
  ['端口停止', 'port stopped'],
  ['端口失败', 'port failed'],
  ['同步查询', 'sync query'],
  ['连接拒绝', 'connection rejected'],
  ['主用目标', 'primary target'],
  ['备用目标', 'backup target'],
  ['主用', 'primary'],
  ['备用', 'secondary'],
  ['电信 Socket 告警服务端', 'CTCC socket alarm server'],
  ['联通 Socket 告警服务端', 'CUCC socket alarm server'],
  ['电信', 'CTCC'],
  ['联通', 'CUCC'],
  ['服务端', 'server'],
  ['成功', 'success'],
  ['失败', 'failed'],
  ['正常', 'normal'],
  ['异常', 'broken'],
  ['待上报', 'pending'],
  ['启用', 'enabled'],
  ['手动执行', 'manual run'],
  ['停用', 'disabled'],
  ['关闭', 'disabled'],
  ['未启用', 'disabled'],
  ['处理中', 'processing'],
  ['生成中', 'generating'],
  ['生成成功', 'generated'],
  ['生成失败', 'generation failed'],
  ['等待发送', 'waiting to send'],
  ['等待启用传输目标', 'waiting for delivery target'],
  ['每 7 天', 'every 7 days'],
  ['7 天', '7 days'],
  ['每月', 'monthly'],
  ['每日', 'daily'],
  ['分钟', 'minutes'],
  ['小时', 'hours'],
  ['暂无在线客户端', 'no online clients'],
  ['配置开关关闭', 'config switch is off'],
  ['当前不会触发上报', 'reporting will not be triggered'],
  ['最近一次', 'latest'],
  ['可查看', 'can be viewed'],
  ['已记录', 'recorded'],
  ['已生成', 'generated'],
  ['已保存', 'saved'],
  ['已刷新', 'refreshed'],
  ['保存失败', 'save failed'],
  ['加载失败', 'load failed'],
  ['复制失败', 'copy failed'],
  ['下载失败', 'download failed'],
  ['请先', 'please first'],
  ['请填写', 'please fill in'],
  ['请先选择', 'please select'],
  ['请输入', 'please enter'],
  ['请选择', 'please select'],
  ['未读取到', 'failed to read'],
  ['已显示', 'showing'],
  ['完整', 'full'],
  ['当前', 'current'],
  ['示例', 'sample'],
  ['返回字段', 'response fields'],
  ['请求方式', 'method'],
  ['接口地址', 'API URL'],
  ['接口名称', 'API name'],
  ['接口类型', 'API type'],
  ['数据类型', 'data type'],
  ['所属模块', 'module'],
  ['字段数量', 'field count'],
  ['字段清单', 'field list'],
  ['字段模板', 'field template'],
  ['字段映射', 'field mapping'],
  ['字段配置', 'field config'],
  ['字段/指标配置', 'field/metric config'],
  ['场景号', 'scene no.'],
  ['场景名称', 'scene name'],
  ['输出内容', 'output content'],
  ['文件规则', 'file rule'],
  ['压缩', 'compression'],
  ['调度', 'schedule'],
  ['系统字段/指标', 'system field/metric'],
  ['系统字段', 'system field'],
  ['取数字段', 'source field'],
  ['输出字段', 'output field'],
  ['输出别名', 'output alias'],
  ['字段名称', 'field name'],
  ['适用范围', 'scope'],
  ['类型/口径', 'type/aggregation'],
  ['单位/渲染', 'unit/renderer'],
  ['业务对象', 'business object'],
  ['业务域', 'domain'],
  ['目标名称', 'target name'],
  ['服务能力', 'service capability'],
  ['会话与心跳', 'session/heartbeat'],
  ['实时关闭', 'realtime off'],
  ['无同步', 'no sync'],
  ['文件补录', 'file backfill'],
  ['生成样例', 'generate sample'],
  ['测试发送', 'test send'],
  ['版本', 'version'],
  ['通知目标', 'notification target'],
  ['通知', 'notification'],
  ['安全配置', 'security config'],
  ['查询/清除策略', 'query/clear policy'],
  ['告警上报', 'alarm reporting'],
  ['启用告警上报', 'enable alarm reporting'],
  ['允许 MIB 查询', 'allow MIB query'],
  ['未开放查询', 'query disabled'],
  ['未启用上报', 'reporting disabled'],
  ['保留原级别', 'keep original severity'],
  ['MIB/清除策略', 'MIB/clear policy'],
  ['Agent 监听', 'agent listen'],
  ['次超时', 'timeouts'],
  ['连接', 'connections'],
  ['对象', 'object'],
  ['制式', 'technology'],
  ['目标/内容', 'target/content'],
  ['目标/地址', 'target/address'],
  ['目标', 'target'],
  ['结果摘要', 'result summary'],
  ['基础信息', 'basic info'],
  ['操作', 'actions'],
  ['状态', 'status'],
  ['时间', 'time'],
  ['事件', 'event'],
  ['摘要', 'summary'],
  ['说明', 'description'],
  ['模块', 'module'],
  ['方法', 'method'],
  ['名称', 'name'],
  ['用户名', 'username'],
  ['密码', 'password'],
  ['创建时间', 'created at'],
  ['更新时间', 'updated at'],
  ['设备序列号', 'device serial number'],
  ['设备名称', 'device name'],
  ['设备信息', 'device info'],
  ['设备状态', 'device status'],
  ['设备组', 'device group'],
  ['设备', 'device'],
  ['小区状态', 'cell status'],
  ['小区名称', 'cell name'],
  ['小区 ID', 'cell ID'],
  ['小区', 'cell'],
  ['告警级别', 'alarm severity'],
  ['告警类型', 'alarm type'],
  ['告警标题', 'alarm title'],
  ['告警状态', 'alarm status'],
  ['告警序号', 'alarm sequence'],
  ['活动告警数', 'active alarm count'],
  ['最高告警级别', 'highest alarm severity'],
  ['告警', 'alarm'],
  ['日志时间', 'log time'],
  ['日志状态', 'log status'],
  ['日志结果', 'log result'],
  ['日志详情', 'log detail'],
  ['日志', 'log'],
  ['文件名', 'file name'],
  ['文件大小', 'file size'],
  ['文件路径', 'file path'],
  ['文件', 'file'],
  ['传输目标', 'delivery target'],
  ['目标主机', 'target host'],
  ['目标端口', 'target port'],
  ['主机地址', 'host'],
  ['端口', 'port'],
  ['协议', 'protocol'],
  ['目录', 'directory'],
  ['类型', 'type'],
  ['账号用途', 'account purpose'],
  ['能力范围', 'scope'],
  ['安全用户', 'security user'],
  ['认证协议', 'auth protocol'],
  ['认证密码', 'auth password'],
  ['加密协议', 'privacy protocol'],
  ['加密密码', 'privacy password'],
  ['通知类型', 'notification type'],
  ['清除级别', 'clear severity'],
  ['监听地址', 'listen address'],
  ['监听端口', 'listen port'],
  ['最大客户端', 'max clients'],
  ['心跳周期', 'heartbeat period'],
  ['心跳次数', 'heartbeat times'],
  ['实时推送', 'realtime push'],
  ['超时秒', 'timeout seconds'],
  ['重试', 'retry'],
  ['序号', 'no.'],
  ['大小', 'size'],
  ['详情', 'details'],
  ['刷新', 'refresh'],
  ['新增', 'add'],
  ['编辑', 'edit'],
  ['删除', 'delete'],
  ['查看', 'view'],
  ['保存', 'save'],
  ['下载', 'download'],
  ['复制', 'copy'],
  ['搜索', 'search'],
  ['全部', 'all'],
  ['默认', 'default'],
  ['不区分制式', 'not technology-specific'],
  ['登录/安全日志', 'login/security log'],
  ['操作日志', 'operation log'],
  ['固定格式', 'fixed format'],
  ['空值占位', 'empty placeholder'],
  ['厂家', 'vendor'],
  ['型号', 'model'],
  ['产品类型', 'product type'],
  ['产品名称', 'product name'],
  ['硬件版本', 'hardware version'],
  ['软件版本', 'software version'],
  ['首次上线时间', 'first online time'],
  ['最近上报时间', 'latest report time'],
  ['最近上线时间', 'latest online time'],
  ['最近离线时间', 'latest offline time'],
  ['在线状态', 'online status'],
  ['管理状态', 'management status'],
  ['运行时长', 'runtime'],
  ['累计在线时长', 'cumulative online time'],
  ['经度', 'longitude'],
  ['纬度', 'latitude'],
  ['高度', 'height'],
  ['频点', 'frequency'],
  ['频段', 'band'],
  ['带宽', 'bandwidth'],
  ['上行', 'uplink'],
  ['下行', 'downlink'],
  ['发射功率', 'tx power'],
  ['双工模式', 'duplex mode'],
  ['同步状态', 'sync status'],
  ['射频状态', 'RF status'],
  ['省份', 'province'],
  ['区域', 'region'],
  ['本机标识', 'local host ID'],
  ['数据版本', 'data version'],
  ['文件时间', 'file time'],
];

function preservePadding(original: string, translated: string): string {
  const leading = original.match(/^\s*/)?.[0] ?? '';
  const trailing = original.match(/\s*$/)?.[0] ?? '';
  return `${leading}${translated}${trailing}`;
}

function normalizeText(value: string): string {
  return value.replace(/\s+/g, ' ').trim();
}

function normalizeLocale(value: string | null | undefined): Locale | undefined {
  if (!value) return undefined;
  const normalized = value.toLowerCase();
  if (normalized === 'en' || normalized.startsWith('en-')) return 'en-US';
  if (normalized === 'zh' || normalized.startsWith('zh-')) return 'zh-CN';
  return undefined;
}

function readPersistedLocale(): Locale | undefined {
  if (typeof window === 'undefined') return undefined;
  try {
    const raw = window.localStorage.getItem('omc-app-store');
    if (!raw) return undefined;
    const parsed = JSON.parse(raw) as { state?: { locale?: string } };
    return normalizeLocale(parsed.state?.locale);
  } catch {
    return undefined;
  }
}

function getDocumentLocale(): Locale | undefined {
  if (typeof document === 'undefined') return undefined;
  return normalizeLocale(document.documentElement.lang);
}

export function useNorthboundLocale(): Locale {
  const storeLocale = useAppStore((state) => state.locale);
  const [documentLocale, setDocumentLocale] = useState<Locale | undefined>(() => getDocumentLocale());
  const [persistedLocale, setPersistedLocale] = useState<Locale | undefined>(() => readPersistedLocale());

  useEffect(() => {
    if (typeof window === 'undefined') return undefined;
    const refresh = () => {
      setDocumentLocale(getDocumentLocale());
      setPersistedLocale(readPersistedLocale());
    };
    refresh();
    window.addEventListener('storage', refresh);
    const timer = window.setInterval(refresh, 1000);
    return () => {
      window.removeEventListener('storage', refresh);
      window.clearInterval(timer);
    };
  }, []);

  return normalizeLocale(storeLocale) ?? persistedLocale ?? documentLocale ?? 'zh-CN';
}

export function translateNorthboundText(value: string, locale: Locale): string {
  if (locale !== 'en-US' || !/[\u4e00-\u9fff]/.test(value)) return value;
  const text = normalizeText(value);
  if (!text) return value;
  const exact = exactEn[text];
  if (exact) return preservePadding(value, exact);
  for (const [pattern, replacement] of phraseRules) {
    if (pattern.test(text)) return preservePadding(value, text.replace(pattern, replacement));
  }
  let result = text;
  for (const [zh, en] of replacements) {
    result = result.split(zh).join(en);
  }
  result = result
    .replace(/：/g, ': ')
    .replace(/，/g, ', ')
    .replace(/。/g, '.')
    .replace(/；/g, '; ')
    .replace(/（/g, ' (')
    .replace(/）/g, ')')
    .replace(/、/g, ' / ')
    .replace(/\s{2,}/g, ' ')
    .trim();
  return preservePadding(value, result);
}

export function useNorthboundI18n(): (value: string) => string {
  const locale = useNorthboundLocale();
  return useMemo(() => (value: string) => translateNorthboundText(value, locale), [locale]);
}

const textPropNames = new Set([
  'title',
  'label',
  'placeholder',
  'aria-label',
  'emptyText',
  'okText',
  'cancelText',
  'checkedChildren',
  'unCheckedChildren',
  'notFoundContent',
  'addonBefore',
  'addonAfter',
  'content',
]);

const domTextOriginals = new WeakMap<Text, string>();
const domAttrOriginals = new WeakMap<Element, Map<string, string>>();
const domTextAttrNames = ['title', 'placeholder', 'aria-label'];
const domSkipSelector = [
  'pre',
  'code',
  'script',
  'style',
  'textarea',
  '[data-northbound-i18n-skip="true"]',
].join(',');

function isSkippedDomNode(node: Node): boolean {
  const element = node.nodeType === Node.ELEMENT_NODE
    ? node as Element
    : node.parentElement;
  return Boolean(element?.closest(domSkipSelector));
}

function translateDomTextNode(node: Text, locale: Locale): void {
  if (isSkippedDomNode(node)) return;
  const current = node.nodeValue ?? '';
  if (!current.trim()) return;
  if (locale === 'en-US') {
    const source = /[\u4e00-\u9fff]/.test(current)
      ? current
      : domTextOriginals.get(node) ?? current;
    domTextOriginals.set(node, source);
    const translated = translateNorthboundText(source, locale);
    if (translated !== current) node.nodeValue = translated;
    return;
  }
  const original = domTextOriginals.get(node);
  if (original && original !== current) node.nodeValue = original;
}

function getAttrOriginals(element: Element): Map<string, string> {
  const existing = domAttrOriginals.get(element);
  if (existing) return existing;
  const created = new Map<string, string>();
  domAttrOriginals.set(element, created);
  return created;
}

function translateDomAttributes(element: Element, locale: Locale): void {
  if (isSkippedDomNode(element)) return;
  const originals = getAttrOriginals(element);
  for (const attrName of domTextAttrNames) {
    const current = element.getAttribute(attrName);
    if (!current || !current.trim()) continue;
    if (locale === 'en-US') {
      const source = /[\u4e00-\u9fff]/.test(current)
        ? current
        : originals.get(attrName) ?? current;
      originals.set(attrName, source);
      const translated = translateNorthboundText(source, locale);
      if (translated !== current) element.setAttribute(attrName, translated);
    } else {
      const original = originals.get(attrName);
      if (original && original !== current) element.setAttribute(attrName, original);
    }
  }
}

function translateDomTree(root: HTMLElement, locale: Locale): void {
  translateDomAttributes(root, locale);
  const walker = document.createTreeWalker(
    root,
    NodeFilter.SHOW_ELEMENT | NodeFilter.SHOW_TEXT,
    {
      acceptNode(node) {
        if (isSkippedDomNode(node)) return NodeFilter.FILTER_REJECT;
        return NodeFilter.FILTER_ACCEPT;
      },
    },
  );
  let current = walker.nextNode();
  while (current) {
    if (current.nodeType === Node.TEXT_NODE) {
      translateDomTextNode(current as Text, locale);
    } else if (current.nodeType === Node.ELEMENT_NODE) {
      translateDomAttributes(current as Element, locale);
    }
    current = walker.nextNode();
  }
}

function translateNode(node: ReactNode, locale: Locale): ReactNode {
  if (locale !== 'en-US') return node;
  if (typeof node === 'string') return translateNorthboundText(node, locale);
  if (typeof node === 'number' || node === null || node === undefined || typeof node === 'boolean') return node;
  if (Array.isArray(node)) return Children.map(node, (child) => translateNode(child, locale));
  if (!isValidElement(node)) return node;
  return translateElement(node, locale);
}

function translateColumns(columns: unknown, locale: Locale): unknown {
  if (!Array.isArray(columns)) return columns;
  return columns.map((column) => {
    if (!column || typeof column !== 'object') return column;
    const next: Record<string, unknown> = { ...(column as Record<string, unknown>) };
    if ('title' in next) next.title = translateNode(next.title as ReactNode, locale);
    if (Array.isArray(next.children)) next.children = translateColumns(next.children, locale);
    if (typeof next.render === 'function') {
      const originalRender = next.render as (...args: unknown[]) => ReactNode;
      next.render = (...args: unknown[]) => translateNode(originalRender(...args), locale);
    }
    return next;
  });
}

function translateItems(items: unknown, locale: Locale): unknown {
  if (!Array.isArray(items)) return items;
  return items.map((item) => {
    if (!item || typeof item !== 'object') return item;
    const next: Record<string, unknown> = { ...(item as Record<string, unknown>) };
    for (const key of ['label', 'title', 'children']) {
      if (key in next) next[key] = translateNode(next[key] as ReactNode, locale);
    }
    return next;
  });
}

function translateObjectTextProps(value: unknown, locale: Locale, keys: string[]): unknown {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return value;
  const next: Record<string, unknown> = { ...(value as Record<string, unknown>) };
  for (const key of keys) {
    if (typeof next[key] === 'string') next[key] = translateNorthboundText(next[key] as string, locale);
  }
  return next;
}

function translateMenu(menu: unknown, locale: Locale): unknown {
  if (!menu || typeof menu !== 'object' || Array.isArray(menu)) return menu;
  const next: Record<string, unknown> = { ...(menu as Record<string, unknown>) };
  if (Array.isArray(next.items)) next.items = translateItems(next.items, locale);
  return next;
}

function translatePagination(pagination: unknown, locale: Locale): unknown {
  if (!pagination || typeof pagination !== 'object' || Array.isArray(pagination)) return pagination;
  const next: Record<string, unknown> = { ...(pagination as Record<string, unknown>) };
  if (typeof next.showTotal === 'function') {
    const original = next.showTotal as (...args: unknown[]) => ReactNode;
    next.showTotal = (...args: unknown[]) => translateNode(original(...args), locale);
  }
  return next;
}

function translateElement(element: ReactElement, locale: Locale): ReactElement {
  const elementType = typeof element.type === 'string' ? element.type : '';
  if (elementType === 'pre' || elementType === 'code') return element;
  const props = element.props as Record<string, unknown>;
  const nextProps: Record<string, unknown> = {};
  let changed = false;

  for (const [key, propValue] of Object.entries(props)) {
    let nextValue = propValue;
    if (textPropNames.has(key)) nextValue = translateNode(propValue as ReactNode, locale);
    if (key === 'columns') nextValue = translateColumns(propValue, locale);
    if (key === 'items') nextValue = translateItems(propValue, locale);
    if (key === 'options') nextValue = translateItems(propValue, locale);
    if (key === 'menu') nextValue = translateMenu(propValue, locale);
    if (key === 'locale') nextValue = translateObjectTextProps(propValue, locale, ['emptyText']);
    if (key === 'pagination') nextValue = translatePagination(propValue, locale);
    if (key === 'children') nextValue = translateNode(propValue as ReactNode, locale);
    if (nextValue !== propValue) changed = true;
    nextProps[key] = nextValue;
  }

  return changed ? cloneElement(element, nextProps) : element;
}

export function NorthboundI18nScope({ children }: { children: ReactNode }) {
  const locale = useNorthboundLocale();
  const rootRef = useRef<HTMLSpanElement | null>(null);

  useLayoutEffect(() => {
    const root = rootRef.current
      ?? document.querySelector<HTMLElement>('[data-northbound-i18n-root="true"]');
    if (!root) return undefined;
    let timer: number | null = null;
    let settleTimer: number | null = null;
    const settleInterval = window.setInterval(() => {
      translateDomTree(root, locale);
    }, 250);
    const run = () => {
      if (timer !== null) return;
      timer = window.setTimeout(() => {
        timer = null;
        translateDomTree(root, locale);
      }, 0);
    };

    translateDomTree(root, locale);
    settleTimer = window.setTimeout(() => {
      window.clearInterval(settleInterval);
      settleTimer = null;
    }, 6000);
    const observer = new MutationObserver(run);
    observer.observe(root, {
      subtree: true,
      childList: true,
      characterData: true,
      attributes: true,
      attributeFilter: domTextAttrNames,
    });

    return () => {
      if (timer !== null) window.clearTimeout(timer);
      if (settleTimer !== null) window.clearTimeout(settleTimer);
      window.clearInterval(settleInterval);
      observer.disconnect();
    };
  }, [locale]);

  return (
    <span
      ref={rootRef}
      data-northbound-i18n-root="true"
      data-northbound-i18n-locale={locale}
      style={{ display: 'contents' }}
    >
      {translateNode(children, locale)}
    </span>
  );
}
