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
const FAP_INSTANCE_RE = /^Device\.Services\.FAPService\.(\d+)\./;
const NR_INSTANCE_RE = /^Device\.Services\.FAPService\.1\.CellConfig\.(\d+)\./;

function isTruthyInUse(value: string | null | undefined): boolean {
  const normalized = String(value ?? '').trim().toLowerCase();
  return normalized === '1'
    || normalized === 'true'
    || normalized === 'on'
    || normalized === 'enable'
    || normalized === 'enabled'
    || normalized === 'yes';
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
}

export interface ResolvedCellInstances {
  ready: boolean;
  isENB: boolean;
  isNR: boolean;
  isBM: boolean;
  loading: boolean;
  instances: number[];
  lteInstances: number[];
  lteConfiguredCellCount: number | null;
  nrCellInstances: number[];
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
}: ResolvedCellInstancesOptions): ResolvedCellInstances {
  const isENB = networkType === 'lte';
  const isNR = networkType === 'nr';
  const isBM = (paramModel ?? '').toUpperCase().startsWith('BM');

  const { data: fapSchema, isLoading: fapSchemaLoading } = useParameterSchema(
    deviceId,
    LTE_FAPSERVICE_PREFIX,
    Boolean(deviceId) && isENB,
  );

  const { data: lteNumOfCellsSchema, isLoading: lteNumOfCellsLoading } = useParameterSchema(
    deviceId,
    LTE_NUM_OF_CELLS_PREFIX,
    Boolean(deviceId) && isENB && !isBM,
  );

  const { data: bmDeviceInfoSchema, isLoading: bmDeviceInfoLoading } = useParameterSchema(
    deviceId,
    BM_DEVICE_INFO_PREFIX,
    Boolean(deviceId) && isENB && isBM,
  );

  const { data: bmGsmSchema, isLoading: bmGsmSchemaLoading } = useParameterSchema(
    deviceId,
    BM_GSM_CELL_PREFIX,
    Boolean(deviceId) && isENB && isBM,
  );

  const { data: nrCellSchema, isLoading: nrCellSchemaLoading } = useParameterSchema(
    deviceId,
    NR_CELLCONFIG_PREFIX,
    Boolean(deviceId) && isNR,
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

  // BM 实例集合：bmCellMode 命中 → 用其上限 + InUse 过滤；
  // bmCellMode 为 null（未识别 mode）→ 退化到「仅 InUse=true」，不限上界，避免选择器整体空白。
  const bmEnabledLteInstances = useMemo<number[]>(() => {
    if (!isENB || !isBM || !fapSchema) return [];
    const limit = bmCellMode?.lteNum ?? Infinity;
    if (bmCellMode && limit <= 0) return [];
    const enabled = new Set<number>();
    for (const p of fapSchema.parameters) {
      const m = LTE_FAP_INUSE_RE.exec(p.path);
      if (!m) continue;
      const instance = Number(m[1]);
      if (instance < 1 || instance > limit) continue;
      if (isTruthyInUse(p.currentValue)) enabled.add(instance);
    }
    return Array.from(enabled).sort((a, b) => a - b);
  }, [isENB, isBM, fapSchema, bmCellMode]);

  const bmEnabledGsmInstances = useMemo<number[]>(() => {
    if (!isENB || !isBM || !bmGsmSchema) return [];
    const limit = bmCellMode?.gsmNum ?? Infinity;
    if (bmCellMode && limit <= 0) return [];
    const enabled = new Set<number>();
    for (const p of bmGsmSchema.parameters) {
      const m = GSM_INUSE_RE.exec(p.path);
      if (!m) continue;
      const instance = Number(m[1]);
      if (instance < 1 || instance > limit) continue;
      if (isTruthyInUse(p.currentValue)) enabled.add(instance);
    }
    return Array.from(enabled).sort((a, b) => a - b);
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
    return [];
  }, [isENB, isBM, bmTech, bmEnabledGsmInstances, bmEnabledLteInstances, lteCellInstances, isNR, nrCellInstances]);

  const loading = useMemo(() => {
    if (isENB) {
      if (isBM) return fapSchemaLoading || bmDeviceInfoLoading || bmGsmSchemaLoading;
      return fapSchemaLoading || lteNumOfCellsLoading;
    }
    if (isNR) return nrCellSchemaLoading;
    return false;
  }, [isENB, isBM, fapSchemaLoading, bmDeviceInfoLoading, bmGsmSchemaLoading, lteNumOfCellsLoading, isNR, nrCellSchemaLoading]);

  const ready = (isENB || isNR) && !loading && Boolean(deviceId);

  return {
    ready,
    isENB,
    isNR,
    isBM,
    loading,
    instances,
    lteInstances,
    lteConfiguredCellCount,
    nrCellInstances,
    bmCellMode,
    bmEnabledLteInstances,
    bmEnabledGsmInstances,
    bmTechOptions,
  };
}
