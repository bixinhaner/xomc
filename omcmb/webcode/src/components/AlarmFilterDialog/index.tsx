import React, { useMemo, useState } from 'react';
import {
  Button,
  Checkbox,
  Col,
  DatePicker,
  Form,
  Input,
  Modal,
  Row,
  Select,
  Space,
  Tabs,
  Tag,
} from 'antd';
import type { AlarmSeverity } from '@core/types/common';
import { useT } from '@/hooks/useT';

const { RangePicker } = DatePicker;

export interface AlarmFilterDialogProps {
  visible: boolean;
  onOk: (filter: AlarmFilterDialogValue) => void;
  onCancel: () => void;
  initialValues?: AlarmFilterDialogValue;
}

export interface AlarmFilterDialogValue {
  severity?: AlarmSeverity[];
  alarmType?: string;
  keyword?: string;
  deviceSns?: string[];
  alarmLocation?: string;
  locationDetail?: string;
  timeRange?: [string, string];
  includeCleared?: boolean;
  includeAcknowledged?: boolean;
  sortBy?: string;
  limit?: number;
}

const AlarmFilterDialog: React.FC<AlarmFilterDialogProps> = ({
  visible,
  onOk,
  onCancel,
  initialValues,
}) => {
  const t = useT();
  const [form] = Form.useForm<AlarmFilterDialogValue>();
  const [activeTab, setActiveTab] = useState('basic');
  const [selectedSeverities, setSelectedSeverities] = useState<AlarmSeverity[]>(
    initialValues?.severity ?? []
  );

  const SEVERITY_OPTIONS: { label: React.ReactNode; value: AlarmSeverity; color: string }[] = useMemo(
    () => [
      { label: t('alarm.severity.critical'), value: 'critical', color: '#F5222D' },
      { label: t('alarm.severity.major'), value: 'major', color: '#FA8C16' },
      { label: t('alarm.severity.minor'), value: 'minor', color: '#FAAD14' },
      { label: t('alarm.severity.warning'), value: 'warning', color: '#1890FF' },
    ],
    [t],
  );

  const ALARM_TYPE_OPTIONS = useMemo(
    () => [
      { label: t('alarm.filter.allTypes'), value: '' },
      { label: t('alarm.filter.type.device'), value: 'device' },
      { label: t('alarm.filter.type.link'), value: 'link' },
      { label: t('alarm.filter.type.performance'), value: 'performance' },
      { label: t('alarm.filter.type.security'), value: 'security' },
      { label: t('alarm.filter.type.environment'), value: 'environment' },
    ],
    [t],
  );

  const handleOk = () => {
    const values = form.getFieldsValue() as AlarmFilterDialogValue;
    onOk({ ...values, severity: selectedSeverities });
  };

  const handleReset = () => {
    form.resetFields();
    setSelectedSeverities([]);
  };

  const toggleSeverity = (sv: AlarmSeverity) => {
    setSelectedSeverities((prev) =>
      prev.includes(sv) ? prev.filter((s) => s !== sv) : [...prev, sv]
    );
  };

  const tabItems = [
    {
      key: 'basic',
      label: t('alarm.filter.basic'),
      children: (
        <div style={{ padding: '12px 0' }}>
          {/* Severity */}
          <Form.Item label={t('alarm.filter.severity')}>
            <Space wrap>
              {SEVERITY_OPTIONS.map((opt) => {
                const isSelected = selectedSeverities.includes(opt.value);
                return (
                  <Tag
                    key={opt.value}
                    color={isSelected ? opt.color : undefined}
                    style={{
                      cursor: 'pointer',
                      border: `1px solid ${opt.color}`,
                      color: isSelected ? '#fff' : opt.color,
                      background: isSelected ? opt.color : 'transparent',
                      padding: '2px 12px',
                      fontSize: 13,
                      userSelect: 'none',
                    }}
                    onClick={() => toggleSeverity(opt.value)}
                  >
                    {opt.label}
                  </Tag>
                );
              })}
            </Space>
          </Form.Item>

          <Row gutter={12}>
            <Col span={12}>
              <Form.Item name="alarmType" label={t('alarm.filter.alarmType')}>
                <Select options={ALARM_TYPE_OPTIONS} placeholder={t('common.pleaseSelect')} allowClear />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="keyword" label={t('alarm.filter.keyword')}>
                <Input placeholder={t('alarm.filter.alarmNameCode')} allowClear />
              </Form.Item>
            </Col>
          </Row>
        </div>
      ),
    },
    {
      key: 'device',
      label: t('alarm.filter.device'),
      children: (
        <div style={{ padding: '12px 0' }}>
          <Form.Item name="deviceSns" label={t('alarm.filter.deviceSn')}>
            <Select
              mode="tags"
              placeholder={t('alarm.filter.deviceSnPlaceholder')}
              tokenSeparators={[',', ';', ' ']}
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item name="deviceModel" label={t('alarm.filter.deviceModel')}>
            <Input placeholder={t('common.placeholder')} allowClear />
          </Form.Item>
          <Form.Item name="deviceVendor" label={t('alarm.filter.vendor')}>
            <Select
              placeholder={t('common.pleaseSelect')}
              allowClear
              options={[
                { label: '华为', value: 'huawei' },
                { label: '中兴', value: 'zte' },
                { label: '爱立信', value: 'ericsson' },
                { label: '诺基亚', value: 'nokia' },
              ]}
            />
          </Form.Item>
        </div>
      ),
    },
    {
      key: 'location',
      label: t('alarm.filter.location'),
      children: (
        <div style={{ padding: '12px 0' }}>
          <Form.Item name="alarmLocation" label={t('alarm.filter.locationId')}>
            <Input placeholder={t('common.placeholder')} allowClear />
          </Form.Item>
          <Form.Item name="locationDetail" label={t('alarm.filter.locationDetail')}>
            <Input placeholder={t('common.placeholder')} allowClear />
          </Form.Item>
          <Form.Item name="region" label={t('alarm.filter.region')}>
            <Input placeholder={t('common.placeholder')} allowClear />
          </Form.Item>
        </div>
      ),
    },
    {
      key: 'timerange',
      label: t('alarm.filter.timeRange'),
      children: (
        <div style={{ padding: '12px 0' }}>
          <Form.Item name="timeRange" label={t('alarm.filter.alarmTime')}>
            <RangePicker showTime style={{ width: '100%' }} placeholder={[t('dateRange.start'), t('dateRange.end')]} />
          </Form.Item>
          <Form.Item name="clearTimeRange" label={t('alarm.filter.clearTime')}>
            <RangePicker showTime style={{ width: '100%' }} placeholder={[t('dateRange.start'), t('dateRange.end')]} />
          </Form.Item>
        </div>
      ),
    },
    {
      key: 'advanced',
      label: t('alarm.filter.advanced'),
      children: (
        <div style={{ padding: '12px 0' }}>
          <Form.Item name="includeCleared" valuePropName="checked" label=" " colon={false}>
            <Checkbox>{t('alarm.filter.includeCleared')}</Checkbox>
          </Form.Item>
          <Form.Item name="includeAcknowledged" valuePropName="checked" label=" " colon={false}>
            <Checkbox>{t('alarm.filter.includeAcknowledged')}</Checkbox>
          </Form.Item>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item name="sortBy" label={t('alarm.filter.sortBy')}>
                <Select
                  placeholder={t('common.pleaseSelect')}
                  allowClear
                  options={[
                    { label: t('alarm.filter.alarmTime'), value: 'eventTime' },
                    { label: t('alarm.filter.severity'), value: 'severity' },
                    { label: t('alarm.filter.alarmIdentifier'), value: 'alarmIdentifier' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="limit" label={t('alarm.filter.maxCount')}>
                <Select
                  placeholder={t('common.pleaseSelect')}
                  allowClear
                  options={[
                    { label: t('alarm.filter.countItems', { count: 100 }), value: 100 },
                    { label: t('alarm.filter.countItems', { count: 500 }), value: 500 },
                    { label: t('alarm.filter.countItems', { count: 1000 }), value: 1000 },
                    { label: t('alarm.filter.noLimit'), value: 0 },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>
        </div>
      ),
    },
  ];

  return (
    <Modal
      title={t('alarm.filter.title')}
      open={visible}
      onCancel={onCancel}
      width={800}
      destroyOnHidden
      footer={
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Button onClick={handleReset}>{t('alarm.filter.resetAll')}</Button>
          <Space>
            <Button onClick={onCancel}>{t('common.cancel')}</Button>
            <Button type="primary" onClick={handleOk}>
              {t('alarm.filter.apply')}
            </Button>
          </Space>
        </div>
      }
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={initialValues}
        size="small"
      >
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={tabItems}
          size="small"
        />
      </Form>
    </Modal>
  );
};

export default AlarmFilterDialog;
