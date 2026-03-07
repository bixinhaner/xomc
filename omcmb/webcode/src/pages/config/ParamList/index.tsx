import { useState, useMemo } from 'react';
import { Tag, message } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useConfigParams } from '@/hooks/api/useConfig';
import { useT } from '@/hooks/useT';

interface ParamListRow extends Record<string, unknown> {
  id: string;
  paramCode: string;
  paramName: string;
  paramType: string;
  defaultValue: string;
  description: string;
  category: string;
  unit: string;
  readonly: boolean;
}

const PARAM_TYPE_COLORS: Record<string, string> = {
  string: 'blue',
  number: 'green',
  boolean: 'orange',
  enum: 'purple',
  ipAddress: 'cyan',
  range: 'geekblue',
};

const CATEGORY_OPTIONS = [
  { label: '射频参数', value: 'rf' },
  { label: '无线资源管理', value: 'rrm' },
  { label: '小区参数', value: 'cell' },
  { label: '传输参数', value: 'transport' },
  { label: '系统参数', value: 'system' },
];

const PARAM_TYPE_OPTIONS = [
  { label: 'string', value: 'string' },
  { label: 'number', value: 'number' },
  { label: 'boolean', value: 'boolean' },
  { label: 'enum', value: 'enum' },
  { label: 'ipAddress', value: 'ipAddress' },
  { label: 'range', value: 'range' },
];

const mockData: ParamListRow[] = [
  { id: '1', paramCode: 'TX_POWER', paramName: '发射功率', paramType: 'number', defaultValue: '43', description: '基站射频发射功率', category: 'rf', unit: 'dBm', readonly: false },
  { id: '2', paramCode: 'RS_POWER', paramName: '参考信号功率', paramType: 'number', defaultValue: '0', description: 'PDSCH参考信号功率偏置', category: 'rf', unit: 'dB', readonly: false },
  { id: '3', paramCode: 'CELL_BW', paramName: '小区带宽', paramType: 'enum', defaultValue: '20MHz', description: '系统带宽配置', category: 'cell', unit: 'MHz', readonly: false },
  { id: '4', paramCode: 'PCI', paramName: '物理小区标识', paramType: 'number', defaultValue: '0', description: '物理小区ID，范围0-503', category: 'cell', unit: '', readonly: false },
  { id: '5', paramCode: 'TAC', paramName: '跟踪区码', paramType: 'number', defaultValue: '1', description: '跟踪区域码', category: 'cell', unit: '', readonly: false },
  { id: '6', paramCode: 'EARFCN_DL', paramName: '下行频点', paramType: 'number', defaultValue: '1575', description: 'E-UTRAN绝对下行频点', category: 'rf', unit: '', readonly: false },
  { id: '7', paramCode: 'HO_THRESHOLD', paramName: '切换门限', paramType: 'number', defaultValue: '-110', description: '同频切换触发门限', category: 'rrm', unit: 'dBm', readonly: false },
  { id: '8', paramCode: 'MGT_IP', paramName: '管理IP地址', paramType: 'ipAddress', defaultValue: '192.168.1.1', description: '基站管理平面IP地址', category: 'transport', unit: '', readonly: true },
];

export default function ParamList() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const { data, isLoading, refetch } = useConfigParams({
    keyword: filters.keyword as string,
    category: filters.category as string,
    page,
    pageSize,
  });

  const tableSource = (data?.items ?? mockData) as unknown as ParamListRow[];

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('common.search'), type: 'input', placeholder: t('common.placeholder') },
    { name: 'paramType', label: t('config.paramType'), type: 'select', options: PARAM_TYPE_OPTIONS },
    { name: 'category', label: t('perf.category'), type: 'select', options: CATEGORY_OPTIONS },
  ], [t]);

  const columns: DataTableColumn<ParamListRow>[] = useMemo(() => [
    { key: 'paramCode', title: t('config.paramCode'), dataIndex: 'paramCode', width: 180, mono: true, copyable: true },
    { key: 'paramName', title: t('config.paramName'), dataIndex: 'paramName', width: 160 },
    {
      key: 'paramType',
      title: t('config.paramType'),
      dataIndex: 'paramType',
      width: 110,
      render: (val) => (
        <Tag color={PARAM_TYPE_COLORS[val as string] ?? 'default'}>{val as string}</Tag>
      ),
    },
    { key: 'defaultValue', title: t('config.defaultValue'), dataIndex: 'defaultValue', width: 110 },
    { key: 'unit', title: t('perf.unit'), dataIndex: 'unit', width: 80 },
    { key: 'description', title: t('table.description'), dataIndex: 'description', width: 260, ellipsis: true },
    {
      key: 'category',
      title: t('perf.category'),
      dataIndex: 'category',
      width: 120,
      render: (val) => {
        const opt = CATEGORY_OPTIONS.find((o) => o.value === val);
        return opt ? opt.label : (val as string);
      },
    },
    {
      key: 'readonly',
      title: t('config.readonly'),
      dataIndex: 'readonly',
      width: 80,
      render: (val) => <Tag color={val ? 'orange' : 'default'}>{val ? t('config.readonly') : t('common.edit')}</Tag>,
    },
  ], [t]);

  return (
    <ListPageLayout title={t('nav.config.paramList')}>
      <FilterBar
        filterId="param-list"
        fields={filterFields}
        onSearch={(vals) => setFilters(vals)}
        onReset={() => setFilters({})}
      />
      <DataTable<ParamListRow>
        tableId="param-list"
        columns={columns}
        dataSource={tableSource}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? tableSource.length}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        onExport={() => void message.info(t('common.exportInProgress'))}
        scroll={{ x: 1100 }}
      />
    </ListPageLayout>
  );
}
