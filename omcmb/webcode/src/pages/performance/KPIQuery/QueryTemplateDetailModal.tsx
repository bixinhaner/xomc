import { Descriptions, Modal, Space, Tag, Typography } from 'antd';

import type { QueryTemplate } from '@core/types/pmQuery';
import { useT } from '@/hooks/useT';
import dayjs from 'dayjs';

const { Text } = Typography;

interface QueryTemplateDetailModalProps {
  open: boolean;
  template: QueryTemplate | null;
  metricLabels: Record<string, string>;
  onClose: () => void;
}

export default function QueryTemplateDetailModal({ open, template, metricLabels, onClose }: QueryTemplateDetailModalProps) {
  const t = useT();
  if (!template) return null;

  const range = template.payload.timeRangePreset === 'custom'
    ? [template.payload.absoluteStart, template.payload.absoluteEnd].filter(Boolean).join(' — ') || '-'
    : t(`perf.kpiQuery.range.${template.payload.timeRangePreset.replaceAll('_', '')}`);
  const granularityLabels: Record<string, string> = {
    '15min': 'perf.dashboard.granular15min',
    hourly: 'perf.dashboard.granularHourly',
    daily: 'perf.dashboard.granularDaily',
    weekly: 'perf.dashboard.granularWeekly',
    monthly: 'perf.dashboard.granularMonthly',
  };
  const formatTimestamp = (value: string) => dayjs(value).format('YYYY-MM-DD HH:mm:ss');

  return (
    <Modal title={t('perf.kpiQuery.detail.title')} open={open} onCancel={onClose} footer={null} width={720} destroyOnHidden>
      <Descriptions bordered size="small" column={1}>
        <Descriptions.Item label={t('perf.kpiQuery.templateName')}>{template.name}</Descriptions.Item>
        <Descriptions.Item label={t('perf.kpiQuery.visibility')}>
          <Tag color={template.visibility === 'public' ? 'blue' : 'default'}>
            {t(template.visibility === 'public' ? 'perf.kpiQuery.publicOption' : 'perf.kpiQuery.privateOption')}
          </Tag>
        </Descriptions.Item>
        <Descriptions.Item label={t('perf.kpiQuery.description')}>{template.description || '-'}</Descriptions.Item>
        <Descriptions.Item label={t('perf.kpiQuery.deviceType')}>{template.payload.deviceType || '-'}</Descriptions.Item>
        <Descriptions.Item label={t('perf.kpiQuery.device')}>
          {template.payload.deviceSns.length === 0
            ? '-'
            : <Space wrap>{template.payload.deviceSns.map((sn) => <Tag key={sn}>{sn}</Tag>)}</Space>}
        </Descriptions.Item>
        <Descriptions.Item label={t('perf.kpiQuery.metric')}>
          {template.payload.metricPaths.length === 0
            ? '-'
            : <Space wrap>{template.payload.metricPaths.map((path) => <Tag key={path}>{metricLabels[path] ?? path}</Tag>)}</Space>}
        </Descriptions.Item>
        <Descriptions.Item label={t('perf.granularity')}>
          {t(granularityLabels[template.payload.granularity] ?? template.payload.granularity)}
        </Descriptions.Item>
        <Descriptions.Item label={t('perf.kpiQuery.timeRange')}>{range}</Descriptions.Item>
        <Descriptions.Item label={t('perf.kpiQuery.detail.createdAt')}><Text>{formatTimestamp(template.createdAt)}</Text></Descriptions.Item>
        <Descriptions.Item label={t('perf.kpiQuery.detail.updatedAt')}><Text>{formatTimestamp(template.updatedAt)}</Text></Descriptions.Item>
      </Descriptions>
    </Modal>
  );
}
