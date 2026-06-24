/**
 * Dashboard KPI Configuration
 *
 * 基于老系统设计重构，支持三种制式的完整Panel结构：
 * - LTE (eNB): 6个Panel (Traffic/Availability/Utilization/Accessibility/Retainability/Mobility)
 * - NR (gNB): 2个Panel (Traffic/Utilization)
 * - GSM: 3个Panel (Accessibility/Retainability/Mobility)
 *
 * 每个Panel支持：
 * - 指标下拉选择
 * - Day/Week视图切换
 * - Today/Yesterday时间对比
 */

// ============================================================================
// Type Definitions
// ============================================================================

export type TechnologyType = 'lte' | 'nr' | 'gsm';

/**
 * KPI单个指标配置
 */
export interface KPIConfig {
  /** KPI key/identifier (backend KPI name) */
  key: string;
  /** Display label (i18n key) */
  label: string;
  /** Chart line color (hex) */
  color: string;
  /** Unit of measurement (i18n key) */
  unit: string;
  /** Unit conversion factor (backend value -> display value) */
  unitConversion?: number;
}

/**
 * Panel类型 - 对应老系统的6种Panel
 */
export type PanelType =
  | 'traffic'           // 业务量/流量
  | 'availability'      // 可用性
  | 'utilization'       // 利用率
  | 'accessibility'     // 接入性
  | 'retainability'     // 保持性
  | 'mobility';         // 移动性

/**
 * 视图模式 - Day/Week切换
 */
export type ViewMode = 'day' | 'week';

/**
 * 时间对比选项 - Today/Yesterday（多选模式）
 */
export type TimeRangeOption = 'today' | 'yesterday';

/**
 * 单个Panel的配置
 */
export interface PanelConfig {
  /** Panel显示名称（i18n key） */
  title: string;
  /** 该Panel支持的指标选项列表 */
  indicators: KPIConfig[];
  /** 默认选中的指标key */
  defaultIndicator: string;
}

/**
 * 单个制式的完整KPI配置
 */
export interface TechnologyKPIConfig {
  /** 该制式支持的Panel列表及顺序 */
  panels: PanelType[];
  /** 各Panel的具体配置（只包含该制式支持的Panel） */
  panelConfigs: Partial<Record<PanelType, PanelConfig>>;
}

// ============================================================================
// 常量定义
// ============================================================================

/**
 * 制式显示名称
 */
export const TECH_LABELS: Record<TechnologyType, string> = {
  lte: 'LTE',
  nr: 'NR',
  gsm: 'GSM',
} as const;

/**
 * Panel类型显示名称
 */
export const PANEL_LABELS: Record<PanelType, string> = {
  traffic: 'dashboard.panel.traffic',
  availability: 'dashboard.panel.availability',
  utilization: 'dashboard.panel.utilization',
  accessibility: 'dashboard.panel.accessibility',
  retainability: 'dashboard.panel.retainability',
  mobility: 'dashboard.panel.mobility',
} as const;

/**
 * 视图模式选项
 */
export const VIEW_MODE_OPTIONS = [
  { label: 'dashboard.viewMode.day', value: 'day' },
  { label: 'dashboard.viewMode.week', value: 'week' },
] as const;

/**
 * 时间对比选项
 */
export const TIME_RANGE_OPTIONS = [
  { label: 'dashboard.timeRange.today', value: 'today' },
  { label: 'dashboard.timeRange.yesterday', value: 'yesterday' },
] as const;

// ============================================================================
// 指标展示元数据 —— 单一数据源（KPI Catalog）
//
// 维护点：要新增 / 修改某个 KPI 的中英文名、单位、换算系数，**只改这一处**。
// 主键约定：能用稳定的指标编号（K/C 开头）就用编号；某些指标暂无对应编号
//          （如 LTE_PDCP_RATE_DL/UL），直接以 symbolic 注册为主键。
// 兼容层：fix(#227) 之前 DB 还存 symbolic 别名，由 LEGACY_KEY_ALIASES 桥接到主键。
// ============================================================================

