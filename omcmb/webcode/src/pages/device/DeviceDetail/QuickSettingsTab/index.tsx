import { useMemo, useState } from 'react';
import { Alert, Empty, Select, Space, Spin, Typography } from 'antd';
import { useIntl } from 'react-intl';
import { useQuickSettingsGroups } from '@core/hooks/api/useQuickSettings';
import { useParameterSchema } from '@core/hooks/api/useDeviceParameters';
import { useQuickSettingsFeedbackStore } from '@core/store/quickSettingsFeedbackStore';
import CellParameterForm from './CellParameterForm';
import MultiInstanceTable from './MultiInstanceTable';

const { Text } = Typography;

interface QuickSettingsTabProps {
  deviceId: string;
  /**
   * 设备 networkType 字段值(deviceApi mapBackendDevice 直接取后端 technology)。
   * 实际取值为 'lte' / 'nr' / 其他。用于决定是否显示 FAPService 实例下拉。
   */
  networkType: string;
}

// FAPService 有效性判定：ECI/TAC/PCI 三个关键参数都非空才认为是有效实例。
const FAPSERVICE_PREFIX = 'Device.Services.FAPService.';
const KEY_PARAM_SUFFIXES = [
  'CellConfig.LTE.RAN.Common.CellIdentity', // ECI
  'CellConfig.LTE.EPC.TAC',
  'CellConfig.LTE.RAN.RF.PhyCellID',
];

function isEmptyValue(v: string | null | undefined): boolean {
  if (v === null || v === undefined) return true;
  const t = v.trim();
  return t === '' || t === '0';
}

/**
 * 设备详情「快速设置」tab — T-0138 per-paramModel 架构。
 *
 * 数据流:
 *  1. deviceId → useQuickSettingsGroups(deviceId) → 后端 device→product→paramModel 链路解析
 *  2. 返回的 groups 为空时,设备详情页层应不渲染本组件(规则:无 XML 不显示);
 *     本组件按规则做防御性兜底(Empty 占位)
 *  3. ENB(networkType='lte'):顶部显示 FAPService 实例下拉(枚举实际存在且 ECI/TAC/PCI 非空的实例)
 *  4. GNB(networkType='nr'):FAPService 固定 1,不显示下拉
 *  5. 单实例分组 → CellParameterForm(整组 Save)
 *  6. 多实例分组 → MultiInstanceTable(行级 Save / AddObject + Set 两步 / DeleteObject)
 */
export default function QuickSettingsTab({ deviceId, networkType }: QuickSettingsTabProps) {
  const intl = useIntl();
  const locale: 'zh-CN' | 'en-US' = intl.locale === 'en-US' ? 'en-US' : 'zh-CN';

  const isENB = networkType === 'lte';

  // ENB:用户在 Select 中选的实例;为 null 时落到第一个有效实例;GNB 固定走 1
  const [userPickedInstance, setUserPickedInstance] = useState<number | null>(null);

  // 头部"刷新"按钮 bump 的 tick → 拼进子组件 key 强制 remount，清掉 form/rowEdits 等组件内 state
  const refreshTick = useQuickSettingsFeedbackStore((s) => s.refreshTicks[deviceId] ?? 0);

  const { data, isLoading, error } = useQuickSettingsGroups(deviceId);

  // ENB 才查 FAPService 实例 schema；GNB 直接跳过（避免 path_prefix 为空时拉全量 schema）
  const { data: fapSchema, isLoading: fapSchemaLoading } = useParameterSchema(
    deviceId,
    FAPSERVICE_PREFIX,
    isENB,
  );

  // 派生：(a) 设备上实际存在的 FAPService 实例集；(b) 过滤后的有效实例集
  const { validInstances } = useMemo(() => {
    if (!isENB || !fapSchema) {
      return { validInstances: [] as number[] };
    }
    const objEntry = fapSchema.objects.find((o) => o.path === FAPSERVICE_PREFIX);
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

    const paramByPath = new Map<string, string | null | undefined>();
    for (const p of fapSchema.parameters) paramByPath.set(p.path, p.currentValue);

    const valid = allInstances.filter((inst) => {
      for (const suffix of KEY_PARAM_SUFFIXES) {
        const path = `${FAPSERVICE_PREFIX}${inst}.${suffix}`;
        if (isEmptyValue(paramByPath.get(path))) return false;
      }
      return true;
    });
    return { validInstances: valid };
  }, [isENB, fapSchema]);

  // 纯派生：用户选项若仍在有效集合中则优先用,否则回退到第一个有效实例;无有效实例时退化为 1
  const fapInstance: number = isENB
    ? userPickedInstance !== null && validInstances.includes(userPickedInstance)
      ? userPickedInstance
      : validInstances[0] ?? 1
    : 1;

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
      {isENB && (
        <Space style={{ marginBottom: 16 }}>
          <Text strong>FAPService 实例:</Text>
          <Select
            value={validInstances.includes(fapInstance) ? fapInstance : undefined}
            onChange={(v) => setUserPickedInstance(Number(v) || null)}
            style={{ width: 120 }}
            loading={fapSchemaLoading}
            disabled={fapSchemaLoading || validInstances.length === 0}
            options={validInstances.map((n) => ({ value: n, label: String(n) }))}
            placeholder={fapSchemaLoading ? '加载中...' : '无有效实例'}
            notFoundContent="无有效实例"
          />
          <Text type="secondary">
            (仅显示 ECI/TAC/PCI 均非空的实例 · 共 {validInstances.length} 个有效)
          </Text>
        </Space>
      )}
      {!isENB && (
        <Alert
          type="info"
          message={`paramModel: ${data?.paramModel};FAPService 固定为 1`}
          showIcon
          style={{ marginBottom: 16 }}
        />
      )}

      {groups.map((group) =>
        group.multiInstance ? (
          <MultiInstanceTable
            key={`${group.id}::${refreshTick}`}
            deviceId={deviceId}
            fapInstance={fapInstance}
            group={group}
            locale={locale}
          />
        ) : (
          <CellParameterForm
            key={`${group.id}::${refreshTick}`}
            deviceId={deviceId}
            fapInstance={fapInstance}
            group={group}
            locale={locale}
          />
        ),
      )}
    </div>
  );
}
