import { useMemo } from 'react';

import { useParameterSchema } from './useDeviceParameters';

/**
 * 统一解析设备「快速设置 / 概览」可见的小区实例集合。
 *
 * 逻辑此前在 QuickSettingsTab/index.tsx 与 DeviceDetail/index.tsx 各自重复实现，
 * 容易在 BM 模式映射、cellModeIdx 越界、InUse 真值识别上漂移；本 hook 收敛之后
 * 两处共享同一规则集。
 *
 * 调用约定：
 *   - networkType 已归一化为 'lte' | 'nr' | 其他；
 *   - paramModel 来自 useQuickSettingsGroups().data?.paramModel（可能为空）；
 *   - bmTech 是 BM 设备的制式切换（'LTE' | 'GSM'），非 BM 设备忽略此参数。
 *
 * 返回字段：
 *   - ready：规则计算是否完成（用于决定是否对概览页 cellRecords 做过滤）；
 *   - instances：当前展示场景下应可见的实例号集合（升序）；
 *   - bmCellMode：解析出的 BM cellModeIdx → {gsmNum, lteNum}（未识别时为 null）；
 *   - bmTechOptions：BM 设备实际可选的制式（基于 InUse + groups 推导）；
 *   - lteConfiguredCellCount：非 BM 的 ENB 设备按 NumOfCells 推导出的小区数；
 *   - lteInstances：FAPService.* 实际存在的实例集合（兜底）。
 */

const LTE_FAPSERVICE_PREFIX = 'Device.Services.FAPService.';
const NR_CELLCONFIG_PREFIX = 'Device.Services.FAPService.1.CellConfig.';
const LTE_NUM_OF_CELLS_PREFIX = 'Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.';
const LTE_NUM_OF_CELLS_PATH = 'Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells';
const BM_DEVICE_INFO_PREFIX = 'Device.DeviceInfo.';
const BM_GSM_CELL_PREFIX = 'Device.Services.GsmBTSCellDT.';
// BSC(独立 GSM BSC 设备,非 BM)的 BTS 多实例前缀。与 BM 的 GsmBTSCellDT 不同,
// 这是 osmo-bsc 风格的 DeviceGSM.Bts.{i}. 路径,每个 {i} 为一个 BTS 实例。
const BSC_BTS_PREFIX = 'DeviceGSM.Bts.';
const BSC_BTS_INSTANCE_RE = /^DeviceGSM\.Bts\.(\d+)\./;

// 非 BM 的 ENB 设备每个 FAPService 最多展示几个 cell。来源：BLN/MLN/BLQ 现网模板
// 上限是 3（CA 最多 3 载波）。如果将来出现 4-CA 设备，改这个常量即可。
export const MAX_LTE_CELLS_PER_FAP = 3;

// BM 设备 cellModeIdx 与 (GSM 实例数, LTE 实例数) 的映射；与 BM 固件文档一致。
// 未在表中的取值（unknown 模式）走 InUse-only 兜底，避免选择器整体空白。
const BM_CELL_MODE_MAP: Record<string, { gsmNum: number; lteNum: number }> = {
  '0': { gsmNum: 3, lteNum: 3 },
  '1': { gsmNum: 3, lteNum: 6 },
  '2': { gsmNum: 0, lteNum: 6 },
  '3': { gsmNum: 0, lteNum: 9 },
  '4': { gsmNum: 3, lteNum: 3 },
  '5': { gsmNum: 6, lteNum: 6 },
};

const LTE_FAP_INUSE_RE = /^Device\.Services\.FAPService\.(\d+)\.FAPControl\.LTE\.InUse$/i;
const GSM_INUSE_RE = /^Device\.Services\.GsmBTSCellDT\.(\d+)\.InUse$/i;
const GSM_INSTANCE_RE = /^Device\.Services\.GsmBTSCellDT\.(\d+)\./i; // #374: GSM 实例存在性
const FAP_INSTANCE_RE = /^Device\.Services\.FAPService\.(\d+)\./;
const NR_INSTANCE_RE = /^Device\.Services\.FAPService\.1\.CellConfig\.(\d+)\./;

// #374: 显式「禁用」判定——仅当 InUse 参数存在且取值明确为假值时才隐藏小区。
// 缺参 / 同步缺失 / undefined / 空串 不算「禁用」（返回 false），这样「实例存在但
// InUse 缺失/未同步」的小区不会被静默吃掉（第 6 个 LTE 小区丢失根因）。
function isExplicitFalsyInUse(value: string | null | undefined): boolean {
  if (value === null || value === undefined) return false;
  const normalized = String(value).trim().toLowerCase();
  if (normalized === '') return false; // 空串视为「未同步」，不当禁用
  return normalized === '0'
    || normalized === 'false'
    || normalized === 'off'
    || normalized === 'disable'
    || normalized === 'disabled'
    || normalized === 'no';
}

