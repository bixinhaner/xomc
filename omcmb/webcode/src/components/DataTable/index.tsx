import React, { useCallback, useMemo, useState } from 'react';
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
  render?: (value: any, record: T, index: number) => React.ReactNode;
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
        const cellVal = record[key];
        if (cellVal === null || cellVal === undefined) return false;
        return String(cellVal).toLowerCase().includes(filterVal.toLowerCase());
      });
    }
    return data;
  }, [dataSource, columnFilters]);

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
                      onClick={() => {
                        void navigator.clipboard.writeText(text);
                        void message.success(t('table.copied'));
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

  const rowSelection: TableProps<T>['rowSelection'] = selectable
    ? {
        fixed: true,
        selectedRowKeys,
        onChange: handleSelectionChange,
      }
    : undefined;

  const rowClassName = useCallback(
    (record: T): string => {
      if (alarmRowStyle) {
        const severity = alarmRowStyle(record);
        if (severity) return `row-${severity}`;
      }
      return '';
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
        density={density}
        onDensityChange={setDensity}
        extraLeft={extraToolbarLeft}
        extraRight={extraToolbarRight}
      />

      <div className={`${styles.tableContainer} omc-data-table`}>
        <Table<T>
          columns={buildColumns}
          dataSource={filteredData}
          loading={loading}
          rowKey={rowKey as string}
          rowSelection={rowSelection}
          rowClassName={rowClassName}
          expandable={expandable}
          size={tableSize}
          scroll={scroll ?? { x: 'max-content' }}
          pagination={false}
          locale={{
            emptyText: <EmptyState description={t('common.noData')} />,
          }}
        />
      </div>

      {showPagination && (
        <div className={styles.paginationWrapper}>
          <Pagination
            current={currentPage}
            pageSize={pageSize}
            total={total ?? filteredData.length}
            showSizeChanger
            showQuickJumper
            showTotal={(total) => t('table.totalItems', { total })}
            pageSizeOptions={['10', '20', '50', '100']}
            onChange={onPageChange}
          />
        </div>
      )}
    </div>
  );
}

export default DataTable;
