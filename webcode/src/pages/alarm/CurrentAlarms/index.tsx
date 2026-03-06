import React, { useCallback, useMemo, useState } from 'react';
import { Badge, Button, Modal, Space, Tag, Typography, message } from 'antd';
import {
  BellOutlined,
  CheckOutlined,
  ClearOutlined,
  SoundOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useCurrentAlarms, useAcknowledgeAlarms, useClearAlarms } from '@/hooks/api/useAlarms';
import { useT } from '@/hooks/useT';
import type { Alarm } from '@/types/alarm';
import type { AlarmFilter } from '@/types/alarm';

const { Text } = Typography;

const SEVERITY_TAG_COLOR: Record<string, string> = {
  critical: 'red',
  major: 'orange',
  minor: 'gold',
  warning: 'blue',
};

function formatDuration(ms: number): string {
  if (ms < 60000) return `${Math.floor(ms / 1000)}s`;
  if (ms < 3600000) return `${Math.floor(ms / 60000)}m`;
  if (ms < 86400000) return `${Math.floor(ms / 3600000)}h`;
  return `${Math.floor(ms / 86400000)}d`;
}

function computeDuration(alarmTime: string): string {
  const now = Date.now();
  const then = new Date(alarmTime).getTime();
  const diff = now - then;
  if (diff < 0) return '-';
  return formatDuration(diff);
}

