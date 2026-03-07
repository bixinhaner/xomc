import http from '../http';
import type { SoftwareVersion, UpgradePlan, VersionStatus, UpgradePlanStatus } from '@/mock/data/software';
import type { PageRequest, PageResponse } from '@/types/pagination';
import { softwareService } from '@/mock/services/softwareService';

// Backend firmware model
interface BackendFirmwareVersion {
  id: string;
  carrier: string;
  product_class: string;
  version: string;
  file_name: string;
  file_size: number;
  minio_path: string;
  compatible_oui: string[];
  release_notes: string;
  status: string;
  created_at: string;
  updated_at: string;
}

// Backend upgrade task model (per-device granularity)
interface BackendUpgradeTask {
  id: string;
  device_id: string;
  firmware_id: string;
  batch_id?: string;
  status: string; // pending, downloading, rebooting, verifying, completed, failed
  error_message?: string;
  retry_count: number;
  max_retries: number;
  started_at?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

function mapFirmwareStatus(status: string): VersionStatus {
  const map: Record<string, VersionStatus> = {
    current: 'current',
    deprecated: 'deprecated',
    beta: 'beta',
    archived: 'archived',
  };
  return map[status] || 'current';
}

function mapFirmware(bf: BackendFirmwareVersion): SoftwareVersion {
  return {
    id: bf.id,
    versionName: `${bf.product_class || ''} ${bf.version}`.trim(),
    versionCode: bf.version,
    deviceType: bf.product_class || '',
    vendor: bf.compatible_oui?.[0] || '',
    releaseDate: bf.created_at,
    status: mapFirmwareStatus(bf.status),
    fileSize: bf.file_size,
    checksum: '',
    downloadUrl: bf.minio_path || '',
    releaseNotes: bf.release_notes || '',
    minHardwareVersion: '',
    features: [],
    bugFixes: [],
  };
}

function mapFirmwareListResponse(
  resp: BackendListResponse<BackendFirmwareVersion>
): PageResponse<SoftwareVersion> {
  return {
    items: (resp.items || []).map(mapFirmware),
    total: resp.total,
    page: resp.page,
    pageSize: resp.page_size,
  };
}

// Aggregate backend upgrade tasks by batch_id into frontend UpgradePlan
function aggregateTasksIntoPlan(
  batchId: string,
  tasks: BackendUpgradeTask[]
): UpgradePlan {
  const completed = tasks.filter((t) => t.status === 'completed').length;
  const failed = tasks.filter((t) => t.status === 'failed').length;
  const total = tasks.length;
  const progress =
    total > 0 ? Math.round(((completed + failed) / total) * 100) : 0;
  const firstTask = tasks[0];

  let status: UpgradePlanStatus = 'pending';
  if (tasks.every((t) => t.status === 'completed')) status = 'success';
  else if (tasks.every((t) => t.status === 'failed')) status = 'failed';
  else if (
    tasks.some((t) =>
      ['downloading', 'rebooting', 'verifying'].includes(t.status)
    )
  )
    status = 'running';
  else if (completed > 0 || failed > 0) status = 'running';

  return {
    id: batchId,
    planName: `Batch ${batchId.substring(0, 8)}`,
    targetVersionId: firstTask?.firmware_id || '',
    targetVersionCode: '',
    deviceSns: [],
    status,
    progress,
    successCount: completed,
    failCount: failed,
    totalCount: total,
    startTime: firstTask?.started_at,
    endTime: tasks.find((t) => t.completed_at)?.completed_at,
    createdAt: firstTask?.created_at || '',
    updatedAt: tasks.reduce(
      (latest, t) => (t.updated_at > latest ? t.updated_at : latest),
      ''
    ),
    creator: '',
    preCheckRequired: false,
    rollbackEnabled: false,
  };
}

export const softwareApi = {
  async getVersions(
    params: { deviceType?: string; status?: string; vendor?: string } & PageRequest
  ): Promise<PageResponse<SoftwareVersion>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.deviceType) query.product_class = params.deviceType;
    if (params.status) query.status = params.status;

