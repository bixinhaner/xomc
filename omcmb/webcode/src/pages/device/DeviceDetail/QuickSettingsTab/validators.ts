import type { ParameterType, ParameterConstraints } from '@core/types/deviceParameter';

interface EnumMeta {
  values: string[];
  labels: string[];
}

type QuickSettingsLocale = 'zh-CN' | 'en-US';

export const LTE_BANDWIDTH_PATH = 'Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth';
export const BM_RU_RF_SWITCH_PATH = 'Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable';

const LTE_BANDWIDTH_ENUM: EnumMeta = {
  values: ['25', '50', '75', '100'],
  labels: ['CELL_BW_25(5M)', 'CELL_BW_50(10M)', 'CELL_BW_75(15M)', 'CELL_BW_100(20M)'],
};

const BM_RU_RF_SWITCH_ENUM: EnumMeta = {
  values: ['1', '0'],
  labels: ['开', '关'],
};

function resolveEnumFallbackByPath(path?: string): EnumMeta | null {
  if (!path) return null;
  // 仅 DLBandwidth 在 quicksettings XML 中保留为可编辑字段；ULBandwidth 已下线。
  if (path.endsWith('.CellConfig.LTE.RAN.RF.DLBandwidth')) return LTE_BANDWIDTH_ENUM;
  if (path.endsWith('.CellConfig.LTE.RAN.RF.X_COM_RadioEnable')) return BM_RU_RF_SWITCH_ENUM;
  return null;
}

export function getEffectiveEnumMeta(constraints?: ParameterConstraints, path?: string): EnumMeta | null {
  if (constraints?.enumValues && constraints.enumValues.length > 0) {
    return {
      values: constraints.enumValues,
      labels: constraints.enumLabels && constraints.enumLabels.length > 0
        ? constraints.enumLabels
        : constraints.enumValues,
    };
  }
  return resolveEnumFallbackByPath(path);
}

export function localizeEnumLabel(label: string, value: string, locale: QuickSettingsLocale): string {
  if (locale === 'zh-CN') return label || value;
  const normalized = String(label || '').trim();
  if (normalized === '开' || normalized === '开启') return 'ON';
  if (normalized === '关' || normalized === '关闭') return 'OFF';
  return label || value;
}

export function formatEnumDisplayValue(
  value: string,
  constraints?: ParameterConstraints,
  path?: string,
  locale: QuickSettingsLocale = 'zh-CN',
): string {
  const meta = getEffectiveEnumMeta(constraints, path);
  if (!meta || !value) return value;
  const index = meta.values.indexOf(value);
  return index >= 0 ? localizeEnumLabel(meta.labels[index] || value, value, locale) : value;
}

const LTE_BANDWIDTH_DISPLAY: Record<string, string> = {
  '25': '5M',
  '50': '10M',
  '75': '15M',
  '100': '20M',
};

export function formatLteBandwidthDisplay(value?: string | number | null): string {
  if (!value) return '-';
  const raw = String(value).trim();
  if (!raw) return '-';
  if (LTE_BANDWIDTH_DISPLAY[raw]) return LTE_BANDWIDTH_DISPLAY[raw];

  const normalized = raw.toLowerCase().replace(/mhz$/, '').replace(/m$/, '').trim();
  const numericValue = Number(normalized);
  if (Number.isFinite(numericValue)) {
    return `${Number.isInteger(numericValue) ? numericValue : numericValue.toFixed(1)}M`;
  }

  return formatEnumDisplayValue(raw, undefined, LTE_BANDWIDTH_PATH);
}

export function validateMmeIpPlmnLimit(
  rows: Array<{ mmeIp?: string | null; plmn?: string | null }>,
  max?: number,
): string | null {
  if (max === undefined || max <= 0) return null;
  const count = rows.filter((row) => String(row.mmeIp ?? '').trim() || String(row.plmn ?? '').trim()).length;
  return count > max ? `最多支持 ${max} 个 MME` : null;
}

function isValidIpv4(value: string): boolean {
  const parts = value.split('.');
  if (parts.length !== 4) return false;
  return parts.every((part) => {
    if (!/^\d+$/.test(part)) return false;
    if (part.length > 1 && part.startsWith('0')) return false;
    const num = Number(part);
    return Number.isInteger(num) && num >= 0 && num <= 255;
  });
}

export function validateMmeIp(value: string): string | null {
  const trimmed = value.trim();
  if (!trimmed) return null;
  return isValidIpv4(trimmed) ? null : 'MME IP 必须为合法 IPv4 地址';
}

export function validatePlmn(value: string): string | null {
  const trimmed = value.trim();
  if (!trimmed) return null;
  return /^\d{5,6}$/.test(trimmed) ? null : 'PLMN 必须为 5-6 位数字';
}

