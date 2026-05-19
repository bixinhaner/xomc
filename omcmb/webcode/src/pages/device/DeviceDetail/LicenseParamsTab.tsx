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
import { Alert, Button, Card, Empty, Table, Tag, Typography, message } from 'antd';
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

function formatDateTime(iso: string | undefined | null): string {
  if (!iso) return '-';
  return new Date(iso).toLocaleString();
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
        // 后端 ErrCodeRuleTaskRunning 复用为"刷新进行中"标识
        const code = extractBizCode(err);
        if (code === 1205) {
          void message.warning(t('device.licenseParam.refreshThrottled'));
          return;
        }
        const msg = err instanceof Error ? err.message : '';
        void message.error(msg ? `${t('device.licenseParam.refreshFailed')}: ${msg}` : t('device.licenseParam.refreshFailed'));
      },
    });
  };

  const columns = useMemo(
    () => [
      {
        title: t('device.licenseParam.col.path'),
        dataIndex: 'standardPath',
        key: 'standardPath',
        width: 400,
        render: (v: string) => <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{v}</Text>,
      },
      {
        title: t('device.licenseParam.col.dataType'),
        dataIndex: 'dataType',
        key: 'dataType',
        width: 110,
        render: (v: string | undefined) => v || '-',
      },
      {
        title: t('device.licenseParam.col.access'),
        dataIndex: 'access',
        key: 'access',
        width: 110,
        render: (v: string | undefined) => v || '-',
      },
      {
        title: t('device.licenseParam.col.value'),
        dataIndex: 'value',
        key: 'value',
        ellipsis: true,
        render: (v: string | undefined) =>
          v && v !== '' ? <Text>{v}</Text> : <Tag>{t('device.licenseParam.valueNotFetched')}</Tag>,
      },
      {
        title: t('device.licenseParam.col.updatedAt'),
        dataIndex: 'lastUpdatedAt',
        key: 'lastUpdatedAt',
        width: 180,
        render: formatDateTime,
      },
    ],
    [t],
  );

  const items = listQuery.data?.items ?? [];
  const isEmpty = !listQuery.isLoading && items.length === 0 && !listQuery.isError;

  return (
    <Card
      bordered={false}
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
      {listQuery.isError && !isEmpty ? (
        <Alert
          type="error"
          showIcon
          message={listQuery.error?.message ?? t('common.error')}
          style={{ marginBottom: 12 }}
        />
      ) : null}

      {isEmpty ? (
        <Empty description={t('device.licenseParam.empty')} />
      ) : (
        <Table<DeviceLicenseParam>
          rowKey="standardPath"
          dataSource={items}
          columns={columns}
          loading={listQuery.isLoading}
          size="small"
          pagination={false}
          scroll={{ x: 1100 }}
        />
      )}
    </Card>
  );
}
