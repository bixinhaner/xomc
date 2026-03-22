import { useState, useMemo } from 'react';
import { Button, Form, Input, InputNumber, Modal, Popconfirm, Space, Tag, message } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, EnvironmentOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useSites } from '@/hooks/api/useTopology';
import type { Site } from '@/types/topology';
import { useT } from '@/hooks/useT';

interface SiteRow extends Record<string, unknown> {
  id: string;
  name: string;
  domainName: string;
  domainId: string;
  address: string;
  longitude: number;
  latitude: number;
  deviceCount: number;
  status: 'active' | 'inactive' | 'maintenance';
}

const STATUS_MAP: Record<string, { color: string; key: string }> = {
  active: { color: 'success', key: 'topology.site.active' },
  inactive: { color: 'default', key: 'topology.site.inactive' },
  maintenance: { color: 'warning', key: 'topology.site.maintenance' },
};

const DOMAIN_OPTIONS = [
  { label: '北京朝阳区', value: 'bj-cy' },
  { label: '北京海淀区', value: 'bj-hd' },
  { label: '上海浦东新区', value: 'sh-pd' },
  { label: '上海静安区', value: 'sh-ja' },
  { label: '广州天河区', value: 'gz-th' },
  { label: '深圳南山区', value: 'sz-ns' },
];

const DOMAIN_NAME_MAP: Record<string, string> = {
  'bj-cy': '北京朝阳区',
  'bj-hd': '北京海淀区',
  'sh-pd': '上海浦东新区',
  'sh-ja': '上海静安区',
  'gz-th': '广州天河区',
  'sz-ns': '深圳南山区',
};

const mockData: SiteRow[] = [
  { id: '1', name: '北京朝阳站点01', domainId: 'bj-cy', domainName: '北京朝阳区', address: '北京市朝阳区建国路88号', longitude: 116.46, latitude: 39.92, deviceCount: 3, status: 'active' },
  { id: '2', name: '北京海淀站点01', domainId: 'bj-hd', domainName: '北京海淀区', address: '北京市海淀区中关村大街1号', longitude: 116.31, latitude: 39.98, deviceCount: 2, status: 'active' },
  { id: '3', name: '北京朝阳站点02', domainId: 'bj-cy', domainName: '北京朝阳区', address: '北京市朝阳区望京街道', longitude: 116.49, latitude: 40.00, deviceCount: 1, status: 'maintenance' },
  { id: '4', name: '上海浦东站点01', domainId: 'sh-pd', domainName: '上海浦东新区', address: '上海市浦东新区张江高科技园区', longitude: 121.60, latitude: 31.21, deviceCount: 4, status: 'active' },
  { id: '5', name: '上海静安站点01', domainId: 'sh-ja', domainName: '上海静安区', address: '上海市静安区南京西路1882号', longitude: 121.45, latitude: 31.23, deviceCount: 2, status: 'active' },
  { id: '6', name: '广州天河站点01', domainId: 'gz-th', domainName: '广州天河区', address: '广州市天河区珠江新城', longitude: 113.33, latitude: 23.12, deviceCount: 3, status: 'active' },
  { id: '7', name: '深圳南山站点01', domainId: 'sz-ns', domainName: '深圳南山区', address: '深圳市南山区科技园', longitude: 113.93, latitude: 22.53, deviceCount: 2, status: 'active' },
];

