import { useState, useMemo } from 'react';
import { Button, Dropdown, Tag, Space, Modal, Form, Input, Select, message, Drawer, Steps, Descriptions, Tooltip } from 'antd';
import type { MenuProps } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, EyeOutlined, MoreOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import {
  useOpsTemplates,
  useCreateOpsTemplate,
  useDeleteOpsTemplates,
} from '@core/hooks/api/useOpsTools';
import type { OpsTemplate, OpsStep } from '@core/mock/data/opsTools';
import { useT } from '@/hooks/useT';
import { usePermission } from '@core/hooks/usePermission';

const stepTypeColorMap: Record<OpsStep['stepType'], string> = {
  mml: 'blue',
  check: 'green',
  wait: 'orange',
  notify: 'purple',
  script: 'cyan',
};

const categoryColorMap: Record<string, string> = {
  巡检运维: 'blue',
  故障处置: 'red',
  性能优化: 'green',
  软件管理: 'cyan',
  网络配置: 'purple',
  维护操作: 'orange',
};

const CATEGORY_OPTIONS_KEYS: ReadonlyArray<{ value: string; labelKey: string }> = [
  { value: '巡检运维', labelKey: 'ops.catInspection' },
  { value: '故障处置', labelKey: 'ops.catFault' },
  { value: '性能优化', labelKey: 'ops.catPerformance' },
  { value: '软件管理', labelKey: 'ops.catSoftware' },
  { value: '网络配置', labelKey: 'ops.catNetwork' },
  { value: '维护操作', labelKey: 'ops.catMaintenance' },
];

const DEVICE_TYPE_OPTIONS = ['eNB', 'gNB', 'CPE', 'eGW'];

