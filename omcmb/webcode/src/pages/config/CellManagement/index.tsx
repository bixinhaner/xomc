import { useState, useMemo } from 'react';
import { Button, Dropdown, Form, Input, InputNumber, Modal, Select, Space, Tag, message } from 'antd';
import { PlusOutlined, DeleteOutlined, PoweroffOutlined, MoreOutlined } from '@ant-design/icons';
import type { MenuProps } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useT } from '@/hooks/useT';

interface CellRow extends Record<string, unknown> {
  id: string;
  cellId: string;
  cellName: string;
  stationName: string;
  stationSn: string;
  pci: number;
  tac: number;
  earfcn: number;
  bandwidth: string;
  status: 'active' | 'inactive' | 'maintenance' | 'fault';
}

const BW_OPTIONS = [
  { label: '1.4 MHz', value: '1.4MHz' },
  { label: '3 MHz', value: '3MHz' },
  { label: '5 MHz', value: '5MHz' },
  { label: '10 MHz', value: '10MHz' },
  { label: '15 MHz', value: '15MHz' },
  { label: '20 MHz', value: '20MHz' },
  { label: '100 MHz (NR)', value: '100MHz' },
];

const mockData: CellRow[] = [
  { id: '1', cellId: 'CELL-00001', cellName: '北京-朝阳-001-A', stationName: '北京朝阳基站01', stationSn: 'ENB00001', pci: 10, tac: 1001, earfcn: 1575, bandwidth: '20MHz', status: 'active' },
  { id: '2', cellId: 'CELL-00002', cellName: '北京-朝阳-001-B', stationName: '北京朝阳基站01', stationSn: 'ENB00001', pci: 11, tac: 1001, earfcn: 1575, bandwidth: '20MHz', status: 'active' },
  { id: '3', cellId: 'CELL-00003', cellName: '北京-朝阳-001-C', stationName: '北京朝阳基站01', stationSn: 'ENB00001', pci: 12, tac: 1001, earfcn: 1575, bandwidth: '20MHz', status: 'maintenance' },
  { id: '4', cellId: 'CELL-00004', cellName: '北京-海淀-001-A', stationName: '北京海淀基站01', stationSn: 'ENB00002', pci: 20, tac: 1002, earfcn: 1600, bandwidth: '20MHz', status: 'active' },
  { id: '5', cellId: 'CELL-00005', cellName: '北京-5G-001-A', stationName: '北京5G基站01', stationSn: 'GNB00001', pci: 100, tac: 2001, earfcn: 630240, bandwidth: '100MHz', status: 'active' },
  { id: '6', cellId: 'CELL-00006', cellName: '上海-浦东-001-A', stationName: '上海浦东基站01', stationSn: 'ENB00003', pci: 50, tac: 3001, earfcn: 3200, bandwidth: '20MHz', status: 'fault' },
];

