import { useState, useMemo } from 'react';
import { Tag } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useMRIndicators } from '@core/hooks/api/useMR';
import { useT } from '@/hooks/useT';

interface MRIndicator {
  id: string;
  indicatorName: string;
  indicatorCode: string;
  unit: string;
  description: string;
  category: string;
  status: 'active' | 'inactive';
  mrType: string;
  valueRange: string;
}

const mockIndicators: MRIndicator[] = [
  { id: 'ind-001', indicatorName: 'RSRP', indicatorCode: 'MR_RSRP', unit: 'dBm', description: '参考信号接收功率，反映小区覆盖质量', category: '覆盖质量', status: 'active', mrType: 'MRO', valueRange: '-140 ~ -44' },
  { id: 'ind-002', indicatorName: 'RSRQ', indicatorCode: 'MR_RSRQ', unit: 'dB', description: '参考信号接收质量，反映小区干扰情况', category: '覆盖质量', status: 'active', mrType: 'MRO', valueRange: '-19.5 ~ -3' },
  { id: 'ind-003', indicatorName: 'SINR', indicatorCode: 'MR_SINR', unit: 'dB', description: '信号与干扰加噪声比', category: '干扰', status: 'active', mrType: 'MRO', valueRange: '-23 ~ 40' },
  { id: 'ind-004', indicatorName: 'PCI', indicatorCode: 'MR_PCI', unit: '无', description: '物理小区标识', category: '无线接入', status: 'active', mrType: 'MRO', valueRange: '0 ~ 503' },
  { id: 'ind-005', indicatorName: '切换触发次数', indicatorCode: 'MR_HO_TRIGGER', unit: '次', description: '终端上报的切换触发次数', category: '移动性', status: 'active', mrType: 'MRE', valueRange: '≥0' },
  { id: 'ind-006', indicatorName: '切换成功次数', indicatorCode: 'MR_HO_SUCCESS', unit: '次', description: '切换成功的次数', category: '移动性', status: 'active', mrType: 'MRE', valueRange: '≥0' },
  { id: 'ind-007', indicatorName: 'CQI分布', indicatorCode: 'MR_CQI', unit: '无', description: '信道质量指示分布统计', category: '系统负荷', status: 'active', mrType: 'MRS', valueRange: '0 ~ 15' },
  { id: 'ind-008', indicatorName: 'TA分布', indicatorCode: 'MR_TA', unit: 'Ts', description: '时间提前量分布，反映覆盖半径', category: '覆盖质量', status: 'active', mrType: 'MRS', valueRange: '0 ~ 1282' },
  { id: 'ind-009', indicatorName: '邻区RSRP', indicatorCode: 'MR_NB_RSRP', unit: 'dBm', description: '邻区参考信号接收功率', category: '覆盖质量', status: 'active', mrType: 'MRO', valueRange: '-140 ~ -44' },
  { id: 'ind-010', indicatorName: '上行RSSI', indicatorCode: 'MR_UL_RSSI', unit: 'dBm', description: '上行接收信号强度指示', category: '干扰', status: 'inactive', mrType: 'MRS', valueRange: '-120 ~ 0' },
];

const mrTypeColorMap: Record<string, string> = { MRO: 'blue', MRE: 'green', MRS: 'orange' };

export default function Indicators() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const { data, isLoading, refetch } = useMRIndicators({ page, pageSize });

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('mr.indicatorNameCode'), type: 'input', placeholder: t('mr.indicatorNameCodePlaceholder') },
    {
      name: 'category',
      label: t('mr.category'),
      type: 'select',
      options: [
        { label: t('mr.radioAccess'), value: '无线接入' },
        { label: t('mr.mobility'), value: '移动性' },
        { label: t('mr.coverageQuality'), value: '覆盖质量' },
        { label: t('mr.systemLoad'), value: '系统负荷' },
        { label: t('mr.interference'), value: '干扰' },
      ],
    },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.enabled'), value: 'active' },
        { label: t('status.disabled'), value: 'inactive' },
      ],
    },
  ], [t]);

  const allIndicators = (data?.items?.length ?? 0) > 0
    ? (data?.items ?? []) as unknown as MRIndicator[]
    : mockIndicators.filter((r) => {
        if (filters.keyword) {
          const kw = String(filters.keyword).toLowerCase();
          if (!r.indicatorName.toLowerCase().includes(kw) && !r.indicatorCode.toLowerCase().includes(kw)) return false;
        }
        if (filters.category && r.category !== filters.category) return false;
        if (filters.status && r.status !== filters.status) return false;
        return true;
      });

  const startIndex = (page - 1) * pageSize;
  const paginated = data?.items ? allIndicators : allIndicators.slice(startIndex, startIndex + pageSize);

  const columns: DataTableColumn<MRIndicator & Record<string, unknown>>[] = useMemo(() => [
    { key: 'indicatorName', title: t('mr.indicatorName'), dataIndex: 'indicatorName', width: 140 },
    { key: 'indicatorCode', title: t('mr.indicatorCode'), dataIndex: 'indicatorCode', width: 160, render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span> },
    { key: 'unit', title: t('mr.unit'), dataIndex: 'unit', width: 70 },
    { key: 'valueRange', title: t('mr.valueRange'), dataIndex: 'valueRange', width: 120 },
    { key: 'description', title: t('table.description'), dataIndex: 'description', ellipsis: true },
    { key: 'category', title: t('mr.category'), dataIndex: 'category', width: 100 },
    {
      key: 'mrType', title: t('mr.mrType'), dataIndex: 'mrType', width: 90,
      render: (val) => <Tag color={mrTypeColorMap[String(val)] ?? 'default'}>{String(val)}</Tag>,
    },
    {
      key: 'status', title: t('table.status'), dataIndex: 'status', width: 80,
      render: (val) => <Tag color={val === 'active' ? 'green' : 'default'}>{val === 'active' ? t('status.enabled') : t('status.disabled')}</Tag>,
    },
  ], [t]);

  return (
    <ListPageLayout title={t('nav.mr.indicators')} subtitle={t('mr.indicatorsSubtitle')}>
      <FilterBar
        filterId="mr-indicators-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable
        tableId="mr-indicators-list"
        columns={columns}
        dataSource={paginated as (MRIndicator & Record<string, unknown>)[]}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? allIndicators.length}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1000 }}
      />
    </ListPageLayout>
  );
}
