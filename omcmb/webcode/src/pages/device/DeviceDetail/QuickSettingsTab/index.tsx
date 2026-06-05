import { useMemo, useState } from 'react';
import { Alert, Empty, Select, Space, Spin, Typography } from 'antd';
import { useIntl } from 'react-intl';
import { useQuickSettingsGroups } from '@core/hooks/api/useQuickSettings';
import { useResolvedCellInstances } from '@core/hooks/api/useResolvedCellInstances';
import { useQuickSettingsFeedbackStore } from '@core/store/quickSettingsFeedbackStore';
import CellParameterForm from './CellParameterForm';
import MultiInstanceTable from './MultiInstanceTable';
import type { QuickSettingsInstanceContext } from './validators';

const { Text } = Typography;

interface QuickSettingsTabProps {
  deviceId: string;
  /**
   * 设备详情页传入的 networkType 兼容历史值 eNB/gNB 和技术值 lte/nr。
   */
  networkType: string;
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
const HIDDEN_GROUP_IDS = new Set(['device-time', 'device-sync']);

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
export default function QuickSettingsTab({ deviceId, networkType }: QuickSettingsTabProps) {
  const intl = useIntl();
  const locale: 'zh-CN' | 'en-US' = intl.locale === 'en-US' ? 'en-US' : 'zh-CN';
  const normalizedNetworkType = normalizeQuickSettingsNetworkType(networkType);

  // LTE 选择 FAPService，NR 选择 CellConfig 小区实例。
  const [userPickedInstance, setUserPickedInstance] = useState<number | null>(null);
  const [bmTech, setBmTech] = useState<'LTE' | 'GSM'>('LTE');

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
  });

  const {
    isENB,
    isNR,
    isBM,
    instances: selectableInstances,
    lteInstances,
    lteConfiguredCellCount,
    nrCellInstances,
    bmCellMode,
    bmTechOptions,
    loading: selectorLoading,
  } = resolved;

  const activeBmTech = bmTechOptions.includes(bmTech)
    ? bmTech
    : (bmTechOptions[0] ?? (bmHasGsmGroups && !bmHasLteGroups ? 'GSM' : 'LTE'));

  const visibleGroups = useMemo(() => {
    const groups = (data?.groups ?? []).filter((group) => !HIDDEN_GROUP_IDS.has(group.id));
    if (!isENB || !isBM) {
      return groups;
    }
    return groups.filter((group) => {
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
  }, [data?.groups, isENB, isBM, activeBmTech]);

  const selectedInstance =
    userPickedInstance !== null && selectableInstances.includes(userPickedInstance)
      ? userPickedInstance
      : selectableInstances[0] ?? 1;

  const instanceContext: QuickSettingsInstanceContext = {
    networkType: normalizedNetworkType,
    fapInstance: isENB ? selectedInstance : 1,
    cellInstance: isNR ? selectedInstance : undefined,
  };
  const selectorLabel = isENB
    ? (isBM
      ? (activeBmTech === 'GSM' ? 'GSM小区实例:' : 'LTE小区实例:')
      : (lteConfiguredCellCount !== null ? '小区实例:' : 'FAPService 实例:'))
    : '小区实例:';
  const selectorHint = isENB
    ? (isBM
      ? `(BM: cellModeIdx=${bmCellMode ? `${bmCellMode.gsmNum}/${bmCellMode.lteNum}` : '-'}(GSM/LTE) · 当前${activeBmTech}已启用 ${selectableInstances.length} 个)`
      : (lteConfiguredCellCount !== null
        ? `(按 ${LTE_NUM_OF_CELLS_PATH}=${lteConfiguredCellCount} 显示 · 共 ${selectableInstances.length} 个小区)`
        : `(显示设备上实际存在的 FAPService 实例 · 共 ${lteInstances.length} 个)`))
    : `(按 Device.Services.FAPService.1.CellConfig.{i}. 枚举 · 共 ${nrCellInstances.length} 个小区)`;
  const selectedKey = isNR
    ? `${instanceContext.fapInstance}-${instanceContext.cellInstance ?? 1}`
    : String(instanceContext.fapInstance);

  if (error) {
    return (
      <div style={{ padding: 16 }}>
        <Alert type="error" message="加载快速设置分组失败" description={String(error)} />
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
    const emptyDesc = isENB && isBM && activeBmTech === 'GSM'
      ? `该设备的 paramModel(${data?.paramModel ?? '未知'})未配置 GSM 快速设置分组`
      : `该设备的 paramModel(${data?.paramModel ?? '未知'})未配置快速设置分组`;
    return (
      <div style={{ padding: 16 }}>
        <Empty description={emptyDesc} />
      </div>
    );
  }

  return (
    <div style={{ padding: 16 }}>
      {(isENB || isNR) && (
        <Space style={{ marginBottom: 16 }}>
          {isENB && isBM && bmTechOptions.length > 1 && (
            <>
              <Text strong>制式:</Text>
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
            style={{ width: 120 }}
            loading={selectorLoading}
            disabled={selectorLoading || selectableInstances.length === 0}
            options={selectableInstances.map((n) => ({ value: n, label: String(n) }))}
            placeholder={selectorLoading ? '加载中...' : isENB ? '无有效实例' : '无小区实例'}
            notFoundContent={isENB ? '无有效实例' : '无小区实例'}
          />
          <Text type="secondary">{selectorHint}</Text>
        </Space>
      )}

      {visibleGroups.map((group) =>
        group.multiInstance ? (
          <MultiInstanceTable
            key={`${group.id}::${refreshTick}::${selectedKey}`}
            deviceId={deviceId}
            group={group}
            instanceContext={instanceContext}
            locale={locale}
          />
        ) : (
          <CellParameterForm
            key={`${group.id}::${refreshTick}::${selectedKey}`}
            deviceId={deviceId}
            group={group}
            instanceContext={instanceContext}
            locale={locale}
          />
        ),
      )}
    </div>
  );
}
