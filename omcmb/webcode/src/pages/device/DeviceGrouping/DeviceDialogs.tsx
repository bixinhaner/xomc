import React, { useCallback, useMemo, useState } from 'react';
import { Alert, Button, Drawer, Form, Input, InputNumber, Modal, Progress, Radio, Select, Typography, Upload } from 'antd';
import { CheckCircleOutlined, DownloadOutlined, InboxOutlined } from '@ant-design/icons';
import type { FormInstance, UploadFile, UploadProps } from 'antd';
import type { Device, EngStatus } from '@/types/device';

const { Dragger } = Upload;
const { Text } = Typography;

interface TargetGroupOption {
  label: string;
  value: string;
}

export interface DeviceDialogsProps {
  // Move to Group Modal
  moveToGroupModalOpen: boolean;
  selectedDeviceIds: React.Key[];
  targetGroupId: string | null;
  targetGroupOptions: TargetGroupOption[];
  onTargetGroupChange: (id: string | null) => void;
  onMoveToGroupOk: () => void;
  onMoveToGroupCancel: () => void;

  // Edit Device Modal
  editDeviceModalOpen: boolean;
  editingDevice: Device | null;
  editDeviceForm: FormInstance<{
    engStatus: EngStatus;
    longitude: number;
    latitude: number;
    gpsHeight: number;
    remark: string;
  }>;
  onEditDeviceOk: () => void;
  onEditDeviceCancel: () => void;

  // Add Device to Group Drawer
  addDeviceDrawerOpen: boolean;
  addDeviceForm: FormInstance<{
    addMethod: 'manual' | 'import';
    deviceSnList: string;
  }>;
  addMethod: string | undefined;
  onAddDeviceDrawerClose: () => void;
  onSaveDevices: () => void;

  // Batch Import Modal
  batchImportModalOpen: boolean;
  onBatchImportModalClose: () => void;
  onBatchImportConfirm: () => void;
  onDownloadTemplate: () => void;

  t: (id: string, values?: Record<string, unknown>) => string;
}

