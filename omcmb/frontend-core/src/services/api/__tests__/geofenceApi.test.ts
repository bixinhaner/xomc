import { beforeEach, describe, expect, it, vi } from 'vitest';

const { getMock, postMock, putMock, deleteMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
  putMock: vi.fn(),
  deleteMock: vi.fn(),
}));

vi.mock('../../http', () => ({
  default: {
    get: getMock,
    post: postMock,
    put: putMock,
    delete: deleteMock,
  },
}));

import { geofenceApi } from '../geofenceApi';

const definitionRaw = {
  id: 'geofence-1',
  name: '杭州保护区',
  carrier: 'cmcc',
  rule_type: 'polygon_allow_zone',
  status: 'enabled',
  current_version_id: 'version-2',
  created_by: 'user-1',
  updated_by: 'user-2',
  created_at: '2026-07-30T08:00:00Z',
  updated_at: '2026-07-31T08:00:00Z',
};

const versionRaw = {
  id: 'version-2',
  geofence_id: 'geofence-1',
  version: 2,
  status: 'published',
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
  bounding_box: {
    min_longitude: 120.1,
    max_longitude: 120.2,
    min_latitude: 30.1,
    max_latitude: 30.2,
  },
  policy: {
    exit_consecutive_samples: 3,
    reentry_consecutive_samples: 2,
    exit_action: 'notify_only',
  },
  created_by: 'user-1',
  published_by: 'user-2',
  created_at: '2026-07-30T08:00:00Z',
  published_at: '2026-07-31T08:00:00Z',
};

beforeEach(() => {
  getMock.mockReset();
  postMock.mockReset();
  putMock.mockReset();
  deleteMock.mockReset();
});

