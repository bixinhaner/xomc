import { useCallback, useMemo } from 'react';
import { Button, Dropdown } from 'antd';
import type { ButtonProps, MenuProps } from 'antd';
import { CheckOutlined, MinusCircleOutlined, SyncOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';

export interface AutoRefreshDropdownProps {
  enabled: boolean;
  intervalSeconds: number;
  onEnabledChange: (enabled: boolean) => void;
  onIntervalChange: (seconds: number) => void;
  size?: ButtonProps['size'];
}

export default function AutoRefreshDropdown({
  enabled,
  intervalSeconds,
  onEnabledChange,
  onIntervalChange,
  size,
}: AutoRefreshDropdownProps) {
  const t = useT();

  const intervalOptions = useMemo(
    () => [
      { label: t('common.15seconds'), value: 15 },
      { label: t('common.30seconds'), value: 30 },
      { label: t('common.1minute'), value: 60 },
      { label: t('common.5minutes'), value: 300 },
    ],
    [t]
  );

  const currentLabel = useMemo(
    () => intervalOptions.find((option) => option.value === intervalSeconds)?.label ?? `${intervalSeconds}秒`,
    [intervalOptions, intervalSeconds]
  );

  const menuItems = useMemo<MenuProps['items']>(
    () => [
      ...intervalOptions.map((option) => ({
        key: `interval-${option.value}`,
        label: option.label,
        icon: enabled && intervalSeconds === option.value ? <CheckOutlined /> : undefined,
      })),
      ...(enabled
        ? [{ key: 'disable', label: t('alarm.autoRefreshOff'), icon: <MinusCircleOutlined /> }]
        : []),
    ],
    [enabled, intervalOptions, intervalSeconds, t]
  );

  const handleMenuClick = useCallback<NonNullable<MenuProps['onClick']>>(
    ({ key }) => {
      if (key === 'disable') {
        onEnabledChange(false);
        return;
      }

      if (key.startsWith('interval-')) {
        onIntervalChange(parseInt(key.replace('interval-', ''), 10));
        onEnabledChange(true);
      }
    },
    [onEnabledChange, onIntervalChange]
  );

  return (
    <Dropdown menu={{ items: menuItems, onClick: handleMenuClick }}>
      <Button icon={<SyncOutlined spin={enabled} />} size={size} type={enabled ? 'primary' : 'default'}>
        {enabled ? currentLabel : t('alarm.stats.autoRefresh')}
      </Button>
    </Dropdown>
  );
}