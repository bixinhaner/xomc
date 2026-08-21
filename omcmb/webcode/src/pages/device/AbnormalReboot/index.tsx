import { useEffect, useState, useMemo } from 'react';
import {
  Button,
  Space,
  Tag,
  Typography,
  Modal,
  Card,
  Table,
  Drawer,
  Descriptions,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { ExportOutlined, BarChartOutlined, ReloadOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import {
  useRebootRecordList,
  useRebootRecordStatByDevice,
} from '@core/hooks/api/useRebootRecord';
import { useSystemLicense } from '@core/hooks/api/useSystemLicense';
import { rebootRecordApi } from '@core/services/api/rebootRecordApi';
import { isDeviceStandardValueVisibleByLicense } from '@core/utils/licenseFeatures';
import type {
  RebootRecord,
  RebootRecordListParams,
  RebootRecordStatParams,
  RebootType,
  DeviceRebootStat,
} from '@core/services/api/rebootRecordApi';
import { getRebootRecordDeviceTypeOptions } from './filterOptions';

// 「启动记录」页面（统一重启记录单列表）
//
// 普通重启(event_logs) 与 异常重启(station_fault_logs) 两张互斥表由后端
// /reboot-records UNION 合成一份列表：「类型」列标记正常/异常，可按重启类型过滤，
// 右上角「统计」按设备聚合（总次数/异常次数，跟随筛选），可导出。

const REBOOT_RECORD_AUTO_REFRESH_MS = 30_000;

const exportTimestamp = (): string => {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, '0');
  return (
    `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}` +
    `${pad(d.getHours())}${pad(d.getMinutes())}${pad(d.getSeconds())}`
  );
};

// 重启原因 / 详因一律按设备上报原文展示，不做翻译（设备报什么就显示什么）。
const showRaw = (value: string | undefined | null): string => (value ?? '').trim() || '-';

const formatRuntime = (seconds: number | null | undefined): string => {
  if (!seconds || seconds <= 0) return '-';
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  return d > 0 ? `${d}d ${h}h ${m}m` : h > 0 ? `${h}h ${m}m` : `${m}m`;
};

const fmtTime = (s: string | undefined): string => s?.replace('T', ' ').slice(0, 19) ?? '-';

function escapeCsvCell(value: unknown): string {
  const normalized = value == null ? '' : String(value);
  return `"${normalized.replace(/"/g, '""')}"`;
}

function downloadCsv(rows: Record<string, unknown>[], filename: string): void {
  const headers = Object.keys(rows[0] ?? {});
  const content = [
    headers.map(escapeCsvCell).join(','),
    ...rows.map((row) => headers.map((header) => escapeCsvCell(row[header])).join(',')),
  ].join('\n');
  const blob = new Blob(['\ufeff' + content], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}

export default function AbnormalReboot() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({ rebootType: 'all' });
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [exporting, setExporting] = useState(false);

  const [detailVisible, setDetailVisible] = useState(false);
  const [selected, setSelected] = useState<RebootRecord | null>(null);

  const [statVisible, setStatVisible] = useState(false);
  const { data: systemLicense, isLoading: systemLicenseLoading } = useSystemLicense();

  useEffect(() => {
    if (
      systemLicenseLoading
      || typeof filters.deviceType !== 'string'
      || isDeviceStandardValueVisibleByLicense(filters.deviceType, systemLicense, false)
    ) {
      return;
    }
    setFilters((prev) => {
      const next = { ...prev };
      delete next.deviceType;
      return next;
    });
    setPage(1);
  }, [filters.deviceType, systemLicense, systemLicenseLoading]);

  // ----- 主列表查询参数（跟随筛选） -----
  const queryParams = useMemo<RebootRecordListParams>(() => {
    const p: RebootRecordListParams = { page, pageSize };
    if (filters.keyword && typeof filters.keyword === 'string') p.deviceSn = filters.keyword;
    if (filters.deviceType && typeof filters.deviceType === 'string') {
      p.deviceType = filters.deviceType;
    }
    if (filters.rebootType && typeof filters.rebootType === 'string') {
      p.rebootType = filters.rebootType as RebootType;
    }
    if (Array.isArray(filters.timeRange) && filters.timeRange.length === 2) {
      const toISO = (v: unknown): string | undefined => {
        if (!v) return undefined;
        if (typeof v === 'string') return v;
        const obj = v as { toISOString?: () => string };
        return typeof obj.toISOString === 'function' ? obj.toISOString() : undefined;
      };
      const s = toISO(filters.timeRange[0]);
      const e = toISO(filters.timeRange[1]);
      if (s) p.startTime = s;
      if (e) p.endTime = e;
    }
    return p;
  }, [filters, page, pageSize]);

  const {
    data,
    isLoading,
    isFetching,
    refetch: refetchRebootRecords,
  } = useRebootRecordList(queryParams, { refetchInterval: REBOOT_RECORD_AUTO_REFRESH_MS });
  const dataSource = data?.items ?? [];
  const total = data?.total ?? 0;

  // ----- 统计参数（同筛选去掉分页） -----
  const statParams = useMemo<RebootRecordStatParams>(() => {
    const { deviceSn, deviceType, rebootType, startTime, endTime } = queryParams;
    return { deviceSn, deviceType, rebootType, startTime, endTime };
  }, [queryParams]);

  const { data: statData, isLoading: statLoading } = useRebootRecordStatByDevice(
    statParams,
    statVisible,
  );
  const statRows = statData?.items ?? [];

  const filterFields: FilterField[] = useMemo(
    () => [
      {
        name: 'keyword',
        label: t('log.exception.column.deviceCode'),
        type: 'input',
        placeholder: t('log.exception.devCodeNameIp'),
      },
      {
        name: 'rebootType',
        label: t('page.rebootRecords.filter.rebootType'),
        type: 'select',
        defaultValue: 'all',
        options: [
          { label: t('page.rebootRecords.type.all'), value: 'all' },
          { label: t('page.rebootRecords.type.normal'), value: 'normal' },
          { label: t('page.rebootRecords.type.abnormal'), value: 'abnormal' },
        ],
      },
      {
        name: 'deviceType',
        label: t('log.exception.column.deviceType'),
        type: 'select',
        options: getRebootRecordDeviceTypeOptions(systemLicense, systemLicenseLoading),
      },
      { name: 'timeRange', label: t('log.timeRange'), type: 'date-range', span: 2 },
    ],
    [systemLicense, systemLicenseLoading, t],
  );

  const columns: DataTableColumn<RebootRecord>[] = useMemo(
    () => [
      {
        key: 'deviceSn',
        title: t('log.exception.column.deviceCode'),
        dataIndex: 'deviceSn',
        width: 180,
        render: (val: unknown) => (
          <Typography.Text style={{ fontFamily: 'monospace' }}>{val as string}</Typography.Text>
        ),
      },
      {
        key: 'deviceName',
        title: t('log.exception.column.deviceName'),
        dataIndex: 'deviceName',
        width: 160,
        ellipsis: true,
        render: (val: unknown) => (val as string) || '-',
      },
      {
        key: 'deviceType',
        title: t('log.exception.column.deviceType'),
        dataIndex: 'deviceType',
        width: 80,
        render: (val: unknown) => (val as string) || '-',
      },
      {
        key: 'operateIp',
        title: t('log.exception.column.baseIp'),
        dataIndex: 'operateIp',
        width: 140,
        render: (val: unknown) =>
          val ? (
            <Typography.Text style={{ fontFamily: 'monospace' }}>{val as string}</Typography.Text>
          ) : (
            '-'
          ),
      },
      {
        key: 'softwareVersion',
        title: t('log.exception.column.softwareVersion'),
        dataIndex: 'softwareVersion',
        width: 120,
        ellipsis: true,
        render: (val: unknown) => (val as string) || '-',
      },
      {
        key: 'isAbnormal',
        title: t('page.rebootRecords.column.type'),
        dataIndex: 'isAbnormal',
        width: 110,
        render: (val: unknown) =>
          val ? (
            <Tag color="red">{t('page.rebootRecords.type.abnormal')}</Tag>
          ) : (
            <Tag color="blue">{t('page.rebootRecords.type.normal')}</Tag>
          ),
      },
      {
        key: 'reason',
        title: t('page.rebootRecords.column.reason'),
        dataIndex: 'reason',
        ellipsis: true,
        // 仅异常重启有设备上报的原因；正常重启不带 HaltReason，留空。
        render: (val: unknown, record: RebootRecord) =>
          record.isAbnormal ? showRaw(val as string) : '-',
      },
      {
        key: 'runtimeBeforeReboot',
        title: t('log.exception.column.runtime'),
        dataIndex: 'runtimeBeforeReboot',
        width: 120,
        render: (val: unknown) => formatRuntime(val as number),
      },
      {
        key: 'rebootTime',
        title: t('log.exception.column.time'),
        dataIndex: 'rebootTime',
        width: 180,
        render: (val: unknown) => fmtTime(val as string),
      },
    ],
    [t],
  );

  const handleExport = async () => {
    if (exporting) return;
    setExporting(true);
    try {
      const resp = await rebootRecordApi.list({ ...queryParams, page: 1, pageSize: 10000 });
      if (!resp.items || resp.items.length === 0) {
        void message.warning(t('log.exception.exportEmpty'));
        return;
      }
      const rows = resp.items.map((r) => ({
        [t('log.exception.column.deviceCode')]: r.deviceSn,
        [t('log.exception.column.deviceName')]: r.deviceName || '-',
        [t('log.exception.column.deviceType')]: r.deviceType || '-',
        [t('log.exception.column.baseIp')]: r.operateIp || '-',
        [t('log.exception.column.softwareVersion')]: r.softwareVersion || '-',
        [t('page.rebootRecords.column.type')]: r.isAbnormal
          ? t('page.rebootRecords.type.abnormal')
          : t('page.rebootRecords.type.normal'),
        [t('page.rebootRecords.column.reason')]: r.isAbnormal ? showRaw(r.reason) : '-',
        [t('log.exception.column.runtime')]: formatRuntime(r.runtimeBeforeReboot),
        [t('log.exception.column.time')]: fmtTime(r.rebootTime),
      }));
      downloadCsv(rows, `${t('page.rebootRecords.title')}_${exportTimestamp()}.csv`);
      void message.success(t('log.exception.exportSuccess', { count: resp.items.length }));
    } catch (e) {
      console.error('export reboot records failed', e);
      void message.error(t('log.exception.exportFailed'));
    } finally {
      setExporting(false);
    }
  };

  // ----- 统计弹窗：列 + 导出 -----
  const statColumns: ColumnsType<DeviceRebootStat> = useMemo(
    () => [
      {
        title: t('log.exception.column.deviceCode'),
        dataIndex: 'deviceSn',
        key: 'deviceSn',
        width: 190,
        render: (val: string) => (
          <Typography.Text style={{ fontFamily: 'monospace' }}>{val}</Typography.Text>
        ),
      },
      {
        title: t('log.event.stat.deviceName'),
        dataIndex: 'deviceName',
        key: 'deviceName',
        render: (val: string) => val || '-',
      },
      {
        title: t('log.exception.column.deviceType'),
        dataIndex: 'deviceType',
        key: 'deviceType',
        width: 140,
        render: (val: string) => val || '-',
      },
      {
        title: t('page.rebootRecords.stat.totalCount'),
        dataIndex: 'totalCount',
        key: 'totalCount',
        width: 120,
        defaultSortOrder: 'descend',
        sorter: (a, b) => a.totalCount - b.totalCount,
        render: (val: number) => <Tag color="blue">{val}</Tag>,
      },
      {
        title: t('page.rebootRecords.stat.abnormalCount'),
        dataIndex: 'abnormalCount',
        key: 'abnormalCount',
        width: 110,
        sorter: (a, b) => a.abnormalCount - b.abnormalCount,
        render: (val: number) => (val > 0 ? <Tag color="red">{val}</Tag> : <span>0</span>),
      },
      {
        title: t('log.event.stat.latestAt'),
        dataIndex: 'latestAt',
        key: 'latestAt',
        width: 180,
        render: (val: string) => fmtTime(val),
      },
    ],
    [t],
  );

  const handleExportStat = () => {
    if (statRows.length === 0) {
      void message.warning(t('log.exception.exportEmpty'));
      return;
    }
    const rows = statRows.map((r) => ({
      [t('log.exception.column.deviceCode')]: r.deviceSn,
      [t('log.event.stat.deviceName')]: r.deviceName || '-',
      [t('log.exception.column.deviceType')]: r.deviceType || '-',
      [t('page.rebootRecords.stat.totalCount')]: r.totalCount,
      [t('page.rebootRecords.stat.abnormalCount')]: r.abnormalCount,
      [t('log.event.stat.latestAt')]: fmtTime(r.latestAt),
    }));
    downloadCsv(rows, `${t('log.event.statModal.title')}_${exportTimestamp()}.csv`);
    void message.success(t('log.exception.exportSuccess', { count: statRows.length }));
  };

  return (
    <ListPageLayout title={t('page.rebootRecords.title')}>
      <FilterBar
        filterId="reboot-record-filter"
        fields={filterFields}
        initialValues={{ rebootType: 'all' }}
        onSearch={(v) => {
          setFilters(v);
          setPage(1);
        }}
        onReset={() => {
          setFilters({ rebootType: 'all' });
          setPage(1);
        }}
      />

      <Space style={{ marginBottom: 8, display: 'flex', justifyContent: 'flex-end' }}>
        <Button
          icon={<ReloadOutlined />}
          onClick={() => {
            void refetchRebootRecords();
          }}
          loading={isFetching && !isLoading}
        >
          {t('common.refresh')}
        </Button>
        <Button icon={<BarChartOutlined />} onClick={() => setStatVisible(true)}>
          {t('log.event.statistics')}
        </Button>
        <Button type="primary" icon={<ExportOutlined />} onClick={handleExport} loading={exporting}>
          {t('log.export')}
        </Button>
      </Space>

      <Card
        size="small"
        variant="outlined"
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{
          body: {
            padding: 0,
            display: 'flex',
            flexDirection: 'column',
            flex: 1,
            overflow: 'hidden',
          },
        }}
      >
        <DataTable<RebootRecord>
          tableId="reboot-record-list"
          columns={columns}
          dataSource={dataSource}
          rowKey={(r) => `${r.source}-${r.id}`}
          total={total}
          currentPage={page}
          pageSize={pageSize}
          onPageChange={(p, s) => {
            setPage(p);
            setPageSize(s);
          }}
          loading={isLoading}
          hideToolbar
          scroll={{ x: 'max-content', y: 'calc(100vh - 460px)' }}
          showRowNumber
          rowNumberTitle={t('log.exception.column.seq')}
          onRow={(record) => ({
            onClick: () => {
              setSelected(record);
              setDetailVisible(true);
            },
            style: { cursor: 'pointer' },
          })}
        />
      </Card>

      {/* 行详情抽屉：异常行展示 HaltReason 完整信息，普通行展示基础信息 */}
      <Drawer
        title={t('log.exception.detail.title')}
        placement="right"
        size={640}
        open={detailVisible}
        onClose={() => setDetailVisible(false)}
      >
        {selected && (
          <div>
            <Card
              size="small"
              title={t('log.exception.detail.devInfo')}
              style={{ marginBottom: 16 }}
            >
              <Descriptions column={2} size="small">
                <Descriptions.Item label={t('log.exception.detail.devCode')}>
                  <Typography.Text style={{ fontFamily: 'monospace' }}>
                    {selected.deviceSn}
                  </Typography.Text>
                </Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.devName')}>
                  {selected.deviceName || '-'}
                </Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.devType')}>
                  {selected.deviceType || '-'}
                </Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.baseIp')}>
                  {selected.operateIp ? (
                    <Typography.Text style={{ fontFamily: 'monospace' }}>
                      {selected.operateIp}
                    </Typography.Text>
                  ) : (
                    '-'
                  )}
                </Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.softwareVersion')}>
                  {selected.softwareVersion || '-'}
                </Descriptions.Item>
              </Descriptions>
            </Card>

            <Card size="small" title={t('log.exception.detail.exceptionInfo')}>
              <Descriptions column={2} size="small">
                <Descriptions.Item label={t('page.rebootRecords.column.type')}>
                  {selected.isAbnormal ? (
                    <Tag color="red">{t('page.rebootRecords.type.abnormal')}</Tag>
                  ) : (
                    <Tag color="blue">{t('page.rebootRecords.type.normal')}</Tag>
                  )}
                </Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.occurTime')}>
                  {fmtTime(selected.rebootTime)}
                </Descriptions.Item>
                <Descriptions.Item label={t('page.rebootRecords.column.reason')} span={2}>
                  {selected.isAbnormal ? showRaw(selected.reason) : '-'}
                </Descriptions.Item>
                {selected.isAbnormal && (
                  <>
                    <Descriptions.Item label={t('log.exception.detail.haltReason')} span={2}>
                      {showRaw(selected.detailReason)}
                    </Descriptions.Item>
                    <Descriptions.Item label={t('log.exception.detail.runtime')}>
                      {formatRuntime(selected.runtimeBeforeReboot)}
                    </Descriptions.Item>
                  </>
                )}
              </Descriptions>
            </Card>
          </div>
        )}
      </Drawer>

      {/* 按设备汇总统计弹窗（总次数/异常次数，跟随筛选） */}
      <Modal
        title={t('log.event.statModal.title')}
        open={statVisible}
        onCancel={() => setStatVisible(false)}
        footer={null}
        width={900}
      >
        <Space style={{ marginBottom: 12, display: 'flex', justifyContent: 'flex-end' }}>
          <Button
            type="primary"
            icon={<ExportOutlined />}
            onClick={handleExportStat}
            disabled={statRows.length === 0}
          >
            {t('log.export')}
          </Button>
        </Space>
        <Table<DeviceRebootStat>
          size="small"
          rowKey="deviceSn"
          loading={statLoading}
          columns={statColumns}
          dataSource={statRows}
          pagination={{ pageSize: 10, size: 'small', showSizeChanger: true }}
          scroll={{ y: 380 }}
        />
      </Modal>
    </ListPageLayout>
  );
}
