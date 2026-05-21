import { useState, useMemo, useCallback } from 'react';
import { Button, Space, Tag, Typography, Modal, Row, Col, Card, Statistic, message } from 'antd';
import { ExportOutlined, BarChartOutlined } from '@ant-design/icons';
import * as XLSX from 'xlsx';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import { useEventLogList } from '@core/hooks/api/useEventLog';
import {
  eventLogApi,
  type EventLog,
  type EventLogListParams,
  type EventType,
} from '@core/services/api/eventLogApi';

// 事件日志 Tab（event_logs 列表 + 统计弹窗 + Excel 导出）
//
// 数据触发：device.RecordBootFromInform 在 CPE 上报 "1 BOOT" 且
// Device.HaltReason.MainReason 为空（普通重启）时落 event_logs。

const exportTimestamp = (): string => {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, '0');
  return (
    `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}` +
    `${pad(d.getHours())}${pad(d.getMinutes())}${pad(d.getSeconds())}`
  );
};

const EVENT_TYPE_I18N: Record<string, string> = {
  boot: 'log.event.reboot',
  // 后续接入更多 event_type 时在此追加
};

export default function EventLogTab() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [statisticVisible, setStatisticVisible] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [exporting, setExporting] = useState(false);

  const labelOfEventType = useCallback(
    (code: string): string => {
      const key = EVENT_TYPE_I18N[code];
      return key ? t(key) : code;
    },
    [t],
  );

  const eventTypeOptions = useMemo(
    () =>
      Object.keys(EVENT_TYPE_I18N).map((code) => ({
        label: labelOfEventType(code),
        value: code,
      })),
    [labelOfEventType],
  );

  const queryParams = useMemo<EventLogListParams>(() => {
    const p: EventLogListParams = { page, pageSize };
    if (filters.keyword && typeof filters.keyword === 'string') {
      p.deviceSn = filters.keyword;
    }
    if (filters.eventType && typeof filters.eventType === 'string') {
      p.eventType = filters.eventType as EventType;
    }
    if (Array.isArray(filters.timeRange) && filters.timeRange.length === 2) {
      const toISO = (v: unknown): string | undefined => {
        if (!v) return undefined;
        if (typeof v === 'string') return v;
        const obj = v as { toISOString?: () => string };
        return typeof obj.toISOString === 'function' ? obj.toISOString() : undefined;
      };
      const s = toISO(filters.timeRange[0]);
      const e = toISO(filters.timeRange[1]);
      if (s) p.startTime = s;
      if (e) p.endTime = e;
    }
    return p;
  }, [filters, page, pageSize]);

  const { data, isLoading } = useEventLogList(queryParams);
  const dataSource = data?.items ?? [];
  const total = data?.total ?? 0;

  const statistics = useMemo(() => {
    const counts: Record<string, number> = {};
    dataSource.forEach((row) => {
      counts[row.eventType] = (counts[row.eventType] || 0) + 1;
    });
    return Object.entries(counts)
      .map(([code, count]) => ({ code, label: labelOfEventType(code), count }))
      .sort((a, b) => b.count - a.count);
  }, [dataSource, labelOfEventType]);

  const filterFields: FilterField[] = useMemo(
    () => [
      {
        name: 'keyword',
        label: t('log.deviceCode'),
        type: 'input',
        placeholder: t('log.inputDeviceCode'),
      },
      {
        name: 'eventType',
        label: t('log.event.eventType'),
        type: 'select',
        options: eventTypeOptions,
      },
      { name: 'timeRange', label: t('log.timeRange'), type: 'date-range', span: 2 },
    ],
    [t, eventTypeOptions],
  );

  const handleExport = async () => {
    if (exporting) return;
    setExporting(true);
    try {
      const resp = await eventLogApi.list({ ...queryParams, page: 1, pageSize: 10000 });
      if (!resp.items || resp.items.length === 0) {
        void message.warning(t('log.exception.exportEmpty'));
        return;
      }
      const rows = resp.items.map((r) => ({
        [t('log.deviceCode')]: r.deviceSn,
        [t('log.exception.column.baseIp')]: r.operateIp || '-',
        [t('log.event.eventType')]: labelOfEventType(r.eventType),
        [t('log.event.column.reason')]: r.eventReason || '-',
        [t('log.event.column.time')]: r.occurredAt?.replace('T', ' ').slice(0, 19) ?? '',
      }));
      const ws = XLSX.utils.json_to_sheet(rows);
      const widthOfStr = (s: string) => {
        let w = 0;
        for (const ch of s) w += /[一-鿿＀-￯]/.test(ch) ? 2 : 1;
        return w;
      };
      ws['!cols'] = Object.keys(rows[0] ?? {}).map((key) => {
        let max = widthOfStr(key);
        for (const row of rows) {
          const v = String((row as Record<string, unknown>)[key] ?? '');
          if (widthOfStr(v) > max) max = widthOfStr(v);
        }
        return { wch: Math.min(Math.max(max + 2, 10), 50) };
      });
      const wb = XLSX.utils.book_new();
      XLSX.utils.book_append_sheet(wb, ws, t('log.eventLog'));
      XLSX.writeFile(wb, `${t('log.eventLog')}_${exportTimestamp()}.xlsx`);
      void message.success(t('log.exception.exportSuccess', { count: resp.items.length }));
    } finally {
      setExporting(false);
    }
  };

  const columns: DataTableColumn<EventLog>[] = useMemo(
    () => [
      {
        key: 'deviceSn',
        title: t('log.deviceCode'),
        dataIndex: 'deviceSn',
        width: 180,
        render: (val: unknown) => (
          <Typography.Text style={{ fontFamily: 'monospace' }}>
            {val as string}
          </Typography.Text>
        ),
      },
      {
        key: 'operateIp',
        title: t('log.exception.column.baseIp'),
        dataIndex: 'operateIp',
        width: 140,
        render: (val: unknown) =>
          val ? (
            <Typography.Text style={{ fontFamily: 'monospace' }}>
              {val as string}
            </Typography.Text>
          ) : (
            '-'
          ),
      },
      {
        key: 'eventType',
        title: t('log.event.eventType'),
        dataIndex: 'eventType',
        width: 120,
        render: (val: unknown) => <Tag color="blue">{labelOfEventType(val as string)}</Tag>,
      },
      {
        key: 'eventReason',
        title: t('log.event.column.reason'),
        dataIndex: 'eventReason',
        ellipsis: true,
        render: (val: unknown) => (val as string) || '-',
      },
      {
        key: 'occurredAt',
        title: t('log.event.column.time'),
        dataIndex: 'occurredAt',
        width: 180,
        render: (val: unknown) => (val as string)?.replace('T', ' ').slice(0, 19) ?? '-',
      },
    ],
    [t, labelOfEventType],
  );

  return (
    <>
      <FilterBar
        filterId="event-log-filter"
        fields={filterFields}
        onSearch={(v) => {
          setFilters(v);
          setPage(1);
        }}
        onReset={() => {
          setFilters({});
          setPage(1);
        }}
      />

      <Space style={{ marginBottom: 8, display: 'flex', justifyContent: 'flex-end' }}>
        <Button icon={<BarChartOutlined />} onClick={() => setStatisticVisible(true)}>
          {t('log.event.statistics')}
        </Button>
        <Button
          type="primary"
          icon={<ExportOutlined />}
          onClick={handleExport}
          loading={exporting}
        >
          {t('log.event.export')}
        </Button>
      </Space>

      <Card
        size="small"
        bordered
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{
          body: {
            padding: 0,
            display: 'flex',
            flexDirection: 'column',
            flex: 1,
            overflow: 'hidden',
          },
        }}
      >
        <DataTable<EventLog>
          tableId="event-log-list"
          columns={columns}
          dataSource={dataSource}
          rowKey="id"
          total={total}
          currentPage={page}
          pageSize={pageSize}
          onPageChange={(p, s) => {
            setPage(p);
            setPageSize(s);
          }}
          loading={isLoading}
          hideToolbar
          scroll={{ x: 'max-content', y: 'calc(100vh - 460px)' }}
          showRowNumber
          rowNumberTitle={t('log.event.seq')}
        />
      </Card>

      <Modal
        title={t('log.event.statModal.title')}
        open={statisticVisible}
        onCancel={() => setStatisticVisible(false)}
        footer={null}
        width={640}
      >
        <Card size="small" title={t('log.event.stat.eventDistribution')}>
          {statistics.length === 0 ? (
            <Typography.Text type="secondary">-</Typography.Text>
          ) : (
            <Row gutter={[16, 16]}>
              {statistics.map((s) => (
                <Col span={12} key={s.code}>
                  <Statistic title={s.label} value={s.count} />
                </Col>
              ))}
            </Row>
          )}
        </Card>
      </Modal>
    </>
  );
}
