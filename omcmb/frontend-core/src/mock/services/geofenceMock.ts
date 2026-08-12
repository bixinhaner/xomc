import type {
  CreateGeofenceDefinitionInput,
  CreateGeofenceDefinitionResult,
  CreateGeofenceManualBindJobInput,
  CreateGeofenceVersionInput,
  GeofenceBatchItem,
  GeofenceBatchItemFilter,
  GeofenceBatchItemPage,
  GeofenceBatchJobAccepted,
  GeofenceBinding,
  GeofenceBindingDetail,
  GeofenceBindingFilter,
  GeofenceBindingInputs,
  GeofenceBindingPage,
  GeofenceCandidatePreview,
  GeofenceControlAction,
  GeofenceBoundingBox,
  GeofenceDefinition,
  GeofenceDefinitionFilter,
  GeofenceGeometry,
  GeofenceLifecycleImpact,
  GeofenceLifecycleTarget,
  GeofenceManualBindJob,
  GeofenceManualBindingPreview,
  GeofenceMapDefinitionList,
  GeofenceMapFilter,
  GeofencePolicy,
  GeofenceSettings,
  GeofenceSettingsPreview,
  GeofenceTransitionInput,
  GeofenceVersion,
  GeofenceVersionList,
  UpdateGeofenceSettingsInput,
} from '../../types/geofence';
import type { geofenceApi } from '../../services/api/geofenceApi';
import { saveBlob } from '../../utils/saveBlob';

const MOCK_NOW = '2026-07-31T08:00:00Z';
const MOCK_ACTOR = 'mock-user';

const polygonGeometry: GeofenceGeometry = {
  type: 'Polygon',
  coordinates: [
    [
      [120.1, 30.1],
      [120.2, 30.1],
      [120.2, 30.2],
      [120.1, 30.1],
    ],
  ],
};

const circleGeometry: GeofenceGeometry = {
  type: 'Circle',
  center: [120.16, 30.16],
  radiusMeters: 1000,
  source: 'accepted_location',
  sourceObservationVersion: 7,
};

const defaultPolicy: GeofencePolicy = {
  exitAction: 'notify_only',
  exitToleranceMeters: 20,
  reentryToleranceMeters: 20,
  exitConsecutiveSamples: 3,
  reentryConsecutiveSamples: 3,
  sampleMaxAgeSeconds: 300,
};

const initialDefinitions: GeofenceDefinition[] = [
  {
    id: 'mock-geofence-polygon',
    name: '杭州多边形保护区',
    carrier: 'cmcc',
    ruleType: 'polygon_allow_zone',
    status: 'enabled',
    currentVersionId: 'mock-version-polygon-1',
    createdBy: MOCK_ACTOR,
    updatedBy: MOCK_ACTOR,
    createdAt: MOCK_NOW,
    updatedAt: MOCK_NOW,
  },
  {
    id: 'mock-geofence-radius',
    name: '杭州基线半径保护区',
    carrier: 'ctcc',
    ruleType: 'baseline_radius',
    ownerDeviceId: 'mock-device-003',
    status: 'disabled',
    currentVersionId: 'mock-version-radius-1',
    createdBy: MOCK_ACTOR,
    updatedBy: MOCK_ACTOR,
    createdAt: MOCK_NOW,
    updatedAt: MOCK_NOW,
  },
];

const initialVersions: GeofenceVersion[] = [
  {
    id: 'mock-version-polygon-1',
    geofenceId: 'mock-geofence-polygon',
    version: 1,
    status: 'published',
    geometry: polygonGeometry,
    boundingBox: boundingBoxFor(polygonGeometry),
    policy: defaultPolicy,
    createdBy: MOCK_ACTOR,
    publishedBy: MOCK_ACTOR,
    createdAt: MOCK_NOW,
    publishedAt: MOCK_NOW,
  },
  {
    id: 'mock-version-radius-1',
    geofenceId: 'mock-geofence-radius',
    version: 1,
    status: 'published',
    geometry: circleGeometry,
    boundingBox: boundingBoxFor(circleGeometry),
    policy: defaultPolicy,
    createdBy: MOCK_ACTOR,
    publishedBy: MOCK_ACTOR,
    createdAt: MOCK_NOW,
    publishedAt: MOCK_NOW,
  },
];

