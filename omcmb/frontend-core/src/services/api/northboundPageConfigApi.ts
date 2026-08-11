import http from '../http';

export type NorthboundPageConfigDomain = 'CM' | 'PM' | 'MR' | 'LOG' | 'INVENTORY';
export type NorthboundPageConfigFormat = 'CSV' | 'XML' | 'TXT';
export type NorthboundPageConfigPeriod = '15M' | '60M' | '24H' | '7D' | '1MO';
export type NorthboundPageConfigCompressionFormat = 'zip' | 'gz';
export type NorthboundPageConfigStatus = 'normal' | 'terminated';
export type NorthboundFieldSupportStatus = 'supported' | 'partial' | 'unsupported';
export type NorthboundProfileKind = 'file' | 'inventory';
export type NorthboundRunStatus = 'running' | 'success' | 'failed' | 'terminated';
export type NorthboundDeliveryScope = 'file' | 'inventory' | 'socket';
export type NorthboundDeliveryProtocol = 'FTP' | 'SFTP';
export type NorthboundDeliveryAuthMode = 'PASSWORD' | 'PRIVATE_KEY';
export type NorthboundDeliveryHostKeyPolicy = 'INSECURE' | 'FINGERPRINT';
export type NorthboundEventArtifactType = 'file' | 'message' | 'json';

export interface NorthboundScenarioObject {
  code: string;
  tech?: string;
  profile?: string;
}

export interface NorthboundFileGroup {
  id: string;
  domain: NorthboundPageConfigDomain;
  format: NorthboundPageConfigFormat;
  period: NorthboundPageConfigPeriod;
  start_minute: number;
  start_time?: string;
  path_template: string;
  file_name_template: string;
  csv_separator?: string;
  compression_enabled: boolean;
  compression_format?: NorthboundPageConfigCompressionFormat;
  objects: NorthboundScenarioObject[];
  selected_fields?: string[];
}

export interface NorthboundFileProfile {
  id: string;
  code: string;
  name: string;
  vendor: string;
  scenario_name: string;
  scenario_name_en: string;
  description: string;
  flags: string[];
  enabled: boolean;
  status: NorthboundPageConfigStatus;
  groups: NorthboundFileGroup[];
  created_at?: string;
  updated_at?: string;
}

export interface NorthboundFileGroupPreview {
  group_id: string;
  domain: NorthboundPageConfigDomain;
  format: NorthboundPageConfigFormat;
  period: NorthboundPageConfigPeriod;
  path_template: string;
  file_name_template: string;
  csv_separator?: string;
  preview_path: string;
  preview_file_name: string;
  compression_enabled: boolean;
  compression_format?: NorthboundPageConfigCompressionFormat;
  preview_artifact_name: string;
  unsupported_token_keys?: string[];
  warnings?: string[];
}

export interface NorthboundFileProfilePreview {
  profile_code: string;
  items: NorthboundFileGroupPreview[];
}

export interface NorthboundInventoryProfile {
  id: string;
  code: string;
  name: string;
  object_code: string;
  tech: string;
  period: NorthboundPageConfigPeriod;
  start_minute: number;
  path_template: string;
  file_name_template: string;
  compression_enabled: boolean;
  compression_format?: NorthboundPageConfigCompressionFormat;
  enabled: boolean;
  status: NorthboundPageConfigStatus;
  fields?: NorthboundInventoryField[];
  created_at?: string;
  updated_at?: string;
}

export interface NorthboundInventoryField {
  key: string;
  output_alias: string;
  system_field: string;
  source: string;
  data_type: string;
  renderer: string;
  enabled: boolean;
}

export interface NorthboundFieldDefinition {
  key: string;
  domain: NorthboundPageConfigDomain;
  object_code: string;
  scope?: string;
  tech?: string;
  output_alias: string;
  system_field: string;
  source: string;
  data_type: string;
  renderer: string;
  product_class?: string;
  metric_type?: string;
  statis_type?: string;
  unit?: string;
  cn_name?: string;
  support_status: NorthboundFieldSupportStatus;
}

