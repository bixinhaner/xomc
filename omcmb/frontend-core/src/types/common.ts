import type React from 'react';

export type DeviceType = 'eNB' | 'gNB' | 'CPE' | 'eGW' | 'all';
export type AlarmSeverity = 'critical' | 'major' | 'minor' | 'warning';
export type Locale = 'zh-CN' | 'en-US';
export type Theme = 'classic' | 'tech' | 'fresh' | 'cyberpunk' | 'minions' | 'tiffany' | 'rmb';
export type Timezone = 'UTC' | 'local';
export type SidebarPosition = 'left' | 'right' | 'top';
export type TabBarPosition = 'top' | 'bottom' | 'left';

export interface SelectOption {
  label: string;
  value: string | number;
  children?: SelectOption[];
  disabled?: boolean;
}

export interface TreeNode {
  key: string;
  title: string;
  children?: TreeNode[];
  isLeaf?: boolean;
  icon?: React.ReactNode;
  disabled?: boolean;
  selectable?: boolean;
}

export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

export interface TabItem {
  key: string;
  label: string;
  path: string;
  closable: boolean;
  // 当 label 为已本地化的纯文本（含动态片段如 SN/名称）时设为 true，
  // 渲染时跳过 react-intl 翻译，避免触发 missing translation 报错。
  labelRaw?: boolean;
  // 可翻译前缀的 i18n key；用于「详情 · 设备名」这类带动态后缀的标题在切语言时重算前缀。
  labelPrefixI18nKey?: string;
  // 标题中不可翻译的动态后缀；与 labelPrefixI18nKey 组合用于重算当前 locale 下的标题。
  labelSuffix?: string;
}