export default function SiteManagement() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [modalVisible, setModalVisible] = useState(false);
  const [editingSite, setEditingSite] = useState<SiteRow | null>(null);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [form] = Form.useForm();

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'name', label: t('table.name'), type: 'input' },
    { name: 'domainId', label: t('table.region'), type: 'select', options: DOMAIN_OPTIONS },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.online'), value: 'active' },
        { label: t('status.disabled'), value: 'inactive' },
        { label: t('status.pending'), value: 'maintenance' },
      ],
    },
  ], [t]);

  const { data, isLoading, refetch } = useSites();
  void (null as unknown as Site);

  const tableSource = (data ?? mockData) as unknown as SiteRow[];

  const filteredSource = tableSource.filter((row) => {
    if (filters.name && !row.name.includes(filters.name as string)) return false;
    if (filters.domainId && row.domainId !== filters.domainId) return false;
    if (filters.status && row.status !== filters.status) return false;
    return true;
  });

  const openEdit = (record: SiteRow) => {
    setEditingSite(record);
    form.setFieldsValue({
      ...record,
      domainId: record.domainId,
    });
    setModalVisible(true);
  };

  const handleSave = () => {
    form.validateFields().then((vals: Record<string, unknown>) => {
      const domainName = DOMAIN_NAME_MAP[vals.domainId as string] ?? vals.domainId;
      void domainName;
      void message.success(t('common.save'));
      setModalVisible(false);
      form.resetFields();
    }).catch(() => undefined);
  };

  const columns: DataTableColumn<SiteRow>[] = useMemo(() => [
    { key: 'name', title: t('table.name'), dataIndex: 'name', width: 200, ellipsis: true },
    { key: 'domainName', title: t('table.region'), dataIndex: 'domainName', width: 140 },
    { key: 'address', title: t('table.site'), dataIndex: 'address', width: 260, ellipsis: true },
    {
      key: 'coordinates',
      title: t('table.site'),
      dataIndex: 'longitude',
      width: 160,
      render: (_, record) => (
        <span style={{ fontFamily: 'monospace', fontSize: 12 }}>
          {record.longitude.toFixed(4)}, {record.latitude.toFixed(4)}
        </span>
      ),
    },
    { key: 'deviceCount', title: t('table.total'), dataIndex: 'deviceCount', width: 90 },
    {
      key: 'status',
      title: t('table.status'),
      dataIndex: 'status',
      width: 90,
      render: (val) => {
        const cfg = STATUS_MAP[val as string] ?? STATUS_MAP.inactive;
        return <Tag color={cfg.color}>{t(cfg.key)}</Tag>;
      },
    },
    {
      key: 'action',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 150,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<EnvironmentOutlined />}
            onClick={() => void message.info(`在地图上定位: ${record.name as string}`)}
          >
            {t('common.view')}
          </Button>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openEdit(record)}>{t('common.edit')}</Button>
          <Popconfirm title={t('common.confirmDelete')} onConfirm={() => void message.success(t('common.deleteSuccess'))}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>{t('common.delete')}</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ], [t]);

  return (
    <ListPageLayout
      title={t('nav.topology.site')}
      extra={
        <Space>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => { setEditingSite(null); form.resetFields(); setModalVisible(true); }}
          >
            {t('common.add')}
          </Button>
          {selectedKeys.length > 0 && (
            <Button>{t('common.more')} ({selectedKeys.length})</Button>
          )}
        </Space>
      }
    >
      <FilterBar
        filterId="site-management"
        fields={filterFields}
        onSearch={(vals) => setFilters(vals)}
        onReset={() => setFilters({})}
      />
      <DataTable<SiteRow>
        tableId="site-management"
        columns={columns}
        dataSource={filteredSource}
        loading={isLoading}
        rowKey="id"
        selectable
        selectedRowKeys={selectedKeys}
        onSelectionChange={(keys) => setSelectedKeys(keys)}
        total={filteredSource.length}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        onExport={() => void message.info(t('common.exportInProgress'))}
        scroll={{ x: 1300 }}
      />

      <Modal
        title={editingSite ? t('common.edit') : t('common.add')}
        open={modalVisible}
        onOk={handleSave}
        onCancel={() => { setModalVisible(false); form.resetFields(); }}
        okText={t('common.save')}
        width={560}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item label={t('table.name')} name="name" rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('table.region')} name="domainId" rules={[{ required: true }]}>
            <InputNumber style={{ width: '100%' }} placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('table.site')} name="address" rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('table.site')} name="longitude" rules={[{ required: true }]}>
            <InputNumber min={73} max={135} precision={6} style={{ width: '100%' }} placeholder="73-135" />
          </Form.Item>
          <Form.Item label={t('table.site')} name="latitude" rules={[{ required: true }]}>
            <InputNumber min={18} max={53} precision={6} style={{ width: '100%' }} placeholder="18-53" />
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
