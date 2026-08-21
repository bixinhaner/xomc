import { useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Collapse,
  Descriptions,
  Drawer,
  Empty,
  Pagination,
  Space,
  Spin,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useDeviceControlActions } from '@core/hooks/api/useDevices';
import type { Device, DeviceControlActionHistory, DeviceControlParameterState } from '@core/types/device';
import { formatSystemTime } from '@core/utils/systemTime';
import { useT } from '@/hooks/useT';

interface DrawerProps {
  device: Device | null;
  open: boolean;
  onClose: () => void;
  onOpenDetail?: () => void;
}

interface ContentProps {
  device: Device;
  enabled?: boolean;
  mode?: 'compact' | 'full';
}

interface ParameterEvidenceRow {
  path: string;
  label: string;
  before?: string;
  requested?: string;
  verified?: string;
}

function parameterLabel(path: string, t: ReturnType<typeof useT>): string {
  if (path.endsWith('.IPSEC_ENABLE')) return t('device.control.parameter.ipsec');
  if (path.includes('.RFTxStatus') || path.endsWith('.RfState')) return t('device.control.parameter.rf');
  return t('device.control.parameter.other');
}

function controlReasonLabel(reasonCode: string, t: ReturnType<typeof useT>): string {
  switch (reasonCode) {
    case 'confirmed_exit':
    case 'confirmed_outside':
    case 'effective_state_outside':
    case 'geofence_outside':
      return t('device.control.reason.confirmedExit');
    case 'confirmed_enter':
    case 'confirmed_inside':
    case 'effective_state_inside':
    case 'geofence_inside':
      return t('device.control.reason.confirmedEnter');
    default:
      return reasonCode || t('device.control.reason.unknown');
  }
}

function confirmedStateLabel(state: string, t: ReturnType<typeof useT>): string {
  switch (state) {
    case 'outside': return t('device.control.confirmedState.outside');
    case 'inside': return t('device.control.confirmedState.inside');
    case 'unknown': return t('device.control.confirmedState.unknown');
    default: return state || '-';
  }
}

function ruleTypeLabel(ruleType: string, t: ReturnType<typeof useT>): string {
  switch (ruleType) {
    case 'polygon_allow_zone': return t('device.control.ruleType.polygon');
    case 'baseline_radius': return t('device.control.ruleType.radius');
    default: return ruleType || '-';
  }
}

function controlSourceLabel(sourceType: DeviceControlActionHistory['sourceType'], sourceName: string, t: ReturnType<typeof useT>): string {
  return sourceName || t(`device.control.source.${sourceType}`);
}

function mergeParameterEvidence(action: DeviceControlActionHistory, t: ReturnType<typeof useT>): ParameterEvidenceRow[] {
  const rows = new Map<string, ParameterEvidenceRow>();
  const merge = (items: DeviceControlParameterState[], key: 'before' | 'requested' | 'verified') => {
    for (const item of items) {
      const row = rows.get(item.path) ?? { path: item.path, label: parameterLabel(item.path, t) };
      row[key] = item.value;
      rows.set(item.path, row);
    }
  };
  merge(action.beforeState, 'before');
  merge(action.requestedState, 'requested');
  merge(action.verifiedState, 'verified');
  return Array.from(rows.values());
}

