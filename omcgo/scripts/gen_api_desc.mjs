// gen_api_desc.mjs — 为 api_endpoints.description 批量生成中文描述。
//
// 用法：
//   1. 把当前数据库的全部端点导出到 /tmp/api_endpoints.txt：
//      docker exec docker-postgres-1 psql -U omcgo -d omcgo -t -A -F "|" \
//        -c "SELECT method, path, api_group FROM api_endpoints ORDER BY api_group, path, method;" \
//        > /tmp/api_endpoints.txt
//   2. node omcgo/scripts/gen_api_desc.mjs
//   3. 输出 SQL 在 /tmp/058.sql，按下一个迁移版本号挪到 omcgo/migrations/seed/ 即可。
//
// 启发式规则（详见 RESOURCE_CN / SEG_CN / camelToCN）：
//   - 资源名取自 api_group → 中文（缺省走原 group 字面）
//   - 末段是已知动词（acknowledge / sync / reboot…）→ 拼成「资源 - 动作」
//   - 末段是 camelCase（如 addIndicatorGroup）→ 拆「动词 + 对象」
//   - 末段是 :id → 按 method 默认（详情/更新/删除）
//   - 兜底「资源 - 列表/创建/批量删除…」
import { readFileSync, writeFileSync } from 'node:fs';

const RESOURCE_CN = {
  'alarms': '告警', 'api-endpoints': 'API 端点', 'api-keys': 'API 密钥',
  'audit-logs': '审计日志', 'auth': '认证', 'backup': '备份',
  'cell': '小区', 'column-configs': '列配置', 'config': '配置',
  'dashboard': '仪表盘', 'datamodels': '数据模型', 'dead-letters': '死信队列',
  'device-groups': '设备分组', 'device-registrations': '设备预登记',
  'device-rules': '设备规则', 'devices': '设备', 'events': '事件',
  'files': '文件', 'firmware': '固件', 'gnb': 'gNB', 'groups': '用户组',
  'healthz': '健康检查', 'interop': '互操作', 'licenses': '许可证',
  'logs': '日志', 'menus': '菜单', 'mml': 'MML', 'mr': '测量报告 MR',
  'northbound': '北向接口', 'notifications': '通知', 'ops': '运维任务',
  'oui': 'OUI', 'permissions': '权限', 'pm': '性能 PM',
  'provisioning': '自动开站', 'readyz': '就绪检查', 'reports': '报表',
  'roles': '角色', 'sites': '站点', 'sysConfig': '系统配置',
  'sysDictionary': '字典', 'sysDictionaryDetail': '字典明细',
  'system': '系统信息', 'tasks': '任务', 'templates': '模板',
  'topology': '拓扑', 'upgrade-sub-tasks': '升级子任务',
  'upgrade-tasks': '升级任务', 'users': '用户',
};

