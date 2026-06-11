/**
 * 通用「可拖拽列宽」工具（#214）。纯逻辑在 resizableColumnsCore.ts（可单测）。
 *
 * antd Table 无内置列宽拖拽：用 react-resizable 包表头单元格右缘做把手，配 antd
 * `components.header.cell` + `onHeaderCell` 注入宽度与回调。长 TR069 路径列（标准/私有 PATH）
 * 拉不宽看不全完整路径——本工具让任何带 `width` 的列可鼠标拖宽，可选持久化到 localStorage。
 *
 * 用法：
 *   const { columns, components, tableClassName } = useResizableColumns(rawCols, { storageKey });
 *   <ResizableColumnsStyle scope={tableClassName} />
 *   <Table columns={columns} components={components} className={tableClassName}
 *          tableLayout="fixed" scroll={{ x: 'max-content' }} />
 */
import { useCallback, useMemo, useState } from 'react';
import { Resizable, type ResizeCallbackData } from 'react-resizable';
import type { ColumnsType, ColumnType } from 'antd/es/table';
import {
  applyColumnResize,
  loadWidths,
  saveWidths,
  MIN_RESIZE_WIDTH,
  RESIZABLE_TABLE_CLASS,
  type ColWidthMap,
} from './resizableColumnsCore';

export { MIN_RESIZE_WIDTH, RESIZABLE_TABLE_CLASS, applyColumnResize } from './resizableColumnsCore';
export type { ColWidthMap } from './resizableColumnsCore';

interface ResizableTitleProps extends React.HTMLAttributes<HTMLTableCellElement> {
  width?: number;
  onResize?: (e: React.SyntheticEvent, data: ResizeCallbackData) => void;
}

/** 可拖拽表头单元格：右侧分隔线作为把手。无 width 的列退化为普通 <th>。 */
export function ResizableTitle({ width, onResize, ...restProps }: ResizableTitleProps) {
  if (width == null) {
    return <th {...restProps} />;
  }
  return (
    <Resizable
      width={width}
      height={0}
      handle={
        <span
          className="omc-col-resize-handle"
          onClick={(e) => e.stopPropagation()}
          onMouseDown={(e) => e.stopPropagation()}
        />
      }
      onResize={onResize}
      draggableOpts={{ enableUserSelectHack: false }}
    >
      <th {...restProps} />
    </Resizable>
  );
}

/** 把手 + 表头相对定位样式（作用域到 scope className，避免污染页面其它表格）。 */
export function ResizableColumnsStyle({ scope = RESIZABLE_TABLE_CLASS }: { scope?: string }) {
  return (
    <style>{`
      .${scope} .ant-table-thead th { position: relative; background-clip: padding-box; }
      .${scope} .omc-col-resize-handle {
        position: absolute; right: -5px; bottom: 0; z-index: 1;
        width: 10px; height: 100%; cursor: col-resize; touch-action: none;
      }
    `}</style>
  );
}

interface UseResizableColumnsOptions {
  /** 提供则把列宽持久化到 localStorage[storageKey]（按浏览器保留）。 */
  storageKey?: string;
  /** 最小列宽，默认 60。 */
  minWidth?: number;
}

interface UseResizableColumnsResult<T> {
  columns: ColumnsType<T>;
  components: { header: { cell: typeof ResizableTitle } };
  tableClassName: string;
}

/**
 * 给 antd 列表注入「可拖拽列宽」。只有**带 `width`** 的列可拖；其余原样返回。
 */
export function useResizableColumns<T>(
  columns: ColumnsType<T>,
  opts: UseResizableColumnsOptions = {},
): UseResizableColumnsResult<T> {
  const { storageKey, minWidth = MIN_RESIZE_WIDTH } = opts;
  const [widths, setWidths] = useState<ColWidthMap>(() => loadWidths(storageKey));

  const handleResize = useCallback(
    (key: string) => (_e: React.SyntheticEvent, { size }: ResizeCallbackData) => {
      setWidths((prev) => {
        const next = applyColumnResize(prev, key, size.width, minWidth);
        saveWidths(storageKey, next);
        return next;
      });
    },
    [storageKey, minWidth],
  );

  const merged = useMemo<ColumnsType<T>>(
    () =>
      columns.map((col) => {
        const c = col as ColumnType<T>;
        const key = String(c.key ?? c.dataIndex ?? '');
        if (c.width == null || !key) return col; // 无 width / 无 key 的列不可拖
        const width = widths[key] ?? (c.width as number);
        return {
          ...col,
          width,
          onHeaderCell: () => ({ width, onResize: handleResize(key) }) as ResizableTitleProps,
        };
      }),
    [columns, widths, handleResize],
  );

  return { columns: merged, components: { header: { cell: ResizableTitle } }, tableClassName: RESIZABLE_TABLE_CLASS };
}
