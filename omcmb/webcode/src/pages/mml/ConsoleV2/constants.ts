import type { MMLOperationType } from '@core/types/mml';
import type { ExecStatus, UnverifiedReason } from './types';

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

/** 执行状态展示元数据（颜色 + 文案；设计 §3.11.2 四态 + 调度态）。 */
export const STATUS_META: Record<ExecStatus, { color: string; text: string }> = {
  pending: { color: 'default', text: '待执行' },
  running: { color: 'processing', text: '执行中' },
  success: { color: 'success', text: '成功' },
  unverified: { color: 'warning', text: '已下发·未核实' },
  mismatch: { color: 'error', text: '未生效' },
  failed: { color: 'error', text: '失败' },
};

/** unverified 原因文案（设计 §3.11.2，列表/详情明确提示，区分「未核实 ≠ 失败」）。 */
export const UNVERIFIED_REASON_TEXT: Record<UnverifiedReason, string> = {
  'write-only': '只写参数·无法核实',
  'reboot-required': '需重启生效·暂不核实',
  'query-failed': '核实查询失败',
};

/** 设备弹框服务端分页每页条数（mock 沿用现控制台 50 条上限）。 */
export const DEVICE_MODAL_PAGE_SIZE = 10;

// 单次执行设备数上限（设计 §3.10.1，2026-06-04 用户决策 200）。
// 依据：MML 执行无显式上限,但扇出单批 BatchCreateTasks 受 PostgreSQL 65535 bind 参数
// 约束(~5000 行硬顶),且每设备一个 Connection Request 受 ACS 准入/限流。200 远低于硬顶、
// ACS 可从容承接,>200 台规模化下发应走脚本任务。「全选满足筛选条件全部」最多选中前 200 台。
export const MAX_SELECT_ALL = 200;
