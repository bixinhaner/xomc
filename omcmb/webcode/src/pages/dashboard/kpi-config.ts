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

import type { TechnologyType } from '@core/types/technology';
export type { TechnologyType } from '@core/types/technology';

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




function kpi(key: string, color: string): KPIConfig {
  return { key, color, label: '', unit: '' };
}

// ============================================================================
// LTE (eNB) KPI 配置 - 6个Panel
//
// 与性能管理模块保持一致，直接使用KPI代码查询pm_metrics表。
// indicators 只声明「key + color」，名字/单位统一从后端指标库取（见 useMetricMetadata）。
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
        kpi('K900010040', '#722ED1'),  // PDCP 速率 DL
        kpi('K900010041', '#13C2C2'),  // PDCP 速率 UL
      ],
      defaultIndicator: 'K900010015',
    },

    // ========== Panel 2: Availability (可用性) ==========
    availability: {
      title: 'dashboard.panel.availability',
      indicators: [
        kpi('K900010006', '#52C41A'),  // 小区可用性，名字来自指标库
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
