import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Pagination, Table, Tooltip, Typography } from 'antd';
import type { TableProps } from 'antd';
import { CopyOutlined } from '@ant-design/icons';
import { message } from 'antd';
import { useT } from '@/hooks/useT';
import Toolbar from './Toolbar';
import EmptyState from './EmptyState';
import ColumnFilter from './ColumnFilter';
import styles from './DataTable.module.css';

export type ColumnGroup = 'common' | 'eNB' | 'gNB' | 'GSM';

export interface DataTableColumn<T> {
  key: string;
  title: string;
  dataIndex?: string;
  width?: number;
  fixed?: 'left' | 'right';
  sorter?: boolean;
  ellipsis?: boolean;
  render?: (value: unknown, record: T, index: number) => React.ReactNode;
  filterable?: boolean;
  filterType?: 'text' | 'select' | 'date';
  filterOptions?: { label: string; value: string }[];
  hidden?: boolean;
  copyable?: boolean;
  mono?: boolean;
  group?: ColumnGroup;
  headerRender?: React.ReactNode;
}

export interface BatchAction {
  key: string;
  label: string;
  icon?: React.ReactNode;
  danger?: boolean;
  disabled?: boolean;
  onClick: (selectedKeys: React.Key[]) => void;
}

export interface DataTableProps<T> {
  tableId: string;
  columns: DataTableColumn<T>[];
  dataSource: T[];
  loading?: boolean;
  rowKey: string | ((record: T) => string);
  selectable?: boolean;
  selectedRowKeys?: React.Key[];
  onSelectionChange?: (keys: React.Key[], rows: T[]) => void;
  total?: number;
  pageSize?: number;
  currentPage?: number;
  onPageChange?: (page: number, size: number) => void;
  batchActions?: BatchAction[];
  onRefresh?: () => void;
  onExport?: (format: 'xlsx' | 'csv') => void;
  alarmRowStyle?: (record: T) => 'critical' | 'major' | 'minor' | 'warning' | null;
  expandable?: TableProps<T>['expandable'];
  defaultDensity?: 'compact' | 'default' | 'comfortable';
  extraToolbarLeft?: React.ReactNode;
  extraToolbarRight?: React.ReactNode;
  scroll?: { x?: number | string; y?: number | string };
  size?: 'small' | 'middle' | 'large';
  showPagination?: boolean;
  showRowNumber?: boolean;
  rowNumberTitle?: string;
  /**
   * 自适应表格高度：true 时按容器实测高度减 thead 注入 antd Table 的 scroll.y。
   * 默认 false——caller 未传 scroll.y 时维持 antd 原生「自然撑高」行为，
   * 避免 ResizeObserver 在 flex/overflow 容器里抖动导致分页栏漂移。
   * DeviceList 等需要分页紧贴底部的页面显式 autoFitHeight 即可。
   */
  autoFitHeight?: boolean;
  /**
   * 「实时刷新」开关切换回调。Toolbar 内的 SyncOutlined 按钮翻转时上抛布尔值；
   * caller 收到后应把它接到 useQuery 的 `refetchInterval`（典型 5000ms）。
   * 无此回调时按钮仍可点（仅本地视觉切换），但不会真正发起周期请求 —— 这正
   * 是历史上 DeviceList "开启实时刷新无效"bug 的成因。
   */
  onRealtimeRefreshChange?: (enabled: boolean) => void;
}

type Density = 'compact' | 'default' | 'comfortable';

const DENSITY_SIZE_MAP: Record<Density, 'small' | 'middle' | 'large'> = {
  compact: 'small',
  default: 'middle',
  comfortable: 'large',
};

