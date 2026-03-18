import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Checkbox, Popover, Tooltip } from 'antd';
import { SettingOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';

interface Column {
  key: string;
  title: string;
  hidden?: boolean;
  group?: string;
}

interface ColumnVisibilityProps {
  tableId: string;
  columns: Column[];
  onChange: (hiddenKeys: string[]) => void;
  onOrderChange?: (orderedKeys: string[]) => void;
}

const VIS_STORAGE_PREFIX = 'omc_col_vis_';

// ─── Main component ─────────────────────────────────────────────────────────

const ColumnVisibility: React.FC<ColumnVisibilityProps> = ({
  tableId,
  columns,
  onChange,
  onOrderChange,
}) => {
  const t = useT();
  const visKey = `${VIS_STORAGE_PREFIX}${tableId}`;
  const isZh = t('common.yes') === '是';

  // ─── hidden keys ────────────────────────────────────────────────────────

  const getInitialHidden = (): string[] => {
    try {
      const stored = localStorage.getItem(visKey);
      if (stored) return JSON.parse(stored) as string[];
    } catch { /* ignore */ }
    return columns.filter((c) => c.hidden).map((c) => c.key);
  };

  const [hiddenKeys, setHiddenKeys] = useState<string[]>(getInitialHidden);
  const [open, setOpen] = useState(false);

  // ─── init ───────────────────────────────────────────────────────────────

  useEffect(() => {
    onChange(hiddenKeys);
    onOrderChange?.(columns.map((c) => c.key));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // ─── handlers ──────────────────────────────────────────────────────────

  const persistHidden = useCallback(
    (next: string[]) => {
      setHiddenKeys(next);
      onChange(next);
      try { localStorage.setItem(visKey, JSON.stringify(next)); } catch { /* ignore */ }
    },
    [onChange, visKey]
  );

  const handleToggle = useCallback(
    (key: string, checked: boolean) => {
      const next = checked ? hiddenKeys.filter((k) => k !== key) : [...hiddenKeys, key];
      persistHidden(next);
    },
    [hiddenKeys, persistHidden]
  );

  const handleReset = useCallback(() => {
    const defaultHidden = columns.filter((c) => c.hidden).map((c) => c.key);
    persistHidden(defaultHidden);
  }, [columns, persistHidden]);

  // ─── filterable columns (exclude fixed columns like actions) ───────────

  const settableColumns = useMemo(
    () => columns.filter((c) => c.key !== 'actions'),
    [columns]
  );

  const totalVisible = settableColumns.filter((c) => !hiddenKeys.includes(c.key)).length;

  // ─── popover content ──────────────────────────────────────────────────

  const content = (
    <div style={{ width: 200 }}>
      <div style={{
        display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        marginBottom: 8, paddingBottom: 8, borderBottom: '1px solid #f0f0f0',
      }}>
        <span style={{ fontSize: 13, color: '#8c8c8c' }}>
          {totalVisible}/{settableColumns.length}
        </span>
        <Button type="link" size="small" onClick={handleReset} style={{ padding: 0 }}>
          {isZh ? '重置' : 'Reset'}
        </Button>
      </div>
      <div style={{ maxHeight: 400, overflowY: 'auto' }}>
        {settableColumns.map((col) => (
          <div key={col.key} style={{ padding: '4px 0' }}>
            <Checkbox
              checked={!hiddenKeys.includes(col.key)}
              onChange={(e) => handleToggle(col.key, e.target.checked)}
            >
              <span style={{ fontSize: 13 }}>{col.title}</span>
            </Checkbox>
          </div>
        ))}
      </div>
    </div>
  );

  // ─── render ────────────────────────────────────────────────────────────

  return (
    <Popover
      content={content}
      trigger="click"
      placement="bottomRight"
      open={open}
      onOpenChange={setOpen}
    >
      <Tooltip title={t('table.columnSettings')}>
        <Button icon={<SettingOutlined />} size="small" />
      </Tooltip>
    </Popover>
  );
};

export default ColumnVisibility;
