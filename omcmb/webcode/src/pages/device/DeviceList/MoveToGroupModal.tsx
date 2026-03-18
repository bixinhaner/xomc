import React, { useCallback, useState } from 'react';
import { Modal, Radio, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useT } from '@/hooks/useT';

interface DeviceGroup {
  id: string;
  groupName: string;
}

interface MoveToGroupModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (groupId: string) => void;
  confirmLoading?: boolean;
}

// TODO: 接入真实 API 后替换为 useDeviceGroupList hook
const MOCK_GROUPS: DeviceGroup[] = [
  { id: '1', groupName: '默认设备组' },
  { id: '2', groupName: '测试设备组' },
  { id: '3', groupName: '生产设备组A' },
  { id: '4', groupName: '生产设备组B' },
];

export default function MoveToGroupModal({ open, onClose, onConfirm, confirmLoading }: MoveToGroupModalProps) {
  const t = useT();
  const [selectedGroupId, setSelectedGroupId] = useState<string>('');
  const [showTip, setShowTip] = useState(false);

  const columns: ColumnsType<DeviceGroup> = [
    {
      title: '',
      width: 50,
      render: (_, record) => (
        <Radio
          checked={selectedGroupId === record.id}
          onChange={() => {
            setSelectedGroupId(record.id);
            setShowTip(false);
          }}
        />
      ),
    },
    {
      title: t('device.groupName'),
      dataIndex: 'groupName',
      key: 'groupName',
    },
  ];

  const handleOk = useCallback(() => {
    if (!selectedGroupId) {
      setShowTip(true);
      return;
    }
    onConfirm(selectedGroupId);
  }, [selectedGroupId, onConfirm]);

  const handleCancel = useCallback(() => {
    setSelectedGroupId('');
    setShowTip(false);
    onClose();
  }, [onClose]);

  return (
    <Modal
      title={t('common.moveToGroup')}
      open={open}
      onOk={handleOk}
      onCancel={handleCancel}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
      confirmLoading={confirmLoading}
      width={620}
      destroyOnClose
    >
      <Table<DeviceGroup>
        dataSource={MOCK_GROUPS}
        columns={columns}
        rowKey="id"
        size="small"
        pagination={{ pageSize: 10, size: 'small' }}
        onRow={(record) => ({
          onClick: () => {
            setSelectedGroupId(record.id);
            setShowTip(false);
          },
          style: { cursor: 'pointer' },
        })}
      />
      {showTip && (
        <div style={{ color: '#ff4d4f', fontSize: 12, marginTop: 4 }}>
          {t('sync.selectGroupTip')}
        </div>
      )}
    </Modal>
  );
}