// 通用动作段
const SEG_CN = {
  acknowledge: '确认', unacknowledge: '取消确认', clear: '清除', read: '标记已读',
  lock: '锁定', unlock: '解锁', 'reset-password': '重置密码', reset: '重置',
  'force-logout': '强制下线', copy: '复制', enable: '启用', disable: '禁用',
  toggle: '切换启用状态', recommend: '设为推荐', sync: '触发同步',
  reboot: '重启', 'batch-reboot': '批量重启', 'param-sync': '参数同步',
  'rf-switch': '射频开关', cancel: '取消', retry: '重试', rollback: '回滚',
  suspend: '暂停', resume: '恢复', terminate: '终止', pause: '暂停',
  start: '启动', advance: '推进灰度阶段', 'pause-canary': '暂停灰度',
  'resume-canary': '恢复灰度', 'abort-canary': '中止灰度', replay: '重放',
  login: '登录', logout: '注销', refresh: '刷新令牌', me: '当前用户信息',
  captcha: '获取验证码', 'change-password': '修改密码', 'switch-role': '切换角色',
  tree: '树形结构', stats: '统计', statistics: '统计',
  history: '历史记录', enums: '枚举字典', geo: '地理位置数据',
  search: '搜索', export: '导出', download: '下载', upload: '上传',
  execute: '执行', apply: '应用', detail: '详情', 'next-priority': '下一可用优先级',
  permanent: '彻底删除', restore: '恢复', recycle: '回收站',
  children: '子节点', schema: '架构 schema', discover: '探测',
  'sync-status': '同步状态', add: '新增', delete: '删除',
  'config-file': '配置文件', 'sub-objects': '子对象', 'param-paths': '参数路径',
  lookup: '查询映射', all: '全量', mappings: '映射', indicators: '指标',
  fields: '字段', health: '健康检查', circuit: '熔断器状态',
  targets: '推送目标', deadletter: '死信', permission: '权限',
  i18n: '国际化文案', commands: '命令', scripts: '脚本', runs: '执行历史',
  'param-versions': '参数版本', 'dangerous-check': '危险命令检查',
  thresholds: '阈值', kpi: 'KPI', definitions: '定义', calculate: '计算',
  aggregated: '聚合', counters: '计数器',
  'check-delete': '校验是否可删除', sort: '排序', 'move-devices': '移动设备',
  records: '记录', segments: '段', incremental: '增量同步', full: '全量同步',
  results: '结果', 'gnb-list': 'gNB 列表', cells: '小区列表',
  template: '模板', confirm: '确认', dictionary: '字典',
  'auto-discovery': '自动发现', baseline: '基线',
  preview: '预览', 'force-sync': '强制同步', revoke: '撤销',
  groups: '分组', migrate: '迁移', rotate: '轮转',
  clone: '克隆', params: '参数', perms: '权限',
  perfmgmt: '性能管理', kpimanage: 'KPI 管理', indicatormg: '指标管理',
  devices: '设备列表', // /device-groups/:id/devices
  pull: '拉取', push: '推送', run: '执行',
  validate: '校验', sync: '触发同步',
  'gnb-list': 'gNB 列表', activate: '激活', deactivate: '停用',
  'force-logout': '强制下线', info: '信息',
  details: '明细',
  permanent: '彻底删除', restore: '恢复',
  // device-info / device-detail
  detail: '详情',
  // dashboard 子项
  'alarm-trend': '告警趋势', 'alarm-type-pie': '告警类型分布',
  'device-status': '设备状态分布', 'kpi-time-series': 'KPI 时间序列',
  'kpi-trend': 'KPI 趋势', 'region-stats': '区域统计',
  summary: '汇总', widgets: '组件',
  // pm 相关
  baselines: '基线', diff: '差异', 'product-classes': '产品型号',
  // notification
  templates: '模板', test: '测试',
  // northbound
  config: '配置', alarms: '告警', pm: 'PM',
  // device 相关
  parameters: '参数', objects: '多实例对象', 'param-sync': '参数同步',
  // upgrade canary
  'upgrade-tasks': '升级任务', 'sub-tasks': '子任务',
  // device-rules
  apply: '应用', 'next-priority': '下一可用优先级',
  // mml 子段
  'param-versions': '参数版本',
  // devices specific
  'recycle': '回收站',
  // backup
  'ftp-configs': 'FTP 配置', 'restore-tasks': '恢复任务', schedules: '调度计划',
  policy: '策略',
  // events
  stream: 'SSE 推送流',
  // dead-letters
  retry: '重试',
  // files
  preview: '预览',
  // permissions
  // mml indicator subgroups
  'isIndicatorInTemplate': '是否在模板中',
  // tasks
  purge: '清理', timeout: '超时清理',
};

// camelCase 末段动作翻译（cell/gnb 这类 RPC 风格端点）
const CAMEL_VERB = {
  add: '新增', addOrModify: '新增或修改', del: '删除', delete: '删除',
  modify: '修改', update: '更新', get: '查询', list: '列表',
  enable: '启用', disable: '禁用', export: '导出', import: '导入',
  query: '查询', save: '保存', set: '设置', clear: '清除',
  start: '启动', stop: '停止', cancel: '取消', confirm: '确认',
  create: '创建', find: '查询', remove: '移除', search: '搜索',
};

const CAMEL_OBJ = {
  Indicator: '指标', IndicatorGroup: '指标分组', BaseKpiCustName: '基础 KPI 自定义名',
  GnbIndicatorsName: 'gNB 指标名', AllIndicator: '全部指标',
  IndicatorListByPage: '指标分页列表', IndicatorInfo: '指标信息',
  IndicatorGroupInfo: '指标分组信息', IndicatorGroupTree: '指标分组树',
  SysDictionary: '字典', SysDictionaryDetail: '字典明细',
};

