import { useMemo } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Alert, Button, Card, Descriptions, Skeleton, Space, Tag, Typography } from 'antd';
import { ArrowLeftOutlined, ReloadOutlined } from '@ant-design/icons';
import { useIndicatorGroupTree, useIndicatorInfo } from '@/hooks/api/useIndicator';
import { useT } from '@/hooks/useT';
import { useAppStore } from '@/store/appStore';
import type { IndicatorGroup } from '@/types/indicator';

const { Title, Text } = Typography;

function formatDateTime(value?: string): string {
  if (!value) return '-';
  const d = new Date(value);
  if (isNaN(d.getTime())) return value;
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function findGroupName(groups: IndicatorGroup[], id: string, locale: string): string {
  for (const group of groups) {
    if (group.id === id) {
      if (locale === 'en-US') return group.enName || group.cnName || id;
      return group.cnName || group.enName || id;
    }
    if (group.children?.length) {
      const found = findGroupName(group.children, id, locale);
      if (found) return found;
    }
  }
  return id || '-';
}

export default function IndicatorDetail() {
  const t = useT();
  const navigate = useNavigate();
  const locale = useAppStore((s) => s.locale);
  const { deviceType: rawDeviceType, indicatorId } = useParams();

  const deviceType = useMemo(() => {
    if (rawDeviceType === 'ENB' || rawDeviceType === 'GSM' || rawDeviceType === 'GNB') {
      return rawDeviceType;
    }
    return 'ENB';
  }, [rawDeviceType]);
  const isGNB = deviceType === 'GNB';

  const { data: indicator, isLoading, refetch } = useIndicatorInfo(indicatorId || '', deviceType);
  const { data: groupTreeData = [] } = useIndicatorGroupTree({ deviceType });

  const groupName = useMemo(() => {
    if (!indicator?.catagoryId) return '-';
    return findGroupName(groupTreeData, indicator.catagoryId, locale);
  }, [groupTreeData, indicator?.catagoryId, locale]);

  const indicatorName = useMemo(() => {
    if (!indicator) return '-';
    if (locale === 'en-US') return indicator.kpiNameEn || indicator.kpiName || indicator.kpiNameZh || '-';
    return indicator.kpiNameZh || indicator.kpiName || indicator.kpiNameEn || '-';
  }, [indicator, locale]);

  const indicatorDefinition = useMemo(() => {
    if (!indicator) return '-';
    if (locale === 'en-US') return indicator.definitionEn || indicator.definition || indicator.definitionZh || '-';
    return indicator.definitionZh || indicator.definition || indicator.definitionEn || '-';
  }, [indicator, locale]);

  if (isLoading) {
    return (
      <div style={{ padding: 24 }}>
        <Skeleton active paragraph={{ rows: 8 }} />
      </div>
    );
  }

  if (!indicator || !indicatorId) {
    return (
      <div style={{ padding: 24 }}>
        <Alert
          type="error"
          message={t('common.noData')}
          description={`ID: "${indicatorId || ''}"`}
          action={<Button onClick={() => void navigate('/performance/kpi-standard')}>{t('common.back')}</Button>}
        />
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Card size="small">
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 16 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <Button icon={<ArrowLeftOutlined />} onClick={() => void navigate('/performance/kpi-standard')}>
              {t('common.back')}
            </Button>
            <div>
              <Title level={4} style={{ margin: 0 }}>
                {indicatorName}
              </Title>
              <Text type="secondary" style={{ fontFamily: 'monospace', fontSize: 13 }}>
                {indicator.kpiId}
              </Text>
            </div>
          </div>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => void refetch()}>
              {t('common.refresh')}
            </Button>
          </Space>
        </div>
      </Card>

      <Card title={t('common.detail')}>
        <Descriptions bordered column={{ xs: 1, sm: 2, md: 3, lg: 4 }} size="small" style={{ marginBottom: 16 }}>
          <Descriptions.Item label={t('kpi.indicatorId')}>{indicator.kpiId}</Descriptions.Item>
          <Descriptions.Item label={t('kpi.indicatorName')}>{indicatorName}</Descriptions.Item>
          <Descriptions.Item label={t('kpi.customName')}>{indicator.custName || '-'}</Descriptions.Item>
          <Descriptions.Item label={t('perf.functionSet')}>{groupName}</Descriptions.Item>
          {!isGNB && <Descriptions.Item label={t('kpi.productType')}>{indicator.productType || '-'}</Descriptions.Item>}
          {!isGNB && (
            <Descriptions.Item label={t('kpi.level')}>
              {indicator.indicatorLevel === 'device' ? 'Device' : indicator.indicatorLevel === 'plmn' ? 'PLMN' : '-'}
            </Descriptions.Item>
          )}
          <Descriptions.Item label={t('kpi.unit')}>{indicator.unit || '-'}</Descriptions.Item>
          <Descriptions.Item label={t('kpi.statisType')}>{indicator.statisType || '-'}</Descriptions.Item>
          <Descriptions.Item label={t('kpi.measure')}>
            <Tag color={indicator.isEnable ? 'success' : 'default'}>
              {indicator.isEnable ? t('common.yes') : t('common.no')}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('kpi.indicatorType')}>
            <Space size={4}>
              <Tag color={indicator.indicatorType === 'counter' ? 'green' : 'blue'}>
                {indicator.indicatorType === 'counter' ? 'Counter' : 'KPI'}
              </Tag>
              <Tag color={indicator.isCustomize ? 'orange' : 'default'}>
                {indicator.isCustomize ? t('kpi.custom') : t('kpi.system')}
              </Tag>
            </Space>
          </Descriptions.Item>
          <Descriptions.Item label={t('kpi.updater')}>{indicator.updater || '-'}</Descriptions.Item>
          <Descriptions.Item label={t('kpi.updateTime')}>{formatDateTime(indicator.updateTime)}</Descriptions.Item>
          <Descriptions.Item label="Device Type">{deviceType}</Descriptions.Item>
        </Descriptions>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div>
            <Text strong>{t('common.description')}</Text>
            <div style={{ marginTop: 8, padding: '12px 14px', border: '1px solid var(--color-border)', borderRadius: 8, background: 'var(--color-bg-container)' }}>
              {indicatorDefinition}
            </div>
          </div>

          {indicator.indicatorType === 'kpi' && (
            <div>
              <Text strong>{t('kpi.calcFormula')}</Text>
              <div style={{ marginTop: 8, padding: '12px 14px', border: '1px solid var(--color-border)', borderRadius: 8, background: '#fafafa', color: 'rgba(0, 0, 0, 0.88)', whiteSpace: 'pre-wrap', wordBreak: 'break-all', fontFamily: 'Consolas, Monaco, monospace' }}>
                {indicator.arithmetic || '-'}
              </div>
            </div>
          )}
        </div>
      </Card>
    </div>
  );
}