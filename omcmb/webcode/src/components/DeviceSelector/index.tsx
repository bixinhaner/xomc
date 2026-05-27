import React, { useMemo, useState } from 'react';
import { Button, Modal, Space, Tabs, Tag, Typography } from 'antd';
import type { Device } from '@core/types/device';
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
        label: '按网元',
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
        label: '按分类',
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
        label: '按模板',
        children: (
          <ByTemplateTab
            selectedSns={selectedSns}
            onSelectionChange={setSelectedSns}
          />
        ),
      },
    ],
    [devices, selectedSns, max]
  );

  return (
    <Modal
      title="选择设备"
      open={visible}
      onOk={handleOk}
      onCancel={handleCancel}
      width={960}
      okText="确认"
      cancelText="取消"
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
            已选 {selectedCount}
          </Tag>
          <Typography.Text type="secondary" style={{ fontSize: 13 }}>
            未选 {unselectedCount}
          </Typography.Text>
          {max && (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              (最多可选 {max} 台)
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
            清空已选
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
