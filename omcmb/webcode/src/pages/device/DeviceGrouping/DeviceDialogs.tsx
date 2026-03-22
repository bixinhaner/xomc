import React, { useMemo } from 'react';
import { Button, Drawer, Form, Input, InputNumber, Modal, Radio, Select, Typography, Upload } from 'antd';
import { DownloadOutlined, UploadOutlined } from '@ant-design/icons';
import type { FormInstance, UploadFile } from 'antd';
import type { Device, EngStatus } from '@/types/device';

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
  fileList: UploadFile[];
  onFileListChange: (fileList: UploadFile[]) => void;
  onAddDeviceDrawerClose: () => void;
  onSaveDevices: () => void;
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
  fileList,
  onFileListChange,
  onAddDeviceDrawerClose,
  onSaveDevices,
  onDownloadTemplate,
  t,
}: DeviceDialogsProps) {
  const ENG_STATUS_OPTIONS = useMemo(() => [
    { label: t('device.engStatus.commissioned'), value: 'commissioned' },
    { label: t('device.engStatus.uncommissioned'), value: 'uncommissioned' },
    { label: t('device.engStatus.decommissioned'), value: 'decommissioned' },
  ], [t]);

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
        </Form>
      </Modal>

      {/* Add Device to Group Drawer */}
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

          {/* Batch import: file upload */}
          {addMethod === 'import' && (
            <div style={{
              padding: '16px',
              background: 'var(--color-fill-quaternary)',
              borderRadius: 8
            }}>
              <div style={{ marginBottom: 4, fontWeight: 500, color: 'var(--color-text)' }}>
                {t('device.importFile')}
              </div>
              <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 12 }}>
                {t('device.importFileFormat')}
              </Text>
              <div style={{ marginBottom: 12 }}>
                <Button
                  icon={<DownloadOutlined />}
                  size="small"
                  onClick={onDownloadTemplate}
                >
                  {t('device.downloadTemplate')}
                </Button>
              </div>
              <Form.Item style={{ marginBottom: 0 }}>
                <Upload
                  accept=".csv,.xlsx,.xls"
                  maxCount={1}
                  fileList={fileList}
                  beforeUpload={() => false}
                  onChange={(info) => {
                    onFileListChange(info.fileList.slice(-1));
                  }}
                >
                  <Button icon={<UploadOutlined />}>{t('device.selectFile')}</Button>
                </Upload>
              </Form.Item>
            </div>
          )}
        </Form>
      </Drawer>
    </>
  );
}
