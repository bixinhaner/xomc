import React, { useCallback, useMemo } from 'react';
import {
  Button,
  Descriptions,
  Drawer,
  Space,
  Tag,
  Timeline,
  Typography,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  InfoCircleOutlined,
  UserOutlined,
} from '@ant-design/icons';
import type { Alarm } from '@/types/alarm';
import { useT } from '@/hooks/useT';

const { Text, Paragraph } = Typography;

export interface AlarmDetailProps {
  alarm: Alarm | null;
  open: boolean;
  onClose: () => void;
}

const SEVERITY_TAG_COLOR: Record<string, string> = {
  critical: 'red',
  major: 'orange',
  minor: 'gold',
  warning: 'blue',
};

const AlarmDetail: React.FC<AlarmDetailProps> = ({ alarm, open, onClose }) => {
  const t = useT();

  const SEVERITY_LABEL: Record<string, string> = useMemo(() => ({
    critical: t('alarm.severity.critical'),
    major: t('alarm.severity.major'),
    minor: t('alarm.severity.minor'),
    warning: t('alarm.severity.warning'),
  }), [t]);

  const handleAcknowledge = useCallback(() => {
    // In a real app, call acknowledge API
    onClose();
  }, [onClose]);

  const handleClear = useCallback(() => {
    // In a real app, call clear API
    onClose();
  }, [onClose]);

  // Mock operation history for the alarm
  const operationHistory = useMemo(() => {
    if (!alarm) return [];
    const history: Array<{
      time: string;
      action: string;
      operator: string;
      note?: string;
      color: string;
      icon: React.ReactNode;
    }> = [
      {
        time: alarm.alarmTime,
        action: t('alarm.time'),
        operator: 'System',
        note: alarm.alarmContent,
        color: alarm.severity === 'critical' ? 'red' : alarm.severity === 'major' ? 'orange' : 'blue',
        icon: <InfoCircleOutlined />,
      },
    ];

    if (alarm.ackStatus === 'acknowledged' && alarm.ackTime) {
      history.push({
        time: alarm.ackTime,
        action: t('alarm.acknowledge'),
        operator: alarm.ackUser ?? 'Admin',
        note: alarm.ackNote ?? t('alarm.ackStatus.acknowledged'),
        color: 'blue',
        icon: <CheckCircleOutlined style={{ color: 'var(--color-primary-600)' }} />,
      });
    }

    if (alarm.clearTime) {
      history.push({
        time: alarm.clearTime,
        action: t('alarm.clear'),
        operator: alarm.ackUser ?? 'System',
        note: t('alarm.clear'),
        color: 'green',
        icon: <CloseCircleOutlined style={{ color: '#52C41A' }} />,
      });
    }

    return history.reverse(); // Most recent first
  }, [alarm, t]);

  if (!alarm) {
    return (
      <Drawer title={t('common.detail')} open={open} onClose={onClose} width={560}>
        <div style={{ padding: '40px 0', textAlign: 'center', color: '#8c8c8c' }}>
          {t('common.noData')}
        </div>
      </Drawer>
    );
  }

  return (
    <Drawer
      title={
        <Space>
          <Tag color={SEVERITY_TAG_COLOR[alarm.severity] ?? 'default'}>
            {SEVERITY_LABEL[alarm.severity] ?? alarm.severity}
          </Tag>
          <span>{alarm.alarmName}</span>
        </Space>
      }
      open={open}
      onClose={onClose}
      width={560}
      extra={
        <Space>
          {alarm.ackStatus === 'unacknowledged' && (
            <Button size="small" type="primary" ghost onClick={handleAcknowledge}>
              {t('alarm.acknowledge')}
            </Button>
          )}
          {alarm.isActive && (
            <Button size="small" danger ghost onClick={handleClear}>
              {t('alarm.clear')}
            </Button>
          )}
        </Space>
      }
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
        {/* Basic Info */}
        <section>
          <Text
            type="secondary"
            strong
            style={{ fontSize: 12, textTransform: 'uppercase', letterSpacing: 1, display: 'block', marginBottom: 10 }}
          >
            {t('common.detail')}
          </Text>
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label={t('alarm.code')}>
              <Text style={{ fontFamily: 'monospace', fontWeight: 600 }}>{alarm.alarmCode}</Text>
            </Descriptions.Item>
            <Descriptions.Item label={t('alarm.name')}>{alarm.alarmName}</Descriptions.Item>
            <Descriptions.Item label={t('alarm.severity')}>
              <Tag color={SEVERITY_TAG_COLOR[alarm.severity] ?? 'default'}>
                {SEVERITY_LABEL[alarm.severity] ?? alarm.severity}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t('alarm.type')}>{alarm.alarmType ?? '-'}</Descriptions.Item>
            <Descriptions.Item label={t('alarm.occurTime')}>
              {new Date(alarm.alarmTime).toLocaleString('zh-CN')}
            </Descriptions.Item>
            {alarm.clearTime && (
              <Descriptions.Item label={t('alarm.clearTime')}>
                {new Date(alarm.clearTime).toLocaleString('zh-CN')}
              </Descriptions.Item>
            )}
            <Descriptions.Item label={t('alarm.ackStatus')}>
              <Tag color={alarm.ackStatus === 'acknowledged' ? 'success' : 'warning'}>
                {alarm.ackStatus === 'acknowledged' ? t('alarm.ackStatus.acknowledged') : t('alarm.ackStatus.unacknowledged')}
              </Tag>
            </Descriptions.Item>
            {alarm.ackUser && (
              <Descriptions.Item label={t('alarm.ackUser')}>
                <Space>
                  <UserOutlined />
                  {alarm.ackUser}
                </Space>
              </Descriptions.Item>
            )}
            <Descriptions.Item label={t('table.status')}>
              <Tag color={alarm.isActive ? 'error' : 'default'}>
                {alarm.isActive ? t('alarm.active') : t('alarm.clear')}
              </Tag>
            </Descriptions.Item>
          </Descriptions>
        </section>

        {/* Device Info */}
        <section>
          <Text
            type="secondary"
            strong
            style={{ fontSize: 12, textTransform: 'uppercase', letterSpacing: 1, display: 'block', marginBottom: 10 }}
          >
            {t('alarm.deviceName')}
          </Text>
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label={t('alarm.deviceSn')}>
              <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{alarm.deviceSn}</Text>
            </Descriptions.Item>
            <Descriptions.Item label={t('alarm.deviceName')}>{alarm.deviceName}</Descriptions.Item>
            <Descriptions.Item label={t('alarm.neType')}>{alarm.neType}</Descriptions.Item>
            <Descriptions.Item label={t('alarm.source')}>{alarm.alarmSource}</Descriptions.Item>
            {alarm.alarmLocation && (
              <Descriptions.Item label={t('alarm.location')}>{alarm.alarmLocation}</Descriptions.Item>
            )}
          </Descriptions>
        </section>

        {/* Alarm Content */}
        <section>
          <Text
            type="secondary"
            strong
            style={{ fontSize: 12, textTransform: 'uppercase', letterSpacing: 1, display: 'block', marginBottom: 10 }}
          >
            {t('alarm.content')}
          </Text>
          <div
            style={{
              background: '#fafafa',
              border: '1px solid #f0f0f0',
              borderRadius: 6,
              padding: '12px 16px',
            }}
          >
            <Paragraph style={{ margin: 0, fontSize: 13, lineHeight: 1.8 }}>
              {alarm.alarmContent}
            </Paragraph>
          </div>
          {alarm.ackNote && (
            <div
              style={{
                marginTop: 8,
                background: '#e6f4ff',
                border: '1px solid #91caff',
                borderRadius: 6,
                padding: '10px 14px',
              }}
            >
              <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4 }}>
                {t('alarm.ackNote')}
              </Text>
              <Paragraph style={{ margin: 0, fontSize: 13 }}>{alarm.ackNote}</Paragraph>
            </div>
          )}
        </section>

        {/* Operation History */}
        <section>
          <Text
            type="secondary"
            strong
            style={{ fontSize: 12, textTransform: 'uppercase', letterSpacing: 1, display: 'block', marginBottom: 14 }}
          >
            {t('table.time')}
          </Text>
          <Timeline
            items={operationHistory.map((h) => ({
              color: h.color,
              dot: h.icon,
              children: (
                <div>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <Text strong style={{ fontSize: 13 }}>
                      {h.action}
                    </Text>
                    <Text type="secondary" style={{ fontSize: 11 }}>
                      {new Date(h.time).toLocaleString('zh-CN')}
                    </Text>
                  </div>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {t('table.operator')}: {h.operator}
                  </Text>
                  {h.note && (
                    <Paragraph
                      style={{ margin: '4px 0 0', fontSize: 12, color: '#595959' }}
                    >
                      {h.note}
                    </Paragraph>
                  )}
                </div>
              ),
            }))}
          />
        </section>
      </div>
    </Drawer>
  );
};

export default AlarmDetail;
