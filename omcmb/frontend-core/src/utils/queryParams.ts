/**
 * Query string 序列化辅助：数组 → 重复键（k=a&k=b，无方括号），标量原样。
 *
 * 用于值内合法含逗号的 query 参数（如 PM 维度 object_ldn = "DeviceGroup=<uuid>,Tech=lte"），
 * CSV-join 会被后端逗号切分破坏整值（issue #401 / #619 同根因）。改用重复键，整值保留。
 *
 * - Axios 默认会发 `key[]=a&key[]=b`（带方括号），方括号键被 gin `QueryArray("key")` 收不到。
 * - 用本函数显式发 `key=a&key=b`（无方括号），与 `QueryArray` / `parseRepeatedQuery` 对齐。
 * - 每个键/值都 encodeURIComponent，避免 UUID / 逗号 / 等号 / 换行被破坏。
 */
export function serializeRepeatedParams(params: Record<string, unknown>): string {
  const parts: string[] = [];
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null) continue;
    const ek = encodeURIComponent(key);
    if (Array.isArray(value)) {
      for (const v of value) {
        if (v === undefined || v === null) continue;
        parts.push(`${ek}=${encodeURIComponent(String(v))}`);
      }
    } else {
      parts.push(`${ek}=${encodeURIComponent(String(value))}`);
    }
  }
  return parts.join('&');
}
