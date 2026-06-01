/**
 * MenuManagement 页面常量。
 * 从 index.tsx 拆出（react-refresh/only-export-components：页面文件只导出组件）。
 * DEFAULT_OPERATIONS 为 export 占位（T-0136：保留以便未来三级节点默认操作按钮复用）。
 */

// 默认的操作按钮（三级节点）
export const DEFAULT_OPERATIONS = [
  { key: 'query', name: '查询' },
  { key: 'add', name: '添加' },
  { key: 'edit', name: '修改' },
  { key: 'delete', name: '删除' },
  { key: 'export', name: '导出' },
  { key: 'import', name: '导入' },
];
