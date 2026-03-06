import { useState, useMemo } from 'react';
import { Button, Tree, Tag, Space, Input, Modal, Form, InputNumber, Select, Switch, message } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

interface DictEntry {
  id: string;
  code: string;
  name: string;
  value: string;
  sortOrder: number;
  status: 'active' | 'inactive';
  remark?: string;
  category: string;
}

const dictionaryCategories: DataNode[] = [
  {
    key: 'device', title: '设备相关',
    children: [
      { key: 'device-type', title: '设备类型' },
      { key: 'device-vendor', title: '设备厂商' },
      { key: 'network-type', title: '网络类型' },
      { key: 'conn-status', title: '连接状态' },
    ],
  },
  {
    key: 'alarm', title: '告警相关',
    children: [
      { key: 'alarm-severity', title: '告警级别' },
      { key: 'alarm-type', title: '告警类型' },
      { key: 'alarm-status', title: '告警状态' },
    ],
  },
  {
    key: 'system', title: '系统相关',
    children: [
      { key: 'user-role', title: '用户角色' },
      { key: 'log-level', title: '日志级别' },
      { key: 'task-status', title: '任务状态' },
    ],
  },
  {
    key: 'file', title: '文件相关',
    children: [
      { key: 'file-type', title: '文件类型' },
      { key: 'file-status', title: '文件状态' },
    ],
  },
];

const mockDictEntries: Record<string, DictEntry[]> = {
  'device-type': [
    { id: 'dt-001', code: 'eNB', name: 'eNB基站', value: 'eNB', sortOrder: 1, status: 'active', category: 'device-type' },
    { id: 'dt-002', code: 'gNB', name: 'gNB基站', value: 'gNB', sortOrder: 2, status: 'active', category: 'device-type' },
    { id: 'dt-003', code: 'RRU', name: '射频单元', value: 'RRU', sortOrder: 3, status: 'active', category: 'device-type' },
    { id: 'dt-004', code: 'AAU', name: '有源天线单元', value: 'AAU', sortOrder: 4, status: 'active', category: 'device-type' },
    { id: 'dt-005', code: 'BBU', name: '基带单元', value: 'BBU', sortOrder: 5, status: 'active', category: 'device-type' },
  ],
  'device-vendor': [
    { id: 'dv-001', code: 'HUAWEI', name: '华为', value: '华为', sortOrder: 1, status: 'active', category: 'device-vendor' },
    { id: 'dv-002', code: 'ZTE', name: '中兴', value: '中兴', sortOrder: 2, status: 'active', category: 'device-vendor' },
    { id: 'dv-003', code: 'ERICSSON', name: '爱立信', value: '爱立信', sortOrder: 3, status: 'active', category: 'device-vendor' },
    { id: 'dv-004', code: 'NOKIA', name: '诺基亚', value: '诺基亚', sortOrder: 4, status: 'active', category: 'device-vendor' },
  ],
  'alarm-severity': [
    { id: 'as-001', code: 'CRITICAL', name: '紧急', value: '1', sortOrder: 1, status: 'active', remark: '最高级别，需立即处理', category: 'alarm-severity' },
    { id: 'as-002', code: 'MAJOR', name: '重要', value: '2', sortOrder: 2, status: 'active', remark: '重要告警，需尽快处理', category: 'alarm-severity' },
    { id: 'as-003', code: 'MINOR', name: '次要', value: '3', sortOrder: 3, status: 'active', remark: '次要告警，可计划处理', category: 'alarm-severity' },
    { id: 'as-004', code: 'WARNING', name: '警告', value: '4', sortOrder: 4, status: 'active', remark: '预警信息', category: 'alarm-severity' },
  ],
  'user-role': [
    { id: 'ur-001', code: 'admin', name: '管理员', value: 'admin', sortOrder: 1, status: 'active', category: 'user-role' },
    { id: 'ur-002', code: 'operator', name: '运维员', value: 'operator', sortOrder: 2, status: 'active', category: 'user-role' },
    { id: 'ur-003', code: 'viewer', name: '只读用户', value: 'viewer', sortOrder: 3, status: 'active', category: 'user-role' },
    { id: 'ur-004', code: 'auditor', name: '审计员', value: 'auditor', sortOrder: 4, status: 'active', category: 'user-role' },
  ],
};

