import http from '../http';

export interface DelegationTokenResponse {
  token: string;
  expires_at: string;
}

export interface AgentApi {
  requestDelegationToken(): Promise<DelegationTokenResponse>;
}

export const agentApi: AgentApi = {
  async requestDelegationToken() {
    const { data } = await http.post<DelegationTokenResponse>('/agent/delegation');
    return data;
  },
};

