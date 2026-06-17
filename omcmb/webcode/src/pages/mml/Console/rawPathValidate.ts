import type { MMLOperationType } from '@core/types/mml';

/**
 * 校验「指定参数」(裸路径)单行 path 是否符合该操作类型的 TR-069 约束（plan A）。
 * 返回错误文案；null = 通过。空 path 视为通过（"至少一行非空"由上层另行约束）。
 *
 * 约束依据 TR-069 AddObject / DeleteObject 语义：
 *  - 裸路径不经字典翻译，不支持 `{i}` 占位符 —— 必须填具体实例号。
 *  - ADD(AddObject)：ObjectName 是对象表路径，以 `.` 结尾、末级不带实例号（新实例号由设备分配）。
 *  - RMV(DeleteObject)：ObjectName 是具体实例路径，以 `.<实例号>.` 结尾。
 */
export function validateRawPath(op: MMLOperationType | string, rawPath: string): string | null {
  const path = rawPath.trim();
  if (!path) return null;
  if (path.includes('{i}')) {
    return '裸路径不支持 {i} 占位符，请填写具体实例号';
  }
  if (op === 'ADD') {
    if (!path.endsWith('.')) return 'AddObject 路径需以 . 结尾（对象表层级）';
    const leaf = path.replace(/\.+$/, '').split('.').pop() ?? '';
    if (/^\d+$/.test(leaf)) return 'AddObject 末级不能是实例号，新实例号由设备分配';
    return null;
  }
  if (op === 'RMV' || op === 'DEL') {
    if (!/\.\d+\.$/.test(path)) {
      return 'DeleteObject 路径需以 .<实例号>. 结尾（指定要删除的实例）';
    }
    return null;
  }
  return null;
}

/** 该操作类型下，按 path 输入给出的占位提示（裸路径模式，均不含 {i}）。 */
export function rawPathPlaceholder(op: MMLOperationType | string): string {
  if (op === 'ADD') return '对象表路径，以 . 结尾，如 …NeighborList.LTECell.';
  if (op === 'RMV' || op === 'DEL') return '具体实例路径，以 .<实例号>. 结尾，如 …LTECell.3.';
  return 'Device.Services.FAPService.1.CellConfig…（具体实例号，勿用 {i}）';
}
