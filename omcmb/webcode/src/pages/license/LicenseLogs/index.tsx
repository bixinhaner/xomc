import { useState, useMemo } from 'react';
import { Tag } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

type LicenseOpType = 'import' | 'activate' | 'revoke' | 'query' | 'renew';
type LicenseLogResult = 'success' | 'failed';

interface LicenseLog {
  id: string;
  operationTime: string;
  operationType: LicenseOpType;
  licenseId: string;
  deviceSn?: string;
  operator: string;
  result: LicenseLogResult;
  detail?: string;
  clientIp: string;
}

const mockLicenseLogs: LicenseLog[] = [
  { id: 'll-001', operationTime: '2024-06-01T09:00:00.000Z', operationType: 'import', licenseId: 'OMC-BASIC-HB-2024-001', operator: 'admin', result: 'success', detail: '成功导入许可证文件', clientIp: '192.168.1.100' },
  { id: 'll-002', operationTime: '2024-06-01T09:05:00.000Z', operationType: 'activate', licenseId: 'OMC-BASIC-HB-2024-001', deviceSn: 'ENB00001', operator: 'admin', result: 'success', detail: '许可证已激活并绑定设备', clientIp: '192.168.1.100' },
  { id: 'll-003', operationTime: '2024-06-01T10:00:00.000Z', operationType: 'query', licenseId: 'OMC-ADV-HD-2024-001', operator: 'operator1', result: 'success', detail: '查询许可证详情', clientIp: '192.168.1.102' },
  { id: 'll-004', operationTime: '2024-05-15T14:00:00.000Z', operationType: 'import', licenseId: 'OMC-ADV-HD-2024-001', operator: 'admin', result: 'success', detail: '成功导入高级版许可证', clientIp: '192.168.1.100' },
  { id: 'll-005', operationTime: '2024-05-15T14:10:00.000Z', operationType: 'activate', licenseId: 'OMC-ADV-HD-2024-001', deviceSn: 'GNB00001', operator: 'admin', result: 'success', clientIp: '192.168.1.100' },
  { id: 'll-006', operationTime: '2024-04-01T10:00:00.000Z', operationType: 'import', licenseId: 'OMC-TRIAL-2024-001', operator: 'operator1', result: 'failed', detail: 'License文件校验失败：签名无效', clientIp: '10.0.0.50' },
  { id: 'll-007', operationTime: '2024-03-15T16:30:00.000Z', operationType: 'renew', licenseId: 'OMC-ADV-HD-2024-001', operator: 'admin', result: 'success', detail: '订阅许可证续期1年', clientIp: '192.168.1.100' },
  { id: 'll-008', operationTime: '2024-03-01T16:00:00.000Z', operationType: 'revoke', licenseId: 'OMC-OLD-2023-001', deviceSn: 'ENB00010', operator: 'admin', result: 'success', detail: '旧版许可证已撤销', clientIp: '192.168.1.100' },
  { id: 'll-009', operationTime: '2024-02-28T09:00:00.000Z', operationType: 'activate', licenseId: 'OMC-5G-HS-2024-001', deviceSn: 'GNB00002', operator: 'admin', result: 'failed', detail: '设备不在许可区域内', clientIp: '192.168.1.100' },
  { id: 'll-010', operationTime: '2024-02-20T11:00:00.000Z', operationType: 'import', licenseId: 'OMC-5G-HS-2024-001', operator: 'admin', result: 'success', detail: '5G许可证导入成功', clientIp: '192.168.1.100' },
];

const opTypeColorMap: Record<LicenseOpType, string> = {
  import: 'blue',
  activate: 'green',
  revoke: 'orange',
  query: 'default',
  renew: 'cyan',
};

export default function LicenseLogs() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const opTypeLabelMap: Record<LicenseOpType, string> = useMemo(() => ({
    import: t('common.import'),
    activate: t('license.activate'),
    revoke: t('license.revoke'),
    query: t('license.query'),
    renew: t('license.renew'),
  }), [t]);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('license.keyword'), type: 'input', placeholder: t('license.keywordPlaceholder') },
    { name: 'timeRange', label: t('license.timeRange'), type: 'date-range' },
    {
      name: 'operationType',
      label: t('license.operationType'),
      type: 'select',
      options: [
        { label: t('common.import'), value: 'import' },
        { label: t('license.activate'), value: 'activate' },
        { label: t('license.revoke'), value: 'revoke' },
        { label: t('license.query'), value: 'query' },
        { label: t('license.renew'), value: 'renew' },
      ],
    },
    {
      name: 'result',
      label: t('table.result'),
      type: 'select',
      options: [
        { label: t('status.success'), value: 'success' },
        { label: t('status.failed'), value: 'failed' },
      ],
    },
  ], [t]);

  const filtered = mockLicenseLogs.filter((log) => {
    if (filters.keyword) {
      const kw = String(filters.keyword).toLowerCase();
      if (!log.licenseId.toLowerCase().includes(kw) && !log.operator.toLowerCase().includes(kw)) return false;
    }
    if (filters.operationType && log.operationType !== filters.operationType) return false;
    if (filters.result && log.result !== filters.result) return false;
    return true;
  });

  const startIndex = (page - 1) * pageSize;
  const paginated = filtered.slice(startIndex, startIndex + pageSize);

  const columns: DataTableColumn<LicenseLog & Record<string, unknown>>[] = useMemo(() => [
    {
      key: 'operationTime', title: t('table.time'), dataIndex: 'operationTime', width: 170,
      render: (val) => new Date(String(val)).toLocaleString('zh-CN'),
    },
    {
      key: 'operationType', title: t('license.operationType'), dataIndex: 'operationType', width: 100,
      render: (val) => {
        const tp = val as LicenseOpType;
        return <Tag color={opTypeColorMap[tp]}>{opTypeLabelMap[tp]}</Tag>;
      },
    },
    {
      key: 'licenseId', title: 'License ID', dataIndex: 'licenseId', width: 220,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>,
    },
    {
      key: 'deviceSn', title: t('device.sn'), dataIndex: 'deviceSn', width: 120,
      render: (val) => val ? <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span> : '—',
    },
    { key: 'operator', title: t('table.operator'), dataIndex: 'operator', width: 100 },
    {
      key: 'result', title: t('table.result'), dataIndex: 'result', width: 90,
      render: (val) => {
        const r = val as LicenseLogResult;
        return <Tag color={r === 'success' ? 'green' : 'red'}>{r === 'success' ? t('status.success') : t('status.failed')}</Tag>;
      },
    },
    {
      key: 'detail', title: t('common.detail'), dataIndex: 'detail', ellipsis: true,
      render: (val) => val ? String(val) : '—',
    },
    {
      key: 'clientIp', title: t('license.clientIp'), dataIndex: 'clientIp', width: 140,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>,
    },
  ], [t, opTypeLabelMap]);

  return (
    <ListPageLayout title={t('nav.license.logs')} subtitle={t('license.logsSubtitle')}>
      <FilterBar
        filterId="license-logs-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable
        tableId="license-logs-list"
        columns={columns}
        dataSource={paginated as (LicenseLog & Record<string, unknown>)[]}
        loading={false}
        rowKey="id"
        total={filtered.length}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => setFilters({ ...filters })}
        onExport={(format) => void console.log(t('common.export'), format)}
        scroll={{ x: 1100 }}
        alarmRowStyle={(record) => {
          const log = record as LicenseLog;
          return log.result === 'failed' ? 'major' : null;
        }}
      />
    </ListPageLayout>
  );
}
