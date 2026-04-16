import { useState, useMemo } from 'react';
import { Button, Tree, Tag, Space, Tooltip, message, Dropdown } from 'antd';
import { EyeOutlined, DownloadOutlined, ExportOutlined, PlusOutlined, MoreOutlined } from '@ant-design/icons';
import type { MenuProps } from 'antd';
import type { DataNode } from 'antd/es/tree';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useReportRecords, useDownloadReport } from '@/hooks/api/useReports';
import { useT } from '@/hooks/useT';

const REPORT_CATEGORY_KEYS = [
  {
    key: 'kpi', titleKey: 'report.category.kpi',
    children: [
      { key: 'kpi-daily', titleKey: 'report.category.kpiDaily' },
      { key: 'kpi-weekly', titleKey: 'report.category.kpiWeekly' },
      { key: 'kpi-monthly', titleKey: 'report.category.kpiMonthly' },
    ],
  },
  {
    key: 'availability', titleKey: 'report.category.availability',
    children: [
      { key: 'avail-station', titleKey: 'report.category.availStation' },
      { key: 'avail-cell', titleKey: 'report.category.availCell' },
    ],
  },
  {
    key: 'capacity', titleKey: 'report.category.capacity',
    children: [
      { key: 'cap-prb', titleKey: 'report.category.capPrb' },
      { key: 'cap-user', titleKey: 'report.category.capUser' },
    ],
  },
  {
    key: 'quality', titleKey: 'report.category.quality',
    children: [
      { key: 'qual-voice', titleKey: 'report.category.qualVoice' },
      { key: 'qual-data', titleKey: 'report.category.qualData' },
    ],
  },
  {
    key: 'mobility', titleKey: 'report.category.mobility',
    children: [
      { key: 'mob-handover', titleKey: 'report.category.mobHandover' },
      { key: 'mob-rach', titleKey: 'report.category.mobRach' },
    ],
  },
];

type ReportStatus = 'generated' | 'generating' | 'failed' | 'scheduled';

interface ReportRecord {
  id: string;
  reportName: string;
  reportType: string;
  period: string;
  generatedTime: string;
  status: ReportStatus;
  fileSize?: number;
  category: string;
}

const mockReportRecords: ReportRecord[] = [
  { id: 'rr-001', reportName: '华北区域eNB日KPI报表_20240601', reportType: '日报', period: '2024-06-01', generatedTime: '2024-06-02T01:00:00.000Z', status: 'generated', fileSize: 1024 * 512, category: 'kpi-daily' },
  { id: 'rr-002', reportName: '华北区域eNB日KPI报表_20240602', reportType: '日报', period: '2024-06-02', generatedTime: '2024-06-03T01:00:00.000Z', status: 'generated', fileSize: 1024 * 480, category: 'kpi-daily' },
  { id: 'rr-003', reportName: '全网周KPI报表_W22_2024', reportType: '周报', period: '2024-W22', generatedTime: '2024-06-03T06:00:00.000Z', status: 'generated', fileSize: 1024 * 1024 * 2, category: 'kpi-weekly' },
  { id: 'rr-004', reportName: '基站可用性日报_20240601', reportType: '日报', period: '2024-06-01', generatedTime: '2024-06-02T02:00:00.000Z', status: 'generated', fileSize: 1024 * 256, category: 'avail-station' },
  { id: 'rr-005', reportName: '全网月KPI报表_2024-05', reportType: '月报', period: '2024-05', generatedTime: '2024-06-01T08:00:00.000Z', status: 'generated', fileSize: 1024 * 1024 * 8, category: 'kpi-monthly' },
  { id: 'rr-006', reportName: 'PRB利用率周报_W23_2024', reportType: '周报', period: '2024-W23', generatedTime: '', status: 'generating', category: 'cap-prb' },
  { id: 'rr-007', reportName: 'VoLTE质量日报_20240603', reportType: '日报', period: '2024-06-03', generatedTime: '', status: 'failed', category: 'qual-voice' },
];

const statusColorMap: Record<ReportStatus, string> = {
  generated: 'green',
  generating: 'processing',
  failed: 'red',
  scheduled: 'blue',
};

function formatFileSize(bytes: number): string {
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  return `${(bytes / 1024).toFixed(2)} KB`;
}

