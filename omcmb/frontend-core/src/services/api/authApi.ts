import http from '../http';
import type { TokenPairResponse } from '../../store/userStore';
import type { User } from '../../types/system';

interface BackendUser {
  id: string;
  username: string;
  display_name: string;
  email: string;
  carrier?: string;
  status: string;
  roles?: Array<{ id: string; name: string; description: string }>;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
}

function mapBackendUserToFrontend(bu: BackendUser): User {
  return {
    id: bu.id,
    username: bu.username,
    displayName: bu.display_name || bu.username,
    email: bu.email || '',
    phone: '',
    role: ((bu.roles && bu.roles.length > 0 ? bu.roles[0].name : 'viewer') as User['role']),
    status: (bu.status as User['status']) || 'active',
    lastLoginTime: bu.last_login_at || '',
    createTime: bu.created_at,
    updateTime: bu.updated_at,
  };
}

export const authApi = {
  async login(username: string, password: string): Promise<TokenPairResponse> {
    const { data } = await http.post<TokenPairResponse>('/auth/login', {
      username,
      password,
    });
    return data;
  },

  async refresh(refreshToken: string): Promise<TokenPairResponse> {
    const { data } = await http.post<TokenPairResponse>('/auth/refresh', {
      refresh_token: refreshToken,
    });
    return data;
  },

  async getMe(): Promise<User> {
    const { data } = await http.get<BackendUser>('/auth/me');
    return mapBackendUserToFrontend(data);
  },
};
