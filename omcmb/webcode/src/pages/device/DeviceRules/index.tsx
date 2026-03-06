import React, { useCallback, useMemo, useState } from 'react';
import {
  Button,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Switch,
  Tag,
  Typography,
  message,
} from 'antd';
import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';

const { Text } = Typography;

interface DeviceRule {
  id: string;
  ruleName: string;
  ruleType: 'threshold' | 'pattern' | 'schedule' | 'event';
  status: boolean;
  conditions: string;
  actions: string;
  createTime: string;
  updateTime: string;
}

const MOCK_RULES: DeviceRule[] = [
  {
    id: '1', ruleName: 'CPU高负载告警规则', ruleType: 'threshold', status: true,
    conditions: 'CPU > 90% 持续 5分钟', actions: '发送告警通知',
    createTime: '2024-01-10 09:00:00', updateTime: '2024-02-15 14:30:00',
  },
  {
    id: '2', ruleName: '设备离线自动重连', ruleType: 'event', status: true,
    conditions: '设备离线超过 60秒', actions: '触发重连, 发送通知',
    createTime: '2024-01-12 10:00:00', updateTime: '2024-01-12 10:00:00',
  },
  {
    id: '3', ruleName: '温度过高保护规则', ruleType: 'threshold', status: false,
    conditions: '设备温度 > 65C', actions: '功率降档, 发送紧急告警',
    createTime: '2024-01-15 11:00:00', updateTime: '2024-02-20 09:15:00',
  },
  {
    id: '4', ruleName: '夜间省电模式', ruleType: 'schedule', status: true,
    conditions: '23:00 - 06:00', actions: '降低功率至 50%',
    createTime: '2024-01-20 08:00:00', updateTime: '2024-01-20 08:00:00',
  },
  {
    id: '5', ruleName: '流量异常检测', ruleType: 'pattern', status: true,
    conditions: '流量突增 > 200%', actions: '记录日志, 触发告警',
    createTime: '2024-02-01 09:00:00', updateTime: '2024-02-28 16:00:00',
  },
  {
    id: '6', ruleName: '内存泄漏检测', ruleType: 'pattern', status: false,
    conditions: '内存使用率持续上升 > 30分钟', actions: '重启进程, 告警通知',
    createTime: '2024-02-10 10:00:00', updateTime: '2024-02-10 10:00:00',
  },
];

const RULE_TYPE_COLOR: Record<DeviceRule['ruleType'], string> = {
  threshold: 'orange',
  pattern: 'blue',
  schedule: 'green',
  event: 'purple',
};

