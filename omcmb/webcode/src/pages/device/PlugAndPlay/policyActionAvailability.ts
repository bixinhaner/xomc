export interface PolicyActionAvailability {
  edit: boolean;
  detect: boolean;
  delete: boolean;
}

export function getPolicyActionAvailability(
  _policy: { enabled: boolean },
): PolicyActionAvailability {
  return {
    edit: true,
    detect: true,
    delete: true,
  };
}

export interface PolicyModuleActionAvailability {
  view: boolean;
  search: boolean;
  download: boolean;
  export: boolean;
  edit: boolean;
  delete: boolean;
  import: boolean;
}

export function getPolicyModuleActionAvailability(
  mode: 'view' | 'edit' | 'create',
): PolicyModuleActionAvailability {
  const canMutate = mode !== 'view';
  return {
    view: true,
    search: true,
    download: true,
    export: true,
    edit: canMutate,
    delete: canMutate,
    import: canMutate,
  };
}
