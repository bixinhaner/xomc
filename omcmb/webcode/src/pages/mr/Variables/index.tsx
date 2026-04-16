import { useState, useMemo } from 'react';
import { Button, Tag, Space, Modal, Form, Input, Select, message } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

interface MRVariable {
  id: string;
  varName: string;
  varType: 'integer' | 'float' | 'enum' | 'boolean' | 'string';
  valueRange: string;
  defaultValue: string;
  description: string;
  unit?: string;
  category: string;
}

const mockVariables: MRVariable[] = [
  { id: 'var-001', varName: 'MR_PERIOD', varType: 'integer', valueRange: '5-3600', defaultValue: '30', description: 'MR采集周期（秒）', unit: '秒', category: '采集配置' },
  { id: 'var-002', varName: 'MR_SAMPLE_NUM', varType: 'integer', valueRange: '1-1000', defaultValue: '200', description: '单次MR采集样本数', unit: '个', category: '采集配置' },
  { id: 'var-003', varName: 'MR_RSRP_THRESHOLD', varType: 'float', valueRange: '-140 ~ -44', defaultValue: '-110', description: 'RSRP过滤阈值，低于此值的测量不上报', unit: 'dBm', category: '过滤配置' },
  { id: 'var-004', varName: 'MR_RSRQ_THRESHOLD', varType: 'float', valueRange: '-19.5 ~ -3', defaultValue: '-15', description: 'RSRQ过滤阈值', unit: 'dB', category: '过滤配置' },
  { id: 'var-005', varName: 'MR_REPORT_MODE', varType: 'enum', valueRange: 'A1/A2/A3/A4/A5/B1/B2', defaultValue: 'A3', description: 'MR上报触发模式', category: '上报配置' },
  { id: 'var-006', varName: 'MR_INTRA_MEAS_ENABLED', varType: 'boolean', valueRange: 'true/false', defaultValue: 'true', description: '是否启用同频测量', category: '测量配置' },
  { id: 'var-007', varName: 'MR_INTER_MEAS_ENABLED', varType: 'boolean', valueRange: 'true/false', defaultValue: 'true', description: '是否启用异频测量', category: '测量配置' },
  { id: 'var-008', varName: 'MR_MAX_NB_CELLS', varType: 'integer', valueRange: '1-8', defaultValue: '6', description: '最多上报邻区个数', unit: '个', category: '采集配置' },
  { id: 'var-009', varName: 'MR_COMPRESS_FORMAT', varType: 'enum', valueRange: 'NONE/GZIP/ZIP', defaultValue: 'GZIP', description: 'MR文件压缩格式', category: '传输配置' },
  { id: 'var-010', varName: 'MR_FILE_NAMING', varType: 'string', valueRange: '1-255字符', defaultValue: '{SN}_{TYPE}_{DATE}.xml', description: 'MR文件命名模板', category: '传输配置' },
];

const varTypeColorMap: Record<MRVariable['varType'], string> = {
  integer: 'blue',
  float: 'cyan',
  enum: 'purple',
  boolean: 'green',
  string: 'default',
};

