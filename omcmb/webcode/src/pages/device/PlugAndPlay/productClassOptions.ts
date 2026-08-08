export interface ProductClassOption {
  label: string;
  value: string;
}

interface ProductClassCatalogItem {
  name?: string;
  tech?: string;
  patterns?: readonly string[];
}

export type ProductTechnology = 'lte' | 'nr' | 'gsm';

export function normalizeProductTechnology(
  technology: string | undefined,
): ProductTechnology | undefined {
  const normalized = technology?.trim().toLowerCase();
  if (!normalized) return undefined;
  if (normalized === 'lte' || normalized === 'enb' || normalized.includes('4g')) return 'lte';
  if (normalized === 'nr' || normalized === 'gnb' || normalized.includes('5g')) return 'nr';
  if (normalized === 'gsm' || normalized.includes('2g')) return 'gsm';
  return undefined;
}

function matchesProductClass(pattern: string, productClass: string): boolean {
  try {
    return new RegExp(pattern).test(productClass);
  } catch {
    return pattern === productClass;
  }
}

export function resolveProductClassTechnology(
  productClass: string,
  catalogProducts: readonly ProductClassCatalogItem[] | undefined,
): ProductTechnology | undefined {
  const matchedProduct = (catalogProducts ?? []).find((product) =>
    (product.patterns ?? []).some((pattern) => matchesProductClass(pattern, productClass)),
  );
  return normalizeProductTechnology(matchedProduct?.tech);
}

function toLiteralProductClass(pattern: string): string | undefined {
  let value = pattern.trim();
  if (value.startsWith('^')) value = value.slice(1);
  if (value.endsWith('$')) value = value.slice(0, -1);

  // 通配正则用于匹配设备，不是可保存到策略中的具体产品类型。
  if (!value || /[\\^$.*+?()[\]{}|]/.test(value)) return undefined;
  return value;
}

export function toSupportedProductClassOptions(
  productClasses: readonly string[] | undefined,
  catalogProducts: readonly ProductClassCatalogItem[] | undefined = [],
  technology?: ProductTechnology,
): ProductClassOption[] {
  const observedClasses = (productClasses ?? [])
    .map((productClass) => productClass.trim())
    .filter(Boolean)
    .filter((productClass) => !technology || resolveProductClassTechnology(productClass, catalogProducts) === technology);
  const technologyProducts = technology
    ? (catalogProducts ?? []).filter(
      (product) => normalizeProductTechnology(product.tech) === technology,
    )
    : catalogProducts ?? [];
  const catalogClasses = technologyProducts
    .flatMap((product) => product.patterns ?? [])
    .map(toLiteralProductClass)
    .filter((productClass): productClass is string => Boolean(productClass));

  return Array.from(new Set([...observedClasses, ...catalogClasses]))
    .sort((left, right) => left.localeCompare(right))
    .map((productClass) => ({ label: productClass, value: productClass }));
}

export function toSupportedProductNameOptions(
  catalogProducts: readonly ProductClassCatalogItem[] | undefined = [],
  technology?: ProductTechnology,
): ProductClassOption[] {
  const products = (catalogProducts ?? []).filter(
    (product) => Boolean(product.name) && (!technology || normalizeProductTechnology(product.tech) === technology),
  );
  const names = products.map((product) => product.name!.trim()).filter(Boolean);

  return Array.from(new Set(names))
    .sort((left, right) => left.localeCompare(right))
    .map((name) => ({ label: name, value: name }));
}

export function resolveProductClassForName(
  productName: string,
  catalogProducts: readonly ProductClassCatalogItem[] | undefined,
): string {
  const product = (catalogProducts ?? []).find(
    (item) => item.name?.trim().toLowerCase() === productName.trim().toLowerCase(),
  );
  const literalPattern = product?.patterns?.map(toLiteralProductClass).find(Boolean);
  return literalPattern || productName.trim();
}
