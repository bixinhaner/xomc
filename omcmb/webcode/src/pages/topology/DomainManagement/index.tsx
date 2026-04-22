import { useState, useMemo } from 'react';
import { Button, Descriptions, Form, Input, InputNumber, Modal, Space, Tag, Tree, Typography, message } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
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
  children?: DomainNode[];
}

const MOCK_DOMAIN_TREE: DomainNode[] = [
  {
    id: 'cn',
    name: '中国区',
    level: 1,
    parentName: '-',
    deviceCount: 120,
    siteCount: 45,
    status: 'active',
    children: [
      {
        id: 'bj',
        name: '北京市',
        level: 2,
        parentName: '中国区',
        deviceCount: 35,
        siteCount: 12,
        status: 'active',
        children: [
          { id: 'bj-cy', name: '朝阳区', level: 3, parentName: '北京市', deviceCount: 18, siteCount: 6, status: 'active' },
          { id: 'bj-hd', name: '海淀区', level: 3, parentName: '北京市', deviceCount: 17, siteCount: 6, status: 'active' },
        ],
      },
      {
        id: 'sh',
        name: '上海市',
        level: 2,
        parentName: '中国区',
        deviceCount: 42,
        siteCount: 15,
        status: 'active',
        children: [
          { id: 'sh-pd', name: '浦东新区', level: 3, parentName: '上海市', deviceCount: 22, siteCount: 8, status: 'active' },
          { id: 'sh-ja', name: '静安区', level: 3, parentName: '上海市', deviceCount: 20, siteCount: 7, status: 'active' },
        ],
      },
      {
        id: 'gd',
        name: '广东省',
        level: 2,
        parentName: '中国区',
        deviceCount: 43,
        siteCount: 18,
        status: 'active',
        children: [
          { id: 'gz', name: '广州市', level: 3, parentName: '广东省', deviceCount: 20, siteCount: 8, status: 'active' },
          { id: 'sz', name: '深圳市', level: 3, parentName: '广东省', deviceCount: 23, siteCount: 10, status: 'active' },
        ],
      },
    ],
  },
];

