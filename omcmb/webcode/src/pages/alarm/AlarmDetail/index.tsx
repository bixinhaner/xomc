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
import type { Alarm, DealState, EventType } from '@core/types/alarm';
import { useT } from '@/hooks/useT';

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

const AlarmDetail: React.FC<AlarmDetailProps> = ({ alarm, open, onClose }) => {
  const t = useT();

  const SEVERITY_LABEL: Record<string, string> = useMemo(() => ({
    critical: t('alarm.severity.critical'),
    major: t('alarm.severity.major'),
    minor: t('alarm.severity.minor'),
    warning: t('alarm.severity.warning'),
  }), [t]);

  // 判断是否已确认 (dealState 为 1 或 3)
  const isConfirmed = alarm?.dealState === '1' || alarm?.dealState === '3';
  // 判断是否已清除 (dealState 为 2 或 3)
  const isCleared = alarm?.dealState === '2' || alarm?.dealState === '3';

  // 格式化时间
  const formatTime = (time?: string) => {
    if (!time) return '-';
    return new Date(String(time)).toLocaleString('zh-CN');
  };

  if (!alarm) {
    return (
      <Drawer title={t('alarm.detail')} open={open} onClose={onClose} width={600}>
        <div style={{ padding: '40px 0', textAlign: 'center', color: '#8c8c8c' }}>
          {t('common.noData')}
        </div>
      </Drawer>
    );
  }

  const severityConfig = SEVERITY_CONFIG[alarm.severity] || SEVERITY_CONFIG.warning;

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
            {SEVERITY_LABEL[alarm.severity] ?? alarm.severity}
          </Tag>
          <span>{alarm.alarmIdentifier || t('alarm.detail')}</span>
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
            {/* 1. 序号 */}
            <Descriptions.Item label={t('alarm.alarmId')}>
              <Text style={{ fontFamily: 'monospace', fontWeight: 600 }}>{alarm.id}</Text>
            </Descriptions.Item>
            {/* 2. 告警编码 */}
            <Descriptions.Item label={t('alarm.alarmIdentifier')}>
              <Text style={{ fontFamily: 'monospace' }}>{alarm.alarmIdentifier}</Text>
            </Descriptions.Item>
            {/* 3. 可能原因 */}
            <Descriptions.Item label={t('alarm.possibleCause')}>
              {alarm.probableCause || '-'}
            </Descriptions.Item>
            {/* 4. 具体故障 */}
            <Descriptions.Item label={t('alarm.specificProblem')}>
              {alarm.description || '-'}
            </Descriptions.Item>
            {/* 5. 严重程度 */}
            <Descriptions.Item label={t('alarm.severity')}>
              <Tag
                style={{
                  color: severityConfig.color,
                  backgroundColor: severityConfig.bgColor,
                  border: 'none',
                }}
              >
                {SEVERITY_LABEL[alarm.severity] ?? alarm.severity}
              </Tag>
            </Descriptions.Item>
            {/* 6. 事件类型 */}
            <Descriptions.Item label={t('alarm.eventType')}>
              {t(EVENT_TYPE_CONFIG[alarm.eventType] || 'common.unknown')}
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
            {/* 9. 告警源 */}
            <Descriptions.Item label={t('alarm.neType')}>
              {alarm.neType}
            </Descriptions.Item>
            {/* 10. 网元定位 */}
            <Descriptions.Item label={t('alarm.equipInfo')}>
              {alarm.equipInfo}
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
              <span style={{ color: DEAL_STATE_CONFIG[alarm.dealState]?.color || '#666' }}>
                {t(DEAL_STATE_CONFIG[alarm.dealState]?.label || 'common.unknown')}
              </span>
            </Descriptions.Item>
            {/* 12. 故障时间 */}
            <Descriptions.Item label={t('alarm.eventTime')}>
              {formatTime(alarm.eventTime)}
            </Descriptions.Item>
            {/* 13. 更新时间 */}
            <Descriptions.Item label={t('alarm.updTime')}>
              {formatTime(alarm.updTime)}
            </Descriptions.Item>
            {/* 14. 确认人 - 已确认时显示 */}
            {isConfirmed && (
              <Descriptions.Item label={t('alarm.dealUser')}>
                <Space>
                  <UserOutlined />
                  {alarm.dealUser || '-'}
                </Space>
              </Descriptions.Item>
            )}
            {/* 15. 确认时间 - 已确认时显示 */}
            {isConfirmed && (
              <Descriptions.Item label={t('alarm.dealTime')}>
                {formatTime(alarm.dealTime)}
              </Descriptions.Item>
            )}
            {/* 16. 告警清除人 - 已清除时显示 */}
            {isCleared && (
              <Descriptions.Item label={t('alarm.clearUser')}>
                <Space>
                  <UserOutlined />
                  {alarm.clearUser || '-'}
                </Space>
              </Descriptions.Item>
            )}
            {/* 17. 告警清除时间 - 已清除时显示 */}
            {isCleared && (
              <Descriptions.Item label={t('alarm.clearTime')}>
                {formatTime(alarm.clearTime)}
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
              <Paragraph style={{ margin: 0 }}>{alarm.dealMemo || '-'}</Paragraph>
            </Descriptions.Item>
            {/* 清除描述 */}
            {isCleared && (
              <Descriptions.Item label={t('alarm.clearMemo')}>
                <Paragraph style={{ margin: 0 }}>{alarm.clearMemo || '-'}</Paragraph>
              </Descriptions.Item>
            )}
          </Descriptions>
        </section>
      </div>
    </Drawer>
  );
};

export default AlarmDetail;
