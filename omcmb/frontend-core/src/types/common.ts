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
}
