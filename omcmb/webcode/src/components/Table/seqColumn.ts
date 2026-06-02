import type { ColumnType } from 'antd/es/table';

// 统一的"序号"列(与 /device/list 的序号列同款样式:窄列、居中、置于最左)。
// 普通 antd Table 不像 DataTable 自带 showRowNumber,这里抽出一个工厂统一生成。
//
// 两种数据形态:
//   1. 客户端分页(dataSource 为全量数组,antd 本地分页)→ 传 dataSource,
//      按记录在全量中的绝对位置给序号,跨页连续(1..N)。
//   2. 服务端分页(dataSource 仅当前页切片)→ 传 current + pageSize,
//      序号 = (current - 1) * pageSize + 行内索引 + 1。
interface SeqColumnParams<T> {
  /** 列标题,通常传 t('table.rowNumber')(中文"序号"/英文"No.")。 */
  title: string;
  /** 服务端分页:当前页码(1-based)。 */
  current?: number;
  /** 服务端分页:每页条数。 */
  pageSize?: number;
  /** 客户端分页:全量数据源,用于按记录绝对位置计算序号。 */
  dataSource?: readonly T[];
}

const BASE = { key: '__seq__', width: 64, align: 'center' as const };

export function makeSeqColumn<T>(params: SeqColumnParams<T>): ColumnType<T> {
  const { title, current, pageSize, dataSource } = params;

  if (dataSource) {
    const indexOf = new Map<T, number>();
    dataSource.forEach((row, i) => indexOf.set(row, i));
    return {
      ...BASE,
      title,
      render: (_value, record) => (indexOf.get(record) ?? 0) + 1,
    };
  }

  const cur = current ?? 1;
  const size = pageSize ?? 0;
  return {
    ...BASE,
    title,
    render: (_value, _record, index) => (cur - 1) * size + (index ?? 0) + 1,
  };
}