    const { data } = await http.get<BackendListResponse<BackendFirmwareVersion>>(
      '/firmware',
      { params: query }
    );
    return mapFirmwareListResponse(data);
  },

  async getVersionById(id: string): Promise<SoftwareVersion | null> {
    try {
      const { data } = await http.get<BackendFirmwareVersion>(`/firmware/${id}`);
      return mapFirmware(data);
    } catch {
      return null;
    }
  },

  async uploadVersion(
    data: Omit<SoftwareVersion, 'id' | 'releaseDate'>
  ): Promise<SoftwareVersion> {
    // For real API, multipart upload needs a File object.
    // When called from the hook with mock-style data (no File),
    // we construct a minimal request. Full multipart upload uses uploadFirmware().
    const formData = new FormData();
    formData.append('carrier', 'cmcc');
    formData.append('version', data.versionCode);
    if (data.deviceType) formData.append('product_class', data.deviceType);
    if (data.releaseNotes) formData.append('release_notes', data.releaseNotes);

    const { data: bf } = await http.post<BackendFirmwareVersion>(
      '/firmware',
      formData,
      {
        headers: { 'Content-Type': 'multipart/form-data' },
        timeout: 120000,
      }
    );
    return mapFirmware(bf);
  },

  // Full multipart upload with File object
  async uploadFirmware(
    file: File,
    metadata: {
      carrier: string;
      version: string;
      productClass?: string;
      releaseNotes?: string;
    }
  ): Promise<SoftwareVersion> {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('carrier', metadata.carrier);
    formData.append('version', metadata.version);
    if (metadata.productClass)
      formData.append('product_class', metadata.productClass);
    if (metadata.releaseNotes)
      formData.append('release_notes', metadata.releaseNotes);

    const { data } = await http.post<BackendFirmwareVersion>(
      '/firmware',
      formData,
      {
        headers: { 'Content-Type': 'multipart/form-data' },
        timeout: 120000,
      }
    );
    return mapFirmware(data);
  },

  async deleteVersions(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.delete(`/firmware/${id}`);
    }
  },

  async getUpgradePlans(
    params: { status?: string } & PageRequest
  ): Promise<PageResponse<UpgradePlan>> {
    const query: Record<string, unknown> = {
      page: 1,
      pageSize: 200, // Fetch all tasks then aggregate by batch
    };
    if (params.status) query.status = params.status;

    const { data } = await http.get<BackendListResponse<BackendUpgradeTask>>(
      '/upgrade-tasks',
      { params: query }
    );

    // Group tasks by batch_id and aggregate into UpgradePlans
    const batchMap = new Map<string, BackendUpgradeTask[]>();
    for (const task of data.items || []) {
      const key = task.batch_id || task.id; // Single tasks use their own id
      if (!batchMap.has(key)) batchMap.set(key, []);
      batchMap.get(key)!.push(task);
    }

    const plans = Array.from(batchMap.entries()).map(([batchId, tasks]) =>
      aggregateTasksIntoPlan(batchId, tasks)
    );

    // Client-side pagination
    const start = (params.page - 1) * params.pageSize;
    const paged = plans.slice(start, start + params.pageSize);

    return {
      items: paged,
      total: plans.length,
      page: params.page,
      pageSize: params.pageSize,
    };
  },

  async getUpgradePlanById(id: string): Promise<UpgradePlan | null> {
    try {
      // Try fetching tasks by batch_id
      const { data } = await http.get<BackendListResponse<BackendUpgradeTask>>(
        '/upgrade-tasks',
        { params: { batch_id: id, page: 1, pageSize: 200 } }
      );

      const tasks = data.items || [];
      if (tasks.length === 0) {
        // Maybe it's a single task
        const { data: task } = await http.get<BackendUpgradeTask>(
          `/upgrade-tasks/${id}`
        );
        return aggregateTasksIntoPlan(id, [task]);
      }
      return aggregateTasksIntoPlan(id, tasks);
    } catch {
      return null;
    }
  },

  async createUpgradePlan(
    data: Omit<
      UpgradePlan,
      | 'id'
      | 'status'
      | 'progress'
      | 'successCount'
      | 'failCount'
      | 'createdAt'
      | 'updatedAt'
    >
  ): Promise<UpgradePlan> {
    // Use batch upgrade endpoint
    const { data: resp } = await http.post<{
      tasks: BackendUpgradeTask[];
      count: number;
    }>('/upgrade-tasks/batch', {
      device_ids: data.deviceSns, // frontend uses SNs but backend expects IDs
      firmware_id: data.targetVersionId,
      concurrency: 5,
    });

    const batchId =
      resp.tasks?.[0]?.batch_id || resp.tasks?.[0]?.id || 'unknown';
    return aggregateTasksIntoPlan(batchId, resp.tasks || []);
  },

  // Not yet implemented in backend — delegate to mock
  cancelUpgradePlan: softwareService.cancelUpgradePlan.bind(softwareService),
  precheck: softwareService.precheck.bind(softwareService),
};
