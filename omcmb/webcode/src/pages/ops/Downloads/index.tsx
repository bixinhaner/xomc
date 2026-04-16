import { useState, useMemo } from 'react';
import { Button, Tag, Space, Progress, message, Tabs } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

type DownloadStatus = 'ready' | 'generating' | 'expired' | 'failed';
type ResourceFileType = 'config' | 'log' | 'firmware' | 'mr' | 'perf';
type ReportFileType = 'lte-standard' | 'station' | 'kpi' | 'mr-analysis' | 'poll-stats';

interface ResourceDownload {
  id: string;
  fileName: string;
  fileType: ResourceFileType;
  deviceSn?: string;
  deviceName?: string;
  fileSize: number;
  status: DownloadStatus;
  progress?: number;
  createTime: string;
  expireTime?: string;
  creator: string;
}

interface ReportDownload {
  id: string;
  reportName: string;
  reportType: ReportFileType;
  timeRange: string;
  fileSize: number;
  status: DownloadStatus;
  progress?: number;
  createTime: string;
  expireTime?: string;
  creator: string;
}

const statusColorMap: Record<DownloadStatus, string> = {
  ready: 'green',
  generating: 'processing',
  expired: 'default',
  failed: 'red',
};

const resourceTypeColorMap: Record<ResourceFileType, string> = {
  config: 'blue',
  log: 'orange',
  firmware: 'purple',
  mr: 'green',
  perf: 'cyan',
};

