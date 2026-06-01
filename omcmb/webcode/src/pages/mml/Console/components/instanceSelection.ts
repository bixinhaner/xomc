/**
 * InstancePicker 纯逻辑工具。
 * 从 InstancePicker.tsx 拆出（react-refresh/only-export-components：
 * 组件文件只导出组件，纯工具集中本文件，便于单元测试直测）。
 */
import type { Statement } from '@core/types/mmlConsole';

// 兼容旧 rmvInstanceIndex（MML parser 单 Index 入口）：union 出当前生效的实例号列表。
export function effectiveIndices(stmt: Statement): number[] {
  if (stmt.rmvInstanceIndices && stmt.rmvInstanceIndices.length > 0) {
    return stmt.rmvInstanceIndices;
  }
  if (typeof stmt.rmvInstanceIndex === 'number') {
    return [stmt.rmvInstanceIndex];
  }
  return [];
}

// antd tags 模式 onChange 给的是 string[]（即便 options.value 是 number）。
// 解析为非负整数集合，drop 非法 / 重复。
export function parseTagValues(raw: string[]): number[] {
  const out = new Set<number>();
  for (const r of raw) {
    const n = Number.parseInt(String(r).trim(), 10);
    if (Number.isFinite(n) && n >= 0) out.add(n);
  }
  return Array.from(out);
}