function buildAntTreeData(nodes: DomainNode[]): DataNode[] {
  return nodes.map((n) => ({
    title: (
      <span>
        {n.name}
        <Tag color="blue" style={{ marginLeft: 4, fontSize: 10 }}>L{n.level}</Tag>
        <span style={{ fontSize: 10, color: '#8c8c8c', marginLeft: 4 }}>{n.deviceCount}设备</span>
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

function flattenDomains(nodes: DomainNode[]): DomainNode[] {
  const result: DomainNode[] = [];
  for (const n of nodes) {
    result.push(n);
    if (n.children) result.push(...flattenDomains(n.children));
  }
  return result;
}

interface DeviceRow extends Record<string, unknown> {
  id: string;
  sn: string;
  name: string;
  type: string;
  status: string;
}

const MOCK_DEVICES: DeviceRow[] = [
  { id: '1', sn: 'ENB00001', name: '北京朝阳基站01', type: 'eNB', status: 'online' },
  { id: '2', sn: 'ENB00002', name: '北京朝阳基站02', type: 'eNB', status: 'online' },
  { id: '3', sn: 'GNB00001', name: '北京5G基站01', type: 'gNB', status: 'online' },
];

export default function DomainManagement() {
  const t = useT();
  const [selectedDomainId, setSelectedDomainId] = useState<string>('');
  const [modalVisible, setModalVisible] = useState(false);
  const [editingDomain, setEditingDomain] = useState<DomainNode | null>(null);
  const [viewMode, setViewMode] = useState<'form' | 'devices'>('form');
  const [form] = Form.useForm();
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);

  const { data: domainTreeData } = useDomainTree();
  void (null as unknown as Domain);
  void domainTreeData;

  const selectedDomain = findDomain(MOCK_DOMAIN_TREE, selectedDomainId);
  const allDomains = flattenDomains(MOCK_DOMAIN_TREE);

  const deviceColumns: DataTableColumn<DeviceRow>[] = useMemo(() => [
    { key: 'sn', title: 'SN', dataIndex: 'sn', width: 130, mono: true, copyable: true },
    { key: 'name', title: t('device.name'), dataIndex: 'name', width: 200 },
    { key: 'type', title: t('table.type'), dataIndex: 'type', width: 90, render: (val) => <Tag color="blue">{val as string}</Tag> },
    { key: 'status', title: t('table.status'), dataIndex: 'status', width: 90, render: (val) => <Tag color={val === 'online' ? 'success' : 'default'}>{val === 'online' ? t('status.online') : t('status.offline')}</Tag> },
  ], [t]);

  const openCreate = () => {
    setEditingDomain(null);
    form.resetFields();
    setModalVisible(true);
  };

  const openEdit = (domain: DomainNode) => {
    setEditingDomain(domain);
    form.setFieldsValue({ name: domain.name, level: domain.level });
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
      <div style={{ padding: '12px 12px 8px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography.Text strong style={{ fontSize: 13 }}>{t('nav.topology.domain')}</Typography.Text>
        <Button size="small" type="primary" icon={<PlusOutlined />} onClick={openCreate}>{t('common.add')}</Button>
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: '0 4px' }}>
        <Tree
          treeData={buildAntTreeData(MOCK_DOMAIN_TREE)}
          onSelect={(keys) => setSelectedDomainId(keys[0] as string ?? '')}
          defaultExpandAll
          showLine
        />
      </div>
    </div>
  );

  return (
    <TreeListPageLayout tree={treePanel}>
      {selectedDomain ? (
        <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
          <div style={{ padding: '12px 16px', borderBottom: '1px solid #f0f0f0', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Typography.Text strong style={{ fontSize: 14 }}>{selectedDomain.name}</Typography.Text>
            <Space>
              <Button.Group size="small">
                <Button type={viewMode === 'form' ? 'primary' : 'default'} onClick={() => setViewMode('form')}>{t('common.detail')}</Button>
                <Button type={viewMode === 'devices' ? 'primary' : 'default'} onClick={() => setViewMode('devices')}>{t('common.view')}</Button>
              </Button.Group>
              <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(selectedDomain)}>{t('common.edit')}</Button>
              <Button size="small" danger icon={<DeleteOutlined />} onClick={() => void message.warning(t('common.confirmDelete'))}>{t('common.delete')}</Button>
            </Space>
          </div>
          <div style={{ flex: 1, overflow: 'auto', padding: 16 }}>
            {viewMode === 'form' ? (
              <Descriptions bordered column={2} size="small">
                <Descriptions.Item label={t('table.name')}>{selectedDomain.name}</Descriptions.Item>
                <Descriptions.Item label={t('table.type')}>L{selectedDomain.level}</Descriptions.Item>
                <Descriptions.Item label={t('table.region')}>{selectedDomain.parentName}</Descriptions.Item>
                <Descriptions.Item label={t('table.status')}>
                  <Tag color={selectedDomain.status === 'active' ? 'success' : 'default'}>
                    {selectedDomain.status === 'active' ? t('status.enabled') : t('status.disabled')}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label={t('table.total')}>{selectedDomain.deviceCount}</Descriptions.Item>
                <Descriptions.Item label={t('table.site')}>{selectedDomain.siteCount}</Descriptions.Item>
                <Descriptions.Item label={t('table.total')} span={2}>
                  {selectedDomain.children?.length ?? 0}
                </Descriptions.Item>
              </Descriptions>
            ) : (
              <DataTable<DeviceRow>
                tableId="domain-devices"
                columns={deviceColumns}
                dataSource={MOCK_DEVICES}
                rowKey="id"
                total={MOCK_DEVICES.length}
                currentPage={page}
                pageSize={pageSize}
                onPageChange={(p) => setPage(p)}
                showPagination={false}
              />
            )}
          </div>
        </div>
      ) : (
        <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', flexDirection: 'column', gap: 8 }}>
          <Typography.Text type="secondary">{t('common.pleaseSelect')}</Typography.Text>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>{t('table.total')}: {allDomains.length}</Typography.Text>
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
            <InputNumber min={1} max={5} style={{ width: '100%' }} placeholder="1-5" />
          </Form.Item>
          <Form.Item label={t('table.region')} name="parentId">
            <InputNumber style={{ width: '100%' }} placeholder={t('common.placeholder')} />
          </Form.Item>
        </Form>
      </Modal>
    </TreeListPageLayout>
  );
}
