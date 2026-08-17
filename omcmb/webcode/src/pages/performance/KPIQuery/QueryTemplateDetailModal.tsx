import { Descriptions, Modal, Space, Tag, Typography, Button, App } from 'antd';
import { ExportOutlined } from '@ant-design/icons';

import type { QueryTemplate } from '@core/types/pmQuery';
import { useAllIndicators } from '@core/hooks/api/useIndicatorsLibrary';
import { formatIndicatorLevel, shouldShowIndicatorLevel } from '@core/utils/indicatorLevelDisplay';
import { saveBlob } from '@core/utils/saveBlob';
import { useT } from '@/hooks/useT';
import dayjs from 'dayjs';
import { buildTemplateMetricExportFilename, buildTemplateMetricExportText } from './templateMetricExport';
import { useTechnologyDictionary } from '@core/hooks/api/useTechnologyDictionary';

const { Text } = Typography;

interface QueryTemplateDetailModalProps {
  open: boolean;
  template: QueryTemplate | null;
  metricLabels: Record<string, string>;
  onClose: () => void;
}

export default function QueryTemplateDetailModal(props: QueryTemplateDetailModalProps) {
  if (!props.template) return null;

  return <QueryTemplateDetailModalContent {...props} template={props.template} />;
}

function QueryTemplateDetailModalContent({ open, template, metricLabels, onClose }: QueryTemplateDetailModalProps & { template: QueryTemplate }) {
  const t = useT();
  const { message } = App.useApp();
  const { labelForRadioMode } = useTechnologyDictionary();
  const deviceType = template.payload.deviceType ?? 'ENB';
  const { data: indicatorsData } = useAllIndicators(deviceType);
  const indicatorById = new Map((indicatorsData?.items ?? []).map((ind) => [ind.id, ind]));

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
  const handleExportMetrics = () => {
    if (template.payload.metricPaths.length === 0) {
      message.warning(t('perf.kpiQuery.exportMetricsEmpty'));
      return;
    }
    saveBlob(
      buildTemplateMetricExportText(template),
      buildTemplateMetricExportFilename(template),
      'text/plain;charset=utf-8',
    );
    message.success(t('perf.kpiQuery.exportMetricsSuccess', { count: template.payload.metricPaths.length }));
  };

  return (
    <Modal
      title={t('perf.kpiQuery.detail.title')}
      open={open}
      onCancel={onClose}
      footer={[
        <Button key="export" icon={<ExportOutlined />} onClick={handleExportMetrics}>
          {t('perf.kpiQuery.exportMetrics')}
        </Button>,
        <Button key="close" type="primary" onClick={onClose}>
          {t('common.close')}
        </Button>,
      ]}
      width={720}
      destroyOnHidden
    >
      <Descriptions bordered size="small" column={1}>
        <Descriptions.Item label={t('perf.kpiQuery.templateName')}>{template.name}</Descriptions.Item>
        <Descriptions.Item label={t('perf.kpiQuery.visibility')}>
          <Tag color={template.visibility === 'public' ? 'blue' : 'default'}>
            {t(template.visibility === 'public' ? 'perf.kpiQuery.publicOption' : 'perf.kpiQuery.privateOption')}
          </Tag>
        </Descriptions.Item>
        <Descriptions.Item label={t('perf.kpiQuery.description')}>{template.description || '-'}</Descriptions.Item>
        <Descriptions.Item label={t('perf.kpiQuery.deviceType')}>
          {template.payload.deviceType ? labelForRadioMode(template.payload.deviceType) : '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('perf.kpiQuery.device')}>
          {template.payload.deviceSns.length === 0
            ? '-'
            : <Space wrap>{template.payload.deviceSns.map((sn) => <Tag key={sn}>{sn}</Tag>)}</Space>}
        </Descriptions.Item>
        <Descriptions.Item label={t('perf.kpiQuery.metric')}>
          {template.payload.metricPaths.length === 0
            ? '-'
            : (
                <Space wrap>
                  {template.payload.metricPaths.map((path) => {
                    const indicator = indicatorById.get(path);
                    return (
                      <Tag key={path}>
                        {metricLabels[path] ?? path}
                        {shouldShowIndicatorLevel(deviceType) && t('perf.query.indicatorLevelInline', {
                          level: formatIndicatorLevel(indicator?.indicatorLevel, t),
                        })}
                      </Tag>
                    );
                  })}
                </Space>
              )}
        </Descriptions.Item>
        <Descriptions.Item label={t('perf.granularity')}>
          {t(granularityLabels[template.payload.granularity] ?? template.payload.granularity)}
        </Descriptions.Item>
        <Descriptions.Item label={t('perf.kpiQuery.timeRange')}>{range}</Descriptions.Item>
        <Descriptions.Item label={t('perf.kpiQuery.regularReport.title')}>
          <Tag color={template.payload.regularReport?.enabled ? 'blue' : 'default'}>
            {t(template.payload.regularReport?.enabled ? 'common.enabled' : 'common.disabled')}
          </Tag>
        </Descriptions.Item>
        {template.payload.regularReport?.enabled && (
          <>
            <Descriptions.Item label={t('perf.kpiQuery.regularReport.sendTime')}>
              {template.payload.regularReport.sendTime}
            </Descriptions.Item>
            <Descriptions.Item label={t('perf.kpiQuery.regularReport.period')}>
              <Space wrap>
                {template.payload.regularReport.periods.map((period) => (
                  <Tag key={period}>{t(`perf.kpiQuery.regularReport.period.${period === 'daily' ? 'day' : period === 'hourly' ? 'hour' : '15min'}`)}</Tag>
                ))}
              </Space>
            </Descriptions.Item>
            <Descriptions.Item label={t('perf.kpiQuery.regularReport.recipients')}>
              {template.payload.regularReport.recipients.join('; ') || '-'}
            </Descriptions.Item>
          </>
        )}
        <Descriptions.Item label={t('perf.kpiQuery.detail.createdAt')}><Text>{formatTimestamp(template.createdAt)}</Text></Descriptions.Item>
        <Descriptions.Item label={t('perf.kpiQuery.detail.updatedAt')}><Text>{formatTimestamp(template.updatedAt)}</Text></Descriptions.Item>
      </Descriptions>
    </Modal>
  );
}
