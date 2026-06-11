import { useState, useMemo } from 'react';
import { Button, Card, Tag, Space, message } from 'antd';
import { EyeOutlined, DownloadOutlined, PlusOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useDownloadReport } from '@core/hooks/api/useReports';
import { useT } from '@/hooks/useT';

type ReportStatus = 'generated' | 'generating' | 'failed';

interface MRAnalysisReport {
  id: string;
  reportName: string;
  analysisType: string;
  timeRange: string;
  status: ReportStatus;
  generatedTime: string;
  fileSize?: number;
  deviceCount: number;
}

const mockReports: MRAnalysisReport[] = [
  { id: 'mr-rpt-001', reportName: '北京区eNB覆盖分析报告_2024W22', analysisType: 'coverage', timeRange: '2024-05-27 ~ 2024-06-02', status: 'generated', generatedTime: '2024-06-03T02:00:00.000Z', fileSize: 1024 * 1024 * 5, deviceCount: 35 },
  { id: 'mr-rpt-002', reportName: '全网干扰分析报告_202406', analysisType: 'interference', timeRange: '2024-06-01 ~ 2024-06-30', status: 'generating', generatedTime: '', deviceCount: 215 },
  { id: 'mr-rpt-003', reportName: '华东区移动性分析_2024Q2', analysisType: 'mobility', timeRange: '2024-04-01 ~ 2024-06-30', status: 'generated', generatedTime: '2024-07-02T08:00:00.000Z', fileSize: 1024 * 1024 * 12, deviceCount: 120 },
  { id: 'mr-rpt-004', reportName: 'gNB负载评估报告_202406', analysisType: 'load', timeRange: '2024-06-01 ~ 2024-06-30', status: 'failed', generatedTime: '2024-07-01T06:00:00.000Z', deviceCount: 18 },
  { id: 'mr-rpt-005', reportName: 'VoLTE质量MR分析_20240601', analysisType: 'quality', timeRange: '2024-06-01', status: 'generated', generatedTime: '2024-06-02T03:00:00.000Z', fileSize: 1024 * 512, deviceCount: 50 },
];

const analysisTypeColorMap: Record<string, string> = {
  coverage: 'blue',
  interference: 'red',
  mobility: 'orange',
  load: 'purple',
  quality: 'green',
};

const statusColorMap: Record<ReportStatus, string> = {
  generated: 'green',
  generating: 'processing',
  failed: 'red',
};

function formatFileSize(bytes: number): string {
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  return `${(bytes / 1024).toFixed(2)} KB`;
}

export default function Reports() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const downloadReport = useDownloadReport();

  const analysisTypeLabelMap: Record<string, string> = useMemo(() => ({
    coverage: t('mr.coverageAnalysis'),
    interference: t('mr.interferenceAnalysis'),
    mobility: t('mr.mobilityAnalysis'),
    load: t('mr.loadAnalysis'),
    quality: t('mr.qualityAnalysis'),
  }), [t]);

  const statusLabelMap: Record<ReportStatus, string> = useMemo(() => ({
    generated: t('mr.generated'),
    generating: t('mr.generating'),
    failed: t('mr.generateFailed'),
  }), [t]);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('mr.reportName'), type: 'input', placeholder: t('mr.reportNamePlaceholder') },
    {
      name: 'analysisType',
      label: t('mr.analysisType'),
      type: 'select',
      options: [
        { label: t('mr.coverageAnalysis'), value: 'coverage' },
        { label: t('mr.interferenceAnalysis'), value: 'interference' },
        { label: t('mr.mobilityAnalysis'), value: 'mobility' },
        { label: t('mr.loadAnalysis'), value: 'load' },
        { label: t('mr.qualityAnalysis'), value: 'quality' },
      ],
    },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('mr.generated'), value: 'generated' },
        { label: t('mr.generating'), value: 'generating' },
        { label: t('status.failed'), value: 'failed' },
      ],
    },
    { name: 'timeRange', label: t('mr.generateTime'), type: 'date-range' },
  ], [t]);

  const filtered = mockReports.filter((r) => {
    if (filters.keyword && !r.reportName.includes(String(filters.keyword))) return false;
    if (filters.analysisType && r.analysisType !== filters.analysisType) return false;
    if (filters.status && r.status !== filters.status) return false;
    return true;
  });

  const startIndex = (page - 1) * pageSize;
  const paginated = filtered.slice(startIndex, startIndex + pageSize);

  const columns: DataTableColumn<MRAnalysisReport & Record<string, unknown>>[] = useMemo(() => [
    { key: 'reportName', title: t('mr.reportName'), dataIndex: 'reportName', ellipsis: true, width: 260 },
    {
      key: 'analysisType', title: t('mr.analysisType'), dataIndex: 'analysisType', width: 110,
      render: (val) => <Tag color={analysisTypeColorMap[String(val)] ?? 'default'}>{analysisTypeLabelMap[String(val)] ?? String(val)}</Tag>,
    },
    { key: 'timeRange', title: t('mr.timeRange'), dataIndex: 'timeRange', width: 180 },
    { key: 'deviceCount', title: t('mr.deviceScope'), dataIndex: 'deviceCount', width: 90, render: (val) => String(val) },
    {
      key: 'status', title: t('table.status'), dataIndex: 'status', width: 100,
      render: (val) => {
        const s = val as ReportStatus;
        return <Tag color={statusColorMap[s]}>{statusLabelMap[s]}</Tag>;
      },
    },
    {
      key: 'generatedTime', title: t('mr.generateTime'), dataIndex: 'generatedTime', width: 160,
      render: (val) => val ? new Date(String(val)).toLocaleString('zh-CN') : '—',
    },
    {
      key: 'fileSize', title: t('mr.fileSize'), dataIndex: 'fileSize', width: 100,
      render: (val) => val ? formatFileSize(Number(val)) : '—',
    },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 120, fixed: 'right',
      render: (_, record) => {
        const r = record as MRAnalysisReport;
        return (
          <Space size={4}>
            <Button type="link" size="small" icon={<EyeOutlined />} disabled={r.status !== 'generated'}>{t('common.view')}</Button>
            <Button type="link" size="small" icon={<DownloadOutlined />} disabled={r.status !== 'generated'}
              onClick={() => downloadReport.mutate(r.id, { onSuccess: () => void message.success(t('mr.downloadTaskCreated')) })}>
              {t('common.download')}
            </Button>
          </Space>
        );
      },
    },
  ], [t, analysisTypeLabelMap, statusLabelMap, downloadReport]);

  return (
    <ListPageLayout
      title={t('nav.mr.reports')}
      subtitle={t('mr.reportsSubtitle')}
      extra={<Button type="primary" icon={<PlusOutlined />} onClick={() => void message.info(t('mr.newReportFeature'))}>{t('mr.newReport')}</Button>}
    >
      <FilterBar
        filterId="mr-reports-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <Card
        size="small"
        variant="outlined"
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
        <DataTable
          tableId="mr-reports-list"
          columns={columns}
          dataSource={paginated as (MRAnalysisReport & Record<string, unknown>)[]}
          loading={false}
          rowKey="id"
          total={filtered.length}
          pageSize={pageSize}
          currentPage={page}
          onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
          scroll={{ x: 1100 }}
        />
      </Card>
    </ListPageLayout>
  );
}
