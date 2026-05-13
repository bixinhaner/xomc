import { useState, useMemo } from 'react';
import { Button, Dropdown, Form, Input, InputNumber, Modal, Space, Tag, message } from 'antd';
import type { MenuProps } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, MoreOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useSites, useDomains } from '@core/hooks/api/useTopology';
import type { Site } from '@core/types/topology';
import { useT } from '@/hooks/useT';

// Row shape extends Site with a derived domainName (resolved from useDomains).
interface SiteRow extends Site, Record<string, unknown> {
  domainName: string;
}

const STATUS_MAP: Record<string, { color: string; key: string }> = {
  active: { color: 'success', key: 'topology.site.active' },
  inactive: { color: 'default', key: 'topology.site.inactive' },
  maintenance: { color: 'warning', key: 'topology.site.maintenance' },
};

export default function SiteManagement() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [modalVisible, setModalVisible] = useState(false);
  const [editingSite, setEditingSite] = useState<SiteRow | null>(null);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [form] = Form.useForm();

  const { data: domainsData } = useDomains();
  const { data: sitesData, isLoading, refetch } = useSites();

  // Build domainId → name map from real domain tree (no hardcoded zh-CN mapping).
  const domainOptions = useMemo(
    () =>
      (domainsData ?? []).map((d) => ({ label: d.name, value: d.id })),
    [domainsData]
  );
  const domainNameMap = useMemo(() => {
    const map: Record<string, string> = {};
    for (const d of domainsData ?? []) {
      map[d.id] = d.name;
    }
    return map;
  }, [domainsData]);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'name', label: t('table.name'), type: 'input' },
    { name: 'domainId', label: t('table.region'), type: 'select', options: domainOptions },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('topology.site.active'), value: 'active' },
        { label: t('topology.site.inactive'), value: 'inactive' },
        { label: t('topology.site.maintenance'), value: 'maintenance' },
      ],
    },
  ], [t, domainOptions]);

  const tableSource: SiteRow[] = useMemo(
    () =>
      (sitesData?.items ?? []).map((site: Site) => ({
        ...site,
        domainName: domainNameMap[site.domainId] ?? site.domainId,
      })),
    [sitesData, domainNameMap]
  );

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
    form.validateFields().then(() => {
      // TODO: backend exposes only GET/POST /sites today (no PUT/DELETE).
      // POST create flow + PUT/DELETE wiring are tracked as a follow-up task.
      void message.success(t('common.save'));
      setModalVisible(false);
      form.resetFields();
      void refetch();
    }).catch(() => undefined);
  };

  const columns: DataTableColumn<SiteRow>[] = useMemo(() => [
    { key: 'name', title: t('table.name'), dataIndex: 'name', width: 200, ellipsis: true },
    { key: 'domainName', title: t('table.region'), dataIndex: 'domainName', width: 140 },
    { key: 'address', title: t('topology.site.address'), dataIndex: 'address', width: 260, ellipsis: true },
    {
      key: 'coordinates',
      title: t('topology.site.coordinates'),
      dataIndex: 'longitude',
      width: 160,
      render: (_, record) => (
        <span style={{ fontFamily: 'monospace', fontSize: 12 }}>
          {record.longitude.toFixed(4)}, {record.latitude.toFixed(4)}
        </span>
      ),
    },
    { key: 'deviceCount', title: t('topology.site.deviceCount'), dataIndex: 'deviceCount', width: 90 },
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
      width: 100,
      fixed: 'right',
      render: (_, record) => {
        const items: MenuProps['items'] = [
          {
            key: 'edit',
            label: t('common.edit'),
            icon: <EditOutlined />,
            onClick: () => openEdit(record),
          },
          {
            key: 'delete',
            label: t('common.delete'),
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => void message.success(t('common.deleteSuccess')),
          },
        ];
        return (
          <Space size={4}>
            <Button
              type="link"
              size="small"
              onClick={() => void message.info(`在地图上定位: ${record.name as string}`)}
            >
              {t('common.view')}
            </Button>
            <Dropdown menu={{ items }} trigger={['click']}>
              <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
            </Dropdown>
          </Space>
        );
      },
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
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('topology.site.address')} name="address" rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('topology.site.longitude')} name="longitude" rules={[{ required: true }]}>
            <InputNumber min={73} max={135} precision={6} style={{ width: '100%' }} placeholder="73-135" />
          </Form.Item>
          <Form.Item label={t('topology.site.latitude')} name="latitude" rules={[{ required: true }]}>
            <InputNumber min={18} max={53} precision={6} style={{ width: '100%' }} placeholder="18-53" />
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
