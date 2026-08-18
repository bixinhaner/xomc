import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { Alert, Button, Card, Empty, Popconfirm, Select, Space, Spin, Tag, Tooltip, Typography, message } from 'antd';
import { ExclamationCircleOutlined, MinusOutlined, PlusOutlined } from '@ant-design/icons';
import { useIntl } from 'react-intl';
import { useQueryClient } from '@tanstack/react-query';
import { useQuickSettingsGroups } from '@core/hooks/api/useQuickSettings';
import { useResolvedCellInstances } from '@core/hooks/api/useResolvedCellInstances';
import { useDeleteObject, useParameterSchema } from '@core/hooks/api/useDeviceParameters';
import { useDeviceTaskStatus } from '@core/hooks/api/useDeviceTask';
import { deviceParameterApi } from '@core/services/api/deviceParameterApi';
import { feedbackKey, useQuickSettingsFeedbackStore } from '@core/store/quickSettingsFeedbackStore';
import type { QuickSettingsGroup } from '@core/types/quicksettings';
import CellParameterForm from './CellParameterForm';
import InstanceSelectorForm from './InstanceSelectorForm';
import MultiInstanceTable, { formatDeviceFaultBrief, formatTime, statusTagSpec } from './MultiInstanceTable';
import FixedScalarSettingsTable from './FixedScalarSettingsTable';
import BscBtsAddModal from './BscBtsAddModal';
import { BSC_BTS_FEEDBACK_GROUP_ID } from './bscBtsFeedback';
import { applyInstanceContext, type QuickSettingsInstanceContext } from './validators';
import { getMlnMmePoolSyncPaths, isMlnIndexedMmePoolModel } from './mmeIpPlmnIndexed';
import { parameterReadbackMapsMatch, waitForReadback } from './parameterReadback';
import { useT } from '@/hooks/useT';

const { Text } = Typography;
const BM_RU_ROUTE_INDEX_PREFIX = 'Device.DeviceInfo.RU.';

const ENB_IPSEC_CONTROL_GROUP: QuickSettingsGroup = {
  id: 'device-ipsec-control',
  titleZh: 'IPSec 配置',
  titleEn: 'IPSec Config',
  multiInstance: false,
  params: [
    {
      name: 'IPSEC_ENABLE',
      titleZh: 'IPSec 开关',
      titleEn: 'IPSec Enable',
      standardPath: 'Device.Services.FAPService.Ipsec.IPSEC_ENABLE',
      enumOptions: [
        { value: '1', label: '开启' },
        { value: '0', label: '关闭' },
      ],
    },
  ],
};

const GNB_IPSEC_CONTROL_GROUP: QuickSettingsGroup = {
  id: 'device-ipsec-control',
  titleZh: 'IPSec 配置',
  titleEn: 'IPSec Config',
  multiInstance: false,
  params: [
    {
      name: 'IPSEC_ENABLE',
      titleZh: 'IPSec 开关',
      titleEn: 'IPSec Enable',
      standardPath: 'Device.IPsec.Enable',
      enumOptions: [
        { value: '1', label: 'ON' },
        { value: '0', label: 'OFF' },
      ],
    },
  ],
};

interface QuickSettingsTabProps {
  deviceId: string;
  /**
   * 设备详情页传入的 networkType 兼容历史值 eNB/gNB 和技术值 lte/nr。
   */
  networkType: string;
  active?: boolean;
  onSyncTargetPathsChange?: (paths: string[]) => void;
}

interface QuickSettingsGroupGateProps {
  active: boolean;
  eager?: boolean;
  title: ReactNode;
  children: (active: boolean) => ReactNode;
}

function QuickSettingsGroupGate({ active, eager = false, title, children }: QuickSettingsGroupGateProps) {
  const ref = useRef<HTMLDivElement | null>(null);
  const [shouldRender, setShouldRender] = useState(eager);

  useEffect(() => {
    if (!active || shouldRender) return;
    if (typeof IntersectionObserver === 'undefined') {
      setShouldRender(true);
      return;
    }
    const node = ref.current;
    if (!node) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          setShouldRender(true);
          observer.disconnect();
        }
      },
      { rootMargin: '160px 0px' },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [active, shouldRender]);

  return (
    <div ref={ref}>
      {shouldRender ? children(active) : (
        <Card title={title} size="small" style={{ marginBottom: 16, minHeight: 96 }}>
          <div style={{ height: 32 }} />
        </Card>
      )}
    </div>
  );
}

function useDeferredActive(active: boolean): boolean {
  const [deferredActive, setDeferredActive] = useState(false);

  useEffect(() => {
    if (!active) {
      setDeferredActive(false);
      return;
    }

    let cancelled = false;
    let raf1 = 0;
    let raf2 = 0;
    raf1 = window.requestAnimationFrame(() => {
      raf2 = window.requestAnimationFrame(() => {
        if (!cancelled) setDeferredActive(true);
      });
    });

    return () => {
      cancelled = true;
      window.cancelAnimationFrame(raf1);
      window.cancelAnimationFrame(raf2);
    };
  }, [active]);

  return deferredActive;
}

