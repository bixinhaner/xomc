import { useState, useMemo } from 'react';
import { Button, Tree, Tag, Space, Input } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import type { Device } from '@/types/device';
import { useT } from '@/hooks/useT';

const classificationTree: DataNode[] = [
  {
    key: 'all', title: '全部设备',
    children: [
      {
        key: '4g', title: '4G设备 (LTE)',
        children: [
          { key: 'enb', title: 'eNB基站 (35台)' },
          { key: 'rru', title: 'RRU (120台)' },
          { key: 'bbu', title: 'BBU (22台)' },
        ],
      },
      {
        key: '5g', title: '5G设备 (NR)',
        children: [
          { key: 'gnb', title: 'gNB基站 (18台)' },
          { key: 'aau', title: 'AAU (54台)' },
          { key: 'cu', title: 'CU (6台)' },
          { key: 'du', title: 'DU (18台)' },
        ],
      },
      {
        key: 'cpe', title: 'CPE终端',
        children: [
          { key: 'indoor-cpe', title: '室内CPE (45台)' },
          { key: 'outdoor-cpe', title: '室外CPE (30台)' },
        ],
      },
    ],
  },
];

const mockDevices: Partial<Device>[] = [
  { id: 'd-001', sn: 'ENB00001', name: '北京-eNB-0001', vendor: '华为', productType: 'eNB', networkType: 'LTE', connStatus: 'online', alarmLevel: 'none', region: '北京', softwareVersion: 'V100R011C10SPC200' },
  { id: 'd-002', sn: 'ENB00002', name: '北京-eNB-0002', vendor: '华为', productType: 'eNB', networkType: 'LTE', connStatus: 'online', alarmLevel: 'minor', region: '北京', softwareVersion: 'V100R011C10SPC200' },
  { id: 'd-003', sn: 'ENB00003', name: '上海-eNB-0001', vendor: '中兴', productType: 'eNB', networkType: 'LTE', connStatus: 'offline', alarmLevel: 'major', region: '上海', softwareVersion: 'V100R011C10SPC100' },
  { id: 'd-004', sn: 'ENB00004', name: '广州-eNB-0001', vendor: '爱立信', productType: 'eNB', networkType: 'LTE', connStatus: 'online', alarmLevel: 'none', region: '广州', softwareVersion: 'V100R011C10SPC200' },
  { id: 'd-005', sn: 'GNB00001', name: '北京-gNB-0001', vendor: '华为', productType: 'gNB', networkType: 'NR', connStatus: 'online', alarmLevel: 'none', region: '北京', softwareVersion: 'V200R001C10SPC100' },
];

const connStatusColorMap: Record<string, string> = { online: 'green', offline: 'red' };

const alarmColorMap: Record<string, string> = { none: 'default', warning: 'gold', minor: 'yellow', major: 'orange', critical: 'red' };

export default function DeviceClassification() {
  const t = useT();
  const [selectedCategory, setSelectedCategory] = useState<string>('all');
  const [searchText, setSearchText] = useState('');

  const filteredDevices = mockDevices.filter((d) => {
    if (!d.sn || !d.name) return false;
    if (searchText && !d.sn.includes(searchText) && !d.name.includes(searchText)) return false;
    if (selectedCategory === 'all') return true;
    if (selectedCategory === '4g') return d.networkType === 'LTE';
    if (selectedCategory === '5g') return d.networkType === 'NR';
    if (selectedCategory === 'enb') return d.productType === 'eNB';
    if (selectedCategory === 'gnb') return d.productType === 'gNB';
    if (selectedCategory === 'rru') return d.productType === 'RRU';
    if (selectedCategory === 'aau') return d.productType === 'AAU';
    return true;
  });

  const columns: DataTableColumn<Partial<Device> & Record<string, unknown>>[] = useMemo(() => [
    { key: 'sn', title: t('device.sn'), dataIndex: 'sn', width: 130, mono: true },
    { key: 'name', title: t('device.name'), dataIndex: 'name', ellipsis: true },
    { key: 'vendor', title: t('table.vendor'), dataIndex: 'vendor', width: 90 },
    { key: 'productType', title: t('device.productType'), dataIndex: 'productType', width: 90 },
    { key: 'networkType', title: t('table.type'), dataIndex: 'networkType', width: 90 },
    { key: 'region', title: t('table.region'), dataIndex: 'region', width: 80 },
    {
      key: 'connStatus', title: t('device.connStatus'), dataIndex: 'connStatus', width: 100,
      render: (val) => <Tag color={connStatusColorMap[String(val)] ?? 'default'}>{String(val) === 'online' ? t('status.online') : t('status.offline')}</Tag>,
    },
    {
      key: 'alarmLevel', title: t('alarm.severity'), dataIndex: 'alarmLevel', width: 100,
      render: (val) => <Tag color={alarmColorMap[String(val)] ?? 'default'}>{String(val) !== 'none' ? t(`alarm.severity.${String(val)}`) : t('status.active')}</Tag>,
    },
    {
      key: 'softwareVersion', title: t('device.softwareVersion'), dataIndex: 'softwareVersion', width: 200,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>,
    },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 80, fixed: 'right',
      render: () => (
        <Button type="link" size="small" icon={<EditOutlined />}>
          {t('common.edit')}
        </Button>
      ),
    },
  ], [t]);

  const treePanel = (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div style={{ padding: '12px 8px', borderBottom: '1px solid #f0f0f0', display: 'flex', gap: 6, alignItems: 'center' }}>
        <span style={{ fontWeight: 500, fontSize: 14 }}>{t('nav.system.deviceClass')}</span>
        <Button type="link" size="small" icon={<PlusOutlined />} style={{ marginLeft: 'auto' }}>{t('common.add')}</Button>
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: 8 }}>
        <Tree
          treeData={classificationTree}
          defaultExpandAll
          selectedKeys={[selectedCategory]}
          onSelect={(keys) => {
            if (keys.length > 0) setSelectedCategory(String(keys[0]));
          }}
        />
      </div>
    </div>
  );

  return (
    <TreeListPageLayout tree={treePanel}>
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        <div style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center', background: '#fafafa' }}>
          <span style={{ fontWeight: 500 }}>
            {t('table.total')} ({filteredDevices.length})
          </span>
          <Space>
            <Input
              placeholder={t('common.search')}
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              style={{ width: 200 }}
              size="small"
              allowClear
            />
            <Button type="primary" size="small" icon={<PlusOutlined />}>{t('common.add')}</Button>
            <Button size="small" danger icon={<DeleteOutlined />} disabled>{t('common.delete')}</Button>
          </Space>
        </div>
        <div style={{ flex: 1, overflow: 'auto' }}>
          <DataTable
            tableId="device-classification-list"
            columns={columns}
            dataSource={filteredDevices as (Partial<Device> & Record<string, unknown>)[]}
            loading={false}
            rowKey="id"
            total={filteredDevices.length}
            pageSize={20}
            currentPage={1}
            selectable
            scroll={{ x: 1000 }}
          />
        </div>
      </div>
    </TreeListPageLayout>
  );
}
