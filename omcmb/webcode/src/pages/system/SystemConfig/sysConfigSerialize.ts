import type {
  BatchUpdateSysConfigItem,
  SysConfigItem,
  SysConfigValueType,
} from '@core/types/system';

/**
 * 系统配置表单值 → sys_configs 批量 upsert 的纯序列化逻辑（抽出便于单测）。
 *
 * sys_configs.value 列是 TEXT；DDL CHECK value_type IN ('string','int','float','bool','json')。
 * 后端按 value_type 解析（如设备离线阈值 enbTimeout/cpeTimeout 必须落 'int'，否则
 * 状态调和读取阈值会失败）—— 这里锁住「整数 → int」的契约（见 #203）。
 */
export function encodeValue(v: unknown): { value: string; valueType: SysConfigValueType } {
  if (v === null || v === undefined) return { value: '', valueType: 'string' };
  if (typeof v === 'boolean') return { value: v ? 'true' : 'false', valueType: 'bool' };
  if (typeof v === 'number') {
    return { value: String(v), valueType: Number.isInteger(v) ? 'int' : 'float' };
  }
  if (typeof v === 'object') {
    try {
      return { value: JSON.stringify(v), valueType: 'json' };
    } catch {
      return { value: '', valueType: 'string' };
    }
  }
  return { value: String(v), valueType: 'string' };
}

/**
 * 把 form 里所有字段（含未在 existing 中的新增 key）打包成批量 upsert items。
 * value_type 优先沿用后端已记录的 valueType，否则按 JS 运行期类型推断。
 */
export function buildBatchItems(
  formValues: Record<string, unknown>,
  existing: SysConfigItem[] | undefined,
): BatchUpdateSysConfigItem[] {
  const existingMap = new Map<string, SysConfigItem>();
  for (const it of existing || []) existingMap.set(it.key, it);

  const items: BatchUpdateSysConfigItem[] = [];
  for (const [key, raw] of Object.entries(formValues)) {
    const existingItem = existingMap.get(key);
    const isWriteOnlySecret = existingItem?.isSecret || key === 'defaultPasswd';
    if (isWriteOnlySecret && String(raw ?? '').trim() === '') continue;
    const enc = encodeValue(raw);
    const dbType = existingItem?.valueType;
    items.push({
      key,
      value: enc.value,
      // 已存在的 key：保留 DB 中的 value_type；新 key：用编码推断的 type。
      value_type: dbType ?? enc.valueType,
    });
  }
  return items;
}
