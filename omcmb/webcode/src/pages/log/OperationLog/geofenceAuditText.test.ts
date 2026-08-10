import { describe, expect, it } from 'vitest';
import {
  localizeAuditAction,
  localizeAuditReason,
} from './geofenceAuditText';

const t = (key: string) => `translated:${key}`;

describe('geofence audit text', () => {
  it('localizes successful and failed geofence action codes', () => {
    expect(localizeAuditAction('config_geofence_create', t)).toBe(
      'translated:geofence.audit.create',
    );
    expect(localizeAuditAction('config_geofence_archive_failed', t)).toBe(
      'translated:geofence.audit.archive',
    );
  });

  it('replaces the active-job backend error chain with a business message', () => {
    expect(
      localizeAuditReason(
        'transition geofence definition: geofence has active batch jobs: resource already exists',
        t,
      ),
    ).toBe('translated:geofence.lifecycle.activeJobsBlockArchive');
  });
});