export default function Templates() {
  const t = useT();
  const canCreate = usePermission('ops:template:create');
  const canEdit = usePermission('ops:template:edit');
  const canDelete = usePermission('ops:template:delete');

  const [keyword, setKeyword] = useState<string>('');
  const [category, setCategory] = useState<string | undefined>(undefined);
  const [targetDeviceType, setTargetDeviceType] = useState<string | undefined>(undefined);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [createVisible, setCreateVisible] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedTemplate, setSelectedTemplate] = useState<OpsTemplate | null>(null);
  const [form] = Form.useForm();

  const queryParams = useMemo(
    () => ({ keyword: keyword || undefined, category, targetDeviceType, page, pageSize }),
    [keyword, category, targetDeviceType, page, pageSize],
  );
  const { data, isLoading, refetch } = useOpsTemplates(queryParams);
  const createMutation = useCreateOpsTemplate();
  const deleteMutation = useDeleteOpsTemplates();

  const items: OpsTemplate[] = data?.items ?? [];
  const total = data?.total ?? 0;

  const stepTypeLabelMap: Record<OpsStep['stepType'], string> = useMemo(
    () => ({
      mml: t('ops.stepTypeMml'),
      check: t('ops.stepTypeCheck'),
      wait: t('ops.stepTypeWait'),
      notify: t('ops.stepTypeNotify'),
      script: t('ops.stepTypeScript'),
    }),
    [t],
  );

  const filterFields: FilterField[] = useMemo(
    () => [
      { name: 'keyword', label: t('ops.templateName'), type: 'input', placeholder: t('ops.templateNamePlaceholder') },
      {
        name: 'category',
        label: t('mr.category'),
        type: 'select',
        options: CATEGORY_OPTIONS_KEYS.map((o) => ({ label: t(o.labelKey), value: o.value })),
      },
      {
        name: 'targetDeviceType',
        label: t('ops.targetDevice'),
        type: 'select',
        options: DEVICE_TYPE_OPTIONS.map((v) => ({ label: v, value: v })),
      },
    ],
    [t],
  );

  const handleCreate = () => {
    form.validateFields().then((vals) => {
      const payload = {
        templateName: vals.templateName as string,
        description: (vals.description as string) ?? '',
        category: vals.category as string,
        targetDeviceTypes: (vals.targetDeviceTypes as string[]) ?? [],
        steps: [],
        estimatedDuration: Number(vals.estimatedDuration ?? 60),
        creator: 'admin',
        tags: [],
      };
      createMutation.mutate(payload, {
        onSuccess: () => {
          void message.success(t('ops.templateCreateSuccess'));
          setCreateVisible(false);
          form.resetFields();
        },
        onError: () => {
          void message.error(t('common.operationFailed'));
        },
      });
    });
  };

  const handleDelete = (id: string) => {
    Modal.confirm({
      title: t('common.confirmDelete'),
      content: t('ops.deleteConfirmContent'),
      okType: 'danger',
      onOk: () =>
        deleteMutation.mutateAsync([id]).then(
          () => message.success(t('common.deleteSuccess')),
          () => message.error(t('common.operationFailed')),
        ),
    });
  };

  const columns: DataTableColumn<OpsTemplate & Record<string, unknown>>[] = useMemo(
    () => [
      { key: 'templateName', title: t('ops.templateName'), dataIndex: 'templateName', ellipsis: true, width: 200 },
      {
        key: 'category',
        title: t('mr.category'),
        dataIndex: 'category',
        width: 100,
        render: (val) => <Tag color={categoryColorMap[String(val)] ?? 'default'}>{String(val)}</Tag>,
      },
      {
        key: 'targetDeviceTypes',
        title: t('ops.applicableDevices'),
        dataIndex: 'targetDeviceTypes',
        width: 160,
        render: (val) => (val as string[]).map((tp) => <Tag key={tp}>{tp}</Tag>),
      },
      { key: 'description', title: t('table.description'), dataIndex: 'description', ellipsis: true },
      {
        key: 'steps',
        title: t('ops.stepCount'),
        dataIndex: 'steps',
        width: 80,
        render: (val) => (val as OpsStep[]).length,
      },
      {
        key: 'estimatedDuration',
        title: t('ops.estimatedDuration'),
        dataIndex: 'estimatedDuration',
        width: 100,
        render: (val) => {
          const secs = Number(val);
          if (secs >= 60) return `${Math.floor(secs / 60)} ${t('ops.minutes')}`;
          return `${secs} ${t('ops.seconds')}`;
        },
      },
      { key: 'useCount', title: t('ops.useCount'), dataIndex: 'useCount', width: 90 },
      { key: 'creator', title: t('mr.creator'), dataIndex: 'creator', width: 90 },
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 110,
        fixed: 'right',
        render: (_, record) => {
          const tp = record as OpsTemplate;
          const moreItems: MenuProps['items'] = [
            { key: 'edit', label: t('common.edit'), icon: <EditOutlined />, disabled: !canEdit },
            { type: 'divider' as const },
            {
              key: 'delete',
              label: t('common.delete'),
              icon: <DeleteOutlined />,
              danger: true,
              disabled: !canDelete,
              onClick: () => canDelete && handleDelete(tp.id),
            },
          ];
          return (
            <Space size={4}>
              <Button
                type="link"
                size="small"
                icon={<EyeOutlined />}
                onClick={() => {
                  setSelectedTemplate(tp);
                  setDetailVisible(true);
                }}
              >
                {t('common.detail')}
              </Button>
              <Dropdown menu={{ items: moreItems }} trigger={['click']}>
                <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
              </Dropdown>
            </Space>
          );
        },
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [t, canEdit, canDelete],
  );

  return (
    <ListPageLayout
      title={t('nav.ops.templates')}
      subtitle={t('ops.templatesSubtitle')}
      extra={
        <Tooltip title={canCreate ? '' : t('common.noPermission')}>
          <Button type="primary" icon={<PlusOutlined />} disabled={!canCreate} onClick={() => setCreateVisible(true)}>
            {t('ops.newTemplate')}
          </Button>
        </Tooltip>
      }
    >
      <FilterBar
        filterId="ops-templates-filter"
        fields={filterFields}
        onSearch={(vals) => {
          setKeyword(String(vals.keyword ?? ''));
          setCategory((vals.category as string) || undefined);
          setTargetDeviceType((vals.targetDeviceType as string) || undefined);
          setPage(1);
        }}
        onReset={() => {
          setKeyword('');
          setCategory(undefined);
          setTargetDeviceType(undefined);
          setPage(1);
        }}
      />
      <DataTable
        tableId="ops-templates-list"
        columns={columns}
        dataSource={items as (OpsTemplate & Record<string, unknown>)[]}
        loading={isLoading}
        rowKey="id"
        total={total}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => {
          setPage(p);
          setPageSize(s);
        }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1100 }}
      />

      <Modal
        title={t('ops.newOpsTemplate')}
        open={createVisible}
        onOk={handleCreate}
        confirmLoading={createMutation.isPending}
        onCancel={() => {
          setCreateVisible(false);
          form.resetFields();
        }}
        width={560}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="templateName" label={t('ops.templateName')} rules={[{ required: true }]}>
            <Input placeholder={t('ops.templateNamePlaceholder')} />
          </Form.Item>
          <Form.Item name="category" label={t('mr.category')} rules={[{ required: true }]}>
            <Select options={CATEGORY_OPTIONS_KEYS.map((o) => ({ label: t(o.labelKey), value: o.value }))} />
          </Form.Item>
          <Form.Item name="targetDeviceTypes" label={t('ops.applicableDeviceTypes')} rules={[{ required: true }]}>
            <Select mode="multiple" options={DEVICE_TYPE_OPTIONS.map((v) => ({ label: v, value: v }))} />
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
        size={680}
      >
        {selectedTemplate && (
          <>
            <Descriptions bordered column={2} size="small" style={{ marginBottom: 24 }}>
              <Descriptions.Item label={t('ops.templateName')} span={2}>
                {selectedTemplate.templateName}
              </Descriptions.Item>
              <Descriptions.Item label={t('mr.category')}>
                <Tag color={categoryColorMap[selectedTemplate.category] ?? 'default'}>{selectedTemplate.category}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label={t('ops.useCount')}>{selectedTemplate.useCount}</Descriptions.Item>
              <Descriptions.Item label={t('ops.applicableDevices')} span={2}>
                {selectedTemplate.targetDeviceTypes.map((tp) => (
                  <Tag key={tp}>{tp}</Tag>
                ))}
              </Descriptions.Item>
              <Descriptions.Item label={t('ops.estimatedDuration')}>
                {selectedTemplate.estimatedDuration >= 60
                  ? `${Math.floor(selectedTemplate.estimatedDuration / 60)} ${t('ops.minutes')}`
                  : `${selectedTemplate.estimatedDuration} ${t('ops.seconds')}`}
              </Descriptions.Item>
              <Descriptions.Item label={t('mr.creator')}>{selectedTemplate.creator}</Descriptions.Item>
              <Descriptions.Item label={t('ops.tags')} span={2}>
                {selectedTemplate.tags.map((tag) => (
                  <Tag key={tag} color="blue">
                    {tag}
                  </Tag>
                ))}
              </Descriptions.Item>
              <Descriptions.Item label={t('table.description')} span={2}>
                {selectedTemplate.description}
              </Descriptions.Item>
            </Descriptions>

            <div style={{ marginBottom: 8, fontWeight: 600, fontSize: 14 }}>
              {t('ops.executionSteps', { count: String(selectedTemplate.steps.length) })}
            </div>
            <Steps
              orientation="vertical"
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
                      <div
                        style={{
                          fontFamily: 'monospace',
                          background: '#f5f5f5',
                          padding: '2px 8px',
                          borderRadius: 4,
                          marginTop: 4,
                          fontSize: 12,
                        }}
                      >
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
