import { useMemo, useState } from 'react';
import { Card, Table, Select, Space, Button, message, Tag } from 'antd';
import {
  useIndicatorList,
  useEnabledIndicators,
  useSetEnabledIndicators,
} from '@core/hooks/api/useIndicatorsLibrary';
import type { DeviceType, IndicatorInfo } from '@core/types/indicatorLibrary';

const DEVICE_TYPE_OPTIONS: { label: string; value: DeviceType }[] = [
  { label: 'ENB', value: 'ENB' },
  { label: 'GSM', value: 'GSM' },
  { label: 'GNB', value: 'GNB' },
];

const OPERATOR_OPTIONS = [
  { label: 'CMCC', value: 'cmcc' },
  { label: 'CTCC', value: 'ctcc' },
  { label: 'CUCC', value: 'cucc' },
  { label: '默认', value: 'default' },
];

export default function EnabledIndicatorsTab() {
  const [deviceType, setDeviceType] = useState<DeviceType>('ENB');
  const [operatorCode, setOperatorCode] = useState('default');

  const { data: indList, isLoading } = useIndicatorList(deviceType);
  const { data: enabledData } = useEnabledIndicators(deviceType, operatorCode);
  const setMut = useSetEnabledIndicators();

  const enabledSet = useMemo(() => new Set(enabledData?.items || []), [enabledData]);
  const items = indList?.items || [];

  const [selected, setSelected] = useState<React.Key[]>([]);

  // 同步初始选中（首次加载或切换 deviceType/operatorCode）
  // 使用受控选中：先按当前 enabled 集合初始化
  useMemo(() => {
    setSelected(Array.from(enabledSet));
  }, [enabledSet]);

  const handleApply = async (enable: boolean) => {
    const ids = selected.map(String);
    if (ids.length === 0) {
      message.warning('请先勾选至少一项');
      return;
    }
    try {
      await setMut.mutateAsync({ deviceType, operatorCode, indicatorIds: ids, enable });
      message.success(`已${enable ? '启用' : '停用'} ${ids.length} 项`);
    } catch (e) {
      message.error((e as Error).message);
    }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 140 },
    { title: '中文名', dataIndex: 'cnName', width: 220 },
    { title: '英文名', dataIndex: 'enName', ellipsis: true },
    {
      title: '当前启用',
      dataIndex: 'id',
      width: 120,
      render: (id: string) => (enabledSet.has(id) ? <Tag color="success">已启用</Tag> : <Tag>未启用</Tag>),
    },
  ];

  return (
    <Card
      size="small"
      title="启用指标 / Enabled Indicators"
      extra={
        <Space>
          <Select
            value={deviceType}
            onChange={(v) => setDeviceType(v)}
            options={DEVICE_TYPE_OPTIONS}
            style={{ width: 100 }}
          />
          <Select
            value={operatorCode}
            onChange={(v) => setOperatorCode(v)}
            options={OPERATOR_OPTIONS}
            style={{ width: 110 }}
          />
          <Button
            type="primary"
            loading={setMut.isPending}
            onClick={() => void handleApply(true)}
          >
            启用所选
          </Button>
          <Button
            danger
            loading={setMut.isPending}
            onClick={() => void handleApply(false)}
          >
            停用所选
          </Button>
        </Space>
      }
    >
      <Table<IndicatorInfo>
        rowKey="id"
        loading={isLoading}
        columns={columns}
        dataSource={items}
        size="small"
        pagination={{ pageSize: 50, showSizeChanger: true }}
        rowSelection={{
          selectedRowKeys: selected,
          onChange: (keys) => setSelected(keys),
          getCheckboxProps: (row) => ({ disabled: false, name: row.id }),
        }}
      />
    </Card>
  );
}