describe('geofenceApi read contract', () => {
  it('reads the minimal system availability contract', async () => {
    getMock.mockResolvedValue({ data: { enabled: true } });

    await expect(geofenceApi.getAvailability()).resolves.toEqual({
      enabled: true,
    });
    expect(getMock).toHaveBeenCalledWith('/geofences/availability');
  });

  it('serializes GIS bounds in the backend order and maps the current version', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [
          {
            definition: definitionRaw,
            current_version: versionRaw,
          },
          {
            definition: {
              ...definitionRaw,
              id: 'geofence-draft',
              status: 'draft',
              current_version_id: undefined,
            },
          },
        ],
      },
    });

    const result = await geofenceApi.listMapDefinitions({
      bounds: {
        minLng: 120,
        maxLng: 121,
        minLat: 30,
        maxLat: 31,
      },
      carrier: 'cmcc',
      status: 'enabled',
      name: '杭州',
    });

    expect(getMock).toHaveBeenCalledWith('/geofences/map', {
      params: {
        bounds: '120,121,30,31',
        carrier: 'cmcc',
        status: 'enabled',
        name: '杭州',
      },
    });
    expect(result.items[0]).toMatchObject({
      definition: {
        id: 'geofence-1',
        ruleType: 'polygon_allow_zone',
        currentVersionId: 'version-2',
        updatedBy: 'user-2',
      },
      currentVersion: {
        geofenceId: 'geofence-1',
        boundingBox: {
          minLongitude: 120.1,
          maxLongitude: 120.2,
          minLatitude: 30.1,
          maxLatitude: 30.2,
        },
        policy: {
          exitConsecutiveSamples: 3,
          reentryConsecutiveSamples: 2,
          exitAction: 'notify_only',
        },
      },
    });
    expect(result.items[1].currentVersion).toBeNull();
  });

  it('maps hierarchical settings without relying on a response interceptor', async () => {
    getMock.mockResolvedValue({
      data: {
        system_mode: 'observe',
        carriers: [
          {
            carrier: 'cmcc',
            mode: 'observe',
            effective_mode: 'observe',
            default_baseline_radius_meters: 1000,
            updated_by: 'user-1',
            updated_at: '2026-07-31T08:00:00Z',
          },
        ],
      },
    });

    const settings = await geofenceApi.getSettings();

    expect(getMock).toHaveBeenCalledWith('/geofences/settings');
    expect(settings).toEqual({
      systemMode: 'observe',
      carriers: [
        {
          carrier: 'cmcc',
          mode: 'observe',
          effectiveMode: 'observe',
          defaultBaselineRadiusMeters: 1000,
          updatedBy: 'user-1',
          updatedAt: '2026-07-31T08:00:00Z',
        },
      ],
    });
  });

  it('maps durable control action readback instead of generic task completion', async () => {
    getMock.mockResolvedValue({
      data: [{
        id: 'action-1',
        device_sn: 'SN001',
        action_key: 'geofence:device:8:deactivate',
        action_type: 'deactivate',
        status: 'partial_failed',
        before_state: [{ path: 'rf', value: '1' }],
        requested_state: [{ path: 'rf', value: '0' }],
        verified_state: [{ path: 'rf', value: '1' }],
        last_error: 'rf expected 0 got 1',
        created_at: '2026-08-05T08:00:00Z',
        completed_at: '2026-08-05T08:00:05Z',
      }],
    });

    const actions = await geofenceApi.listControlActions('geofence-1');

    expect(getMock).toHaveBeenCalledWith('/geofences/geofence-1/control-actions');
    expect(actions).toEqual([expect.objectContaining({
      id: 'action-1',
      commandKey: 'geofence:device:8:deactivate',
      actionType: 'deactivate',
      status: 'partial_failed',
      requestedState: [{ path: 'rf', value: '0' }],
      verifiedState: [{ path: 'rf', value: '1' }],
      lastError: 'rf expected 0 got 1',
    })]);
  });

  it('maps version history and definition list responses', async () => {
    getMock
      .mockResolvedValueOnce({ data: { items: [versionRaw] } })
      .mockResolvedValueOnce({ data: [definitionRaw] })
      .mockResolvedValueOnce({ data: definitionRaw });

    const versions = await geofenceApi.listVersions('geofence-1');
    const definitions = await geofenceApi.listDefinitions({
      carrier: 'cmcc',
      status: 'enabled',
      name: '杭州',
    });
    const definition = await geofenceApi.getDefinition('geofence-1');

    expect(getMock).toHaveBeenNthCalledWith(
      1,
      '/geofences/geofence-1/versions',
    );
    expect(getMock).toHaveBeenNthCalledWith(2, '/geofences', {
      params: { carrier: 'cmcc', status: 'enabled', name: '杭州' },
    });
    expect(getMock).toHaveBeenNthCalledWith(
      3,
      '/geofences/geofence-1',
    );
    expect(versions.items[0].publishedBy).toBe('user-2');
    expect(definitions[0].ruleType).toBe('polygon_allow_zone');
    expect(definition.currentVersionId).toBe('version-2');
  });

  it('maps permission-filtered binding pages without introducing coordinates', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [
          {
            id: 'binding-1',
            device_id: 'device-1',
            geofence_id: 'geofence-1',
            rule_type: 'polygon_allow_zone',
            status: 'active',
            bind_source: 'manual',
            bound_by: 'user-1',
            bound_at: '2026-07-31T08:00:00Z',
            device_sn: 'SN001',
            device_name: '杭州站点',
            device_carrier: 'cmcc',
            device_group_id: 'group-1',
            device_group_name: '杭州',
            has_location: true,
            evaluation: {
              confirmed_state: 'inside',
              candidate_state: 'exit',
              candidate_count: 2,
              last_observation_version: 12,
              last_observed_at: '2026-07-31T08:05:00Z',
              last_distance_to_boundary: 35.5,
              evaluation_health: 'healthy',
            },
          },
        ],
        total: 1,
        page: 2,
        page_size: 20,
      },
    });

    const page = await geofenceApi.listBindings('geofence-1', {
      status: 'current',
      keyword: 'SN001',
      page: 2,
      pageSize: 20,
    });

    expect(getMock).toHaveBeenCalledWith(
      '/geofences/geofence-1/bindings',
      {
        params: {
          status: 'current',
          keyword: 'SN001',
          page: 2,
          page_size: 20,
        },
      },
    );
    expect(page).toMatchObject({
      total: 1,
      page: 2,
      pageSize: 20,
      items: [
        {
          id: 'binding-1',
          deviceId: 'device-1',
          deviceSN: 'SN001',
          deviceGroupId: 'group-1',
          hasLocation: true,
          evaluation: {
            confirmedState: 'inside',
            candidateState: 'exit',
            candidateCount: 2,
            lastObservationVersion: 12,
            lastObservedAt: '2026-07-31T08:05:00Z',
            lastDistanceToBoundary: 35.5,
            evaluationHealth: 'healthy',
          },
        },
      ],
    });
    expect(page.items[0]).not.toHaveProperty('latitude');
    expect(page.items[0]).not.toHaveProperty('longitude');
  });

  it('downloads the permission-filtered binding CSV with the server filename', async () => {
    const blob = new Blob(['device_sn\nSN001\n'], {
      type: 'text/csv',
    });
    getMock.mockResolvedValue({
      data: blob,
      headers: {
        'content-disposition':
          'attachment; filename="geofence-bindings.csv"',
      },
    });
    const createObjectURL = vi
      .spyOn(window.URL, 'createObjectURL')
      .mockReturnValue('blob:geofence-bindings');
    const revokeObjectURL = vi
      .spyOn(window.URL, 'revokeObjectURL')
      .mockImplementation(() => undefined);
    let downloadedFilename = '';
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(function captureDownloadName() {
        downloadedFilename = this.download;
      });

    await geofenceApi.exportBindings('geofence-1', {
      status: 'active',
      keyword: 'SN001',
    });

    expect(getMock).toHaveBeenCalledWith(
      '/geofences/geofence-1/bindings/export',
      {
        params: {
          status: 'active',
          keyword: 'SN001',
        },
        responseType: 'blob',
      },
    );
    expect(createObjectURL).toHaveBeenCalledOnce();
    expect(downloadedFilename).toBe('geofence-bindings.csv');
    expect(revokeObjectURL).toHaveBeenCalledWith(
      'blob:geofence-bindings',
    );

    createObjectURL.mockRestore();
    revokeObjectURL.mockRestore();
    click.mockRestore();
  });

  it('maps manual binding job progress and item pagination', async () => {
    getMock
      .mockResolvedValueOnce({
        data: {
          id: 'job-1',
          job_type: 'geofence_manual_bind',
          status: 'running',
          geofence_id: 'geofence-1',
          requested_by: 'user-1',
          reason: '补录站点',
          attempt: 1,
          max_attempts: 3,
          created_at: '2026-07-31T08:00:00Z',
          started_at: '2026-07-31T08:00:01Z',
          progress: {
            total: 2,
            pending: 1,
            succeeded: 1,
            skipped: 0,
            failed: 0,
          },
        },
      })
      .mockResolvedValueOnce({
        data: {
          items: [
            {
              id: 'item-1',
              job_id: 'job-1',
              geofence_id: 'geofence-1',
              input_key: 'sn:SN001',
              input_kind: 'device_sn',
              input_value: 'SN001',
              device_id: 'device-1',
              device_sn: 'SN001',
              status: 'succeeded',
              binding_id: 'binding-1',
              attempt: 1,
              created_at: '2026-07-31T08:00:00Z',
              updated_at: '2026-07-31T08:00:02Z',
            },
          ],
          total: 1,
          page: 1,
          page_size: 50,
        },
      });

    const job = await geofenceApi.getManualBindJob('job-1');
    const items = await geofenceApi.listManualBindItems('job-1', {
      status: 'succeeded',
      page: 1,
      pageSize: 50,
    });

    expect(job).toMatchObject({
      jobType: 'geofence_manual_bind',
      geofenceId: 'geofence-1',
      requestedBy: 'user-1',
      maxAttempts: 3,
      progress: { pending: 1, succeeded: 1 },
    });
    expect(items.items[0]).toMatchObject({
      jobId: 'job-1',
      inputKey: 'sn:SN001',
      inputKind: 'device_sn',
      deviceSN: 'SN001',
      bindingId: 'binding-1',
    });
    expect(items.pageSize).toBe(50);
  });
});

