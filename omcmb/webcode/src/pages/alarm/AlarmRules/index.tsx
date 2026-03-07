import React, { useCallback, useMemo, useState } from 'react';
import { Button, Form, Input, Modal, Select, Space, Switch, Tag, message } from 'antd';
import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useAlarmRules, useDeleteAlarmRules, useUpdateAlarmRule, useCreateAlarmRule } from '@/hooks/api/useAlarms';
import { useT } from '@/hooks/useT';
import type { AlarmRule } from '@/types/alarm';

const SEVERITY_TAG_COLOR: Record<string, string> = {
  critical: 'red', major: 'orange', minor: 'gold', warning: 'blue',
};

const RULE_TYPE_LABEL: Record<string, string> = {
  threshold: 'threshold',
  correlation: 'correlation',
  suppression: 'suppression',
  escalation: 'escalation',
};

export default function AlarmRules() {
  const t = useT();
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize] = useState(20);
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<AlarmRule | null>(null);
  const [form] = Form.useForm();

  const SEVERITY_LABEL: Record<string, string> = useMemo(() => ({
    critical: t('alarm.severity.critical'),
    major: t('alarm.severity.major'),
    minor: t('alarm.severity.minor'),
    warning: t('alarm.severity.warning'),
  }), [t]);

  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    { name: 'ruleName', label: t('table.name'), type: 'input' },
    {
      name: 'ruleType',
      label: t('table.type'),
      type: 'select',
      options: [
        { label: 'threshold', value: 'threshold' },
        { label: 'correlation', value: 'correlation' },
        { label: 'suppression', value: 'suppression' },
        { label: 'escalation', value: 'escalation' },
      ],
    },
    {
      name: 'severity',
      label: t('alarm.severity'),
      type: 'select',
      options: [
        { label: t('alarm.severity.critical'), value: 'critical' },
        { label: t('alarm.severity.major'), value: 'major' },
        { label: t('alarm.severity.minor'), value: 'minor' },
        { label: t('alarm.severity.warning'), value: 'warning' },
      ],
    },
  ], [t]);

  const queryParams = useMemo(() => ({ page: currentPage, pageSize }), [currentPage, pageSize]);

  const { data, isLoading, refetch } = useAlarmRules(queryParams);
  const deleteRules = useDeleteAlarmRules();
  const updateRule = useUpdateAlarmRule();
  const createRule = useCreateAlarmRule();

  const rules: AlarmRule[] = data?.items ?? [];
  const total = data?.total ?? 0;

  // Client-side filter
  const filteredRules = useMemo(() => {
    return rules.filter((r) => {
      if (filterParams.ruleName && !r.ruleName.toLowerCase().includes(String(filterParams.ruleName).toLowerCase())) return false;
      if (filterParams.ruleType && r.ruleType !== filterParams.ruleType) return false;
      if (filterParams.severity && r.severity !== filterParams.severity) return false;
      return true;
    });
  }, [rules, filterParams]);

  const handleToggle = useCallback(
    async (rule: AlarmRule, checked: boolean) => {
      await updateRule.mutateAsync({ id: rule.id, data: { enabled: checked } as Partial<AlarmRule> });
      void message.success(checked ? t('common.enable') : t('common.disable'));
    },
    [updateRule, t]
  );

  const handleDelete = useCallback(
    (ids: string[]) => {
      Modal.confirm({
        title: t('common.confirmDelete'),
        content: t('common.deleteConfirmMsg', { count: ids.length }),
        okText: t('common.confirmDelete'),
        okType: 'danger',
        onOk: async () => {
          await deleteRules.mutateAsync(ids);
          setSelectedRowKeys([]);
          void message.success(t('common.deleteSuccess'));
        },
      });
    },
    [deleteRules, t]
  );

  const handleEdit = useCallback((rule: AlarmRule) => {
    setEditingRule(rule);
    form.setFieldsValue({
      ruleName: rule.ruleName,
      ruleType: rule.ruleType,
      severity: rule.severity,
    });
    setEditModalOpen(true);
  }, [form]);

  const handleCreate = useCallback(() => {
    setEditingRule(null);
    form.resetFields();
    setEditModalOpen(true);
  }, [form]);

  const handleSave = useCallback(async () => {
    try {
      const values = await form.validateFields() as Partial<AlarmRule>;
      if (editingRule) {
        await updateRule.mutateAsync({ id: editingRule.id, data: values });
        void message.success(t('status.success'));
      } else {
        await createRule.mutateAsync({
          ruleName: (values as { ruleName: string }).ruleName,
          ruleType: (values as { ruleType: string }).ruleType,
          severity: (values as { severity: AlarmRule['severity'] }).severity,
          enabled: true,
          conditions: [],
          actions: [],
        } as Parameters<typeof createRule.mutateAsync>[0]);
        void message.success(t('status.success'));
      }
      setEditModalOpen(false);
    } catch {
      // validation error
    }
  }, [editingRule, form, updateRule, createRule, t]);

  const columns = useMemo(
    (): DataTableColumn<AlarmRule>[] => [
      { key: 'ruleName', title: t('table.name'), dataIndex: 'ruleName', width: 200, ellipsis: true },
      {
        key: 'ruleType',
        title: t('table.type'),
        dataIndex: 'ruleType',
        width: 120,
        render: (v) => <Tag>{RULE_TYPE_LABEL[String(v)] ?? String(v)}</Tag>,
      },
      {
        key: 'severity',
        title: t('alarm.severity'),
        dataIndex: 'severity',
        width: 100,
        render: (_val, record) => (
          <Tag color={SEVERITY_TAG_COLOR[record.severity] ?? 'default'}>
            {SEVERITY_LABEL[record.severity] ?? record.severity}
          </Tag>
        ),
      },
      {
        key: 'enabled',
        title: t('table.status'),
        dataIndex: 'enabled',
        width: 90,
        render: (_val, record) => (
          <Switch
            checked={record.enabled}
            size="small"
            loading={updateRule.isPending}
            onChange={(checked) => void handleToggle(record, checked)}
          />
        ),
      },
      {
        key: 'createTime',
        title: t('table.createTime'),
        dataIndex: 'createTime',
        width: 160,
        render: (v) => v ? new Date(String(v)).toLocaleString('zh-CN') : '-',
      },
      {
        key: 'updateTime',
        title: t('table.updateTime'),
        dataIndex: 'updateTime',
        width: 160,
        render: (v) => v ? new Date(String(v)).toLocaleString('zh-CN') : '-',
      },
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 120,
        fixed: 'right',
        render: (_val, record) => (
          <Space size={4}>
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
            >
              {t('common.edit')}
            </Button>
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => handleDelete([record.id])}
            >
              {t('common.delete')}
            </Button>
          </Space>
        ),
      },
    ],
    [handleToggle, handleEdit, handleDelete, updateRule.isPending, t, SEVERITY_LABEL]
  );

  const batchActions = useMemo(
    (): BatchAction[] => [
      {
        key: 'batch-delete',
        label: t('common.batchDelete'),
        icon: <DeleteOutlined />,
        danger: true,
        onClick: (keys) => handleDelete(keys as string[]),
      },
    ],
    [handleDelete, t]
  );

  return (
    <>
      <ListPageLayout
        title={t('nav.alarm.rules')}
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            {t('common.add')}
          </Button>
        }
      >
        <FilterBar
          filterId="alarm-rules"
          fields={FILTER_FIELDS}
          onSearch={(v) => { setFilterParams(v); setCurrentPage(1); }}
          onReset={() => { setFilterParams({}); setCurrentPage(1); }}
          collapsedRows={1}
        />

        <DataTable<AlarmRule>
          tableId="alarm-rules-table"
          columns={columns}
          dataSource={filteredRules}
          loading={isLoading}
          rowKey="id"
          selectable
          selectedRowKeys={selectedRowKeys}
          onSelectionChange={(keys) => setSelectedRowKeys(keys)}
          total={total}
          pageSize={pageSize}
          currentPage={currentPage}
          onPageChange={(p) => setCurrentPage(p)}
          batchActions={batchActions}
          onRefresh={() => void refetch()}
          defaultDensity="compact"
        />
      </ListPageLayout>

      <Modal
        title={editingRule ? t('common.edit') : t('common.add')}
        open={editModalOpen}
        onOk={() => void handleSave()}
        onCancel={() => setEditModalOpen(false)}
        okText={t('common.save')}
        confirmLoading={updateRule.isPending || createRule.isPending}
        width={520}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="ruleName" label={t('table.name')} rules={[{ required: true, message: t('common.placeholder') }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item name="ruleType" label={t('table.type')} rules={[{ required: true, message: t('common.pleaseSelect') }]}>
            <Select
              placeholder={t('common.pleaseSelect')}
              options={[
                { label: 'threshold', value: 'threshold' },
                { label: 'correlation', value: 'correlation' },
                { label: 'suppression', value: 'suppression' },
                { label: 'escalation', value: 'escalation' },
              ]}
            />
          </Form.Item>
          <Form.Item name="severity" label={t('alarm.severity')} rules={[{ required: true, message: t('common.pleaseSelect') }]}>
            <Select
              placeholder={t('common.pleaseSelect')}
              options={[
                { label: t('alarm.severity.critical'), value: 'critical' },
                { label: t('alarm.severity.major'), value: 'major' },
                { label: t('alarm.severity.minor'), value: 'minor' },
                { label: t('alarm.severity.warning'), value: 'warning' },
              ]}
            />
          </Form.Item>
          <Form.Item name="description" label={t('table.description')}>
            <Input.TextArea rows={3} placeholder={t('common.placeholder')} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