export default function DeviceDialogs({
  moveToGroupModalOpen,
  selectedDeviceIds,
  targetGroupId,
  targetGroupOptions,
  onTargetGroupChange,
  onMoveToGroupOk,
  onMoveToGroupCancel,
  editDeviceModalOpen,
  editingDevice,
  editDeviceForm,
  onEditDeviceOk,
  onEditDeviceCancel,
  addDeviceDrawerOpen,
  addDeviceForm,
  addMethod,
  onAddDeviceDrawerClose,
  onSaveDevices,
  batchImportModalOpen,
  onBatchImportModalClose,
  onBatchImportConfirm,
  onDownloadTemplate,
  t,
}: DeviceDialogsProps) {
  // Batch import state
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [importing, setImporting] = useState(false);
  const [importProgress, setImportProgress] = useState(0);
  const [importDone, setImportDone] = useState(false);

  const ENG_STATUS_OPTIONS = useMemo(() => [
    { label: t('device.engStatus.commissioned'), value: 'commissioned' },
    { label: t('device.engStatus.uncommissioned'), value: 'uncommissioned' },
    { label: t('device.engStatus.decommissioned'), value: 'decommissioned' },
  ], [t]);

  // Reset batch import state
  const resetBatchImportState = useCallback(() => {
    setFileList([]);
    setImporting(false);
    setImportProgress(0);
    setImportDone(false);
  }, []);

  // Handle batch import modal close
  const handleBatchImportClose = useCallback(() => {
    resetBatchImportState();
    onBatchImportModalClose();
  }, [resetBatchImportState, onBatchImportModalClose]);

  // Upload props for batch import
  const uploadProps: UploadProps = useMemo(() => ({
    name: 'file',
    multiple: false,
    accept: '.csv',
    fileList,
    beforeUpload: (file) => {
      const isCsv = file.name.toLowerCase().endsWith('.csv') ||
        file.type === 'text/csv' ||
        file.type === 'application/vnd.ms-excel';
      if (!isCsv) {
        return false;
      }
      const isLt10M = file.size / 1024 / 1024 < 10;
      if (!isLt10M) {
        return false;
      }
      setFileList([file]);
      setImportDone(false);
      return false;
    },
    onRemove: () => {
      setFileList([]);
      setImportDone(false);
    },
  }), [fileList]);

  // Execute batch import
  const handleBatchImport = useCallback(async () => {
    if (fileList.length === 0) {
      return;
    }

    setImporting(true);
    setImportProgress(0);

    // Simulate import progress
    for (let p = 0; p <= 100; p += 10) {
      await new Promise<void>((resolve) => setTimeout(resolve, 100));
      setImportProgress(p);
    }

    setImporting(false);
    setImportDone(true);
  }, [fileList]);

  // Handle batch import modal OK
  const handleBatchImportOk = useCallback(() => {
    if (importDone) {
      onBatchImportConfirm();
      handleBatchImportClose();
    } else if (fileList.length > 0) {
      void handleBatchImport();
    }
  }, [importDone, onBatchImportConfirm, handleBatchImportClose, handleBatchImport, fileList.length]);

  return (
    <>
      {/* Move to Group Modal */}
      <Modal
        title={t('device.batch.moveToGroup')}
        open={moveToGroupModalOpen}
        onOk={onMoveToGroupOk}
        onCancel={onMoveToGroupCancel}
        okText={t('common.confirm')}
      >
        <div style={{ marginTop: 16 }}>
          <Text type="secondary">
            {t('device.batch.selectedDevices', { count: selectedDeviceIds.length })}
          </Text>
          <Form.Item label={t('device.batch.targetGroup')} style={{ marginTop: 16 }}>
            <Select
              style={{ width: '100%' }}
              placeholder={t('device.batch.selectGroupPlaceholder')}
              value={targetGroupId}
              onChange={onTargetGroupChange}
              options={targetGroupOptions}
              showSearch
              optionFilterProp="label"
            />
          </Form.Item>
        </div>
      </Modal>

      {/* Edit Device Modal */}
      <Modal
        title={`${t('common.edit')} - ${editingDevice?.sn ?? ''}`}
        open={editDeviceModalOpen}
        onOk={onEditDeviceOk}
        onCancel={onEditDeviceCancel}
        okText={t('common.save')}
      >
        <Form form={editDeviceForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="engStatus"
            label={t('device.installStatus')}
            rules={[{ required: true, message: t('common.pleaseSelect') }]}
          >
            <Select options={ENG_STATUS_OPTIONS} />
          </Form.Item>
          <Form.Item
            name="longitude"
            label={t('device.longitude')}
            rules={[{ required: true, message: t('common.placeholder') }]}
          >
            <InputNumber style={{ width: '100%' }} precision={6} />
          </Form.Item>
          <Form.Item
            name="latitude"
            label={t('device.latitude')}
            rules={[{ required: true, message: t('common.placeholder') }]}
          >
            <InputNumber style={{ width: '100%' }} precision={6} />
          </Form.Item>
          <Form.Item
            name="gpsHeight"
            label={t('device.height')}
            rules={[{ required: true, message: t('common.placeholder') }]}
          >
            <InputNumber style={{ width: '100%' }} precision={1} />
          </Form.Item>
          <Form.Item
            name="remark"
            label={t('device.remark')}
          >
            <Input.TextArea rows={2} maxLength={200} showCount />
          </Form.Item>
        </Form>
      </Modal>

      {/* Add Device to Group Drawer (Manual only) */}
      <Drawer
        title={t('device.addDeviceToGroup')}
        open={addDeviceDrawerOpen}
        onClose={onAddDeviceDrawerClose}
        width={520}
        destroyOnClose
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button onClick={onAddDeviceDrawerClose}>{t('common.cancel')}</Button>
            <Button type="primary" onClick={onSaveDevices}>
              {t('common.confirm')}
            </Button>
          </div>
        }
      >
        <Form form={addDeviceForm} layout="vertical">
          {/* Add method */}
          <div style={{
            padding: '16px',
            background: 'var(--color-fill-quaternary)',
            borderRadius: 8,
            marginBottom: 16
          }}>
            <div style={{ marginBottom: 4, fontWeight: 500, color: 'var(--color-text)' }}>
              {t('device.addMethod')}
            </div>
            <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 12 }}>
              {t('device.addMethodDesc')}
            </Text>
            <Form.Item
              name="addMethod"
              rules={[{ required: true }]}
              style={{ marginBottom: 0 }}
            >
              <Radio.Group>
                <Radio value="manual">{t('device.manualAdd')}</Radio>
                <Radio value="import">{t('device.batchImport')}</Radio>
              </Radio.Group>
            </Form.Item>
          </div>

          {/* Manual add: SN list */}
          {addMethod === 'manual' && (
            <div style={{
              padding: '16px',
              background: 'var(--color-fill-quaternary)',
              borderRadius: 8
            }}>
              <div style={{ marginBottom: 4, fontWeight: 500, color: 'var(--color-text)' }}>
                {t('device.deviceSnList')}
              </div>
              <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 12 }}>
                {t('device.snListFormat')}
              </Text>
              <Form.Item
                name="deviceSnList"
                rules={[{ required: true, message: t('device.snListRequired') }]}
                style={{ marginBottom: 0 }}
              >
                <Input.TextArea
                  rows={8}
                  placeholder={t('device.snListPlaceholder')}
                  maxLength={5000}
                  showCount
                />
              </Form.Item>
            </div>
          )}
        </Form>
      </Drawer>

      {/* Batch Import Modal */}
      <Modal
        title={t('device.batchImport')}
        open={batchImportModalOpen}
        onOk={handleBatchImportOk}
        onCancel={handleBatchImportClose}
        okText={importDone ? t('common.finish') : t('common.import')}
        cancelText={importDone ? t('common.close') : t('common.cancel')}
        confirmLoading={importing}
        width={520}
        destroyOnClose
      >
        {/* Download template button */}
        <div style={{ marginBottom: 12 }}>
          <Button
            icon={<DownloadOutlined />}
            size="small"
            onClick={onDownloadTemplate}
          >
            {t('device.downloadTemplate')}
          </Button>
        </div>

        <Dragger {...uploadProps} disabled={importing || importDone}>
          <p className="ant-upload-drag-icon">
            <InboxOutlined style={{ fontSize: 40, color: 'var(--color-primary-600)' }} />
          </p>
          <p className="ant-upload-text">{t('device.clickOrDragUpload')}</p>
          <p className="ant-upload-hint" style={{ fontSize: 12, color: '#8c8c8c' }}>
            CSV
          </p>
        </Dragger>

        {importing && (
          <div style={{ marginTop: 16 }}>
            <span style={{ fontSize: 13, color: '#666' }}>{t('common.loading')}</span>
            <Progress percent={importProgress} status="active" />
          </div>
        )}

        {importDone && (
          <Alert
            type="success"
            showIcon
            icon={<CheckCircleOutlined />}
            message={t('device.importSuccess')}
            style={{ marginTop: 16 }}
          />
        )}
      </Modal>
    </>
  );
}
