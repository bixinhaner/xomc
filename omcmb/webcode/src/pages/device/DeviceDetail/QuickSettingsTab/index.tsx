import { useMemo, useState } from 'react';
import { Alert, Button, Empty, Popconfirm, Select, Space, Spin, Typography, message } from 'antd';
import { MinusOutlined, PlusOutlined } from '@ant-design/icons';
import { useIntl } from 'react-intl';
import { useQuickSettingsGroups } from '@core/hooks/api/useQuickSettings';
import { useResolvedCellInstances } from '@core/hooks/api/useResolvedCellInstances';
import { useAddObject, useDeleteObject, useParameterSchema } from '@core/hooks/api/useDeviceParameters';
import { useQuickSettingsFeedbackStore } from '@core/store/quickSettingsFeedbackStore';
import CellParameterForm from './CellParameterForm';
import InstanceSelectorForm from './InstanceSelectorForm';
import MultiInstanceTable from './MultiInstanceTable';
import type { QuickSettingsInstanceContext } from './validators';
import { useT } from '@/hooks/useT';

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
export default function QuickSettingsTab({ deviceId, networkType }: QuickSettingsTabProps) {
  const intl = useIntl();
  const t = useT();
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
    // BSC 重用 fapInstance 字段。applyInstanceContext 的非-NR 分支会把第一个 {i}
    // 替换为 fapInstance,而 BSC 的 standardPath 只含一个 {i}(DeviceGSM.Bts.{i}.<leaf>),
    // 正好被完整替换为选中的 BTS 实例号,无需新增独立的 btsInstance 字段。
    fapInstance: (isENB || isBSC) ? selectedInstance : 1,
    cellInstance: isNR ? selectedInstance : undefined,
  };
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

  // BSC BTS 实例增删：复用 useAddObject/useDeleteObject，成功后由 hook
  // 自动 invalidate parameter-schema 查询，useResolvedCellInstances 会重拉。
  const addObjectMutation = useAddObject();
  const deleteObjectMutation = useDeleteObject();
  const btsAddPending = addObjectMutation.isPending;
  const btsDeletePending = deleteObjectMutation.isPending;

  // BSC 下拉显示 "实例号 · IpaUnitId=xxx"，让运维能直接看出 BTS 与 IPA 单元映射。
  // 复用 useResolvedCellInstances 已经发过的同一份 schema 查询（react-query
  // 按 (deviceId, 'DeviceGSM.Bts.') key 去重，不会额外触发请求）。
  const { data: bscBtsSchema } = useParameterSchema(deviceId, BSC_BTS_OBJECT_PREFIX, isBSC);
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

  const handleAddBts = async () => {
    try {
      await addObjectMutation.mutateAsync({ deviceId, objectPath: BSC_BTS_OBJECT_PREFIX });
      message.success(t('device.quickSettings.btsAddSubmitted'));
    } catch (err) {
      message.error(t('device.quickSettings.btsAddFailed', { reason: String(err) }));
    }
  };
  const handleDeleteBts = async () => {
    if (!Number.isFinite(selectedInstance) || selectedInstance <= 0) return;
    try {
      await deleteObjectMutation.mutateAsync({
        deviceId,
        objectPath: `${BSC_BTS_OBJECT_PREFIX}${selectedInstance}.`,
      });
      message.success(t('device.quickSettings.btsDeleteSubmitted', { id: selectedInstance }));
      setUserPickedInstance(null);
    } catch (err) {
      message.error(t('device.quickSettings.btsDeleteFailed', { reason: String(err) }));
    }
  };

  if (error) {
    return (
      <div style={{ padding: 16 }}>
        <Alert type="error" message={t('device.quickSettings.loadGroupsFailed')} description={String(error)} />
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
    <div style={{ padding: 16 }}>
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
                disabled={selectableInstances.length === 0 || btsDeletePending}
              >
                <Button
                  size="small"
                  danger
                  icon={<MinusOutlined />}
                  loading={btsDeletePending}
                  disabled={selectableInstances.length === 0}
                  title={t('device.quickSettings.btsDelete')}
                >
                  {t('device.quickSettings.btsDelete')}
                </Button>
              </Popconfirm>
            </>
          )}
          <Text type="secondary">{selectorHint}</Text>
        </Space>
      )}

      {visibleGroups.map((group) => {
        // 若该 group 是其他 group 的 parentSelector，则把对应子 groups 嵌入到 InstanceSelectorForm 中渲染。
        const childGroups = visibleGroups.filter((g) => g.parentSelector === group.id);
        if (group.style === 'table' && childGroups.length > 0) {
          return (
            <InstanceSelectorForm
              key={`${group.id}::${refreshTick}::${selectedKey}`}
              deviceId={deviceId}
              selectorGroup={group}
              childGroups={childGroups}
              instanceContext={instanceContext}
              locale={locale}
            />
          );
        }
        // 子 groups (parentSelector 非空) 已嵌入到上面的 selector，这里跳过独立渲染。
        if (group.parentSelector) return null;
        return group.multiInstance ? (
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
        );
      })}
    </div>
  );
}
