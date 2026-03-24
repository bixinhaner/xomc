import React, { useState } from 'react';
import { Button, Space, Tooltip } from 'antd';
import { ReloadOutlined, SyncOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';
import styles from './DataTable.module.css';
import ColumnVisibility from './ColumnVisibility';
import DensityToggle from './DensityToggle';
import ExportButton from './ExportButton';

type Density = 'compact' | 'default' | 'comfortable';

interface BatchAction {
  key: string;
  label: string;
  icon?: React.ReactNode;
  danger?: boolean;
  onClick: (selectedKeys: React.Key[]) => void;
}

interface ColumnDef {
  key: string;
  title: string;
  hidden?: boolean;
  group?: string;
}

interface ToolbarProps {
  tableId: string;
  columns: ColumnDef[];
  selectedRowKeys: React.Key[];
  batchActions?: BatchAction[];
  onRefresh?: () => void;
  onExport?: (format: 'xlsx' | 'csv') => void;
  onColumnVisibilityChange: (hiddenKeys: string[]) => void;
  onColumnOrderChange?: (orderedKeys: string[]) => void;
  onRefreshLockChange?: (locked: boolean) => void;
  density: Density;
  onDensityChange: (d: Density) => void;
  extraLeft?: React.ReactNode;
  extraRight?: React.ReactNode;
}

const Toolbar: React.FC<ToolbarProps> = ({
  tableId,
  columns,
  selectedRowKeys,
  batchActions = [],
  onRefresh,
  onExport,
  onColumnVisibilityChange,
  onColumnOrderChange,
  onRefreshLockChange,
  density,
  onDensityChange,
  extraLeft,
  extraRight,
}) => {
  const t = useT();
  const hasSelection = selectedRowKeys.length > 0;
  const [realtimeRefreshEnabled, setRealtimeRefreshEnabled] = useState(false);

  return (
    <div className={styles.toolbar}>
      {/* Left side */}
      <div className={styles.toolbarLeft}>
        {extraLeft}
        {batchActions.length > 0 && (
          <>
            {hasSelection && (
              <span className={styles.selectionBadge}>
                {t('table.selected', { count: selectedRowKeys.length })}
              </span>
            )}
            {batchActions.map((action) => (
              <Button
                key={action.key}
                size="small"
                danger={action.danger}
                icon={action.icon}
                disabled={!hasSelection}
                onClick={() => action.onClick(selectedRowKeys)}
              >
                {action.label}
              </Button>
            ))}
          </>
        )}
      </div>

      {/* Right side */}
      <Space size={4} className={styles.toolbarRight}>
        {extraRight}
        <Tooltip title={realtimeRefreshEnabled ? t('table.disableRealtimeRefresh') : t('table.enableRealtimeRefresh')}>
          <Button
            icon={<SyncOutlined spin={realtimeRefreshEnabled} />}
            size="small"
            type={realtimeRefreshEnabled ? 'primary' : 'default'}
            ghost={realtimeRefreshEnabled}
            onClick={() => {
              const next = !realtimeRefreshEnabled;
              setRealtimeRefreshEnabled(next);
              onRefreshLockChange?.(next);
            }}
          />
        </Tooltip>
        <ColumnVisibility
          tableId={tableId}
          columns={columns}
          onChange={onColumnVisibilityChange}
          onOrderChange={onColumnOrderChange}
        />
        <DensityToggle density={density} onChange={onDensityChange} />
        {onExport && <ExportButton onExport={onExport} />}
        {onRefresh && (
          <Tooltip title={t('common.refresh')}>
            <Button
              icon={<ReloadOutlined />}
              size="small"
              disabled={realtimeRefreshEnabled}
              onClick={onRefresh}
            />
          </Tooltip>
        )}
      </Space>
    </div>
  );
};

export default Toolbar;
