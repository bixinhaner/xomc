import { describe, expect, it } from 'vitest';
import { getPolicySubmitAvailability } from './policySubmitAvailability';

describe('getPolicySubmitAvailability', () => {
  it('blocks an edit submit until the persisted policy and product catalog are ready', () => {
    expect(getPolicySubmitAvailability({
      isEdit: true,
      policyLoading: true,
      hasPersistedPolicy: false,
      productCatalogLoading: false,
      submitting: false,
    })).toEqual({ disabled: true, loading: true });

    expect(getPolicySubmitAvailability({
      isEdit: true,
      policyLoading: false,
      hasPersistedPolicy: true,
      productCatalogLoading: true,
      submitting: false,
    })).toEqual({ disabled: true, loading: true });
  });

  it('allows one submit after edit dependencies are ready and blocks while submitting', () => {
    expect(getPolicySubmitAvailability({
      isEdit: true,
      policyLoading: false,
      hasPersistedPolicy: true,
      productCatalogLoading: false,
      submitting: false,
    })).toEqual({ disabled: false, loading: false });

    expect(getPolicySubmitAvailability({
      isEdit: true,
      policyLoading: false,
      hasPersistedPolicy: true,
      productCatalogLoading: false,
      submitting: true,
    })).toEqual({ disabled: true, loading: true });
  });
});
