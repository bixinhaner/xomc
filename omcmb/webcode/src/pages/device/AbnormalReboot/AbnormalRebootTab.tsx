import { useState, useMemo } from 'react';
import { Button, Typography, message, Drawer, Descriptions, Card, Space } from 'antd';
import { ExportOutlined } from '@ant-design/icons';
import * as XLSX from 'xlsx';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import { useAbnormalRebootList } from '@core/hooks/api/useDeviceAbnormalReboot';
import { deviceAbnormalRebootApi } from '@core/services/api/deviceAbnormalRebootApi';
import type {
  AbnormalReboot,
  AbnormalRebootListParams,
} from '@core/services/api/deviceAbnormalRebootApi';

// 异常重启记录 Tab（station_fault_logs 列表 + 详情抽屉 + Excel 导出）
//
// 数据触发：device.RecordBootFromInform 在 CPE 上报 "1 BOOT" 且
// Device.HaltReason.MainReason 非空时识别即落库。

// CPE 上报的 HaltReason 是 snake_case enum（如 halt_reboot / lte_worker_unhealthy）。
// 走 i18n key `reboot.haltMain.<value>` 和 `reboot.haltDetail.<value>` 翻译；
// 没维护过的值原样回退展示，避免对未来新增厂商上报值"硬挡"。
const localizeHaltReason = (
  value: string | undefined | null,
  kind: 'haltMain' | 'haltDetail',
  t: (key: string) => string,
): string => {
  const raw = (value ?? '').trim();
  if (!raw) return '-';
  const key = `reboot.${kind}.${raw}`;
  const translated = t(key);
  return translated && translated !== key ? translated : raw;
};

const formatRuntime = (seconds: number, t: (key: string) => string): string => {
  if (!seconds || seconds <= 0) return '-';
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const parts: string[] = [];
  if (days > 0) parts.push(`${days}${t('common.day')}`);
  if (hours > 0) parts.push(`${hours}${t('common.hour')}`);
  if (minutes > 0 && days === 0) parts.push(`${minutes}${t('common.minute')}`);
  return parts.length > 0 ? parts.join('') : '< 1m';
};

const exportTimestamp = (): string => {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, '0');
  return (
    `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}` +
    `${pad(d.getHours())}${pad(d.getMinutes())}${pad(d.getSeconds())}`
  );
};

