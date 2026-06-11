import { useState, useMemo } from 'react';
import { Button, Tag, Drawer, Badge } from 'antd';
import { EyeOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import JSONViewer from '@/components/JSONViewer';
import type { OpsCommandRecord } from '@core/mock/data/opsTools';
import { useOpsCommandRecords } from '@core/hooks/api/useOpsTools';
import { useT } from '@/hooks/useT';

function formatDuration(ms: number): string {
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)} s`;
  return `${ms} ms`;
}

export default function CommandManagement() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedRecord, setSelectedRecord] = useState<OpsCommandRecord | null>(null);

  const queryParams = useMemo(
    () => ({
      deviceSn: filters.deviceSn ? String(filters.deviceSn) : undefined,
      operator: filters.operator ? String(filters.operator) : undefined,
      success:
        filters.result === 'success' ? true : filters.result === 'failed' ? false : undefined,
      page,
      pageSize,
    }),
    [filters, page, pageSize],
  );
  const { data, isLoading, refetch } = useOpsCommandRecords(queryParams);
  const records: OpsCommandRecord[] = data?.items ?? [];
  const total = data?.total ?? 0;

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('ops.commandOrDevice'), type: 'input', placeholder: t('ops.commandOrDevicePlaceholder') },
    { name: 'deviceSn', label: t('device.sn'), type: 'input', placeholder: t('ops.deviceSnPlaceholder') },
    { name: 'operator', label: t('table.operator'), type: 'input', placeholder: t('ops.operatorPlaceholder') },
    {
      name: 'result',
      label: t('ops.executeResult'),
      type: 'select',
      options: [
        { label: t('status.success'), value: 'success' },
        { label: t('status.failed'), value: 'failed' },
      ],
    },
    { name: 'timeRange', label: t('ops.executeTime'), type: 'date-range' },
  ], [t]);

  // keyword 是前端模糊匹配，后端目前不支持 — 在已分页结果上叠加 client-side filter。
  const paginated = records.filter((r) => {
    if (filters.keyword) {
      const kw = String(filters.keyword).toLowerCase();
      if (!r.commandText.toLowerCase().includes(kw) && !r.deviceSn.toLowerCase().includes(kw)) return false;
    }
    return true;
  });

  const columns: DataTableColumn<OpsCommandRecord & Record<string, unknown>>[] = useMemo(() => [
    {
      key: 'executeTime', title: t('ops.executeTime'), dataIndex: 'executeTime', width: 170,
      render: (val) => new Date(String(val)).toLocaleString('zh-CN'),
    },
    {
      key: 'commandText', title: t('ops.command'), dataIndex: 'commandText', width: 240,
      render: (val) => (
        <span style={{ fontFamily: 'monospace', fontSize: 12, background: '#f5f5f5', padding: '1px 6px', borderRadius: 3 }}>
          {String(val)}
        </span>
      ),
    },
    {
      key: 'deviceName', title: t('device.name'), dataIndex: 'deviceName', width: 160, ellipsis: true,
    },
    {
      key: 'deviceSn', title: t('device.sn'), dataIndex: 'deviceSn', width: 110,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>,
    },
    { key: 'operator', title: t('table.operator'), dataIndex: 'operator', width: 100 },
    {
      key: 'duration', title: t('ops.duration'), dataIndex: 'duration', width: 90,
      render: (val) => {
        const ms = Number(val);
        const color = ms > 5000 ? '#ff4d4f' : ms > 2000 ? '#faad14' : '#52c41a';
        return <span style={{ color, fontFamily: 'monospace', fontSize: 12 }}>{formatDuration(ms)}</span>;
      },
    },
    {
      key: 'success', title: t('table.result'), dataIndex: 'success', width: 90,
      render: (val) => {
        const ok = Boolean(val);
        return (
          <Badge
            status={ok ? 'success' : 'error'}
            text={
              <Tag color={ok ? 'green' : 'red'} style={{ margin: 0 }}>
                {ok ? t('status.success') : t('status.failed')}
              </Tag>
            }
          />
        );
      },
    },
    {
      key: 'output', title: t('ops.outputSummary'), dataIndex: 'output', ellipsis: true,
      render: (val, record) => {
        const r = record as OpsCommandRecord;
        if (!r.success && r.errorMessage) {
          return <span style={{ color: '#ff4d4f', fontSize: 12 }}>{r.errorMessage}</span>;
        }
        return <span style={{ color: '#666', fontSize: 12 }}>{String(val).split('\n')[0]}</span>;
      },
    },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 80, fixed: 'right',
      render: (_, record) => {
        const r = record as OpsCommandRecord;
        return (
          <Button type="link" size="small" icon={<EyeOutlined />}
            onClick={() => { setSelectedRecord(r); setDetailVisible(true); }}>
            {t('common.detail')}
          </Button>
        );
      },
    },
  ], [t]);

  const detailData = selectedRecord ? {
    id: selectedRecord.id,
    command: selectedRecord.commandText,
    device: {
      sn: selectedRecord.deviceSn,
      name: selectedRecord.deviceName,
    },
    operator: selectedRecord.operator,
    executeTime: selectedRecord.executeTime,
    duration: `${selectedRecord.duration} ms`,
    success: selectedRecord.success,
    output: selectedRecord.output ? selectedRecord.output.split('\n') : [],
    errorMessage: selectedRecord.errorMessage ?? null,
  } : null;

  return (
    <ListPageLayout title={t('nav.ops.commands')} subtitle={t('ops.commandsSubtitle')}>
      <FilterBar
        filterId="ops-command-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable
        tableId="ops-command-list"
        columns={columns}
        dataSource={paginated as (OpsCommandRecord & Record<string, unknown>)[]}
        loading={isLoading}
        rowKey="id"
        total={total}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1300 }}
        alarmRowStyle={(record) => {
          const r = record as OpsCommandRecord;
          return !r.success ? 'major' : null;
        }}
      />

      <Drawer
        title={selectedRecord ? `${t('ops.commandDetail')} — ${selectedRecord.commandText}` : t('ops.commandDetail')}
        open={detailVisible}
        onClose={() => setDetailVisible(false)}
        size={680}
      >
        {selectedRecord && detailData && (
          <div>
            <div style={{ marginBottom: 16, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
              <Tag color={selectedRecord.success ? 'green' : 'red'}>
                {selectedRecord.success ? t('ops.executeSuccess') : t('ops.executeFailed')}
              </Tag>
              <span style={{ fontFamily: 'monospace', fontSize: 12, background: '#f5f5f5', padding: '1px 8px', borderRadius: 3 }}>
                {selectedRecord.commandText}
              </span>
            </div>

            <div style={{ marginBottom: 8, fontWeight: 600 }}>{t('ops.executeInfo')}</div>
            <JSONViewer data={detailData} defaultExpanded maxHeight={200} style={{ marginBottom: 16 }} />

            <div style={{ marginBottom: 8, fontWeight: 600 }}>{t('ops.outputResult')}</div>
            <div
              style={{
                background: '#1a1a1a',
                borderRadius: 6,
                padding: '12px 16px',
                fontFamily: 'monospace',
                fontSize: 13,
                lineHeight: 1.7,
                minHeight: 120,
                maxHeight: 360,
                overflowY: 'auto',
              }}
            >
              {selectedRecord.success ? (
                selectedRecord.output
                  ? selectedRecord.output.split('\n').map((line, i) => (
                    <div key={i} style={{ color: '#52c41a' }}>{line}</div>
                  ))
                  : <span style={{ color: '#555' }}>{t('ops.noOutput')}</span>
              ) : (
                <>
                  {selectedRecord.errorMessage && (
                    <div style={{ color: '#ff4d4f', marginBottom: 8 }}>{selectedRecord.errorMessage}</div>
                  )}
                  <div style={{ color: '#555' }}>{t('ops.executeFailedNoOutput')}</div>
                </>
              )}
            </div>
          </div>
        )}
      </Drawer>
    </ListPageLayout>
  );
}