const initialBindings: GeofenceBindingDetail[] = [
  {
    id: 'mock-binding-001',
    deviceId: 'mock-device-001',
    geofenceId: 'mock-geofence-polygon',
    ruleType: 'polygon_allow_zone',
    status: 'active',
    bindSource: 'manual',
    boundBy: MOCK_ACTOR,
    boundAt: MOCK_NOW,
    deviceSN: 'MOCK-SN-001',
    deviceName: '杭州西湖站',
    deviceCarrier: 'cmcc',
    deviceGroupId: 'mock-group-hangzhou',
    deviceGroupName: '杭州',
    hasLocation: true,
    evaluation: {
      confirmedState: 'inside',
      candidateCount: 0,
      lastObservationVersion: 7,
      lastObservedAt: MOCK_NOW,
      lastDistanceToBoundary: -120,
      evaluationHealth: 'healthy',
    },
  },
  {
    id: 'mock-binding-002',
    deviceId: 'mock-device-002',
    geofenceId: 'mock-geofence-polygon',
    ruleType: 'polygon_allow_zone',
    status: 'suspended',
    bindSource: 'manual',
    boundBy: MOCK_ACTOR,
    boundAt: MOCK_NOW,
    deviceSN: 'MOCK-SN-002',
    deviceName: '杭州无定位站',
    deviceCarrier: 'cmcc',
    deviceGroupId: 'mock-group-hangzhou',
    deviceGroupName: '杭州',
    hasLocation: false,
    evaluation: {
      confirmedState: 'unknown',
      candidateCount: 0,
      evaluationHealth: 'healthy',
    },
  },
];

const knownDevices = [
  {
    id: 'mock-device-001',
    sn: 'MOCK-SN-001',
    name: '杭州西湖站',
    carrier: 'cmcc',
    hasLocation: true,
  },
  {
    id: 'mock-device-002',
    sn: 'MOCK-SN-002',
    name: '杭州无定位站',
    carrier: 'cmcc',
    hasLocation: false,
  },
  {
    id: 'mock-device-003',
    sn: 'MOCK-SN-003',
    name: '杭州滨江站',
    carrier: 'cmcc',
    hasLocation: true,
  },
] as const;

const initialSettings: GeofenceSettings = {
  systemMode: 'observe',
  carriers: [
    {
      carrier: 'cmcc',
      mode: 'observe',
      effectiveMode: 'observe',
      defaultBaselineRadiusMeters: 1000,
      updatedBy: MOCK_ACTOR,
      updatedAt: MOCK_NOW,
    },
    {
      carrier: 'ctcc',
      mode: 'off',
      effectiveMode: 'off',
      defaultBaselineRadiusMeters: 1000,
      updatedBy: MOCK_ACTOR,
      updatedAt: MOCK_NOW,
    },
    {
      carrier: 'cucc',
      mode: 'off',
      effectiveMode: 'off',
      defaultBaselineRadiusMeters: 1000,
      updatedBy: MOCK_ACTOR,
      updatedAt: MOCK_NOW,
    },
  ],
};

let definitions: GeofenceDefinition[] = [];
let versions: GeofenceVersion[] = [];
let bindings: GeofenceBindingDetail[] = [];
let settings: GeofenceSettings;
let jobs: GeofenceManualBindJob[] = [];
let jobItems: GeofenceBatchItem[] = [];
let jobReads = new Map<string, number>();
let definitionSequence = 0;
let versionSequence = 0;
let bindingSequence = 2;
let jobSequence = 0;

function clone<T>(value: T): T {
  return structuredClone(value);
}

function boundingBoxFor(
  geometry: GeofenceGeometry,
): GeofenceBoundingBox {
  if (geometry.type === 'Polygon') {
    const points = geometry.coordinates.flat();
    return {
      minLongitude: Math.min(...points.map(([lng]) => lng)),
      minLatitude: Math.min(...points.map(([, lat]) => lat)),
      maxLongitude: Math.max(...points.map(([lng]) => lng)),
      maxLatitude: Math.max(...points.map(([, lat]) => lat)),
    };
  }
  const latitudeDegrees = geometry.radiusMeters / 111_320;
  const longitudeDegrees =
    geometry.radiusMeters /
    (111_320 *
      Math.max(Math.cos((geometry.center[1] * Math.PI) / 180), 0.01));
  return {
    minLongitude: geometry.center[0] - longitudeDegrees,
    minLatitude: geometry.center[1] - latitudeDegrees,
    maxLongitude: geometry.center[0] + longitudeDegrees,
    maxLatitude: geometry.center[1] + latitudeDegrees,
  };
}

