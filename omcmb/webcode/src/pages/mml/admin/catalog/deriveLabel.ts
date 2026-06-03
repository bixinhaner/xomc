/**
 * deriveLabel — 从 path 末段派生人类可读显示名。
 *
 * 规则（与 AddSubFieldsModal 原 deriveLabelEn 一致）：取 path 最后一段，
 * 在驼峰边界（小写后接大写）插空格，`_` / `-` 转空格。
 *   Device.Cell.UserLabel → "User Label"
 *   FOO_BAR               → "FOO BAR"
 *
 * 用于：批量添加预览的派生显示名 + PATH 单条编辑的占位符动态化。
 */
export function deriveLabel(path: string): string {
  const idx = path.lastIndexOf('.');
  const leaf = idx >= 0 ? path.slice(idx + 1) : path;
  let out = '';
  let prevLower = false;
  for (let i = 0; i < leaf.length; i++) {
    const ch = leaf[i];
    const isUpper = ch >= 'A' && ch <= 'Z';
    const isLower = ch >= 'a' && ch <= 'z';
    if (i > 0 && prevLower && isUpper) out += ' ';
    if (ch === '_' || ch === '-') {
      out += ' ';
    } else {
      out += ch;
    }
    prevLower = isLower;
  }
  return out;
}