export interface NorthboundPageConfigOverview {
  file_profiles: number;
  inventory_profiles: number;
  field_definitions: number;
  supported_domains: NorthboundPageConfigDomain[];
  supported_periods: NorthboundPageConfigPeriod[];
}

export interface NorthboundValidateRequest {
  profile_kind: 'file' | 'inventory' | 'socket' | 'snmp' | 'api';
  domain: NorthboundPageConfigDomain;
  format: NorthboundPageConfigFormat;
  period: NorthboundPageConfigPeriod;
  compression_enabled?: boolean;
  compression_format?: NorthboundPageConfigCompressionFormat;
  objects?: NorthboundScenarioObject[];
  fields?: string[];
  metric_paths?: string[];
}

export type NorthboundUpdateFileProfileRequest = Partial<
  Pick<
    NorthboundFileProfile,
    | 'name'
    | 'vendor'
    | 'scenario_name'
    | 'scenario_name_en'
    | 'description'
    | 'flags'
    | 'enabled'
    | 'status'
    | 'groups'
  >
>;

export type NorthboundUpdateInventoryProfileRequest = Partial<
  Pick<
    NorthboundInventoryProfile,
    | 'name'
    | 'object_code'
    | 'tech'
    | 'period'
    | 'start_minute'
    | 'path_template'
    | 'file_name_template'
    | 'compression_enabled'
    | 'compression_format'
    | 'enabled'
    | 'status'
    | 'fields'
  >
>;

export interface NorthboundValidationResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
}

export interface NorthboundRunProfileRequest {
  group_id?: string;
  window_start?: string;
  window_end?: string;
  limit?: number;
}

