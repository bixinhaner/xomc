/**
 * MML Console 实例 arity 相关纯工具函数。
 * 从 InstanceArityInput.tsx 拆出（react-refresh/only-export-components：
 * 组件文件只导出组件，纯工具集中本文件）。
 */

/**
 * Greek 字母对应每层的 selector key。
 * arity 通常 ≤ 3（v2.3 catalog 最深 MU→Slot→EU），余量到 ε 应对未来扩展。
 */
const SELECTOR_KEYS = ['iα', 'iβ', 'iγ', 'iδ', 'iε'] as const;

/** 按 arity 生成 selector key 数组。 */
export function selectorKeysForArity(arity: number): string[] {
  return SELECTOR_KEYS.slice(0, Math.min(arity, SELECTOR_KEYS.length));
}

/**
 * deriveArityFromSubFields — 由 sub_field.tr069Path 推断 instance arity。
 * 后端 BuildTree 尚未暴露 mml_command_groups.instance_arity，前端 derive。
 *
 * 同 command 内所有 sub_field 应同 arity（catalog 由组级 instance_arity 决定）；
 * 异常时取最大值，由后端校验数量 mismatch 兜底。
 */
export function deriveArityFromSubFields(
  subFields: ReadonlyArray<{ tr069Path?: string }>,
): number {
  let max = 0;
  for (const sf of subFields) {
    const path = sf.tr069Path ?? '';
    let count = 0;
    let idx = path.indexOf('.{i}.');
    while (idx !== -1) {
      count++;
      idx = path.indexOf('.{i}.', idx + 5);
    }
    if (count > max) max = count;
  }
  return max;
}