export default function DeviceRules() {
  const t = useT();
  const [rules, setRules] = useState<DeviceRule[]>(MOCK_RULES);
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [currentPage, setCurrentPage] = useState(1);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<DeviceRule | null>(null);
  const [form] = Form.useForm();

  const RULE_TYPE_LABEL: Record<DeviceRule['ruleType'], string> = useMemo(() => ({
    threshold: t('alarm.type'),
    pattern: t('alarm.type'),
    schedule: t('alarm.type'),
    event: t('alarm.type'),
  }), [t]);

  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    { name: 'ruleName', label: t('table.name'), type: 'input' },
    {
      name: 'ruleType',
      label: t('table.type'),
      type: 'select',
      options: [
        { label: 'threshold', value: 'threshold' },
        { label: 'pattern', value: 'pattern' },
        { label: 'schedule', value: 'schedule' },
        { label: 'event', value: 'event' },
      ],
    },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.enabled'), value: 'true' },
        { label: t('status.disabled'), value: 'false' },
      ],
    },
  ], [t]);

  const filteredRules = useMemo(() => {
    return rules.filter((r) => {
      if (filterParams.ruleName && !r.ruleName.toLowerCase().includes(String(filterParams.ruleName).toLowerCase())) return false;
      if (filterParams.ruleType && r.ruleType !== filterParams.ruleType) return false;
      if (filterParams.status !== undefined && filterParams.status !== '') {
        const isEnabled = filterParams.status === 'true';
        if (r.status !== isEnabled) return false;
      }
      return true;
    });
  }, [rules, filterParams]);

  const handleToggle = useCallback((id: string, checked: boolean) => {
    setRules((prev) => prev.map((r) => (r.id === id ? { ...r, status: checked } : r)));
    void message.success(checked ? t('common.enable') : t('common.disable'));
  }, [t]);

  const handleDelete = useCallback((id: string, name: string) => {
    Modal.confirm({
      title: t('common.confirmDelete'),
      content: t('common.deleteConfirmMsg', { count: 1 }),
      okText: t('common.confirmDelete'),
      okType: 'danger',
      onOk: () => {
        setRules((prev) => prev.filter((r) => r.id !== id));
        void message.success(t('common.deleteSuccess'));
      },
    });
  }, [t]);

  const handleEdit = useCallback((rule: DeviceRule) => {
    setEditingRule(rule);
    form.setFieldsValue({
      ruleName: rule.ruleName,
      ruleType: rule.ruleType,
      conditions: rule.conditions,
      actions: rule.actions,
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
      const values = await form.validateFields() as { ruleName: string; ruleType: DeviceRule['ruleType']; conditions: string; actions: string };
      if (editingRule) {
        setRules((prev) =>
          prev.map((r) =>
            r.id === editingRule.id
              ? { ...r, ...values, updateTime: new Date().toLocaleString('zh-CN') }
              : r
          )
        );
        void message.success(t('status.success'));
      } else {
        const newRule: DeviceRule = {
          id: String(Date.now()),
          ruleName: values.ruleName,
          ruleType: values.ruleType,
          status: true,
          conditions: values.conditions,
          actions: values.actions,
          createTime: new Date().toLocaleString('zh-CN'),
          updateTime: new Date().toLocaleString('zh-CN'),
        };
        setRules((prev) => [newRule, ...prev]);
        void message.success(t('status.success'));
      }
      setEditModalOpen(false);
    } catch {
      // validation error
    }
  }, [editingRule, form, t]);

  const columns = useMemo(
    (): DataTableColumn<DeviceRule>[] => [
      { key: 'ruleName', title: t('table.name'), dataIndex: 'ruleName', width: 200, ellipsis: true },
      {
        key: 'ruleType',
        title: t('table.type'),
        dataIndex: 'ruleType',
        width: 110,
        render: (_val, record) => (
          <Tag color={RULE_TYPE_COLOR[record.ruleType]}>{record.ruleType}</Tag>
        ),
      },
      {
        key: 'status',
        title: t('table.status'),
        dataIndex: 'status',
        width: 90,
        render: (_val, record) => (
          <Switch
            checked={record.status}
            size="small"
            onChange={(checked) => handleToggle(record.id, checked)}
          />
        ),
      },
      { key: 'conditions', title: t('alarm.content'), dataIndex: 'conditions', width: 200, ellipsis: true },
      { key: 'actions', title: t('table.operation'), dataIndex: 'actions', width: 180, ellipsis: true },
      { key: 'createTime', title: t('table.createTime'), dataIndex: 'createTime', width: 160 },
      { key: 'updateTime', title: t('table.updateTime'), dataIndex: 'updateTime', width: 160 },
      {
        key: 'ops',
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
              onClick={() => handleDelete(record.id, record.ruleName)}
            >
              {t('common.delete')}
            </Button>
          </Space>
        ),
      },
    ],
    [handleToggle, handleEdit, handleDelete, t]
  );

  return (
    <>
      <ListPageLayout
        title={t('nav.device.rules')}
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            {t('common.add')}
          </Button>
        }
      >
        <FilterBar
          filterId="device-rules"
          fields={FILTER_FIELDS}
          onSearch={(v) => { setFilterParams(v); setCurrentPage(1); }}
          onReset={() => { setFilterParams({}); setCurrentPage(1); }}
          collapsedRows={1}
        />

        <DataTable<DeviceRule>
          tableId="device-rules-table"
          columns={columns}
          dataSource={filteredRules}
          loading={false}
          rowKey="id"
          selectable
          total={filteredRules.length}
          pageSize={20}
          currentPage={currentPage}
          onPageChange={(p) => setCurrentPage(p)}
          defaultDensity="compact"
        />
      </ListPageLayout>

      <Modal
        title={editingRule ? t('common.edit') : t('common.add')}
        open={editModalOpen}
        onOk={() => void handleSave()}
        onCancel={() => setEditModalOpen(false)}
        okText={t('common.save')}
        width={520}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="ruleName" label={t('table.name')} rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item name="ruleType" label={t('table.type')} rules={[{ required: true }]}>
            <Select
              placeholder={t('common.pleaseSelect')}
              options={[
                { label: 'threshold', value: 'threshold' },
                { label: 'pattern', value: 'pattern' },
                { label: 'schedule', value: 'schedule' },
                { label: 'event', value: 'event' },
              ]}
            />
          </Form.Item>
          <Form.Item name="conditions" label={t('alarm.content')} rules={[{ required: true }]}>
            <Input.TextArea rows={3} placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item name="actions" label={t('table.operation')} rules={[{ required: true }]}>
            <Input.TextArea rows={3} placeholder={t('common.placeholder')} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