function findDefinition(id: string): GeofenceDefinition {
  const definition = definitions.find((item) => item.id === id);
  if (!definition) throw new Error(`geofence ${id} not found`);
  return definition;
}

function findBinding(id: string): GeofenceBindingDetail {
  const binding = bindings.find((item) => item.id === id);
  if (!binding) throw new Error(`geofence binding ${id} not found`);
  return binding;
}

function effectiveMode(
  systemMode: GeofenceSettings['systemMode'],
  carrierMode: GeofenceSettings['systemMode'],
) {
  const rank = { off: 0, observe: 1, enforce: 2 } as const;
  return rank[systemMode] <= rank[carrierMode]
    ? systemMode
    : carrierMode;
}

function materializeSettings(
  input: UpdateGeofenceSettingsInput,
): GeofenceSettings {
  return {
    systemMode: input.systemMode,
    carriers: input.carriers.map((carrier) => ({
      ...carrier,
      effectiveMode: effectiveMode(
        input.systemMode,
        carrier.mode,
      ),
      updatedBy: MOCK_ACTOR,
      updatedAt: MOCK_NOW,
    })),
  };
}

function matchesDefinition(
  definition: GeofenceDefinition,
  filter: GeofenceDefinitionFilter,
) {
  return (
    (!filter.carrier ||
      definition.carrier === filter.carrier) &&
    (!filter.status || definition.status === filter.status) &&
    (!filter.name || definition.name.includes(filter.name))
  );
}

function intersectsBounds(
  box: GeofenceBoundingBox,
  bounds: NonNullable<GeofenceMapFilter['bounds']>,
) {
  return !(
    box.maxLongitude < bounds.minLng ||
    box.minLongitude > bounds.maxLng ||
    box.maxLatitude < bounds.minLat ||
    box.minLatitude > bounds.maxLat
  );
}

function lifecycleFingerprint(
  id: string,
  target: GeofenceLifecycleTarget,
) {
  const definition = findDefinition(id);
  const bindingCount = bindings.filter(
    (binding) =>
      binding.geofenceId === id && binding.status !== 'removed',
  ).length;
  return [
    'mock-lifecycle',
    definition.id,
    definition.currentVersionId ?? '',
    definition.status,
    target,
    bindingCount,
  ].join(':');
}

function inputFacts(inputs: GeofenceBindingInputs) {
  const raw = [
    ...(inputs.deviceIds ?? []).map((value) => ({
      key: `id:${value}`,
      value,
      kind: 'device_id' as const,
    })),
    ...(inputs.deviceSNs ?? []).map((value) => ({
      key: `sn:${value.trim()}`,
      value: value.trim(),
      kind: 'device_sn' as const,
    })),
  ].filter((input) => input.value);
  return Array.from(
    new Map(raw.map((input) => [input.key, input])).values(),
  );
}

function bindingPreview(
  id: string,
  inputs: GeofenceBindingInputs,
): GeofenceManualBindingPreview {
  const definition = findDefinition(id);
  const facts = inputFacts(inputs);
  const items = facts.map((input) => {
    const device = knownDevices.find((candidate) =>
      input.kind === 'device_id'
        ? candidate.id === input.value
        : candidate.sn === input.value,
    );
    const currentBinding = device
      ? bindings.find(
          (binding) =>
            binding.deviceId === device.id &&
            binding.ruleType === definition.ruleType &&
            binding.status === 'active',
        )
      : undefined;
    const decision =
      !device
        ? 'skipped'
        : !currentBinding
          ? 'eligible'
          : currentBinding.geofenceId === id
            ? 'skipped'
            : 'move';
    return {
      inputKey: input.key,
      input: input.value,
      deviceId: device?.id,
      deviceSN: device?.sn,
      decision,
      reasonCode: !device
        ? 'device_not_found'
        : decision === 'move'
          ? 'reassigned'
          : currentBinding
            ? 'already_bound'
          : undefined,
      sourceBindingId: decision === 'move' ? currentBinding?.id : undefined,
      sourceGeofenceId:
        decision === 'move' ? currentBinding?.geofenceId : undefined,
    } as GeofenceManualBindingPreview['items'][number];
  });
  const currentVersionId = definition.currentVersionId;
  if (!currentVersionId) {
    throw new Error('geofence has no published version');
  }
  const fingerprint = [
    'mock-binding',
    id,
    currentVersionId,
    ...items.map(
      (item) =>
        `${item.inputKey}:${item.deviceId ?? ''}:${item.decision}:${item.reasonCode ?? ''}`,
    ),
  ].join('|');
  return {
    geofenceId: id,
    geofenceVersionId: currentVersionId,
    ruleType: definition.ruleType,
    inputCount: items.length,
    eligibleCount: items.filter(
      (item) => item.decision === 'eligible',
    ).length,
    moveCount: items.filter(
      (item) => item.decision === 'move',
    ).length,
    skippedCount: items.filter(
      (item) => item.decision === 'skipped',
    ).length,
    items,
    previewFingerprint: fingerprint,
  };
}

