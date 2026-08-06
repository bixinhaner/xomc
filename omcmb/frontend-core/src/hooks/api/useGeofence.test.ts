import { describe, expect, it, vi } from 'vitest';
import type {
  GeofenceBindingFilter,
  GeofenceMapFilter,
} from '../../types/geofence';
import {
  geofenceJobRefetchInterval,
  geofenceKeys,
  invalidateGeofenceBindingCaches,
  invalidateGeofenceCompletedJobCaches,
  invalidateGeofenceDefinitionCaches,
  invalidateGeofenceSettingsCaches,
} from './useGeofence';

describe('geofence query contract', () => {
  it('builds stable domain keys and includes map filters', () => {
    const filter: GeofenceMapFilter = {
      bounds: {
        minLng: 120,
        maxLng: 121,
        minLat: 30,
        maxLat: 31,
      },
      carrier: 'cmcc',
      status: 'enabled',
    };
    const bindingFilter: GeofenceBindingFilter = {
      status: 'active',
      page: 2,
      pageSize: 20,
    };

    expect(geofenceKeys.settings()).toEqual([
      'geofence',
      'settings',
    ]);
    expect(geofenceKeys.availability()).toEqual([
      'geofence',
      'availability',
    ]);
    expect(geofenceKeys.map(filter)).toEqual([
      'geofence',
      'map',
      filter,
    ]);
    expect(geofenceKeys.detail('geofence-1')).toEqual([
      'geofence',
      'definitions',
      'detail',
      'geofence-1',
    ]);
    expect(geofenceKeys.versions('geofence-1')).toEqual([
      'geofence',
      'definitions',
      'detail',
      'geofence-1',
      'versions',
    ]);
    expect(
      geofenceKeys.bindings('geofence-1', bindingFilter),
    ).toEqual([
      'geofence',
      'definitions',
      'detail',
      'geofence-1',
      'bindings',
      bindingFilter,
    ]);
    expect(geofenceKeys.job('job-1')).toEqual([
      'geofence',
      'jobs',
      'detail',
      'job-1',
    ]);
    expect(geofenceKeys.jobItemsRoot('job-1')).toEqual([
      'geofence',
      'jobs',
      'detail',
      'job-1',
      'items',
    ]);
  });

  it.each(['pending', 'running', 'zombie'] as const)(
    'continues polling for %s jobs',
    (status) => {
      expect(geofenceJobRefetchInterval(status)).toBe(2000);
      expect(geofenceJobRefetchInterval(status, 500)).toBe(500);
    },
  );

  it.each(['succeeded', 'failed', 'canceled'] as const)(
    'stops polling for terminal %s jobs',
    (status) => {
      expect(geofenceJobRefetchInterval(status)).toBe(false);
    },
  );

  it('stops polling when no job is available', () => {
    expect(geofenceJobRefetchInterval(undefined)).toBe(false);
  });
});

describe('geofence cache invalidation contract', () => {
  it('invalidates settings, definitions and map after settings update', async () => {
    const invalidateQueries = vi.fn().mockResolvedValue(undefined);

    await invalidateGeofenceSettingsCaches({ invalidateQueries });

    expect(invalidateQueries.mock.calls).toEqual([
      [{ queryKey: geofenceKeys.settings() }],
      [{ queryKey: geofenceKeys.availability() }],
      [{ queryKey: geofenceKeys.definitions() }],
      [{ queryKey: geofenceKeys.maps() }],
    ]);
  });

  it('invalidates one definition, its versions and the map after definition writes', async () => {
    const invalidateQueries = vi.fn().mockResolvedValue(undefined);

    await invalidateGeofenceDefinitionCaches(
      { invalidateQueries },
      'geofence-1',
    );

    expect(invalidateQueries.mock.calls).toEqual([
      [{ queryKey: geofenceKeys.definitions() }],
      [{ queryKey: geofenceKeys.detail('geofence-1') }],
      [{ queryKey: geofenceKeys.versions('geofence-1') }],
      [{ queryKey: geofenceKeys.maps() }],
    ]);
  });

  it('invalidates binding pages and the parent map when a binding job reaches terminal state', async () => {
    const invalidateQueries = vi.fn().mockResolvedValue(undefined);

    await invalidateGeofenceBindingCaches(
      { invalidateQueries },
      'geofence-1',
    );

    expect(invalidateQueries.mock.calls).toEqual([
      [
        {
          queryKey:
            geofenceKeys.bindingLists('geofence-1'),
        },
      ],
      [{ queryKey: geofenceKeys.maps() }],
    ]);
  });

  it('also invalidates job items when a binding job reaches terminal state', async () => {
    const invalidateQueries = vi.fn().mockResolvedValue(undefined);

    await invalidateGeofenceCompletedJobCaches(
      { invalidateQueries },
      'job-1',
      'geofence-1',
    );

    expect(invalidateQueries.mock.calls).toEqual([
      [
        {
          queryKey:
            geofenceKeys.bindingLists('geofence-1'),
        },
      ],
      [{ queryKey: geofenceKeys.maps() }],
      [{ queryKey: geofenceKeys.jobItemsRoot('job-1') }],
    ]);
  });
});
