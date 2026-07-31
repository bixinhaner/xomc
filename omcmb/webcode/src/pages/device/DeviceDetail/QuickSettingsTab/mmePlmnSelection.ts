const SERVING_PLMN_RESTRICTED_MODELS = new Set(['MLN', 'BLQ', 'BM']);

export interface MmePlmnRowLike {
  mmeIp?: string | null;
  plmn?: string | null;
}

export interface MmePlmnMembershipError {
  row: number;
  plmn: string;
}

export interface ServingPlmnParameterLike {
  parameterPath: string;
  parameterValue?: unknown;
}

export function isServingPlmnRestrictedModel(paramModel: string | undefined): boolean {
  return SERVING_PLMN_RESTRICTED_MODELS.has(
    String(paramModel ?? '').trim().toUpperCase(),
  );
}

export function getServingPlmnOptions(raw: unknown): string[] {
  const values = String(raw ?? '')
    .split(/[;,\n]+/)
    .map((value) => value.trim())
    .filter(Boolean);
  return Array.from(new Set(values));
}

export function getServingPlmnSearchQuery(paramModel: string | undefined): string {
  return String(paramModel ?? '').trim().toUpperCase() === 'BLQ'
    ? 'PLMNList'
    : 'ExistPlmnidList';
}

export function getServingPlmnOptionsForModel(
  paramModel: string | undefined,
  fapInstance: number,
  parameters: readonly ServingPlmnParameterLike[],
): string[] {
  const model = String(paramModel ?? '').trim().toUpperCase();
  if (model === 'BLQ') {
    const pathPattern = new RegExp(
      `^Device\\.Services\\.FAPService\\.${fapInstance}\\.CellConfig\\.LTE\\.EPC\\.PLMNList\\.\\d+\\.PLMNID$`,
    );
    return Array.from(new Set(
      parameters
        .filter((item) => pathPattern.test(item.parameterPath))
        .map((item) => String(item.parameterValue ?? '').trim())
        .filter(Boolean),
    ));
  }

  const aggregatePath = `Device.Services.FAPService.${fapInstance}.FAPControl.LTE.Gateway.ExistPlmnidList`;
  const aggregate = parameters.find((item) => item.parameterPath === aggregatePath);
  return getServingPlmnOptions(aggregate?.parameterValue);
}

export function findMmePlmnOutsideServingList(
  rows: readonly MmePlmnRowLike[],
  servingPlmns: readonly string[],
): MmePlmnMembershipError | null {
  const allowed = new Set(servingPlmns.map((value) => String(value).trim()).filter(Boolean));
  for (let index = 0; index < rows.length; index += 1) {
    const plmn = String(rows[index]?.plmn ?? '').trim();
    if (plmn && !allowed.has(plmn)) {
      return {
        row: index + 1,
        plmn,
      };
    }
  }
  return null;
}