/**
 * #374: 纯函数——从 schema 的 parameters/objects 解析「存在即可见、仅显式 InUse
 * 假值隐藏」的实例集合。LTE 与 GSM 共用，便于单测（hook 内 useMemo 调它）。
 *
 * @param parameters schema.parameters（含 path + currentValue）
 * @param objectCurrentInstances 对应对象路径的 currentInstances（schema 显式列出的实例）
 * @param inUseRe 匹配该制式 InUse path 的正则（捕获组 1 = 实例号）
 * @param instanceRe 匹配该制式『实例存在』path 的正则（捕获组 1 = 实例号）
 * @param limit 上界（bmCellMode 的 lteNum/gsmNum；未识别 mode 传 Infinity）
 */
export function resolveExistsVisibleInstances(
  parameters: ReadonlyArray<{ path: string; currentValue?: string | null }>,
  objectCurrentInstances: ReadonlyArray<number>,
  inUseRe: RegExp,
  instanceRe: RegExp,
  limit: number,
): number[] {
  if (limit <= 0) return [];
  const inUseByInstance = new Map<number, string | null | undefined>();
  const existing = new Set<number>();
  for (const p of parameters) {
    const inUseMatch = inUseRe.exec(p.path);
    if (inUseMatch) inUseByInstance.set(Number(inUseMatch[1]), p.currentValue);
    const instMatch = instanceRe.exec(p.path);
    if (instMatch) existing.add(Number(instMatch[1]));
  }
  for (const n of objectCurrentInstances) existing.add(n);

  const enabled = new Set<number>();
  for (const instance of existing) {
    if (instance < 1 || instance > limit) continue;
    if (isExplicitFalsyInUse(inUseByInstance.get(instance))) continue;
    enabled.add(instance);
  }
  return Array.from(enabled).sort((a, b) => a - b);
}

export interface ResolvedCellInstancesOptions {
  /** 设备 id；空字符串/undefined 时所有派生 schema 请求均 disabled。*/
  deviceId: string;
  /** 已归一化的 networkType（'lte' | 'nr' | ...）。*/
  networkType: string;
  /** 来自 useQuickSettingsGroups().data?.paramModel；用于检测 BM 设备。*/
  paramModel?: string;
  /** BM 设备的制式切换；非 BM 设备无效。默认 'LTE'。*/
  bmTech?: 'LTE' | 'GSM';
  /** 给 QuickSettingsTab 用：BM 制式切换提示是否需要根据 quicksettings groups 决定可见性。*/
  hasBmGsmGroups?: boolean;
  hasBmLteGroups?: boolean;
  /** 隐藏 quick settings 页签保活时关闭 schema 查询，避免切 tab 触发 fan-out。 */
  enabled?: boolean;
}

export interface ResolvedCellInstances {
  ready: boolean;
  isENB: boolean;
  isNR: boolean;
  isBM: boolean;
  /** BSC(独立 GSM 设备,paramModel === 'BSC')，顶部需要 BTS 实例选择器。*/
  isBSC: boolean;
  loading: boolean;
  instances: number[];
  lteInstances: number[];
  lteConfiguredCellCount: number | null;
  nrCellInstances: number[];
  /** BSC 设备枚举出的 BTS 实例号(升序)。 */
  bscBtsInstances: number[];
  bmCellMode: { gsmNum: number; lteNum: number } | null;
  bmEnabledLteInstances: number[];
  bmEnabledGsmInstances: number[];
  bmTechOptions: Array<'LTE' | 'GSM'>;
}

