import { Drawer, Descriptions, Tag, Space, Button } from 'antd';
import type { OperationLog } from '@core/types/system';
import { useT } from '@/hooks/useT';

interface LogDetailDrawerProps {
  open: boolean;
  log: OperationLog | null;
  onClose: () => void;
}

// 结果颜色映射
const resultColorMap: Record<string, string> = {
  '1': 'success',
  '0': 'error',
};

// 结果文本映射
const resultTextMap: Record<string, string> = {
  '1': '成功',
  '0': '失败',
};

export default function LogDetailDrawer({ open, log, onClose }: LogDetailDrawerProps) {
  const t = useT();

  if (!log) return null;

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
          {log.logName || log.module || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('log.detail')}>
          <div style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
            {log.detail || log.content || '-'}
          </div>
        </Descriptions.Item>
        <Descriptions.Item label={t('log.result')}>
          <Tag color={resultColorMap[log.result] || 'default'}>
            {resultTextMap[log.result] || log.result}
          </Tag>
        </Descriptions.Item>
        <Descriptions.Item label={t('log.reason')}>
          <div style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all', color: log.result === 'failure' ? '#ff4d4f' : undefined }}>
            {log.reason || log.message || '-'}
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
