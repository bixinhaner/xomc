import { Alert, Button, Modal, Space, Typography, theme } from 'antd';
import { useT } from '@/hooks/useT';
import type { Device } from '@core/types/device';
import { formatSystemTime } from '@core/utils/systemTime';
import { parseGpsObservationSource } from './deviceGpsObservation';

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
  const accepted = device?.locationSync.accepted;
  const reported = device?.locationSync.reported;
  const acceptedLongitude = accepted?.longitude ?? device?.longitude ?? '--';
  const acceptedLatitude = accepted?.latitude ?? device?.latitude ?? '--';
  const reportedLongitude = reported?.longitude ?? '--';
  const reportedLatitude = reported?.latitude ?? '--';
  // Older observations may not carry height; the list's latest synchronized
  // GPS height is the compatible fallback until the next parameter sync.
  const reportedGPSHeight = reported?.gpsHeight ?? device?.gpsHeight ?? '--';
  const observationSource = reported
    ? parseGpsObservationSource(reported.sourcePath)
    : null;
  const observedAt = reported?.observedAt
    ? formatSystemTime(reported.observedAt)
    : '--';

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
          <Space orientation="vertical" size={4}>
            <Typography.Text>
              {t('device.gpsAcceptedCoordinates', {
                longitude: acceptedLongitude,
                latitude: acceptedLatitude,
              })}
            </Typography.Text>
            <Typography.Text>
              {t('device.gpsReportedCoordinates', {
                longitude: reportedLongitude,
                latitude: reportedLatitude,
                gpsHeight: reportedGPSHeight,
              })}
            </Typography.Text>
            {device?.locationSync.distanceMeters != null && (
              <Typography.Text>
                {t('device.gpsHorizontalDifference', {
                  distance: Math.round(device.locationSync.distanceMeters),
                })}
              </Typography.Text>
            )}
            {reported && observationSource && (
              <>
                <Typography.Text type="secondary">
                  {t('device.gpsObservedAt', { observedAt })}
                </Typography.Text>
                <Typography.Text type="secondary">
                  {t('device.gpsCoordinateSlot', { slot: observationSource.slot })}
                </Typography.Text>
              </>
            )}
          </Space>
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
