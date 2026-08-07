import { useState, useMemo, useCallback, useEffect } from 'react';
import { Modal, Input, Button, Space, Tag, Table, App, Typography, Tooltip, Empty } from 'antd';
import type { TableColumnsType } from 'antd';
import {
  SearchOutlined,
  PlusOutlined,
  DeleteOutlined,
  ClearOutlined,
  EditOutlined,
} from '@ant-design/icons';
import { useT } from '@/hooks/useT';
import { provisionApi } from '@core/services/api/provisionApi';

const { Text } = Typography;

interface Device {
  id: string;
  serialNumber: string;
  cellName: string;
  softwareVersion: string;
  product: string;
  groupName: string;
  connectionStatus: 'online' | 'offline';
}

interface Policy {
  policyId: string;
  policyName: string;
  productClass: string;
}

interface Props {
  open: boolean;
  policy: Policy | null;
  onClose: () => void;
  onSuccess: () => void;
}

export default function DetectDialog({ open, policy, onClose, onSuccess }: Props) {
  const t = useT();
  const { message } = App.useApp();

  // State
  const [searchText, setSearchText] = useState('');
  const [selectedDevices, setSelectedDevices] = useState<Device[]>([]);
  const [batchInputOpen, setBatchInputOpen] = useState(false);
  const [batchInputValue, setBatchInputValue] = useState('');
  const [batchInputError, setBatchInputError] = useState('');
  const [loading, setLoading] = useState(false);
  const [devices, setDevices] = useState<Device[]>([]);

  // Reset state when dialog opens
  useEffect(() => {
    if (open) {
      setSearchText('');
      setSelectedDevices([]);
      setBatchInputValue('');
      setBatchInputError('');
      if (policy) {
        setLoading(true);
        void provisionApi.detectDevices(policy.policyId)
          .then((items) => setDevices(items.map((d) => ({
            id: d.id,
            serialNumber: d.serialNumber,
            cellName: d.deviceName,
            softwareVersion: d.firmwareVersion,
            product: d.productName,
            groupName: d.groupName,
            connectionStatus: d.isOnline ? 'online' : 'offline',
          }))))
          .catch(() => message.error(t('common.operationFailed')))
          .finally(() => setLoading(false));
      }
    }
  }, [open, policy, message, t]);

  // Filtered available devices
  const filteredDevices = useMemo(() => {
    if (!searchText) return devices;
    const search = searchText.toLowerCase();
    return devices.filter(d =>
      d.serialNumber.toLowerCase().includes(search) ||
      d.cellName.toLowerCase().includes(search)
    );
  }, [devices, searchText]);

  // Available devices (not selected)
  const availableDevices = useMemo(() => {
    const selectedSns = new Set(selectedDevices.map(d => d.serialNumber));
    return filteredDevices.filter(d => !selectedSns.has(d.serialNumber));
  }, [filteredDevices, selectedDevices]);

  // Add device to selected
  const handleAddDevice = useCallback((device: Device) => {
    setSelectedDevices(prev => [...prev, device]);
  }, []);

  // Remove device from selected
  const handleRemoveDevice = useCallback((serialNumber: string) => {
    setSelectedDevices(prev => prev.filter(d => d.serialNumber !== serialNumber));
  }, []);

  // Clear all selected
  const handleClearSelected = useCallback(() => {
    setSelectedDevices([]);
  }, []);

  const handleSelectAllAvailable = useCallback(() => {
    const remaining = Math.max(0, 50 - selectedDevices.length);
    if (remaining === 0) {
      message.warning(t('provision.batchSelectLimit', { max: 50 }));
      return;
    }
    const additions = availableDevices.slice(0, remaining);
    setSelectedDevices((previous) => [...previous, ...additions]);
    if (additions.length < availableDevices.length) {
      message.warning(t('provision.batchSelectLimit', { max: 50 }));
    }
  }, [availableDevices, message, selectedDevices.length, t]);

  // Batch input validation
  const validateBatchInput = useCallback((value: string): boolean => {
    if (!value.trim()) {
      setBatchInputError(t('provision.serialNumberRequired'));
      return false;
    }

    const sns = value.replace(/[(\r\n)\r\n\s；;]+/g, ';').split(';').filter(s => s.trim());
    const regex = /^(\d|[a-zA-Z]|-|\s){1,30}$/;

    for (const sn of sns) {
      if (!regex.test(sn.trim())) {
        setBatchInputError(t('provision.serialNumberFormat'));
        return false;
      }
    }

    setBatchInputError('');
    return true;
  }, [t]);

  // Handle batch input submit
  const handleBatchInputSubmit = useCallback(() => {
    if (!validateBatchInput(batchInputValue)) return;

    const sns = batchInputValue.replace(/[(\r\n)\r\n\s；;]+/g, ';').split(';').filter(s => s.trim());
    const newDevices: Device[] = [];

    for (const sn of sns) {
      const device = devices.find(d => d.serialNumber.toLowerCase() === sn.trim().toLowerCase());
      if (device && !selectedDevices.some(sd => sd.serialNumber === device.serialNumber)) {
        newDevices.push(device);
      }
    }

    if (newDevices.length > 0) {
      setSelectedDevices(prev => [...prev, ...newDevices]);
      message.success(t('provision.addedDevices', { count: newDevices.length }));
    } else {
      message.warning(t('provision.noNewDevices'));
    }

    setBatchInputOpen(false);
    setBatchInputValue('');
  }, [batchInputValue, devices, selectedDevices, validateBatchInput, message, t]);

  // Handle submit
  const handleSubmit = useCallback(async () => {
    if (selectedDevices.length === 0) {
      message.warning(t('provision.selectDeviceRequired'));
      return;
    }

    if (!policy) return;
    try {
      setLoading(true);
      await provisionApi.executePolicy(policy.policyId, selectedDevices.map((d) => d.id));
      setLoading(false);
      onSuccess();
    } catch (error) {
      setLoading(false);
      message.error(error instanceof Error && error.message
        ? error.message
        : t('common.operationFailed'));
    }
  }, [policy, selectedDevices, onSuccess, message, t]);

  // Available devices columns
  const availableColumns: TableColumnsType<Device> = useMemo(() => [
    {
      title: '',
      key: 'action',
      width: 40,
      render: (_, record) => (
        <Tooltip title={t('common.add')}>
          <Button
            type="text"
            size="small"
            icon={<PlusOutlined />}
            onClick={() => handleAddDevice(record)}
          />
        </Tooltip>
      ),
    },
    {
      title: '',
      key: 'status',
      width: 40,
      render: (_, record) => (
        <Tooltip title={record.connectionStatus === 'online' ? t('provision.online') : t('provision.offline')}>
          <Tag
            color={record.connectionStatus === 'online' ? 'success' : 'default'}
            style={{ width: 8, height: 8, borderRadius: '50%', padding: 0 }}
          />
        </Tooltip>
      ),
    },
    {
      title: t('provision.deviceCode'),
      dataIndex: 'serialNumber',
      width: 120,
    },
    {
      title: t('provision.hostName'),
      dataIndex: 'cellName',
      width: 120,
      ellipsis: true,
    },
    {
      title: t('provision.version'),
      dataIndex: 'softwareVersion',
      width: 80,
    },
    {
      title: t('provision.productName'),
      dataIndex: 'product',
      width: 60,
    },
    {
      title: t('provision.deviceGroup'),
      dataIndex: 'groupName',
      width: 100,
      ellipsis: true,
    },
  ], [t, handleAddDevice]);

  // Selected devices columns
  const selectedColumns: TableColumnsType<Device> = useMemo(() => [
    {
      title: '',
      key: 'action',
      width: 40,
      render: (_, record) => (
        <Tooltip title={t('common.delete')}>
          <Button
            type="text"
            size="small"
            danger
            icon={<DeleteOutlined />}
            onClick={() => handleRemoveDevice(record.serialNumber)}
          />
        </Tooltip>
      ),
    },
    {
      title: t('provision.deviceCode'),
      dataIndex: 'serialNumber',
      width: 120,
    },
    {
      title: t('provision.hostName'),
      dataIndex: 'cellName',
      width: 120,
      ellipsis: true,
    },
  ], [t, handleRemoveDevice]);

  return (
    <>
      <Modal
        title={
          <Space>
            <span>{t('provision.deviceList')}</span>
            <Text type="secondary" style={{ fontSize: 12, fontWeight: 'normal' }}>
              ({t('provision.detectHint')})
            </Text>
          </Space>
        }
        open={open}
        onCancel={onClose}
        width={1100}
        footer={
          <Space>
            <Button onClick={onClose}>{t('common.cancel')}</Button>
            <Button type="primary" loading={loading} onClick={handleSubmit}>
              {t('common.confirm')}
            </Button>
          </Space>
        }
        styles={{ body: { padding: '16px 24px' } }}
      >
        <div style={{ display: 'flex', gap: 16, height: 400 }}>
          {/* Left: Available devices */}
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column', border: '1px solid #f0f0f0', borderRadius: 6, overflow: 'hidden' }}>
            <div style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Text strong>{t('provision.availableDevices')}</Text>
              <Space size={8}>
                <Button
                  size="small"
                  type="primary"
                  onClick={handleSelectAllAvailable}
                  disabled={availableDevices.length === 0}
                >
                  {t('provision.selectAll')}
                </Button>
                <Input
                  placeholder={t('provision.searchDeviceCode')}
                  prefix={<SearchOutlined />}
                  value={searchText}
                  onChange={(e) => setSearchText(e.target.value)}
                  style={{ width: 180 }}
                  size="small"
                  allowClear
                />
                <Button
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => setBatchInputOpen(true)}
                >
                  {t('common.batchImport')}
                </Button>
              </Space>
            </div>
            <div style={{ flex: 1, overflow: 'auto' }}>
              <Table
                columns={availableColumns}
                dataSource={availableDevices}
                rowKey="serialNumber"
                size="small"
                pagination={false}
                showHeader={true}
                locale={{ emptyText: <Empty description={t('common.noData')} image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
              />
            </div>
          </div>

          {/* Right: Selected devices */}
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column', border: '1px solid #f0f0f0', borderRadius: 6, overflow: 'hidden' }}>
            <div style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Space>
                <Text strong>{t('provision.selectedDevices')}</Text>
                <Tag color="blue">{selectedDevices.length}</Tag>
              </Space>
              <Button
                size="small"
                danger
                icon={<ClearOutlined />}
                onClick={handleClearSelected}
                disabled={selectedDevices.length === 0}
              >
                {t('common.clear')}
              </Button>
            </div>
            <div style={{ flex: 1, overflow: 'auto' }}>
              <Table
                columns={selectedColumns}
                dataSource={selectedDevices}
                rowKey="serialNumber"
                size="small"
                pagination={false}
                showHeader={true}
                locale={{ emptyText: <Empty description={t('provision.noSelectedDevices')} image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
              />
            </div>
          </div>
        </div>
      </Modal>

      {/* Batch Input Modal */}
      <Modal
        title={t('common.batchImport')}
        open={batchInputOpen}
        onCancel={() => {
          setBatchInputOpen(false);
          setBatchInputValue('');
          setBatchInputError('');
        }}
        width={630}
        footer={
          <Space>
            <Button onClick={() => {
              setBatchInputOpen(false);
              setBatchInputValue('');
              setBatchInputError('');
            }}>
              {t('common.cancel')}
            </Button>
            <Button type="primary" onClick={handleBatchInputSubmit}>
              {t('common.confirm')}
            </Button>
          </Space>
        }
      >
        <div style={{ marginBottom: 8 }}>
          <Text>{t('provision.serialNumber')}</Text>
        </div>
        <Input.TextArea
          rows={4}
          value={batchInputValue}
          onChange={(e) => {
            setBatchInputValue(e.target.value);
            if (batchInputError) validateBatchInput(e.target.value);
          }}
          placeholder={t('provision.batchInputPlaceholder')}
          status={batchInputError ? 'error' : undefined}
        />
        {batchInputError && (
          <Text type="danger" style={{ fontSize: 12 }}>{batchInputError}</Text>
        )}
        <div style={{ marginTop: 8, color: '#bbb', fontSize: 12 }}>
          <Text type="secondary">{t('provision.batchInputHint')}</Text>
        </div>
      </Modal>
    </>
  );
}
