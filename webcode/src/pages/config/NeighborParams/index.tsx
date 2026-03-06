import { useState, useMemo } from 'react';
import { Button, Form, Input, Modal, Popconfirm, Select, Space, Tag, message } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useNeighborParams } from '@/hooks/api/useConfig';
import type { NeighborParam } from '@/types/config';
import { useT } from '@/hooks/useT';

interface NeighborRow extends Record<string, unknown> {
  id: string;
  sourceCellId: string;
  sourceCellName: string;
  targetCellId: string;
  targetCellName: string;
  neighborType: 'intra-freq' | 'inter-freq' | 'inter-rat';
  handoverThreshold: string;
  status: 'active' | 'inactive';
  createTime: string;
}

const NEIGHBOR_TYPE_MAP: Record<string, { color: string; text: string }> = {
  'intra-freq': { color: 'blue', text: 'intra-freq' },
  'inter-freq': { color: 'purple', text: 'inter-freq' },
  'inter-rat': { color: 'orange', text: 'inter-rat' },
};

const mockData: NeighborRow[] = [
  { id: '1', sourceCellId: 'CELL-00001', sourceCellName: '北京-朝阳-001-A', targetCellId: 'CELL-00002', targetCellName: '北京-朝阳-001-B', neighborType: 'intra-freq', handoverThreshold: '-110dBm', status: 'active', createTime: '2026-01-15 10:00:00' },
  { id: '2', sourceCellId: 'CELL-00001', sourceCellName: '北京-朝阳-001-A', targetCellId: 'CELL-00004', targetCellName: '北京-海淀-001-A', neighborType: 'intra-freq', handoverThreshold: '-112dBm', status: 'active', createTime: '2026-01-15 10:00:00' },
  { id: '3', sourceCellId: 'CELL-00002', sourceCellName: '北京-朝阳-001-B', targetCellId: 'CELL-00005', targetCellName: '北京-5G-001-A', neighborType: 'inter-rat', handoverThreshold: '-108dBm', status: 'active', createTime: '2026-02-01 09:00:00' },
  { id: '4', sourceCellId: 'CELL-00003', sourceCellName: '北京-朝阳-001-C', targetCellId: 'CELL-00001', targetCellName: '北京-朝阳-001-A', neighborType: 'intra-freq', handoverThreshold: '-110dBm', status: 'inactive', createTime: '2026-02-10 14:00:00' },
];

export default function NeighborParams() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRow, setEditingRow] = useState<NeighborRow | null>(null);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [form] = Form.useForm();

  const { data, isLoading, refetch } = useNeighborParams({ page, pageSize });
  void (null as unknown as NeighborParam);

  const tableSource = (data?.items ?? mockData) as unknown as NeighborRow[];

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'sourceCellName', label: t('nav.config.cell'), type: 'input' },
    { name: 'targetCellName', label: t('nav.config.neighbor'), type: 'input' },
    {
      name: 'neighborType',
      label: t('table.type'),
      type: 'select',
      options: [
        { label: 'intra-freq', value: 'intra-freq' },
        { label: 'inter-freq', value: 'inter-freq' },
        { label: 'inter-rat', value: 'inter-rat' },
      ],
    },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.enabled'), value: 'active' },
        { label: t('status.disabled'), value: 'inactive' },
      ],
    },
  ], [t]);

  const openEdit = (record: NeighborRow) => {
    setEditingRow(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleSave = () => {
    form.validateFields().then(() => {
      void message.success(t('common.save'));
      setModalVisible(false);
      form.resetFields();
    }).catch(() => undefined);
  };

  const columns: DataTableColumn<NeighborRow>[] = useMemo(() => [
    { key: 'sourceCellId', title: t('nav.config.cell') + ' ID', dataIndex: 'sourceCellId', width: 130, mono: true },
    { key: 'sourceCellName', title: t('nav.config.cell'), dataIndex: 'sourceCellName', width: 200, ellipsis: true },
    { key: 'targetCellId', title: t('nav.config.neighbor') + ' ID', dataIndex: 'targetCellId', width: 130, mono: true },
    { key: 'targetCellName', title: t('nav.config.neighbor'), dataIndex: 'targetCellName', width: 200, ellipsis: true },
    {
      key: 'neighborType',
      title: t('table.type'),
      dataIndex: 'neighborType',
      width: 100,
      render: (val) => {
        const cfg = NEIGHBOR_TYPE_MAP[val as string];
        return cfg ? <Tag color={cfg.color}>{cfg.text}</Tag> : <Tag>{val as string}</Tag>;
      },
    },
    { key: 'handoverThreshold', title: t('perf.threshold'), dataIndex: 'handoverThreshold', width: 120 },
    {
      key: 'status',
      title: t('table.status'),
      dataIndex: 'status',
      width: 90,
      render: (val) => <Tag color={val === 'active' ? 'success' : 'default'}>{val === 'active' ? t('status.enabled') : t('status.disabled')}</Tag>,
    },
    { key: 'createTime', title: t('table.createTime'), dataIndex: 'createTime', width: 160 },
    {
      key: 'action',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
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
      title={t('nav.config.neighbor')}
      extra={
        <Space>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => { setEditingRow(null); form.resetFields(); setModalVisible(true); }}
          >
            {t('common.add')}
          </Button>
          {selectedKeys.length > 0 && (
            <Button danger>{t('common.batchDelete')} ({selectedKeys.length})</Button>
          )}
        </Space>
      }
    >
      <FilterBar
        filterId="neighbor-params"
        fields={filterFields}
        onSearch={(vals) => setFilters(vals)}
        onReset={() => setFilters({})}
      />
      <DataTable<NeighborRow>
        tableId="neighbor-params"
        columns={columns}
        dataSource={tableSource}
        loading={isLoading}
        rowKey="id"
        selectable
        selectedRowKeys={selectedKeys}
        onSelectionChange={(keys) => setSelectedKeys(keys)}
        total={data?.total ?? tableSource.length}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => { void refetch(); }}
        onExport={() => { void message.info(t('common.exportInProgress')); }}
        scroll={{ x: 1400 }}
      />

      <Modal
        title={editingRow ? t('common.edit') : t('common.add')}
        open={modalVisible}
        onOk={handleSave}
        onCancel={() => { setModalVisible(false); form.resetFields(); }}
        okText={t('common.save')}
        width={560}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item label={t('nav.config.cell') + ' ID'} name="sourceCellId" rules={[{ required: true, message: t('common.placeholder') }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('nav.config.neighbor') + ' ID'} name="targetCellId" rules={[{ required: true, message: t('common.placeholder') }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('table.type')} name="neighborType" rules={[{ required: true, message: t('common.pleaseSelect') }]}>
            <Select
              options={[
                { label: 'intra-freq', value: 'intra-freq' },
                { label: 'inter-freq', value: 'inter-freq' },
                { label: 'inter-rat', value: 'inter-rat' },
              ]}
              placeholder={t('common.pleaseSelect')}
            />
          </Form.Item>
          <Form.Item label={t('perf.threshold')} name="handoverThreshold">
            <Input placeholder="-110dBm" />
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
