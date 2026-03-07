import React, { useMemo } from 'react';
import { Button, Dropdown, Tooltip } from 'antd';
import type { MenuProps } from 'antd';
import { ColumnHeightOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';

type Density = 'compact' | 'default' | 'comfortable';

interface DensityToggleProps {
  density: Density;
  onChange: (density: Density) => void;
}

const DensityToggle: React.FC<DensityToggleProps> = ({ density, onChange }) => {
  const t = useT();
  const token = useThemeToken();

  const densityLabels = useMemo<Record<Density, string>>(
    () => ({
      compact: t('table.density.compact'),
      default: t('table.density.default'),
      comfortable: t('table.density.comfortable'),
    }),
    [t]
  );

  const items: MenuProps['items'] = (
    ['compact', 'default', 'comfortable'] as Density[]
  ).map((d) => ({
    key: d,
    label: densityLabels[d],
    onClick: () => onChange(d),
    style: d === density ? { color: token.colorPrimary, fontWeight: 500 } : undefined,
  }));

  return (
    <Tooltip title={t('table.density')}>
      <Dropdown menu={{ items }} placement="bottomRight" trigger={['click']}>
        <Button icon={<ColumnHeightOutlined />} size="small" />
      </Dropdown>
    </Tooltip>
  );
};

export default DensityToggle;
