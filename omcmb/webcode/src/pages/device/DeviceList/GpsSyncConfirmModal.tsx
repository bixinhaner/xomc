import { Alert, Button, Modal, Space, Typography, theme } from 'antd';
import { useT } from '@/hooks/useT';
import type { Device } from '@core/types/device';

interface GpsSyncConfirmModalProps {
  open: boolean;
  device: Pick<Device, 'longitude' | 'latitude' | 'gpsHeight' | 'locationSync'> | null;
  onCancel: () => void;
  onConfirm: () => void;
}

export default function GpsSyncConfirmModal({
  open,
  device,
  onCancel,
  onConfirm,
}: GpsSyncConfirmModalProps) {
  const t = useT();
  const { token } = theme.useToken();
  const canSynchronize = device?.locationSync.status === 'pending'
    && device.locationSync.reported != null;
  const isSynchronized = device?.locationSync.status === 'in_sync';

  return (
    <Modal
      open={open}
      centered
      title={t(canSynchronize ? 'common.operationConfirm' : 'device.gpsCoordinateDetailsTitle')}
      onCancel={onCancel}
      onOk={canSynchronize ? onConfirm : undefined}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
      footer={canSynchronize ? undefined : (
        <Button type="primary" onClick={onCancel}>
          {t('common.close')}
        </Button>
      )}
      destroyOnHidden
    >
      <Space orientation="vertical" size={16} style={{ width: '100%' }}>
        {canSynchronize && (
          <Typography.Text>{t('device.gpsSyncConfirmTitle')}</Typography.Text>
        )}
        <div
          style={{
            padding: `${token.paddingSM}px ${token.padding}px`,
            border: `1px solid ${token.colorBorderSecondary}`,
            borderRadius: token.borderRadiusLG,
            background: token.colorFillAlter,
          }}
        >
          <Typography.Text>
            {t('device.gpsSyncCoordinates', {
              longitude: device?.longitude ?? '--',
              latitude: device?.latitude ?? '--',
              gpsHeight: device?.gpsHeight ?? '--',
            })}
          </Typography.Text>
        </div>
        <Alert
          type={canSynchronize ? 'warning' : isSynchronized ? 'success' : 'info'}
          showIcon
          title={t(
            canSynchronize
              ? 'device.gpsInconsistent'
              : isSynchronized
                ? 'device.gpsInSync'
                : 'device.gpsSyncNoReport',
          )}
        />
      </Space>
    </Modal>
  );
}
