/**
 * T-0130 GPV instance index parser.
 *
 * 从 GPV (GetParameterValues) 响应的 ParameterList 中提取目标对象下的实例索引集合。
 *
 * TR-069 GPV 响应：当请求 `Device.Foo.` 时，CPE 返回该对象下所有 leaf 参数，
 * 形如 `Device.Foo.1.Bar = "x"`, `Device.Foo.2.Bar = "y"`。本 util 提取
 * 紧跟 targetObject 后的纯数字段作为实例索引（去重 + 升序）。
 *
 * 设计取舍：
 * - 仅返回数字索引（多实例对象规约）。命名实例（如 `Device.Foo.WAN.`）不在本期范围。
 * - targetObject 必须以 "." 结尾（TR-069 对象路径规约）；否则视为非法返回 []。
 * - 空 list / 无匹配 / 非法 input 一律返 []（不抛错）— UI 据空集渲染"无实例"。
 *
 * @example
 *   parseGpvInstances(
 *     [
 *       { name: "Device.IP.Interface.1.Enable", value: "true" },
 *       { name: "Device.IP.Interface.1.IPv4Address.1.IPAddress", value: "10.0.0.1" },
 *       { name: "Device.IP.Interface.3.Enable", value: "false" },
 *     ],
 *     "Device.IP.Interface.",
 *   )
 *   → [1, 3]
 */

export interface GpvParameter {
  name: string;
  value?: string;
  type?: string;
}

/**
 * 从 GPV ParameterList 中提取 targetObject 下的实例索引集合。
 *
 * @param parameters GPV 响应的 ParameterList
 * @param targetObject 目标对象路径，必须以 "." 结尾（如 "Device.Foo."）
 * @returns 实例索引数组（去重 + 升序）；非法 input 或无匹配 → []
 */
export function parseGpvInstances(
  parameters: ReadonlyArray<GpvParameter>,
  targetObject: string,
): number[] {
  if (!targetObject || !targetObject.endsWith('.')) {
    return [];
  }
  if (!Array.isArray(parameters) || parameters.length === 0) {
    return [];
  }

  // 转义 targetObject 中的正则元字符（. 是高频字符必须转义）。
  // 匹配模式: `^<targetObject>(\d+)\.` — 紧跟 targetObject 的数字段后必须有 "."
  // （确保不会误抓非实例标量）
  const escaped = targetObject.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const pattern = new RegExp(`^${escaped}(\\d+)\\.`);

  const indices = new Set<number>();
  for (const p of parameters) {
    if (!p || typeof p.name !== 'string') continue;
    const match = pattern.exec(p.name);
    if (!match) continue;
    const idx = Number.parseInt(match[1], 10);
    if (Number.isFinite(idx) && idx >= 0) {
      indices.add(idx);
    }
  }

  return Array.from(indices).sort((a, b) => a - b);
}
