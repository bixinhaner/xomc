/**
 * mmlResultParser — 把 device_tasks.result (ACS 写入的 {method, raw_response, ...})
 * 解析为终端可读的"每 path 一行"结构化结果。
 *
 * 用户决策（2026-05-26）：终端输出保留整段原始 response body（便于排查协议层问题），
 * 但额外在前面插入逐 path 解析结果，提升日常可读性 —— 用户不必去 XML 里找
 * <Name>/<Value> 对，直接读 "Device.X.Y = 'foo' (string)"。
 *
 * 适用方法：
 *   - GetParameterValuesResponse → params: [{name, value, type}]（LST 主战场）
 *   - SetParameterValuesResponse → status: 0=立即生效 1=需重启
 *   - AddObjectResponse → instanceNumber + status
 *   - DeleteObjectResponse / RebootResponse / FactoryResetResponse → status
 *
 * 非上述方法或 raw_response 缺失 → 返 null，调用方静默跳过，仍 fallback 到 raw JSON。
 *
 * 解析方式：浏览器 DOMParser（jsdom 单测环境也支持），XML 命名空间用 querySelectorAll
 * 的 local-name 模糊匹配，兼容厂商不同 ns 前缀（cwmp:/ SOAP-ENV: 等）。
 */

export type MmlResultKind = 'gpv' | 'spv' | 'add' | 'delete' | 'reboot' | 'unknown';

export interface ParsedParamValue {
  name: string;
  value: string;
  /** XSI 类型，如 "xsd:string" / "xsd:int" / "xsd:boolean"；可空。 */
  type?: string;
}

export interface ParsedMmlResult {
  kind: MmlResultKind;
  /** GPV: 解析出的 path/值 对 */
  params?: ParsedParamValue[];
  /** SPV / Add / Delete / Reboot: <Status> 值。0=成功立即生效；1=成功需重启。 */
  status?: number;
  /** AddObject: <InstanceNumber> 值 */
  instanceNumber?: number;
}

const MAX_DOM_PARSE_XML_CHARS = 500_000;

export interface ParseMmlResultOptions {
  /** Limit parsed GPV params for summary/table views. Omitted means keep full parser semantics. */
  maxParams?: number;
  /** Optional inclusion predicate applied before maxParams. */
  includeParam?: (param: ParsedParamValue) => boolean;
}

/** ACS 写入 device_tasks.result 的 shape（与 handler.go resultMap 保持一致）。 */
interface DeviceTaskResultEnvelope {
  method?: string;
  raw_response?: string;
  instance_number?: number;
  /**
   * issue #424：ACS 已把 GPV 响应里的私有 path 回译为标准 path 的 (name=标准path, value, type) 列表。
   * 存在时优先用它（按"执行的标准 path"匹配结果），缺失时回退解析 raw_response（私有 path）。
   */
  standard_parameter_values?: ParsedParamValue[];
}

/**
 * 解析 SSE frame.result。返回 null 表示当前结果非 RPC 响应形态（或缺
 * raw_response），调用方应回退到原始 JSON dump。
 */
export function parseMmlDeviceTaskResult(result: unknown, options: ParseMmlResultOptions = {}): ParsedMmlResult | null {
  if (!result || typeof result !== 'object') return null;
  const r = result as DeviceTaskResultEnvelope;
  if (!r.raw_response || typeof r.raw_response !== 'string') return null;

  const method = r.method ?? '';
  const xml = r.raw_response;

  if (method.includes('GetParameterValuesResponse')) {
    // issue #424：优先用 ACS 回译后的标准 path（standard_parameter_values）；缺失才回退
    // 解析 raw_response（私有 path）。前者让结果按"执行的标准 path"匹配，不再因私有/标准
    // 不一致而显示为空。
    const std = r.standard_parameter_values;
    if (Array.isArray(std) && std.length > 0) {
      const params = std
        .filter((p) => p && typeof p.name === 'string' && p.name)
        .map((p) => ({ name: p.name, value: String(p.value ?? ''), type: p.type }))
        .filter((p) => options.includeParam?.(p) ?? true)
        .slice(0, options.maxParams);
      if (params.length > 0) return { kind: 'gpv', params };
    }
    return { kind: 'gpv', params: parseGPVResponse(xml, options) };
  }
  if (method.includes('SetParameterValuesResponse')) {
    return { kind: 'spv', status: parseStatusFromResponse(xml) };
  }
  if (method.includes('AddObjectResponse')) {
    return {
      kind: 'add',
      // backend 已在 handler.go AddObject 分支预解析 InstanceNumber 写入 result.instance_number；
      // 缺则回退 XML 抽取。两路兜底确保不丢。
      instanceNumber: r.instance_number ?? parseInstanceFromResponse(xml),
      status: parseStatusFromResponse(xml),
    };
  }
  if (method.includes('DeleteObjectResponse')) {
    return { kind: 'delete', status: parseStatusFromResponse(xml) };
  }
  if (method.includes('RebootResponse') || method.includes('FactoryResetResponse')) {
    // Reboot / FactoryReset 响应是空 body（cwmp:RebootResponse 无子元素），收到即视为接受。
    return { kind: 'reboot', status: 0 };
  }
  return null;
}

