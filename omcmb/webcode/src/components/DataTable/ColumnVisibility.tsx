import React, { useEffect, useState } from 'react';
import { Button, Checkbox, Popover, Tooltip } from 'antd';
import { SettingOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';

interface Column {
  key: string;
  title: string;
  hidden?: boolean;
}

interface ColumnVisibilityProps {
  tableId: string;
  columns: Column[];
  onChange: (hiddenKeys: string[]) => void;
}

const STORAGE_PREFIX = 'omc_col_vis_';

const ColumnVisibility: React.FC<ColumnVisibilityProps> = ({
  tableId,
  columns,
  onChange,
}) => {
  const t = useT();
  const storageKey = `${STORAGE_PREFIX}${tableId}`;

  const getInitialHidden = (): string[] => {
    try {
      const stored = localStorage.getItem(storageKey);
      if (stored) return JSON.parse(stored) as string[];
    } catch {
      // ignore
    }
    return columns.filter((c) => c.hidden).map((c) => c.key);
  };

  const [hiddenKeys, setHiddenKeys] = useState<string[]>(getInitialHidden);

  useEffect(() => {
    onChange(hiddenKeys);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleChange = (key: string, checked: boolean) => {
    const next = checked
      ? hiddenKeys.filter((k) => k !== key)
      : [...hiddenKeys, key];
    setHiddenKeys(next);
    onChange(next);
    try {
      localStorage.setItem(storageKey, JSON.stringify(next));
    } catch {
      // ignore
    }
  };

  const handleShowAll = () => {
    setHiddenKeys([]);
    onChange([]);
    try {
      localStorage.removeItem(storageKey);
    } catch {
      // ignore
    }
  };

  const content = (
    <div style={{ minWidth: 160 }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 8,
          borderBottom: '1px solid #f0f0f0',
          paddingBottom: 8,
        }}
      >
        <span style={{ fontWeight: 500, fontSize: 13 }}>{t('table.columnDisplay')}</span>
        <Button type="link" size="small" onClick={handleShowAll} style={{ padding: 0 }}>
          {t('table.showAll')}
        </Button>
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
        {columns.map((col) => (
          <Checkbox
            key={col.key}
            checked={!hiddenKeys.includes(col.key)}
            onChange={(e) => handleChange(col.key, e.target.checked)}
          >
            <span style={{ fontSize: 13 }}>{col.title}</span>
          </Checkbox>
        ))}
      </div>
    </div>
  );

  return (
    <Tooltip title={t('table.columnSettings')}>
      <Popover
        content={content}
        trigger="click"
        placement="bottomRight"
        overlayStyle={{ zIndex: 1050 }}
      >
        <Button icon={<SettingOutlined />} size="small" />
      </Popover>
    </Tooltip>
  );
};

export default ColumnVisibility;
