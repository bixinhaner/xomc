// KPI 指标库类型（T-0098-P4 数据字典平台化）
// 后端：internal/pm/indicator/rest_handler.go IndicatorInfo / GroupInfo / Unit

export type DeviceType = 'ENB' | 'GSM' | 'GNB';

export interface IndicatorInfo {
  id: string;
  name: string;
  cnName?: string;
  enName?: string;
  groupId?: string;
  groupName?: string;
  counterType?: string;
  indicatorLevel?: string;       // ENB only, GNB 无此字段
  unit?: string;
  description?: string;
  isCounter?: boolean;
  // 统计类型（对齐老 OMC perf_indicators.statis_type）。取值枚举见 STATIS_TYPE_VALUES；
  // KPI 新建/编辑表单用下拉渲染。后端 BackendIndicator.statis_type 下发，mapIndicator 透传。
  statisType?: string;
  // PM-P3:编号版公式(perf_indicators_*.arithmetic)。派生 KPI 为编号算术式(如
  // (C000060011+C000060022)/1000),原始计数为自身编号。界面公式展示用此字段
  // (对运维编号才是工作语言);标准名版 formula 表退为工程内部物,不再用于展示。
  arithmetic?: string;
  productClass?: string;
  operatorCode?: string;
  isEnabled?: boolean;
  deviceType: DeviceType;
  // 内置指标判定（后端 is_build_in === '1'）：编辑时「归属分组」只读（XML 真相源覆盖）。
  isBuildIn?: boolean;
}

export interface IndicatorListFilter {
  groupId?: string;
  keyword?: string;
  operatorCode?: string;
  productClass?: string;
  indicatorLevel?: string;
  isEnabled?: boolean;
  isCounter?: boolean;
  platformName?: string;
  page?: number;
  pageSize?: number;
}

export interface IndicatorGroup {
  id: string;
  name: string;
  parentId?: string;
  description?: string;
  operatorCode?: string;
  deviceType: DeviceType;
  // 内置组判定（后端 is_build_in === '1'）：内置组禁删、禁编辑。
  isBuildIn?: boolean;
  children?: IndicatorGroup[];
}

export interface PlatformFormula {
  platformName: string;
  indicatorId: string;
  formula: string;
  description?: string;
}

export interface CreateIndicatorInput {
  name: string;
  // create 接口由后端生成真实 ID。id 仅为兼容历史调用方/测试输入，真实 API 序列化会丢弃。
  id?: string;
  cnName?: string;
  enName?: string;
  groupId?: string;
  counterType?: string;
  indicatorLevel?: string;
  unit?: string;
  description?: string;
  productClass?: string;
  operatorCode?: string;
  // 后端 CreateIndicatorRequest 直收字段（model.go）：data_type / statis_type /
  // arithmetic / is_counter（'0'|'1' 字符串，与 mapIndicator 归一化一致）/
  // en_description / cn_description。
  dataType?: string;
  statisType?: string;
  arithmetic?: string;
  isCounter?: string;
  enDescription?: string;
  cnDescription?: string;
  // 详情态（URL ?platform= 锁定）新建必须把当前 platform 透传给后端，否则
  // perf_formulas_<dt> 无关联行 → 列表 EXISTS 过滤会把刚建的指标过滤掉，
  // 表现为"保存成功但查不到"。列表态（主页全局新建）不传，保持原行为。
  //
  // 与 formulas 并存（向后兼容）：当 formulas 非空时，service 优先走批量分支并忽略 platform；
  // formulas 为空但 platform 非空时，service 走旧的单条占位逻辑（保持已部署旧前端的行为）。
  platform?: string;
  // formulas：issue #640 C 方案，新建时一并提交的多平台公式集合。
  // 非空时后端在同事务内为每条 (platformName, formula) 跑 FormulaValidator + BatchCreate；
  // 任一公式语法非法或事务任一环节失败 → 整体回滚（指标 + 已批公式都不落库）。
  // 同 platformName 重复在前端阻断，service 层兜底校验。
  formulas?: FormulaDraft[];
}

