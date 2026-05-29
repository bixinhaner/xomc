import { useMemo, useState } from 'react';
import { Alert, Empty, Select, Space, Spin, Typography } from 'antd';
import { useIntl } from 'react-intl';
import { useQuickSettingsGroups } from '@core/hooks/api/useQuickSettings';
import { useParameterSchema } from '@core/hooks/api/useDeviceParameters';
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

const LTE_FAPSERVICE_PREFIX = 'Device.Services.FAPService.';
const NR_CELLCONFIG_PREFIX = 'Device.Services.FAPService.1.CellConfig.';

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
 */
export default function QuickSettingsTab({ deviceId, networkType }: QuickSettingsTabProps) {
  const intl = useIntl();
  const locale: 'zh-CN' | 'en-US' = intl.locale === 'en-US' ? 'en-US' : 'zh-CN';
  const normalizedNetworkType = normalizeQuickSettingsNetworkType(networkType);

  const isENB = normalizedNetworkType === 'lte';
  const isNR = normalizedNetworkType === 'nr';

  // LTE 选择 FAPService，NR 选择 CellConfig 小区实例。
  const [userPickedInstance, setUserPickedInstance] = useState<number | null>(null);

  // 头部"刷新"按钮 bump 的 tick → 拼进子组件 key 强制 remount，清掉 form/rowEdits 等组件内 state
  const refreshTick = useQuickSettingsFeedbackStore((s) => s.refreshTicks[deviceId] ?? 0);

  const { data, isLoading, error } = useQuickSettingsGroups(deviceId);

  // ENB 才查 FAPService 实例 schema；GNB 直接跳过（避免 path_prefix 为空时拉全量 schema）
  const { data: fapSchema, isLoading: fapSchemaLoading } = useParameterSchema(
    deviceId,
    LTE_FAPSERVICE_PREFIX,
    isENB,
  );

  const { data: nrCellSchema, isLoading: nrCellSchemaLoading } = useParameterSchema(
    deviceId,
    NR_CELLCONFIG_PREFIX,
    isNR,
  );

  const lteInstances = useMemo(() => {
    if (!isENB || !fapSchema) {
      return [] as number[];
    }
    const objEntry = fapSchema.objects.find((o) => o.path === LTE_FAPSERVICE_PREFIX);
    const candidateInstances = objEntry?.currentInstances ?? [];
    // 即便 objects 没给出 currentInstances(老 schema 数据),也尝试从 parameters 路径推断
    const fallbackSet = new Set<number>();
    if (candidateInstances.length === 0) {
      for (const p of fapSchema.parameters) {
        const m = /^Device\.Services\.FAPService\.(\d+)\./.exec(p.path);
        if (m) fallbackSet.add(Number(m[1]));
      }
    }
    const allInstances = candidateInstances.length > 0 ? candidateInstances : Array.from(fallbackSet).sort((a, b) => a - b);

    return allInstances;
  }, [isENB, fapSchema]);

  const nrCellInstances = useMemo(() => {
    if (!isNR || !nrCellSchema) {
      return [] as number[];
    }
    const objEntry = nrCellSchema.objects.find((o) => o.path === NR_CELLCONFIG_PREFIX);
    const candidateInstances = objEntry?.currentInstances ?? [];
    if (candidateInstances.length > 0) {
      return [...candidateInstances].sort((a, b) => a - b);
    }

    const fallbackSet = new Set<number>();
    for (const p of nrCellSchema.parameters) {
      const m = /^Device\.Services\.FAPService\.1\.CellConfig\.(\d+)\./.exec(p.path);
      if (m) fallbackSet.add(Number(m[1]));
    }
    return Array.from(fallbackSet).sort((a, b) => a - b);
  }, [isNR, nrCellSchema]);

  const selectableInstances = isENB ? lteInstances : isNR ? nrCellInstances : [];
  const selectedInstance =
    userPickedInstance !== null && selectableInstances.includes(userPickedInstance)
      ? userPickedInstance
      : selectableInstances[0] ?? 1;

  const instanceContext: QuickSettingsInstanceContext = {
    networkType: normalizedNetworkType,
    fapInstance: isENB ? selectedInstance : 1,
    cellInstance: isNR ? selectedInstance : undefined,
  };
  const selectorLabel = isENB ? 'FAPService 实例:' : '小区实例:';
  const selectorHint = isENB
    ? `(显示设备上实际存在的 FAPService 实例 · 共 ${lteInstances.length} 个)`
    : `(按 Device.Services.FAPService.1.CellConfig.{i}. 枚举 · 共 ${nrCellInstances.length} 个小区)`;
  const selectorLoading = isENB ? fapSchemaLoading : nrCellSchemaLoading;
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

  const groups = data?.groups ?? [];
  if (groups.length === 0) {
    return (
      <div style={{ padding: 16 }}>
        <Empty description={`该设备的 paramModel(${data?.paramModel ?? '未知'})未配置快速设置分组`} />
      </div>
    );
  }

  return (
    <div style={{ padding: 16 }}>
      {(isENB || isNR) && (
        <Space style={{ marginBottom: 16 }}>
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

      {groups.map((group) =>
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
