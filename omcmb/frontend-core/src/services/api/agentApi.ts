import http from '../http';
import type {
  AgentAdminConfig,
  AgentAdminConfigUpdate,
  AgentProvisionResult,
  AgentRuntimeServerConfig,
  AgentVisibilityConfig,
} from '../../types/agentConfig';

export interface AgentApi {
  getVisibilityConfig(): Promise<AgentVisibilityConfig>;
  getRuntimeConfig(): Promise<AgentRuntimeServerConfig>;
  getAdminConfig(): Promise<AgentAdminConfig>;
  saveAdminConfig(payload: AgentAdminConfigUpdate): Promise<AgentAdminConfig>;
  testAdminConfig(payload: AgentAdminConfigUpdate): Promise<AgentProvisionResult>;
  syncAdminConfig(payload: AgentAdminConfigUpdate): Promise<AgentAdminConfig>;
}

export const agentApi: AgentApi = {
  async getVisibilityConfig() {
    const { data } = await http.get<AgentVisibilityConfig>('/agent/visibility');
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
