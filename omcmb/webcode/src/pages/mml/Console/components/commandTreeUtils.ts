/**
 * commandTreeUtils — CommandTree.tsx 的纯字符串辅助函数。
 *
 * 拆出独立文件的目的：单元测试时直接 import 这两个 helper，避免 import CommandTree.tsx
 * 主组件牵连整条 React + antd + axios 依赖链。
 */

/**
 * chapterSortKey 把章节码归一化为可排序字符串。
 *
 * 与后端 `group_tree_repository.go::chapterSortKey` 行为对偶：
 *   - 非空：原样返回（SA<SB<...<SR 字典序天然正确）
 *   - 空 / undefined（老 catalog 未分章）：映射为高位 sentinel "~~~~~" 排末位
 */
export function chapterSortKey(chapter: string | undefined): string {
  return chapter && chapter !== '' ? chapter : '~~~~~';
}

/**
 * stripOpSuffix 去掉 backend displayName 末尾的 `(OP CODE)` 部分。
 *
 * 输入示例：
 *   "设备信息(LST DEVICE_INFO)"  → "设备信息"
 *   "设备信息"                    → "设备信息"（无括号原样返回）
 *
 * 用途：CommandTree 用 OP Tag + object name 渲染叶子；避免与括号尾视觉重复。
 */
export function stripOpSuffix(displayName: string): string {
  const i = displayName.lastIndexOf('(');
  if (i < 0) return displayName;
  const closing = displayName.lastIndexOf(')');
  if (closing < i) return displayName;
  return displayName.slice(0, i).trimEnd();
}
