import { useState, useMemo } from 'react';
import { Button, Tag, Typography, App } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import {
  useStationLogList,
  useDownloadStationLog,
} from '@core/hooks/api/useStationLog';
import type { StationLogFile } from '@core/services/api/stationLogApi';

type LogType = StationLogFile['logType'];

const PAGE_SIZE = 20;

function formatBytes(bytes: number): string {
  if (!bytes) return '-';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export default function DeviceLog() {
  const t = useT();
  const { message } = App.useApp();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(PAGE_SIZE);

  const deviceId = typeof filters.deviceId === 'string' ? filters.deviceId.trim() : '';
  const logType =
    filters.logType === 'running' || filters.logType === 'fault'
      ? (filters.logType as LogType)
      : undefined;

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(deviceId ? { deviceId } : {}),
      ...(logType ? { logType } : {}),
    }),
    [page, pageSize, deviceId, logType]
  );

  const { data, isLoading } = useStationLogList(params);
  const download = useDownloadStationLog();

  const rows = data?.items ?? [];
  const total = data?.total ?? 0;

  const TYPE_MAP: Record<LogType, { color: string; text: string }> = useMemo(
    () => ({
      running: { color: 'blue', text: t('log.runningLog') },
      fault: { color: 'error', text: t('log.faultLog') },
    }),
    [t]
  );

  const filterFields: FilterField[] = useMemo(
    () => [
      { name: 'deviceId', label: t('log.deviceCode'), type: 'input', placeholder: t('log.inputDeviceCode') },
      {
        name: 'logType',
        label: t('log.type'),
        type: 'select',
        options: [
          { label: t('log.runningLog'), value: 'running' },
          { label: t('log.faultLog'), value: 'fault' },
        ],
      },
    ],
    [t]
  );

  const handleDownload = (record: StationLogFile) => {
    download.mutate(record.id, {
      onError: () => void message.error(t('common.downloadFailed')),
    });
  };

  const columns: DataTableColumn<StationLogFile & Record<string, unknown>>[] = useMemo(
    () => [
      {
        key: 'deviceSn',
        title: t('device.sn'),
        dataIndex: 'deviceSn',
        width: 160,
        mono: true,
        render: (val) => <Typography.Text style={{ fontFamily: 'monospace' }}>{(val as string) || '-'}</Typography.Text>,
      },
      {
        key: 'logType',
        title: t('log.type'),
        dataIndex: 'logType',
        width: 110,
        render: (val) => {
          const cfg = TYPE_MAP[val as LogType] ?? TYPE_MAP.running;
          return <Tag color={cfg.color}>{cfg.text}</Tag>;
        },
      },
      {
        key: 'fileName',
        title: t('log.logFileName'),
        dataIndex: 'fileName',
        width: 320,
        ellipsis: true,
        render: (val, record) => (val as string) || record.objectPath || '-',
      },
      {
        key: 'fileSize',
        title: t('log.logFileSize'),
        dataIndex: 'fileSize',
        width: 110,
        render: (val) => formatBytes(val as number),
      },
      {
        key: 'faultReason',
        title: t('log.faultReason'),
        dataIndex: 'faultReason',
        width: 200,
        ellipsis: true,
        render: (val) => (val as string) || '-',
      },
      {
        key: 'collectedAt',
        title: t('log.collectedAt'),
        dataIndex: 'collectedAt',
        width: 180,
        render: (val) => (val as string) || '-',
      },
      {
        key: 'action',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 100,
        fixed: 'right',
        render: (_, record) => (
          <Button
            type="link"
            size="small"
            icon={<DownloadOutlined />}
            loading={download.isPending}
            onClick={() => handleDownload(record)}
          >
            {t('common.download')}
          </Button>
        ),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [t, TYPE_MAP, download.isPending]
  );

  return (
    <ListPageLayout title={t('log.deviceLog')}>
      <FilterBar
        filterId="device-log-filter"
        fields={filterFields}
        onSearch={(vals) => {
          setFilters(vals);
          setPage(1);
        }}
        onReset={() => {
          setFilters({});
          setPage(1);
        }}
      />
      <DataTable<StationLogFile & Record<string, unknown>>
        tableId="device-log-list"
        columns={columns}
        dataSource={rows as (StationLogFile & Record<string, unknown>)[]}
        loading={isLoading}
        rowKey="id"
        total={total}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => {
          setPage(p);
          setPageSize(s);
        }}
        scroll={{ x: 'max-content', y: 'calc(100vh - 320px)' }}
        showRowNumber
        rowNumberTitle={t('common.rowNumber')}
      />
    </ListPageLayout>
  );
}
