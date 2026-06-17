import { useState, useMemo } from 'react';
import { Button, Tree, Tag, Space, message, Dropdown } from 'antd';
import { EyeOutlined, DownloadOutlined, ExportOutlined, PlusOutlined, MoreOutlined } from '@ant-design/icons';
import type { MenuProps } from 'antd';
import type { DataNode } from 'antd/es/tree';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import {
  useDownloadReport,
  useReportRecords,
  useGenerateReport,
  useReportDefinitions,
} from '@core/hooks/api/useReports';
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

// T-0022: shapes + mock data centralised in @core/mock/data/reports.
import { formatSystemTime } from '@core/utils/systemTime';
import {
  mockLTEReportRecords,
  type LTEReportRecord,
  type LTEReportStatus,
} from '@core/mock/data/reports';

const statusColorMap: Record<LTEReportStatus, string> = {
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
  const generateReport = useGenerateReport();
  // Probe definitions/records endpoints to keep the page wired to real data
  // flow even while the categorised display still consumes the local LTE
  // mock listing (no categorised endpoint exists server-side yet).
  const { data: definitionsData } = useReportDefinitions({ page: 1, pageSize: 50 });
  useReportRecords({ page: 1, pageSize: 50 });

  const currentRecords = mockLTEReportRecords.filter((r) => r.category === selectedCategory);
  const startIndex = (page - 1) * pageSize;
  const paginated = currentRecords.slice(startIndex, startIndex + pageSize);

  const columns: DataTableColumn<LTEReportRecord & Record<string, unknown>>[] = useMemo(() => [
    { key: 'reportName', title: t('table.name'), dataIndex: 'reportName', ellipsis: true },
    { key: 'reportType', title: t('table.type'), dataIndex: 'reportType', width: 90, render: (val) => <Tag>{String(val)}</Tag> },
    { key: 'period', title: t('perf.timeRange'), dataIndex: 'period', width: 120 },
    {
      key: 'generatedTime', title: t('table.createTime'), dataIndex: 'generatedTime', width: 160,
      render: (val) => val ? formatSystemTime(String(val)) : '—',
    },
    {
      key: 'status', title: t('table.status'), dataIndex: 'status', width: 100,
      render: (val) => {
        const s = val as LTEReportStatus;
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
        const r = record as LTEReportRecord;
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
          <Button
            type="primary"
            size="small"
            icon={<PlusOutlined />}
            loading={generateReport.isPending}
            disabled={!definitionsData || definitionsData.items.length === 0}
            onClick={async () => {
              const firstDef = definitionsData?.items?.[0];
              if (!firstDef) {
                void message.warning(t('common.pleaseSelect'));
                return;
              }
              try {
                await generateReport.mutateAsync({ definitionId: firstDef.id });
                void message.success(t('common.save'));
              } catch {
                void message.error(t('status.failed'));
              }
            }}
          >
            {t('common.add')}
          </Button>
        </div>
        <div style={{ flex: 1, overflow: 'auto' }}>
          <DataTable
            tableId="lte-report-list"
            columns={columns}
            dataSource={paginated as (LTEReportRecord & Record<string, unknown>)[]}
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
