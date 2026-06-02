// 设备列表每页条数（服务端分页）——每次最多选择 50 条设备执行命令
export const DEVICE_PAGE_SIZE = 50;

// 设备列表单行高度（px，与 List.Item 实测一致）+ 可视行数。
// 列表固定为 13 行高度，超过 13 条出现滚动条。
export const DEVICE_ROW_HEIGHT = 48;
export const DEVICE_VISIBLE_ROWS = 13;

// 命令树一次性加载数量（后端最大限制 999）
export const COMMAND_PAGE_SIZE = 999;
