import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Badge, Button, Checkbox, Collapse, Drawer, Input, Tooltip } from 'antd';
import { HolderOutlined, SearchOutlined, SettingOutlined } from '@ant-design/icons';
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
import type { ColumnGroup } from './index';

interface Column {
  key: string;
  title: string;
  hidden?: boolean;
  group?: ColumnGroup;
}

interface ColumnVisibilityProps {
  tableId: string;
  columns: Column[];
  onChange: (hiddenKeys: string[]) => void;
  onOrderChange?: (orderedKeys: string[]) => void;
}

const VIS_STORAGE_PREFIX = 'omc_col_vis_';
const ORDER_STORAGE_PREFIX = 'omc_col_order_';

const GROUP_LABELS: Record<string, string> = {
  common: '公共字段',
  eNB: 'eNB 字段',
  gNB: 'gNB 字段',
  GSM: 'GSM 字段',
};

const GROUP_LABELS_EN: Record<string, string> = {
  common: 'Common',
  eNB: 'eNB',
  gNB: 'gNB',
  GSM: 'GSM',
};

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
    padding: '3px 4px',
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
  const labels = isZh ? GROUP_LABELS : GROUP_LABELS_EN;

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

  // ─── drawer & search state ─────────────────────────────────────────────

  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');

  // ─── group structure ───────────────────────────────────────────────────

  const groups = useMemo(() => {
    const groupSet = new Set<string>();
    for (const col of columns) {
      if (col.group) groupSet.add(col.group);
    }
    return Array.from(groupSet);
  }, [columns]);

  // ─── search filter helper ──────────────────────────────────────────────

  const matchSearch = useCallback(
    (col: Column) => {
      if (!search) return true;
      return col.title.toLowerCase().includes(search.toLowerCase());
    },
    [search]
  );

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

  const handleGroupToggle = useCallback(
    (groupKey: string, checked: boolean) => {
      const groupCols = orderedColumns.filter((c) => c.group === groupKey);
      const groupKeys = groupCols.map((c) => c.key);
      let next: string[];
      if (checked) {
        // show all in group
        next = hiddenKeys.filter((k) => !groupKeys.includes(k));
      } else {
        // hide all in group
        const toHide = groupKeys.filter((k) => !hiddenKeys.includes(k));
        next = [...hiddenKeys, ...toHide];
      }
      persistHidden(next);
    },
    [orderedColumns, hiddenKeys, persistHidden]
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

  // ─── render helpers ────────────────────────────────────────────────────

  const getGroupStats = useCallback(
    (groupKey: string) => {
      const cols = orderedColumns.filter((c) => c.group === groupKey);
      const visible = cols.filter((c) => !hiddenKeys.includes(c.key)).length;
      return { total: cols.length, visible };
    },
    [orderedColumns, hiddenKeys]
  );

  const getGroupCheckState = useCallback(
    (groupKey: string): { checked: boolean; indeterminate: boolean } => {
      const { total, visible } = getGroupStats(groupKey);
      return {
        checked: visible === total && total > 0,
        indeterminate: visible > 0 && visible < total,
      };
    },
    [getGroupStats]
  );

  // Total visible count
  const totalVisible = columns.filter((c) => !hiddenKeys.includes(c.key)).length;

  // ─── collapse items ────────────────────────────────────────────────────

  const collapseItems = groups.map((groupKey) => {
    const groupCols = orderedColumns.filter((c) => c.group === groupKey && matchSearch(c));
    const { total, visible } = getGroupStats(groupKey);
    const checkState = getGroupCheckState(groupKey);

    return {
      key: groupKey,
      label: (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, width: '100%' }}>
          <span onClick={(e) => e.stopPropagation()}>
            <Checkbox
              checked={checkState.checked}
              indeterminate={checkState.indeterminate}
              onChange={(e) => handleGroupToggle(groupKey, e.target.checked)}
            />
          </span>
          <span style={{ flex: 1 }}>{labels[groupKey] ?? groupKey}</span>
          <Badge
            count={`${visible}/${total}`}
            style={{
              backgroundColor: visible === 0 ? '#d9d9d9' : '#1677ff',
              fontSize: 11,
            }}
            overflowCount={999}
          />
        </div>
      ),
      children: groupCols.length > 0 ? (
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
          <SortableContext items={groupCols.map((c) => c.key)} strategy={verticalListSortingStrategy}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
              {groupCols.map((col) => (
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
      ) : (
        <div style={{ color: '#bfbfbf', fontSize: 12, padding: '8px 0' }}>
          {isZh ? '无匹配列' : 'No matching columns'}
        </div>
      ),
    };
  });

  // ─── render ────────────────────────────────────────────────────────────

  return (
    <>
      <Tooltip title={t('table.columnSettings')}>
        <Button icon={<SettingOutlined />} size="small" onClick={() => setOpen(true)} />
      </Tooltip>

      <Drawer
        title={
          <span>
            {t('table.columnSettings')}
            <span style={{ fontSize: 12, color: '#8c8c8c', marginLeft: 8, fontWeight: 'normal' }}>
              {totalVisible}/{columns.length}
            </span>
          </span>
        }
        open={open}
        onClose={() => setOpen(false)}
        width={380}
        styles={{ body: { padding: '12px 16px' } }}
        extra={
          <Button size="small" onClick={handleReset}>
            {isZh ? '重置' : 'Reset'}
          </Button>
        }
      >
        <Input
          placeholder={isZh ? '搜索列名...' : 'Search columns...'}
          prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
          allowClear
          size="small"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          style={{ marginBottom: 12 }}
        />

        <Collapse
          defaultActiveKey={['common']}
          size="small"
          items={collapseItems}
          style={{ background: 'transparent' }}
        />
      </Drawer>
    </>
  );
};

export default ColumnVisibility;
