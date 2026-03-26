import { useState, useMemo } from 'react';
import { Tag, Button, Modal } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useSystemLogs } from '@/hooks/api/useLogs';
import type { SystemLog } from '@/mock/data/logs';
import { useT } from '@/hooks/useT';

type LogLevel = 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';

interface SystemLogDetail extends SystemLog {
  stackTrace?: string;
}

const mockSystemLogs: SystemLogDetail[] = [
  { id: 'sl-001', timestamp: '2024-06-01T08:00:00.001Z', source: 'auth', level: 'INFO', message: '用户 admin 登录成功，来自 IP 10.0.0.1' },
  { id: 'sl-002', timestamp: '2024-06-01T08:00:01.234Z', source: 'device', level: 'INFO', message: '设备 ENB00001 注册成功，版本: V100R011C10SPC200' },
  { id: 'sl-003', timestamp: '2024-06-01T08:01:00.567Z', source: 'alarm', level: 'WARN', message: '收到 Major 级告警 ALM-001，设备: ENB00001' },
  { id: 'sl-004', timestamp: '2024-06-01T08:01:30.890Z', source: 'database', level: 'ERROR', message: '数据库连接池耗尽，当前连接数: 100/100', stackTrace: 'java.sql.SQLException: Connection pool exhausted\n\tat com.omc.db.pool.ConnectionPool.getConnection(ConnectionPool.java:156)\n\tat com.omc.db.service.DatabaseService.query(DatabaseService.java:89)\n\tat com.omc.alarm.service.AlarmService.saveAlarm(AlarmService.java:234)', details: '连接池大小: 100，当前等待队列: 45' },
  { id: 'sl-005', timestamp: '2024-06-01T08:02:00.123Z', source: 'device', level: 'DEBUG', message: 'ENB00001 心跳处理完成，延迟 150ms' },
  { id: 'sl-006', timestamp: '2024-06-01T08:03:00.456Z', source: 'scheduler', level: 'INFO', message: '性能采集任务 perf-task-001 开始执行，目标设备 25 台' },
  { id: 'sl-007', timestamp: '2024-06-01T08:05:00.789Z', source: 'file', level: 'WARN', message: '存储空间使用率达到 80%，当前使用: 800GB/1TB' },
  { id: 'sl-008', timestamp: '2024-06-01T08:07:30.012Z', source: 'gateway', level: 'ERROR', message: 'API限流触发，来自 10.5.0.200 的请求被拒绝，1分钟内已超过100次', stackTrace: 'com.omc.gateway.RateLimitException: Rate limit exceeded\n\tat com.omc.gateway.filter.RateLimitFilter.filter(RateLimitFilter.java:67)\n\tat org.springframework.cloud.gateway.filter.GatewayFilter.filter(GatewayFilter.java:55)' },
  { id: 'sl-009', timestamp: '2024-06-01T08:10:00.345Z', source: 'alarm', level: 'ERROR', message: '告警服务连接异常，内存使用率过高', stackTrace: 'java.lang.OutOfMemoryError: Java heap space\n\tat java.util.Arrays.copyOf(Arrays.java:3210)\n\tat java.util.ArrayList.grow(ArrayList.java:265)\n\tat com.omc.alarm.processor.AlarmProcessor.processAlarms(AlarmProcessor.java:445)' },
  { id: 'sl-010', timestamp: '2024-06-01T08:12:00.678Z', source: 'performance', level: 'INFO', message: '性能采集任务完成: 成功 24 台，失败 1 台' },
];

const levelStyleMap: Record<LogLevel, { color: string; bold?: boolean }> = {
  DEBUG: { color: 'default' },
  INFO: { color: 'blue' },
  WARN: { color: 'orange' },
  ERROR: { color: 'red' },
};