// FormulaDraft 是新建态本地草稿条目，对齐后端 FormulaInput（json: platform_name / formula）。
// 与持久化的 PlatformFormula 区分：drafts 还没 indicatorId（尚未创建），只在 Modal 内 state 持有。
export interface FormulaDraft {
  platformName: string;
  formula: string;
}

// platform 仅 Create 路径有意义（创建时同事务写占位 formula）；后端
// UpdateIndicatorRequest 无 Platform 字段，update 不能也不应携带 platform。
export type UpdateIndicatorInput = Partial<Omit<CreateIndicatorInput, 'id' | 'platform'>>;

// 统计类型枚举（与后端 perf_indicators.statis_type / pm_metrics.statis_type CHECK 约束对齐）。
// UI 下拉选择项取该数组；各皮肤渲染时可用 i18n key `product.kpi.indicator.statisType.<value>` 本地化。
export const STATIS_TYPE_VALUES = ['sum', 'avg', 'max', 'min', 'pct'] as const;
export type StatisTypeValue = (typeof STATIS_TYPE_VALUES)[number];

// 指标单位字典（对齐老 OMC indicator_unit 表 + 现有 data/indicator-library/*.xml unitId 全集）。
// UI 「单位」字段渲染为下拉，选项值与后端 unit_id 直传一致；显示文本即为值本身（如 "%" / "Byte/s" /
// "Kbps"），不做本地化（运维语言）。如需追加新单位，扩展本数组即可——保持单一事实源。
export const INDICATOR_UNIT_OPTIONS = [
  '%',
  'bit',
  'Byte',
  'Byte/s',
  'char',
  'dBm',
  'Erl',
  'Gbit',
  'GByte',
  'Kb/PRB',
  'Kbit',
  'KByte',
  'KByte/s',
  'Kbps',
  'MByte',
  'Mbps',
  'milliseconds',
  'ms',
  'no',
  'number',
  'ppm',
  's',
  'seconds',
  'time',
  'W',
] as const;
export type IndicatorUnitValue = (typeof INDICATOR_UNIT_OPTIONS)[number];

// 指标等级（对齐老 OMC perf_indicators.indicator_level）：仅 device / plmn 两个对外可选项；
// 老数据存量值 'both' 仍允许回显（UI 用「额外当前值并入选项」模式兼容），新建/编辑只能从这两项中选。
export const INDICATOR_LEVEL_OPTIONS = [
  { value: 'device', label: 'Device' },
  { value: 'plmn', label: 'PLMN' },
] as const;
export type IndicatorLevelValue = (typeof INDICATOR_LEVEL_OPTIONS)[number]['value'];

// 指标类型（替代旧「计数器」Switch，命名更贴用户语义 — 数据来源维度）：
//   counter = 直接采集：设备 PM 文件上报的原始计数器（落库即用，arithmetic 一般为空）
//   kpi     = 公式计算：派生 KPI，必须有 arithmetic 编号公式（如 (C000060011+C000060022)/1000）
// 与后端字段映射：counter ⇄ is_counter='1'；kpi ⇄ is_counter='0'。
// 注：后端 service.CreateIndicator 会按公式自动校验 — 若 kpi 类型的 arithmetic 解析后只引用单个
//   指标 ID 且无运算符，会自动改回 is_counter='1'（这是合理的双保险，避免用户误把原始计数当 KPI）。
export const INDICATOR_TYPE_OPTIONS = [
  { value: 'counter', isCounter: '1' as const },
  { value: 'kpi', isCounter: '0' as const },
] as const;
export type IndicatorTypeValue = (typeof INDICATOR_TYPE_OPTIONS)[number]['value'];

export interface CreateGroupInput {
  name: string;
  // create 接口由后端生成真实 ID。id 仅为兼容历史调用方/测试输入，真实 API 序列化会丢弃。
  id?: string;
  parentId?: string;
  description?: string;
  operatorCode?: string;
}

