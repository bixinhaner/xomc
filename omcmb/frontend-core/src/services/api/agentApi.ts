import http from '../http';
import type {
  AgentAdminConfig,
  AgentAdminConfigUpdate,
  AgentProvisionResult,
  AgentRuntimeServerConfig,
} from '../../types/agentConfig';

export interface DelegationTokenResponse {
  token: string;
  expires_at: string;
}

export interface AgentApi {
  requestDelegationToken(): Promise<DelegationTokenResponse>;
  getRuntimeConfig(): Promise<AgentRuntimeServerConfig>;
  getAdminConfig(): Promise<AgentAdminConfig>;
  saveAdminConfig(payload: AgentAdminConfigUpdate): Promise<AgentAdminConfig>;
  testAdminConfig(payload: AgentAdminConfigUpdate): Promise<AgentProvisionResult>;
  syncAdminConfig(payload: AgentAdminConfigUpdate): Promise<AgentAdminConfig>;
}

export const agentApi: AgentApi = {
  async requestDelegationToken() {
    const { data } = await http.post<DelegationTokenResponse>('/agent/delegation');
    return data;
  },
  async getRuntimeConfig() {
    const { data } = await http.get<AgentRuntimeServerConfig>('/agent/config');
    return data;
  },
  async getAdminConfig() {
    const { data } = await http.get<AgentAdminConfig>('/admin/agent-config');
    return data;
  },
  async saveAdminConfig(payload) {
    const { data } = await http.post<AgentAdminConfig>('/admin/agent-config', payload);
    return data;
  },
  async testAdminConfig(payload) {
    const { data } = await http.post<AgentProvisionResult>('/admin/agent-config/test', payload);
    return data;
  },
  async syncAdminConfig(payload) {
    const { data } = await http.post<AgentAdminConfig>('/admin/agent-config/sync', payload);
    return data;
  },
};
