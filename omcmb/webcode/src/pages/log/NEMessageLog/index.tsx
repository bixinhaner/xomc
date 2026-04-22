import { useState, useMemo } from 'react';
import { Tag, Tooltip, Modal, Button } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import JSONViewer from '@/components/JSONViewer';
import { useNEMessageLogs } from '@core/hooks/api/useLogs';
import { useT } from '@/hooks/useT';

const messageTypeColorMap: Record<string, string> = {
  register: 'blue',
  heartbeat: 'default',
  alarm: 'red',
  performance: 'green',
  config_response: 'cyan',
  version_report: 'purple',
};

const messageTypeLabelMap: Record<string, string> = {
  register: '注册请求',
  heartbeat: '心跳',
  alarm: '告警上报',
  performance: '性能上报',
  config_response: '配置响应',
  version_report: '版本上报',
};

interface NEMessageRecord {
  id: string;
  timestamp: string;
  messageType: string;
  deviceSn: string;
  deviceName: string;
  requestIp: string;
  requestContent: string;
  responseContent: string;
  requestJson?: Record<string, unknown>;
  responseJson?: Record<string, unknown>;
}

const mockNEMessages: NEMessageRecord[] = [
  {
    id: 'nem-001',
    timestamp: '2024-06-01T08:00:00.123Z',
    messageType: 'register',
    deviceSn: 'ENB00001',
    deviceName: '北京-eNB-0001',
    requestIp: '10.1.0.101',
    requestContent: '{"msgType":"register","sn":"ENB00001","swVersion":"V100R011C10SPC200"}',
    responseContent: '{"code":0,"msg":"success","token":"eyJhbGciOiJIUzI1NiJ9"}',
    requestJson: { msgType: 'register', sn: 'ENB00001', swVersion: 'V100R011C10SPC200', hwVersion: 'BBU3900V5', ip: '10.1.0.101' },
    responseJson: { code: 0, msg: 'success', token: 'eyJhbGciOiJIUzI1NiJ9...', sessionExpiry: 3600 },
  },
  {
    id: 'nem-002',
    timestamp: '2024-06-01T08:01:00.456Z',
    messageType: 'alarm',
    deviceSn: 'ENB00001',
    deviceName: '北京-eNB-0001',
    requestIp: '10.1.0.101',
    requestContent: '{"msgType":"alarm","alarmCode":"ALM-001","severity":"major","description":"Board temperature high"}',
    responseContent: '{"code":0,"msg":"alarm received"}',
    requestJson: { msgType: 'alarm', alarmCode: 'ALM-001', severity: 'major', description: 'Board temperature high', timestamp: '2024-06-01T08:00:55.000Z' },
    responseJson: { code: 0, msg: 'alarm received', alarmId: 'ai-001' },
  },
  {
    id: 'nem-003',
    timestamp: '2024-06-01T08:02:00.789Z',
    messageType: 'heartbeat',
    deviceSn: 'GNB00001',
    deviceName: '北京-gNB-0001',
    requestIp: '10.1.1.101',
    requestContent: '{"msgType":"heartbeat","sn":"GNB00001","status":"normal"}',
    responseContent: '{"code":0,"msg":"ok"}',
    requestJson: { msgType: 'heartbeat', sn: 'GNB00001', status: 'normal', cpuUsage: 35, memUsage: 62 },
    responseJson: { code: 0, msg: 'ok', serverTime: '2024-06-01T08:02:00.789Z' },
  },
  {
    id: 'nem-004',
    timestamp: '2024-06-01T08:03:12.100Z',
    messageType: 'performance',
    deviceSn: 'ENB00002',
    deviceName: '北京-eNB-0002',
    requestIp: '10.1.0.102',
    requestContent: '{"msgType":"performance","sn":"ENB00002","counters":{"prb_utilization":45.2,"call_drop_rate":0.001}}',
    responseContent: '{"code":0,"msg":"perf data received"}',
    requestJson: { msgType: 'performance', sn: 'ENB00002', period: '15min', counters: { prb_utilization: 45.2, call_drop_rate: 0.001, handover_success_rate: 99.8 } },
    responseJson: { code: 0, msg: 'perf data received', recordId: 'perf-20240601-0002' },
  },
];

function truncate(str: string, maxLen = 60): string {
  if (str.length <= maxLen) return str;
  return str.substring(0, maxLen) + '...';
}

