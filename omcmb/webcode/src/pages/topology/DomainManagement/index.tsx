import { useState, useMemo } from 'react';
import { Button, Form, Input, Modal, Space, Tag, Tree, Typography, message, Tabs, Select } from 'antd';
import { PlusOutlined, EditOutlined, ReloadOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useDomainTree } from '@core/hooks/api/useTopology';
import type { Domain } from '@core/types/topology';
import { useT } from '@/hooks/useT';

interface DomainNode {
  id: string;
  name: string;
  level: number;
  parentName: string;
  deviceCount: number;
  siteCount: number;
  status: 'active' | 'inactive';
  carrier?: string;
  children?: DomainNode[];
}

// Mock domain tree data matching the structure in image 1
const MOCK_DOMAIN_TREE: DomainNode[] = [
  {
    id: 'root',
    name: 'omc-topo',
    level: 0,
    parentName: '-',
    deviceCount: 30000,
    siteCount: 150,
    status: 'active',
    children: [
      {
        id: 'cmcc-domain',
        name: '设备域',
        level: 1,
        parentName: 'omc-topo',
        deviceCount: 12000,
        siteCount: 50,
        status: 'active',
        carrier: 'cmcc',
        children: [
          {
            id: 'cmcc-bj',
            name: '北京移动',
            level: 2,
            parentName: '设备域',
            deviceCount: 2000,
            siteCount: 10,
            status: 'active',
            carrier: 'cmcc',
          },
          {
            id: 'cmcc-sh',
            name: '上海移动',
            level: 2,
            parentName: '设备域',
            deviceCount: 1500,
            siteCount: 8,
            status: 'active',
            carrier: 'cmcc',
          },
          {
            id: 'cmcc-gd',
            name: '广东移动',
            level: 2,
            parentName: '设备域',
            deviceCount: 1800,
            siteCount: 12,
            status: 'active',
            carrier: 'cmcc',
          },
        ],
      },
      {
        id: 'ctcc-domain',
        name: '域',
        level: 1,
        parentName: 'omc-topo',
        deviceCount: 10000,
        siteCount: 40,
        status: 'active',
        carrier: 'ctcc',
        children: [
          {
            id: 'ctcc-js',
            name: '江苏电信',
            level: 2,
            parentName: '域',
            deviceCount: 1500,
            siteCount: 8,
            status: 'active',
            carrier: 'ctcc',
          },
          {
            id: 'ctcc-zj',
            name: '浙江电信',
            level: 2,
            parentName: '域',
            deviceCount: 1200,
            siteCount: 7,
            status: 'active',
            carrier: 'ctcc',
          },
        ],
      },
      {
        id: 'cucc-domain',
        name: '未分组设备',
        level: 1,
        parentName: 'omc-topo',
        deviceCount: 8000,
        siteCount: 30,
        status: 'active',
        carrier: 'cucc',
      },
    ],
  },
];

interface DeviceRow extends Record<string, unknown> {
  id: string;
  serialNumber: string;
  manufacturer: string;
  modelName: string;
  ipAddress: string;
  status: string;
  carrier: string;
  technology: string;
  siteName: string;
}

// Mock device data
const generateMockDevices = (carrier: string, count: number): DeviceRow[] => {
  const manufacturers = ['Huawei', 'ZTE', 'Ericsson', 'Nokia', 'BaiCells'];
  const models = ['AAU5613', 'ZXSDR-B8200', 'AIR6488', 'FXEB', 'BC-ENB-100'];
  const sites = ['Beijing-Site-01', 'Beijing-Site-02', 'Shanghai-Site-01', 'Guangzhou-Site-01'];
  const statuses = ['online', 'offline', 'active', 'inactive'];

  return Array.from({ length: count }, (_, i) => ({
    id: `dev-${carrier}-${i}`,
    serialNumber: `${carrier.toUpperCase()}-${String(i + 1).padStart(6, '0')}`,
    manufacturer: manufacturers[Math.floor(Math.random() * manufacturers.length)],
    modelName: models[Math.floor(Math.random() * models.length)],
    ipAddress: `10.${carrier === 'cmcc' ? '1' : carrier === 'ctcc' ? '2' : '3'}.${Math.floor(i / 256)}.${i % 256}`,
    status: statuses[Math.floor(Math.random() * statuses.length)],
    carrier,
    technology: Math.random() > 0.5 ? 'LTE' : 'NR',
    siteName: sites[Math.floor(Math.random() * sites.length)],
  }));
};

const MOCK_DEVICES: Record<string, DeviceRow[]> = {
  'cmcc': generateMockDevices('cmcc', 50),
  'ctcc': generateMockDevices('ctcc', 30),
  'cucc': generateMockDevices('cucc', 20),
};

function buildAntTreeData(nodes: DomainNode[]): DataNode[] {
  return nodes.map((n) => ({
    title: (
      <span>
        {n.name}
        {n.deviceCount > 0 && (
          <span style={{ fontSize: 11, color: '#8c8c8c', marginLeft: 6 }}>
            {n.deviceCount}
          </span>
        )}
      </span>
    ),
    key: n.id,
    children: n.children ? buildAntTreeData(n.children) : undefined,
  }));
}

