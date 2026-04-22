import { useState, useMemo } from 'react';
import { Button, Dropdown, Tag, Space, Modal, Form, Input, Select, message, Drawer, Steps, Descriptions, Tabs } from 'antd';
import type { MenuProps } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, EyeOutlined, DownloadOutlined, MoreOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { mockOpsTemplates } from '@core/mock/data/opsTools';
import type { OpsTemplate, OpsStep } from '@core/mock/data/opsTools';
import { useT } from '@/hooks/useT';

const stepTypeColorMap: Record<OpsStep['stepType'], string> = {
  mml: 'blue',
  check: 'green',
  wait: 'orange',
  notify: 'purple',
  script: 'cyan',
};

const categoryColorMap: Record<string, string> = {
  '巡检运维': 'blue',
  '故障处置': 'red',
  '性能优化': 'green',
  '软件管理': 'cyan',
  '网络配置': 'purple',
  '维护操作': 'orange',
};

const mockGenericTemplates = [
  { id: 'gen-001', name: 'MML命令批量下发模板', description: '通用批量MML命令下发，支持多设备', version: 'v1.2', fileSize: '12 KB', updateTime: '2024-05-01', downloads: 312 },
  { id: 'gen-002', name: '配置备份脚本模板', description: '设备配置自动备份模板，含差异对比', version: 'v2.0', fileSize: '8 KB', updateTime: '2024-04-15', downloads: 189 },
  { id: 'gen-003', name: '告警批量清除模板', description: '批量清除历史告警的通用模板', version: 'v1.0', fileSize: '5 KB', updateTime: '2024-03-20', downloads: 98 },
  { id: 'gen-004', name: '性能参数采集模板', description: '标准性能计数器采集和导出模板', version: 'v1.3', fileSize: '15 KB', updateTime: '2024-06-01', downloads: 256 },
];