export default function AbnormalRebootTab() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [logDetailVisible, setLogDetailVisible] = useState(false);
  const [selectedLog, setSelectedLog] = useState<AbnormalReboot | null>(null);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [exporting, setExporting] = useState(false);

  const queryParams = useMemo<AbnormalRebootListParams>(() => {
    const params: AbnormalRebootListParams = { page, pageSize };
    if (filters.keyword && typeof filters.keyword === 'string') {
      params.deviceSn = filters.keyword;
    }
    if (filters.deviceType && typeof filters.deviceType === 'string') {
      params.deviceType = filters.deviceType;
    }
    if (Array.isArray(filters.timeRange) && filters.timeRange.length === 2) {
      const toISO = (v: unknown): string | undefined => {
        if (!v) return undefined;
        if (typeof v === 'string') return v;
        const obj = v as { toISOString?: () => string };
        return typeof obj.toISOString === 'function' ? obj.toISOString() : undefined;
      };
      const startISO = toISO(filters.timeRange[0]);
      const endISO = toISO(filters.timeRange[1]);
      if (startISO) params.startTime = startISO;
      if (endISO) params.endTime = endISO;
    }
    return params;
  }, [filters, page, pageSize]);

  const { data, isLoading } = useAbnormalRebootList(queryParams);
  const dataSource = data?.items ?? [];
  const total = data?.total ?? 0;

  const filterFields: FilterField[] = useMemo(
    () => [
      {
        name: 'keyword',
        label: t('log.exception.column.deviceCode'),
        type: 'input',
        placeholder: t('log.exception.devCodeNameIp'),
      },
      {
        name: 'deviceType',
        label: t('log.exception.column.deviceType'),
        type: 'select',
        options: [
          { label: 'eNB', value: 'eNB' },
          { label: 'gNB', value: 'gNB' },
        ],
      },
      { name: 'timeRange', label: t('log.timeRange'), type: 'date-range', span: 2 },
    ],
    [t],
  );

  const handleViewDetail = (record: AbnormalReboot) => {
    setSelectedLog(record);
    setLogDetailVisible(true);
  };

  const handleExport = async () => {
    if (exporting) return;
    setExporting(true);
    try {
      const resp = await deviceAbnormalRebootApi.list({
        ...queryParams,
        page: 1,
        pageSize: 10000,
      });
      if (!resp.items || resp.items.length === 0) {
        void message.warning(t('log.exception.exportEmpty'));
        return;
      }
      const rows = resp.items.map((r) => ({
        [t('log.exception.column.deviceCode')]: r.deviceSn,
        [t('log.exception.column.deviceName')]: r.deviceName || '-',
        [t('log.exception.column.deviceType')]: r.deviceType || '-',
        [t('log.exception.column.baseIp')]: r.operateIp || '-',
        [t('log.exception.column.softwareVersion')]: r.softwareVersion || '-',
        [t('log.exception.column.haltMainReason')]: localizeHaltReason(r.haltMainReason, 'haltMain', t),
        [t('log.exception.column.haltReason')]: localizeHaltReason(r.haltDetailReason, 'haltDetail', t),
        [t('log.exception.column.time')]:
          r.collectedAt?.replace('T', ' ').slice(0, 19) ?? '',
        [t('log.exception.column.runtime')]: formatRuntime(r.runtimeBeforeReboot, t),
      }));
      const widthOfStr = (s: string): number => {
        let w = 0;
        for (const ch of s) w += /[一-鿿＀-￯]/.test(ch) ? 2 : 1;
        return w;
      };
      const ws = XLSX.utils.json_to_sheet(rows);
      ws['!cols'] = Object.keys(rows[0] ?? {}).map((key) => {
        let max = widthOfStr(key);
        for (const row of rows) {
          const v = String((row as Record<string, unknown>)[key] ?? '');
          if (widthOfStr(v) > max) max = widthOfStr(v);
        }
        return { wch: Math.min(Math.max(max + 2, 10), 50) };
      });
      const wb = XLSX.utils.book_new();
      XLSX.utils.book_append_sheet(wb, ws, t('log.exception.exportSheetName'));
      XLSX.writeFile(wb, `${t('log.exception.exportFileName')}_${exportTimestamp()}.xlsx`);
      void message.success(t('log.exception.exportSuccess', { count: resp.items.length }));
    } catch (e) {
      console.error('export abnormal reboot failed', e);
      void message.error(t('log.exception.exportFailed'));
    } finally {
      setExporting(false);
    }
  };

  const columns: DataTableColumn<AbnormalReboot>[] = useMemo(
    () => [
      {
        key: 'deviceSn',
        title: t('log.exception.column.deviceCode'),
        dataIndex: 'deviceSn',
        width: 180,
        render: (val: unknown) => (
          <Typography.Text style={{ fontFamily: 'monospace' }}>
            {val as string}
          </Typography.Text>
        ),
      },
      {
        key: 'deviceName',
        title: t('log.exception.column.deviceName'),
        dataIndex: 'deviceName',
        width: 180,
        ellipsis: true,
        render: (val: unknown) => (val as string) || '-',
      },
      {
        key: 'deviceType',
        title: t('log.exception.column.deviceType'),
        dataIndex: 'deviceType',
        width: 90,
        render: (val: unknown) => (val as string) || '-',
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
        key: 'softwareVersion',
        title: t('log.exception.column.softwareVersion'),
        dataIndex: 'softwareVersion',
        width: 120,
        render: (val: unknown) => (val as string) || '-',
      },
      {
        key: 'haltMainReason',
        title: t('log.exception.column.haltMainReason'),
        dataIndex: 'haltMainReason',
        width: 140,
        ellipsis: true,
        render: (val: unknown) => localizeHaltReason(val as string | undefined, 'haltMain', t),
      },
      {
        key: 'haltDetailReason',
        title: t('log.exception.column.haltReason'),
        dataIndex: 'haltDetailReason',
        width: 180,
        ellipsis: true,
        render: (val: unknown) => localizeHaltReason(val as string | undefined, 'haltDetail', t),
      },
      {
        key: 'collectedAt',
        title: t('log.exception.column.time'),
        dataIndex: 'collectedAt',
        width: 180,
        render: (val: unknown) => (val as string)?.replace('T', ' ').slice(0, 19) ?? '-',
      },
      {
        key: 'runtimeBeforeReboot',
        title: t('log.exception.column.runtime'),
        dataIndex: 'runtimeBeforeReboot',
        width: 120,
        render: (val: unknown) => formatRuntime(val as number, t),
      },
    ],
    [t],
  );

  return (
    <>
      <FilterBar
        filterId="abnormal-reboot-filter"
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
        <Button
          type="primary"
          icon={<ExportOutlined />}
          onClick={handleExport}
          loading={exporting}
        >
          {t('log.export')}
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
        <DataTable<AbnormalReboot>
          tableId="abnormal-reboot-list"
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
          rowNumberTitle={t('log.exception.column.seq')}
          onRow={(record) => ({
            onClick: () => handleViewDetail(record),
            style: { cursor: 'pointer' },
          })}
        />
      </Card>

      <Drawer
        title={t('log.exception.detail.title')}
        placement="right"
        width={640}
        open={logDetailVisible}
        onClose={() => setLogDetailVisible(false)}
      >
        {selectedLog && (
          <div>
            <Card size="small" title={t('log.exception.detail.devInfo')} style={{ marginBottom: 16 }}>
              <Descriptions column={2} size="small">
                <Descriptions.Item label={t('log.exception.detail.devCode')}>
                  <Typography.Text style={{ fontFamily: 'monospace' }}>
                    {selectedLog.deviceSn}
                  </Typography.Text>
                </Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.devName')}>
                  {selectedLog.deviceName || '-'}
                </Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.devType')}>
                  {selectedLog.deviceType || '-'}
                </Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.baseIp')}>
                  {selectedLog.operateIp ? (
                    <Typography.Text style={{ fontFamily: 'monospace' }}>
                      {selectedLog.operateIp}
                    </Typography.Text>
                  ) : (
                    '-'
                  )}
                </Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.softwareVersion')}>
                  {selectedLog.softwareVersion || '-'}
                </Descriptions.Item>
              </Descriptions>
            </Card>

            <Card size="small" title={t('log.exception.detail.exceptionInfo')} style={{ marginBottom: 16 }}>
              <Descriptions column={2} size="small">
                <Descriptions.Item label={t('log.exception.detail.occurTime')}>
                  {selectedLog.collectedAt?.replace('T', ' ').slice(0, 19)}
                </Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.runtime')}>
                  {formatRuntime(selectedLog.runtimeBeforeReboot, t)}
                </Descriptions.Item>
                <Descriptions.Item label={t('log.exception.column.haltMainReason')} span={2}>
                  {localizeHaltReason(selectedLog.haltMainReason, 'haltMain', t)}
                </Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.haltReason')} span={2}>
                  {localizeHaltReason(selectedLog.haltDetailReason, 'haltDetail', t)}
                </Descriptions.Item>
              </Descriptions>
            </Card>
          </div>
        )}
      </Drawer>
    </>
  );
}
