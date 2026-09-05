import http from '../http';
import type {
  AgentAdminConfig,
  AgentAdminConfigUpdate,
  AgentConversation,
  AgentProvisionResult,
  AgentRuntimeServerConfig,
  AgentVisibilityConfig,
} from '../../types/agentConfig';
import type { AgentArtifactRef, AgentAttachmentRef, AgentPanelMessage } from '../../agentkit';
import { useMock } from '../apiSwitch';

export interface AgentConversationHistory {
  conversationId: string;
  messages: Array<Omit<AgentPanelMessage, 'createdAt'> & { createdAt: string | number }>;
}

export interface AgentApi {
  getVisibilityConfig(): Promise<AgentVisibilityConfig>;
  getRuntimeConfig(): Promise<AgentRuntimeServerConfig>;
  getConversation(): Promise<AgentConversation>;
  startConversation(): Promise<AgentConversation>;
  getConversationMessages(): Promise<AgentConversationHistory>;
  uploadAttachment(file: File): Promise<AgentAttachmentRef>;
  removeAttachment(attachmentId: string): Promise<void>;
  cancelRun(runId: string): Promise<boolean>;
  getArtifactContent(artifact: AgentArtifactRef, disposition: 'inline' | 'attachment'): Promise<Blob>;
  getAdminConfig(): Promise<AgentAdminConfig>;
  saveAdminConfig(payload: AgentAdminConfigUpdate): Promise<AgentAdminConfig>;
  testAdminConfig(payload: AgentAdminConfigUpdate): Promise<AgentProvisionResult>;
  syncAdminConfig(payload: AgentAdminConfigUpdate): Promise<AgentAdminConfig>;
}

export const agentApi: AgentApi = {
  async getVisibilityConfig() {
    if (useMock) {
      return { visible: true, enabled: true, status: 'connected', lastValidatedAt: new Date().toISOString(), lastError: '' };
    }
    const { data } = await http.get<AgentVisibilityConfig>('/agent/visibility');
    return data;
  },
  async getRuntimeConfig() {
    if (useMock) {
      return {
        visible: true, enabled: true, endpoint: '/api/v1/agent/runtime', connectorId: 'mock-xomc',
        status: 'connected', lastValidatedAt: new Date().toISOString(), lastError: '', configuredSource: 'server',
        policy: { allowedMethods: ['GET'], blockedPathPrefixes: ['/api/v1/auth/*', '/api/v1/admin/*'], toolTimeoutSeconds: 30, maxResponseBytes: 1_048_576 },
      };
    }
    const { data } = await http.get<AgentRuntimeServerConfig>('/agent/config');
    return data;
  },
  async getConversation() {
    if (useMock) return { conversationId: 'mock-agent-conversation' };
    const { data } = await http.get<AgentConversation>('/agent/conversation');
    return data;
  },
  async startConversation() {
    if (useMock) return { conversationId: `mock-agent-conversation-${Date.now()}` };
    const { data } = await http.post<AgentConversation>('/agent/conversation');
    return data;
  },
  async getConversationMessages() {
    if (useMock) return { conversationId: 'mock-agent-conversation', messages: [] };
    const { data } = await http.get<AgentConversationHistory>('/agent/conversation/messages');
    return data;
  },
  async uploadAttachment(file) {
    const form = new FormData();
    form.append('file', file);
    const { data } = await http.post<{ attachment: AgentAttachmentRef }>('/agent/attachments', form, {
      headers: { 'Content-Type': undefined },
      timeout: 60000,
    });
    return data.attachment;
  },
  async removeAttachment(attachmentId) {
    await http.delete(`/agent/attachments/${encodeURIComponent(attachmentId)}`);
  },
  async cancelRun(runId) {
    const { data } = await http.post<{ cancelled: boolean }>(`/agent/runs/${encodeURIComponent(runId)}/cancel`);
    return data.cancelled;
  },
  async getArtifactContent(artifact, disposition) {
    const { data } = await http.get<Blob>(`/agent/artifacts/${encodeURIComponent(artifact.artifactId)}/content`, {
      params: { disposition },
      responseType: 'blob',
      timeout: 60000,
    });
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