/** KPI 展示元数据条目：i18n 标签 + i18n 单位 + 数值换算系数。 */
export interface KPIDisplayMeta {
  /** i18n key（dashboard.kpi.*）。 */
  label: string;
  /** i18n unit key（unit.*）；无单位省略。 */
  unit?: string;
  /** 后端原始值 → 展示值的乘数；默认 1。 */
  unitConversion?: number;
}

/**
 * 指标展示元数据「单一数据源」。
 *
 * 一个 KPI 编号在不同 panel 想显示不同名字（如 K900010006 在 availability /
 * accessibility 下语义有别）时：catalog 给「默认 / 主语义」名，具体 panel 调用
 * `kpi(key, color, { label })` 做局部 override。
 */
const KPI_CATALOG: Readonly<Record<string, KPIDisplayMeta>> = {
  // ===== LTE (eNB) =====
  // 参考：data/indicator-library/enb/*.xml
  K900010002: { label: 'dashboard.kpi.rrcSetupSr',        unit: 'unit.percent' },
  K900010005: { label: 'dashboard.kpi.erabSetupSr',       unit: 'unit.percent' },
  K900010006: { label: 'dashboard.kpi.wirelessSetupSr',   unit: 'unit.percent' },
  K900010013: { label: 'dashboard.kpi.ulPrbUtilRate',     unit: 'unit.percent' },
  K900010014: { label: 'dashboard.kpi.dlPrbUtilRate',     unit: 'unit.percent' },
  K900010015: { label: 'dashboard.kpi.totalDataVolumeDl', unit: 'unit.gb' },
  K900010016: { label: 'dashboard.kpi.totalDataVolumeUl', unit: 'unit.gb' },
  K900010017: { label: 'dashboard.kpi.hoIntraEnbOutSr',   unit: 'unit.percent' },
  K900010021: { label: 'dashboard.kpi.hoInterEnbOutSr',   unit: 'unit.percent' },
  K900010022: { label: 'dashboard.kpi.hoIntraEnbInSr',    unit: 'unit.percent' },
  K900010026: { label: 'dashboard.kpi.hoInterEnbInSr',    unit: 'unit.percent' },
  K900010027: { label: 'dashboard.kpi.erabDropRate',      unit: 'unit.percent' },
  K900010029: { label: 'dashboard.kpi.csfbSr',            unit: 'unit.percent' },
  // 无 K 编号的 LTE 指标（指标库待补，目前 symbolic 即主键，避免下拉显示 raw key）
  LTE_PDCP_RATE_DL: { label: 'dashboard.kpi.throughputDl', unit: 'unit.mbps' },
  LTE_PDCP_RATE_UL: { label: 'dashboard.kpi.throughputUl', unit: 'unit.mbps' },

  // ===== NR (gNB) =====
  // 参考：data/indicator-library/GNB.xml
  KGNB0505: { label: 'dashboard.kpi.ulPrbUtilRate',     unit: 'unit.percent' },
  KGNB0506: { label: 'dashboard.kpi.dlPrbUtilRate',     unit: 'unit.percent' },
  KGNB0510: { label: 'dashboard.kpi.totalDataVolumeUl', unit: 'unit.gb' },
  KGNB0511: { label: 'dashboard.kpi.totalDataVolumeDl', unit: 'unit.gb' },
  KGNB0516: { label: 'dashboard.kpi.throughputUl',      unit: 'unit.mbps' },
  KGNB0517: { label: 'dashboard.kpi.throughputDl',      unit: 'unit.mbps' },

  // ===== GSM =====
  // 参考：data/indicator-library/GSM.xml
  KGSM0101: { label: 'dashboard.kpi.handoverSr',   unit: 'unit.percent' },
  KGSM0102: { label: 'dashboard.kpi.callSetupSr',  unit: 'unit.percent' },
  KGSM0103: { label: 'dashboard.kpi.callDropRate', unit: 'unit.percent' },
};

/**
 * 旧 symbolic 别名 → 稳定指标编号（fix(#227) 持久化层未迁完的兼容桥）。
 *
 * 来源：早于 fix(#227) 的版本里 `dashboard_kpi_layouts` 表与后端 defaultLayoutJSON
 * 仍存 symbolic key。前端读到时把它规整成 catalog 主键再取展示元数据。
 *
 * `label` 是可选 override：当旧 alias 在其历史语境下的名字与主键默认名字不一致时使用
 * （如 `LTE_CELL_AVAILABLE` 叫「小区可用性」，而主键 K900010006 默认叫「无线初始
 * 连接成功率」）。不传表示跟随主键。
 *
 * 后端持久化层完全迁完之后本表可整张删除。
 */