function baseBinding(
  binding: GeofenceBindingDetail,
): GeofenceBinding {
  const {
    deviceSN: _deviceSN,
    deviceName: _deviceName,
    deviceCarrier: _deviceCarrier,
    deviceGroupId: _deviceGroupId,
    deviceGroupName: _deviceGroupName,
    hasLocation: _hasLocation,
    evaluation: _evaluation,
    ...base
  } = binding;
  return clone(base);
}

function completeJob(job: GeofenceManualBindJob) {
  if (job.status === 'succeeded') return;
  const items = jobItems.filter((item) => item.jobId === job.id);
  for (const item of items) {
    if (item.status !== 'pending') continue;
    const device = knownDevices.find(
      (candidate) => candidate.id === item.deviceId,
    );
    if (!device) {
      item.status = 'skipped';
      item.reasonCode = 'device_not_found';
      item.finishedAt = MOCK_NOW;
      item.updatedAt = MOCK_NOW;
      continue;
    }
    bindingSequence += 1;
    const definition = findDefinition(job.geofenceId);
    const sourceBinding = bindings.find(
      (binding) =>
        binding.deviceId === device.id &&
        binding.ruleType === definition.ruleType &&
        binding.status === 'active' &&
        binding.geofenceId !== job.geofenceId,
    );
    if (sourceBinding) {
      sourceBinding.status = 'removed';
      sourceBinding.removedBy = MOCK_ACTOR;
      sourceBinding.removedAt = MOCK_NOW;
      sourceBinding.removeReason = 'reassigned';
    }
    const bindingId = `mock-binding-${String(bindingSequence).padStart(3, '0')}`;
    bindings.push({
      id: bindingId,
      deviceId: device.id,
      geofenceId: job.geofenceId,
      ruleType: definition.ruleType,
      status: 'active',
      bindSource: 'manual',
      boundBy: MOCK_ACTOR,
      boundAt: MOCK_NOW,
      deviceSN: device.sn,
      deviceName: device.name,
      deviceCarrier: device.carrier,
      deviceGroupId: 'mock-group-hangzhou',
      deviceGroupName: '杭州',
      hasLocation: device.hasLocation,
      evaluation: {
        confirmedState: 'unknown',
        candidateCount: 0,
        evaluationHealth: 'healthy',
      },
    });
    item.status = 'succeeded';
    item.bindingId = bindingId;
    item.finishedAt = MOCK_NOW;
    item.updatedAt = MOCK_NOW;
  }
  job.status = 'succeeded';
  job.finishedAt = MOCK_NOW;
  job.progress = {
    total: items.length,
    pending: 0,
    succeeded: items.filter(
      (item) => item.status === 'succeeded',
    ).length,
    skipped: items.filter(
      (item) => item.status === 'skipped',
    ).length,
    failed: 0,
  };
}

export function resetGeofenceMock() {
  definitions = clone(initialDefinitions);
  versions = clone(initialVersions);
  bindings = clone(initialBindings);
  settings = clone(initialSettings);
  jobs = [];
  jobItems = [];
  jobReads = new Map();
  definitionSequence = 0;
  versionSequence = 0;
  bindingSequence = 2;
  jobSequence = 0;
}

resetGeofenceMock();