export function useResolvedCellInstances({
  deviceId,
  networkType,
  paramModel,
  bmTech = 'LTE',
  hasBmGsmGroups = false,
  hasBmLteGroups = false,
  enabled = true,
}: ResolvedCellInstancesOptions): ResolvedCellInstances {
  const isENB = networkType === 'lte';
  const isNR = networkType === 'nr';
  const isBM = (paramModel ?? '').toUpperCase().startsWith('BM');
  // BSC(独立 GSM 设备) 按 paramModel 判定;与 BM/ENB/NR 互斥。
  const isBSC = (paramModel ?? '').toUpperCase() === 'BSC';

  const { data: fapSchema, isLoading: fapSchemaLoading } = useParameterSchema(
    deviceId,
    LTE_FAPSERVICE_PREFIX,
    enabled && Boolean(deviceId) && isENB,
  );

  const { data: lteNumOfCellsSchema, isLoading: lteNumOfCellsLoading } = useParameterSchema(
    deviceId,
    LTE_NUM_OF_CELLS_PREFIX,
    enabled && Boolean(deviceId) && isENB && !isBM,
  );

  const { data: bmDeviceInfoSchema, isLoading: bmDeviceInfoLoading } = useParameterSchema(
    deviceId,
    BM_DEVICE_INFO_PREFIX,
    enabled && Boolean(deviceId) && isENB && isBM,
  );

  const { data: bmGsmSchema, isLoading: bmGsmSchemaLoading } = useParameterSchema(
    deviceId,
    BM_GSM_CELL_PREFIX,
    enabled && Boolean(deviceId) && isENB && isBM,
  );

  const { data: nrCellSchema, isLoading: nrCellSchemaLoading } = useParameterSchema(
    deviceId,
    NR_CELLCONFIG_PREFIX,
    enabled && Boolean(deviceId) && isNR,
  );

  const { data: bscBtsSchema, isLoading: bscBtsSchemaLoading } = useParameterSchema(
    deviceId,
    BSC_BTS_PREFIX,
    enabled && Boolean(deviceId) && isBSC,
  );

  const lteInstances = useMemo<number[]>(() => {
    if (!isENB || !fapSchema) return [];
    const objEntry = fapSchema.objects.find((o) => o.path === LTE_FAPSERVICE_PREFIX);
    const candidate = objEntry?.currentInstances ?? [];
    if (candidate.length > 0) return [...candidate].sort((a, b) => a - b);
    const fallback = new Set<number>();
    for (const p of fapSchema.parameters) {
      const m = FAP_INSTANCE_RE.exec(p.path);
      if (m) fallback.add(Number(m[1]));
    }
    return Array.from(fallback).sort((a, b) => a - b);
  }, [isENB, fapSchema]);

  const lteConfiguredCellCount = useMemo<number | null>(() => {
    if (!isENB || isBM || !lteNumOfCellsSchema) return null;
    const numOfCellsParam = lteNumOfCellsSchema.parameters.find(
      (p) => p.path.toLowerCase() === LTE_NUM_OF_CELLS_PATH.toLowerCase(),
    );
    const parsed = Number.parseInt(String(numOfCellsParam?.currentValue ?? ''), 10);
    if (!Number.isFinite(parsed) || parsed < 1) return null;
    return Math.min(MAX_LTE_CELLS_PER_FAP, parsed);
  }, [isENB, isBM, lteNumOfCellsSchema]);

  const lteCellInstances = useMemo<number[]>(() => {
    if (!isENB || isBM) return [];
    if (lteConfiguredCellCount !== null) {
      return Array.from({ length: lteConfiguredCellCount }, (_, idx) => idx + 1);
    }
    return lteInstances;
  }, [isENB, isBM, lteConfiguredCellCount, lteInstances]);

  const bmCellMode = useMemo<ResolvedCellInstances['bmCellMode']>(() => {
    if (!isENB || !isBM || !bmDeviceInfoSchema) return null;
    const cellModeParam = bmDeviceInfoSchema.parameters.find(
      (p) => p.path.toLowerCase() === 'device.deviceinfo.cellmodeidx',
    );
    const modeKey = String(cellModeParam?.currentValue ?? '').trim();
    return BM_CELL_MODE_MAP[modeKey] ?? null;
  }, [isENB, isBM, bmDeviceInfoSchema]);

  // BM 实例集合：bmCellMode 命中 → 用其上限；bmCellMode 为 null（未识别 mode）→
  // 不限上界，避免选择器整体空白。
  //
  // #374 兜底：原实现只把「InUse 命中且为真值」的实例计入，对「实例存在(有
  // NumOfCells/PCI 等参数)但 InUse 缺失 / 同步缺失 / 空串」的实例无兜底——3GMS+6LTE
  // 时只要第 6 个 FAPService 的 InUse 缺失就被静默丢掉。改为「存在即可见」：枚举
  // 在 [1, limit] 范围内『存在』的 FAPService 实例（FAP_INSTANCE_RE / currentInstances），
  // 仅当其 InUse 参数存在且显式为假值（'0'/'false'…）才隐藏；缺参/未同步保留可见。
  // 与后端 GSM AssembleGSMCells 的 hasAnyGSMInUse『InUse 缺失不静默吃掉』口径一致。
  const bmEnabledLteInstances = useMemo<number[]>(() => {
    if (!isENB || !isBM || !fapSchema) return [];
    const limit = bmCellMode?.lteNum ?? Infinity;
    const objEntry = fapSchema.objects.find((o) => o.path === LTE_FAPSERVICE_PREFIX);
    return resolveExistsVisibleInstances(
      fapSchema.parameters,
      objEntry?.currentInstances ?? [],
      LTE_FAP_INUSE_RE,
      FAP_INSTANCE_RE,
      limit,
    );
  }, [isENB, isBM, fapSchema, bmCellMode]);

  // #374: GSM 侧与 LTE 对齐——存在即可见，仅显式 InUse 假值隐藏；与后端
  // AssembleGSMCells 的 hasAnyGSMInUse『InUse 缺失不静默吃掉』口径一致。
  const bmEnabledGsmInstances = useMemo<number[]>(() => {
    if (!isENB || !isBM || !bmGsmSchema) return [];
    const limit = bmCellMode?.gsmNum ?? Infinity;
    const objEntry = bmGsmSchema.objects.find((o) => o.path === BM_GSM_CELL_PREFIX);
    return resolveExistsVisibleInstances(
      bmGsmSchema.parameters,
      objEntry?.currentInstances ?? [],
      GSM_INUSE_RE,
      GSM_INSTANCE_RE,
      limit,
    );
  }, [isENB, isBM, bmGsmSchema, bmCellMode]);

  const nrCellInstances = useMemo<number[]>(() => {
    if (!isNR || !nrCellSchema) return [];
    const objEntry = nrCellSchema.objects.find((o) => o.path === NR_CELLCONFIG_PREFIX);
    const candidate = objEntry?.currentInstances ?? [];
    if (candidate.length > 0) return [...candidate].sort((a, b) => a - b);
    const fallback = new Set<number>();
    for (const p of nrCellSchema.parameters) {
      const m = NR_INSTANCE_RE.exec(p.path);
      if (m) fallback.add(Number(m[1]));
    }
    return Array.from(fallback).sort((a, b) => a - b);
  }, [isNR, nrCellSchema]);

  // BSC 枚举 DeviceGSM.Bts.{i}. 实例集合。优先用 schema 指定的 currentInstances,
  // 其次从 parameters[].path 正则 fallback;与 LTE/NR 枚举策略一致。
  // 业务约定 BTS 实例号从 1 开始(与 LTE FAPService.{i} / NR Cell.{i} 一致),
  // osmo-bsc 0 号槽位是内部模板,不暴露给运维。
  const bscBtsInstances = useMemo<number[]>(() => {
    if (!isBSC || !bscBtsSchema) return [];
    const objEntry = bscBtsSchema.objects.find((o) => o.path === BSC_BTS_PREFIX);
    const candidate = objEntry?.currentInstances ?? [];
    if (candidate.length > 0) return [...candidate].filter((n) => n > 0).sort((a, b) => a - b);
    const fallback = new Set<number>();
    for (const p of bscBtsSchema.parameters) {
      const m = BSC_BTS_INSTANCE_RE.exec(p.path);
      if (m) {
        const n = Number(m[1]);
        if (n > 0) fallback.add(n);
      }
    }
    return Array.from(fallback).sort((a, b) => a - b);
  }, [isBSC, bscBtsSchema]);

  const bmTechOptions = useMemo<Array<'LTE' | 'GSM'>>(() => {
    const options: Array<'LTE' | 'GSM'> = [];
    if (bmEnabledLteInstances.length > 0 || hasBmLteGroups) options.push('LTE');
    if (bmEnabledGsmInstances.length > 0 || hasBmGsmGroups) options.push('GSM');
    return options;
  }, [bmEnabledLteInstances.length, bmEnabledGsmInstances.length, hasBmLteGroups, hasBmGsmGroups]);

  const instances = useMemo<number[]>(() => {
    if (isENB) {
      if (isBM) return bmTech === 'GSM' ? bmEnabledGsmInstances : bmEnabledLteInstances;
      return lteCellInstances;
    }
    if (isNR) return nrCellInstances;
    if (isBSC) return bscBtsInstances;
    return [];
  }, [isENB, isBM, bmTech, bmEnabledGsmInstances, bmEnabledLteInstances, lteCellInstances, isNR, nrCellInstances, isBSC, bscBtsInstances]);

  const loading = useMemo(() => {
    if (isENB) {
      if (isBM) return fapSchemaLoading || bmDeviceInfoLoading || bmGsmSchemaLoading;
      return fapSchemaLoading || lteNumOfCellsLoading;
    }
    if (isNR) return nrCellSchemaLoading;
    if (isBSC) return bscBtsSchemaLoading;
    return false;
  }, [isENB, isBM, fapSchemaLoading, bmDeviceInfoLoading, bmGsmSchemaLoading, lteNumOfCellsLoading, isNR, nrCellSchemaLoading, isBSC, bscBtsSchemaLoading]);

  const ready = (isENB || isNR || isBSC) && !loading && Boolean(deviceId);

  return {
    ready,
    isENB,
    isNR,
    isBM,
    isBSC,
    loading,
    instances,
    lteInstances,
    lteConfiguredCellCount,
    nrCellInstances,
    bscBtsInstances,
    bmCellMode,
    bmEnabledLteInstances,
    bmEnabledGsmInstances,
    bmTechOptions,
  };
}