describe('geofenceApi write contract', () => {
  const settingsInput = {
    systemMode: 'observe' as const,
    carriers: [
      {
        carrier: 'cmcc',
        mode: 'observe' as const,
        defaultBaselineRadiusMeters: 800,
      },
    ],
  };

  const settingsBody = {
    system_mode: 'observe',
    carriers: [
      {
        carrier: 'cmcc',
        mode: 'observe',
        effective_mode: 'observe',
        default_baseline_radius_meters: 800,
        updated_at: '2026-07-31T08:00:00Z',
      },
    ],
  };
  const settingsRequest = {
    system_mode: 'observe',
    carriers: [
      {
        carrier: 'cmcc',
        mode: 'observe',
        default_baseline_radius_meters: 800,
      },
    ],
  };

  it('serializes settings for preview and update and maps preview impact', async () => {
    postMock.mockResolvedValue({
      data: {
        current: {
          ...settingsBody,
          system_mode: 'off',
        },
        proposed: settingsBody,
        enabled_geofences: 2,
        active_bindings: 12,
        newly_observed_devices: 8,
      },
    });
    putMock.mockResolvedValue({ data: settingsBody });

    const preview = await geofenceApi.previewSettings(settingsInput);
    const updated = await geofenceApi.updateSettings(settingsInput);

    expect(postMock).toHaveBeenCalledWith(
      '/geofences/settings/preview',
      settingsRequest,
    );
    expect(putMock).toHaveBeenCalledWith(
      '/geofences/settings',
      settingsRequest,
    );
    expect(preview).toMatchObject({
      current: { systemMode: 'off' },
      proposed: { systemMode: 'observe' },
      enabledGeofences: 2,
      activeBindings: 12,
      newlyObservedDevices: 8,
    });
    expect(updated.systemMode).toBe('observe');
  });

  it('creates definitions and immutable versions with explicit wire fields', async () => {
    postMock
      .mockResolvedValueOnce({
        data: {
          definition: definitionRaw,
          draft_version: {
            ...versionRaw,
            id: 'version-draft',
            version: 1,
            status: 'draft',
            published_by: undefined,
            published_at: undefined,
          },
        },
      })
      .mockResolvedValueOnce({
        data: {
          ...versionRaw,
          id: 'version-3',
          version: 3,
          status: 'draft',
        },
      })
      .mockResolvedValueOnce({ data: { status: 'executed' } });

    const input = {
      name: '杭州保护区',
      carrier: 'cmcc',
      ruleType: 'polygon_allow_zone' as const,
      geometry: versionRaw.geometry,
      policy: {
        exitAction: 'notify_only' as const,
        exitConsecutiveSamples: 3,
      },
    };
    const created = await geofenceApi.createDefinition(input);
    const draft = await geofenceApi.createDraftVersion(
      'geofence-1',
      {
        geometry: input.geometry,
        policy: input.policy,
      },
    );
    const published = await geofenceApi.publishDraft(
      'geofence-1',
      'version-3',
    );

    expect(postMock).toHaveBeenNthCalledWith(1, '/geofences', {
      name: '杭州保护区',
      carrier: 'cmcc',
      rule_type: 'polygon_allow_zone',
      owner_device_id: undefined,
      geometry: versionRaw.geometry,
      policy: {
        exit_action: 'notify_only',
        exit_consecutive_samples: 3,
      },
    });
    expect(postMock).toHaveBeenNthCalledWith(
      2,
      '/geofences/geofence-1/versions',
      {
        geometry: versionRaw.geometry,
        policy: {
          exit_action: 'notify_only',
          exit_consecutive_samples: 3,
        },
      },
    );
    expect(postMock).toHaveBeenNthCalledWith(
      3,
      '/geofences/geofence-1/publish',
      { version_id: 'version-3' },
    );
    expect(created.draftVersion.status).toBe('draft');
    expect(draft.id).toBe('version-3');
    expect(published).toEqual({ status: 'executed' });
  });

  it.each([
    ['enabled', 'enable'],
    ['disabled', 'disable'],
    ['archived', 'archive'],
  ] as const)(
    'uses preview fingerprint for %s lifecycle transition',
    async (target, pathAction) => {
      postMock
        .mockResolvedValueOnce({
          data: {
            geofence_id: 'geofence-1',
            current_version_id: 'version-2',
            current_status: 'disabled',
            target_status: target,
            binding_count: 12,
            device_count: 10,
            deactivation_device_count: 4,
            active_batch_job_count: 0,
            preview_fingerprint: 'fingerprint-1',
          },
        })
        .mockResolvedValueOnce({ data: { status: 'executed' } });

      const preview =
        await geofenceApi.previewLifecycleTransition(
          'geofence-1',
          target,
        );
      await geofenceApi.transitionLifecycle(
        'geofence-1',
        target,
        {
          reason: '业务确认',
          previewFingerprint: preview.previewFingerprint,
        },
      );

      expect(postMock).toHaveBeenNthCalledWith(
        1,
        `/geofences/geofence-1/${pathAction}-preview`,
      );
      expect(postMock).toHaveBeenNthCalledWith(
        2,
        `/geofences/geofence-1/${pathAction}`,
        {
          reason: '业务确认',
          preview_fingerprint: 'fingerprint-1',
        },
      );
      expect(preview).toMatchObject({
        currentStatus: 'disabled',
        targetStatus: target,
        bindingCount: 12,
        deactivationDeviceCount: 4,
        activeBatchJobCount: 0,
      });
    },
  );

  it('previews manual binding before creating the async job', async () => {
    postMock
      .mockResolvedValueOnce({
        data: {
          geofence_id: 'geofence-1',
          geofence_version_id: 'version-2',
          rule_type: 'polygon_allow_zone',
          input_count: 2,
          eligible_count: 1,
          move_count: 0,
          skipped_count: 1,
          items: [
            {
              input_key: 'sn:SN001',
              input: 'SN001',
              device_id: 'device-1',
              device_sn: 'SN001',
              decision: 'eligible',
            },
            {
              input_key: 'sn:UNKNOWN',
              input: 'UNKNOWN',
              decision: 'skipped',
              reason_code: 'device_not_found',
            },
          ],
          preview_fingerprint: 'sha256:preview',
        },
      })
      .mockResolvedValueOnce({ data: { job_id: 'job-1' } });

    const inputs = {
      deviceIds: ['device-1'],
      deviceSNs: ['SN001'],
    };
    const preview = await geofenceApi.previewManualBindings(
      'geofence-1',
      inputs,
    );
    const accepted = await geofenceApi.createManualBindJob(
      'geofence-1',
      {
        ...inputs,
        previewFingerprint: preview.previewFingerprint,
        reason: '批量纳管',
        scheduledAt: '2026-07-31T09:00:00Z',
      },
    );

    expect(postMock).toHaveBeenNthCalledWith(
      1,
      '/geofences/geofence-1/binding-preview',
      {
        device_ids: ['device-1'],
        device_sns: ['SN001'],
      },
    );
    expect(postMock).toHaveBeenNthCalledWith(
      2,
      '/geofences/geofence-1/bindings',
      {
        device_ids: ['device-1'],
        device_sns: ['SN001'],
        preview_fingerprint: 'sha256:preview',
        reason: '批量纳管',
        scheduled_at: '2026-07-31T09:00:00Z',
      },
    );
    expect(preview).toMatchObject({
      geofenceVersionId: 'version-2',
      eligibleCount: 1,
      moveCount: 0,
      skippedCount: 1,
      items: [
        { inputKey: 'sn:SN001', deviceSN: 'SN001' },
        { reasonCode: 'device_not_found' },
      ],
    });
    expect(accepted).toEqual({ jobId: 'job-1' });
  });

  it('uses explicit reason for destructive binding transitions', async () => {
    const bindingRaw = {
      id: 'binding-1',
      device_id: 'device-1',
      geofence_id: 'geofence-1',
      rule_type: 'polygon_allow_zone',
      status: 'suspended',
      bind_source: 'manual',
      bound_by: 'user-1',
      bound_at: '2026-07-31T08:00:00Z',
    };
    postMock
      .mockResolvedValueOnce({ data: bindingRaw })
      .mockResolvedValueOnce({
        data: { ...bindingRaw, status: 'active' },
      });
    deleteMock.mockResolvedValue({
      data: {
        ...bindingRaw,
        status: 'removed',
        remove_reason: '错误绑定',
      },
    });

    const suspended = await geofenceApi.suspendBinding(
      'binding-1',
      '现场核查',
    );
    const resumed = await geofenceApi.resumeBinding(
      'binding-1',
      '维护完成',
    );
    const removed = await geofenceApi.removeBinding(
      'binding-1',
      '错误绑定',
    );

    expect(postMock).toHaveBeenNthCalledWith(
      1,
      '/geofence-bindings/binding-1/suspend',
      { reason: '现场核查' },
    );
    expect(postMock).toHaveBeenNthCalledWith(
      2,
      '/geofence-bindings/binding-1/resume',
      { reason: '维护完成' },
    );
    expect(deleteMock).toHaveBeenCalledWith(
      '/geofence-bindings/binding-1',
      { data: { reason: '错误绑定' } },
    );
    expect(suspended.status).toBe('suspended');
    expect(resumed.status).toBe('active');
    expect(removed.removeReason).toBe('错误绑定');
  });
});