// 父段（penultimate）context
const PARENT_CN = {
  perfmgmt: '性能管理',
  kpimanage: 'KPI 管理',
  pm: '性能',
  indicatormg: '指标管理',
  push: '推送',
  export: '导出',
  active: '活跃',
  history: '历史',
  recycle: '回收站',
  batch: '批量',
  canary: '灰度',
  'alarm-libraries': '告警库',
  'alarm-filters': '告警过滤规则',
  'param-versions': '参数版本',
  parameters: '参数',
  objects: '多实例对象',
  schedules: '调度',
  ftp: 'FTP',
  configs: '配置',
  policy: '策略',
  restore: '恢复',
  permissions: '权限',
  api: 'API',
  mappings: '映射',
  indicators: '指标',
  i18n: '国际化',
};

// 多 token camelCase 动词（addOrModify、getOrCreate 等）。
// 命中后用 prefix 翻译做 verb，剩余部分继续走对象解析。
const COMPOUND_VERBS = [
  { re: /^addOrModify/, cn: '新增或修改' },
  { re: /^getOrCreate/, cn: '查询或创建' },
  { re: /^isIndicatorInTemplate$/, cn: '是否在模板中' },
];

function camelToCN(seg) {
  for (const cv of COMPOUND_VERBS) {
    const m = seg.match(cv.re);
    if (m) {
      const tail = seg.slice(m[0].length);
      if (!tail) return cv.cn;
      const obj = CAMEL_OBJ[tail] || tail.split(/(?=[A-Z])/).map(s => CAMEL_OBJ[s] || s).join('');
      return cv.cn + obj;
    }
  }
  // 把 addIndicatorGroup → 新增指标分组
  const m = seg.match(/^([a-z]+)([A-Z][A-Za-z0-9]*)?$/);
  if (!m) return null;
  const [, verbRaw, objRaw] = m;
  const verb = CAMEL_VERB[verbRaw];
  if (!verb) return null;
  if (!objRaw) return verb;
  const obj = CAMEL_OBJ[objRaw];
  if (obj) return `${verb}${obj}`;
  const segs = objRaw.split(/(?=[A-Z])/).map(s => CAMEL_OBJ[s] || s);
  return verb + segs.join('');
}

function lastMeaning(seg, group) {
  // seg 与资源 group 同名时不视为动词，避免「用户组 - 分组」这种重复
  if (group && seg === group) return null;
  if (SEG_CN[seg]) return SEG_CN[seg];
  if (/[A-Z]/.test(seg)) {
    const c = camelToCN(seg);
    if (c) return c;
  }
  return null;
}

