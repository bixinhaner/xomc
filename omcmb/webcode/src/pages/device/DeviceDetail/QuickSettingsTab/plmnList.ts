export interface PlmnRow {
  key: string;
  plmn: string;
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