export interface NorthboundFileRun {
  id: string;
  profile_kind: NorthboundProfileKind;
  profile_code: string;
  group_id: string;
  domain: NorthboundPageConfigDomain;
  object_code: string;
  status: NorthboundRunStatus;
  window_start?: string;
  window_end?: string;
  artifact_path: string;
  artifact_name: string;
  artifact_content?: string;
  artifact_size: number;
  row_count: number;
  compression_enabled: boolean;
  compression_format?: NorthboundPageConfigCompressionFormat;
  error_message?: string;
  summary?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface NorthboundRunProfileResponse {
  profile_kind: NorthboundProfileKind;
  profile_code: string;
  items: NorthboundFileRun[];
  total: number;
}

export interface NorthboundDeliveryTarget {
  id?: string;
  scope: NorthboundDeliveryScope;
  owner_code: string;
  key: string;
  name: string;
  enabled: boolean;
  protocol: NorthboundDeliveryProtocol;
  host: string;
  port: number;
  username: string;
  credential?: string;
  credential_set?: boolean;
  auth_mode: NorthboundDeliveryAuthMode;
  remote_root: string;
  retry_times: number;
  timeout_seconds: number;
  passive_mode: boolean;
  host_key_policy?: NorthboundDeliveryHostKeyPolicy;
  host_key_fingerprint?: string;
}

export interface NorthboundDeliveryTargetQuery {
  scope?: NorthboundDeliveryScope;
  owner_code?: string;
}

export interface NorthboundReplaceDeliveryTargetsRequest {
  scope: NorthboundDeliveryScope;
  owner_code?: string;
  items: NorthboundDeliveryTarget[];
}

export interface NorthboundSNMPAlarmField {
  order: number;
  field: string;
  oid: string;
  type: string;
  source: string;
  required: boolean;
}

export interface NorthboundSNMPAlarmTarget {
  id?: string;
  key: string;
  name: string;
  enabled: boolean;
  version: 'v2' | 'v3';
  notification_type: 'Trap' | 'Inform';
  listen_ip: string;
  listen_port: number;
  target_host: string;
  target_port: number;
  community?: string;
  community_set?: boolean;
  security_name?: string;
  auth_protocol?: string;
  auth_credential?: string;
  auth_credential_set?: boolean;
  priv_protocol?: string;
  priv_credential?: string;
  priv_credential_set?: boolean;
  clear_severity_policy: string;
  mib_query_enabled: boolean;
  timeout_seconds: number;
  retries: number;
  mib_fields?: NorthboundSNMPAlarmField[];
}

export interface NorthboundSocketAccount {
  key: string;
  enabled: boolean;
  channel: string;
  username: string;
  type: 'msg' | 'ftp';
  credential?: string;
  credential_set?: boolean;
  purpose: string;
}

export interface NorthboundSocketAlarmConfig {
  id?: string;
  key: string;
  name: string;
  enabled: boolean;
  profile: 'CTCC' | 'CUCC';
  mode: 'server';
  listen_ip: string;
  listen_port: number;
  max_clients: number;
  realtime_push_enabled: boolean;
  client_sync_enabled: boolean;
  heartbeat_seconds: number;
  heartbeat_times: number;
  idle_timeout_seconds: number;
  accounts: NorthboundSocketAccount[];
}

export interface NorthboundAPIConfig {
  id?: string;
  key: string;
  name: string;
  method: 'GET' | 'POST' | 'PUT' | 'DELETE';
  path: string;
  kind: string;
  data_type: string;
  enabled: boolean;
  old_system_supported: boolean;
  current_supported: boolean;
  source: string;
  response_contract?: Record<string, unknown>;
}

export interface NorthboundAPIUser {
  id?: string;
  username: string;
  enabled: boolean;
  password?: string;
  password_set?: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface NorthboundReplaceAPIUsersRequest {
  items: NorthboundAPIUser[];
}

export interface NorthboundPageConfigEvent {
  id: string;
  capability: 'file' | 'inventory' | 'delivery' | 'snmp' | 'socket' | 'api';
  owner_code: string;
  target_key: string;
  event_type: string;
  status: NorthboundRunStatus;
  artifact_type: NorthboundEventArtifactType;
  artifact_name: string;
  artifact_path: string;
  payload?: string;
  payload_content_type: string;
  error_message?: string;
  summary?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface NorthboundFieldQuery {
  domain?: NorthboundPageConfigDomain;
  object?: string;
  tech?: string;
  profile?: string;
}

export interface NorthboundRunQuery {
  profile_kind?: NorthboundProfileKind;
  profile_code?: string;
  status?: NorthboundRunStatus;
  latest_per_profile?: boolean;
  limit?: number;
  offset?: number;
}

export interface NorthboundEventQuery {
  capability?: NorthboundPageConfigEvent['capability'];
  owner_code?: string;
  target_key?: string;
  event_type?: string;
  status?: NorthboundRunStatus;
  limit?: number;
  offset?: number;
  include_payload?: boolean;
}

interface ListResponse<T> {
  items: T[];
  total: number;
}

export const northboundPageConfigApi = {
  async getOverview(): Promise<NorthboundPageConfigOverview> {
    const { data } = await http.get<NorthboundPageConfigOverview>('/northbound/page-config/overview');
    return data;
  },

  async getFileProfiles(): Promise<ListResponse<NorthboundFileProfile>> {
    const { data } = await http.get<ListResponse<NorthboundFileProfile>>(
      '/northbound/page-config/file/profiles',
    );
    return data;
  },

  async createFileProfile(req: NorthboundFileProfile): Promise<NorthboundFileProfile> {
    const { data } = await http.post<NorthboundFileProfile>(
      '/northbound/page-config/file/profiles',
      req,
    );
    return data;
  },

  async updateFileProfile(
    id: string,
    req: NorthboundUpdateFileProfileRequest,
  ): Promise<NorthboundFileProfile> {
    const { data } = await http.put<NorthboundFileProfile>(
      `/northbound/page-config/file/profiles/${id}`,
      req,
    );
    return data;
  },

  async previewFileProfile(id: string): Promise<NorthboundFileProfilePreview> {
    const { data } = await http.get<NorthboundFileProfilePreview>(
      `/northbound/page-config/file/profiles/${id}/preview`,
    );
    return data;
  },

  async runFileProfile(
    id: string,
    req: NorthboundRunProfileRequest = {},
  ): Promise<NorthboundRunProfileResponse> {
    const { data } = await http.post<NorthboundRunProfileResponse>(
      `/northbound/page-config/file/profiles/${id}/run`,
      req,
    );
    return data;
  },

  async getInventoryProfiles(): Promise<ListResponse<NorthboundInventoryProfile>> {
    const { data } = await http.get<ListResponse<NorthboundInventoryProfile>>(
      '/northbound/page-config/inventory/profiles',
    );
    return data;
  },

  async updateInventoryProfile(
    id: string,
    req: NorthboundUpdateInventoryProfileRequest,
  ): Promise<NorthboundInventoryProfile> {
    const { data } = await http.put<NorthboundInventoryProfile>(
      `/northbound/page-config/inventory/profiles/${id}`,
      req,
    );
    return data;
  },

  async runInventoryProfile(
    id: string,
    req: NorthboundRunProfileRequest = {},
  ): Promise<NorthboundFileRun> {
    const { data } = await http.post<NorthboundFileRun>(
      `/northbound/page-config/inventory/profiles/${id}/run`,
      req,
    );
    return data;
  },

  async listRuns(query: NorthboundRunQuery = {}): Promise<ListResponse<NorthboundFileRun>> {
    const { data } = await http.get<ListResponse<NorthboundFileRun>>(
      '/northbound/page-config/runs',
      { params: query },
    );
    return data;
  },

  async getRun(id: string): Promise<NorthboundFileRun> {
    const { data } = await http.get<NorthboundFileRun>(
      `/northbound/page-config/runs/${id}`,
    );
    return data;
  },

  async downloadRun(id: string): Promise<Blob> {
    const response = await http.get<Blob>(
      `/northbound/page-config/runs/${id}/download`,
      { responseType: 'blob' },
    );
    return response.data;
  },

  async getDeliveryTargets(
    query: NorthboundDeliveryTargetQuery = {},
  ): Promise<ListResponse<NorthboundDeliveryTarget>> {
    const { data } = await http.get<ListResponse<NorthboundDeliveryTarget>>(
      '/northbound/page-config/delivery/targets',
      { params: query },
    );
    return data;
  },

  async replaceDeliveryTargets(
    req: NorthboundReplaceDeliveryTargetsRequest,
  ): Promise<ListResponse<NorthboundDeliveryTarget>> {
    const { data } = await http.put<ListResponse<NorthboundDeliveryTarget>>(
      '/northbound/page-config/delivery/targets',
      req,
    );
    return data;
  },

  async testDeliveryTarget(req: NorthboundDeliveryTarget): Promise<NorthboundPageConfigEvent> {
    const { data } = await http.post<NorthboundPageConfigEvent>(
      '/northbound/page-config/delivery/targets/test',
      req,
    );
    return data;
  },

  async getSNMPAlarmTargets(): Promise<ListResponse<NorthboundSNMPAlarmTarget> & { mib_fields?: NorthboundSNMPAlarmField[] }> {
    const { data } = await http.get<ListResponse<NorthboundSNMPAlarmTarget> & { mib_fields?: NorthboundSNMPAlarmField[] }>(
      '/northbound/page-config/alarm/snmp/targets',
    );
    return data;
  },

  async updateSNMPAlarmTarget(
    key: string,
    req: NorthboundSNMPAlarmTarget,
  ): Promise<NorthboundSNMPAlarmTarget> {
    const { data } = await http.put<NorthboundSNMPAlarmTarget>(
      `/northbound/page-config/alarm/snmp/targets/${key}`,
      req,
    );
    return data;
  },

  async testSNMPAlarmTarget(key: string): Promise<NorthboundPageConfigEvent> {
    const { data } = await http.post<NorthboundPageConfigEvent>(
      `/northbound/page-config/alarm/snmp/targets/${key}/test`,
    );
    return data;
  },

  async getSocketAlarmConfigs(): Promise<ListResponse<NorthboundSocketAlarmConfig>> {
    const { data } = await http.get<ListResponse<NorthboundSocketAlarmConfig>>(
      '/northbound/page-config/alarm/socket/configs',
    );
    return data;
  },

  async updateSocketAlarmConfig(
    key: string,
    req: NorthboundSocketAlarmConfig,
  ): Promise<NorthboundSocketAlarmConfig> {
    const { data } = await http.put<NorthboundSocketAlarmConfig>(
      `/northbound/page-config/alarm/socket/configs/${key}`,
      req,
    );
    return data;
  },

  async testSocketAlarmConfig(key: string): Promise<NorthboundPageConfigEvent> {
    const { data } = await http.post<NorthboundPageConfigEvent>(
      `/northbound/page-config/alarm/socket/configs/${key}/test`,
    );
    return data;
  },

  async getAPIConfigs(): Promise<ListResponse<NorthboundAPIConfig>> {
    const { data } = await http.get<ListResponse<NorthboundAPIConfig>>(
      '/northbound/page-config/api/configs',
    );
    return data;
  },

  async updateAPIConfig(key: string, enabled: boolean): Promise<NorthboundAPIConfig> {
    const { data } = await http.put<NorthboundAPIConfig>(
      `/northbound/page-config/api/configs/${key}`,
      { enabled },
    );
    return data;
  },

  async updateAllAPIConfigs(enabled: boolean): Promise<ListResponse<NorthboundAPIConfig>> {
    const { data } = await http.put<ListResponse<NorthboundAPIConfig>>(
      '/northbound/page-config/api/configs',
      { enabled },
    );
    return data;
  },

  async testAPIConfig(key: string): Promise<NorthboundPageConfigEvent> {
    const { data } = await http.post<NorthboundPageConfigEvent>(
      `/northbound/page-config/api/configs/${key}/test`,
    );
    return data;
  },

  async getAPIUsers(): Promise<ListResponse<NorthboundAPIUser>> {
    const { data } = await http.get<ListResponse<NorthboundAPIUser>>(
      '/northbound/page-config/api/users',
    );
    return data;
  },

  async replaceAPIUsers(
    req: NorthboundReplaceAPIUsersRequest,
  ): Promise<ListResponse<NorthboundAPIUser>> {
    const { data } = await http.put<ListResponse<NorthboundAPIUser>>(
      '/northbound/page-config/api/users',
      req,
    );
    return data;
  },

  async listEvents(query: NorthboundEventQuery = {}): Promise<ListResponse<NorthboundPageConfigEvent>> {
    const { data } = await http.get<ListResponse<NorthboundPageConfigEvent>>(
      '/northbound/page-config/events',
      { params: query },
    );
    return data;
  },

  async getEvent(id: string): Promise<NorthboundPageConfigEvent> {
    const { data } = await http.get<NorthboundPageConfigEvent>(
      `/northbound/page-config/events/${id}`,
    );
    return data;
  },

  async getFields(query: NorthboundFieldQuery = {}): Promise<ListResponse<NorthboundFieldDefinition>> {
    const { data } = await http.get<ListResponse<NorthboundFieldDefinition>>(
      '/northbound/page-config/fields',
      { params: query },
    );
    return data;
  },

  async validate(req: NorthboundValidateRequest): Promise<NorthboundValidationResult> {
    const { data } = await http.post<NorthboundValidationResult>(
      '/northbound/page-config/validate',
      req,
    );
    return data;
  },
};