interface LegacyAliasEntry {
  /** 桥接到的 catalog 主键。 */
  kCode: string;
  /** alias 历史语境下的 i18n label；不传为 undefined 时跟随主键 catalog 默认。 */
  label?: string;
}

const LEGACY_KEY_ALIASES: Readonly<Record<string, LegacyAliasEntry>> = {
  // LTE
  LTE_PDCP_VOLUME_DL:  { kCode: 'K900010015' },
  LTE_PDCP_VOLUME_UL:  { kCode: 'K900010016' },
  // LTE_CELL_AVAILABLE 与 WIRELESS_SETUP_SR 同指 K900010006，但历史上前者表「小区可用性」
  // 后者表「无线初始连接成功率」，在 catalog 主键默认名（wirelessSetupSr）之外额外为
  // 前者 override，保存原语义。
  LTE_CELL_AVAILABLE:  { kCode: 'K900010006', label: 'dashboard.kpi.cellAvailable' },
  WIRELESS_SETUP_SR:   { kCode: 'K900010006' },
  LTE_PRB_UTIL_DL:     { kCode: 'K900010014' },
  LTE_PRB_UTIL_UL:     { kCode: 'K900010013' },
  RRC_CONN_SETUP_SR:   { kCode: 'K900010002' },
  ERAB_SETUP_SR:       { kCode: 'K900010005' },
  CSFB_SR:             { kCode: 'K900010029' },
  ERAB_DROP_RATE:      { kCode: 'K900010027' },
  HO_INTRA_ENB_OUT_SR: { kCode: 'K900010017' },
  HO_INTRA_ENB_IN_SR:  { kCode: 'K900010022' },
  HO_INTER_ENB_OUT_SR: { kCode: 'K900010021' },
  HO_INTER_ENB_IN_SR:  { kCode: 'K900010026' },
  // NR
  NR_PDCP_VOLUME_DL: { kCode: 'KGNB0511' },
  NR_PDCP_VOLUME_UL: { kCode: 'KGNB0510' },
  NR_PDCP_RATE_DL:   { kCode: 'KGNB0517' },
  NR_PDCP_RATE_UL:   { kCode: 'KGNB0516' },
  NR_PRB_UTIL_DL:    { kCode: 'KGNB0506' },
  NR_PRB_UTIL_UL:    { kCode: 'KGNB0505' },
  // GSM
  GSM_CALL_SETUP_SR:  { kCode: 'KGSM0102' },
  GSM_CALL_DROP_RATE: { kCode: 'KGSM0103' },
  GSM_HO_SR:          { kCode: 'KGSM0101' },
};

/**
 * 把任意 metric key 规整成 catalog 主键。
 *
 * 调用方拿到的 key 可能是：稳定指标编号（已是主键） / 旧 symbolic 别名 / 完全未登记的新 key。
 * 通过 alias 命中的桥到主键，否则原样返回（让上层继续兜底）。
 */
export function canonicalizeMetricKey(key: string): string {
  return LEGACY_KEY_ALIASES[key]?.kCode ?? key;
}

/**
 * 查询指标展示元数据；未登记返回 undefined。
 *
 * alias 带 label override 时以 alias label 覆盖主键默认 label；unit / conversion 始终跟主键。
 */
export function getKPIDisplayMeta(key: string): KPIDisplayMeta | undefined {
  const alias = LEGACY_KEY_ALIASES[key];
  const base = KPI_CATALOG[alias?.kCode ?? key];
  if (!base) return undefined;
  return alias?.label ? { ...base, label: alias.label } : base;
}

/**
 * 构造一条 panel 内的 KPI 指标配置。
 *
 * `key` / `color` 由 panel 决定（同一指标在不同图配色可不同），label/unit 默认从
 * KPI_CATALOG 取；仅特例（同编号在不同 panel 显示不同名）才传 override.label。
 */