function normalizeQuickSettingsNetworkType(networkType: string): string {
  switch (networkType) {
    case 'eNB':
      return 'lte';
    case 'gNB':
      return 'nr';
    default:
      return networkType;
  }
}

const LTE_NUM_OF_CELLS_PATH = 'Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells';
const HIDDEN_GROUP_IDS = new Set(['device-sync']);
const DEVICE_LEVEL_IPSEC_GROUP_IDS = new Set(['device-ipsec', 'gnb-ipsec']);
const IPSEC_GROUP_IDS = new Set(['device-ipsec-control', 'device-ipsec', 'gnb-ipsec']);
const OUTER_GROUP_IDS = new Set([
  'device-time', 'bm-sync-source', 'gnb-sync-source',
  'gnb-network-interface', 'gnb-network-default-route', 'gnb-network-dscp',
  'gnb-network-dscp-list', 'gnb-network-static-route',
  'bm-network-settings',
  'device-ipsec-control', 'device-ipsec', 'gnb-ipsec',
]);
const FIXED_NETWORK_TABLE_MODELS = new Set(['BLN', 'BLQ', 'MLN', 'MLQ']);
function isFixedNetworkGroup(group: QuickSettingsGroup): boolean {
  return /^device-(?:wan|static-route)-\d+$/.test(group.id);
}
function getWanTableInsertionParamName(group: QuickSettingsGroup): string | undefined {
  const linkSpeedParam = group.params.find((param) => (
    param.name === 'LinkSpeed'
    || param.standardPath?.endsWith('.WanLinkSpeed')
    || param.titleEn === 'Link Speed Negotiated'
  ));
  return linkSpeedParam?.name ?? group.params.find((param) => param.name === 'ConnectType')?.name;
}
const isOuterGroup = (groupId: string): boolean => (
  OUTER_GROUP_IDS.has(groupId)
  || groupId === 'device-wan'
);
// BSC 设备 BTS 多实例父路径。额外的顶部 ＋/✖ 按钮调用 AddObject/DeleteObject
// 在该路径下管理 BTS 实例。
const BSC_BTS_OBJECT_PREFIX = 'DeviceGSM.Bts.';

/**
 * 设备详情「快速设置」tab — T-0138 per-paramModel 架构。
 *
 * 数据流:
 *  1. deviceId → useQuickSettingsGroups(deviceId) → 后端 device→product→paramModel 链路解析
 *  2. 返回的 groups 为空时,设备详情页层应不渲染本组件(规则:无 XML 不显示);
 *     本组件按规则做防御性兜底(Empty 占位)
 *  3. ENB(networkType='lte'):顶部显示 FAPService 实例下拉(枚举实际存在实例)
 *  4. GNB(networkType='nr'):按 Device.Services.FAPService.1.CellConfig.{i}. 枚举小区实例并允许切换
 *  5. 单实例分组 → CellParameterForm(整组 Save)
 *  6. 多实例分组 → MultiInstanceTable(行级 Save / AddObject + Set 两步 / DeleteObject)
 *
 * 小区实例解析（lte/nr/BM/cellModeIdx/InUse 联合规则）已统一到
 * useResolvedCellInstances，DeviceDetail 概览页与本组件共享同一计算路径。
 */
