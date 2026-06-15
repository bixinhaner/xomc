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
// LTE (eNB) KPI 配置 - 6个Panel
//
// KPI代码参考：data/indicator-library/enb/*.xml
// 与性能管理模块保持一致，直接使用KPI代码查询pm_metrics表
// ============================================================================

const LTE_CONFIG: TechnologyKPIConfig = {
  panels: ['traffic', 'availability', 'utilization', 'accessibility', 'retainability', 'mobility'],
  panelConfigs: {
    // ========== Panel 1: Traffic (业务量) ==========
    traffic: {
      title: 'dashboard.panel.traffic',
      indicators: [
        // K900010015: Data Volume DL (下行数据业务流量)
        { key: 'K900010015', label: 'dashboard.kpi.totalDataVolumeDl', color: '#1677FF', unit: 'unit.gb' },
        // K900010016: Data Volume UL (上行数据业务流量)
        { key: 'K900010016', label: 'dashboard.kpi.totalDataVolumeUl', color: '#10B981', unit: 'unit.gb' },
        // TODO: PDCP速率指标暂无对应KPI代码，待KPI模块补充后添加
        // { key: 'LTE_PDCP_RATE_DL', label: 'dashboard.kpi.throughputDl', color: '#1677FF', unit: 'unit.mbps' },
        // { key: 'LTE_PDCP_RATE_UL', label: 'dashboard.kpi.throughputUl', color: '#10B981', unit: 'unit.mbps' },
      ],
      defaultIndicator: 'K900010015',
    },

    // ========== Panel 2: Availability (可用性) ==========
    availability: {
      title: 'dashboard.panel.availability',
      indicators: [
        // K900010006: Initial establishment success rate (无线初始连接成功率)
        { key: 'K900010006', label: 'dashboard.kpi.cellAvailable', color: '#52C41A', unit: 'unit.percent' },
      ],
      defaultIndicator: 'K900010006',
    },

    // ========== Panel 3: Utilization (利用率) ==========
    utilization: {
      title: 'dashboard.panel.utilization',
      indicators: [
        // K900010014: Downlink PRB utilization rate (下行PRB平均占用率)
        { key: 'K900010014', label: 'dashboard.kpi.dlPrbUtilRate', color: '#FA8C16', unit: 'unit.percent' },
        // K900010013: Uplink PRB utilization rate (上行PRB平均占用率)
        { key: 'K900010013', label: 'dashboard.kpi.ulPrbUtilRate', color: '#EB2F96', unit: 'unit.percent' },
      ],
      defaultIndicator: 'K900010014',
    },

    // ========== Panel 4: Accessibility (接入性) ==========
    accessibility: {
      title: 'dashboard.panel.accessibility',
      indicators: [
        // K900010006: Initial establishment success rate (无线初始连接成功率)
        { key: 'K900010006', label: 'dashboard.kpi.wirelessSetupSr', color: '#52C41A', unit: 'unit.percent' },
        // K900010002: RRC setup success rate (RRC连接建立成功率)
        { key: 'K900010002', label: 'dashboard.kpi.rrcSetupSr', color: '#1677FF', unit: 'unit.percent' },
        // K900010005: E-RAB setup success rate (E-RAB建立成功率)
        { key: 'K900010005', label: 'dashboard.kpi.erabSetupSr', color: '#FAAD14', unit: 'unit.percent' },
        // K900010029: CSFB Success Rate (CSFB成功率)
        { key: 'K900010029', label: 'dashboard.kpi.csfbSr', color: '#EB2F96', unit: 'unit.percent' },
      ],
      defaultIndicator: 'K900010006',
    },

    // ========== Panel 5: Retainability (保持性) ==========
    retainability: {
      title: 'dashboard.panel.retainability',
      indicators: [
        // K900010027: E-RAB drop rate (E-RAB掉线率)
        { key: 'K900010027', label: 'dashboard.kpi.erabDropRate', color: '#F5222D', unit: 'unit.percent' },
      ],
      defaultIndicator: 'K900010027',
    },

    // ========== Panel 6: Mobility (移动性) ==========
    mobility: {
      title: 'dashboard.panel.mobility',
      indicators: [
        // K900010017: HO.IntraFreqOutSucc.Rate (同频切换成功率-切出)
        { key: 'K900010017', label: 'dashboard.kpi.hoIntraEnbOutSr', color: '#1677FF', unit: 'unit.percent' },
        // K900010022: HO IntraFreqInSucc Rate (同频切换成功率-切入)
        { key: 'K900010022', label: 'dashboard.kpi.hoIntraEnbInSr', color: '#10B981', unit: 'unit.percent' },
        // K900010021: HO InterEnbOutSucc Rate (eNB间切换成功率-切出)
        { key: 'K900010021', label: 'dashboard.kpi.hoInterEnbOutSr', color: '#FA8C16', unit: 'unit.percent' },
        // K900010026: HO InterEnbInSucc Rate (eNB间切换成功率-切入)
        { key: 'K900010026', label: 'dashboard.kpi.hoInterEnbInSr', color: '#EB2F96', unit: 'unit.percent' },
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
        // KGNB0511: PDCP下行业务字节数
        { key: 'KGNB0511', label: 'dashboard.kpi.totalDataVolumeDl', color: '#1677FF', unit: 'unit.gb' },
        // KGNB0510: PDCP上行业务字节数
        { key: 'KGNB0510', label: 'dashboard.kpi.totalDataVolumeUl', color: '#10B981', unit: 'unit.gb' },
        // KGNB0517: 下行用户平均速率
        { key: 'KGNB0517', label: 'dashboard.kpi.throughputDl', color: '#1677FF', unit: 'unit.mbps' },
        // KGNB0516: 上行用户平均速率
        { key: 'KGNB0516', label: 'dashboard.kpi.throughputUl', color: '#10B981', unit: 'unit.mbps' },
      ],
      defaultIndicator: 'KGNB0511',
    },

    // ========== Panel 2: Utilization (利用率) ==========
    utilization: {
      title: 'dashboard.panel.utilization',
      indicators: [
        // KGNB0506: 下行PRB平均利用率
        { key: 'KGNB0506', label: 'dashboard.kpi.dlPrbUtilRate', color: '#FA8C16', unit: 'unit.percent' },
        // KGNB0505: 上行PRB平均利用率
        { key: 'KGNB0505', label: 'dashboard.kpi.ulPrbUtilRate', color: '#EB2F96', unit: 'unit.percent' },
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
        // KGSM0102: CallSetupSuccRate (电话成功率)
        { key: 'KGSM0102', label: 'dashboard.kpi.callSetupSr', color: '#52C41A', unit: 'unit.percent' },
      ],
      defaultIndicator: 'KGSM0102',
    },

    // ========== Panel 2: Retainability (保持性) ==========
    retainability: {
      title: 'dashboard.panel.retainability',
      indicators: [
        // KGSM0103: CallDropRate (电话掉线率)
        { key: 'KGSM0103', label: 'dashboard.kpi.callDropRate', color: '#F5222D', unit: 'unit.percent' },
      ],
      defaultIndicator: 'KGSM0103',
    },

    // ========== Panel 3: Mobility (移动性) ==========
    mobility: {
      title: 'dashboard.panel.mobility',
      indicators: [
        // KGSM0101: HandoverSuccessRate (Handover切换成功率)
        { key: 'KGSM0101', label: 'dashboard.kpi.handoverSr', color: '#FA8C16', unit: 'unit.percent' },
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
 * 按 symbolic key 反查指标的展示配置（label/color/unit）。
 *
 * issue #213 S2：首页改读全局布局后，图要画的指标来自布局的 metrics（symbolic key），
 * 需据 key 取本地化标签 / 线色 / 单位。回退默认布局与全局配置的指标都在 kpi-config 里登记，
 * 故按 key 全表扫描即可命中。未登记的 key 返回 undefined（调用方兜底）。
 */
export function getKPIConfigByKey(key: string): KPIConfig | undefined {
  for (const tech of Object.keys(KPI_BY_TECH) as TechnologyType[]) {
    const config = KPI_BY_TECH[tech];
    for (const panelType of config.panels) {
      const found = config.panelConfigs[panelType]?.indicators.find((ind) => ind.key === key);
      if (found) return found;
    }
  }
  return undefined;
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
