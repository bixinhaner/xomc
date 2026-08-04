import type { MMLOperationType } from '@core/types/mml';
import { MML_MAX_TASK_DEVICES, MML_PREVIEW_PAGE_SIZE } from '@core/utils/mmlTaskScale';
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

export const OP_LABEL_I18N_KEYS: Record<string, string> = {
  LST: 'mml.console.taskName.opVerb.LST',
  MOD: 'mml.console.taskName.opVerb.MOD',
  ADD: 'mml.console.taskName.opVerb.ADD',
  RMV: 'mml.console.taskName.opVerb.RMV',
  DSP: 'mml.console.taskName.opVerb.DSP',
  ACT: 'mml.console.taskName.opVerb.ACT',
  DEA: 'mml.console.taskName.opVerb.DEA',
  RST: 'mml.console.taskName.opVerb.RST',
  CLR: 'mml.console.taskName.opVerb.CLR',
  UPG: 'mml.console.taskName.opVerb.UPG',
};

export function opLabel(op: MMLOperationType | string | undefined): string {
  if (!op) return '-';
  return OP_LABELS[op] ?? op;
}

export function opLabelI18nKey(op: MMLOperationType | string | undefined): string | undefined {
  if (!op) return undefined;
  return OP_LABEL_I18N_KEYS[op];
}

export function opColor(op: MMLOperationType | string | undefined): string {
  if (!op) return 'default';
  return OP_COLORS[op] ?? 'default';
}

/** 是否「读」类操作（结果以参数值矩阵呈现）。其余为「写」类（状态 + 故障矩阵）。 */
export function isReadOp(op: MMLOperationType | string | undefined): boolean {
  return op === 'LST' || op === 'DSP';
}

/** 执行状态展示元数据（颜色 + i18n key；设计 §3.11.2 四态 + 调度态）。
 *  文案走 react-intl，消费方用 t(textKey) 解析（i18n guard #226）。 */
export const STATUS_META: Record<ExecStatus, { color: string; textKey: string }> = {
  pending: { color: 'default', textKey: 'mml.consoleV2.result.status.pending' },
  running: { color: 'processing', textKey: 'mml.consoleV2.result.status.running' },
  success: { color: 'success', textKey: 'mml.consoleV2.result.status.success' },
  unverified: { color: 'warning', textKey: 'mml.consoleV2.result.status.unverified' },
  mismatch: { color: 'error', textKey: 'mml.consoleV2.result.status.mismatch' },
  failed: { color: 'error', textKey: 'mml.consoleV2.result.status.failed' },
};

/** unverified 原因 i18n key（设计 §3.11.2，列表/详情明确提示，区分「未核实 ≠ 失败」）。 */
export const UNVERIFIED_REASON_TEXT: Record<UnverifiedReason, string> = {
  'write-only': 'mml.consoleV2.result.unverifiedReason.writeOnly',
  'reboot-required': 'mml.consoleV2.result.unverifiedReason.rebootRequired',
  'query-failed': 'mml.consoleV2.result.unverifiedReason.queryFailed',
};

/** 设备弹框服务端分页每页条数（mock 沿用现控制台 50 条上限）。 */
export const DEVICE_MODAL_PAGE_SIZE = MML_PREVIEW_PAGE_SIZE;

// 三步弹框（选择设备 / 选择命令 / 配置参数）统一固定高度，切换命令/标签页时不抖动，
// 整体高度参考「选择设备」弹框（其表格 scroll.y=320 + 表头/分页 ≈ 总高基准，不改动作为参照）。
/** 「选择命令」左右两栏主体高度（含搜索行后总高 ≈「选择设备」）。 */
export const COMMAND_MODAL_BODY_HEIGHT = 450;
/** 「配置参数」标签页内容区固定高度，超出竖向滚动（§需求 2），使总高不随命令变化（§需求 1）。 */
export const CONFIG_TAB_HEIGHT = 330;

// 单次执行设备数上限（设计 §3.10.1，2026-06-04 用户决策 200）。
// 依据：MML 执行无显式上限,但扇出单批 BatchCreateTasks 受 PostgreSQL 65535 bind 参数
// 约束(~5000 行硬顶),且每设备一个 Connection Request 受 ACS 准入/限流。200 远低于硬顶、
// ACS 可从容承接,>200 台规模化下发应走脚本任务。「全选满足筛选条件全部」最多选中前 200 台。
export const MAX_SELECT_ALL = MML_MAX_TASK_DEVICES;
