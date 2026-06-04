import type { MMLOperationType } from '@core/types/mml';
import type { ExecStatus } from './types';

// 操作类型彩色 Tag —— 与现有 MML 控制台 CommandTree.tsx 的配色保持一致，
// 让用户在两个版本间辨识 LST(读)/MOD(改)/ADD(增)/RMV(删) 的危险等级心智不变。
export const OP_COLORS: Record<string, string> = {
  LST: 'blue',
  MOD: 'orange',
  ADD: 'green',
  RMV: 'red',
  DSP: 'cyan',
  ACT: 'geekblue',
  DEA: 'gold',
  RST: 'volcano',
  CLR: 'magenta',
  UPG: 'purple',
};

export const OP_LABELS: Record<string, string> = {
  LST: '查询',
  MOD: '修改',
  ADD: '新增',
  RMV: '删除',
  DSP: '显示',
  ACT: '激活',
  DEA: '去激活',
  RST: '复位',
  CLR: '清除',
  UPG: '升级',
};

export function opLabel(op: MMLOperationType | string | undefined): string {
  if (!op) return '-';
  return OP_LABELS[op] ?? op;
}

export function opColor(op: MMLOperationType | string | undefined): string {
  if (!op) return 'default';
  return OP_COLORS[op] ?? 'default';
}

/** 是否「读」类操作（结果以参数值矩阵呈现）。其余为「写」类（状态 + 故障矩阵）。 */
export function isReadOp(op: MMLOperationType | string | undefined): boolean {
  return op === 'LST' || op === 'DSP';
}

/** 执行状态展示元数据（颜色 + 文案）。 */
export const STATUS_META: Record<ExecStatus, { color: string; text: string }> = {
  pending: { color: 'default', text: '待执行' },
  running: { color: 'processing', text: '执行中' },
  success: { color: 'success', text: '成功' },
  failed: { color: 'error', text: '失败' },
};

/** 设备弹框服务端分页每页条数（mock 沿用现控制台 50 条上限）。 */
export const DEVICE_MODAL_PAGE_SIZE = 10;