export default function Templates() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [genericFilters, setGenericFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [templates, setTemplates] = useState<OpsTemplate[]>(mockOpsTemplates);
  const [createVisible, setCreateVisible] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedTemplate, setSelectedTemplate] = useState<OpsTemplate | null>(null);
  const [form] = Form.useForm();

  const stepTypeLabelMap: Record<OpsStep['stepType'], string> = useMemo(() => ({
    mml: t('ops.stepTypeMml'),
    check: t('ops.stepTypeCheck'),
    wait: t('ops.stepTypeWait'),
    notify: t('ops.stepTypeNotify'),
    script: t('ops.stepTypeScript'),
  }), [t]);

  const commissionFilterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('ops.templateName'), type: 'input', placeholder: t('ops.templateNamePlaceholder') },
    {
      name: 'category',
      label: t('mr.category'),
      type: 'select',
      options: [
        { label: t('ops.catInspection'), value: '巡检运维' },
        { label: t('ops.catFault'), value: '故障处置' },
        { label: t('ops.catPerformance'), value: '性能优化' },
        { label: t('ops.catSoftware'), value: '软件管理' },
        { label: t('ops.catNetwork'), value: '网络配置' },
        { label: t('ops.catMaintenance'), value: '维护操作' },
      ],
    },
    {
      name: 'targetDeviceType',
      label: t('ops.targetDevice'),
      type: 'select',
      options: [
        { label: 'eNB', value: 'eNB' },
        { label: 'gNB', value: 'gNB' },
        { label: 'CPE', value: 'CPE' },
        { label: 'eGW', value: 'eGW' },
      ],
    },
  ], [t]);

  const genericFilterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('ops.templateName'), type: 'input', placeholder: t('ops.templateNamePlaceholder') },
  ], [t]);

  const filtered = templates.filter((tp) => {
    if (filters.keyword && !tp.templateName.includes(String(filters.keyword))) return false;
    if (filters.category && tp.category !== filters.category) return false;
    if (filters.targetDeviceType && !tp.targetDeviceTypes.includes(String(filters.targetDeviceType))) return false;
    return true;
  });

  const startIndex = (page - 1) * pageSize;
  const paginated = filtered.slice(startIndex, startIndex + pageSize);

  const filteredGeneric = mockGenericTemplates.filter((tp) => {
    if (genericFilters.keyword && !tp.name.includes(String(genericFilters.keyword))) return false;
    return true;
  });

  const handleCreate = () => {
    form.validateFields().then((vals) => {
      const newTemplate: OpsTemplate = {
        id: `opst-${Date.now()}`,
        templateName: vals.templateName as string,
        description: (vals.description as string) ?? '',
        category: vals.category as string,
        targetDeviceTypes: (vals.targetDeviceTypes as string[]) ?? [],
        steps: [],
        estimatedDuration: Number(vals.estimatedDuration ?? 60),
        creator: 'admin',
        createTime: new Date().toISOString(),
        updateTime: new Date().toISOString(),
        useCount: 0,
        tags: [],
      };
      setTemplates((prev) => [newTemplate, ...prev]);
      void message.success(t('ops.templateCreateSuccess'));
      setCreateVisible(false);
      form.resetFields();
    });
  };

  const handleDelete = (id: string) => {
    Modal.confirm({
      title: t('common.confirmDelete'),
      content: t('ops.deleteConfirmContent'),
      okType: 'danger',
      onOk: () => {
        setTemplates((prev) => prev.filter((tp) => tp.id !== id));
        void message.success(t('common.deleteSuccess'));
      },
    });
  };

  const commissionColumns: DataTableColumn<OpsTemplate & Record<string, unknown>>[] = useMemo(() => [
    { key: 'templateName', title: t('ops.templateName'), dataIndex: 'templateName', ellipsis: true, width: 200 },
    {
      key: 'category', title: t('mr.category'), dataIndex: 'category', width: 100,
      render: (val) => <Tag color={categoryColorMap[String(val)] ?? 'default'}>{String(val)}</Tag>,
    },
    {
      key: 'targetDeviceTypes', title: t('ops.applicableDevices'), dataIndex: 'targetDeviceTypes', width: 160,
      render: (val) => (val as string[]).map((tp) => <Tag key={tp}>{tp}</Tag>),
    },
    { key: 'description', title: t('table.description'), dataIndex: 'description', ellipsis: true },
    {
      key: 'steps', title: t('ops.stepCount'), dataIndex: 'steps', width: 80,
      render: (val) => (val as OpsStep[]).length,
    },
    {
      key: 'estimatedDuration', title: t('ops.estimatedDuration'), dataIndex: 'estimatedDuration', width: 100,
      render: (val) => {
        const secs = Number(val);
        if (secs >= 60) return `${Math.floor(secs / 60)} ${t('ops.minutes')}`;
        return `${secs} ${t('ops.seconds')}`;
      },
    },
    { key: 'useCount', title: t('ops.useCount'), dataIndex: 'useCount', width: 90 },
    { key: 'creator', title: t('mr.creator'), dataIndex: 'creator', width: 90 },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 100, fixed: 'right',
      render: (_, record) => {
        const tp = record as OpsTemplate;
        const moreItems: MenuProps['items'] = [
          { key: 'edit', label: t('common.edit'), icon: <EditOutlined /> },
          { type: 'divider' as const },
          { key: 'delete', label: t('common.delete'), icon: <DeleteOutlined />, danger: true, onClick: () => handleDelete(tp.id) },
        ];
        return (
          <Space size={4}>
            <Button type="link" size="small" icon={<EyeOutlined />}
              onClick={() => { setSelectedTemplate(tp); setDetailVisible(true); }}>
              {t('common.detail')}
            </Button>
            <Dropdown
              menu={{ items: moreItems }}
              trigger={['click']}
            >
              <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
            </Dropdown>
          </Space>
        );
      },
    },
  ], [t]);

  const genericColumns: DataTableColumn<(typeof mockGenericTemplates)[0] & Record<string, unknown>>[] = useMemo(() => [
    { key: 'name', title: t('ops.templateName'), dataIndex: 'name', ellipsis: true, width: 240 },
    { key: 'description', title: t('table.description'), dataIndex: 'description', ellipsis: true },
    { key: 'version', title: t('table.version'), dataIndex: 'version', width: 80 },
    { key: 'fileSize', title: t('mr.fileSize'), dataIndex: 'fileSize', width: 90 },
    { key: 'downloads', title: t('ops.downloadCount'), dataIndex: 'downloads', width: 100 },
    { key: 'updateTime', title: t('table.updateTime'), dataIndex: 'updateTime', width: 120 },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 90, fixed: 'right',
      render: () => (
        <Button type="link" size="small" icon={<DownloadOutlined />}
          onClick={() => void message.success(t('mr.downloadTaskCreated'))}>
          {t('common.download')}
        </Button>
      ),
    },
  ], [t]);

  return (
    <ListPageLayout
      title={t('nav.ops.templates')}
      subtitle={t('ops.templatesSubtitle')}
      extra={<Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>{t('ops.newTemplate')}</Button>}
    >
      <Tabs
        items={[
          {
            key: 'commission',
            label: t('ops.commissionTemplates'),
            children: (
              <>
                <FilterBar
                  filterId="ops-templates-commission-filter"
                  fields={commissionFilterFields}
                  onSearch={(vals) => { setFilters(vals); setPage(1); }}
                  onReset={() => { setFilters({}); setPage(1); }}
                />
                <DataTable
                  tableId="ops-templates-commission-list"
                  columns={commissionColumns}
                  dataSource={paginated as (OpsTemplate & Record<string, unknown>)[]}
                  loading={false}
                  rowKey="id"
                  total={filtered.length}
                  pageSize={pageSize}
                  currentPage={page}
                  onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
                  onRefresh={() => setTemplates([...mockOpsTemplates])}
                  scroll={{ x: 1100 }}
                />
              </>
            ),
          },
          {
            key: 'generic',
            label: t('ops.genericTemplates'),
            children: (
              <>
                <FilterBar
                  filterId="ops-templates-generic-filter"
                  fields={genericFilterFields}
                  onSearch={(vals) => setGenericFilters(vals)}
                  onReset={() => setGenericFilters({})}
                />
                <DataTable
                  tableId="ops-templates-generic-list"
                  columns={genericColumns}
                  dataSource={filteredGeneric as ((typeof mockGenericTemplates)[0] & Record<string, unknown>)[]}
                  loading={false}
                  rowKey="id"
                  total={filteredGeneric.length}
                  pageSize={10}
                  currentPage={1}
                  scroll={{ x: 900 }}
                />
              </>
            ),
          },
        ]}
      />

      <Modal
        title={t('ops.newOpsTemplate')}
        open={createVisible}
        onOk={handleCreate}
        onCancel={() => { setCreateVisible(false); form.resetFields(); }}
        width={560}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="templateName" label={t('ops.templateName')} rules={[{ required: true }]}>
            <Input placeholder={t('ops.templateNamePlaceholder')} />
          </Form.Item>
          <Form.Item name="category" label={t('mr.category')} rules={[{ required: true }]}>
            <Select options={[
              { label: t('ops.catInspection'), value: '巡检运维' },
              { label: t('ops.catFault'), value: '故障处置' },
              { label: t('ops.catPerformance'), value: '性能优化' },
              { label: t('ops.catSoftware'), value: '软件管理' },
              { label: t('ops.catNetwork'), value: '网络配置' },
              { label: t('ops.catMaintenance'), value: '维护操作' },
            ]} />
          </Form.Item>
          <Form.Item name="targetDeviceTypes" label={t('ops.applicableDeviceTypes')} rules={[{ required: true }]}>
            <Select
              mode="multiple"
              options={[
                { label: 'eNB', value: 'eNB' },
                { label: 'gNB', value: 'gNB' },
                { label: 'CPE', value: 'CPE' },
                { label: 'eGW', value: 'eGW' },
              ]}
            />
          </Form.Item>
          <Form.Item name="description" label={t('table.description')}>
            <Input.TextArea rows={3} placeholder={t('ops.descriptionPlaceholder')} />
          </Form.Item>
          <Form.Item name="estimatedDuration" label={t('ops.estimatedDurationSeconds')}>
            <Input type="number" placeholder={t('ops.durationExample')} />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={selectedTemplate ? `${t('ops.templateDetail')} — ${selectedTemplate.templateName}` : t('ops.templateDetail')}
        open={detailVisible}
        onClose={() => setDetailVisible(false)}
        width={680}
      >
        {selectedTemplate && (
          <>
            <Descriptions bordered column={2} size="small" style={{ marginBottom: 24 }}>
              <Descriptions.Item label={t('ops.templateName')} span={2}>{selectedTemplate.templateName}</Descriptions.Item>
              <Descriptions.Item label={t('mr.category')}>
                <Tag color={categoryColorMap[selectedTemplate.category] ?? 'default'}>{selectedTemplate.category}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label={t('ops.useCount')}>{selectedTemplate.useCount}</Descriptions.Item>
              <Descriptions.Item label={t('ops.applicableDevices')} span={2}>
                {selectedTemplate.targetDeviceTypes.map((tp) => <Tag key={tp}>{tp}</Tag>)}
              </Descriptions.Item>
              <Descriptions.Item label={t('ops.estimatedDuration')}>
                {selectedTemplate.estimatedDuration >= 60
                  ? `${Math.floor(selectedTemplate.estimatedDuration / 60)} ${t('ops.minutes')}`
                  : `${selectedTemplate.estimatedDuration} ${t('ops.seconds')}`}
              </Descriptions.Item>
              <Descriptions.Item label={t('mr.creator')}>{selectedTemplate.creator}</Descriptions.Item>
              <Descriptions.Item label={t('ops.tags')} span={2}>
                {selectedTemplate.tags.map((tag) => <Tag key={tag} color="blue">{tag}</Tag>)}
              </Descriptions.Item>
              <Descriptions.Item label={t('table.description')} span={2}>{selectedTemplate.description}</Descriptions.Item>
            </Descriptions>

            <div style={{ marginBottom: 8, fontWeight: 600, fontSize: 14 }}>
              {t('ops.executionSteps', { count: String(selectedTemplate.steps.length) })}
            </div>
            <Steps
              direction="vertical"
              size="small"
              items={selectedTemplate.steps.map((step) => ({
                title: (
                  <span>
                    <Tag color={stepTypeColorMap[step.stepType]} style={{ marginRight: 8 }}>
                      {stepTypeLabelMap[step.stepType]}
                    </Tag>
                    {step.stepName}
                  </span>
                ),
                description: (
                  <div style={{ color: '#666' }}>
                    {step.description}
                    {step.command && (
                      <div style={{ fontFamily: 'monospace', background: '#f5f5f5', padding: '2px 8px', borderRadius: 4, marginTop: 4, fontSize: 12 }}>
                        {step.command}
                      </div>
                    )}
                    {step.condition && (
                      <div style={{ color: 'var(--color-primary-600)', fontSize: 12, marginTop: 4 }}>
                        {t('ops.condition')}: {step.condition}
                      </div>
                    )}
                    {step.waitSeconds && (
                      <div style={{ color: '#fa8c16', fontSize: 12, marginTop: 4 }}>
                        {t('ops.waitSeconds', { seconds: String(step.waitSeconds) })}
                      </div>
                    )}
                    {step.rollbackCommand && (
                      <div style={{ color: '#ff4d4f', fontSize: 12, marginTop: 4 }}>
                        {t('ops.rollback')}: {step.rollbackCommand}
                      </div>
                    )}
                  </div>
                ),
                status: 'process' as const,
              }))}
            />
          </>
        )}
      </Drawer>
    </ListPageLayout>
  );
}
