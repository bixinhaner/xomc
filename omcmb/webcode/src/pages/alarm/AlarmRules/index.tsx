import React, { useCallback, useMemo, useState } from 'react';
import { App, Button, Space, Switch, Tag, Typography } from 'antd';
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
import { useAlarmRules, useDeleteAlarmRules, useUpdateAlarmRule } from '@/hooks/api/useAlarms';
import { useT } from '@/hooks/useT';
import type { AlarmRule } from '@/types/alarm';

const { Text } = Typography;

// 执行动作配置
const RULE_TYPE_CONFIG: Record<string, { label: string; color: string }> = {
  '0': { label: 'alarm.ruleType.forbidReport', color: 'red' },
  '1': { label: 'alarm.ruleType.noStoreNoShow', color: 'orange' },
  '2': { label: 'alarm.ruleType.storeNoShow', color: 'gold' },
  '3': { label: 'alarm.ruleType.autoConfirm', color: 'green' },
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

export default function AlarmRules() {
  const t = useT();
  const { modal } = App.useApp();
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});

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

  const rules: AlarmRule[] = data?.items ?? [];
  const total = data?.total ?? 0;

  // 启用/禁用规则
  const handleToggle = useCallback(
    async (rule: AlarmRule, checked: boolean) => {
      try {
        await updateRule.mutateAsync({ id: rule.id, data: { enabled: checked } as Partial<AlarmRule> });
        void refetch();
      } catch {
        // error handled by hook
      }
    },
    [updateRule, refetch]
  );

  // 编辑规则
  const handleEdit = useCallback((rule: AlarmRule) => {
    // TODO: 打开编辑抽屉/页面
    console.log('Edit rule:', rule);
  }, []);

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
        width: 120,
        fixed: 'left',
        render: (_val, record) => (
          <Space size={4}>
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
            loading={updateRule.isPending}
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
    [handleToggle, handleEdit, handleDelete, updateRule.isPending, t]
  );

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setFilterParams(values);
    setCurrentPage(1);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
  }, []);

  const handleAdd = useCallback(() => {
    // TODO: 打开添加抽屉/页面
    console.log('Add new rule');
  }, []);

  return (
    <ListPageLayout
      title={t('nav.alarm.rules')}
      subTitle={t('alarm.rulesDesc')}
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
        defaultDensity="compact"
      />
    </ListPageLayout>
  );
}
