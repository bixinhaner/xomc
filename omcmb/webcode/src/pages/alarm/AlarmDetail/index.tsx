import React, { useMemo } from 'react';
import {
  Descriptions,
  Drawer,
  Space,
  Tag,
  Typography,
} from 'antd';
import {
  UserOutlined,
} from '@ant-design/icons';
import { useAlarmById } from '@core/hooks/api/useAlarms';
import type { Alarm, DealState, EventType } from '@core/types/alarm';
import { useT } from '@/hooks/useT';
import { formatBaseStationTypeLabel } from '../utils/baseStationType';
import { formatSystemTime } from '@core/utils/systemTime';

const { Text, Paragraph } = Typography;

export interface AlarmDetailProps {
  alarm: Alarm | null;
  open: boolean;
  onClose: () => void;
}

// 告警级别颜色 - 专业配色方案
const SEVERITY_CONFIG: Record<string, { color: string; bgColor: string }> = {
  critical: { color: '#E53935', bgColor: '#FFEBEE' },
  major: { color: '#FB8C00', bgColor: '#FFF3E0' },
  minor: { color: '#FDD835', bgColor: '#FFFDE7' },
  warning: { color: '#42A5F5', bgColor: '#E3F2FD' },
};

// 告警状态配置
const DEAL_STATE_CONFIG: Record<DealState, { label: string; color: string }> = {
  '0': { label: 'alarm.dealState.unconfirmedUncleared', color: '#E88282' },
  '1': { label: 'alarm.dealState.confirmedUncleared', color: '#E88282' },
  '2': { label: 'alarm.dealState.unconfirmedCleared', color: '#67D972' },
  '3': { label: 'alarm.dealState.confirmedCleared', color: '#67D972' },
};

// 事件类型配置
const EVENT_TYPE_CONFIG: Record<EventType, string> = {
  'communication': 'alarm.eventType.communication',
  'qualityOfService': 'alarm.eventType.qualityOfService',
  'processingError': 'alarm.eventType.processingError',
  'device': 'alarm.eventType.device',
  'environment': 'alarm.eventType.environment',
  'performance': 'alarm.eventType.performance',
};

const LONG_TEXT_STYLE: React.CSSProperties = {
  margin: 0,
  whiteSpace: 'pre-wrap',
  overflowWrap: 'anywhere',
  wordBreak: 'break-word',
};

