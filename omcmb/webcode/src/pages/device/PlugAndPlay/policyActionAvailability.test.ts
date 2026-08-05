import { describe, expect, it } from 'vitest';
import {
  getPolicyActionAvailability,
  getPolicyModuleActionAvailability,
} from './policyActionAvailability';

describe('plug-and-play policy action availability', () => {
  it('allows editing and deleting an enabled policy', () => {
    expect(getPolicyActionAvailability({ enabled: true })).toEqual({
      edit: true,
      detect: true,
      delete: true,
    });
  });

  it('allows editing and deleting a disabled policy', () => {
    expect(getPolicyActionAvailability({ enabled: false })).toEqual({
      edit: true,
      detect: true,
      delete: true,
    });
  });
});

describe('plug-and-play policy module action availability', () => {
  it('keeps read-only actions available in the detail page', () => {
    expect(getPolicyModuleActionAvailability('view')).toEqual({
      view: true,
      search: true,
      download: true,
      export: true,
      edit: false,
      delete: false,
      import: false,
    });
  });

  it('allows module editing and deletion in the edit page', () => {
    expect(getPolicyModuleActionAvailability('edit')).toEqual({
      view: true,
      search: true,
      download: true,
      export: true,
      edit: true,
      delete: true,
      import: true,
    });
  });
});
