import { beforeEach, describe, expect, it } from 'vitest';
import {
  geofenceMockService,
  resetGeofenceMock,
} from '../geofenceMock';

beforeEach(() => {
  resetGeofenceMock();
});

describe('geofenceMockService', () => {
  it('publishes a newly created draft as an enabled fence', async () => {
    const created = await geofenceMockService.createDefinition({
      name: '本地流程围栏',
      carrier: 'cmcc',
      ruleType: 'polygon_allow_zone',
      geometry: {
        type: 'Polygon',
        coordinates: [
          [
            [120.1, 30.1],
            [120.2, 30.1],
            [120.2, 30.2],
            [120.1, 30.1],
          ],
        ],
      },
      policy: { exitAction: 'notify_only' },
    });

    await geofenceMockService.publishDraft(
      created.definition.id,
      created.draftVersion.id,
    );

    await expect(
      geofenceMockService.getDefinition(created.definition.id),
    ).resolves.toMatchObject({
      status: 'enabled',
      currentVersionId: created.draftVersion.id,
    });
  });

  it('provides deterministic polygon and baseline-radius map rules', async () => {
    const first = await geofenceMockService.listMapDefinitions({});
    const second = await geofenceMockService.listMapDefinitions({});

    expect(first).toEqual(second);
    expect(
      first.items.map(
        (item) => item.definition.ruleType,
      ),
    ).toEqual([
      'polygon_allow_zone',
      'baseline_radius',
    ]);
    expect(first.items[0].currentVersion?.geometry.type).toBe(
      'Polygon',
    );
    expect(first.items[1].currentVersion?.geometry.type).toBe(
      'Circle',
    );
  });

  it('includes active, suspended and missing-location binding scenarios', async () => {
    const page = await geofenceMockService.listBindings(
      'mock-geofence-polygon',
      { page: 1, pageSize: 50 },
    );

    expect(page.items.map((item) => item.status)).toEqual([
      'active',
      'suspended',
    ]);
    expect(page.items.some((item) => !item.hasLocation)).toBe(
      true,
    );
    expect(page.items[0]).not.toHaveProperty('latitude');
    expect(page.items[0]).not.toHaveProperty('longitude');
  });

  it('requires the current manual-binding preview fingerprint', async () => {
    const inputs = { deviceSNs: ['MOCK-SN-003', 'UNKNOWN'] };
    const preview =
      await geofenceMockService.previewManualBindings(
        'mock-geofence-polygon',
        inputs,
      );

    await expect(
      geofenceMockService.createManualBindJob(
        'mock-geofence-polygon',
        {
          ...inputs,
          previewFingerprint: 'stale',
          reason: '批量纳管',
        },
      ),
    ).rejects.toThrow('stale preview');

    await expect(
      geofenceMockService.createManualBindJob(
        'mock-geofence-polygon',
        {
          ...inputs,
          previewFingerprint: preview.previewFingerprint,
          reason: '批量纳管',
        },
      ),
    ).resolves.toEqual({ jobId: 'mock-job-1' });
  });

  it('requires lifecycle preview and never treats disable as device recovery', async () => {
    const preview =
      await geofenceMockService.previewLifecycleTransition(
        'mock-geofence-polygon',
        'disabled',
      );

    await expect(
      geofenceMockService.transitionLifecycle(
        'mock-geofence-polygon',
        'disabled',
        {
          reason: '维护窗口',
          previewFingerprint: 'stale',
        },
      ),
    ).rejects.toThrow('stale preview');

    await geofenceMockService.transitionLifecycle(
      'mock-geofence-polygon',
      'disabled',
      {
        reason: '维护窗口',
        previewFingerprint: preview.previewFingerprint,
      },
    );

    const definition =
      await geofenceMockService.getDefinition(
        'mock-geofence-polygon',
      );
    const bindings = await geofenceMockService.listBindings(
      'mock-geofence-polygon',
      {},
    );
    expect(definition.status).toBe('disabled');
    expect(bindings.items.map((item) => item.status)).toEqual([
      'active',
      'suspended',
    ]);
  });
});