export type UpdateGroupInput = Partial<Omit<CreateGroupInput, 'id'>>;

export interface EnabledIndicatorsRequest {
  deviceType: DeviceType;
  operatorCode: string;
  indicatorIds: string[];
  enable: boolean;
}


// ── T-0180 P4: 自定义 XML 分层目录 + drill-down 视图类型 ──────────────────

// 后端 ?tech= query 用 enb/gsm/gnb 小写;前端 DeviceType 用 ENB/GSM/GNB 大写
// (历史保留)。techFromDeviceType 在 API 层做映射,UI 用 DeviceType 即可。
export type TechLower = 'enb' | 'gsm' | 'gnb';

export function deviceTypeToTech(dt: DeviceType): TechLower {
  return dt.toLowerCase() as TechLower;
}

export function techToDeviceType(t: TechLower): DeviceType {
  return t.toUpperCase() as DeviceType;
}

// 设备类型 → 制式码（对齐 devices.technology 小写）：eNB=LTE / gNB=NR(5G) / GSM=2G。
// 注意与 deviceTypeToTech 区分：那个返回设备类型小写（enb/gnb/gsm，给指标库 ?tech= 用），
// 本函数返回网络制式码（lte/nr/gsm，给设备清单 networkType / DevicePickerModal.technology 用）。
export type NetworkTech = 'lte' | 'nr' | 'gsm';

const DEVICE_TYPE_TO_NETWORK_TECH: Record<DeviceType, NetworkTech> = {
  ENB: 'lte',
  GNB: 'nr',
  GSM: 'gsm',
};

export function deviceTypeToNetworkTech(dt: DeviceType): NetworkTech {
  return DEVICE_TYPE_TO_NETWORK_TECH[dt] ?? 'lte';
}

// IndicatorSource 与 ParamModelSource 同枚举(builtin/custom/unknown);
// 后端 source.go::ClassifySource 派生,前端 SummaryTab 渲染"来源"列 + 守门删除。
export type IndicatorSource = 'builtin' | 'custom' | 'unknown';

// IndicatorPlatformSummary 是 /indicators/summary 端点单行 — (制式, 平台) 二元组。
// 一个 XML 文件即一个平台,一级列表"一个平台一条"。
// loadedFrom/source:后端 MAX(formula.loaded_from) + ClassifySource 派生,
// 一级列表渲染"加载源/来源"两展示列(删除操作仍留在 XMLFilesModal)。
export interface IndicatorPlatformSummary {
  tech: TechLower;
  platform: string;       // platform_name(rela_platform_indicator_formula_*.platform_name)
  indicators: number;     // 该平台的指标计数
  loadedFrom: string;     // 该平台对应的 XML 文件相对路径(后端 MAX(formula.loaded_from))
  source: IndicatorSource;// builtin/custom/unknown(后端由 sidecar 派生)
  deletable: boolean;     // 一级列表删除按钮可见性(后端 IsDeletable;内置置灰,2026-06-05)
  description?: string;   // 按 (tech, platform) 维度的可编辑描述
}

// IndicatorFile 是 /indicators/files?tech= 单行 — 含 source/deletable 派生 + DB 计数。
export interface IndicatorFile {
  loadedFrom: string;
  source: 'builtin' | 'custom' | 'unknown';
  deletable: boolean;
  count: number;
  onDisk: boolean;
}

export interface IndicatorUploadResult {
  uploaded: boolean;
  filename: string;
  loadedFrom: string;
  tech: TechLower;
  platform: string;     // 内容主键(<indicatorModel platform="...">)
  overwritten: boolean; // force 覆盖了既有文件(旧文件已备份 .bak.<ts>)
  reloaded: boolean;    // 同步触发 Loader.Reload 是否成功
}

export interface IndicatorDeleteFileResult {
  deleted: boolean;
  loadedFrom: string;
  tech: TechLower;
  rowsAffected: number;
  backup: string;     // ".deleted.<ts>" 文件名,空串=无备份(文件已 gone)
}