function formatFileSize(bytes: number): string {
  if (bytes >= 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  return `${(bytes / 1024).toFixed(2)} KB`;
}

const mockResourceDownloads: ResourceDownload[] = [
  { id: 'dl-001', fileName: 'ENB00001_config_20240601.xml', fileType: 'config', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', fileSize: 1024 * 156, status: 'ready', createTime: '2024-06-01T09:00:00.000Z', expireTime: '2024-06-08T09:00:00.000Z', creator: 'admin' },
  { id: 'dl-002', fileName: 'ENB00001_runtime_log_20240601.tar.gz', fileType: 'log', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', fileSize: 1024 * 1024 * 45, status: 'ready', createTime: '2024-06-01T10:00:00.000Z', expireTime: '2024-06-03T10:00:00.000Z', creator: 'operator01' },
  { id: 'dl-003', fileName: 'V100R011C10SPC200_eNB_firmware.zip', fileType: 'firmware', fileSize: 1024 * 1024 * 512, status: 'ready', createTime: '2024-05-15T00:00:00.000Z', creator: 'admin' },
  { id: 'dl-004', fileName: 'GNB00001_MRO_20240601.tar.gz', fileType: 'mr', deviceSn: 'GNB00001', deviceName: '北京-gNB-0001', fileSize: 1024 * 1024 * 128, status: 'generating', progress: 65, createTime: new Date(Date.now() - 300000).toISOString(), creator: 'operator01' },
  { id: 'dl-005', fileName: 'ENB00010_perf_20240601_15min.csv', fileType: 'perf', deviceSn: 'ENB00010', deviceName: '上海-eNB-0010', fileSize: 1024 * 1024 * 8, status: 'ready', createTime: '2024-06-01T02:00:00.000Z', expireTime: '2024-06-08T02:00:00.000Z', creator: 'admin' },
  { id: 'dl-006', fileName: 'ENB00020_config_20240515.xml', fileType: 'config', deviceSn: 'ENB00020', deviceName: '广州-eNB-0020', fileSize: 1024 * 180, status: 'expired', createTime: '2024-05-15T09:00:00.000Z', expireTime: '2024-05-22T09:00:00.000Z', creator: 'operator02' },
  { id: 'dl-007', fileName: 'ALL_perf_20240601_1h_batch.zip', fileType: 'perf', fileSize: 1024 * 1024 * 256, status: 'failed', createTime: '2024-06-01T03:00:00.000Z', creator: 'admin' },
  { id: 'dl-008', fileName: 'ENB00001_ENB00002_MRE_20240601.zip', fileType: 'mr', fileSize: 1024 * 1024 * 32, status: 'ready', createTime: '2024-06-01T01:00:00.000Z', expireTime: '2024-06-04T01:00:00.000Z', creator: 'operator01' },
];

const mockReportDownloads: ReportDownload[] = [
  { id: 'rpt-dl-001', reportName: '华北区LTE标准日报-20240601', reportType: 'lte-standard', timeRange: '2024-06-01', fileSize: 1024 * 1024 * 2, status: 'ready', createTime: '2024-06-01T06:00:00.000Z', expireTime: '2024-06-15T06:00:00.000Z', creator: 'system' },
  { id: 'rpt-dl-002', reportName: '全网站点月报-202405', reportType: 'station', timeRange: '2024-05-01 ~ 2024-05-31', fileSize: 1024 * 1024 * 8, status: 'ready', createTime: '2024-06-01T02:00:00.000Z', expireTime: '2024-07-01T02:00:00.000Z', creator: 'system' },
  { id: 'rpt-dl-003', reportName: 'KPI历史分析-北京eNB-20240601', reportType: 'kpi', timeRange: '2024-05-25 ~ 2024-06-01', fileSize: 1024 * 1024 * 5, status: 'generating', progress: 40, createTime: new Date(Date.now() - 120000).toISOString(), creator: 'operator01' },
  { id: 'rpt-dl-004', reportName: 'MR专项分析报告-全网-202406W1', reportType: 'mr-analysis', timeRange: '2024-06-01 ~ 2024-06-07', fileSize: 1024 * 1024 * 15, status: 'ready', createTime: '2024-06-07T08:00:00.000Z', expireTime: '2024-06-21T08:00:00.000Z', creator: 'admin' },
  { id: 'rpt-dl-005', reportName: '轮询统计汇总-20240601', reportType: 'poll-stats', timeRange: '2024-06-01', fileSize: 1024 * 512, status: 'ready', createTime: '2024-06-01T10:00:00.000Z', expireTime: '2024-06-08T10:00:00.000Z', creator: 'system' },
  { id: 'rpt-dl-006', reportName: 'LTE标准日报-20240531', reportType: 'lte-standard', timeRange: '2024-05-31', fileSize: 1024 * 1024 * 2, status: 'expired', createTime: '2024-05-31T06:00:00.000Z', expireTime: '2024-06-07T06:00:00.000Z', creator: 'system' },
  { id: 'rpt-dl-007', reportName: '大区KPI对比报告-Q2-2024', reportType: 'kpi', timeRange: '2024-04-01 ~ 2024-06-30', fileSize: 1024 * 1024 * 25, status: 'failed', createTime: '2024-06-01T08:00:00.000Z', creator: 'admin' },
];

export default function Downloads() {
  const t = useT();
  const [resourceFilters, setResourceFilters] = useState<Record<string, unknown>>({});
  const [reportFilters, setReportFilters] = useState<Record<string, unknown>>({});
  const [resourcePage, setResourcePage] = useState(1);
  const [reportPage, setReportPage] = useState(1);
  const [pageSize] = useState(10);
  const [downloads, setDownloads] = useState<ResourceDownload[]>(mockResourceDownloads);
  const [reports, setReports] = useState<ReportDownload[]>(mockReportDownloads);

  const statusLabelMap: Record<DownloadStatus, string> = useMemo(() => ({
    ready: t('ops.downloadReady'),
    generating: t('mr.generating'),
    expired: t('ops.downloadExpired'),
    failed: t('status.failed'),
  }), [t]);

  const resourceTypeLabelMap: Record<ResourceFileType, string> = useMemo(() => ({
    config: t('ops.fileTypeConfig'),
    log: t('ops.fileTypeLog'),
    firmware: t('ops.fileTypeFirmware'),
    mr: t('ops.fileTypeMR'),
    perf: t('ops.fileTypePerf'),
  }), [t]);

  const reportTypeLabelMap: Record<ReportFileType, string> = useMemo(() => ({
    'lte-standard': t('ops.reportTypeLTE'),
    'station': t('ops.reportTypeStation'),
    'kpi': t('ops.reportTypeKPI'),
    'mr-analysis': t('ops.reportTypeMR'),
    'poll-stats': t('ops.reportTypePoll'),
  }), [t]);

  const resourceFilterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('ops.fileNameOrSn'), type: 'input', placeholder: t('ops.fileNameOrSnPlaceholder') },
    {
      name: 'fileType',
      label: t('ops.fileType'),
      type: 'select',
      options: [
        { label: t('ops.fileTypeConfig'), value: 'config' },
        { label: t('ops.fileTypeLog'), value: 'log' },
        { label: t('ops.fileTypeFirmware'), value: 'firmware' },
        { label: t('ops.fileTypeMR'), value: 'mr' },
        { label: t('ops.fileTypePerf'), value: 'perf' },
      ],
    },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('ops.downloadReady'), value: 'ready' },
        { label: t('mr.generating'), value: 'generating' },
        { label: t('ops.downloadExpired'), value: 'expired' },
        { label: t('status.failed'), value: 'failed' },
      ],
    },
  ], [t]);

  const reportFilterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('ops.reportName'), type: 'input', placeholder: t('ops.reportNamePlaceholder') },
    {
      name: 'reportType',
      label: t('ops.reportType'),
      type: 'select',
      options: [
        { label: t('ops.reportTypeLTE'), value: 'lte-standard' },
        { label: t('ops.reportTypeStation'), value: 'station' },
        { label: t('ops.reportTypeKPI'), value: 'kpi' },
        { label: t('ops.reportTypeMR'), value: 'mr-analysis' },
        { label: t('ops.reportTypePoll'), value: 'poll-stats' },
      ],
    },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('ops.downloadReady'), value: 'ready' },
        { label: t('mr.generating'), value: 'generating' },
        { label: t('ops.downloadExpired'), value: 'expired' },
        { label: t('status.failed'), value: 'failed' },
      ],
    },
  ], [t]);

  const filteredResources = downloads.filter((d) => {
    if (resourceFilters.keyword) {
      const kw = String(resourceFilters.keyword).toLowerCase();
      if (!d.fileName.toLowerCase().includes(kw) && !(d.deviceSn ?? '').toLowerCase().includes(kw)) return false;
    }
    if (resourceFilters.fileType && d.fileType !== resourceFilters.fileType) return false;
    if (resourceFilters.status && d.status !== resourceFilters.status) return false;
    return true;
  });

  const filteredReports = reports.filter((r) => {
    if (reportFilters.keyword && !r.reportName.includes(String(reportFilters.keyword))) return false;
    if (reportFilters.reportType && r.reportType !== reportFilters.reportType) return false;
    if (reportFilters.status && r.status !== reportFilters.status) return false;
    return true;
  });

  const handleDownload = (fileName: string) => {
    void message.success(`${t('ops.startDownload')}: ${fileName}`);
  };

  const handleRetry = (id: string, type: 'resource' | 'report') => {
    if (type === 'resource') {
      setDownloads((prev) => prev.map((d) => d.id === id ? { ...d, status: 'generating', progress: 0 } : d));
      setTimeout(() => {
        setDownloads((prev) => prev.map((d) => d.id === id ? { ...d, status: 'ready', progress: undefined } : d));
        void message.success(t('ops.fileRegenerated'));
      }, 2000);
    } else {
      setReports((prev) => prev.map((r) => r.id === id ? { ...r, status: 'generating', progress: 0 } : r));
      setTimeout(() => {
        setReports((prev) => prev.map((r) => r.id === id ? { ...r, status: 'ready', progress: undefined } : r));
        void message.success(t('ops.reportRegenerated'));
      }, 2000);
    }
  };

  const resourceColumns: DataTableColumn<ResourceDownload & Record<string, unknown>>[] = useMemo(() => [
    { key: 'fileName', title: t('mr.fileName'), dataIndex: 'fileName', ellipsis: true, width: 260 },
    {
      key: 'fileType', title: t('table.type'), dataIndex: 'fileType', width: 100,
      render: (val) => {
        const tp = val as ResourceFileType;
        return <Tag color={resourceTypeColorMap[tp]}>{resourceTypeLabelMap[tp]}</Tag>;
      },
    },
    {
      key: 'deviceName', title: t('ops.sourceDevice'), dataIndex: 'deviceName', width: 160, ellipsis: true,
      render: (val, record) => {
        const d = record as ResourceDownload;
        if (!d.deviceSn) return <span style={{ color: '#999' }}>—</span>;
        return (
          <span>
            <span style={{ fontFamily: 'monospace', fontSize: 11, marginRight: 4 }}>{d.deviceSn}</span>
            {val ? String(val) : ''}
          </span>
        );
      },
    },
    {
      key: 'fileSize', title: t('mr.fileSize'), dataIndex: 'fileSize', width: 100,
      render: (val) => formatFileSize(Number(val)),
    },
    {
      key: 'status', title: t('table.status'), dataIndex: 'status', width: 100,
      render: (val, record) => {
        const d = record as ResourceDownload;
        const s = val as DownloadStatus;
        if (s === 'generating' && d.progress !== undefined) {
          return <Progress percent={d.progress} size="small" style={{ width: 90 }} />;
        }
        return <Tag color={statusColorMap[s]}>{statusLabelMap[s]}</Tag>;
      },
    },
    {
      key: 'createTime', title: t('table.createTime'), dataIndex: 'createTime', width: 160,
      render: (val) => new Date(String(val)).toLocaleString('zh-CN'),
    },
    {
      key: 'expireTime', title: t('ops.expireTime'), dataIndex: 'expireTime', width: 160,
      render: (val) => {
        if (!val) return <span style={{ color: '#999' }}>—</span>;
        const expiry = new Date(String(val));
        const now = new Date();
        const daysLeft = Math.ceil((expiry.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));
        const color = daysLeft < 0 ? '#ff4d4f' : daysLeft < 3 ? '#faad14' : '#666';
        return <span style={{ color }}>{expiry.toLocaleDateString('zh-CN')}</span>;
      },
    },
    { key: 'creator', title: t('mr.creator'), dataIndex: 'creator', width: 90 },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 80, fixed: 'right',
      render: (_, record) => {
        const d = record as ResourceDownload;
        if (d.status === 'ready') {
          return (
            <Button type="link" size="small"
              onClick={() => handleDownload(d.fileName)}>
              {t('common.download')}
            </Button>
          );
        }
        if (d.status === 'failed') {
          return (
            <Button type="link" size="small"
              onClick={() => handleRetry(d.id, 'resource')}>
              {t('ops.retry')}
            </Button>
          );
        }
        return <span style={{ color: '#999', fontSize: 12 }}>—</span>;
      },
    },
  ], [t, resourceTypeLabelMap, statusLabelMap]);

  const reportColumns: DataTableColumn<ReportDownload & Record<string, unknown>>[] = useMemo(() => [
    { key: 'reportName', title: t('ops.reportName'), dataIndex: 'reportName', ellipsis: true, width: 280 },
    {
      key: 'reportType', title: t('table.type'), dataIndex: 'reportType', width: 120,
      render: (val) => <Tag>{reportTypeLabelMap[val as ReportFileType] ?? String(val)}</Tag>,
    },
    { key: 'timeRange', title: t('ops.statsPeriod'), dataIndex: 'timeRange', width: 180, ellipsis: true },
    {
      key: 'fileSize', title: t('mr.fileSize'), dataIndex: 'fileSize', width: 90,
      render: (val) => formatFileSize(Number(val)),
    },
    {
      key: 'status', title: t('table.status'), dataIndex: 'status', width: 100,
      render: (val, record) => {
        const r = record as ReportDownload;
        const s = val as DownloadStatus;
        if (s === 'generating' && r.progress !== undefined) {
          return <Progress percent={r.progress} size="small" style={{ width: 90 }} />;
        }
        return <Tag color={statusColorMap[s]}>{statusLabelMap[s]}</Tag>;
      },
    },
    {
      key: 'createTime', title: t('ops.generateTime'), dataIndex: 'createTime', width: 160,
      render: (val) => new Date(String(val)).toLocaleString('zh-CN'),
    },
    {
      key: 'expireTime', title: t('ops.expireTime'), dataIndex: 'expireTime', width: 120,
      render: (val) => {
        if (!val) return <span style={{ color: '#999' }}>—</span>;
        const expiry = new Date(String(val));
        const now = new Date();
        const daysLeft = Math.ceil((expiry.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));
        const color = daysLeft < 0 ? '#ff4d4f' : daysLeft < 3 ? '#faad14' : '#666';
        return <span style={{ color }}>{expiry.toLocaleDateString('zh-CN')}</span>;
      },
    },
    { key: 'creator', title: t('mr.creator'), dataIndex: 'creator', width: 90 },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 80, fixed: 'right',
      render: (_, record) => {
        const r = record as ReportDownload;
        if (r.status === 'ready') {
          return (
            <Button type="link" size="small"
              onClick={() => handleDownload(r.reportName)}>
              {t('common.download')}
            </Button>
          );
        }
        if (r.status === 'failed') {
          return (
            <Button type="link" size="small"
              onClick={() => handleRetry(r.id, 'report')}>
              {t('ops.retry')}
            </Button>
          );
        }
        return <span style={{ color: '#999', fontSize: 12 }}>—</span>;
      },
    },
  ], [t, reportTypeLabelMap, statusLabelMap]);

  return (
    <ListPageLayout title={t('nav.ops.downloads')} subtitle={t('ops.downloadsSubtitle')}>
      <Tabs
        items={[
          {
            key: 'resource',
            label: t('ops.resourceFileDownload'),
            children: (
              <>
                <FilterBar
                  filterId="ops-downloads-resource-filter"
                  fields={resourceFilterFields}
                  onSearch={(vals) => { setResourceFilters(vals); setResourcePage(1); }}
                  onReset={() => { setResourceFilters({}); setResourcePage(1); }}
                />
                <DataTable
                  tableId="ops-downloads-resource-list"
                  columns={resourceColumns}
                  dataSource={filteredResources.slice((resourcePage - 1) * pageSize, resourcePage * pageSize) as (ResourceDownload & Record<string, unknown>)[]}
                  loading={false}
                  rowKey="id"
                  total={filteredResources.length}
                  pageSize={pageSize}
                  currentPage={resourcePage}
                  onPageChange={(p) => setResourcePage(p)}
                  onRefresh={() => setDownloads([...mockResourceDownloads])}
                  scroll={{ x: 1300 }}
                  alarmRowStyle={(record) => {
                    const d = record as ResourceDownload;
                    return d.status === 'failed' ? 'major' : null;
                  }}
                />
              </>
            ),
          },
          {
            key: 'report',
            label: t('ops.reportDownload'),
            children: (
              <>
                <FilterBar
                  filterId="ops-downloads-report-filter"
                  fields={reportFilterFields}
                  onSearch={(vals) => { setReportFilters(vals); setReportPage(1); }}
                  onReset={() => { setReportFilters({}); setReportPage(1); }}
                />
                <DataTable
                  tableId="ops-downloads-report-list"
                  columns={reportColumns}
                  dataSource={filteredReports.slice((reportPage - 1) * pageSize, reportPage * pageSize) as (ReportDownload & Record<string, unknown>)[]}
                  loading={false}
                  rowKey="id"
                  total={filteredReports.length}
                  pageSize={pageSize}
                  currentPage={reportPage}
                  onPageChange={(p) => setReportPage(p)}
                  onRefresh={() => setReports([...mockReportDownloads])}
                  scroll={{ x: 1300 }}
                />
              </>
            ),
          },
        ]}
      />
    </ListPageLayout>
  );
}
