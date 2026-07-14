/**
 * LicenseParamsTab — DeviceDetail "License 参数" tab。
 *
 * 后端 GET/POST /devices/:id/license-params；Q3 设计：刷新按钮"手动刷新"，
 * 点击 = 下发 GPV 任务 + 立刻 refetch 一次当前 DB 值。无轮询、无进度条；
 * 用户需再点一次刷新（或离开页面再回来）才能看到 CPE 响应回来的新值。
 *
 * 替换 DeviceDetail/index.tsx 里的硬编码 mock license tab。
 */
import { useMemo } from 'react';
import { Alert, Button, Card, Empty, Spin, Table, Typography, message } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';

import { useT } from '@/hooks/useT';
import {
  useDeviceLicenseParams,
  useRefreshDeviceLicenseParams,
} from '@core/hooks/api/useDeviceLicenseParams';
import type { DeviceLicenseParam } from '@core/services/api/deviceLicenseParamApi';

const { Text } = Typography;

interface LicenseParamsTabProps {
  deviceId: string;
}

interface LicenseRow {
  key: string;
  order: number;
  id: string;
  description: string;
  validityPeriod: string;
  capacity: string;
  remainTime: string;
}

const licenseItemFieldPattern = /(?:^|\.)(?:LicenseItem|Capacity)\.(\d+)\.(ID|Description|Value|ValidPeriod|RemainingPeriod|State)$/i;

function buildLicenseRows(items: DeviceLicenseParam[]): LicenseRow[] {
  const rowMap = new Map<string, LicenseRow>();

  for (const item of items) {
    const match = item.standardPath.match(licenseItemFieldPattern);
    if (!match) continue;

    const order = Number.parseInt(match[1], 10);
    const field = match[2].toLowerCase();
    const key = match[1];
    const row = rowMap.get(key) ?? {
      key,
      order,
      id: '',
      description: '',
      validityPeriod: '',
      capacity: '',
      remainTime: '',
    };

    switch (field) {
      case 'id':
        row.id = item.value || row.id;
        break;
      case 'description':
        row.description = item.value || row.description;
        break;
      case 'value':
      case 'state':
        if (!row.capacity && item.value) {
          row.capacity = item.value;
        }
        break;
      case 'validperiod':
        row.validityPeriod = item.value || row.validityPeriod;
        break;
      case 'remainingperiod':
        row.remainTime = item.value || row.remainTime;
        break;
      default:
        break;
    }

    rowMap.set(key, row);
  }

  return Array.from(rowMap.values())
    .sort((left, right) => left.order - right.order)
    .map((row) => ({
      ...row,
      id: row.id || `#${row.order}`,
    }));
}

function extractBizCode(err: unknown): number {
  if (!err || typeof err !== 'object') return 0;
  const e = err as Record<string, unknown>;
  if (typeof e.bizCode === 'number') return e.bizCode;
  const resp = e.response as Record<string, unknown> | undefined;
  if (resp && typeof resp === 'object') {
    const data = resp.data as Record<string, unknown> | undefined;
    if (data && typeof data === 'object') {
      if (typeof data.biz_code === 'number') return data.biz_code;
      if (typeof data.bizCode === 'number') return data.bizCode;
    }
  }
  return 0;
}

export default function LicenseParamsTab({ deviceId }: LicenseParamsTabProps) {
  const t = useT();
  const listQuery = useDeviceLicenseParams(deviceId);
  const refreshMutation = useRefreshDeviceLicenseParams(deviceId);

  const handleRefresh = () => {
    refreshMutation.mutate(undefined, {
      onSuccess: () => {
        void message.success(t('device.licenseParam.refreshHint'));
      },
      onError: (err) => {
        // 后端 ErrCodeRuleTaskRunning(1305) 复用为"刷新进行中"标识；1205 兼容旧设计文档/旧服务。
        const code = extractBizCode(err);
        const msg = err instanceof Error ? err.message : '';
        if (code === 1305 || code === 1205 || msg.includes('license refresh already running')) {
          void message.warning(t('device.licenseParam.refreshThrottled'));
          return;
        }
        void message.error(msg ? `${t('device.licenseParam.refreshFailed')}: ${msg}` : t('device.licenseParam.refreshFailed'));
      },
    });
  };

  const items = listQuery.data?.items ?? [];
  const rows = useMemo(() => buildLicenseRows(items), [items]);
  const columns = useMemo(
    () => [
      {
        title: t('device.licenseParam.col.id'),
        dataIndex: 'id',
        key: 'id',
        width: 140,
        render: (value: string) => <Text style={{ fontFamily: 'monospace' }}>{value || '-'}</Text>,
      },
      {
        title: t('device.licenseParam.col.description'),
        dataIndex: 'description',
        key: 'description',
        ellipsis: true,
        render: (value: string) => value || '-',
      },
      {
        title: t('device.licenseParam.col.validityPeriod'),
        dataIndex: 'validityPeriod',
        key: 'validityPeriod',
        width: 180,
        render: (value: string) => value || '-',
      },
      {
        title: t('device.licenseParam.col.capacity'),
        dataIndex: 'capacity',
        key: 'capacity',
        width: 180,
        render: (value: string) => value || '-',
      },
      {
        title: t('device.licenseParam.col.remainTime'),
        dataIndex: 'remainTime',
        key: 'remainTime',
        width: 180,
        render: (value: string) => value || '-',
      },
      {
        title: t('device.licenseParam.col.operate'),
        key: 'operate',
        width: 120,
        render: () => '-',
      },
    ],
    [t],
  );

  const isEmpty = !listQuery.isLoading && rows.length === 0 && !listQuery.isError;
  const isInitialLoading = listQuery.isLoading && items.length === 0;

  return (
    <Card
      variant="borderless"
      title={t('device.licenseParam.title')}
      extra={
        <Button
          icon={<ReloadOutlined />}
          onClick={handleRefresh}
          loading={refreshMutation.isPending || listQuery.isFetching}
        >
          {t('device.licenseParam.refresh')}
        </Button>
      }
    >
      {listQuery.isError ? (
        <Alert
          type="error"
          showIcon
          message={listQuery.error?.message ?? t('common.error')}
          style={{ marginBottom: 12 }}
        />
      ) : null}

      {isInitialLoading ? (
        <div style={{ padding: '32px 0', display: 'flex', justifyContent: 'center' }}>
          <Spin />
        </div>
      ) : isEmpty ? (
        <Empty description={t('device.licenseParam.empty')} />
      ) : (
        <Table<LicenseRow>
          rowKey="key"
          dataSource={rows}
          columns={columns}
          loading={listQuery.isLoading}
          size="small"
          pagination={false}
          scroll={{ x: 1160 }}
        />
      )}
    </Card>
  );
}
