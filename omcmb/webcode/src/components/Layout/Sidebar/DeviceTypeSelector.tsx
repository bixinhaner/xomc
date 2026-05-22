import { useMemo } from 'react';
import { Select } from 'antd';
import { useAppStore } from '@core/store/appStore';
import type { DeviceType } from '@core/store/appStore';
import { useT } from '@/hooks/useT';

interface Props {
  collapsed: boolean;
}

export default function DeviceTypeSelector({ collapsed }: Props) {
  const deviceType = useAppStore((s) => s.deviceType);
  const setDeviceType = useAppStore((s) => s.setDeviceType);
  const t = useT();

  const deviceOptions = useMemo(() => [
    { value: 'all' as DeviceType, label: t('device.type.all') },
    { value: 'eNB' as DeviceType, label: 'eNB' },
    { value: 'gNB' as DeviceType, label: 'gNB' },
    { value: 'CPE' as DeviceType, label: 'CPE' },
    { value: 'eGW' as DeviceType, label: 'eGW' },
  ], [t]);

  if (collapsed) return null;

  return (
    <div
      style={{
        padding: '10px 12px 6px',
        borderBottom: `1px solid var(--color-sidebar-divider)`,
      }}
    >
      <div
        style={{
          fontSize: 11,
          color: 'var(--color-sidebar-text-muted)',
          marginBottom: 6,
          textTransform: 'uppercase',
          letterSpacing: '0.5px',
        }}
      >
        {t('device.productClass')}
      </div>
      <Select<DeviceType>
        value={deviceType}
        onChange={setDeviceType}
        options={deviceOptions}
        size="small"
        style={{ width: '100%' }}
        styles={{
          popup: {
            root: { zIndex: 1100 },
          },
        }}
      />
    </div>
  );
}
