import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Checkbox, Popover, Tooltip } from 'antd';
import { HolderOutlined, SettingOutlined } from '@ant-design/icons';
import {
  DndContext,
  closestCenter,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import type { DragEndEvent } from '@dnd-kit/core';
import {
  SortableContext,
  useSortable,
  verticalListSortingStrategy,
  arrayMove,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
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

// 导出(如 device/list 列表导出)需读取同一份"列设置"状态,故前缀对外暴露,
// 避免常量在多处硬编码漂移。
export const VIS_STORAGE_PREFIX = 'omc_col_vis_';
export const ORDER_STORAGE_PREFIX = 'omc_col_order_';

// ─── Sortable row ───────────────────────────────────────────────────────────

interface SortableItemProps {
  id: string;
  title: string;
  checked: boolean;
  onToggle: (key: string, checked: boolean) => void;
}

const SortableItem: React.FC<SortableItemProps> = ({ id, title, checked, onToggle }) => {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id });

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    display: 'flex',
    alignItems: 'center',
    gap: 6,
    padding: '3px 0',
    borderRadius: 4,
    background: isDragging ? '#e6f4ff' : undefined,
    cursor: 'default',
  };

  return (
    <div ref={setNodeRef} style={style} {...attributes}>
      <span {...listeners} style={{ cursor: 'grab', color: '#bfbfbf', display: 'flex', flexShrink: 0 }}>
        <HolderOutlined />
      </span>
      <Checkbox
        checked={checked}
        onChange={(e) => onToggle(id, e.target.checked)}
        style={{ width: '100%' }}
      >
        <span style={{ fontSize: 13 }}>{title}</span>
      </Checkbox>
    </div>
  );
};

// ─── Main component ─────────────────────────────────────────────────────────

const ColumnVisibility: React.FC<ColumnVisibilityProps> = ({
  tableId,
  columns,
  onChange,
  onOrderChange,
}) => {
  const t = useT();
  const visKey = `${VIS_STORAGE_PREFIX}${tableId}`;
  const orderKey = `${ORDER_STORAGE_PREFIX}${tableId}`;
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

  // ─── column order ───────────────────────────────────────────────────────

  const getInitialOrder = (): string[] => {
    try {
      const stored = localStorage.getItem(orderKey);
      if (stored) {
        const parsed = JSON.parse(stored) as string[];
        if (parsed.length > 0) return parsed;
      }
    } catch { /* ignore */ }
    return columns.map((c) => c.key);
  };

  const [orderedKeys, setOrderedKeys] = useState<string[]>(getInitialOrder);

  const effectiveOrder = useMemo(() => {
    const set = new Set(orderedKeys);
    const extra = columns.filter((c) => !set.has(c.key)).map((c) => c.key);
    return extra.length > 0 ? [...orderedKeys, ...extra] : orderedKeys;
  }, [orderedKeys, columns]);

  const orderedColumns = useMemo(() => {
    const map = new Map(columns.map((c) => [c.key, c]));
    return effectiveOrder.map((k) => map.get(k)).filter(Boolean) as Column[];
  }, [columns, effectiveOrder]);

  // ─── init ───────────────────────────────────────────────────────────────

  useEffect(() => {
    onChange(hiddenKeys);
    onOrderChange?.(effectiveOrder);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const [open, setOpen] = useState(false);

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
    const defaultOrder = columns.map((c) => c.key);
    setOrderedKeys(defaultOrder);
    onOrderChange?.(defaultOrder);
    try { localStorage.removeItem(orderKey); } catch { /* ignore */ }
  }, [columns, persistHidden, onOrderChange, orderKey]);

  // ─── drag-and-drop ────────────────────────────────────────────────────

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 5 } }));

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over } = event;
      if (!over || active.id === over.id) return;

      const oldIndex = effectiveOrder.indexOf(active.id as string);
      const newIndex = effectiveOrder.indexOf(over.id as string);
      if (oldIndex === -1 || newIndex === -1) return;

      const newOrder = arrayMove(effectiveOrder, oldIndex, newIndex);
      setOrderedKeys(newOrder);
      onOrderChange?.(newOrder);
      try { localStorage.setItem(orderKey, JSON.stringify(newOrder)); } catch { /* ignore */ }
    },
    [effectiveOrder, onOrderChange, orderKey]
  );

  // ─── filterable columns (exclude fixed columns like actions) ───────────

  const settableColumns = useMemo(
    () => orderedColumns.filter((c) => c.key !== 'actions'),
    [orderedColumns]
  );

  const totalVisible = settableColumns.filter((c) => !hiddenKeys.includes(c.key)).length;

  // ─── popover content ──────────────────────────────────────────────────

  const content = (
    <div style={{ width: 220 }}>
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
      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
        <SortableContext items={settableColumns.map((c) => c.key)} strategy={verticalListSortingStrategy}>
          <div style={{ maxHeight: 400, overflowY: 'auto' }}>
            {settableColumns.map((col) => (
              <SortableItem
                key={col.key}
                id={col.key}
                title={col.title}
                checked={!hiddenKeys.includes(col.key)}
                onToggle={handleToggle}
              />
            ))}
          </div>
        </SortableContext>
      </DndContext>
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