function findDomain(nodes: DomainNode[], id: string): DomainNode | null {
  for (const n of nodes) {
    if (n.id === id) return n;
    if (n.children) {
      const found = findDomain(n.children, id);
      if (found) return found;
    }
  }
  return null;
}

export default function DomainManagement() {
  const t = useT();
  const [selectedDomainId, setSelectedDomainId] = useState<string>('');
  const [modalVisible, setModalVisible] = useState(false);
  const [editingDomain, setEditingDomain] = useState<DomainNode | null>(null);
  const [form] = Form.useForm();
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [activeTab, setActiveTab] = useState<'devices' | 'sites'>('devices');

  const { data: domainTreeData } = useDomainTree();
  void (null as unknown as Domain);
  void domainTreeData;

  const selectedDomain = findDomain(MOCK_DOMAIN_TREE, selectedDomainId);
  const carrier = selectedDomain?.carrier || 'cmcc';
  const deviceData = MOCK_DEVICES[carrier] || MOCK_DEVICES.cmcc;

  // Filter fields matching image 2
  const filterFields: FilterField[] = useMemo(() => [
    {
      name: 'manufacturer',
      label: t('table.manufacturer') || '厂商',
      type: 'select',
      options: [
        { label: 'Huawei', value: 'Huawei' },
        { label: 'ZTE', value: 'ZTE' },
        { label: 'Ericsson', value: 'Ericsson' },
        { label: 'Nokia', value: 'Nokia' },
        { label: 'BaiCells', value: 'BaiCells' },
      ],
    },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.online') || '在线', value: 'online' },
        { label: t('status.offline') || '离线', value: 'offline' },
        { label: t('status.active') || '激活', value: 'active' },
        { label: t('status.inactive') || '未激活', value: 'inactive' },
      ],
    },
    {
      name: 'technology',
      label: t('device.technology') || '制式',
      type: 'select',
      options: [
        { label: 'LTE', value: 'LTE' },
        { label: 'NR', value: 'NR' },
      ],
    },
  ], [t]);

  const filteredDevices = useMemo(() => {
    return deviceData.filter((device) => {
      if (filters.manufacturer && device.manufacturer !== filters.manufacturer) return false;
      if (filters.status && device.status !== filters.status) return false;
      if (filters.technology && device.technology !== filters.technology) return false;
      return true;
    });
  }, [deviceData, filters]);

  const deviceColumns: DataTableColumn<DeviceRow>[] = useMemo(() => [
    {
      key: 'serialNumber',
      title: t('device.sn') || '设备SN',
      dataIndex: 'serialNumber',
      width: 140,
      mono: true,
      copyable: true
    },
    {
      key: 'manufacturer',
      title: t('table.manufacturer') || '厂商',
      dataIndex: 'manufacturer',
      width: 100
    },
    {
      key: 'modelName',
      title: t('device.model') || '型号',
      dataIndex: 'modelName',
      width: 130
    },
    {
      key: 'ipAddress',
      title: 'IP地址',
      dataIndex: 'ipAddress',
      width: 130,
      mono: true,
    },
    {
      key: 'technology',
      title: t('device.technology') || '制式',
      dataIndex: 'technology',
      width: 80,
      render: (val) => <Tag color={val === 'NR' ? 'blue' : 'green'}>{val as string}</Tag>,
    },
    {
      key: 'siteName',
      title: t('topology.site.name') || '站点',
      dataIndex: 'siteName',
      width: 150,
      ellipsis: true,
    },
    {
      key: 'status',
      title: t('table.status'),
      dataIndex: 'status',
      width: 90,
      render: (val) => {
        const colorMap: Record<string, string> = {
          online: 'success',
          offline: 'default',
          active: 'processing',
          inactive: 'error',
        };
        const labelMap: Record<string, string> = {
          online: '在线',
          offline: '离线',
          active: '激活',
          inactive: '未激活',
        };
        return <Tag color={colorMap[val as string]}>{labelMap[val as string] || String(val ?? '')}</Tag>;
      },
    },
    {
      key: 'action',
      title: t('table.operation'),
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Button type="link" size="small" onClick={() => void message.info(`查看设备: ${record.serialNumber}`)}>
            {t('common.view')}
          </Button>
          <Button type="link" size="small" onClick={() => void message.info(`配置设备: ${record.serialNumber}`)}>
            {t('common.config')}
          </Button>
        </Space>
      ),
    },
  ], [t]);

  // Site columns for sites tab
  const siteColumns: DataTableColumn<Record<string, unknown>>[] = useMemo(() => [
    { key: 'name', title: t('table.name'), dataIndex: 'name', width: 200 },
    { key: 'address', title: t('topology.site.address'), dataIndex: 'address', width: 250 },
    { key: 'deviceCount', title: t('topology.site.deviceCount'), dataIndex: 'deviceCount', width: 100 },
    {
      key: 'status',
      title: t('table.status'),
      dataIndex: 'status',
      width: 90,
      render: (val) => <Tag color="success">{val as string}</Tag>,
    },
  ], [t]);

  const openEdit = (domain: DomainNode) => {
    setEditingDomain(domain);
    form.setFieldsValue({ name: domain.name });
    setModalVisible(true);
  };

  const handleSave = () => {
    form.validateFields().then(() => {
      void message.success(t('common.save'));
      setModalVisible(false);
    }).catch(() => undefined);
  };

  const treePanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{ padding: '12px 12px 8px', borderBottom: '1px solid #f0f0f0' }}>
        <Typography.Text strong style={{ fontSize: 13 }}>{t('nav.topology.domain')}</Typography.Text>
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: '8px 4px' }}>
        <Tree
          treeData={buildAntTreeData(MOCK_DOMAIN_TREE)}
          onSelect={(keys) => {
            setSelectedDomainId(keys[0] as string ?? '');
            setActiveTab('devices');
            setFilters({});
          }}
          defaultExpandAll
          showLine
          selectedKeys={selectedDomainId ? [selectedDomainId] : []}
        />
      </div>
    </div>
  );

  return (
    <TreeListPageLayout tree={treePanel}>
      {selectedDomain ? (
        <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
          {/* Header matching image 2 */}
          <div style={{
            padding: '12px 16px',
            borderBottom: '1px solid #f0f0f0',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            background: '#fafafa'
          }}>
            <Space direction="vertical" size={0}>
              <Typography.Text strong style={{ fontSize: 15 }}>
                {t('nav.topology.domain')} - {selectedDomain.name}
              </Typography.Text>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {t('table.total')}: {selectedDomain.deviceCount} | {t('topology.site.name')}: {selectedDomain.siteCount}
              </Typography.Text>
            </Space>
            <Space>
              <Button size="small" icon={<ReloadOutlined />} onClick={() => void message.info(t('common.refresh'))}>
                {t('common.refresh')}
              </Button>
              <Button
                size="small"
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => void message.info('添加设备')}
              >
                {t('common.add')}
              </Button>
              <Button
                size="small"
                icon={<EditOutlined />}
                onClick={() => openEdit(selectedDomain)}
              >
                {t('common.edit')}
              </Button>
            </Space>
          </div>

          {/* Tabs for devices and sites */}
          <Tabs
            activeKey={activeTab}
            onChange={(key) => setActiveTab(key as 'devices' | 'sites')}
            style={{ margin: 0 }}
            items={[
              {
                key: 'devices',
                label: `${t('device.list')} (${filteredDevices.length})`,
              },
              {
                key: 'sites',
                label: `${t('nav.topology.site')} (${selectedDomain.siteCount})`,
              },
            ]}
          />

          {/* Filter bar */}
          <div style={{ padding: '12px 16px 8px' }}>
            <FilterBar
              filterId="domain-devices"
              fields={filterFields}
              onSearch={(vals) => setFilters(vals)}
              onReset={() => setFilters({})}
            />
          </div>

          {/* Data table */}
          <div style={{ flex: 1, overflow: 'hidden' }}>
            {activeTab === 'devices' ? (
              <DataTable<DeviceRow>
                tableId="domain-devices"
                columns={deviceColumns}
                dataSource={filteredDevices}
                rowKey="id"
                total={filteredDevices.length}
                currentPage={page}
                pageSize={pageSize}
                onPageChange={(p) => setPage(p)}
                onRefresh={() => void message.info(t('common.refresh'))}
                scroll={{ x: 1000 }}
              />
            ) : (
              <DataTable
                tableId="domain-sites"
                columns={siteColumns}
                dataSource={[]}
                rowKey="id"
                total={0}
                currentPage={1}
                pageSize={20}
                scroll={{ x: 800 }}
              />
            )}
          </div>
        </div>
      ) : (
        <div style={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          flexDirection: 'column',
          gap: 12
        }}>
          <img
            src="/images/empty-tree.svg"
            alt=""
            style={{ width: 120, height: 120, opacity: 0.5 }}
            onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; }}
          />
          <Typography.Text type="secondary">{t('common.pleaseSelect')}</Typography.Text>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {t('nav.topology.domain')} {t('common.tree')}
          </Typography.Text>
        </div>
      )}

      <Modal
        title={editingDomain ? `${t('common.edit')}: ${editingDomain.name}` : t('common.add')}
        open={modalVisible}
        onOk={handleSave}
        onCancel={() => { setModalVisible(false); form.resetFields(); }}
        okText={t('common.save')}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item label={t('table.name')} name="name" rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('table.type')} name="level" rules={[{ required: true }]}>
            <Select style={{ width: '100%' }} placeholder="请选择">
              <Select.Option value={1}>一级域</Select.Option>
              <Select.Option value={2}>二级域</Select.Option>
              <Select.Option value={3}>三级域</Select.Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </TreeListPageLayout>
  );
}