export function validateMmeIpPlmnRows(
  rows: Array<{ mmeIp?: string | null; plmn?: string | null }>,
): string | null {
  for (let i = 0; i < rows.length; i += 1) {
    const mmeIp = String(rows[i]?.mmeIp ?? '').trim();
    const plmn = String(rows[i]?.plmn ?? '').trim();
    if (!mmeIp && !plmn) continue;
    if (!mmeIp) return `第 ${i + 1} 行 MME IP 不能为空`;
    if (!plmn) return `第 ${i + 1} 行 PLMN 不能为空`;
    const ipErr = validateMmeIp(mmeIp);
    if (ipErr) return `第 ${i + 1} 行 ${ipErr}`;
    const plmnErr = validatePlmn(plmn);
    if (plmnErr) return `第 ${i + 1} 行 ${plmnErr}`;
  }
  return null;
}

export interface QuickSettingsInstanceContext {
  networkType: string;
  fapInstance: number;
  cellInstance?: number;
}

const DEVICE_LEVEL_QUICK_SETTINGS_GROUP_IDS = new Set([
  'device-time',
  'gnb-sync-source',
  'device-ipsec-control',
  'device-ipsec',
  'gnb-ipsec',
]);

export function getFeedbackScopeContext(
  groupId: string,
  context: QuickSettingsInstanceContext,
): QuickSettingsInstanceContext {
  if (!DEVICE_LEVEL_QUICK_SETTINGS_GROUP_IDS.has(groupId)) {
    return context;
  }
  return {
    networkType: context.networkType,
    fapInstance: 1,
  };
}

/**
 * 校验单个参数输入值。返回错误信息字符串或 null（表示通过）。
 *
 * 复制自 ParameterTreeTab/ParameterEditModal.tsx 的 validateValue（保持行为一致，避免双源漂移）。
 */
export function validateValue(
  value: string,
  parameterType: ParameterType,
  constraints?: ParameterConstraints,
): string | null {
  if (!value && parameterType !== 'string') {
    return '请输入值';
  }

  if (parameterType === 'int') {
    const num = Number(value);
    if (!Number.isInteger(num)) return '请输入整数';
    if (constraints?.minValue !== undefined && num < constraints.minValue) {
      return `最小值为 ${constraints.minValue}`;
    }
    if (constraints?.maxValue !== undefined && num > constraints.maxValue) {
      return `最大值为 ${constraints.maxValue}`;
    }
  }

  if (parameterType === 'unsignedInt') {
    const num = Number(value);
    if (!Number.isInteger(num) || num < 0) return '请输入非负整数';
    if (constraints?.minValue !== undefined && num < constraints.minValue) {
      return `最小值为 ${constraints.minValue}`;
    }
    if (constraints?.maxValue !== undefined && num > constraints.maxValue) {
      return `最大值为 ${constraints.maxValue}`;
    }
  }

  if (parameterType === 'string' && constraints) {
    // 后端 ParamMapping.MinValue/MaxValue 字段对 string 类型语义为"长度"（XML 同字段名
    // 复用，按 parameterType 解释）。前端兼容显式 maxLength/minLength 优先，未给时退回
    // 用 minValue/maxValue 当 length 边界。
    const maxLen = constraints.maxLength ?? constraints.maxValue;
    const minLen = constraints.minLength ?? constraints.minValue;
    if (maxLen !== undefined && value.length > maxLen) {
      return `最大长度为 ${maxLen}`;
    }
    if (minLen !== undefined && value.length < minLen) {
      return `最小长度为 ${minLen}`;
    }
    if (constraints.pattern) {
      try {
        const re = new RegExp(constraints.pattern);
        if (!re.test(value)) return `不匹配模式: ${constraints.pattern}`;
      } catch {
        // ignore invalid regex
      }
    }
  }

  if (constraints?.enumValues && constraints.enumValues.length > 0) {
    if (!constraints.enumValues.includes(value)) {
      return `允许的值: ${constraints.enumValues.join(', ')}`;
    }
  }

  return null;
}

export function validateLteQOffsetValue(value: string): string | null {
  const trimmed = value.trim();
  if (!trimmed) return null;
  const num = Number(trimmed);
  if (!Number.isInteger(num)) return '请输入整数';
  if (num < -24 || num > 24 || num % 2 !== 0) {
    return 'QOffset 仅支持 -24 到 24 的偶数';
  }
  return null;
}

export function parseQuickSettingsMultiCheckboxValue(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.map((item) => String(item ?? '').trim()).filter(Boolean);
  }
  return String(value ?? '')
    .split(/[,-]/)
    .map((s) => s.trim())
    .filter(Boolean);
}

export function serializeQuickSettingsMultiCheckboxValue(value: unknown): string {
  return parseQuickSettingsMultiCheckboxValue(value).join('-');
}

const BSC_CODEC_SUPPORT_OPTIONS = ['fr', 'hr', 'efr', 'amr'];

export function isBscCodecSupportParam(nameOrPath?: string | null): boolean {
  const raw = String(nameOrPath ?? '').trim();
  return raw === 'CodecSupport' || /^DeviceGSM\.Bts\.(?:\{i\}|\d+)\.CodecSupport$/.test(raw);
}

