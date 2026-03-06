import { useState, useMemo } from 'react';
import { Tag } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

type AlarmOperationType = 'confirm' | 'clear' | 'sync' | 'suppress' | 'unsuppress';

interface AlarmLogRecord {
  id: string;
  operationTime: string;
  operator: string;
  operationType: AlarmOperationType;
  alarmCode: string;
  alarmName: string;
  deviceSn: string;
  severity: 'critical' | 'major' | 'minor' | 'warning';
  remark?: string;
}

const mockAlarmLogs: AlarmLogRecord[] = [
  { id: 'al-001', operationTime: '2024-06-01T09:00:00.000Z', operator: 'admin', operationType: 'confirm', alarmCode: 'ALM-001', alarmName: '板卡温度过高', deviceSn: 'ENB00001', severity: 'major', remark: '已通知运维人员' },
  { id: 'al-002', operationTime: '2024-06-01T09:30:00.000Z', operator: 'operator1', operationType: 'clear', alarmCode: 'ALM-001', alarmName: '板卡温度过高', deviceSn: 'ENB00001', severity: 'major', remark: '设备已恢复正常' },
  { id: 'al-003', operationTime: '2024-06-01T10:00:00.000Z', operator: 'admin', operationType: 'sync', alarmCode: 'ALL', alarmName: '全量同步', deviceSn: 'ENB00002', severity: 'minor' },
  { id: 'al-004', operationTime: '2024-06-01T10:15:00.000Z', operator: 'operator2', operationType: 'confirm', alarmCode: 'ALM-055', alarmName: 'GPS时钟丢失', deviceSn: 'GNB00001', severity: 'critical', remark: '已上报紧急处理' },
  { id: 'al-005', operationTime: '2024-06-01T11:00:00.000Z', operator: 'admin', operationType: 'suppress', alarmCode: 'ALM-100', alarmName: '主备切换告警', deviceSn: 'ENB00003', severity: 'warning', remark: '计划维护期间屏蔽' },
  { id: 'al-006', operationTime: '2024-06-01T12:00:00.000Z', operator: 'admin', operationType: 'unsuppress', alarmCode: 'ALM-100', alarmName: '主备切换告警', deviceSn: 'ENB00003', severity: 'warning', remark: '维护结束' },
  { id: 'al-007', operationTime: '2024-06-01T13:30:00.000Z', operator: 'operator1', operationType: 'clear', alarmCode: 'ALM-055', alarmName: 'GPS时钟丢失', deviceSn: 'GNB00001', severity: 'critical', remark: 'GPS信号恢复' },
];

const opTypeColorMap: Record<AlarmOperationType, string> = {
  confirm: 'blue',
  clear: 'green',
  sync: 'cyan',
  suppress: 'orange',
  unsuppress: 'default',
};

const opTypeLabelMap: Record<AlarmOperationType, string> = {
  confirm: '确认',
  clear: '清除',
  sync: '同步',
  suppress: '屏蔽',
  unsuppress: '解除屏蔽',
};

const severityColorMap: Record<string, string> = {
  critical: 'red',
  major: 'orange',
  minor: 'yellow',
  warning: 'gold',
};

export default function AlarmLog() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const filtered = mockAlarmLogs.filter((r) => {
    if (filters.keyword) {
      const kw = String(filters.keyword).toLowerCase();
      if (!r.alarmCode.toLowerCase().includes(kw) && !r.alarmName.toLowerCase().includes(kw) && !r.deviceSn.toLowerCase().includes(kw)) return false;
    }
    if (filters.operationType && r.operationType !== filters.operationType) return false;
    if (filters.operator && !r.operator.includes(String(filters.operator))) return false;
    return true;
  });

  const startIndex = (page - 1) * pageSize;
  const paginated = filtered.slice(startIndex, startIndex + pageSize);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('common.search'), type: 'input', placeholder: t('common.search') },
    { name: 'timeRange', label: t('perf.timeRange'), type: 'date-range' },
    {
      name: 'operationType',
      label: t('table.type'),
      type: 'select',
      options: [
        { label: '确认', value: 'confirm' },
        { label: '清除', value: 'clear' },
        { label: '同步', value: 'sync' },
        { label: '屏蔽', value: 'suppress' },
        { label: '解除屏蔽', value: 'unsuppress' },
      ],
    },
    { name: 'operator', label: t('table.operator'), type: 'input', placeholder: t('table.operator') },
  ], [t]);

  const columns: DataTableColumn<AlarmLogRecord & Record<string, unknown>>[] = useMemo(() => [
    {
      key: 'operationTime',
      title: t('table.time'),
      dataIndex: 'operationTime',
      width: 180,
      render: (val) => new Date(String(val)).toLocaleString('zh-CN'),
    },
    { key: 'operator', title: t('table.operator'), dataIndex: 'operator', width: 100 },
    {
      key: 'operationType',
      title: t('table.type'),
      dataIndex: 'operationType',
      width: 110,
      render: (val) => {
        const tp = val as AlarmOperationType;
        return <Tag color={opTypeColorMap[tp]}>{opTypeLabelMap[tp]}</Tag>;
      },
    },
    {
      key: 'alarmCode',
      title: t('alarm.code'),
      dataIndex: 'alarmCode',
      width: 100,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>,
    },
    { key: 'alarmName', title: t('alarm.name'), dataIndex: 'alarmName', width: 160, ellipsis: true },
    {
      key: 'deviceSn',
      title: t('device.sn'),
      dataIndex: 'deviceSn',
      width: 120,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>,
    },
    {
      key: 'severity',
      title: t('alarm.severity'),
      dataIndex: 'severity',
      width: 100,
      render: (val) => <Tag color={severityColorMap[String(val)] ?? 'default'}>{t(`alarm.severity.${String(val)}`)}</Tag>,
    },
    { key: 'remark', title: t('table.description'), dataIndex: 'remark', ellipsis: true, render: (val) => val ? String(val) : '—' },
  ], [t]);

  return (
    <ListPageLayout title={t('nav.log.alarm')} subtitle={t('nav.log.alarm')}>
      <FilterBar
        filterId="alarm-log-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable
        tableId="alarm-log-list"
        columns={columns}
        dataSource={paginated as (AlarmLogRecord & Record<string, unknown>)[]}
        loading={false}
        rowKey="id"
        total={filtered.length}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => setFilters({ ...filters })}
        scroll={{ x: 1000 }}
      />
    </ListPageLayout>
  );
}
