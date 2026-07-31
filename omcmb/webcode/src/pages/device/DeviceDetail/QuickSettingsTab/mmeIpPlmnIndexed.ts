import type { ParameterUpdateRequest } from '@core/types/deviceParameter';

export const MLN_MME_POOL_MAX_ROWS = 16;
export const MLN_MME_POOL_STANDARD_PREFIX =
  'Device.Services.FAPService.CellConfig.LTE.MmePoolConfigParam.';

const MLN_MME_POOL_PRIVATE_PREFIX =
  'Device.Services.FAPService.MmePoolConfigParam.';
const EMPTY_MME_IP = '0.0.0.0';
const EMPTY_PLMN = '000000';

export interface MmePoolValue {
  parameterPath: string;
  parameterValue: string;
}

export interface MmePoolRow {
  key: string;
  mmeIp: string;
  plmn: string;
}

export function isMlnIndexedMmePoolModel(paramModel: string | undefined): boolean {
  return String(paramModel ?? '').trim().toUpperCase() === 'MLN';
}

function standardMmeIpPath(index: number): string {
  return `${MLN_MME_POOL_STANDARD_PREFIX}${index}.MMEIp1`;
}

function standardPlmnPath(index: number): string {
  return `${MLN_MME_POOL_STANDARD_PREFIX}${index}.PLMNID`;
}

function privateMmeIpPath(index: number): string {
  return `${MLN_MME_POOL_PRIVATE_PREFIX}${index}.MMEIp`;
}

function privatePlmnPath(index: number): string {
  return `${MLN_MME_POOL_PRIVATE_PREFIX}${index}.PLMNID`;
}

function valueMap(values: readonly MmePoolValue[]): Map<string, string> {
  return new Map(values.map((item) => [
    item.parameterPath,
    String(item.parameterValue ?? '').trim(),
  ]));
}

function indexedValue(
  values: ReadonlyMap<string, string>,
  standardPath: string,
  privatePath: string,
): string | undefined {
  return values.get(standardPath) ?? values.get(privatePath);
}

function isUnusedSlot(mmeIp: string, plmn: string): boolean {
  return (!mmeIp || mmeIp === EMPTY_MME_IP) && (!plmn || plmn === EMPTY_PLMN);
}

export function parseMlnMmePoolRows(values: readonly MmePoolValue[]): MmePoolRow[] {
  const current = valueMap(values);
  const rows: MmePoolRow[] = [];

  for (let index = 1; index <= MLN_MME_POOL_MAX_ROWS; index += 1) {
    const mmeIp = indexedValue(current, standardMmeIpPath(index), privateMmeIpPath(index)) ?? '';
    const plmn = indexedValue(current, standardPlmnPath(index), privatePlmnPath(index)) ?? '';
    if (isUnusedSlot(mmeIp, plmn)) continue;
    rows.push({
      key: `mme-pool-${index}`,
      mmeIp,
      plmn,
    });
  }

  return rows;
}

export function buildMlnMmePoolUpdates(
  rows: readonly MmePoolRow[],
  currentValues: readonly MmePoolValue[],
): ParameterUpdateRequest[] {
  const current = valueMap(currentValues);
  const normalizedRows = rows
    .map((row) => ({
      key: row.key,
      mmeIp: String(row.mmeIp ?? '').trim(),
      plmn: String(row.plmn ?? '').trim(),
    }))
    .filter((row) => row.mmeIp || row.plmn)
    .slice(0, MLN_MME_POOL_MAX_ROWS);
  const updates: ParameterUpdateRequest[] = [];

  for (let index = 1; index <= MLN_MME_POOL_MAX_ROWS; index += 1) {
    const row = normalizedRows[index - 1];
    const nextMmeIp = row?.mmeIp || EMPTY_MME_IP;
    const nextPlmn = row?.plmn || EMPTY_PLMN;
    const mmeIpPath = standardMmeIpPath(index);
    const plmnPath = standardPlmnPath(index);
    const currentMmeIp = indexedValue(current, mmeIpPath, privateMmeIpPath(index));
    const currentPlmn = indexedValue(current, plmnPath, privatePlmnPath(index));

    if (currentMmeIp !== nextMmeIp) {
      updates.push({
        parameterPath: mmeIpPath,
        parameterValue: nextMmeIp,
        parameterType: 'string',
      });
    }
    if (currentPlmn !== nextPlmn) {
      updates.push({
        parameterPath: plmnPath,
        parameterValue: nextPlmn,
        parameterType: 'string',
      });
    }
  }

  return updates;
}

export function getMlnMmePoolSyncPaths(): string[] {
  return Array.from({ length: MLN_MME_POOL_MAX_ROWS }, (_, offset) => {
    const index = offset + 1;
    return [standardMmeIpPath(index), standardPlmnPath(index)];
  }).flat();
}
