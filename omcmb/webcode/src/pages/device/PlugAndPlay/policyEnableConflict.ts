export interface EnablePolicyCandidate {
  policyId: string;
  productNames: string[];
  enabled: boolean;
}

export function findEnabledPolicyProductConflict<T extends EnablePolicyCandidate>(
  candidate: T,
  policies: readonly T[],
): T | undefined {
  const candidateProducts = new Set(
    candidate.productNames.map((name) => name.trim().toLowerCase()).filter(Boolean),
  );
  return policies.find((policy) => (
    policy.policyId !== candidate.policyId
    && policy.enabled
    && policy.productNames.some((name) => candidateProducts.has(name.trim().toLowerCase()))
  ));
}
