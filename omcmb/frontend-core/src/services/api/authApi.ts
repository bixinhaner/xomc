import http from '../http';
import type { TokenPairResponse } from '../../store/userStore';
import type { User } from '../../types/system';
import {
  preparePasswordPayload,
  invalidatePublicKeyCache,
} from '../crypto/passwordCipher';

/** 验证码挑战响应（后端 CaptchaChallenge） */
export interface CaptchaChallenge {
  captchaId: string;
  image: string; // data:image/png;base64,...
}

/** 登录时附带的验证码参数 */
export interface CaptchaCredentials {
  captchaId: string;
  captchaAnswer: string;
}

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
  /**
   * 登录接口
   * @param captcha 可选，当后端要求验证码时（biz_code=7010）附带
   */
  async login(
    username: string,
    password: string,
    captcha?: CaptchaCredentials,
  ): Promise<TokenPairResponse> {
    const { encryptedPassword, keyId } = await preparePasswordPayload(password);
    const body: Record<string, unknown> = {
      username,
      encrypted_password: encryptedPassword,
      key_id: keyId,
    };
    // 附带验证码（snake_case 给后端）
    if (captcha) {
      body.captcha_id = captcha.captchaId;
      body.captcha_answer = captcha.captchaAnswer;
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

  /** 获取验证码图片（免登录） */
  async getCaptcha(): Promise<CaptchaChallenge> {
    const { data } = await http.get<{ captcha_id: string; image: string }>('/auth/captcha');
    return {
      captchaId: data.captcha_id,
      image: data.image,
    };
  },
};
