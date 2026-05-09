import { useState, useMemo } from 'react';
import { Tag, Tooltip } from 'antd';
import { useSearchParams } from 'react-router-dom';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import { useLicenseLogs } from '@core/hooks/api/useLicense';
import type {
  LicenseLog,
  LicenseLogType,
  LicenseLogResult,
  LicenseLogQuery,
} from '@core/services/api/licenseApi';

// 与后端 internal/license/license_log_model.go 保持一致：9 种 log_type + 4 种 result
// 通过 useLicenseLogs hook → GET /licenses/logs 真实接口。

const logTypeColorMap: Record<LicenseLogType, string> = {
  import: 'blue',
  activate: 'green',
  revoke: 'orange',
  query_detail: 'default',
  enforcement_capacity: 'volcano',
  enforcement_expiry: 'volcano',
  capacity_alert: 'gold',
  expiry_alert: 'gold',
  auto_expire: 'purple',
};

const resultColorMap: Record<LicenseLogResult, string> = {
  success: 'green',
  failed: 'red',
  denied: 'orange',
  warning: 'gold',
};

export default function LicenseLogs() {
  const t = useT();
  const [searchParams] = useSearchParams();
  const initialLicenseId = searchParams.get('license_id') ?? undefined;

  const [filters, setFilters] = useState<Record<string, unknown>>(
    initialLicenseId ? { licenseId: initialLicenseId } : {},
  );
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const logTypeLabelMap: Record<LicenseLogType, string> = useMemo(
    () => ({
      import: t('license.logs.type.import'),
      activate: t('license.logs.type.activate'),
      revoke: t('license.logs.type.revoke'),
      query_detail: t('license.logs.type.queryDetail'),
      enforcement_capacity: t('license.logs.type.enforcementCapacity'),
      enforcement_expiry: t('license.logs.type.enforcementExpiry'),
      capacity_alert: t('license.logs.type.capacityAlert'),
      expiry_alert: t('license.logs.type.expiryAlert'),
      auto_expire: t('license.logs.type.autoExpire'),
    }),
    [t],
  );

  const resultLabelMap: Record<LicenseLogResult, string> = useMemo(
    () => ({
      success: t('license.logs.result.success'),
      failed: t('license.logs.result.failed'),
      denied: t('license.logs.result.denied'),
      warning: t('license.logs.result.warning'),
    }),
    [t],
  );

  const filterFields: FilterField[] = useMemo(
    () => [
      {
        name: 'licenseId',
        label: t('license.logs.filterLicenseId'),
        type: 'input',
        placeholder: t('license.logs.licenseIdPlaceholder'),
      },
      {
        name: 'logType',
        label: t('license.logs.filterType'),
        type: 'select',
        options: (Object.entries(logTypeLabelMap) as [LicenseLogType, string][]).map(
          ([value, label]) => ({ value, label }),
        ),
      },
      {
        name: 'result',
        label: t('license.logs.filterResult'),
        type: 'select',
        options: (Object.entries(resultLabelMap) as [LicenseLogResult, string][]).map(
          ([value, label]) => ({ value, label }),
        ),
      },
      {
        name: 'search',
        label: t('license.logs.filterSearch'),
        type: 'input',
        placeholder: t('license.logs.searchPlaceholder'),
      },
    ],
    [t, logTypeLabelMap, resultLabelMap],
  );

  const queryParams: LicenseLogQuery = useMemo(() => {
    const q: LicenseLogQuery = { page, pageSize };
    if (typeof filters.licenseId === 'string' && filters.licenseId.trim()) {
      q.licenseId = filters.licenseId.trim();
    }
    if (typeof filters.logType === 'string' && filters.logType) {
      q.logTypes = [filters.logType as LicenseLogType];
    }
    if (typeof filters.result === 'string' && filters.result) {
      q.results = [filters.result as LicenseLogResult];
    }
    if (typeof filters.search === 'string' && filters.search.trim()) {
      q.search = filters.search.trim();
    }
    return q;
  }, [filters, page, pageSize]);

  const { data, isLoading, refetch } = useLicenseLogs(queryParams);
  const items = data?.items ?? [];
  const total = data?.total ?? 0;

  const columns: DataTableColumn<LicenseLog & Record<string, unknown>>[] = useMemo(
    () => [
      {
        key: 'createdAt',
        title: t('table.time'),
        dataIndex: 'createdAt',
        width: 170,
        render: (val) => (val ? new Date(String(val)).toLocaleString('zh-CN') : '—'),
      },
      {
        key: 'logType',
        title: t('license.logs.type.label'),
        dataIndex: 'logType',
        width: 160,
        render: (val) => {
          const tp = val as LicenseLogType;
          return <Tag color={logTypeColorMap[tp]}>{logTypeLabelMap[tp] ?? tp}</Tag>;
        },
      },
      {
        key: 'licenseId',
        title: 'License ID',
        dataIndex: 'licenseId',
        width: 280,
        render: (val) =>
          val ? (
            <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>
          ) : (
            <span style={{ color: 'var(--color-text-secondary)' }}>—</span>
          ),
      },
      {
        key: 'actorUserId',
        title: t('license.logs.actor'),
        dataIndex: 'actorUserId',
        width: 180,
        render: (val) =>
          val ? (
            <span style={{ fontFamily: 'monospace', fontSize: 12 }}>
              {String(val).slice(0, 8)}…
            </span>
          ) : (
            <Tag>{t('license.logs.systemActor')}</Tag>
          ),
      },
      {
        key: 'result',
        title: t('table.result'),
        dataIndex: 'result',
        width: 100,
        render: (val) => {
          const r = val as LicenseLogResult;
          return <Tag color={resultColorMap[r]}>{resultLabelMap[r] ?? r}</Tag>;
        },
      },
      {
        key: 'details',
        title: t('common.detail'),
        dataIndex: 'details',
        ellipsis: true,
        render: (val) => {
          const d = val as Record<string, unknown> | undefined;
          if (!d || Object.keys(d).length === 0) return '—';
          const summary =
            typeof d.summary === 'string'
              ? d.summary
              : JSON.stringify(d).slice(0, 200);
          return (
            <Tooltip
              title={
                <pre style={{ margin: 0, fontSize: 11, maxWidth: 480 }}>
                  {JSON.stringify(d, null, 2)}
                </pre>
              }
            >
              <span style={{ cursor: 'help' }}>{summary}</span>
            </Tooltip>
          );
        },
      },
      {
        key: 'clientIp',
        title: t('license.clientIp'),
        dataIndex: 'clientIp',
        width: 140,
        render: (val) =>
          val ? (
            <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>
          ) : (
            <span style={{ color: 'var(--color-text-secondary)' }}>—</span>
          ),
      },
    ],
    [t, logTypeLabelMap, resultLabelMap],
  );

  return (
    <ListPageLayout
      title={t('nav.license.logs')}
      subtitle={t('license.logsSubtitle')}
    >
      <FilterBar
        filterId="license-logs-filter"
        fields={filterFields}
        defaultValues={initialLicenseId ? { licenseId: initialLicenseId } : undefined}
        onSearch={(vals) => {
          setFilters(vals);
          setPage(1);
        }}
        onReset={() => {
          setFilters({});
          setPage(1);
        }}
      />
      <DataTable
        tableId="license-logs-list"
        columns={columns}
        dataSource={items as (LicenseLog & Record<string, unknown>)[]}
        loading={isLoading}
        rowKey="id"
        total={total}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => {
          setPage(p);
          setPageSize(s);
        }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1300 }}
        alarmRowStyle={(record) => {
          const log = record as LicenseLog;
          if (log.result === 'denied') return 'major';
          if (log.result === 'failed') return 'critical';
          if (log.result === 'warning') return 'minor';
          return null;
        }}
      />
    </ListPageLayout>
  );
}