export default function CurrentAlarms() {
  const t = useT();
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [filterParams, setFilterParams] = useState<AlarmFilter>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  const SEVERITY_LABEL: Record<string, string> = useMemo(() => ({
    critical: t('alarm.severity.critical'),
    major: t('alarm.severity.major'),
    minor: t('alarm.severity.minor'),
    warning: t('alarm.severity.warning'),
  }), [t]);

  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    {
      name: 'severity',
      label: t('alarm.severity'),
      type: 'multi-select',
      options: [
        { label: t('alarm.severity.critical'), value: 'critical' },
        { label: t('alarm.severity.major'), value: 'major' },
        { label: t('alarm.severity.minor'), value: 'minor' },
        { label: t('alarm.severity.warning'), value: 'warning' },
      ],
    },
    {
      name: 'ackStatus',
      label: t('alarm.ackStatus'),
      type: 'select',
      options: [
        { label: t('alarm.ackStatus.unacknowledged'), value: 'unacknowledged' },
        { label: t('alarm.ackStatus.acknowledged'), value: 'acknowledged' },
      ],
    },
    { name: 'deviceSn', label: t('alarm.deviceSn'), type: 'input' },
    { name: 'alarmCode', label: t('alarm.code'), type: 'input' },
    { name: 'alarmName', label: t('alarm.name'), type: 'input' },
    { name: 'timeRange', label: t('table.time'), type: 'date-range' },
  ], [t]);

  const queryParams = useMemo(
    () => ({ ...filterParams, page: currentPage, pageSize }),
    [filterParams, currentPage, pageSize]
  );

  const { data, isLoading, refetch } = useCurrentAlarms(queryParams);
  const acknowledgeAlarms = useAcknowledgeAlarms();
  const clearAlarms = useClearAlarms();

  const alarms: Alarm[] = data?.items ?? [];
  const total = data?.total ?? 0;

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setFilterParams({
      severity: values.severity as AlarmFilter['severity'],
      ackStatus: values.ackStatus as AlarmFilter['ackStatus'],
      deviceSn: values.deviceSn as string | undefined,
      alarmCode: values.alarmCode as string | undefined,
      alarmName: values.alarmName as string | undefined,
    });
    setCurrentPage(1);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
  }, []);

  const handleAcknowledge = useCallback(
    (ids: string[]) => {
      Modal.confirm({
        title: t('alarm.acknowledge'),
        content: t('common.ackConfirmMsg', { count: ids.length }),
        okText: t('common.confirm'),
        icon: <CheckOutlined style={{ color: 'var(--color-primary-600)' }} />,
        onOk: async () => {
          await acknowledgeAlarms.mutateAsync({ ids });
          setSelectedRowKeys([]);
          void message.success(t('common.ackSuccess'));
        },
      });
    },
    [acknowledgeAlarms, t]
  );

  const handleClear = useCallback(
    (ids: string[]) => {
      Modal.confirm({
        title: t('alarm.clear'),
        content: t('common.clearConfirmMsg', { count: ids.length }),
        okText: t('alarm.clear'),
        okType: 'danger',
        icon: <ClearOutlined />,
        onOk: async () => {
          await clearAlarms.mutateAsync(ids);
          setSelectedRowKeys([]);
          void message.success(t('common.clearSuccess'));
        },
      });
    },
    [clearAlarms, t]
  );

  const alarmRowStyle = useCallback(
    (record: Alarm): 'critical' | 'major' | 'minor' | 'warning' | null => {
      return record.severity as 'critical' | 'major' | 'minor' | 'warning';
    },
    []
  );

  const columns = useMemo(
    (): DataTableColumn<Alarm>[] => [
      {
        key: 'severity',
        title: t('alarm.severity'),
        dataIndex: 'severity',
        width: 80,
        render: (_val, record) => (
          <Tag color={SEVERITY_TAG_COLOR[record.severity] ?? 'default'}>
            {SEVERITY_LABEL[record.severity] ?? record.severity}
          </Tag>
        ),
      },
      {
        key: 'alarmCode',
        title: t('alarm.code'),
        dataIndex: 'alarmCode',
        width: 100,
        mono: true,
        render: (v) => <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(v)}</Text>,
      },
      {
        key: 'alarmName',
        title: t('alarm.name'),
        dataIndex: 'alarmName',
        width: 160,
        ellipsis: true,
      },
      {
        key: 'deviceSn',
        title: t('alarm.deviceSn'),
        dataIndex: 'deviceSn',
        width: 160,
        mono: true,
        copyable: true,
        render: (v) => <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(v)}</Text>,
      },
      { key: 'deviceName', title: t('alarm.deviceName'), dataIndex: 'deviceName', width: 140, ellipsis: true },
      {
        key: 'alarmContent',
        title: t('alarm.content'),
        dataIndex: 'alarmContent',
        width: 200,
        ellipsis: true,
        render: (v) => (
          <Text type="secondary" ellipsis title={String(v)}>
            {String(v)}
          </Text>
        ),
      },
      {
        key: 'alarmTime',
        title: t('alarm.occurTime'),
        dataIndex: 'alarmTime',
        width: 160,
        render: (v) => new Date(String(v)).toLocaleString('zh-CN'),
      },
      {
        key: 'duration',
        title: t('alarm.duration'),
        dataIndex: 'alarmTime',
        width: 100,
        render: (v) => computeDuration(String(v)),
      },
      {
        key: 'ackStatus',
        title: t('alarm.ackStatus'),
        dataIndex: 'ackStatus',
        width: 100,
        render: (_val, record) => (
          <Tag color={record.ackStatus === 'acknowledged' ? 'success' : 'warning'}>
            {record.ackStatus === 'acknowledged' ? t('alarm.ackStatus.acknowledged') : t('alarm.ackStatus.unacknowledged')}
          </Tag>
        ),
      },
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 120,
        fixed: 'right',
        render: (_val, record) => (
          <Space size={4}>
            {record.ackStatus === 'unacknowledged' && (
              <Button
                type="link"
                size="small"
                icon={<CheckOutlined />}
                onClick={() => handleAcknowledge([record.id])}
              >
                {t('alarm.acknowledge')}
              </Button>
            )}
            <Button
              type="link"
              size="small"
              danger
              icon={<ClearOutlined />}
              onClick={() => handleClear([record.id])}
            >
              {t('alarm.clear')}
            </Button>
          </Space>
        ),
      },
    ],
    [handleAcknowledge, handleClear, t, SEVERITY_LABEL]
  );

  const batchActions = useMemo(
    (): BatchAction[] => [
      {
        key: 'batch-ack',
        label: t('common.batchAck'),
        icon: <CheckOutlined />,
        onClick: (keys) => handleAcknowledge(keys as string[]),
      },
      {
        key: 'batch-clear',
        label: t('common.batchClear'),
        icon: <ClearOutlined />,
        danger: true,
        onClick: (keys) => handleClear(keys as string[]),
      },
    ],
    [handleAcknowledge, handleClear, t]
  );

  // Count unacknowledged
  const unackCount = alarms.filter((a) => a.ackStatus === 'unacknowledged').length;

  return (
    <ListPageLayout
      title={t('nav.alarm.current')}
      extra={
        <Space>
          {/* Real-time connection indicator */}
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6, fontSize: 13 }}>
            <Badge status="success" />
            <span style={{ color: '#52C41A' }}>{t('common.realTimeConn')}</span>
          </span>
          {unackCount > 0 && (
            <Tag color="red" icon={<BellOutlined />}>
              {unackCount} {t('common.unacked')}
            </Tag>
          )}
        </Space>
      }
    >
      <FilterBar
        filterId="current-alarms"
        fields={FILTER_FIELDS}
        onSearch={handleSearch}
        onReset={handleReset}
        collapsedRows={1}
      />

      <DataTable<Alarm>
        tableId="current-alarms-table"
        columns={columns}
        dataSource={alarms}
        loading={isLoading}
        rowKey="id"
        selectable
        selectedRowKeys={selectedRowKeys}
        onSelectionChange={(keys) => setSelectedRowKeys(keys)}
        total={total}
        pageSize={pageSize}
        currentPage={currentPage}
        onPageChange={(page, size) => {
          setCurrentPage(page);
          setPageSize(size);
        }}
        batchActions={batchActions}
        onRefresh={() => void refetch()}
        alarmRowStyle={alarmRowStyle as (record: Alarm) => 'critical' | 'major' | 'minor' | 'warning' | null}
        defaultDensity="compact"
      />
    </ListPageLayout>
  );
}
