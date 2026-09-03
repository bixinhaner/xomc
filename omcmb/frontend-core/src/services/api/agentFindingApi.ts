import http from '../http';
import { useMock } from '../apiSwitch';
import type { AgentFinding, AgentFindingContinue } from '../../types/agentFinding';

export const agentFindingApi = {
  async get(id: string): Promise<AgentFinding> {
    if (useMock) return mockFinding(id);
    const { data } = await http.get<AgentFinding>(`/agent/findings/${encodeURIComponent(id)}`);
    return data;
  },

  async markRead(id: string): Promise<void> {
    if (useMock) return;
    await http.post(`/agent/findings/${encodeURIComponent(id)}/read`);
  },

  async dismiss(id: string): Promise<void> {
    if (useMock) return;
    await http.post(`/agent/findings/${encodeURIComponent(id)}/dismiss`);
  },

  async continueAgent(id: string): Promise<AgentFindingContinue> {
    if (useMock) return { context: { agentFindingId: id }, message: '请基于系统生成的主动分析继续调查：任务失败主要由设备离线导致' };
    const { data } = await http.post<AgentFindingContinue>(`/agent/findings/${encodeURIComponent(id)}/continue`);
    return data;
  },
};

function mockFinding(id: string): AgentFinding {
  return {
    id, deliveryId: 'delivery-mock-1', remoteFindingId: 'remote-finding-mock-1', runId: 'run-mock-1',
    scenarioKey: 'task-failure-analysis', title: '任务失败主要由设备离线导致',
    summary: '任务执行窗口内设备 SN001 处于离线状态，且近 24 小时已有 3 次同类失败。建议先恢复设备连通性，再由用户在 AgentPanel 中确认后续处理。',
    severity: 'high', confidence: 0.86,
    facts: [
      { id: 'fact-1', text: '任务在设备离线期间失败。', evidenceRefs: ['tool:get.devices.by_id'] },
      { id: 'fact-2', text: '近 24 小时存在 3 次同类失败。', evidenceRefs: ['tool:get.devices.tasks.by_task_id'] },
    ],
    hypotheses: [{ id: 'hyp-1', text: '设备连接中断是本次失败的主要原因。', confidence: 0.86, evidenceRefs: ['fact-1', 'fact-2'] }],
    details: { failureCategory: 'device_offline', recentFailureCount: 3, taskType: '配置下发', deviceSerialNumber: 'SN001' },
    suggestedActions: [
      { type: 'open-resource', label: '查看设备', resourceRole: 'device' },
      { type: 'continue-agent', label: '继续询问 Agent', promptKey: 'task-failure-followup' },
      { type: 'dismiss', label: '忽略' },
    ],
    presentation: { surfaces: ['attention', 'task-detail'], sections: ['summary', 'facts', 'hypotheses', 'details', 'resources'] },
    resources: [
      { type: 'task', id: 'task-123', role: 'task', label: '配置任务' },
      { type: 'device', id: 'device-456', role: 'device', label: 'SN001' },
    ],
    read: false, dismissed: false, createdAt: new Date().toISOString(),
  };
}
