import { useState } from 'react';
import { Alert, Empty, InputNumber, Space, Spin, Typography } from 'antd';
import { useIntl } from 'react-intl';
import { useQuickSettingsGroups } from '@core/hooks/api/useQuickSettings';
import type { TechCode } from '@core/types/quicksettings';
import CellParameterForm from './CellParameterForm';
import MultiInstanceTable from './MultiInstanceTable';

const { Text } = Typography;

interface QuickSettingsTabProps {
  deviceId: string;
  /** 设备 networkType：'eNB' | 'gNB' | 其他 */
  networkType: string;
}

const NETWORK_TYPE_TO_TECH: Record<string, TechCode> = {
  eNB: 'lte',
  gNB: 'nr',
};

/**
 * 设备详情「快速设置」tab — T-0138。
 *
 * 数据流：
 *  1. networkType → TechCode → 调 useQuickSettingsGroups(tech) 拉分组
 *  2. ENB：显示 FAPService 实例下拉（默认 1，1~12），切换后刷新表单/列表值
 *  3. GNB：固定 FAPService.1，不显示下拉
 *  4. 单实例分组 → CellParameterForm（整组 Save）
 *  5. 多实例分组 → MultiInstanceTable（行级 Save / AddObject + Set 两步 / DeleteObject）
 */
export default function QuickSettingsTab({ deviceId, networkType }: QuickSettingsTabProps) {
  const intl = useIntl();
  const locale: 'zh-CN' | 'en-US' = intl.locale === 'en-US' ? 'en-US' : 'zh-CN';
  const tech = NETWORK_TYPE_TO_TECH[networkType];

  // ENB：实例 1~12，默认 1；GNB：固定 1
  const [fapInstance, setFapInstance] = useState<number>(1);

  const { data, isLoading, error } = useQuickSettingsGroups(tech);

  if (!tech) {
    return (
      <div style={{ padding: 16 }}>
        <Alert
          type="info"
          message="该设备制式暂不支持快速设置"
          description={`networkType=${networkType}（仅 eNB / gNB 提供分组化参数）`}
        />
      </div>
    );
  }

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
        <Empty description="该制式暂无快速设置分组" />
      </div>
    );
  }

  const isENB = tech === 'lte';

  return (
    <div style={{ padding: 16 }}>
      {isENB && (
        <Space style={{ marginBottom: 16 }}>
          <Text strong>FAPService 实例：</Text>
          <InputNumber
            min={1}
            max={12}
            value={fapInstance}
            onChange={(v) => setFapInstance(Number(v) || 1)}
            style={{ width: 100 }}
          />
          <Text type="secondary">（设备最多 12 个 FAPService）</Text>
        </Space>
      )}
      {!isENB && (
        <Alert
          type="info"
          message="GNB 设备 FAPService 固定为 1，仅显示小区参数分组"
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