function DataTable<T>(
  props: DataTableProps<T>
): React.ReactElement {
  const {
    tableId,
    columns,
    dataSource,
    loading = false,
    rowKey,
    selectable = false,
    selectedRowKeys: controlledSelectedKeys,
    onSelectionChange,
    total,
    pageSize = 20,
    currentPage = 1,
    onPageChange,
    batchActions = [],
    onRefresh,
    onExport,
    alarmRowStyle,
    expandable,
    defaultDensity = 'default',
    extraToolbarLeft,
    extraToolbarRight,
    scroll,
    size,
    showPagination = true,
    showRowNumber = false,
    rowNumberTitle,
    autoFitHeight = false,
    onRealtimeRefreshChange,
  } = props;

  const t = useT();

  const [density, setDensity] = useState<Density>(defaultDensity);
  const [hiddenKeys, setHiddenKeys] = useState<string[]>([]);
  const [columnFilters, setColumnFilters] = useState<Record<string, string | undefined>>({});
  const [internalSelectedKeys, setInternalSelectedKeys] = useState<React.Key[]>([]);

  const selectedRowKeys = controlledSelectedKeys ?? internalSelectedKeys;

  const handleSelectionChange = useCallback(
    (keys: React.Key[], rows: T[]) => {
      setInternalSelectedKeys(keys);
      onSelectionChange?.(keys, rows);
    },
    [onSelectionChange]
  );

  const handleColumnFilter = useCallback(
    (columnKey: string, value: string | undefined) => {
      setColumnFilters((prev) => ({ ...prev, [columnKey]: value }));
    },
    []
  );

  const [columnOrder, setColumnOrder] = useState<string[]>([]);

  const orderedColumns = useMemo(() => {
    if (columnOrder.length === 0) return columns;
    const colMap = new Map(columns.map((c) => [c.key, c]));
    const ordered: DataTableColumn<T>[] = [];
    for (const key of columnOrder) {
      const col = colMap.get(key);
      if (col) {
        ordered.push(col);
        colMap.delete(key);
      }
    }
    // append any columns not in the order (e.g. newly added)
    for (const col of colMap.values()) ordered.push(col);
    return ordered;
  }, [columns, columnOrder]);

  const filteredData = useMemo(() => {
    let data = dataSource;
    for (const [key, filterVal] of Object.entries(columnFilters)) {
      if (!filterVal) continue;
      data = data.filter((record) => {
        const cellVal = (record as Record<string, unknown>)[key];
        if (cellVal === null || cellVal === undefined) return false;
        return String(cellVal).toLowerCase().includes(filterVal.toLowerCase());
      });
    }
    return data;
  }, [dataSource, columnFilters]);

  const pagedData = useMemo(() => {
    if (!showPagination) return filteredData;
    // Server-side pagination: onPageChange is provided, data is already paginated
    if (onPageChange) return filteredData;
    // Client-side pagination: slice the data
    const start = (currentPage - 1) * pageSize;
    return filteredData.slice(start, start + pageSize);
  }, [filteredData, showPagination, currentPage, pageSize, onPageChange]);

  // 自适应表格高度：仅当 autoFitHeight=true && caller 未显式传 scroll.y 时启用，
  // 按 tableContainer 的实际可用高度减去 thead 后传给 antd Table 的 body。
  //
  // 默认关闭的原因：caller 把 DataTable 嵌进 flex/overflow:hidden 容器（例如系统
  // 管理的 Card body）时，Table 切到 scroll.y 模式会改变内部 DOM 高度（sticky
  // thead + 横向滚动条占位），反向触发 ResizeObserver → setState 反馈环，分页栏
  // 每帧 1-2px 向上漂移直至盖住列表。
  const tableContainerRef = useRef<HTMLDivElement>(null);
  const [autoBodyY, setAutoBodyY] = useState<number | undefined>(undefined);

  useEffect(() => {
    if (!autoFitHeight) return;
    if (scroll?.y !== undefined) return;
    const el = tableContainerRef.current;
    if (!el) return;

    let lastY: number | undefined;
    const recalc = () => {
      const thead = el.querySelector('.ant-table-thead') as HTMLElement | null;
      // 默认 thead 约 40px（small 密度），首帧 thead 还没渲染时用兜底值
      // 避免 body 撑过头反向滚出来。
      const headerH = thead?.offsetHeight ?? 40;
      const y = Math.max(0, el.clientHeight - headerH - 2);
      // 1px 抖动死区：thead 在 scroll 切换瞬间 offsetHeight 可能 ±1 来回
      // 抖动，没有死区会触发无限 setState → ResizeObserver → recalc 循环。
      if (y > 0 && (lastY === undefined || Math.abs(y - lastY) > 1)) {
        lastY = y;
        setAutoBodyY(y);
      }
    };

    recalc();
    const ro = new ResizeObserver(recalc);
    ro.observe(el);
    // antd Table thead 在首次 effect 时往往还没挂入 DOM；50ms 后再算一次。
    const t = window.setTimeout(recalc, 50);
    return () => {
      ro.disconnect();
      window.clearTimeout(t);
    };
  }, [autoFitHeight, scroll]);

  const effectiveScroll = useMemo<TableProps<T>['scroll']>(() => {
    if (scroll?.y !== undefined) return scroll;
    if (!autoFitHeight) {
      // 未启用自适应 → 维持 antd 原生行为：仅注入默认 x='max-content'
      return { ...(scroll ?? {}), x: scroll?.x ?? 'max-content' };
    }
    return {
      ...(scroll ?? {}),
      x: scroll?.x ?? 'max-content',
      ...(autoBodyY !== undefined ? { y: autoBodyY } : {}),
    };
  }, [scroll, autoFitHeight, autoBodyY]);

  const buildColumns = useMemo((): TableProps<T>['columns'] => {
    return orderedColumns
      .filter((col) => !hiddenKeys.includes(col.key))
      .map((col) => {
        const titleContent = col.headerRender ?? col.title;
        const titleNode = (
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 2 }}>
            {titleContent}
            {col.filterable && (
              <ColumnFilter
                columnKey={col.key}
                filterType={col.filterType}
                filterOptions={col.filterOptions}
                value={columnFilters[col.key]}
                onFilter={handleColumnFilter}
              />
            )}
          </span>
        );

        const renderFn = col.render
          ? col.render
          : (value: unknown) => {
              const text = value !== null && value !== undefined ? String(value) : '-';
              const displayNode = col.mono ? (
                <span className="cell-mono">{text}</span>
              ) : (
                text
              );

              if (col.copyable && value !== null && value !== undefined) {
                return (
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                    {col.ellipsis ? (
                      <Tooltip title={text}>
                        <Typography.Text
                          ellipsis
                          style={col.mono ? { fontFamily: 'monospace' } : undefined}
                        >
                          {text}
                        </Typography.Text>
                      </Tooltip>
                    ) : (
                      displayNode
                    )}
                    <CopyOutlined
                      style={{ fontSize: 11, color: '#bfbfbf', cursor: 'pointer' }}
                      onClick={(e) => {
                        e.stopPropagation();
                        const doCopy = async () => {
                          try {
                            if (navigator.clipboard && window.isSecureContext) {
                              await navigator.clipboard.writeText(text);
                            } else {
                              // Fallback for non-secure context (HTTP)
                              const textarea = document.createElement('textarea');
                              textarea.value = text;
                              textarea.style.position = 'fixed';
                              textarea.style.left = '-9999px';
                              document.body.appendChild(textarea);
                              textarea.select();
                              document.execCommand('copy');
                              document.body.removeChild(textarea);
                            }
                            void message.success(t('table.copied'));
                          } catch {
                            void message.error(t('common.copyFailed'));
                          }
                        };
                        void doCopy();
                      }}
                    />
                  </span>
                );
              }
              return displayNode;
            };

        return {
          key: col.key,
          title: titleNode,
          dataIndex: col.dataIndex,
          width: col.width,
          fixed: col.fixed,
          sorter: col.sorter,
          ellipsis: col.ellipsis,
          render: renderFn as TableProps<T>['columns'] extends (infer C)[]
            ? C extends { render?: infer R }
              ? R
              : never
            : never,
        };
      });
  }, [orderedColumns, hiddenKeys, columnFilters, handleColumnFilter]);

  const rowSelection: TableProps<T>['rowSelection'] = (selectable || showRowNumber)
    ? {
        fixed: true,
        ...(selectable ? { selectedRowKeys, onChange: handleSelectionChange } : {}),
        columnWidth: showRowNumber ? (selectable ? 90 : 60) : 40,
        renderCell: (_checked, _record, index, originNode) => {
          const rowNumber = (currentPage - 1) * pageSize + (index ?? 0) + 1;
          if (showRowNumber && selectable) {
            return (
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, lineHeight: '1' }}>
                <span style={{ minWidth: 30, textAlign: 'center', color: 'var(--color-neutral-600)', fontSize: 13 }}>
                  {rowNumber}
                </span>
                {originNode}
              </div>
            );
          }
          if (showRowNumber) {
            return (
              <span style={{ minWidth: 30, textAlign: 'center', color: 'var(--color-neutral-600)', fontSize: 12 }}>
                {rowNumber}
              </span>
            );
          }
          return originNode;
        },
        columnTitle: showRowNumber && !selectable
          ? () => (
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <span style={{ minWidth: 30, textAlign: 'center', fontSize: 12 }}>
                  {rowNumberTitle ?? t('table.rowNumber')}
                </span>
              </div>
            )
          : undefined,
      }
    : undefined;

  const rowClassName = useCallback(
    (record: T, index?: number): string => {
      const classes: string[] = [];
      if (index !== undefined) {
        classes.push(index % 2 === 0 ? 'row-odd' : 'row-even');
      }
      if (alarmRowStyle) {
        const severity = alarmRowStyle(record);
        if (severity) classes.push(`row-${severity}`);
      }
      return classes.join(' ');
    },
    [alarmRowStyle]
  );

  const tableSize = size ?? DENSITY_SIZE_MAP[density];

  const columnDefs = orderedColumns.map((c) => ({ key: c.key, title: c.title, hidden: c.hidden, group: c.group }));

  return (
    <div className={styles.dataTableWrapper}>
      <Toolbar
        tableId={tableId}
        columns={columnDefs}
        selectedRowKeys={selectedRowKeys}
        batchActions={batchActions}
        onRefresh={onRefresh}
        onExport={onExport}
        onColumnVisibilityChange={setHiddenKeys}
        onColumnOrderChange={setColumnOrder}
        onRefreshLockChange={onRealtimeRefreshChange}
        density={density}
        onDensityChange={setDensity}
        extraLeft={extraToolbarLeft}
        extraRight={extraToolbarRight}
      />

      <div ref={tableContainerRef} className={`${styles.tableContainer} omc-data-table`}>
        <Table<T>
          columns={buildColumns}
          dataSource={pagedData}
          loading={loading}
          rowKey={rowKey as string}
          rowSelection={rowSelection}
          rowClassName={rowClassName}
          expandable={expandable}
          size={tableSize}
          scroll={effectiveScroll}
          pagination={false}
          locale={{
            emptyText: <EmptyState description={t('common.noData')} />,
          }}
        />
      </div>

      {showPagination && (
        <div className={styles.paginationWrapper}>
          <div className={styles.paginationInfo}>
            {t('table.totalItems', { total: total ?? filteredData.length })}
          </div>
          <div className={styles.paginationControls}>
            <Pagination
              current={currentPage}
              pageSize={pageSize}
              total={total ?? filteredData.length}
              showSizeChanger
              showQuickJumper
              pageSizeOptions={['10', '20', '50', '100']}
              onChange={onPageChange ?? (() => {})}
            />
          </div>
        </div>
      )}
    </div>
  );
}

export default DataTable;
