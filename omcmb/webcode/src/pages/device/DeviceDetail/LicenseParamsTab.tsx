/**
 * LicenseParamsTab — DeviceDetail "License 参数" tab。
 *
 * 后端 GET /devices/:id/license-params；刷新动作由 DeviceDetail 页头统一派发，
 * 与快速设置复用同一套参数同步队列、轮询和缓存失效逻辑。
 *
 * 替换 DeviceDetail/index.tsx 里的硬编码 mock license tab。
 */
import { useEffect, useMemo } from 'react';
import { Alert, Card, Empty, Spin, Table, Typography } from 'antd';

import { useT } from '@/hooks/useT';
import { useDeviceLicenseParams } from '@core/hooks/api/useDeviceLicenseParams';
import type { DeviceLicenseParam } from '@core/services/api/deviceLicenseParamApi';

const { Text } = Typography;

interface LicenseParamsTabProps {
  deviceId: string;
  onSyncTargetPathsChange?: (paths: string[]) => void;
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

export default function LicenseParamsTab({ deviceId, onSyncTargetPathsChange }: LicenseParamsTabProps) {
  const t = useT();
  const listQuery = useDeviceLicenseParams(deviceId);

  const items = useMemo(() => listQuery.data?.items ?? [], [listQuery.data?.items]);
  const licenseParamPaths = useMemo(
    () => Array.from(new Set(items.map((item) => item.standardPath).filter(Boolean))),
    [items],
  );
  useEffect(() => {
    onSyncTargetPathsChange?.(licenseParamPaths);
  }, [licenseParamPaths, onSyncTargetPathsChange]);

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