const AlarmDetail: React.FC<AlarmDetailProps> = ({ alarm, open, onClose }) => {
  const t = useT();
  const { data: fetchedAlarm } = useAlarmById(alarm?.id ?? '');
  const resolvedAlarm = fetchedAlarm ?? alarm;

  const SEVERITY_LABEL: Record<string, string> = useMemo(() => ({
    critical: t('alarm.severity.critical'),
    major: t('alarm.severity.major'),
    minor: t('alarm.severity.minor'),
    warning: t('alarm.severity.warning'),
  }), [t]);

  const additionalInfoEntries = useMemo(
    () => Object.entries(resolvedAlarm?.additionalInfo ?? {}).filter(
      ([key]) => key !== 'additional_text' && key !== 'managed_object_instance' && key !== 'additional_information'
    ),
    [resolvedAlarm]
  );

  // 判断是否已确认 (dealState 为 1 或 3)
  const isConfirmed = resolvedAlarm?.dealState === '1' || resolvedAlarm?.dealState === '3';
  // 判断是否已清除 (dealState 为 2 或 3)
  const isCleared = resolvedAlarm?.dealState === '2' || resolvedAlarm?.dealState === '3';

  // 格式化时间
  const formatTime = (time?: string) => {
    if (!time) return '-';
    return formatSystemTime(String(time));
  };

  if (!resolvedAlarm) {
    return (
      <Drawer title={t('alarm.detail')} open={open} onClose={onClose} size={600}>
        <div style={{ padding: '40px 0', textAlign: 'center', color: '#8c8c8c' }}>
          {t('common.noData')}
        </div>
      </Drawer>
    );
  }

  const severityConfig = SEVERITY_CONFIG[resolvedAlarm.severity] || SEVERITY_CONFIG.warning;
  const additionalText = resolvedAlarm.additionalText || resolvedAlarm.additionalInfo?.additional_text;
  const additionalInformation = resolvedAlarm.additionalInfo?.additional_information;

  return (
    <Drawer
      title={
        <Space>
          <Tag
            style={{
              color: severityConfig.color,
              backgroundColor: severityConfig.bgColor,
              border: 'none',
            }}
          >
            {SEVERITY_LABEL[resolvedAlarm.severity] ?? resolvedAlarm.severity}
          </Tag>
          <span style={{ whiteSpace: 'normal', overflowWrap: 'anywhere', wordBreak: 'break-word' }}>
            {resolvedAlarm.probableCause || resolvedAlarm.description || t('alarm.detail')}
          </span>
        </Space>
      }
      open={open}
      onClose={onClose}
      width={600}
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
        {/* 基本信息 */}
        <section>
          <Text
            type="secondary"
            strong
            style={{ fontSize: 12, textTransform: 'uppercase', letterSpacing: 1, display: 'block', marginBottom: 10 }}
          >
            {t('alarm.basicInfo')}
          </Text>
          <Descriptions column={1} size="small" bordered>
            {/* 1. 可能原因 */}
            <Descriptions.Item label={t('alarm.alarmIdentifier')}>
              <Text style={{ fontFamily: 'monospace' }}>{resolvedAlarm.alarmIdentifier || '-'}</Text>
            </Descriptions.Item>
            {/* 2. 可能原因 */}
            <Descriptions.Item label={t('alarm.possibleCause')}>
              {resolvedAlarm.probableCause || '-'}
            </Descriptions.Item>
            {/* 3. 严重程度 */}
            <Descriptions.Item label={t('alarm.severity')}>
              <Tag
                style={{
                  color: severityConfig.color,
                  backgroundColor: severityConfig.bgColor,
                  border: 'none',
                }}
              >
                {SEVERITY_LABEL[resolvedAlarm.severity] ?? resolvedAlarm.severity}
              </Tag>
            </Descriptions.Item>
            {/* 5. 事件类型 */}
            <Descriptions.Item label={t('alarm.eventType')}>
              {t(EVENT_TYPE_CONFIG[resolvedAlarm.eventType] || 'common.unknown')}
            </Descriptions.Item>
          </Descriptions>
        </section>

        {/* 设备信息 */}
        <section>
          <Text
            type="secondary"
            strong
            style={{ fontSize: 12, textTransform: 'uppercase', letterSpacing: 1, display: 'block', marginBottom: 10 }}
          >
            {t('alarm.deviceInfo')}
          </Text>
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label={t('alarm.deviceSn')}>
              <Text style={{ fontFamily: 'monospace' }}>{resolvedAlarm.deviceSn || '-'}</Text>
            </Descriptions.Item>
            {/* 9. 告警源 */}
            <Descriptions.Item label={t('alarm.neTypeCol')}>
              {formatBaseStationTypeLabel(resolvedAlarm.neType)}
            </Descriptions.Item>
          </Descriptions>
        </section>

        {/* 状态与时间 */}
        <section>
          <Text
            type="secondary"
            strong
            style={{ fontSize: 12, textTransform: 'uppercase', letterSpacing: 1, display: 'block', marginBottom: 10 }}
          >
            {t('alarm.statusAndTime')}
          </Text>
          <Descriptions column={1} size="small" bordered>
            {/* 11. 告警状态 */}
            <Descriptions.Item label={t('alarm.dealState')}>
              <span style={{ color: DEAL_STATE_CONFIG[resolvedAlarm.dealState]?.color || '#666' }}>
                {t(DEAL_STATE_CONFIG[resolvedAlarm.dealState]?.label || 'common.unknown')}
              </span>
            </Descriptions.Item>
            {/* 12. 故障时间 */}
            <Descriptions.Item label={t('alarm.eventTime')}>
              {formatTime(resolvedAlarm.eventTime)}
            </Descriptions.Item>
            {/* 13. 更新时间 */}
            <Descriptions.Item label={t('alarm.updTime')}>
              {formatTime(resolvedAlarm.updTime)}
            </Descriptions.Item>
            {/* 14. 确认人 - 已确认时显示 */}
            {isConfirmed && (
              <Descriptions.Item label={t('alarm.dealUser')}>
                <Space>
                  <UserOutlined />
                  {resolvedAlarm.dealUser || '-'}
                </Space>
              </Descriptions.Item>
            )}
            {/* 16. 告警清除人 - 已清除时显示 */}
            {isCleared && (
              <Descriptions.Item label={t('alarm.clearUser')}>
                <Space>
                  <UserOutlined />
                  {resolvedAlarm.clearUser || '-'}
                </Space>
              </Descriptions.Item>
            )}
            {/* 17. 告警清除时间 - 已清除时显示 */}
            {isCleared && (
              <Descriptions.Item label={t('alarm.clearTime')}>
                {formatTime(resolvedAlarm.clearTime)}
              </Descriptions.Item>
            )}
          </Descriptions>
        </section>

        {/* 处理信息 */}
        <section>
          <Text
            type="secondary"
            strong
            style={{ fontSize: 12, textTransform: 'uppercase', letterSpacing: 1, display: 'block', marginBottom: 10 }}
          >
            {t('alarm.handleInfo')}
          </Text>
          <Descriptions column={1} size="small" bordered>
            {/* 确认描述 */}
            <Descriptions.Item label={t('alarm.dealMemo')}>
              <Paragraph style={LONG_TEXT_STYLE}>{resolvedAlarm.dealMemo || '-'}</Paragraph>
            </Descriptions.Item>
            {/* 清除描述 */}
            {isCleared && (
              <Descriptions.Item label={t('alarm.clearMemo')}>
                <Paragraph style={LONG_TEXT_STYLE}>{resolvedAlarm.clearMemo || '-'}</Paragraph>
              </Descriptions.Item>
            )}
          </Descriptions>
        </section>

        <section>
          <Text
            type="secondary"
            strong
            style={{ fontSize: 12, textTransform: 'uppercase', letterSpacing: 1, display: 'block', marginBottom: 10 }}
          >
            {t('common.details')}
          </Text>
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label={t('alarm.additionalText')}>
              <Paragraph style={LONG_TEXT_STYLE}>{additionalText || '-'}</Paragraph>
            </Descriptions.Item>
            <Descriptions.Item label={t('alarm.additionalInfo')}>
              {additionalInformation || additionalInfoEntries.length > 0 ? (
                <Space orientation="vertical" size={4} style={{ width: '100%' }}>
                  {additionalInformation ? <Paragraph style={LONG_TEXT_STYLE}>{additionalInformation}</Paragraph> : null}
                  {additionalInfoEntries.map(([key, value]) => (
                    <div key={key} style={{ display: 'flex', gap: 8, alignItems: 'flex-start' }}>
                      <Text type="secondary" style={{ minWidth: 140, fontFamily: 'monospace' }}>{key}</Text>
                      <Paragraph style={{ ...LONG_TEXT_STYLE, flex: 1 }}>{value || '-'}</Paragraph>
                    </div>
                  ))}
                </Space>
              ) : (
                '-'
              )}
            </Descriptions.Item>
          </Descriptions>
        </section>
      </div>
    </Drawer>
  );
};

export default AlarmDetail;