export default function DataDictionary() {
  const t = useT();
  const [selectedCategory, setSelectedCategory] = useState<string>('device-type');
  const [searchText, setSearchText] = useState('');
  const [entries, setEntries] = useState<Record<string, DictEntry[]>>(mockDictEntries);
  const [editVisible, setEditVisible] = useState(false);
  const [editEntry, setEditEntry] = useState<DictEntry | null>(null);
  const [form] = Form.useForm();

  const currentEntries = (entries[selectedCategory] ?? []).filter((e) => {
    if (!searchText) return true;
    return e.code.includes(searchText) || e.name.includes(searchText) || e.value.includes(searchText);
  });

  const handleSave = () => {
    form.validateFields().then((vals) => {
      if (editEntry) {
        setEntries((prev) => ({
          ...prev,
          [selectedCategory]: (prev[selectedCategory] ?? []).map((e) =>
            e.id === editEntry.id ? { ...e, ...vals } : e,
          ),
        }));
        void message.success(t('common.save'));
      } else {
        const newEntry: DictEntry = {
          id: `entry-${Date.now()}`,
          ...vals,
          category: selectedCategory,
        };
        setEntries((prev) => ({
          ...prev,
          [selectedCategory]: [...(prev[selectedCategory] ?? []), newEntry],
        }));
        void message.success(t('common.save'));
      }
      setEditVisible(false);
      form.resetFields();
      setEditEntry(null);
    });
  };

  const columns: DataTableColumn<DictEntry & Record<string, unknown>>[] = useMemo(() => [
    { key: 'code', title: t('alarm.code'), dataIndex: 'code', width: 130, render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span> },
    { key: 'name', title: t('table.name'), dataIndex: 'name', width: 130 },
    { key: 'value', title: t('perf.value'), dataIndex: 'value', width: 120 },
    { key: 'sortOrder', title: t('table.index'), dataIndex: 'sortOrder', width: 70 },
    {
      key: 'status', title: t('table.status'), dataIndex: 'status', width: 90,
      render: (val) => <Tag color={val === 'active' ? 'green' : 'default'}>{val === 'active' ? t('status.enabled') : t('status.disabled')}</Tag>,
    },
    { key: 'remark', title: t('table.description'), dataIndex: 'remark', ellipsis: true, render: (val) => val ? String(val) : '—' },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 120,
      render: (_, record) => {
        const entry = record as DictEntry;
        return (
          <Space size="small">
            <Button type="link" size="small" icon={<EditOutlined />} onClick={() => { setEditEntry(entry); form.setFieldsValue(entry); setEditVisible(true); }}>{t('common.edit')}</Button>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}
              onClick={() => {
                setEntries((prev) => ({ ...prev, [selectedCategory]: (prev[selectedCategory] ?? []).filter((e) => e.id !== entry.id) }));
                void message.success(t('common.deleteSuccess'));
              }}>
              {t('common.delete')}
            </Button>
          </Space>
        );
      },
    },
  ], [t, form, selectedCategory]);

  const treePanel = (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div style={{ padding: '12px 8px', borderBottom: '1px solid #f0f0f0' }}>
        <span style={{ fontWeight: 500 }}>{t('nav.system.dataDict')}</span>
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: 8 }}>
        <Tree
          treeData={dictionaryCategories}
          defaultExpandAll
          selectedKeys={[selectedCategory]}
          onSelect={(keys) => { if (keys.length > 0) setSelectedCategory(String(keys[0])); }}
        />
      </div>
    </div>
  );

  return (
    <TreeListPageLayout tree={treePanel}>
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        <div style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center', background: '#fafafa' }}>
          <span style={{ fontWeight: 500 }}>{t('table.total')} ({currentEntries.length})</span>
          <Space>
            <Input placeholder={t('common.search')} prefix={<SearchOutlined />} value={searchText}
              onChange={(e) => setSearchText(e.target.value)} style={{ width: 200 }} size="small" allowClear />
            <Button type="primary" size="small" icon={<PlusOutlined />}
              onClick={() => { setEditEntry(null); form.resetFields(); setEditVisible(true); }}>
              {t('common.add')}
            </Button>
          </Space>
        </div>
        <div style={{ flex: 1, overflow: 'auto' }}>
          <DataTable
            tableId="data-dictionary-list"
            columns={columns}
            dataSource={currentEntries as (DictEntry & Record<string, unknown>)[]}
            loading={false}
            rowKey="id"
            total={currentEntries.length}
            pageSize={20}
            currentPage={1}
          />
        </div>
      </div>

      <Modal
        title={editEntry ? t('common.edit') : t('common.add')}
        open={editVisible}
        onOk={handleSave}
        onCancel={() => { setEditVisible(false); form.resetFields(); setEditEntry(null); }}
        width={480}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="code" label={t('alarm.code')} rules={[{ required: true }]}>
            <Input placeholder={t('alarm.code')} />
          </Form.Item>
          <Form.Item name="name" label={t('table.name')} rules={[{ required: true }]}>
            <Input placeholder={t('table.name')} />
          </Form.Item>
          <Form.Item name="value" label={t('perf.value')} rules={[{ required: true }]}>
            <Input placeholder={t('perf.value')} />
          </Form.Item>
          <Form.Item name="sortOrder" label={t('table.index')} initialValue={99}>
            <InputNumber min={1} max={9999} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="status" label={t('table.status')} initialValue="active">
            <Select options={[{ label: t('status.enabled'), value: 'active' }, { label: t('status.disabled'), value: 'inactive' }]} />
          </Form.Item>
          <Form.Item name="remark" label={t('table.description')}>
            <Input.TextArea rows={2} placeholder={t('table.description')} />
          </Form.Item>
        </Form>
      </Modal>
    </TreeListPageLayout>
  );
}