function parse(path, method, group) {
  let p = path;
  if (p.startsWith('/api/v1/')) p = p.slice('/api/v1/'.length);
  else if (p.startsWith('/')) p = p.slice(1);
  let parts = p ? p.split('/') : [];
  const res = RESOURCE_CN[group] || group;

  if (parts[0] === 'admin') parts = parts.slice(1);

  const last = parts[parts.length - 1] || '';
  const isIdLast = last.startsWith(':');
  const M = method.toUpperCase();

  if (group === 'auth') {
    const m = lastMeaning(last, group);
    return m ? `${res} - ${m}` : `${res} - ${last}`;
  }
  if (group === 'healthz' || group === 'readyz') return res;
  if (group === 'system') return '系统运行信息';

  // 找 :id 之后的子段（最近一次 :xxx 之后的所有段）
  let lastIdAt = -1;
  for (let i = parts.length - 1; i >= 0; i--) {
    if (parts[i].startsWith(':')) { lastIdAt = i; break; }
  }
  const subAfterId = lastIdAt >= 0 ? parts.slice(lastIdAt + 1) : [];

  let verbSeg = null;
  if (subAfterId.length) {
    for (let i = subAfterId.length - 1; i >= 0; i--) {
      if (lastMeaning(subAfterId[i], group)) { verbSeg = subAfterId[i]; break; }
    }
    if (!verbSeg) verbSeg = subAfterId[subAfterId.length - 1];
  } else if (!isIdLast && lastMeaning(last, group)) {
    verbSeg = last;
  } else if (!isIdLast && /[A-Z]/.test(last) && last !== group) {
    // camelCase last segment（cell/gnb）；与 group 同名时跳过避免循环命中默认分支
    verbSeg = last;
  }

  // isIdLast：尝试用前一段当 verb（如 /export/config/:deviceId 的 config）
  if (isIdLast && !verbSeg && parts.length >= 2) {
    const prev = parts[parts.length - 2];
    if (!prev.startsWith(':') && lastMeaning(prev, group)) {
      verbSeg = prev;
    }
  }

  // 父段（用于丰富语义）：取 verbSeg 之前的非 :id 非根 group 段
  let parentCn = '';
  if (verbSeg) {
    const verbIdx = parts.lastIndexOf(verbSeg);
    if (verbIdx > 0) {
      const parent = parts[verbIdx - 1];
      if (!parent.startsWith(':') && parent !== group && PARENT_CN[parent]) {
        parentCn = PARENT_CN[parent];
      }
    }
  }

  const dedup = (a, b) => a === b ? a : (a + b);

  if (verbSeg) {
    const meaning = lastMeaning(verbSeg, group);
    // verbSeg 没有翻译且原文还是 group 名时，撤回让默认分支按 method 处理
    if (!meaning && verbSeg === group) {
      verbSeg = null;
    }
  }
  if (verbSeg) {
    let vCn = lastMeaning(verbSeg, group) || verbSeg;
    if (parentCn) {
      vCn = `${parentCn}${vCn}`;
    }
    // 跳过 X-X 重复
    if (vCn === res || vCn === group) return res;
    // isIdLast 用了前段当 verb，再加上「详情/更新/删除」语义
    if (isIdLast) {
      if (M === 'GET') return `${res} - ${vCn}详情`;
      if (M === 'PUT') return `${res} - ${vCn}更新`;
      if (M === 'DELETE') return `${res} - ${vCn}删除`;
      if (M === 'PATCH') return `${res} - ${vCn}部分更新`;
    }
    return `${res} - ${vCn}`;
  }

  const hasId = parts.some(s => s.startsWith(':'));

  if (isIdLast) {
    if (M === 'GET') return `${res} - 详情`;
    if (M === 'PUT') return `${res} - 更新`;
    if (M === 'DELETE') return `${res} - 删除`;
    if (M === 'PATCH') return `${res} - 部分更新`;
  }
  if (!hasId) {
    if (M === 'GET') return `${res} - 列表`;
    if (M === 'POST') return `${res} - 创建`;
    if (M === 'PUT') return `${res} - 批量更新`;
    if (M === 'DELETE') return `${res} - 批量删除`;
  }
  return `${res} - ${M}`;
}

const sqlEsc = s => s.replace(/'/g, "''");

const lines = readFileSync('/tmp/api_endpoints.txt', 'utf-8').split('\n').filter(Boolean);
const rows = lines.map(l => {
  const [method, path, group] = l.split('|');
  return { method, path, group, desc: parse(path, method, group) };
});

const out = [];
out.push('-- +goose Up');
out.push('-- 批量为 api_endpoints.description 补全（按 path+method+group 启发式推断）。');
out.push('-- 仅覆盖当前 description 为空的行；手工填入的描述不被覆盖。');
out.push('-- 数据由 scripts/gen_api_desc 派生（path+method 精确匹配），新增路由后再次同步即可。');
out.push('UPDATE api_endpoints SET description = CASE');
for (const r of rows) {
  out.push(`    WHEN method = '${r.method}' AND path = '${sqlEsc(r.path)}' THEN '${sqlEsc(r.desc)}'`);
}
out.push('    ELSE description');
out.push("END WHERE COALESCE(description, '') = '';");
out.push('');
out.push('-- +goose Down');
out.push('-- 回滚：仅清空本次脚本覆盖过的 path+method 行（避免影响手工记录）。');
out.push('UPDATE api_endpoints SET description = \'\' WHERE (method, path) IN (');
for (let i = 0; i < rows.length; i++) {
  const r = rows[i];
  const sep = i < rows.length - 1 ? ',' : '';
  out.push(`    ('${r.method}', '${sqlEsc(r.path)}')${sep}`);
}
out.push(');');

writeFileSync('/tmp/058.sql', out.join('\n') + '\n');

// Spot-check：打印每组前 3 条样本
console.error('rows:', rows.length);
const grouped = {};
for (const r of rows) (grouped[r.group] ||= []).push(r);
for (const g of Object.keys(grouped).sort()) {
  for (const r of grouped[g].slice(0, 3)) {
    console.error(`  ${r.group.padEnd(22)} ${r.method.padEnd(6)} ${r.path.padEnd(70)} → ${r.desc}`);
  }
}
