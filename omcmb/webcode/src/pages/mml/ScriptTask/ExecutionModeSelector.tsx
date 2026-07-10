import { Radio, theme } from 'antd';

import type { TranslateFn } from '@/hooks/useT';
import type { MMLTaskExecuteMode } from '@core/types/mml';

interface ExecutionModeSelectorProps {
  value: MMLTaskExecuteMode;
  deviceBoundDetected: boolean;
  onChange: (mode: MMLTaskExecuteMode) => void;
  t: TranslateFn;
}

const OPTIONS: Array<{
  value: MMLTaskExecuteMode;
  labelKey: string;
  descriptionKey: string;
}> = [
  {
    value: 'common',
    labelKey: 'mml.executeModeCommon',
    descriptionKey: 'mml.executeModeCommonDescription',
  },
  {
    value: 'device_bound',
    labelKey: 'mml.executeModeDeviceBound',
    descriptionKey: 'mml.executeModeDeviceBoundDescription',
  },
];

export default function ExecutionModeSelector({
  value,
  deviceBoundDetected,
  onChange,
  t,
}: ExecutionModeSelectorProps) {
  const { token } = theme.useToken();

  return (
    <Radio.Group
      value={value}
      onChange={(event) => onChange(event.target.value as MMLTaskExecuteMode)}
      style={{ width: '100%' }}
    >
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))',
          gap: 12,
        }}
      >
        {OPTIONS.map((option) => {
          const selected = value === option.value;
          const disabled = option.value === 'common' && deviceBoundDetected;
          return (
            <Radio
              key={option.value}
              value={option.value}
              disabled={disabled}
              style={{
                alignItems: 'flex-start',
                minHeight: 82,
                marginInlineEnd: 0,
                padding: '14px 16px',
                border: `1px solid ${selected ? token.colorPrimary : token.colorBorderSecondary}`,
                borderRadius: token.borderRadiusLG,
                background: selected ? token.colorPrimaryBg : token.colorBgContainer,
                boxShadow: selected ? `0 0 0 2px ${token.colorPrimaryBg}` : 'none',
                opacity: disabled ? 0.6 : 1,
              }}
            >
              <span style={{ display: 'block', paddingInlineStart: 4 }}>
                <TypographyLine strong>{t(option.labelKey)}</TypographyLine>
                <TypographyLine secondary>{t(option.descriptionKey)}</TypographyLine>
              </span>
            </Radio>
          );
        })}
      </div>
    </Radio.Group>
  );
}

function TypographyLine({
  children,
  strong = false,
  secondary = false,
}: {
  children: string;
  strong?: boolean;
  secondary?: boolean;
}) {
  const { token } = theme.useToken();
  return (
    <span
      style={{
        display: 'block',
        color: secondary ? token.colorTextSecondary : token.colorText,
        fontSize: secondary ? 12 : 14,
        fontWeight: strong ? 600 : 400,
        lineHeight: secondary ? 1.6 : 1.5,
        marginTop: secondary ? 4 : 0,
      }}
    >
      {children}
    </span>
  );
}