function kpi(
  key: string,
  color: string,
  override?: { label?: string; unit?: string; unitConversion?: number },
): KPIConfig {
  const meta = getKPIDisplayMeta(key);
  return {
    key,
    color,
    label: override?.label ?? meta?.label ?? key,
    unit: override?.unit ?? meta?.unit ?? '',
    unitConversion: override?.unitConversion ?? meta?.unitConversion,
  };
}

// ============================================================================
// LTE (eNB) KPI 配置 - 6个Panel
//
// 与性能管理模块保持一致，直接使用KPI代码查询pm_metrics表。
// indicators 只声明「key + color」，label/unit 通过 kpi() 从 KPI_CATALOG 注入。
// ============================================================================

const LTE_CONFIG: TechnologyKPIConfig = {
  panels: ['traffic', 'availability', 'utilization', 'accessibility', 'retainability', 'mobility'],
  panelConfigs: {
    // ========== Panel 1: Traffic (业务量) ==========
    traffic: {
      title: 'dashboard.panel.traffic',
      indicators: [
        kpi('K900010015', '#1677FF'),  // Data Volume DL
        kpi('K900010016', '#10B981'),  // Data Volume UL
        // TODO: PDCP 速率指标待 KPI 模块补 K 编号；目前以 symbolic 走 catalog。
        // kpi('LTE_PDCP_RATE_DL', '#1677FF'),
        // kpi('LTE_PDCP_RATE_UL', '#10B981'),
      ],
      defaultIndicator: 'K900010015',
    },

    // ========== Panel 2: Availability (可用性) ==========
    availability: {
      title: 'dashboard.panel.availability',
      indicators: [
        // K900010006 catalog 默认名为 wirelessSetupSr（接入侧主语义），availability
        // panel 沿用历史「小区可用性」语义，本地 override label。
        kpi('K900010006', '#52C41A', { label: 'dashboard.kpi.cellAvailable' }),
      ],
      defaultIndicator: 'K900010006',
    },

    // ========== Panel 3: Utilization (利用率) ==========
    utilization: {
      title: 'dashboard.panel.utilization',
      indicators: [
        kpi('K900010014', '#FA8C16'),  // 下行 PRB 平均占用率
        kpi('K900010013', '#EB2F96'),  // 上行 PRB 平均占用率
      ],
      defaultIndicator: 'K900010014',
    },

    // ========== Panel 4: Accessibility (接入性) ==========
    accessibility: {
      title: 'dashboard.panel.accessibility',
      indicators: [
        kpi('K900010006', '#52C41A'),  // 无线初始连接成功率
        kpi('K900010002', '#1677FF'),  // RRC 连接建立成功率
        kpi('K900010005', '#FAAD14'),  // E-RAB 建立成功率
        kpi('K900010029', '#EB2F96'),  // CSFB 成功率
      ],
      defaultIndicator: 'K900010006',
    },

    // ========== Panel 5: Retainability (保持性) ==========
    retainability: {
      title: 'dashboard.panel.retainability',
      indicators: [
        kpi('K900010027', '#F5222D'),  // E-RAB 掉线率
      ],
      defaultIndicator: 'K900010027',
    },

    // ========== Panel 6: Mobility (移动性) ==========
    mobility: {
      title: 'dashboard.panel.mobility',
      indicators: [
        kpi('K900010017', '#1677FF'),  // 同频切换成功率-切出
        kpi('K900010022', '#10B981'),  // 同频切换成功率-切入
        kpi('K900010021', '#FA8C16'),  // eNB 间切换成功率-切出
        kpi('K900010026', '#EB2F96'),  // eNB 间切换成功率-切入
      ],
      defaultIndicator: 'K900010017',
    },
  },
};

// ============================================================================
// NR (gNB) KPI 配置 - 2个Panel
//
// KPI代码参考：data/indicator-library/GNB.xml
// 与性能管理模块保持一致，直接使用KPI代码查询pm_metrics表
// ============================================================================

