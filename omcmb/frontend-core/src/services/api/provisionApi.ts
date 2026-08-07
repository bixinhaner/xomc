import http from '../http';
import type { PageRequest, PageResponse } from '../../types/pagination';
import { filenameFromContentDisposition, saveBlob } from '../../utils/saveBlob';

// --- Backend response types ---

interface BackendProvisioningTask {
  id: string;
  device_id: string;
  serial_number?: string;
  template_id: string | null;
  policy_id?: string | null;
  xml_file_id?: string | null;
  device_task_id?: string | null;
  status: string;
  current_step: number;
  current_step_name?: string;
  total_steps: number;
  error_message: string;
  retry_count: number;
  max_retries: number;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
}

// --- Frontend types ---

export interface ProvisioningTask {
  id: string;
  deviceId: string;
  serialNumber: string;
  templateId: string | null;
  policyId: string | null;
  xmlFileId: string | null;
  deviceTaskId: string | null;
  status: string;
  currentStep: number;
  currentStepName: string;
  totalSteps: number;
  errorMessage: string;
  retryCount: number;
  maxRetries: number;
  startedAt: string | null;
  completedAt: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface CreateProvisioningTaskRequest {
  deviceId: string;
}

interface BackendPlugAndPlayPolicy {
  id: string;
  name: string;
  enabled: boolean;
  product_class: string;
  product_classes?: string[];
  execute_type: 'auto' | 'manual';
  priority: number;
  upgrade_enabled: boolean;
  target_version?: string;
  license_enabled: boolean;
  self_config_enabled: boolean;
  config: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface PlugAndPlayPolicy {
  id: string;
  name: string;
  enabled: boolean;
  productClass: string;
  productClasses: string[];
  executeType: 'auto' | 'manual';
  priority: number;
  upgradeEnabled: boolean;
  targetVersion: string;
  licenseEnabled: boolean;
  selfConfigEnabled: boolean;
  config: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export type SavePlugAndPlayPolicyRequest = Omit<PlugAndPlayPolicy, 'id' | 'createdAt' | 'updatedAt'>;

interface BackendDetectedDevice {
  id: string;
  serial_number: string;
  device_name?: string;
  firmware_version?: string;
  product_class: string;
  group_name?: string;
  is_online: boolean;
}

export interface PlugAndPlayDevice {
  id: string;
  serialNumber: string;
  deviceName: string;
  firmwareVersion: string;
  productClass: string;
  groupName: string;
  isOnline: boolean;
}

/**
 * UI 状态码（即插即用「执行状态」列表使用的简化状态机）：
 * 0-成功 1-失败 2-执行中 3-未执行 4-跳过。
 * 后端 ProvisioningState（discovered/identifying/matching/configuring/verifying/
 * discovering/syncing/completed/failed）映射到此简化态。
 */
export type ProvisioningTaskStatusCode = '0' | '1' | '2' | '3' | '4';

/**
 * 即插即用「执行状态」表的视图模型。后端 ProvisioningTask 字段较少
 * （无 productClass/policyName/executeType/licenseFile 等业务列），缺失字段在
 * mapTaskToExecuteView 中以空串占位，避免页面渲染崩溃。
 */
export interface ProvisioningExecuteView {
  taskId: string;
  deviceId: string;
  policyId: string;
  serialNumber: string;
  status: ProvisioningTaskStatusCode;
  startTime: string;
  endTime: string;
  /** 「Step N/M」形式的进度，由 currentStep/totalSteps 派生。 */
  executeProcedure: string;
  currentStep: number;
  totalSteps: number;
  currentStepName: string;
  xmlFileId: string | null;
  failureReason: string;
  retryCount: number;
  maxRetries: number;
}

/** 后端生命周期状态 → UI 简化状态码。 */
export function provisioningStatusCode(status: string): ProvisioningTaskStatusCode {
  switch (status) {
    case 'completed':
      return '0';
    case 'failed':
      return '1';
    default:
      // discovered / identifying / matching / configuring / verifying /
      // discovering / syncing —— 均视为「执行中」。
      return '2';
  }
}

/** 后端 ProvisioningTask → 即插即用执行状态视图模型。 */
export function mapTaskToExecuteView(t: ProvisioningTask): ProvisioningExecuteView {
  const procedure =
    t.totalSteps > 0 ? `${t.currentStep}/${t.totalSteps}` : '';
  return {
    taskId: t.id,
    deviceId: t.deviceId,
    policyId: t.policyId ?? '',
    serialNumber: t.serialNumber,
    status: provisioningStatusCode(t.status),
    startTime: t.startedAt ?? '',
    endTime: t.completedAt ?? '',
    executeProcedure: procedure,
    currentStep: t.currentStep,
    totalSteps: t.totalSteps,
    currentStepName: t.currentStepName,
    xmlFileId: t.xmlFileId,
    failureReason: t.errorMessage,
    retryCount: t.retryCount,
    maxRetries: t.maxRetries,
  };
}

// --- Mapping functions ---

function mapBackendTask(t: BackendProvisioningTask): ProvisioningTask {
  return {
    id: t.id,
    deviceId: t.device_id,
    serialNumber: t.serial_number ?? '',
    templateId: t.template_id,
    policyId: t.policy_id ?? null,
    xmlFileId: t.xml_file_id ?? null,
    deviceTaskId: t.device_task_id ?? null,
    status: t.status,
    currentStep: t.current_step,
    currentStepName: t.current_step_name ?? '',
    totalSteps: t.total_steps,
    errorMessage: t.error_message || '',
    retryCount: t.retry_count,
    maxRetries: t.max_retries,
    startedAt: t.started_at,
    completedAt: t.completed_at,
    createdAt: t.created_at,
    updatedAt: t.updated_at,
  };
}

function mapPolicy(p: BackendPlugAndPlayPolicy): PlugAndPlayPolicy {
  const productClasses = p.product_classes?.length
    ? p.product_classes
    : [p.product_class].filter(Boolean);
  return {
    id: p.id, name: p.name, enabled: p.enabled, productClass: productClasses[0] ?? '',
    productClasses,
    executeType: p.execute_type, priority: p.priority, upgradeEnabled: p.upgrade_enabled,
    targetVersion: p.target_version ?? '', licenseEnabled: p.license_enabled,
    selfConfigEnabled: p.self_config_enabled, config: p.config ?? {},
    createdAt: p.created_at, updatedAt: p.updated_at,
  };
}

function policyBody(p: SavePlugAndPlayPolicyRequest) {
  const productClasses = p.productClasses.length ? p.productClasses : [p.productClass].filter(Boolean);
  return {
    name: p.name, enabled: p.enabled, product_class: productClasses[0] ?? '', product_classes: productClasses,
    execute_type: p.executeType, priority: p.priority,
    upgrade_enabled: p.upgradeEnabled, target_version: p.targetVersion,
    license_enabled: p.licenseEnabled, self_config_enabled: p.selfConfigEnabled,
    config: p.config,
  };
}

// --- Exported service ---

export const provisionApi = {
  async getPolicies(params: PageRequest & { productClass?: string; search?: string }): Promise<PageResponse<PlugAndPlayPolicy>> {
    const { data } = await http.get<{ items: BackendPlugAndPlayPolicy[]; total: number }>(
      '/provisioning/policies',
      { params: { page: params.page, page_size: params.pageSize, product_class: params.productClass, search: params.search } }
    );
    return { items: (data.items ?? []).map(mapPolicy), total: data.total, page: params.page, pageSize: params.pageSize };
  },

  async getPolicy(id: string): Promise<PlugAndPlayPolicy> {
    const { data } = await http.get<BackendPlugAndPlayPolicy>(`/provisioning/policies/${id}`);
    return mapPolicy(data);
  },

  async createPolicy(policy: SavePlugAndPlayPolicyRequest): Promise<PlugAndPlayPolicy> {
    const { data } = await http.post<BackendPlugAndPlayPolicy>('/provisioning/policies', policyBody(policy));
    return mapPolicy(data);
  },

  async updatePolicy(id: string, policy: SavePlugAndPlayPolicyRequest): Promise<PlugAndPlayPolicy> {
    const { data } = await http.put<BackendPlugAndPlayPolicy>(`/provisioning/policies/${id}`, policyBody(policy));
    return mapPolicy(data);
  },

  async setPolicyEnabled(id: string, enabled: boolean): Promise<PlugAndPlayPolicy> {
    const { data } = await http.patch<BackendPlugAndPlayPolicy>(`/provisioning/policies/${id}/enabled`, { enabled });
    return mapPolicy(data);
  },

  async deletePolicy(id: string): Promise<void> {
    await http.delete(`/provisioning/policies/${id}`);
  },

  async detectDevices(id: string, search?: string): Promise<PlugAndPlayDevice[]> {
    const { data } = await http.get<{ items: BackendDetectedDevice[] }>(`/provisioning/policies/${id}/devices`, {
      params: { page: 1, page_size: 100, search },
    });
    return (data.items ?? []).map((d) => ({
      id: d.id, serialNumber: d.serial_number, deviceName: d.device_name ?? '',
      firmwareVersion: d.firmware_version ?? '', productClass: d.product_class,
      groupName: d.group_name ?? '', isOnline: d.is_online,
    }));
  },

  async executePolicy(id: string, deviceIds: string[]): Promise<ProvisioningTask[]> {
    const { data } = await http.post<{ items: BackendProvisioningTask[] }>(
      `/provisioning/policies/${id}/execute`, { device_ids: deviceIds }
    );
    return (data.items ?? []).map(mapBackendTask);
  },

  async getTasks(
    params: {
      status?: string;
      deviceId?: string;
      policyOnly?: boolean;
    } & PageRequest
  ): Promise<PageResponse<ProvisioningTask>> {
    const query: Record<string, unknown> = {
      page: params.page,
      page_size: params.pageSize,
    };
    if (params.status) query.status = params.status;
    if (params.deviceId) query.device_id = params.deviceId;
    if (params.policyOnly) query.policy_only = true;

    const { data } = await http.get<{ items: BackendProvisioningTask[]; total: number }>(
      '/provisioning/tasks',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendTask),
      total: data.total,
      page: params.page,
      pageSize: params.pageSize,
    };
  },

  async getTask(id: string): Promise<ProvisioningTask> {
    const { data } = await http.get<BackendProvisioningTask>(`/provisioning/tasks/${id}`);
    return mapBackendTask(data);
  },

  async createTask(req: CreateProvisioningTaskRequest): Promise<ProvisioningTask> {
    const body = { device_id: req.deviceId };
    const { data } = await http.post<BackendProvisioningTask>('/provisioning/tasks', body);
    return mapBackendTask(data);
  },

  async retryTask(id: string): Promise<ProvisioningTask> {
    const { data } = await http.post<BackendProvisioningTask>(`/provisioning/tasks/${id}/retry`);
    return mapBackendTask(data);
  },

  async retryPolicyTask(input: { policyId: string; deviceId: string }): Promise<ProvisioningTask[]> {
    const { data } = await http.post<{ items: BackendProvisioningTask[] }>(
      `/provisioning/policies/${input.policyId}/execute`,
      { device_ids: [input.deviceId] },
    );
    return (data.items ?? []).map(mapBackendTask);
  },

  async downloadXML(id: string): Promise<void> {
    const response = await http.get(`/provisioning/xml-files/${encodeURIComponent(id)}/download`, {
      responseType: 'blob',
    });
    const filename = filenameFromContentDisposition(
      response.headers['content-disposition'] as string | undefined,
      `provisioning-${id}.xml`,
    );
    saveBlob(response.data as BlobPart, filename, 'application/xml;charset=utf-8');
  },
};
