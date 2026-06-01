import { useCallback, useMemo, useState } from 'react';
import { App, Button, Card, Dropdown, Space, Switch, Tag, Typography, message } from 'antd';
import type { MenuProps } from 'antd';
import {
  CheckCircleOutlined,
  DeleteOutlined,
  MoreOutlined,
  PlusOutlined,
  StopOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { BatchAction, DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useAlarmRules, useCreateAlarmRule, useDeleteAlarmRules, useUpdateAlarmRule } from '@core/hooks/api/useAlarms';
import { useT } from '@/hooks/useT';
import type { AlarmRule, AlarmRuleCondition, AlarmRuleAction } from '@core/types/alarm';
import AlarmRuleDrawer, { type AlarmRuleFormData } from './AlarmRuleDrawer';

const { Text } = Typography;

// 执行动作配置 (后端 action 值)
const RULE_TYPE_CONFIG: Record<string, { label: string; color: string }> = {
  default: { label: 'alarm.ruleType.default', color: 'blue' },
  ignore: { label: 'alarm.ruleType.forbidReport', color: 'red' },
  auto_acknowledge: { label: 'alarm.ruleType.autoConfirm', color: 'green' },
  auto_clear: { label: 'alarm.ruleType.autoClear', color: 'orange' },
};

type DrawerMode = 'add' | 'edit' | 'view';

function getRuleFilterDimension(rule: AlarmRule, t: (key: string, values?: Record<string, unknown>) => string): string {
  if (rule.conditions.some((condition) => condition.field === 'alarm_identifier')) {
    return t('alarm.ruleFilterType.alarmIdentifier');
  }
  if (rule.conditions.some((condition) => condition.field === 'alarm_source')) {
    return t('alarm.ruleFilterType.alarmSource');
  }
  if (rule.conditions.some((condition) => condition.field === 'device_group_id')) {
    return t('alarm.ruleFilterType.deviceGroup');
  }
  if (rule.conditions.some((condition) => condition.field === 'device_id')) {
    return t('alarm.ruleFilterType.device');
  }
  return rule.isDefault ? t('alarm.defaultRule') : '-';
}

export default function AlarmRules() {
  const t = useT();
  const { modal } = App.useApp();
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [togglingId, setTogglingId] = useState<string | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 抽屉状态
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<DrawerMode>('add');
  const [currentRule, setCurrentRule] = useState<AlarmRule | null>(null);

  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    {
      name: 'searchText',
      label: t('alarm.ruleName'),
      type: 'input',
      placeholder: t('alarm.searchPlaceholder'),
    },
  ], [t]);

  const queryParams = useMemo(() => ({ ...filterParams, page: currentPage, pageSize }), [filterParams, currentPage, pageSize]);

  const { data, isLoading, refetch } = useAlarmRules(queryParams);
  const deleteRules = useDeleteAlarmRules();
  const updateRule = useUpdateAlarmRule();
  const createRule = useCreateAlarmRule();

  const rules: AlarmRule[] = data?.items ?? [];
  const total = data?.total ?? 0;
  const selectedRules = useMemo(
    () => rules.filter((rule) => selectedRowKeys.includes(rule.id)),
    [rules, selectedRowKeys]
  );

  // 启用/禁用规则
  const handleToggle = useCallback(
    async (rule: AlarmRule, checked: boolean) => {
      setTogglingId(rule.id);
      try {
        await updateRule.mutateAsync({ id: rule.id, data: { enabled: checked } });
        message.success(t(checked ? 'alarm.ruleEnabled' : 'alarm.ruleDisabled'));
        void refetch();
      } catch (error) {
        message.error(t('alarm.ruleToggleFailed'));
        console.error('Toggle rule failed:', error);
      } finally {
        setTogglingId(null);
      }
    },
    [updateRule, refetch, t]
  );

  // 打开添加抽屉
  const handleAdd = useCallback(() => {
    setCurrentRule(null);
    setDrawerMode('add');
    setDrawerOpen(true);
  }, []);

  // 打开编辑抽屉
  const handleEdit = useCallback((rule: AlarmRule) => {
    if (rule.enabled) {
      modal.warning({
        title: t('common.warning'),
        content: t('alarm.cannotEditEnabledRule'),
      });
      return;
    }
    setCurrentRule(rule);
    setDrawerMode('edit');
    setDrawerOpen(true);
  }, [modal, t]);

  // 打开查看抽屉
  const handleView = useCallback((rule: AlarmRule) => {
    setCurrentRule(rule);
    setDrawerMode('view');
    setDrawerOpen(true);
  }, []);

  // 关闭抽屉
  const handleDrawerClose = useCallback(() => {
    setDrawerOpen(false);
    setCurrentRule(null);
  }, []);

  // 提交表单
  const handleDrawerSubmit = useCallback(async (formData: AlarmRuleFormData) => {
    // 构造 conditions 从表单数据
    const conditions: AlarmRuleCondition[] = [];
    if (formData.selectedAlarms.length > 0) {
      conditions.push({ field: 'alarm_identifier', operator: 'contains', value: formData.selectedAlarms });
    }
    if (formData.deviceSelectionMode === 'devices' && formData.selectedDevices.length > 0) {
      conditions.push({ field: 'device_id', operator: 'contains', value: formData.selectedDevices });
    }
    if (formData.deviceSelectionMode === 'groups' && formData.selectedGroups.length > 0) {
      conditions.push({ field: 'device_group_id', operator: 'contains', value: formData.selectedGroups });
    }

    // 从 ruleType 派生 actions
    const actions: AlarmRuleAction[] = [];
    if (formData.ruleType === 'ignore') {
      actions.push({ type: 'suppress', target: 'ignore' });
    } else if (formData.ruleType === 'auto_acknowledge') {
      actions.push({ type: 'suppress', target: 'auto_acknowledge' });
    } else if (formData.ruleType === 'auto_clear') {
      actions.push({ type: 'suppress', target: 'auto_clear' });
    }

    const ruleData = {
      ruleName: formData.ruleName,
      enabled: formData.status,
      ruleType: formData.ruleType,
      severity: 'warning' as const,
      conditions,
      actions,
    };

    if (drawerMode === 'add') {
      await createRule.mutateAsync(ruleData);
    } else if (drawerMode === 'edit' && currentRule) {
      await updateRule.mutateAsync({ id: currentRule.id, data: ruleData });
    }
    void refetch();
  }, [drawerMode, currentRule, createRule, updateRule, refetch]);

  // 删除规则
  const handleDelete = useCallback(
    (rule: AlarmRule) => {
      if (rule.isDefault) {
        modal.warning({
          title: t('common.warning'),
          content: t('alarm.cannotDeleteDefaultRule'),
        });
        return;
      }
      if (rule.enabled) {
        modal.warning({
          title: t('common.warning'),
          content: t('alarm.cannotDeleteEnabledRule'),
        });
        return;
      }
      modal.confirm({
        title: t('common.confirmDelete'),
        content: t('alarm.deleteRuleConfirm', { name: rule.ruleName }),
        okText: t('common.delete'),
        okType: 'danger',
        onOk: async () => {
          await deleteRules.mutateAsync([rule.id]);
          void refetch();
        },
      });
    },
    [deleteRules, modal, t, refetch]
  );

  const handleBatchDelete = useCallback(
    (keys: React.Key[]) => {
      const targetRules = rules.filter((rule) => keys.includes(rule.id));
      if (targetRules.length === 0) {
        return;
      }
      if (targetRules.some((rule) => rule.isDefault)) {
        modal.warning({
          title: t('common.warning'),
          content: t('alarm.cannotDeleteDefaultRule'),
        });
        return;
      }
      if (targetRules.some((rule) => rule.enabled)) {
        modal.warning({
          title: t('common.warning'),
          content: t('alarm.cannotDeleteEnabledRule'),
        });
        return;
      }

      modal.confirm({
        title: t('common.confirmDelete'),
        content: t('alarm.batchDeleteRuleConfirm', { count: targetRules.length }),
        okText: t('common.delete'),
        okType: 'danger',
        onOk: async () => {
          await deleteRules.mutateAsync(targetRules.map((rule) => rule.id));
          setSelectedRowKeys([]);
          void refetch();
        },
      });
    },
    [deleteRules, modal, refetch, rules, t]
  );

  const handleBatchToggle = useCallback(
    (targetEnabled: boolean) => {
      const targetRules = rules.filter((rule) => selectedRowKeys.includes(rule.id) && rule.enabled !== targetEnabled);
      if (targetRules.length === 0) {
        return;
      }

      modal.confirm({
        title: targetEnabled ? t('common.enable') : t('common.disable'),
        content: t(targetEnabled ? 'alarm.batchEnableRuleConfirm' : 'alarm.batchDisableRuleConfirm', { count: targetRules.length }),
        okText: targetEnabled ? t('common.enable') : t('common.disable'),
        onOk: async () => {
          await Promise.all(
            targetRules.map((rule) => updateRule.mutateAsync({ id: rule.id, data: { enabled: targetEnabled } }))
          );
          message.success(
            t(targetEnabled ? 'alarm.batchRuleEnabledSuccess' : 'alarm.batchRuleDisabledSuccess', { count: targetRules.length })
          );
          setSelectedRowKeys([]);
          void refetch();
        },
      });
    },
    [modal, refetch, rules, selectedRowKeys, t, updateRule]
  );

  const batchActions = useMemo<BatchAction[]>(
    () => [
      {
        key: 'enable',
        label: t('common.enable'),
        icon: <CheckCircleOutlined />,
        disabled: selectedRules.length === 0 || selectedRules.every((rule) => rule.enabled),
        onClick: () => handleBatchToggle(true),
      },
      {
        key: 'disable',
        label: t('common.disable'),
        icon: <StopOutlined />,
        disabled: selectedRules.length === 0 || selectedRules.every((rule) => !rule.enabled),
        onClick: () => handleBatchToggle(false),
      },
      {
        key: 'delete',
        label: t('common.batchDelete'),
        icon: <DeleteOutlined />,
        danger: true,
        onClick: handleBatchDelete,
      },
    ],
    [handleBatchDelete, handleBatchToggle, selectedRules, t]
  );

  const columns = useMemo(
    (): DataTableColumn<AlarmRule>[] => [
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 140,
        fixed: 'right',
        render: (_val, record) => {
          const moreItems: MenuProps['items'] = [
            { key: 'delete', label: t('common.delete'), icon: <DeleteOutlined />, danger: true, disabled: record.enabled || record.isDefault, onClick: () => handleDelete(record) },
          ];
          return (
            <Space size={4}>
              <Button type="link" size="small" onClick={() => handleView(record)}>
                {t('common.view')}
              </Button>
              <Button type="link" size="small" disabled={record.enabled} onClick={() => handleEdit(record)}>
                {t('common.edit')}
              </Button>
              <Dropdown menu={{ items: moreItems }} trigger={['click']}>
                <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
              </Dropdown>
            </Space>
          );
        },
      },
      {
        key: 'status',
        title: t('alarm.status'),
        dataIndex: 'enabled',
        width: 100,
        render: (_val, record) => (
          <Switch
            checked={record.enabled}
            size="small"
            loading={togglingId === record.id}
            onChange={(checked) => void handleToggle(record, checked)}
          />
        ),
      },
      {
        key: 'ruleName',
        title: t('alarm.ruleName'),
        dataIndex: 'ruleName',
        width: 200,
        ellipsis: true,
        render: (val: unknown, record) => (
          <Space size={4}>
            {record.isDefault && <Tag color="blue">{t('alarm.defaultRule')}</Tag>}
            <span>{String(val ?? '')}</span>
          </Space>
        ),
      },
      {
        key: 'filterType',
        title: t('alarm.ruleFilterType'),
        width: 120,
        render: (_val: unknown, record) => {
          const dimension = getRuleFilterDimension(record, t);
          return <Text type={dimension === '-' ? 'secondary' : undefined}>{dimension}</Text>;
        },
      },
      {
        key: 'ruleType',
        title: t('alarm.ruleType'),
        dataIndex: 'ruleType',
        width: 140,
        render: (val: unknown) => {
          const s = String(val ?? '');
          const config = RULE_TYPE_CONFIG[s];
          return (
            <Tag color={config?.color || 'default'}>
              {config ? t(config.label) : s}
            </Tag>
          );
        },
      },
      {
        key: 'userCode',
        title: t('alarm.operator'),
        dataIndex: 'userCode',
        width: 120,
        render: (val: unknown) => String(val ?? '') || '-',
      },
      {
        key: 'updateTime',
        title: t('alarm.updateTime'),
        dataIndex: 'updateTime',
        width: 160,
        render: (v) => v ? new Date(String(v)).toLocaleString('zh-CN') : '-',
      },
    ],
    [handleToggle, handleEdit, handleView, handleDelete, togglingId, t]
  );

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setFilterParams(values);
    setCurrentPage(1);
    setSelectedRowKeys([]);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
    setSelectedRowKeys([]);
  }, []);

  return (
    <ListPageLayout
      title={t('nav.alarm.rules')}
      subtitle={t('alarm.rulesDesc')}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          {t('common.add')}
        </Button>
      }
    >
      <FilterBar
        filterId="alarm-rules"
        fields={FILTER_FIELDS}
        onSearch={handleSearch}
        onReset={handleReset}
        collapsedRows={1}
      />

      <Card
        size="small"
        bordered
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
        <DataTable<AlarmRule>
          tableId="alarm-rules-table"
          columns={columns}
          dataSource={rules}
          loading={isLoading}
          rowKey="id"
          selectable
          selectedRowKeys={selectedRowKeys}
          onSelectionChange={(keys) => setSelectedRowKeys(keys)}
          total={total}
          pageSize={pageSize}
          currentPage={currentPage}
          batchActions={batchActions}
          onPageChange={(page, size) => {
            setCurrentPage(page);
            setPageSize(size);
            setSelectedRowKeys([]);
          }}
          onRefresh={() => void refetch()}
          defaultDensity="default"
          scroll={{ x: 'max-content', y: 'calc(100vh - 450px)' }}
        />
      </Card>

      <AlarmRuleDrawer
        open={drawerOpen}
        mode={drawerMode}
        rule={currentRule}
        existingNames={rules.map(r => r.ruleName)}
        onClose={handleDrawerClose}
        onSubmit={handleDrawerSubmit}
      />
    </ListPageLayout>
  );
}
