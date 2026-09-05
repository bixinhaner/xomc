import http from '../http';
import type { ProactiveOverview, ProactiveScenario } from '../../types/proactiveAgent';

export const proactiveAgentApi = {
  async overview(): Promise<ProactiveOverview> {
    const { data } = await http.get<ProactiveOverview>('/agent/admin/overview');
    return data;
  },
  async updateScenario(key: string, update: Partial<Pick<ProactiveScenario, 'status' | 'rolloutMode' | 'rolloutPercentage'>>): Promise<ProactiveScenario> {
    const { data } = await http.patch<ProactiveScenario>(`/agent/admin/scenarios/${encodeURIComponent(key)}`, update);
    return data;
  },
  async cancelRun(id: string): Promise<void> {
    await http.post(`/agent/admin/runs/${encodeURIComponent(id)}/cancel`);
  },
};
