import React, { useMemo, useState } from 'react';
import { Button, Modal, Space, Tabs, Tag, Typography } from 'antd';
import type { Device } from '@core/types/device';
import { useT } from '@/hooks/useT';
import ByElementTab from './ByElementTab';
import ByClassificationTab from './ByClassificationTab';
import ByTemplateTab from './ByTemplateTab';

export interface DeviceSelectorProps {
  visible: boolean;
  onOk: (selectedSns: string[]) => void;
  onCancel: () => void;
  max?: number;
  defaultSelected?: string[];
  devices?: Device[];
}

// Mock device data for development (in production, pass via devices prop)
const MOCK_DEVICES: Device[] = [];

const DeviceSelector: React.FC<DeviceSelectorProps> = ({
  visible,
  onOk,
  onCancel,
  max,
  defaultSelected = [],
  devices = MOCK_DEVICES,
}) => {
  const t = useT();
  const [selectedSns, setSelectedSns] = useState<string[]>(defaultSelected);
  const [activeTab, setActiveTab] = useState('by-element');

  const selectedCount = selectedSns.length;
  const unselectedCount = devices.length - selectedSns.filter((sn) =>
    devices.some((d) => d.sn === sn)
  ).length;

  const handleOk = () => {
    onOk(selectedSns);
  };

  const handleCancel = () => {
    setSelectedSns(defaultSelected);
    onCancel();
  };

  const tabItems = useMemo(
    () => [
      {
        key: 'by-element',
        label: t('deviceSelector.tabByElement'),
        children: (
          <ByElementTab
            devices={devices}
            selectedSns={selectedSns}
            onSelectionChange={setSelectedSns}
            max={max}
          />
        ),
      },
      {
        key: 'by-classification',
        label: t('deviceSelector.tabByClass'),
        children: (
          <ByClassificationTab
            devices={devices}
            selectedSns={selectedSns}
            onSelectionChange={setSelectedSns}
          />
        ),
      },
      {
        key: 'by-template',
        label: t('deviceSelector.tabByTemplate'),
        children: (
          <ByTemplateTab
            selectedSns={selectedSns}
            onSelectionChange={setSelectedSns}
          />
        ),
      },
    ],
    [devices, selectedSns, max, t]
  );

  return (
    <Modal
      title={t('deviceSelector.title')}
      open={visible}
      onOk={handleOk}
      onCancel={handleCancel}
      width={960}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
      destroyOnHidden
      styles={{ body: { padding: '12px 24px' } }}
    >
      {/* Header counts */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 16,
          marginBottom: 12,
          padding: '8px 0',
          borderBottom: '1px solid #f0f0f0',
        }}
      >
        <Space>
          <Tag color="blue" style={{ fontSize: 13, padding: '2px 8px' }}>
            {t('deviceSelector.selected', { count: selectedCount })}
          </Tag>
          <Typography.Text type="secondary" style={{ fontSize: 13 }}>
            {t('deviceSelector.unselected', { count: unselectedCount })}
          </Typography.Text>
          {max && (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {t('deviceSelector.maxHint', { max })}
            </Typography.Text>
          )}
        </Space>
        {selectedCount > 0 && (
          <Button
            type="link"
            size="small"
            danger
            onClick={() => setSelectedSns([])}
            style={{ marginLeft: 'auto' }}
          >
            {t('deviceSelector.clearSelected')}
          </Button>
        )}
      </div>

      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={tabItems}
        size="small"
      />
    </Modal>
  );
};

export default DeviceSelector;
