import { useCallback, useMemo, useState } from 'react';
import { App, Button, Space, Switch, Tag, Typography, message } from 'antd';
import {
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useAlarmRules, useCreateAlarmRule, useDeleteAlarmRules, useUpdateAlarmRule, useToggleAlarmRule } from '@/hooks/api/useAlarms';
import { useT } from '@/hooks/useT';
import type { AlarmRule, AlarmRuleCondition, AlarmRuleAction } from '@/types/alarm';
import AlarmRuleDrawer, { type AlarmRuleFormData } from './AlarmRuleDrawer';

const { Text } = Typography;

// 执行动作配置 (后端 action 值)
const RULE_TYPE_CONFIG: Record<string, { label: string; color: string }> = {
  default: { label: 'alarm.ruleType.default', color: 'blue' },
  ignore: { label: 'alarm.ruleType.forbidReport', color: 'red' },
  auto_acknowledge: { label: 'alarm.ruleType.autoConfirm', color: 'green' },
  auto_clear: { label: 'alarm.ruleType.autoClear', color: 'orange' },
};

// 告警源配置
const DEVICE_TYPE_CONFIG: Record<string, string> = {
  'ENB': 'ENB',
  'UPS': 'UPS',
  'CPE': 'CPE',
  'GNB': 'GNB',
  'WCG': 'WCG',
  'GSM': 'GSM',
};

type DrawerMode = 'add' | 'edit' | 'view';

export default function AlarmRules() {
  const t = useT();
  const { modal } = App.useApp();
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [togglingId, setTogglingId] = useState<string | null>(null);

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
  const toggleRule = useToggleAlarmRule();
  const createRule = useCreateAlarmRule();

  const rules: AlarmRule[] = data?.items ?? [];
  const total = data?.total ?? 0;

  // 启用/禁用规则
  const handleToggle = useCallback(
    async (rule: AlarmRule, checked: boolean) => {
      setTogglingId(rule.id);
      try {
        // Use dedicated toggle endpoint when enabling; use updateRule for disabling
        if (checked) {
          await toggleRule.mutateAsync(rule.id);
        } else {
          await updateRule.mutateAsync({ id: rule.id, data: { enabled: false } });
        }
        message.success(t(checked ? 'alarm.ruleEnabled' : 'alarm.ruleDisabled'));
        void refetch();
      } catch (error) {
        message.error(t('alarm.ruleToggleFailed'));
        console.error('Toggle rule failed:', error);
      } finally {
        setTogglingId(null);
      }
    },
    [toggleRule, updateRule, refetch, t]
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
      conditions.push({ field: 'alarm_code', operator: 'contains', value: formData.selectedAlarms });
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

  const columns = useMemo(
    (): DataTableColumn<AlarmRule>[] => [
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 160,
        fixed: 'left',
        render: (_val, record) => (
          <Space size={4}>
            <Button
              type="link"
              size="small"
              icon={<EyeOutlined />}
              onClick={() => handleView(record)}
            >
              {t('common.view')}
            </Button>
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              disabled={record.enabled}
              onClick={() => handleEdit(record)}
            >
              {t('common.edit')}
            </Button>
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              disabled={record.enabled || record.isDefault}
              onClick={() => handleDelete(record)}
            >
              {t('common.delete')}
            </Button>
          </Space>
        ),
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
        render: (val: string, record) => (
          <Space size={4}>
            {record.isDefault && <Tag color="blue">{t('alarm.defaultRule')}</Tag>}
            <span>{val}</span>
          </Space>
        ),
      },
      {
        key: 'deviceType',
        title: t('alarm.deviceType'),
        dataIndex: 'deviceType',
        width: 120,
        render: (val: string, record) => {
          if (record.isDefault) {
            return <Text type="secondary">ALL</Text>;
          }
          return DEVICE_TYPE_CONFIG[val] || val || '-';
        },
      },
      {
        key: 'ruleType',
        title: t('alarm.ruleType'),
        dataIndex: 'ruleType',
        width: 140,
        render: (val: string) => {
          const config = RULE_TYPE_CONFIG[val];
          return (
            <Tag color={config?.color || 'default'}>
              {config ? t(config.label) : val}
            </Tag>
          );
        },
      },
      {
        key: 'userCode',
        title: t('alarm.operator'),
        dataIndex: 'userCode',
        width: 120,
        render: (val: string) => val || '-',
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
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
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

      <DataTable<AlarmRule>
        tableId="alarm-rules-table"
        columns={columns}
        dataSource={rules}
        loading={isLoading}
        rowKey="id"
        total={total}
        pageSize={pageSize}
        currentPage={currentPage}
        onPageChange={(page, size) => {
          setCurrentPage(page);
          setPageSize(size);
        }}
        onRefresh={() => void refetch()}
        defaultDensity="default"
      />

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