export function DeviceControlReasonContent({ device, enabled = true, mode = 'full' }: ContentProps) {
  const t = useT();
  const [page, setPage] = useState(1);
  const pageSize = mode === 'compact' ? 1 : 20;
  const { data, isLoading, isError } = useDeviceControlActions(device.id, page, pageSize, enabled);

  const parameterColumns = useMemo<ColumnsType<ParameterEvidenceRow>>(() => [
    {
      title: t('device.control.parameter'),
      dataIndex: 'label',
      width: 180,
      render: (label, record) => (
        <Tooltip title={record.path}>
          <Typography.Text>{label}</Typography.Text>
        </Tooltip>
      ),
    },
    { title: t('device.control.beforeValue'), dataIndex: 'before', render: (value) => value ?? '-' },
    { title: t('device.control.requestedValue'), dataIndex: 'requested', render: (value) => value ?? '-' },
    { title: t('device.control.verifiedValue'), dataIndex: 'verified', render: (value) => value ?? '-' },
  ], [t]);

  const renderAction = (action: DeviceControlActionHistory, condensedMeta = false) => {
    const evaluation = action.evaluation;
    const technicalItems = [{
      key: 'technical',
      label: t('device.control.technicalDetails'),
      children: (
        <Space orientation="vertical" size={12} style={{ width: '100%' }}>
          <Descriptions size="small" column={condensedMeta ? 1 : 2} bordered>
            <Descriptions.Item label={t('device.control.reasonCode')}>
              <Typography.Text code>{action.reasonCode || '-'}</Typography.Text>
            </Descriptions.Item>
            <Descriptions.Item label={t('device.control.effectiveStateVersion')}>
              {action.effectiveStateVersion}
            </Descriptions.Item>
            <Descriptions.Item label={t('device.control.observationVersion')}>
              {action.observationVersion ?? '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('device.control.ruleType')}>
              {evaluation?.ruleType || '-'}
            </Descriptions.Item>
          </Descriptions>
          <Table<ParameterEvidenceRow>
            size="small"
            rowKey="path"
            columns={parameterColumns}
            dataSource={mergeParameterEvidence(action, t)}
            pagination={false}
            locale={{ emptyText: t('device.control.noParameterEvidence') }}
            scroll={{ x: 620 }}
          />
        </Space>
      ),
    }];

    return (
      <Space orientation="vertical" size={12} style={{ width: '100%' }}>
        {action.lastError && <Alert type="error" showIcon message={action.lastError} />}
        <Descriptions size="small" column={condensedMeta ? 1 : 2} bordered>
          <Descriptions.Item label={t('device.control.actionType')}>
            {t(`device.control.action.${action.actionType}`)}
          </Descriptions.Item>
          <Descriptions.Item label={t('table.status')}>
            <Tag color={action.status === 'verified' ? 'success' : action.status.includes('failed') || action.status === 'evidence_missing' ? 'warning' : 'processing'}>
              {t(`device.control.status.${action.status}`)}
            </Tag>
          </Descriptions.Item>
          {!condensedMeta && (
            <>
              <Descriptions.Item label={t('device.control.reason')} span={2}>
                {controlReasonLabel(action.reasonCode, t)}
              </Descriptions.Item>
              <Descriptions.Item label={t('device.control.triggeredAt')} span={2}>
                {formatSystemTime(action.createdAt)}
              </Descriptions.Item>
            </>
          )}
        </Descriptions>

        {evaluation && (
          <Descriptions
            title={t('device.control.evaluationEvidence')}
            size="small"
            column={condensedMeta ? 1 : 2}
            bordered
          >
            <Descriptions.Item label={t('device.control.position')}>
              {evaluation.latitude.toFixed(6)}, {evaluation.longitude.toFixed(6)}
            </Descriptions.Item>
            <Descriptions.Item label={t('device.control.observedAt')}>
              {formatSystemTime(evaluation.observedAt)}
            </Descriptions.Item>
            <Descriptions.Item label={t('device.control.ruleType')}>
              {ruleTypeLabel(evaluation.ruleType, t)}
            </Descriptions.Item>
            <Descriptions.Item label={t('device.control.confirmedState')}>
              {confirmedStateLabel(evaluation.confirmedState, t)}
            </Descriptions.Item>
            <Descriptions.Item label={t('device.control.signedDistance')} span={2}>
              {evaluation.signedDistanceMeters == null ? '-' : `${evaluation.signedDistanceMeters.toFixed(1)} m`}
            </Descriptions.Item>
          </Descriptions>
        )}

        <Collapse ghost size="small" items={technicalItems} />
      </Space>
    );
  };

  const summary = device.controlSummary;
  const latestAction = data?.items[0];
  return (
    <Space orientation="vertical" size={16} style={{ width: '100%' }}>
      {summary && (
        <>
          <Card
            size="small"
            title={(
              <Space size={8} wrap>
                <Tag
                  bordered={false}
                  color={summary.phase === 'deactivated' ? 'error' : summary.phase.includes('failed') ? 'warning' : 'processing'}
                >
                  {t(`device.control.phase.${summary.phase}`)}
                </Tag>
                <Typography.Text strong>{summary.sourceName || t('device.control.source.geofence')}</Typography.Text>
              </Space>
            )}
            extra={<Typography.Text type="secondary">{formatSystemTime(summary.triggeredAt)}</Typography.Text>}
          >
            <Typography.Text type="secondary">{t('device.control.reason')}</Typography.Text>
            <Typography.Paragraph style={{ margin: '4px 0 0', fontSize: 15 }}>
              {controlReasonLabel(summary.reasonCode, t)}
            </Typography.Paragraph>
          </Card>
          {summary.lastError && <Alert type="error" showIcon message={summary.lastError} />}
        </>
      )}

      {mode === 'compact' ? (
        isLoading ? (
          <div style={{ display: 'flex', justifyContent: 'center', padding: 24 }}><Spin size="small" /></div>
        ) : isError ? (
          <Alert type="error" showIcon message={t('device.control.loadFailed')} />
        ) : latestAction ? (
          <Space orientation="vertical" size={12} style={{ width: '100%' }}>
            <Typography.Title level={5} style={{ margin: 0 }}>
              {t('device.control.latestResult')}
            </Typography.Title>
            {renderAction(latestAction, true)}
          </Space>
        ) : null
      ) : isLoading ? (
        <div style={{ display: 'flex', justifyContent: 'center', padding: 48 }}><Spin /></div>
      ) : isError ? (
        <Alert type="error" showIcon message={t('device.control.loadFailed')} />
      ) : !data?.items.length ? (
        <Empty description={t('device.control.noHistory')} />
      ) : (
        <>
          <Collapse
            items={data.items.map((action) => ({
              key: action.id,
              label: (
                <Space wrap>
                  <Tag color={action.actionType === 'deactivate' ? 'red' : 'green'}>
                    {t(`device.control.action.${action.actionType}`)}
                  </Tag>
                  <Tag>{t(`device.control.status.${action.status}`)}</Tag>
                  <span>{controlSourceLabel(action.sourceType, action.sourceName, t)}</span>
                  <Typography.Text type="secondary">{formatSystemTime(action.createdAt)}</Typography.Text>
                </Space>
              ),
              children: renderAction(action),
            }))}
          />
          {data.total > pageSize && (
            <Pagination
              current={page}
              pageSize={pageSize}
              total={data.total}
              showSizeChanger={false}
              onChange={setPage}
              style={{ textAlign: 'right' }}
            />
          )}
        </>
      )}
    </Space>
  );
}

export default function DeviceControlReasonDrawer({ device, open, onClose, onOpenDetail }: DrawerProps) {
  const t = useT();
  return (
    <Drawer
      title={t('device.control.quickTitle', { sn: device?.sn ?? '' })}
      size={520}
      open={open}
      onClose={onClose}
      destroyOnClose
      extra={onOpenDetail ? <Button type="primary" onClick={onOpenDetail}>{t('device.control.viewInDetail')}</Button> : undefined}
    >
      {device && <DeviceControlReasonContent device={device} enabled={open} mode="compact" />}
    </Drawer>
  );
}
