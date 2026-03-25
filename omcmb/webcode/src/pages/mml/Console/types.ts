import type { MMLCommand } from '@/types/mml';

// 设备类型
export interface ConsoleDevice {
  sn: string;
  name: string;
  type: string;
  productType: string;
  status: 'online' | 'offline' | 'alarm';
}

// 设备状态颜色映射
export const STATUS_COLORS: Record<string, string> = {
  online: '#52c41a',
  offline: '#d9d9d9',
  alarm: '#fa8c16',
};

// 产品类型选项
export const PRODUCT_TYPE_OPTIONS = [
  { label: 'PM-B4860', value: 'PM-B4860' },
  { label: 'QAFA', value: 'QAFA' },
  { label: 'BaiBNX', value: 'BaiBNX' },
  { label: 'BaiBS5163', value: 'BaiBS5163' },
  { label: 'BaiBS5263', value: 'BaiBS5263' },
  { label: 'BTS', value: 'BTS' },
  { label: 'BSC', value: 'BSC' },
];

// 终端行类型
export interface TerminalLine {
  text: string;
  type?: 'stdout' | 'stderr' | 'info' | 'success';
  timestamp?: string;
}

// MML 控制台状态
export interface MMLConsoleState {
  selectedDevices: ConsoleDevice[];
  selectedCommand: MMLCommand | null;
  outputLines: TerminalLine[];
  paramValues: Record<string, string | number | boolean>;
}

// 组件 Props 类型
export interface DeviceTreeProps {
  selectedDevices: ConsoleDevice[];
  onSelectionChange: (devices: ConsoleDevice[]) => void;
  onBatchInput: () => void;
}

export interface CommandTreeProps {
  selectedCommand: MMLCommand | null;
  onSelectionChange: (command: MMLCommand | null) => void;
  commands: MMLCommand[];
}

export interface TerminalPanelProps {
  lines: TerminalLine[];
  onClear: () => void;
  onDownload: () => void;
}

export interface CommandInputProps {
  selectedDevices: ConsoleDevice[];
  selectedCommand: MMLCommand | null;
  paramValues: Record<string, string | number | boolean>;
  onExecute: () => void;
  loading?: boolean;
}

export interface ParamConfigPanelProps {
  command: MMLCommand | null;
  values: Record<string, string | number | boolean>;
  onChange: (values: Record<string, string | number | boolean>) => void;
}
