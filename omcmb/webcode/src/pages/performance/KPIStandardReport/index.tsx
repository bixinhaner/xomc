import { useState, useMemo } from 'react';
import { Button, Input, Space, Tag, Tree, Typography, message } from 'antd';
import { SearchOutlined, DownloadOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

interface KPIReportRow extends Record<string, unknown> {
  id: string;
  kpiName: string;
  kpiCode: string;
  unit: string;
  currentValue: number;
  threshold: number;
  status: 'normal' | 'warning' | 'critical';
  trend: number[];
  category: string;
}

const KPI_TREE_DATA: DataNode[] = [
  {
    title: '全部KPI',
    key: 'all',
    children: [
      {
        title: '无线接入',
        key: 'radio-access',
        children: [
          { title: 'RRC接入', key: 'rrc' },
          { title: 'E-RAB', key: 'erab' },
          { title: '切换', key: 'handover' },
        ],
      },
      {
        title: '无线资源',
        key: 'radio-resource',
        children: [
          { title: '用户数', key: 'user-count' },
          { title: '上行吞吐量', key: 'ul-throughput' },
          { title: '下行吞吐量', key: 'dl-throughput' },
        ],
      },
      {
        title: '质量指标',
        key: 'quality',
        children: [
          { title: 'PDCP误包率', key: 'pdcp-loss' },
          { title: '无线可用率', key: 'availability' },
        ],
      },
    ],
  },
];

const mockData: KPIReportRow[] = [
  { id: '1', kpiName: 'RRC建立成功率', kpiCode: 'RRC_SR', unit: '%', currentValue: 99.2, threshold: 95, status: 'normal', trend: [98, 99, 99.5, 98.8, 99.2, 99.1, 99.2], category: 'rrc' },
  { id: '2', kpiName: 'E-RAB建立成功率', kpiCode: 'ERAB_SR', unit: '%', currentValue: 98.7, threshold: 95, status: 'normal', trend: [97, 98, 98.5, 98.7, 98.6, 98.8, 98.7], category: 'erab' },
  { id: '3', kpiName: '切换成功率', kpiCode: 'HO_SR', unit: '%', currentValue: 93.5, threshold: 95, status: 'warning', trend: [96, 95, 94, 93, 93.5, 94, 93.5], category: 'handover' },
  { id: '4', kpiName: '下行峰值吞吐量', kpiCode: 'DL_THROUGHPUT', unit: 'Mbps', currentValue: 145.6, threshold: 100, status: 'normal', trend: [120, 135, 140, 145, 142, 148, 145.6], category: 'dl-throughput' },
  { id: '5', kpiName: '上行峰值吞吐量', kpiCode: 'UL_THROUGHPUT', unit: 'Mbps', currentValue: 45.2, threshold: 30, status: 'normal', trend: [40, 42, 44, 45, 44.5, 46, 45.2], category: 'ul-throughput' },
  { id: '6', kpiName: '最大在线用户数', kpiCode: 'MAX_USERS', unit: '个', currentValue: 856, threshold: 1000, status: 'normal', trend: [800, 820, 840, 856, 850, 860, 856], category: 'user-count' },
  { id: '7', kpiName: '无线可用率', kpiCode: 'AVAILABILITY', unit: '%', currentValue: 99.95, threshold: 99.9, status: 'normal', trend: [99.9, 99.95, 99.92, 99.95, 99.98, 99.95, 99.95], category: 'availability' },
  { id: '8', kpiName: 'PDCP丢包率', kpiCode: 'PDCP_LOSS', unit: '%', currentValue: 0.08, threshold: 0.1, status: 'normal', trend: [0.05, 0.06, 0.07, 0.08, 0.07, 0.09, 0.08], category: 'pdcp-loss' },
];

// Simple sparkline mini-chart using SVG
function Sparkline({ data, color = 'var(--color-primary-600)' }: { data: number[]; color?: string }) {
  if (!data.length) return null;
  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max - min || 1;
  const w = 60;
  const h = 20;
  const pts = data.map((v, i) => `${(i / (data.length - 1)) * w},${h - ((v - min) / range) * h}`).join(' ');
  return (
    <svg width={w} height={h} style={{ display: 'block' }}>
      <polyline
        points={pts}
        fill="none"
        stroke={color}
        strokeWidth={1.5}
      />
    </svg>
  );
}

export default function KPIStandardReport() {
  const t = useT();
  const [selectedCategory, setSelectedCategory] = useState<string>('');
  const [searchValue, setSearchValue] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const filteredSource = selectedCategory && selectedCategory !== 'all'
    ? mockData.filter((r) => r.category === selectedCategory)
    : mockData;

  const STATUS_MAP: Record<string, { color: string; text: string }> = useMemo(() => ({
    normal: { color: 'success', text: t('status.success') },
    warning: { color: 'warning', text: t('alarm.severity.warning') },
    critical: { color: 'error', text: t('alarm.severity.critical') },
  }), [t]);

  const columns: DataTableColumn<KPIReportRow>[] = useMemo(() => [
    { key: 'kpiName', title: t('perf.kpiName'), dataIndex: 'kpiName', width: 180 },
    { key: 'kpiCode', title: t('perf.kpiCode'), dataIndex: 'kpiCode', width: 150, mono: true, copyable: true },
    { key: 'unit', title: t('perf.unit'), dataIndex: 'unit', width: 70 },
    { key: 'currentValue', title: t('perf.value'), dataIndex: 'currentValue', width: 100 },
    { key: 'threshold', title: t('perf.threshold'), dataIndex: 'threshold', width: 90 },
    {
      key: 'status',
      title: t('table.status'),
      dataIndex: 'status',
      width: 90,
      render: (val) => {
        const cfg = STATUS_MAP[val as string] ?? STATUS_MAP.normal;
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    {
      key: 'trend',
      title: t('perf.timeRange'),
      dataIndex: 'trend',
      width: 80,
      render: (val, record) => {
        const arr = val as number[];
        const color = record.status === 'critical' ? '#ff4d4f' : record.status === 'warning' ? '#faad14' : '#52c41a';
        return <Sparkline data={arr} color={color} />;
      },
    },
    {
      key: 'action',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 80,
      fixed: 'right',
      render: () => (
        <Button type="link" size="small" icon={<DownloadOutlined />}>
          {t('common.detail')}
        </Button>
      ),
    },
  ], [t, STATUS_MAP]);

  const treePanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{ padding: '12px 12px 8px' }}>
        <Typography.Text strong style={{ fontSize: 13 }}>{t('perf.category')}</Typography.Text>
      </div>
      <div style={{ padding: '0 12px 8px' }}>
        <Input
          size="small"
          placeholder={t('common.search')}
          prefix={<SearchOutlined />}
          value={searchValue}
          onChange={(e) => setSearchValue(e.target.value)}
          allowClear
        />
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: '0 4px' }}>
        <Tree
          treeData={KPI_TREE_DATA}
          onSelect={(keys) => {
            const key = keys[0] as string;
            setSelectedCategory(key ?? '');
          }}
          defaultExpandAll
          showLine
        />
      </div>
    </div>
  );

  return (
    <TreeListPageLayout tree={treePanel}>
      <div style={{ padding: '12px 16px', borderBottom: '1px solid #f0f0f0', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Typography.Text strong style={{ fontSize: 14 }}>{t('nav.performance.kpiStandard')}</Typography.Text>
        <Space>
          <Button icon={<DownloadOutlined />} onClick={() => void message.info(t('common.exportInProgress'))}>{t('common.export')}</Button>
        </Space>
      </div>
      <div style={{ flex: 1, overflow: 'auto' }}>
        <DataTable<KPIReportRow>
          tableId="kpi-standard-report"
          columns={columns}
          dataSource={filteredSource}
          loading={false}
          rowKey="id"
          total={filteredSource.length}
          currentPage={page}
          pageSize={pageSize}
          onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
          onExport={() => void message.info(t('common.exportInProgress'))}
          scroll={{ x: 900 }}
        />
      </div>
    </TreeListPageLayout>
  );
}