export const geofenceMockService = {
  async getAvailability() {
    return { enabled: settings.systemMode !== 'off' };
  },

  async getSettings(): Promise<GeofenceSettings> {
    return clone(settings);
  },

  async previewSettings(
    input: UpdateGeofenceSettingsInput,
  ): Promise<GeofenceSettingsPreview> {
    const proposed = materializeSettings(input);
    return {
      current: clone(settings),
      proposed,
      enabledGeofences: definitions.filter(
        (definition) => definition.status === 'enabled',
      ).length,
      activeBindings: bindings.filter(
        (binding) => binding.status === 'active',
      ).length,
      newlyObservedDevices:
        proposed.systemMode === 'observe' &&
        settings.systemMode === 'off'
          ? bindings.filter(
              (binding) => binding.status === 'active',
            ).length
          : 0,
    };
  },

  async updateSettings(
    input: UpdateGeofenceSettingsInput,
  ): Promise<GeofenceSettings> {
    settings = materializeSettings(input);
    return clone(settings);
  },

  async listMapDefinitions(
    filter: GeofenceMapFilter = {},
  ): Promise<GeofenceMapDefinitionList> {
    const items = definitions
      .filter((definition) => definition.status !== 'archived')
      .filter((definition) => matchesDefinition(definition, filter))
      .map((definition) => {
        const currentVersion =
          versions.find(
            (version) =>
              version.id === definition.currentVersionId,
          ) ?? null;
        return { definition, currentVersion };
      })
      .filter(
        (item) =>
          !filter.bounds ||
          !item.currentVersion ||
          intersectsBounds(
            item.currentVersion.boundingBox,
            filter.bounds,
          ),
      );
    return clone({ items });
  },

  async listDefinitions(
    filter: GeofenceDefinitionFilter = {},
  ): Promise<GeofenceDefinition[]> {
    return clone(
      definitions.filter((definition) =>
        matchesDefinition(definition, filter),
      ),
    );
  },

  async previewCandidates(id: string): Promise<GeofenceCandidatePreview> {
    const definition = findDefinition(id);
    const candidateBindings = bindings.filter(
      (binding) => binding.geofenceId === id && binding.status === 'active',
    );
    return clone({
      geofenceId: definition.id,
      versionId: definition.currentVersionId ?? '',
      inside: candidateBindings.map((binding) => ({
        id: binding.deviceId,
        serialNumber: binding.deviceSN,
        name: binding.deviceName,
        latitude: 30.15,
        longitude: 120.15,
      })),
      outside: [],
      noLocation: [],
    });
  },

  async listControlActions(_id: string): Promise<GeofenceControlAction[]> {
    return [];
  },

  async getDefinition(id: string): Promise<GeofenceDefinition> {
    return clone(findDefinition(id));
  },

  async renameDefinition(id: string, name: string): Promise<void> {
    findDefinition(id).name = name.trim();
  },

  async createDefinition(
    input: CreateGeofenceDefinitionInput,
  ): Promise<CreateGeofenceDefinitionResult> {
    definitionSequence += 1;
    versionSequence += 1;
    const id = `mock-geofence-new-${definitionSequence}`;
    const versionId = `mock-version-new-${versionSequence}`;
    const definition: GeofenceDefinition = {
      id,
      name: input.name,
      carrier: input.carrier,
      ruleType: input.ruleType,
      ownerDeviceId: input.ownerDeviceId,
      status: 'draft',
      createdBy: MOCK_ACTOR,
      updatedBy: MOCK_ACTOR,
      createdAt: MOCK_NOW,
      updatedAt: MOCK_NOW,
    };
    const draftVersion: GeofenceVersion = {
      id: versionId,
      geofenceId: id,
      version: 1,
      status: 'draft',
      geometry: clone(input.geometry),
      boundingBox: boundingBoxFor(input.geometry),
      policy: clone(input.policy),
      createdBy: MOCK_ACTOR,
      createdAt: MOCK_NOW,
    };
    definitions.push(definition);
    versions.push(draftVersion);
    return clone({ definition, draftVersion });
  },

  async listVersions(id: string): Promise<GeofenceVersionList> {
    findDefinition(id);
    return {
      items: clone(
        versions
          .filter((version) => version.geofenceId === id)
          .sort((a, b) => b.version - a.version),
      ),
    };
  },

  async createDraftVersion(
    id: string,
    input: CreateGeofenceVersionInput,
  ): Promise<GeofenceVersion> {
    findDefinition(id);
    versionSequence += 1;
    const previous = versions.filter(
      (version) => version.geofenceId === id,
    );
    const version: GeofenceVersion = {
      id: `mock-version-new-${versionSequence}`,
      geofenceId: id,
      version:
        Math.max(0, ...previous.map((item) => item.version)) + 1,
      status: 'draft',
      geometry: clone(input.geometry),
      boundingBox: boundingBoxFor(input.geometry),
      policy: clone(input.policy),
      createdBy: MOCK_ACTOR,
      createdAt: MOCK_NOW,
    };
    versions.push(version);
    return clone(version);
  },

  async publishDraft(
    id: string,
    versionId: string,
  ): Promise<{ status: 'executed' }> {
    const definition = findDefinition(id);
    const version = versions.find(
      (item) =>
        item.id === versionId &&
        item.geofenceId === id &&
        item.status === 'draft',
    );
    if (!version) throw new Error('draft version not found');
    for (const item of versions) {
      if (
        item.geofenceId === id &&
        item.status === 'published'
      ) {
        item.status = 'superseded';
      }
    }
    version.status = 'published';
    version.publishedBy = MOCK_ACTOR;
    version.publishedAt = MOCK_NOW;
    definition.currentVersionId = version.id;
    definition.status = 'enabled';
    definition.updatedAt = MOCK_NOW;
    return { status: 'executed' };
  },

  async previewLifecycleTransition(
    id: string,
    target: GeofenceLifecycleTarget,
  ): Promise<GeofenceLifecycleImpact> {
    const definition = findDefinition(id);
    const activeBindings = bindings.filter(
      (binding) =>
        binding.geofenceId === id &&
        binding.status !== 'removed',
    );
    return {
      geofenceId: id,
      currentVersionId: definition.currentVersionId,
      currentStatus: definition.status,
      targetStatus: target,
      bindingCount: activeBindings.length,
      deviceCount: new Set(
        activeBindings.map((binding) => binding.deviceId),
      ).size,
      activeBatchJobCount: jobs.filter(
        (job) =>
          job.geofenceId === id &&
          (job.status === 'pending' ||
            job.status === 'running'),
      ).length,
      previewFingerprint: lifecycleFingerprint(id, target),
    };
  },

  async transitionLifecycle(
    id: string,
    target: GeofenceLifecycleTarget,
    input: GeofenceTransitionInput,
  ): Promise<{ status: 'executed' }> {
    if (
      input.previewFingerprint !==
      lifecycleFingerprint(id, target)
    ) {
      throw new Error('stale preview');
    }
    const definition = findDefinition(id);
    definition.status = target;
    definition.updatedBy = MOCK_ACTOR;
    definition.updatedAt = MOCK_NOW;
    return { status: 'executed' };
  },

  async listBindings(
    id: string,
    filter: GeofenceBindingFilter = {},
  ): Promise<GeofenceBindingPage> {
    findDefinition(id);
    const page = filter.page ?? 1;
    const pageSize = filter.pageSize ?? 50;
    const keyword = filter.keyword?.toLowerCase();
    const filtered = bindings
      .filter((binding) => binding.geofenceId === id)
      .filter(
        (binding) =>
          !filter.status ||
          (filter.status === 'current'
            ? binding.status === 'active' || binding.status === 'suspended'
            : binding.status === filter.status),
      )
      .filter(
        (binding) =>
          !keyword ||
          binding.deviceSN.toLowerCase().includes(keyword) ||
          binding.deviceName.toLowerCase().includes(keyword),
      );
    const offset = (page - 1) * pageSize;
    return {
      items: clone(filtered.slice(offset, offset + pageSize)),
      total: filtered.length,
      page,
      pageSize,
    };
  },

  async exportBindings(
    id: string,
    filter: Pick<
      GeofenceBindingFilter,
      'status' | 'keyword'
    > = {},
  ): Promise<void> {
    findDefinition(id);
    const keyword = filter.keyword?.toLowerCase();
    const rows = bindings
      .filter((binding) => binding.geofenceId === id)
      .filter(
        (binding) =>
          !filter.status ||
          (filter.status === 'current'
            ? binding.status === 'active' || binding.status === 'suspended'
            : binding.status === filter.status),
      )
      .filter(
        (binding) =>
          !keyword ||
          binding.deviceSN.toLowerCase().includes(keyword) ||
          binding.deviceName.toLowerCase().includes(keyword),
      )
      .map((binding) => [
        binding.deviceSN,
        binding.deviceName,
        binding.deviceCarrier,
        binding.deviceGroupName ?? '',
        binding.status,
        binding.boundAt,
        binding.evaluation.confirmedState,
        binding.evaluation.evaluationHealth,
      ]);
    const csvRows = [
      [
        'device_sn',
        'device_name',
        'carrier',
        'device_group',
        'binding_status',
        'bound_at',
        'confirmed_state',
        'evaluation_health',
      ],
      ...rows,
    ];
    const csv = csvRows
      .map((row) =>
        row
          .map(
            (cell) =>
              `"${String(cell).replaceAll('"', '""')}"`,
          )
          .join(','),
      )
      .join('\r\n');
    saveBlob(
      `\ufeff${csv}`,
      'geofence-bindings.csv',
      'text/csv;charset=utf-8',
    );
  },

  async previewManualBindings(
    id: string,
    inputs: GeofenceBindingInputs,
  ): Promise<GeofenceManualBindingPreview> {
    return clone(bindingPreview(id, inputs));
  },

  async createManualBindJob(
    id: string,
    input: CreateGeofenceManualBindJobInput,
  ): Promise<GeofenceBatchJobAccepted> {
    const preview = bindingPreview(id, input);
    if (
      preview.previewFingerprint !== input.previewFingerprint
    ) {
      throw new Error('stale preview');
    }
    if (preview.eligibleCount + preview.moveCount === 0) {
      throw new Error('no eligible devices');
    }
    jobSequence += 1;
    const jobId = `mock-job-${jobSequence}`;
    const eligible = preview.items.filter(
      (item) => item.decision === 'eligible' || item.decision === 'move',
    );
    jobs.push({
      id: jobId,
      jobType: 'geofence_manual_bind',
      status: 'pending',
      geofenceId: id,
      requestedBy: MOCK_ACTOR,
      reason: input.reason,
      attempt: 0,
      maxAttempts: 3,
      createdAt: MOCK_NOW,
      progress: {
        total: eligible.length,
        pending: eligible.length,
        succeeded: 0,
        skipped: 0,
        failed: 0,
      },
    });
    jobItems.push(
      ...eligible.map((item, index): GeofenceBatchItem => ({
        id: `${jobId}-item-${index + 1}`,
        jobId,
        geofenceId: id,
        inputKey: item.inputKey,
        inputKind: item.inputKey.startsWith('id:')
          ? 'device_id'
          : 'device_sn',
        inputValue: item.input,
        deviceId: item.deviceId,
        deviceSN: item.deviceSN,
        status: 'pending' as const,
        attempt: 0,
        createdAt: MOCK_NOW,
        updatedAt: MOCK_NOW,
      })),
    );
    return { jobId };
  },

  async getManualBindJob(
    id: string,
  ): Promise<GeofenceManualBindJob> {
    const job = jobs.find((item) => item.id === id);
    if (!job) throw new Error(`geofence job ${id} not found`);
    const reads = (jobReads.get(id) ?? 0) + 1;
    jobReads.set(id, reads);
    if (job.status === 'pending') {
      job.status = 'running';
      job.startedAt = MOCK_NOW;
      job.attempt = 1;
    } else if (job.status === 'running' && reads >= 2) {
      completeJob(job);
    }
    return clone(job);
  },

  async listManualBindItems(
    id: string,
    filter: GeofenceBatchItemFilter = {},
  ): Promise<GeofenceBatchItemPage> {
    const page = filter.page ?? 1;
    const pageSize = filter.pageSize ?? 50;
    const filtered = jobItems
      .filter((item) => item.jobId === id)
      .filter(
        (item) =>
          !filter.status || item.status === filter.status,
      );
    const offset = (page - 1) * pageSize;
    return {
      items: clone(filtered.slice(offset, offset + pageSize)),
      total: filtered.length,
      page,
      pageSize,
    };
  },

  async suspendBinding(
    id: string,
    _reason: string,
  ): Promise<GeofenceBinding> {
    const binding = findBinding(id);
    binding.status = 'suspended';
    return baseBinding(binding);
  },

  async resumeBinding(
    id: string,
    _reason: string,
  ): Promise<GeofenceBinding> {
    const binding = findBinding(id);
    binding.status = 'active';
    return baseBinding(binding);
  },

  async removeBinding(
    id: string,
    reason: string,
  ): Promise<GeofenceBinding> {
    const binding = findBinding(id);
    binding.status = 'removed';
    binding.removedBy = MOCK_ACTOR;
    binding.removedAt = MOCK_NOW;
    binding.removeReason = reason;
    return baseBinding(binding);
  },
} satisfies typeof geofenceApi;