/**
 * 提取 GetParameterValuesResponse 里所有 ParameterValueStruct。
 * 用 local-name 匹配以避开 cwmp: / SOAP-ENV: 等不同命名空间前缀。
 */
function parseGPVResponse(xml: string, options: ParseMmlResultOptions = {}): ParsedParamValue[] {
  if (xml.length > MAX_DOM_PARSE_XML_CHARS) {
    return parseGPVResponseLight(xml, options);
  }
  if (typeof DOMParser === 'undefined') return [];
  let doc: Document;
  try {
    doc = new DOMParser().parseFromString(xml, 'text/xml');
  } catch {
    return [];
  }
  // 浏览器 DOMParser 失败时 parse error 节点会插入 documentElement
  if (doc.getElementsByTagName('parsererror').length > 0) {
    return [];
  }
  const structs = findByLocalName(doc.documentElement, 'ParameterValueStruct');
  const out: ParsedParamValue[] = [];
  for (const s of structs) {
    const nameEl = firstChildByLocalName(s, 'Name');
    const valEl = firstChildByLocalName(s, 'Value');
    const name = (nameEl?.textContent ?? '').trim();
    if (!name) continue;
    const value = (valEl?.textContent ?? '').trim();
    const type = valEl ? readTypeAttr(valEl) : undefined;
    const param = { name, value, type };
    if (!(options.includeParam?.(param) ?? true)) continue;
    out.push(param);
    if (options.maxParams && out.length >= options.maxParams) break;
  }
  return out;
}

function parseGPVResponseLight(xml: string, options: ParseMmlResultOptions = {}): ParsedParamValue[] {
  const out: ParsedParamValue[] = [];
  const structRe = /<(?:[\w-]+:)?ParameterValueStruct\b[^>]*>([\s\S]*?)<\/(?:[\w-]+:)?ParameterValueStruct>/g;
  let match: RegExpExecArray | null;
  while ((match = structRe.exec(xml))) {
    const block = match[1];
    const name = readXmlTagText(block, 'Name');
    if (!name) continue;
    const value = readXmlTagText(block, 'Value') ?? '';
    const type = readValueType(block);
    const param = { name, value, type };
    if (!(options.includeParam?.(param) ?? true)) continue;
    out.push(param);
    if (options.maxParams && out.length >= options.maxParams) break;
  }
  return out;
}

function readXmlTagText(block: string, tag: string): string | undefined {
  const re = new RegExp(`<(?:[\\w-]+:)?${tag}\\b[^>]*>([\\s\\S]*?)<\\/(?:[\\w-]+:)?${tag}>`);
  const m = block.match(re);
  return m ? decodeXmlText(m[1].trim()) : undefined;
}

function readValueType(block: string): string | undefined {
  const m = block.match(/<(?:[\w-]+:)?Value\b([^>]*)>/);
  const attrs = m?.[1] ?? '';
  const type = attrs.match(/\b(?:xsi:)?type=["']([^"']+)["']/);
  return type?.[1];
}

function decodeXmlText(value: string): string {
  return value
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&apos;/g, "'")
    .replace(/&amp;/g, '&');
}

/** 从 xsi:type / type 属性里读取类型（xsi 命名空间不同实现）。 */
function readTypeAttr(el: Element): string | undefined {
  const t = el.getAttribute('xsi:type') ?? el.getAttribute('type');
  return t ? t : undefined;
}

/** 从 XML 文本里抽 <Status>N</Status>（无命名空间，cwmp 标准未带 ns）。 */
function parseStatusFromResponse(xml: string): number | undefined {
  const m = xml.match(/<(?:[a-zA-Z][\w-]*:)?Status>\s*(\d+)\s*<\//);
  return m ? Number(m[1]) : undefined;
}

function parseInstanceFromResponse(xml: string): number | undefined {
  const m = xml.match(/<(?:[a-zA-Z][\w-]*:)?InstanceNumber>\s*(\d+)\s*<\//);
  return m ? Number(m[1]) : undefined;
}

/** 在 root 子树里递归找所有 localName 等于 target 的元素（忽略命名空间前缀）。 */
function findByLocalName(root: Element | null, target: string): Element[] {
  if (!root) return [];
  const out: Element[] = [];
  const walk = (el: Element) => {
    if (el.localName === target) out.push(el);
    for (const child of Array.from(el.children)) walk(child);
  };
  walk(root);
  return out;
}

/** 在 parent 直接子节点里找第一个 localName 等于 target 的元素。 */
function firstChildByLocalName(parent: Element, target: string): Element | null {
  for (const child of Array.from(parent.children)) {
    if (child.localName === target) return child;
  }
  return null;
}
