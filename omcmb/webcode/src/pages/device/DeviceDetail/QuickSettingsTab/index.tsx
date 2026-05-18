import { useState } from 'react';
import { Alert, Empty, InputNumber, Space, Spin, Typography } from 'antd';
import { useIntl } from 'react-intl';
import { useQuickSettingsGroups } from '@core/hooks/api/useQuickSettings';
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

/**
 * 设备详情「快速设置」tab — T-0138 per-paramModel 架构。
 *
 * 数据流:
 *  1. deviceId → useQuickSettingsGroups(deviceId) → 后端 device→product→paramModel 链路解析
 *  2. 返回的 groups 为空时,设备详情页层应不渲染本组件(规则:无 XML 不显示);
 *     本组件按规则做防御性兜底(Empty 占位)
 *  3. ENB(networkType='lte'):顶部显示 FAPService 实例下拉(默认 1,1~12)
 *  4. GNB(networkType='nr'):FAPService 固定 1,不显示下拉
 *  5. 单实例分组 → CellParameterForm(整组 Save)
 *  6. 多实例分组 → MultiInstanceTable(行级 Save / AddObject + Set 两步 / DeleteObject)
 */
export default function QuickSettingsTab({ deviceId, networkType }: QuickSettingsTabProps) {
  const intl = useIntl();
  const locale: 'zh-CN' | 'en-US' = intl.locale === 'en-US' ? 'en-US' : 'zh-CN';

  // ENB:实例 1~12,默认 1;GNB:固定 1
  const [fapInstance, setFapInstance] = useState<number>(1);

  const { data, isLoading, error } = useQuickSettingsGroups(deviceId);

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

  const isENB = networkType === 'lte';

  return (
    <div style={{ padding: 16 }}>
      {isENB && (
        <Space style={{ marginBottom: 16 }}>
          <Text strong>FAPService 实例:</Text>
          <InputNumber
            min={1}
            max={12}
            value={fapInstance}
            onChange={(v) => setFapInstance(Number(v) || 1)}
            style={{ width: 100 }}
          />
          <Text type="secondary">(设备最多 12 个 FAPService)</Text>
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
            key={group.id}
            deviceId={deviceId}
            fapInstance={fapInstance}
            group={group}
            locale={locale}
          />
        ) : (
          <CellParameterForm
            key={group.id}
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