export default function Variables() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [variables, setVariables] = useState<MRVariable[]>(mockVariables);
  const [editVisible, setEditVisible] = useState(false);
  const [editVar, setEditVar] = useState<MRVariable | null>(null);
  const [form] = Form.useForm();

  const varTypeLabelMap: Record<MRVariable['varType'], string> = useMemo(() => ({
    integer: t('mr.varTypeInteger'),
    float: t('mr.varTypeFloat'),
    enum: t('mr.varTypeEnum'),
    boolean: t('mr.varTypeBoolean'),
    string: t('mr.varTypeString'),
  }), [t]);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('mr.varName'), type: 'input', placeholder: t('mr.varNamePlaceholder') },
    {
      name: 'varType',
      label: t('table.type'),
      type: 'select',
      options: [
        { label: t('mr.varTypeInteger'), value: 'integer' },
        { label: t('mr.varTypeFloat'), value: 'float' },
        { label: t('mr.varTypeEnum'), value: 'enum' },
        { label: t('mr.varTypeBoolean'), value: 'boolean' },
        { label: t('mr.varTypeString'), value: 'string' },
      ],
    },
    { name: 'category', label: t('mr.category'), type: 'input', placeholder: t('mr.categoryPlaceholder') },
  ], [t]);

  const filtered = variables.filter((v) => {
    if (filters.keyword) {
      const kw = String(filters.keyword).toLowerCase();
      if (!v.varName.toLowerCase().includes(kw) && !v.description.toLowerCase().includes(kw)) return false;
    }
    if (filters.varType && v.varType !== filters.varType) return false;
    if (filters.category && !v.category.includes(String(filters.category))) return false;
    return true;
  });

  const startIndex = (page - 1) * pageSize;
  const paginated = filtered.slice(startIndex, startIndex + pageSize);

  const handleSave = () => {
    form.validateFields().then((vals) => {
      if (editVar) {
        setVariables((prev) => prev.map((v) => v.id === editVar.id ? { ...v, ...vals } : v));
        void message.success(t('mr.varUpdateSuccess'));
      } else {
        const newVar: MRVariable = { id: `var-${Date.now()}`, ...vals };
        setVariables((prev) => [newVar, ...prev]);
        void message.success(t('mr.varCreateSuccess'));
      }
      setEditVisible(false);
      form.resetFields();
      setEditVar(null);
    });
  };

  const columns: DataTableColumn<MRVariable & Record<string, unknown>>[] = useMemo(() => [
    { key: 'varName', title: t('mr.varName'), dataIndex: 'varName', width: 180, render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span> },
    {
      key: 'varType', title: t('table.type'), dataIndex: 'varType', width: 90,
      render: (val) => {
        const tp = val as MRVariable['varType'];
        return <Tag color={varTypeColorMap[tp] ?? 'default'}>{varTypeLabelMap[tp] ?? String(val)}</Tag>;
      },
    },
    { key: 'valueRange', title: t('mr.valueRange'), dataIndex: 'valueRange', width: 150 },
    { key: 'defaultValue', title: t('mr.defaultValue'), dataIndex: 'defaultValue', width: 120 },
    { key: 'unit', title: t('mr.unit'), dataIndex: 'unit', width: 60, render: (val) => val ? String(val) : '—' },
    { key: 'category', title: t('mr.category'), dataIndex: 'category', width: 100 },
    { key: 'description', title: t('table.description'), dataIndex: 'description', ellipsis: true },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 120, fixed: 'right',
      render: (_, record) => {
        const v = record as MRVariable;
        return (
          <Space size={4}>
            <Button type="link" size="small" icon={<EditOutlined />}
              onClick={() => { setEditVar(v); form.setFieldsValue(v); setEditVisible(true); }}>
              {t('common.edit')}
            </Button>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}
              onClick={() => { setVariables((prev) => prev.filter((item) => item.id !== v.id)); void message.success(t('common.deleteSuccess')); }}>
              {t('common.delete')}
            </Button>
          </Space>
        );
      },
    },
  ], [t, varTypeLabelMap, form]);

  return (
    <ListPageLayout
      title={t('nav.mr.variables')}
      subtitle={t('mr.variablesSubtitle')}
      extra={
        <Button type="primary" icon={<PlusOutlined />}
          onClick={() => { setEditVar(null); form.resetFields(); setEditVisible(true); }}>
          {t('mr.addVariable')}
        </Button>
      }
    >
      <FilterBar
        filterId="mr-variables-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable
        tableId="mr-variables-list"
        columns={columns}
        dataSource={paginated as (MRVariable & Record<string, unknown>)[]}
        loading={false}
        rowKey="id"
        total={filtered.length}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => setVariables([...mockVariables])}
        scroll={{ x: 1000 }}
      />

      <Modal
        title={editVar ? t('mr.editVariable') : t('mr.addVariable')}
        open={editVisible}
        onOk={handleSave}
        onCancel={() => { setEditVisible(false); form.resetFields(); setEditVar(null); }}
        width={520}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="varName" label={t('mr.varName')} rules={[{ required: true, message: t('mr.varNameRequired') }, { pattern: /^[A-Z_][A-Z0-9_]*$/, message: t('mr.varNamePattern') }]}>
            <Input placeholder={t('mr.varNameExample')} style={{ fontFamily: 'monospace' }} />
          </Form.Item>
          <Form.Item name="varType" label={t('mr.varType')} rules={[{ required: true }]}>
            <Select options={Object.entries(varTypeLabelMap).map(([val, label]) => ({ label, value: val }))} />
          </Form.Item>
          <Form.Item name="valueRange" label={t('mr.valueRange')} rules={[{ required: true }]}>
            <Input placeholder={t('mr.valueRangePlaceholder')} />
          </Form.Item>
          <Form.Item name="defaultValue" label={t('mr.defaultValue')} rules={[{ required: true }]}>
            <Input placeholder={t('mr.defaultValuePlaceholder')} />
          </Form.Item>
          <Form.Item name="unit" label={t('mr.unit')}>
            <Input placeholder={t('mr.unitPlaceholder')} />
          </Form.Item>
          <Form.Item name="category" label={t('mr.category')} rules={[{ required: true }]}>
            <Input placeholder={t('mr.categoryExample')} />
          </Form.Item>
          <Form.Item name="description" label={t('table.description')} rules={[{ required: true }]}>
            <Input.TextArea rows={2} placeholder={t('mr.descriptionPlaceholder')} />
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
