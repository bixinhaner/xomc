interface PolicySubmitAvailabilityInput {
  isEdit: boolean;
  policyLoading: boolean;
  hasPersistedPolicy: boolean;
  productCatalogLoading: boolean;
  submitting: boolean;
}

export function getPolicySubmitAvailability({
  isEdit,
  policyLoading,
  hasPersistedPolicy,
  productCatalogLoading,
  submitting,
}: PolicySubmitAvailabilityInput) {
  const loading = submitting || productCatalogLoading || (isEdit && policyLoading);

  return {
    loading,
    disabled: loading || (isEdit && !hasPersistedPolicy),
  };
}