export default function CellManagement() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [modalVisible, setModalVisible] = useState(false);
  const [editingCell, setEditingCell] = useState<CellRow | null>(null);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [form] = Form.useForm();

  const STATUS_MAP: Record<string, { color: string; text: string }> = useMemo(() => ({
    active: { color: 'success', text: t('status.enabled') },
    inactive: { color: 'default', text: t('status.disabled') },
    maintenance: { color: 'warning', text: t('status.pending') },
    fault: { color: 'error', text: t('status.failed') },
  }), [t]);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'cellName', label: t('table.name'), type: 'input' },
    { name: 'stationSn', label: t('device.sn'), type: 'input' },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.enabled'), value: 'active' },
        { label: t('status.disabled'), value: 'inactive' },
        { label: t('status.pending'), value: 'maintenance' },
        { label: t('status.failed'), value: 'fault' },
      ],
    },
  ], [t]);

  const filteredData = mockData.filter((row) => {
    if (filters.cellName && !row.cellName.includes(filters.cellName as string)) return false;
    if (filters.stationSn && !row.stationSn.includes(filters.stationSn as string)) return false;
    if (filters.status && row.status !== filters.status) return false;
    return true;
  });

  const openEdit = (record: CellRow) => {
    setEditingCell(record);
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

  const columns: DataTableColumn<CellRow>[] = useMemo(() => [
    { key: 'cellId', title: t('nav.config.cell'), dataIndex: 'cellId', width: 130, mono: true, copyable: true },
    { key: 'cellName', title: t('table.name'), dataIndex: 'cellName', width: 200, ellipsis: true },
    { key: 'stationName', title: t('device.name'), dataIndex: 'stationName', width: 180, ellipsis: true },
    { key: 'pci', title: 'PCI', dataIndex: 'pci', width: 70 },
    { key: 'tac', title: 'TAC', dataIndex: 'tac', width: 80 },
    { key: 'earfcn', title: 'EARFCN', dataIndex: 'earfcn', width: 90 },
    { key: 'bandwidth', title: t('perf.unit'), dataIndex: 'bandwidth', width: 90 },
    {
      key: 'status',
      title: t('table.status'),
      dataIndex: 'status',
      width: 90,
      render: (val) => {
        const cfg = STATUS_MAP[val as string] ?? STATUS_MAP.inactive;
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    {
      key: 'action',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 100,
      fixed: 'right',
      render: (_, record) => {
        const menuItems: MenuProps['items'] = [
          { key: 'toggle', label: record.status === 'active' ? t('common.disable') : t('common.enable'), icon: <PoweroffOutlined />, onClick: () => void message.info(`${record.status === 'active' ? t('common.disable') : t('common.enable')}: ${record.cellName as string}`) },
          { key: 'delete', label: t('common.delete'), icon: <DeleteOutlined />, danger: true, onClick: () => void message.success(t('common.deleteSuccess')) },
        ];
        return (
          <Space size={4}>
            <Button type="link" size="small" onClick={() => openEdit(record)}>{t('common.edit')}</Button>
            <Dropdown menu={{ items: menuItems }} trigger={['click']}>
              <Button type="link" size="small" icon={<MoreOutlined />} />
            </Dropdown>
          </Space>
        );
      },
    },
  ], [t, STATUS_MAP]);

  return (
    <ListPageLayout
      title={t('nav.config.cell')}
      extra={
        <Space>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => { setEditingCell(null); form.resetFields(); setModalVisible(true); }}
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
        filterId="cell-management"
        fields={filterFields}
        onSearch={(vals) => setFilters(vals)}
        onReset={() => setFilters({})}
      />
      <DataTable<CellRow>
        tableId="cell-management"
        columns={columns}
        dataSource={filteredData}
        loading={false}
        rowKey="id"
        selectable
        selectedRowKeys={selectedKeys}
        onSelectionChange={(keys) => setSelectedKeys(keys)}
        total={filteredData.length}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onExport={() => void message.info(t('common.exportInProgress'))}
        scroll={{ x: 1200 }}
      />

      <Modal
        title={editingCell ? t('common.edit') : t('common.add')}
        open={modalVisible}
        onOk={handleSave}
        onCancel={() => { setModalVisible(false); form.resetFields(); }}
        okText={t('common.save')}
        width={600}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item label={t('table.name')} name="cellName" rules={[{ required: true, message: t('common.placeholder') }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('device.sn')} name="stationSn" rules={[{ required: true, message: t('common.placeholder') }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label="PCI" name="pci" rules={[{ required: true, message: t('common.placeholder') }]}>
            <InputNumber min={0} max={503} style={{ width: '100%' }} placeholder="0-503" />
          </Form.Item>
          <Form.Item label="TAC" name="tac" rules={[{ required: true, message: t('common.placeholder') }]}>
            <InputNumber min={0} max={65535} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="EARFCN" name="earfcn" rules={[{ required: true, message: t('common.placeholder') }]}>
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label={t('perf.unit')} name="bandwidth" rules={[{ required: true, message: t('common.pleaseSelect') }]}>
            <Select options={BW_OPTIONS} placeholder={t('common.pleaseSelect')} />
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
