import { Checkbox, Space, Typography } from 'antd';
import { useT } from '@/hooks/useT';
import { getOrderedSelectedPathKeys } from '../pathSelection';
import type { CommandParamPath } from '../types';

const { Text } = Typography;

interface CommandPathSelectorProps {
  paths: CommandParamPath[];
  value: string[];
  onChange: (selectedPathKeys: string[]) => void;
}

export default function CommandPathSelector({ paths, value, onChange }: CommandPathSelectorProps) {
  const t = useT();
  const selected = getOrderedSelectedPathKeys(paths, value);
  const allKeys = paths.map((path) => path.path);
  const allSelected = paths.length > 0 && selected.length === paths.length;

  return (
    <Space orientation="vertical" size={8} style={{ width: '100%' }}>
      <Space size={12}>
        <Checkbox
          checked={allSelected}
          indeterminate={selected.length > 0 && !allSelected}
          onChange={(event) => onChange(event.target.checked ? allKeys : [])}
        >
          {t('mml.consoleV2.cmdSelect.selectAll')}
        </Checkbox>
        <Text type="secondary">
          {t('mml.consoleV2.cmdSelect.selectedCount', {
            selected: selected.length,
            total: paths.length,
          })}
        </Text>
      </Space>

      <Checkbox.Group
        style={{ display: 'flex', flexDirection: 'column', gap: 6, width: '100%' }}
        value={selected}
        onChange={(keys) => onChange(getOrderedSelectedPathKeys(paths, keys as string[]))}
      >
        {paths.map((path) => (
          <Checkbox key={path.path} value={path.path} style={{ whiteSpace: 'nowrap' }}>
            <Text>{path.label}</Text>{' '}
            <Text type="secondary" code style={{ fontSize: 11 }}>
              {path.path}
            </Text>
          </Checkbox>
        ))}
      </Checkbox.Group>
    </Space>
  );
}