export default function SystemLogPage() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedLog, setSelectedLog] = useState<SystemLogDetail | null>(null);

  const { data, isLoading, refetch } = useSystemLogs({
    level: filters.level as SystemLog['level'] | undefined,
    source: filters.source as string | undefined,
    keyword: filters.keyword as string | undefined,
    page,
    pageSize,
  });

  const allLogs = (data?.items?.length ?? 0) > 0
    ? (data?.items ?? []) as unknown as SystemLogDetail[]
    : mockSystemLogs.filter((log) => {
        if (filters.level && log.level !== filters.level) return false;
        if (filters.source && log.source !== filters.source) return false;
        if (filters.keyword && !log.message.toLowerCase().includes(String(filters.keyword).toLowerCase())) return false;
        return true;
      });

  const startIndex = (page - 1) * pageSize;
  const paginated = data?.items ? allLogs : allLogs.slice(startIndex, startIndex + pageSize);

  const filterFields: FilterField[] = useMemo(() => [
    {
      name: 'source',
      label: t('table.type'),
      type: 'select',
      options: [
        { label: '认证模块', value: 'auth' },
        { label: '设备管理', value: 'device' },
        { label: '告警服务', value: 'alarm' },
        { label: '性能服务', value: 'performance' },
        { label: '文件服务', value: 'file' },
        { label: '调度服务', value: 'scheduler' },
        { label: '数据库', value: 'database' },
        { label: 'API网关', value: 'gateway' },
      ],
    },
    {
      name: 'level',
      label: t('alarm.severity'),
      type: 'select',
      options: [
        { label: 'DEBUG', value: 'DEBUG' },
        { label: 'INFO', value: 'INFO' },
        { label: 'WARN', value: 'WARN' },
        { label: 'ERROR', value: 'ERROR' },
      ],
    },
    { name: 'keyword', label: t('common.search'), type: 'input', placeholder: t('common.search') },
  ], [t]);

  const columns: DataTableColumn<SystemLogDetail & Record<string, unknown>>[] = useMemo(() => [
    {
      key: 'timestamp',
      title: t('table.time'),
      dataIndex: 'timestamp',
      width: 210,
      render: (val) => {
        const d = new Date(String(val));
        return <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{d.toLocaleString('zh-CN')}.{d.getMilliseconds().toString().padStart(3, '0')}</span>;
      },
    },
    {
      key: 'source',
      title: t('table.type'),
      dataIndex: 'source',
      width: 110,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12, color: '#595959' }}>{String(val)}</span>,
    },
    {
      key: 'level',
      title: t('alarm.severity'),
      dataIndex: 'level',
      width: 90,
      render: (val) => {
        const level = val as LogLevel;
        const style = levelStyleMap[level] ?? { color: 'default' };
        return (
          <Tag color={style.color}>
            {String(val).toUpperCase()}
          </Tag>
        );
      },
    },
    {
      key: 'message',
      title: t('table.description'),
      dataIndex: 'message',
      ellipsis: true,
      render: (val) => (
        <span style={{
          fontSize: 13,
          fontFamily: 'monospace',
          color: '#262626',
          display: 'block',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
        }}>
          {String(val)}
        </span>
      ),
    },
    {
      key: 'detail',
      title: t('common.detail'),
      dataIndex: 'id',
      width: 70,
      fixed: 'right',
      render: (_, record) => {
        const log = record as SystemLogDetail;
        const hasDetail = log.level === 'error' || log.level === 'fatal' || Boolean(log.stackTrace);
        return hasDetail ? (
          <Button type="link" size="small"
            onClick={() => { setSelectedLog(log); setDetailVisible(true); }}>
            {t('common.detail')}
          </Button>
        ) : null;
      },
    },
  ], [t]);

  return (
    <ListPageLayout title={t('nav.log.system')} subtitle={t('nav.log.system')}>
      <FilterBar
        filterId="system-log-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable
        tableId="system-log-list"
        columns={columns}
        dataSource={paginated as (SystemLogDetail & Record<string, unknown>)[]}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? allLogs.length}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1000 }}
        alarmRowStyle={(record) => {
          const log = record as SystemLogDetail;
          if (log.level === 'ERROR') return 'major';
          if (log.level === 'WARN') return 'warning';
          return null;
        }}
      />

      <Modal
        title={t('common.detail')}
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={null}
        width={700}
      >
        {selectedLog && (
          <div>
            <div style={{ marginBottom: 12 }}>
              <Tag color={levelStyleMap[selectedLog.level as LogLevel]?.color ?? 'default'}>
                {selectedLog.level.toUpperCase()}
              </Tag>
              <span style={{ marginLeft: 8, fontFamily: 'monospace', fontSize: 12, color: '#666' }}>{selectedLog.source}</span>
              <span style={{ float: 'right', fontSize: 12, color: '#999' }}>
                {new Date(selectedLog.timestamp).toLocaleString('zh-CN')}
              </span>
            </div>
            <div style={{
              background: '#1a1a1a',
              color: '#39d353',
              fontFamily: 'monospace',
              fontSize: 13,
              padding: 16,
              borderRadius: 4,
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-all',
              maxHeight: 400,
              overflow: 'auto',
            }}>
              {selectedLog.message}
              {selectedLog.stackTrace && (
                <>
                  {'\n\n'}
                  <span style={{ color: '#ff6b6b' }}>{selectedLog.stackTrace}</span>
                </>
              )}
            </div>
          </div>
        )}
      </Modal>
    </ListPageLayout>
  );
}