export default function NEMessageLog() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [jsonModalVisible, setJsonModalVisible] = useState(false);
  const [jsonModalData, setJsonModalData] = useState<{ title: string; request?: Record<string, unknown>; response?: Record<string, unknown> } | null>(null);

  const { data, isLoading, refetch } = useNEMessageLogs({
    deviceSn: filters.deviceSn as string | undefined,
    messageType: filters.messageType as string | undefined,
    page,
    pageSize,
  });

  const allMessages = (data?.items?.length ?? 0) > 0
    ? (data?.items ?? []) as unknown as NEMessageRecord[]
    : mockNEMessages;

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'deviceSn', label: t('device.sn'), type: 'input', placeholder: t('device.sn') },
    {
      name: 'messageType',
      label: t('table.type'),
      type: 'select',
      options: [
        { label: '注册请求', value: 'register' },
        { label: '心跳', value: 'heartbeat' },
        { label: '告警上报', value: 'alarm' },
        { label: '性能上报', value: 'performance' },
        { label: '配置响应', value: 'config_response' },
        { label: '版本上报', value: 'version_report' },
      ],
    },
    { name: 'keyword', label: t('common.search'), type: 'input', placeholder: t('common.search') },
  ], [t]);

  const columns: DataTableColumn<NEMessageRecord & Record<string, unknown>>[] = useMemo(() => [
    {
      key: 'timestamp',
      title: t('table.time'),
      dataIndex: 'timestamp',
      width: 190,
      render: (val) => {
        const d = new Date(String(val));
        return <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{d.toLocaleString('zh-CN')}.{d.getMilliseconds().toString().padStart(3, '0')}</span>;
      },
    },
    {
      key: 'messageType',
      title: t('table.type'),
      dataIndex: 'messageType',
      width: 110,
      render: (val) => {
        const s = String(val);
        return <Tag color={messageTypeColorMap[s] ?? 'default'}>{messageTypeLabelMap[s] ?? s}</Tag>;
      },
    },
    {
      key: 'deviceSn',
      title: t('device.sn'),
      dataIndex: 'deviceSn',
      width: 120,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>,
    },
    { key: 'deviceName', title: t('device.name'), dataIndex: 'deviceName', width: 160, ellipsis: true },
    {
      key: 'requestIp',
      title: t('table.ip'),
      dataIndex: 'requestIp',
      width: 130,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>,
    },
    {
      key: 'requestContent',
      title: t('common.detail'),
      dataIndex: 'requestContent',
      ellipsis: true,
      render: (val) => (
        <Tooltip title={String(val)}>
          <span style={{ fontSize: 12, color: '#595959' }}>{truncate(String(val), 50)}</span>
        </Tooltip>
      ),
    },
    {
      key: 'responseContent',
      title: t('table.result'),
      dataIndex: 'responseContent',
      ellipsis: true,
      render: (val) => (
        <Tooltip title={String(val)}>
          <span style={{ fontSize: 12, color: '#595959' }}>{truncate(String(val), 40)}</span>
        </Tooltip>
      ),
    },
    {
      key: 'actions',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 80,
      fixed: 'right',
      render: (_, record) => {
        const r = record as NEMessageRecord;
        return (
          <Button
            type="link"
            size="small"
            onClick={() => {
              setJsonModalData({
                title: `${r.deviceSn} - ${messageTypeLabelMap[r.messageType] ?? r.messageType}`,
                request: r.requestJson,
                response: r.responseJson,
              });
              setJsonModalVisible(true);
            }}
          >
            {t('common.detail')}
          </Button>
        );
      },
    },
  ], [t]);

  return (
    <ListPageLayout title={t('nav.log.neMessage')} subtitle={t('nav.log.neMessage')}>
      <FilterBar
        filterId="ne-message-log-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable
        tableId="ne-message-log-list"
        columns={columns}
        dataSource={allMessages as (NEMessageRecord & Record<string, unknown>)[]}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? allMessages.length}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        expandable={{
          expandedRowRender: (record) => {
            const r = record as NEMessageRecord;
            return (
              <div style={{ display: 'flex', gap: 16 }}>
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: 12, color: '#666', marginBottom: 4 }}>{t('common.detail')}:</div>
                  <JSONViewer data={r.requestJson ?? {}} />
                </div>
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: 12, color: '#666', marginBottom: 4 }}>{t('table.result')}:</div>
                  <JSONViewer data={r.responseJson ?? {}} />
                </div>
              </div>
            );
          },
        }}
        scroll={{ x: 1200 }}
      />

      <Modal
        title={jsonModalData?.title}
        open={jsonModalVisible}
        onCancel={() => setJsonModalVisible(false)}
        footer={null}
        width={700}
      >
        {jsonModalData && (
          <div style={{ display: 'flex', gap: 16 }}>
            <div style={{ flex: 1 }}>
              <div style={{ fontWeight: 500, marginBottom: 8 }}>{t('common.detail')}</div>
              <JSONViewer data={jsonModalData.request ?? {}} />
            </div>
            <div style={{ flex: 1 }}>
              <div style={{ fontWeight: 500, marginBottom: 8 }}>{t('table.result')}</div>
              <JSONViewer data={jsonModalData.response ?? {}} />
            </div>
          </div>
        )}
      </Modal>
    </ListPageLayout>
  );
}
