export type PlmnRow = {
  key: string;
  plmn: string;
};

export interface PlmnBatchState {
  existingRows: PlmnRow[];
  deletedKeys: ReadonlySet<string>;
  editedValues: ReadonlyMap<string, string>;
  addedRows: PlmnRow[];
}

function isPlmnRows(value: unknown): value is PlmnRow[] {
  return Array.isArray(value)
    && value.every((item) => item && typeof item === 'object' && 'plmn' in item);
}

export function toPlmnRows(value: unknown): PlmnRow[] {
  if (isPlmnRows(value)) {
    return value.map((row, index) => ({
      key: row.key || `plmn-${index}`,
      plmn: String(row.plmn ?? ''),
    }));
  }
  return parsePlmnList(value);
}

export function isPlmnRowLimitReached(rows: PlmnRow[], maxRows: number): boolean {
  return rows.length >= maxRows;
}

export function buildEffectivePlmnRows({
  existingRows,
  deletedKeys,
  editedValues,
  addedRows,
}: PlmnBatchState): PlmnRow[] {
  return [
    ...existingRows
      .filter((row) => !deletedKeys.has(row.key))
      .map((row) => ({
        ...row,
        plmn: editedValues.get(row.key) ?? row.plmn,
      })),
    ...addedRows,
  ];
}

export function parsePlmnList(raw: unknown): PlmnRow[] {
  const text = String(raw ?? '').trim();
  if (!text) return [];

  return text
    .split(/[;,\n]+/)
    .map((plmn) => plmn.trim())
    .filter(Boolean)
    .map((plmn, index) => ({
      key: `plmn-${index}-${plmn}`,
      plmn,
    }));
}

export function serializePlmnList(rows: PlmnRow[]): string {
  return rows
    .map((row) => String(row.plmn ?? '').trim())
    .filter(Boolean)
    .join(',');
}

export type PlmnListValidationError = 'format' | 'duplicate' | 'limit';

export function validatePlmnList(
  rows: PlmnRow[],
  maxRows: number,
): PlmnListValidationError | null {
  const values = rows
    .map((row) => String(row.plmn ?? '').trim())
    .filter(Boolean);
  if (values.length > maxRows) {
    return 'limit';
  }
  if (values.some((plmn) => !/^\d{5,6}$/.test(plmn))) {
    return 'format';
  }
  if (new Set(values).size !== values.length) {
    return 'duplicate';
  }
  return null;
}
