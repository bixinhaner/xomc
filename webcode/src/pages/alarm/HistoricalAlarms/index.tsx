import React, { useCallback, useMemo, useState } from 'react';
import { Button, Space, Tag, Typography } from 'antd';
import { DownloadOutlined, EyeOutlined } from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useHistoricalAlarms } from '@/hooks/api/useAlarms';
import { useT } from '@/hooks/useT';
import type { Alarm } from '@/types/alarm';
import type { AlarmFilter } from '@/types/alarm';

const { Text } = Typography;

const SEVERITY_TAG_COLOR: Record<string, string> = {
  critical: 'red', major: 'orange', minor: 'gold', warning: 'blue',
};

function formatDuration(ms: number): string {
  if (ms < 60000) return `${Math.floor(ms / 1000)}s`;
  if (ms < 3600000) return `${Math.floor(ms / 60000)}m`;
  if (ms < 86400000) return `${Math.floor(ms / 3600000)}h`;
  return `${Math.floor(ms / 86400000)}d`;
}

export default function HistoricalAlarms() {
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
    { name: 'timeRange', label: t('alarm.occurTime'), type: 'date-range' },
  ], [t]);

  const queryParams = useMemo(
    () => ({ ...filterParams, page: currentPage, pageSize }),
    [filterParams, currentPage, pageSize]
  );

  const { data, isLoading, refetch } = useHistoricalAlarms(queryParams);
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
      { key: 'alarmName', title: t('alarm.name'), dataIndex: 'alarmName', width: 160, ellipsis: true },
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
          <Text type="secondary" title={String(v)}>
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
        key: 'clearTime',
        title: t('alarm.clearTime'),
        dataIndex: 'clearTime',
        width: 160,
        render: (v) => v ? new Date(String(v)).toLocaleString('zh-CN') : <Text type="secondary">-</Text>,
      },
      {
        key: 'duration',
        title: t('alarm.duration'),
        dataIndex: 'alarmTime',
        width: 100,
        render: (_val, record) => {
          if (!record.clearTime) return '-';
          const diff = new Date(record.clearTime).getTime() - new Date(record.alarmTime).getTime();
          return diff > 0 ? formatDuration(diff) : '-';
        },
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
        key: 'ackUser',
        title: t('alarm.ackUser'),
        dataIndex: 'ackUser',
        width: 90,
        render: (v) => v ? String(v) : <Text type="secondary">-</Text>,
      },
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 80,
        fixed: 'right',
        render: (_val, record) => (
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => {
              // In a real app this would open AlarmDetail drawer
              void record;
            }}
          >
            {t('common.detail')}
          </Button>
        ),
      },
    ],
    [t, SEVERITY_LABEL]
  );

  return (
    <ListPageLayout
      title={t('nav.alarm.history')}
      extra={
        <Button icon={<DownloadOutlined />}>
          {t('common.export')}
        </Button>
      }
    >
      <FilterBar
        filterId="historical-alarms"
        fields={FILTER_FIELDS}
        onSearch={handleSearch}
        onReset={handleReset}
        collapsedRows={1}
      />

      <DataTable<Alarm>
        tableId="historical-alarms-table"
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
        onRefresh={() => void refetch()}
        defaultDensity="compact"
      />
    </ListPageLayout>
  );
}