export function normalizeBscCodecSupportValue(value: unknown): string[] {
  const selected = new Set(parseQuickSettingsMultiCheckboxValue(value).map((item) => item.toLowerCase()));
  selected.add('fr');
  return BSC_CODEC_SUPPORT_OPTIONS.filter((item) => selected.has(item));
}

export function serializeBscCodecSupportValue(value: unknown): string {
  return normalizeBscCodecSupportValue(value).filter((item) => item !== 'fr').join('-');
}

function toParameterType(type?: string | null): ParameterType | null {
  const normalized = String(type ?? '').trim();
  if (normalized === 'multiCheckbox') {
    return 'string';
  }
  switch (normalized.toLowerCase()) {
    case 'string':
      return 'string';
    case 'int':
      return 'int';
    case 'unsignedint':
      return 'unsignedInt';
    case 'boolean':
      return 'boolean';
    case 'datetime':
      return 'dateTime';
    case 'base64':
      return 'base64';
    case 'hexbinary':
      return 'hexBinary';
    case 'object':
      return 'object';
    default:
      return null;
  }
}

export function resolveQuickSettingsParameterType(
  quickSettingsType?: string,
  schemaType?: string,
  rawType?: string,
): ParameterType {
  return toParameterType(quickSettingsType)
    ?? toParameterType(schemaType)
    ?? toParameterType(rawType)
    ?? 'string';
}

/**
 * 把设备上送的字符串(可能是 "true"/"false"、"True"/"FALSE"、"1"/"0")
 * 归一到 XML <option value="..."/> 的真实值,确保 Select 能选中正确项。
 * 仅在 enumOptions 非空时生效;不存在等价匹配时原样返回(保留原始值,Select 留空)。
 *
 * 同时用于保存前脏检查:表单初值与设备当前值(oldVal)必须经同一归一化后比较,
 * 否则设备上报 "true" 而选项值为 "1"/"0" 时,未修改也会判脏导致无效下发(issue #317)。
 */
export function normalizeEnumValue(raw: unknown, enumOptions?: { value: string; label: string }[]): string {
  const s = String(raw ?? '');
  if (!enumOptions || enumOptions.length === 0) return s;
  if (enumOptions.some((o) => o.value === s)) return s;
  const ci = s.toLowerCase();
  const direct = enumOptions.find((o) => o.value.toLowerCase() === ci);
  if (direct) return direct.value;
  const labelMatch = enumOptions.find((o) => o.label.toLowerCase() === ci);
  if (labelMatch) return labelMatch.value;
  // BOOLEAN 等价:true/1 与 false/0 互转,适配 TR-069 BOOLEAN 字段两种序列化。
  if ((ci === 'true' || ci === '1') && enumOptions.some((o) => o.value === '1')) return '1';
  if ((ci === 'false' || ci === '0') && enumOptions.some((o) => o.value === '0')) return '0';
  if ((ci === 'true' || ci === '1')) {
    const m = enumOptions.find((o) => o.value.toLowerCase() === 'true');
    if (m) return m.value;
  }
  if ((ci === 'false' || ci === '0')) {
    const m = enumOptions.find((o) => o.value.toLowerCase() === 'false');
    if (m) return m.value;
  }
  return s;
}

interface ApplyInstanceContextOptions {
  preserveTrailingInstance?: boolean;
}

/**
 * 根据当前网络制式把快速设置路径中的实例占位符解析成具体路径。
 *
 * - LTE: 仅替换最外层 FAPService.{i}
 * - NR: 先解析 FAPService / CellConfig 两层实例，再把更深层未区分的列表实例保守落到 1
 *
 * preserveTrailingInstance 用于多实例 objectPath，保留末尾那层 {i}. 给表格行实例继续拼接。
 */
export function applyInstanceContext(
  path: string,
  context: QuickSettingsInstanceContext,
  options?: ApplyInstanceContextOptions,
): string {
  const preserveTrailingInstance = options?.preserveTrailingInstance && path.endsWith('{i}.');
  const trailingPlaceholder = '__QS_TRAILING_INSTANCE__';

  let resolved = preserveTrailingInstance
    ? `${path.slice(0, -4)}${trailingPlaceholder}.`
    : path;

  if (context.networkType === 'nr') {
    const cellInstance = context.cellInstance ?? 1;
    resolved = resolved.replace('FAPService.{i}', `FAPService.${context.fapInstance}`);
    resolved = resolved.replace('CellConfig.{i}', `CellConfig.${cellInstance}`);
    resolved = resolved.replace(/\{i\}/g, '1');
  } else {
    resolved = resolved.replace('{i}', String(context.fapInstance));
  }

  if (preserveTrailingInstance) {
    resolved = resolved.replace(trailingPlaceholder, '{i}');
  }

  return resolved;
}
