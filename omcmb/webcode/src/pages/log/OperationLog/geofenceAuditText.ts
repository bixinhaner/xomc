type Translate = (key: string) => string;

const GEOFENCE_ACTION_KEYS: Record<string, string> = {
  config_geofence_create: 'geofence.audit.create',
  config_geofence_modify: 'geofence.audit.modify',
  config_geofence_publish: 'geofence.audit.publish',
  config_geofence_enable: 'geofence.audit.enable',
  config_geofence_disable: 'geofence.audit.disable',
  config_geofence_archive: 'geofence.audit.archive',
  config_geofence_settings_update: 'geofence.audit.settingsUpdate',
  config_geofence_bind: 'geofence.audit.bind',
  config_geofence_binding_suspend: 'geofence.audit.bindingSuspend',
  config_geofence_binding_resume: 'geofence.audit.bindingResume',
  config_geofence_binding_remove: 'geofence.audit.bindingRemove',
};

export function localizeAuditAction(action: string, t: Translate): string {
  const baseAction = action.endsWith('_failed')
    ? action.slice(0, -'_failed'.length)
    : action;
  const key = GEOFENCE_ACTION_KEYS[baseAction];
  return key ? t(key) : action || '-';
}

export function localizeAuditReason(reason: string, t: Translate): string {
  if (reason.includes('geofence has active batch jobs')) {
    return t('geofence.lifecycle.activeJobsBlockArchive');
  }
  if (reason.includes('geofence lifecycle preview is stale')) {
    return t('geofence.lifecycle.previewExpired');
  }
  if (reason.includes('geofence name already exists')) {
    return t('geofence.validation.nameDuplicate');
  }
  if (reason.includes('invalid geofence name')) {
    return t('geofence.validation.nameInvalid');
  }
  return reason;
}
