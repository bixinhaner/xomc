// MML 控制台 V2 —— 页面本地类型定义
//
// V2 改版（设计见 docs/design/mml-console-redesign-20260603.md）以「结果表格」为主舞台，
// 设备/命令选择收纳进顶部选择条 + 弹框。当前阶段为前端 mock 实现，类型独立于真实
// MMLCommand，待后端 results-schema / export 端点就绪后再桥接。

import type { MMLOperationType } from '@core/types/mml';

/** 设备在线状态 */
export type DeviceStatus = 'online' | 'offline' | 'alarm';

/** 设备选择弹框中的一行设备 */
export interface DeviceItem {
  sn: string;
  productName: string;
  productClass: string;
  status: DeviceStatus;
  groupName: string;
}

/** 命令选择弹框中的一条命令 */
export interface CommandItem {
  id: string;
  groupName: string;
  commandCode: string;
  commandName: string;
  operationType: MMLOperationType;
  description: string;
  /** 该命令涉及的参数路径（LST 查询列 / MOD 写入项的来源） */
  paramPaths: CommandParamPath[];
}

/** 命令绑定的参数路径 */
export interface CommandParamPath {
  /** TR-069 标准路径 */
  path: string;
  /** 展示用短标签（结果表格列头） */
  label: string;
  /** 是否可写（MOD/ADD 时可填值） */
  writable: boolean;
}

/** 单设备执行状态 */
export type ExecStatus = 'pending' | 'running' | 'success' | 'failed';

/** 结果表格的一行（= 一台设备） */
export interface ResultRow {
  deviceSn: string;
  status: ExecStatus;
  /** path -> 值（LST 查询结果 / MOD 回显），失败时为空 */
  cells: Record<string, string>;
  /** 失败故障码（MOD/ADD/RMV 写失败时填充） */
  faultCode?: string;
  /** 原始报文（SSE 文本流备查） */
  raw: string;
  /** 耗时（ms） */
  elapsedMs: number;
}

/** 结果表格动态列定义（来自命令的 paramPaths） */
export interface ResultColumn {
  key: string;
  label: string;
  path: string;
}

/** 执行模式 */
export type ExecMode = 'whole' | 'single-path';

/** 左操作区模式：标准参数（结构化）/ 参数路径指定（裸路径专家）。 */
export type OperationMode = 'standard' | 'raw';

/** 裸路径模式的一行（path + value，value 仅 MOD 使用）。 */
export interface RawPathRow {
  id: number;
  path: string;
  value: string;
}

/** 裸路径模式编辑态。 */
export interface RawPathPayload {
  operationType: MMLOperationType;
  rows: RawPathRow[];
}

/** 执行请求（标准 / 裸路径两模式的判别联合）。 */
export type ExecRequest =
  | { mode: 'standard'; checkedPaths: string[] }
  | { mode: 'raw'; operationType: MMLOperationType; rows: RawPathRow[] };

/** 一次执行的元信息（驱动结果表格列语义 + 「查看」详情的任务信息区）。 */
export interface ExecMeta {
  operationType: MMLOperationType;
  /** 是否「读」类操作（true→参数值矩阵，false→状态+故障矩阵）。 */
  read: boolean;
  /** 原始报文头部标识 / 命令码。 */
  label: string;
  /** 标准模式的命令名（裸路径模式为空）。 */
  commandName?: string;
}

/** 导出格式 */
export type ExportFormat = 'csv' | 'xlsx' | 'json';