export default function LTEStandardReport() {
  const t = useT();
  const reportCategories: DataNode[] = useMemo(() =>
    REPORT_CATEGORY_KEYS.map((cat) => ({
      key: cat.key,
      title: t(cat.titleKey),
      children: cat.children.map((child) => ({
        key: child.key,
        title: t(child.titleKey),
      })),
    })),
  [t]);
  const [selectedCategory, setSelectedCategory] = useState<string>('kpi-daily');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const downloadReport = useDownloadReport();

  const currentRecords = mockReportRecords.filter((r) => r.category === selectedCategory);
  const startIndex = (page - 1) * pageSize;
  const paginated = currentRecords.slice(startIndex, startIndex + pageSize);

  const columns: DataTableColumn<ReportRecord & Record<string, unknown>>[] = useMemo(() => [
    { key: 'reportName', title: t('table.name'), dataIndex: 'reportName', ellipsis: true },
    { key: 'reportType', title: t('table.type'), dataIndex: 'reportType', width: 90, render: (val) => <Tag>{String(val)}</Tag> },
    { key: 'period', title: t('perf.timeRange'), dataIndex: 'period', width: 120 },
    {
      key: 'generatedTime', title: t('table.createTime'), dataIndex: 'generatedTime', width: 160,
      render: (val) => val ? new Date(String(val)).toLocaleString('zh-CN') : '—',
    },
    {
      key: 'status', title: t('table.status'), dataIndex: 'status', width: 100,
      render: (val) => {
        const s = val as ReportStatus;
        return <Tag color={statusColorMap[s]}>{s === 'generated' ? t('status.success') : s === 'generating' ? t('status.running') : s === 'failed' ? t('status.failed') : t('status.pending')}</Tag>;
      },
    },
    {
      key: 'fileSize', title: t('table.total'), dataIndex: 'fileSize', width: 100,
      render: (val) => val ? formatFileSize(Number(val)) : '—',
    },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 100, fixed: 'right',
      render: (_, record) => {
        const r = record as ReportRecord;
        const items: MenuProps['items'] = [
          { key: 'download', label: t('common.download'), icon: <DownloadOutlined />, disabled: r.status !== 'generated',
            onClick: () => downloadReport.mutate(r.id, { onSuccess: () => void message.success(t('common.download')) }),
          },
          { key: 'export', label: t('common.export'), icon: <ExportOutlined />, disabled: r.status !== 'generated' },
        ];
        return (
          <Space size={4}>
            <Button type="link" size="small" icon={<EyeOutlined />} disabled={r.status !== 'generated'}>{t('common.view')}</Button>
            <Dropdown menu={{ items }} trigger={['click']}>
              <Button type="text" size="small" icon={<MoreOutlined />} />
            </Dropdown>
          </Space>
        );
      },
    },
  ], [t, downloadReport]);

  const treePanel = (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div style={{ padding: '12px 8px', borderBottom: '1px solid #f0f0f0' }}>
        <span style={{ fontWeight: 500 }}>{t('nav.report.lteStandard')}</span>
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: 8 }}>
        <Tree
          treeData={reportCategories}
          defaultExpandAll
          selectedKeys={[selectedCategory]}
          onSelect={(keys) => { if (keys.length > 0) { setSelectedCategory(String(keys[0])); setPage(1); } }}
        />
      </div>
    </div>
  );

  return (
    <TreeListPageLayout tree={treePanel}>
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        <div style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center', background: '#fafafa' }}>
          <span style={{ fontWeight: 500 }}>{t('table.total')} ({currentRecords.length})</span>
          <Button type="primary" size="small" icon={<PlusOutlined />}
            onClick={() => void message.info(t('common.featureInDev'))}>
            {t('common.add')}
          </Button>
        </div>
        <div style={{ flex: 1, overflow: 'auto' }}>
          <DataTable
            tableId="lte-report-list"
            columns={columns}
            dataSource={paginated as (ReportRecord & Record<string, unknown>)[]}
            loading={false}
            rowKey="id"
            total={currentRecords.length}
            pageSize={pageSize}
            currentPage={page}
            onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
            scroll={{ x: 900 }}
          />
        </div>
      </div>
    </TreeListPageLayout>
  );
}