const NR_CONFIG: TechnologyKPIConfig = {
  panels: ['traffic', 'utilization'],
  panelConfigs: {
    // ========== Panel 1: Traffic (业务量) ==========
    traffic: {
      title: 'dashboard.panel.traffic',
      indicators: [
        kpi('KGNB0511', '#1677FF'),  // PDCP 下行业务字节数
        kpi('KGNB0510', '#10B981'),  // PDCP 上行业务字节数
        kpi('KGNB0517', '#1677FF'),  // 下行用户平均速率
        kpi('KGNB0516', '#10B981'),  // 上行用户平均速率
      ],
      defaultIndicator: 'KGNB0511',
    },

    // ========== Panel 2: Utilization (利用率) ==========
    utilization: {
      title: 'dashboard.panel.utilization',
      indicators: [
        kpi('KGNB0506', '#FA8C16'),  // 下行 PRB 平均利用率
        kpi('KGNB0505', '#EB2F96'),  // 上行 PRB 平均利用率
      ],
      defaultIndicator: 'KGNB0506',
    },
  },
};

// ============================================================================
// GSM KPI 配置 - 3个Panel
//
// KPI代码参考：data/indicator-library/GSM.xml
// 与性能管理模块保持一致，直接使用KPI代码查询pm_metrics表
// ============================================================================

const GSM_CONFIG: TechnologyKPIConfig = {
  panels: ['accessibility', 'retainability', 'mobility'],
  panelConfigs: {
    // ========== Panel 1: Accessibility (接入性) ==========
    accessibility: {
      title: 'dashboard.panel.accessibility',
      indicators: [
        kpi('KGSM0102', '#52C41A'),  // 电话成功率
      ],
      defaultIndicator: 'KGSM0102',
    },

    // ========== Panel 2: Retainability (保持性) ==========
    retainability: {
      title: 'dashboard.panel.retainability',
      indicators: [
        kpi('KGSM0103', '#F5222D'),  // 电话掉线率
      ],
      defaultIndicator: 'KGSM0103',
    },

    // ========== Panel 3: Mobility (移动性) ==========
    mobility: {
      title: 'dashboard.panel.mobility',
      indicators: [
        kpi('KGSM0101', '#FA8C16'),  // Handover 切换成功率
      ],
      defaultIndicator: 'KGSM0101',
    },
  },
};

// ============================================================================
// 统一配置导出
// ============================================================================

/**
 * 按制式索引的KPI配置
 */
export const KPI_BY_TECH: Record<TechnologyType, TechnologyKPIConfig> = {
  lte: LTE_CONFIG,
  nr: NR_CONFIG,
  gsm: GSM_CONFIG,
};

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * 获取指定制式支持的Panel列表
 */
export function getPanelsForTech(tech: TechnologyType): PanelType[] {
  return KPI_BY_TECH[tech]?.panels || [];
}

/**
 * 获取指定制式和Panel类型的配置
 */
export function getPanelConfig(
  tech: TechnologyType,
  panelType: PanelType
): PanelConfig | undefined {
  return KPI_BY_TECH[tech]?.panelConfigs[panelType];
}

/**
 * 获取指定制式支持的所有KPI名称（用于批量查询）
 */
export function getAllKPIKeysForTech(tech: TechnologyType): string[] {
  const config = KPI_BY_TECH[tech];
  if (!config) return [];

  const keys = new Set<string>();
  config.panels.forEach(panelType => {
    const panelConfig = config.panelConfigs[panelType];
    panelConfig?.indicators.forEach(indicator => {
      keys.add(indicator.key);
    });
  });

  return Array.from(keys);
}

/**
 * 根据制式获取Panel布局配置
 * 返回每行的Panel列表，用于渲染网格布局
 */
export function getPanelLayout(tech: TechnologyType): PanelType[][] {
  const panels = getPanelsForTech(tech);

  if (tech === 'lte') {
    // LTE: 2×3布局（每行2个，共3行）
    const layout = [
      [panels[0], panels[1]],  // Traffic, Availability
      [panels[2], panels[3]],  // Utilization, Accessibility
      [panels[4], panels[5]],  // Retainability, Mobility
    ];
    return layout;
  } else if (tech === 'nr') {
    // NR: 1×2布局（一行2个）
    return [
      [panels[0], panels[1]],  // Traffic, Utilization
    ];
  } else {
    // GSM: 第一行2个，第二行1个占满
    return [
      [panels[0], panels[1]],  // Accessibility, Retainability
      [panels[2]],            // Mobility（占满整行）
    ];
  }
}
