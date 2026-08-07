export type ParamConfigSheetRows = Record<string, Record<string, unknown>[]>;

const RETIRED_PARAM_CONFIG_FIELDS: Record<string, readonly string[]> = {
  DEVICE: ['Time Zone Term'],
  INTERFACE: ['OMC IP'],
  NETWORK: ['OMC IP'],
};

export function sanitizeRetiredParamConfigFields<T extends ParamConfigSheetRows>(
  sheets: T,
): T {
  return Object.fromEntries(Object.entries(sheets).map(([sheetName, rows]) => {
    const retired = RETIRED_PARAM_CONFIG_FIELDS[sheetName] ?? [];
    if (retired.length === 0) return [sheetName, rows.map((row) => ({ ...row }))];
    return [sheetName, rows.map((row) => {
      const sanitized = { ...row };
      retired.forEach((field) => delete sanitized[field]);
      return sanitized;
    })];
  })) as T;
}