export default function QuickSettingsTab({ deviceId, networkType, active = true, onSyncTargetPathsChange }: QuickSettingsTabProps) {
  const intl = useIntl();
  const t = useT();
  const locale: 'zh-CN' | 'en-US' = intl.locale === 'en-US' ? 'en-US' : 'zh-CN';
  const normalizedNetworkType = normalizeQuickSettingsNetworkType(networkType);
  const queryActive = useDeferredActive(active);
  const [ipsecControlValue, setIpsecControlValue] = useState<string | undefined>(undefined);

  // LTE 选择 FAPService，NR 选择 CellConfig 小区实例。
  const [userPickedInstance, setUserPickedInstance] = useState<number | null>(null);
  const [bmTech, setBmTech] = useState<'LTE' | 'GSM'>('LTE');
  // BSC「新增 BTS」弹窗开关 — 见 BscBtsAddModal.tsx,内部完成 AddObject + SetParameterValues。
  const [btsAddModalOpen, setBtsAddModalOpen] = useState(false);

  // 头部"刷新"按钮 bump 的 tick → 拼进子组件 key 强制 remount，清掉 form/rowEdits 等组件内 state
  const refreshTick = useQuickSettingsFeedbackStore((s) => s.refreshTicks[deviceId] ?? 0);

  const { data, isLoading, error } = useQuickSettingsGroups(deviceId);
  const paramModel = data?.paramModel ?? '';

  // BM 模式下根据 quicksettings groups 的 path 判定是否存在 LTE/GSM 分组（影响制式选择器可见性）
  const bmHasGsmGroups = useMemo(() => {
    if (!paramModel.toUpperCase().startsWith('BM')) return false;
    return (data?.groups ?? []).some((group) => {
      if (group.objectPath?.startsWith('Device.Services.GsmBTSCellDT.')) return true;
      return group.params.some((param) =>
        (param.standardPath ?? '').startsWith('Device.Services.GsmBTSCellDT.'),
      );
    });
  }, [paramModel, data?.groups]);

  const bmHasLteGroups = useMemo(() => {
    if (!paramModel.toUpperCase().startsWith('BM')) return false;
    return (data?.groups ?? []).some((group) => {
      if (group.objectPath?.startsWith('Device.Services.FAPService.')) return true;
      return group.params.some((param) =>
        (param.standardPath ?? '').startsWith('Device.Services.FAPService.'),
      );
    });
  }, [paramModel, data?.groups]);

  const resolved = useResolvedCellInstances({
    deviceId,
    networkType: normalizedNetworkType,
    paramModel,
    bmTech,
    hasBmGsmGroups: bmHasGsmGroups,
    hasBmLteGroups: bmHasLteGroups,
    enabled: queryActive,
  });

  const {
    isENB,
    isNR,
    isBM,
    isBSC,
    instances: selectableInstances,
    lteInstances,
    lteConfiguredCellCount,
    nrCellInstances,
    bscBtsInstances,
    bmCellMode,
    bmTechOptions,
    loading: selectorLoading,
  } = resolved;
  const hasDeviceIpsecGroup = useMemo(
    () => (data?.groups ?? []).some((group) => DEVICE_LEVEL_IPSEC_GROUP_IDS.has(group.id)),
    [data?.groups],
  );

  useEffect(() => {
    setIpsecControlValue(undefined);
  }, [deviceId, refreshTick]);

  const activeBmTech = bmTechOptions.includes(bmTech)
    ? bmTech
    : (bmTechOptions[0] ?? (bmHasGsmGroups && !bmHasLteGroups ? 'GSM' : 'LTE'));

  const visibleGroups = useMemo(() => {
    const groups = (data?.groups ?? []).filter((group) => {
      if (HIDDEN_GROUP_IDS.has(group.id)) return false;
      return true;
    });
    const filteredGroups = (!isENB || !isBM)
      ? groups
      : groups.filter((group) => {
      const paths = [group.objectPath ?? '', ...group.params.map((param) => param.standardPath ?? '')];
      const hasGsmScopedPath = paths.some((path) => path.startsWith('Device.Services.GsmBTSCellDT.'));
      const hasLteScopedPath = paths.some((path) => path.startsWith('Device.Services.FAPService.'));

      if (!hasGsmScopedPath && !hasLteScopedPath) {
        return true;
      }
      if (activeBmTech === 'GSM') {
        return hasGsmScopedPath;
      }
      return hasLteScopedPath;
    });

    const needsBlqIpsecControl = hasDeviceIpsecGroup
      && filteredGroups.some((group) => DEVICE_LEVEL_IPSEC_GROUP_IDS.has(group.id))
      && !filteredGroups.some((group) => group.id === 'device-ipsec-control');
    if (!needsBlqIpsecControl) {
      return filteredGroups;
    }
    const deviceIpsecIndex = filteredGroups.findIndex((group) => DEVICE_LEVEL_IPSEC_GROUP_IDS.has(group.id));
    if (deviceIpsecIndex < 0) {
      return filteredGroups;
    }
    const nextGroups = [...filteredGroups];
    const targetIpsecGroup = filteredGroups[deviceIpsecIndex];
    nextGroups.splice(
      deviceIpsecIndex,
      0,
      targetIpsecGroup?.id === 'gnb-ipsec' ? GNB_IPSEC_CONTROL_GROUP : ENB_IPSEC_CONTROL_GROUP,
    );
    return nextGroups;
  }, [data?.groups, isENB, isBM, activeBmTech, hasDeviceIpsecGroup]);

  const outerGroups = useMemo(
    () => visibleGroups.filter((group) => isOuterGroup(group.id)),
    [visibleGroups],
  );
  const nonIpsecOuterGroups = useMemo(
    () => outerGroups.filter((group) => !IPSEC_GROUP_IDS.has(group.id)),
    [outerGroups],
  );
  const wanOuterGroups = useMemo(
    () => nonIpsecOuterGroups.filter((group) => group.id === 'device-wan'),
    [nonIpsecOuterGroups],
  );
  const nonWanOuterGroups = useMemo(
    () => nonIpsecOuterGroups.filter((group) => group.id !== 'device-wan'),
    [nonIpsecOuterGroups],
  );
  const ipsecOuterGroups = useMemo(
    () => outerGroups.filter((group) => IPSEC_GROUP_IDS.has(group.id)),
    [outerGroups],
  );
  const fixedNetworkGroups = useMemo(
    () => FIXED_NETWORK_TABLE_MODELS.has(paramModel.toUpperCase())
      ? visibleGroups.filter(isFixedNetworkGroup)
      : [],
    [paramModel, visibleGroups],
  );
  const fixedWanGroups = useMemo(
    () => fixedNetworkGroups.filter((group) => group.id.startsWith('device-wan-')),
    [fixedNetworkGroups],
  );
  const fixedStaticRouteGroups = useMemo(
    () => fixedNetworkGroups.filter((group) => group.id.startsWith('device-static-route-')),
    [fixedNetworkGroups],
  );

  useEffect(() => {
    if (!visibleGroups.some((group) => group.id === 'device-ipsec-control')) {
      setIpsecControlValue(undefined);
    }
  }, [visibleGroups]);

  const instanceScopedGroups = useMemo(
    () => visibleGroups.filter((group) => (
      !isOuterGroup(group.id)
      && !(FIXED_NETWORK_TABLE_MODELS.has(paramModel.toUpperCase()) && isFixedNetworkGroup(group))
    )),
    [paramModel, visibleGroups],
  );

  const selectedInstance =
    userPickedInstance !== null && selectableInstances.includes(userPickedInstance)
      ? userPickedInstance
      : selectableInstances[0] ?? 1;

  const instanceContext: QuickSettingsInstanceContext = useMemo(() => ({
    networkType: normalizedNetworkType,
    // BSC 重用 fapInstance 字段。applyInstanceContext 的非-NR 分支会把第一个 {i}
    // 替换为 fapInstance,而 BSC 的 standardPath 只含一个 {i}(DeviceGSM.Bts.{i}.<leaf>),
    // 正好被完整替换为选中的 BTS 实例号,无需新增独立的 btsInstance 字段。
    fapInstance: (isENB || isBSC) ? selectedInstance : 1,
    cellInstance: isNR ? selectedInstance : undefined,
  }), [normalizedNetworkType, isENB, isBSC, isNR, selectedInstance]);
  const selectorLabel = isENB
    ? (isBM
      ? (activeBmTech === 'GSM' ? t('device.quickSettings.cellInstanceGsm') : t('device.quickSettings.cellInstanceLte'))
      : (lteConfiguredCellCount !== null ? t('device.quickSettings.cellInstance') : t('device.quickSettings.fapInstance')))
    : isBSC
      ? t('device.quickSettings.btsInstance')
      : t('device.quickSettings.cellInstance');
  const selectorHint = isENB
    ? (isBM
      ? t('device.quickSettings.hintBm', { mode: bmCellMode ? `${bmCellMode.gsmNum}/${bmCellMode.lteNum}` : '-', tech: activeBmTech, count: selectableInstances.length })
      : (lteConfiguredCellCount !== null
        ? t('device.quickSettings.hintLte', { path: LTE_NUM_OF_CELLS_PATH, configured: lteConfiguredCellCount, count: selectableInstances.length })
        : t('device.quickSettings.hintFap', { count: lteInstances.length })))
    : isBSC
      ? t('device.quickSettings.hintBsc', { count: bscBtsInstances.length })
      : t('device.quickSettings.hintNr', { count: nrCellInstances.length });
  const selectedKey = isNR
    ? `${instanceContext.fapInstance}-${instanceContext.cellInstance ?? 1}`
    : String(instanceContext.fapInstance);

  const renderGroup = (group: typeof visibleGroups[number], keySuffix: string, eager = false) => {
    const childGroups = instanceScopedGroups.filter((g) => (
      g.parentSelector === group.id
      || (group.id === 'gnb-network-interface' && g.parentSelector === 'gnb-interface-vlan')
    ));
    if (group.parentSelector) return null;
    const wanTableInsertionParamName = group.id === 'device-wan'
      ? getWanTableInsertionParamName(group)
      : undefined;
    const title = locale === 'en-US' ? (group.titleEn || group.titleZh) : (group.titleZh || group.titleEn);
    return (
      <QuickSettingsGroupGate
        key={`${group.id}::gate::${keySuffix}`}
        active={queryActive}
        eager={eager}
        title={title}
      >
        {(childActive) => {
          if (group.style === 'table' && childGroups.length > 0) {
            return (
              <InstanceSelectorForm
                key={`${group.id}::${refreshTick}::${keySuffix}`}
                deviceId={deviceId}
                active={childActive}
                selectorGroup={group}
                childGroups={childGroups}
                instanceContext={instanceContext}
                locale={locale}
              />
            );
          }
          return group.multiInstance ? (
            <MultiInstanceTable
              key={`${group.id}::${refreshTick}::${keySuffix}`}
              deviceId={deviceId}
              active={childActive}
              group={group}
              instanceContext={instanceContext}
              locale={locale}
              ipsecControlValue={ipsecControlValue}
            />
          ) : (
            <CellParameterForm
              key={`${group.id}::${refreshTick}::${keySuffix}`}
              deviceId={deviceId}
              paramModel={paramModel}
              active={childActive}
              group={group}
              instanceContext={instanceContext}
              locale={locale}
              onIpsecControlChange={group.id === 'device-ipsec-control' ? setIpsecControlValue : undefined}
              actionMode={group.id === 'device-ipsec-control' ? 'staged' : 'standalone'}
              afterParamName={fixedWanGroups.length > 0 ? wanTableInsertionParamName : undefined}
              afterParamContent={fixedWanGroups.length > 0 && wanTableInsertionParamName ? (
                <FixedScalarSettingsTable
                  key={`device-wan-embedded-table::${refreshTick}`}
                  deviceId={deviceId}
                  active={childActive}
                  groups={fixedWanGroups}
                  locale={locale}
                  kind="wan"
                  embedded
                />
              ) : undefined}
            />
          );
        }}
      </QuickSettingsGroupGate>
    );
  };

  const syncTargetPaths = useMemo(() => {
    const paths = new Set<string>();
    for (const group of visibleGroups) {
      if (group.objectPath) {
        paths.add(applyInstanceContext(group.objectPath, instanceContext, { preserveTrailingInstance: true }));
      }
      for (const param of group.params) {
        if (
          group.id === 'enb-mme'
          && param.name === 'MmeIpPlmnList'
          && isMlnIndexedMmePoolModel(paramModel)
        ) {
          getMlnMmePoolSyncPaths().forEach((path) => paths.add(path));
          continue;
        }
        if (param.standardPath) {
          paths.add(applyInstanceContext(param.standardPath, instanceContext));
        }
        if (param.extraInfoPath) {
          paths.add(applyInstanceContext(param.extraInfoPath, instanceContext));
        }
      }
      const hasBmRuRouteBinding = group.params.some((param) => (
        param.name === 'GsmCellWithRuRelation' || param.name === 'LteCellWithRuList'
      ));
      if (hasBmRuRouteBinding) {
        paths.add(BM_RU_ROUTE_INDEX_PREFIX);
      }
      // 某些 quicksettings XML 会遗漏 Device.Time.Enable 的标准路径声明，
      // 这里兜底补齐，确保头部“刷新”一定会同步 NTP 开关值。
      if (group.id === 'device-time') {
        paths.add(applyInstanceContext('Device.Time.Enable', instanceContext));
      }
    }
    return Array.from(paths).filter(Boolean).sort();
  }, [visibleGroups, instanceContext, paramModel]);

  useEffect(() => {
    onSyncTargetPathsChange?.(syncTargetPaths);
  }, [onSyncTargetPathsChange, syncTargetPaths]);

  // BSC BTS 实例删除：复用 useDeleteObject;新增走 BscBtsAddModal 内部的 useAddObject。
  const deleteObjectMutation = useDeleteObject();
  // 新增按钮本身不再表示 pending — AddObject/SetParameterValues 由 Modal 内部的 confirmLoading 反映。
  const btsAddPending = false;
  const btsDeletePending = deleteObjectMutation.isPending;

  // BSC 下拉显示 "实例号 · IpaUnitId=xxx"，让运维能直接看出 BTS 与 IPA 单元映射。
  // 复用 useResolvedCellInstances 已经发过的同一份 schema 查询（react-query
  // 按 (deviceId, 'DeviceGSM.Bts.') key 去重，不会额外触发请求）。
  const { data: bscBtsSchema, refetch: refetchBscBtsSchema } = useParameterSchema(
    deviceId,
    BSC_BTS_OBJECT_PREFIX,
    queryActive && isBSC,
  );
  const bscBtsIpaUnitIdByInstance = useMemo(() => {
    const m = new Map<number, string>();
    if (!isBSC || !bscBtsSchema) return m;
    const RE = /^DeviceGSM\.Bts\.(\d+)\.IpaUnitId$/;
    for (const p of bscBtsSchema.parameters) {
      const match = RE.exec(p.path);
      if (!match) continue;
      const v = p.currentValue;
      if (v == null || v === '') continue;
      m.set(Number(match[1]), String(v));
    }
    return m;
  }, [isBSC, bscBtsSchema]);

  const queryClient = useQueryClient();

  // BSC 顶部"上次操作"反馈:与 Trx 行级 Tag 同一份 store。Modal 在 SPV 入队后写入
  // taskId,QuickSettingsTab 顶层用 useDeviceTaskStatus 轮询,Tag 自动从队列中 → 已发送
  // → 成功/失败/超时 滚动展示;失败时附被拒原因摘要 + Tooltip 全文。
  const bscFbKey = useMemo(
    () => feedbackKey(deviceId, BSC_BTS_FEEDBACK_GROUP_ID, 0),
    [deviceId],
  );
  const bscLastAction = useQuickSettingsFeedbackStore((s) => {
    const f = s.entries[bscFbKey];
    return f && f.kind === 'multi' ? f : null;
  });
  const setBscFeedback = useQuickSettingsFeedbackStore((s) => s.setFeedback);
  const patchBscFeedback = useQuickSettingsFeedbackStore((s) => s.patchFeedback);
  const { data: bscLastTask } = useDeviceTaskStatus(bscLastAction?.taskId);
  const bscAwaitingReadback = Boolean(
    bscLastAction?.submitStatus === 'queued'
    && bscLastAction.taskId
    && bscLastAction.syncedForTaskId !== bscLastAction.taskId
    && !['failed', 'expired', 'cancelled'].includes(bscLastTask?.status ?? ''),
  );

  useEffect(() => {
    if (!isBSC || !bscLastAction?.taskId || !bscLastTask) return;
    if (bscLastTask.status !== 'completed') return;
    if (bscLastAction.syncedForTaskId === bscLastTask.id) return;
    let cancelled = false;
    const abortController = new AbortController();
    void waitForReadback({
      read: async () => {
        deviceParameterApi.invalidateParameterSchemaCache(deviceId, BSC_BTS_OBJECT_PREFIX);
        return (await refetchBscBtsSchema()).data;
      },
      matches: (schema) => {
        if (!schema) return false;
        const expected = new Map(Object.entries(bscLastAction.expectedReadback ?? {}));
        const actual = new Map(
          schema.parameters.map((item) => [item.path, String(item.currentValue ?? '')]),
        );
        if (!parameterReadbackMapsMatch(expected, actual)) return false;
        const instances = schema.objects
          .find((item) => item.path === (bscLastAction.readbackObjectPath ?? BSC_BTS_OBJECT_PREFIX))
          ?.currentInstances.map(String) ?? [];
        const addedOk = (bscLastAction.addedInstIds ?? []).every((id) => instances.includes(id));
        const deletedOk = (bscLastAction.deletedInstIds ?? []).every((id) => !instances.includes(id));
        return addedOk && deletedOk;
      },
      intervalMs: 500,
      timeoutMs: 30_000,
      signal: abortController.signal,
    }).then(() => {
      if (!cancelled) patchBscFeedback(bscFbKey, { syncedForTaskId: bscLastTask.id });
    }).catch((error: unknown) => {
      if (error instanceof Error && error.name === 'AbortError') return;
      if (!cancelled) {
        message.error(t('device.multi.readbackFailed', { group: 'BTS' }));
      }
    });
    return () => {
      cancelled = true;
      abortController.abort();
    };
  }, [bscFbKey, bscLastAction, bscLastTask, deviceId, isBSC, patchBscFeedback, refetchBscBtsSchema, t]);

  // 任务终态(completed/failed/expired/cancelled)→ 让 schema/parameters/tree 都过期,
  // 触发实例下拉重新拉取(看到新增/已删的实例号)。
  // 额外:add 动作 + SPV 终态 failed + 有 param_faults + 已知 instanceNumber → 自动触发
  // DeleteObject 回滚刚才创建的 BTS,避免设备侧残留"半成品"实例。
  useEffect(() => {
    if (!isBSC || !bscLastAction?.taskId || !bscLastTask) return;
    if (bscLastTask.status !== 'completed' && bscLastTask.status !== 'failed'
      && bscLastTask.status !== 'expired' && bscLastTask.status !== 'cancelled') return;
    if (bscLastAction.invalidatedForTaskId === bscLastTask.id) return;
    deviceParameterApi.invalidateParameterSchemaCache(deviceId);
    void queryClient.invalidateQueries({ queryKey: ['devices', 'parameter-schema', deviceId] });
    void queryClient.invalidateQueries({ queryKey: ['devices', 'parameters', deviceId] });
    void queryClient.invalidateQueries({ queryKey: ['devices', 'parameter-tree', deviceId] });
    patchBscFeedback(bscFbKey, { invalidatedForTaskId: bscLastTask.id });

    // 自动回滚:仅当 action==='add' 且 SPV 任务 failed 且有具体 param_faults 且记得新增的实例号时触发
    const faults = (bscLastTask.result as { param_faults?: Array<{ parameter_name?: string; fault_string?: string }> } | undefined)?.param_faults ?? [];
    if (
      bscLastAction.action === 'add'
      && bscLastTask.status === 'failed'
      && faults.length > 0
      && typeof bscLastAction.instanceNumber === 'number'
      && bscLastAction.instanceNumber > 0
    ) {
      const inst = bscLastAction.instanceNumber;
      const brief = formatDeviceFaultBrief(bscLastTask.errorMessage);
      (async () => {
        try {
          const { taskId: rollbackTaskId } = await deleteObjectMutation.mutateAsync({
            deviceId,
            objectPath: `${BSC_BTS_OBJECT_PREFIX}${inst}.`,
          });
          setBscFeedback(bscFbKey, {
            kind: 'multi',
            action: 'add_rollback',
            submitStatus: 'queued',
            taskId: rollbackTaskId,
            detail: `BTS #${inst}`,
            at: Date.now(),
            instanceNumber: inst,
            originFaultBrief: brief,
            deletedInstIds: [String(inst)],
            readbackObjectPath: BSC_BTS_OBJECT_PREFIX,
          });
          // 当前下拉若停在被回滚的实例上,清掉让 selector 自动落回首条存活实例
          setUserPickedInstance((prev) => (prev === inst ? null : prev));
        } catch (err) {
          setBscFeedback(bscFbKey, {
            kind: 'multi',
            action: 'add_rollback',
            submitStatus: 'failed_to_queue',
            detail: `BTS #${inst} · 自动删除入队失败:${String(err)},请手动删除`,
            at: Date.now(),
            instanceNumber: inst,
            originFaultBrief: brief,
          });
          message.error(`BTS #${inst} 新增失败,自动回滚入队失败:${String(err)}`);
        }
      })();
    }
  }, [
    isBSC, bscLastAction, bscLastTask, queryClient, deviceId, bscFbKey,
    patchBscFeedback, setBscFeedback, deleteObjectMutation,
  ]);

  const handleAddBts = () => {
    // 弹出参数填写 Modal,确认后由 BscBtsAddModal 内部完成 AddObject + SetParameterValues 与提示。
    setBtsAddModalOpen(true);
  };
  const handleBtsAddSuccess = (instanceNumber: number) => {
    // 新增成功后切到新实例,便于继续编辑。
    // Tag 的状态/查询失效由顶层 useEffect 监听 bscLastTask 终态统一处理。
    setUserPickedInstance(instanceNumber);
  };
  const handleDeleteBts = async () => {
    if (!Number.isFinite(selectedInstance) || selectedInstance <= 0) return;
    try {
      const { taskId } = await deleteObjectMutation.mutateAsync({
        deviceId,
        objectPath: `${BSC_BTS_OBJECT_PREFIX}${selectedInstance}.`,
      });
      // 入队成功 — 写 feedback 让顶部 Tag 立即显示"删除 队列中 · BTS #N";
      // useDeviceTaskStatus(taskId) 自动轮询,终态由顶层 useEffect 触发刷新。
      setBscFeedback(bscFbKey, {
        kind: 'multi',
        action: 'delete',
        submitStatus: 'queued',
        taskId,
        deletedInstIds: [String(selectedInstance)],
        readbackObjectPath: BSC_BTS_OBJECT_PREFIX,
        detail: `BTS #${selectedInstance}`,
        at: Date.now(),
      });
      setUserPickedInstance(null);
    } catch (err) {
      setBscFeedback(bscFbKey, {
        kind: 'multi',
        action: 'delete',
        submitStatus: 'failed_to_queue',
        detail: `BTS #${selectedInstance} · 入队失败:${String(err)}`,
        at: Date.now(),
      });
      message.error(t('device.quickSettings.btsDeleteFailed', { reason: String(err) }));
    }
  };

  if (error) {
    return (
      <div style={{ padding: 16 }}>
        <Alert type="error" title={t('device.quickSettings.loadGroupsFailed')} description={String(error)} />
      </div>
    );
  }

  if (isLoading) {
    return (
      <div style={{ padding: 24, textAlign: 'center' }}>
        <Spin />
      </div>
    );
  }

  if (visibleGroups.length === 0) {
    const pm = data?.paramModel ?? t('device.quickSettings.unknown');
    const emptyDesc = isENB && isBM && activeBmTech === 'GSM'
      ? t('device.quickSettings.noGsmGroups', { pm })
      : t('device.quickSettings.noGroups', { pm });
    return (
      <div style={{ padding: 16 }}>
        <Empty description={emptyDesc} />
      </div>
    );
  }

  return (
    <div style={{ minHeight: '100%', padding: 12, background: 'var(--color-neutral-100, #f4f6f9)' }}>
      {nonWanOuterGroups.map((group, index) => renderGroup(group, 'outer', index < 2))}

      {wanOuterGroups.map((group) => renderGroup(group, 'wan'))}

      {(fixedWanGroups.length > 0 || fixedStaticRouteGroups.length > 0) && (
        <>
          {fixedWanGroups.length > 0 && wanOuterGroups.length === 0 && (
            <FixedScalarSettingsTable
              key={`device-wan-table::${refreshTick}`}
              deviceId={deviceId}
              active={queryActive}
              groups={fixedWanGroups}
              locale={locale}
              kind="wan"
            />
          )}
          {fixedStaticRouteGroups.length > 0 && (
            <FixedScalarSettingsTable
              key={`device-static-route-table::${refreshTick}`}
              deviceId={deviceId}
              active={queryActive}
              groups={fixedStaticRouteGroups}
              locale={locale}
              kind="static-route"
            />
          )}
        </>
      )}

      {ipsecOuterGroups.map((group) => renderGroup(group, 'outer-ipsec'))}

      {(isENB || isNR || isBSC) && (
        <Space style={{ marginBottom: 16 }}>
          {isENB && isBM && bmTechOptions.length > 1 && (
            <>
              <Text strong>{t('device.quickSettings.radioMode')}:</Text>
              <Select
                value={activeBmTech}
                onChange={(v) => {
                  setBmTech(v as 'LTE' | 'GSM');
                  setUserPickedInstance(null);
                }}
                style={{ width: 120 }}
                options={bmTechOptions.map((t) => ({ value: t, label: t }))}
              />
            </>
          )}
          <Text strong>{selectorLabel}</Text>
          <Select
            value={selectableInstances.includes(selectedInstance) ? selectedInstance : undefined}
            onChange={(v) => setUserPickedInstance(Number(v) || null)}
            style={{ width: isBSC ? 260 : 120 }}
            loading={selectorLoading}
            disabled={selectorLoading || selectableInstances.length === 0}
            options={selectableInstances.map((n) => {
              if (isBSC) {
                const ipa = bscBtsIpaUnitIdByInstance.get(n);
                return { value: n, label: ipa ? `${n} · IpaUnitId=${ipa}` : String(n) };
              }
              return { value: n, label: String(n) };
            })}
            placeholder={selectorLoading ? t('common.loading') : isENB ? t('device.quickSettings.noValidInstance') : t('device.quickSettings.noCellInstance')}
            notFoundContent={isENB ? t('device.quickSettings.noValidInstance') : t('device.quickSettings.noCellInstance')}
          />
          {isBSC && (
            <>
              <Button
                size="small"
                icon={<PlusOutlined />}
                loading={btsAddPending}
                disabled={bscAwaitingReadback}
                onClick={handleAddBts}
                title={t('device.quickSettings.btsAdd')}
              >
                {t('device.quickSettings.btsAdd')}
              </Button>
              <Popconfirm
                title={t('device.quickSettings.btsDeleteConfirm', { id: selectedInstance })}
                onConfirm={handleDeleteBts}
                okText={t('common.confirm')}
                cancelText={t('common.cancel')}
                disabled={selectableInstances.length === 0 || btsDeletePending || bscAwaitingReadback}
              >
                <Button
                  size="small"
                  danger
                  icon={<MinusOutlined />}
                  loading={btsDeletePending}
                  disabled={selectableInstances.length === 0 || bscAwaitingReadback}
                  title={t('device.quickSettings.btsDelete')}
                >
                  {t('device.quickSettings.btsDelete')}
                </Button>
              </Popconfirm>
              {bscLastAction && (() => {
                const visibleBscTaskStatus = bscLastTask?.status === 'completed'
                  && bscLastAction.taskId
                  && bscLastAction.syncedForTaskId !== bscLastTask.id
                  ? 'sent'
                  : bscLastTask?.status;
                // 普通新增/保存/删除走 statusTagSpec(失败时由外层渲染附加 briefFault + Tooltip 全文)。
                // add_rollback 是 SPV 部分被拒后自动 DeleteObject 的反馈:
                //  - 入队失败:红色"自动回滚入队失败,请手动删除"
                //  - delete completed:橙色"新增失败已自动回滚"
                //  - delete failed/expired/cancelled:红色"回滚未完成,请手动删除"
                //  - 其余进行中态:蓝色"正在自动回滚..."
                let spec: { color: string; icon: React.ReactNode; label: string };
                if (bscLastAction.action === 'add_rollback') {
                  if (bscLastAction.submitStatus === 'failed_to_queue') {
                    spec = { color: 'error', icon: <ExclamationCircleOutlined />, label: '新增失败 · 自动回滚入队失败' };
                  } else {
                    switch (visibleBscTaskStatus) {
                      case 'completed':
                        spec = { color: 'warning', icon: <ExclamationCircleOutlined />, label: '新增失败已自动回滚' };
                        break;
                      case 'failed':
                      case 'expired':
                      case 'cancelled':
                        spec = { color: 'error', icon: <ExclamationCircleOutlined />, label: '新增失败 · 自动回滚未完成,请手动删除' };
                        break;
                      default:
                        spec = { color: 'processing', icon: <ExclamationCircleOutlined />, label: '新增失败,正在自动回滚...' };
                    }
                  }
                } else {
                  spec = statusTagSpec(bscLastAction, visibleBscTaskStatus, t);
                }

                const isFailedTooltip = visibleBscTaskStatus === 'failed' && Boolean(bscLastTask?.errorMessage);
                const briefFault = bscLastAction.action === 'add_rollback'
                  ? (bscLastAction.originFaultBrief ?? '')
                  : (isFailedTooltip ? formatDeviceFaultBrief(bscLastTask?.errorMessage) : '');
                const tag = (
                  <Tag icon={spec.icon} color={spec.color}>
                    {spec.label} · {bscLastAction.detail}
                    {briefFault ? ` · ${briefFault}` : ''} · {formatTime(bscLastAction.at)}
                  </Tag>
                );
                // Tooltip 仅在普通失败时挂(回滚分支无完整 errorMessage)
                return isFailedTooltip && bscLastAction.action !== 'add_rollback' ? (
                  <Tooltip title={bscLastTask?.errorMessage} placement="bottomRight">
                    {tag}
                  </Tooltip>
                ) : tag;
              })()}
            </>
          )}
          <Text type="secondary">{selectorHint}</Text>
        </Space>
      )}

      {instanceScopedGroups.map((group, index) => renderGroup(group, selectedKey, index === 0))}

      {isBSC && (
        <BscBtsAddModal
          open={btsAddModalOpen}
          deviceId={deviceId}
          locale={locale}
          configGroup={visibleGroups.find((g) => g.id === 'bsc-bts-config')}
          handoverGroup={visibleGroups.find((g) => g.id === 'bsc-bts-handover')}
          onClose={() => setBtsAddModalOpen(false)}
          onSuccess={handleBtsAddSuccess}
        />
      )}
    </div>
  );
}
