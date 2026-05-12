import http from '../http';
import type { TokenPairResponse } from '../../store/userStore';
import type { User } from '../../types/system';
import {
  preparePasswordPayload,
  isPlaintextPayload,
  invalidatePublicKeyCache,
} from '../crypto/passwordCipher';

interface BackendUser {
  id: string;
  username: string;
  display_name: string;
  email: string;
  carrier?: string;
  status: string;
  source?: 'builtIn' | 'admin' | 'LDAP';
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
    // T-0098-P4-02：派生超管标志（与后端 user.IsSuperAdmin() 即 source==='builtIn' 同义）
    isSuperAdmin: bu.source === 'builtIn',
    source: bu.source,
    status: (bu.status as User['status']) || 'active',
    lastLoginTime: bu.last_login_at || '',
    createTime: bu.created_at,
    updateTime: bu.updated_at,
  };
}

export const authApi = {
  async login(username: string, password: string): Promise<TokenPairResponse> {
    // T-0120 双路径：secure context → RSA-OAEP 加密；非 secure context →
    // 明文 fallback（后端需 LoginCrypto.AllowPlaintext=true）
    const payload = await preparePasswordPayload(password);
    const body: Record<string, unknown> = { username };
    if (isPlaintextPayload(payload)) {
      body.password = payload.plaintextPassword;
    } else {
      body.encrypted_password = payload.encryptedPassword;
      body.key_id = payload.keyId;
    }
    try {
      const { data } = await http.post<TokenPairResponse>('/auth/login', body);
      return data;
    } catch (err: unknown) {
      // 401 时主动失效公钥缓存：后端可能轮换了密钥，下次登录重新拉取。
      const status =
        typeof err === 'object' && err !== null && 'response' in err
          ? (err as { response?: { status?: number } }).response?.status
          : undefined;
      if (status === 401) invalidatePublicKeyCache();
      throw err;
    }
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
