import { Drawer, Descriptions, Tag, Space, Button } from 'antd';
import type { OperationLog } from '@core/types/system';
import { useT } from '@/hooks/useT';
import {
  localizeAuditAction,
  localizeAuditReason,
} from '../geofenceAuditText';

interface LogDetailDrawerProps {
  open: boolean;
  log: OperationLog | null;
  onClose: () => void;
}

// 结果颜色映射
const resultColorMap: Record<string, string> = {
  '1': 'success',
  '0': 'error',
  success: 'success',
  failure: 'error',
};

export default function LogDetailDrawer({ open, log, onClose }: LogDetailDrawerProps) {
  const t = useT();

  if (!log) return null;

  const rawDetailText = log.detail || log.content || log.message || log.reason || '-';
  const detailText = localizeAuditReason(rawDetailText, t);
  const isFailure = String(log.result).toLowerCase() === 'failure' || String(log.result) === '0';
  const reasonText = isFailure
    ? localizeAuditReason(log.reason || log.message || '-', t)
    : '-';
  const resultTextMap: Record<string, string> = {
    '1': t('log.success'),
    '0': t('log.failure'),
    success: t('log.success'),
    failure: t('log.failure'),
  };

  return (
    <Drawer
      title={t('log.detail')}
      placement="right"
      size={520}
      open={open}
      onClose={onClose}
      footer={
        <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
          <Button onClick={onClose}>{t('common.close')}</Button>
        </Space>
      }
    >
      <Descriptions column={1} bordered size="small">
        <Descriptions.Item label={t('log.operator')}>
          {log.operator}
        </Descriptions.Item>
        <Descriptions.Item label={t('log.clientIp')}>
          <span style={{ fontFamily: 'monospace' }}>{log.clientIp}</span>
        </Descriptions.Item>
        <Descriptions.Item label={t('log.logName')}>
          {localizeAuditAction(log.logName || log.module || '', t)}
        </Descriptions.Item>
        <Descriptions.Item label={t('log.detail')}>
          <div style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
            {detailText}
          </div>
        </Descriptions.Item>
        <Descriptions.Item label={t('log.result')}>
          <Tag color={resultColorMap[log.result] || 'default'}>
            {resultTextMap[log.result] || log.result}
          </Tag>
        </Descriptions.Item>
        <Descriptions.Item label={t('log.reason')}>
          <div style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all', color: isFailure ? '#ff4d4f' : undefined }}>
            {reasonText}
          </div>
        </Descriptions.Item>
        <Descriptions.Item label={t('log.startTime')}>
          {log.startTime || log.operationTime || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('log.endTime')}>
          {log.endTime || '-'}
        </Descriptions.Item>
      </Descriptions>
    </Drawer>
  );
}
