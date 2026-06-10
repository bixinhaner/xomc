/**
 * authApi 契约测试（#22 关键 API）：
 *   - login：发 encrypted_password + key_id（密码不明文上线），透传 token 对。
 *   - login 401：主动失效公钥缓存（后端可能轮换密钥），并继续抛错。
 *   - login 其它错误（500/网络）：不动公钥缓存，原样抛错。
 *   - refresh：带 refresh_token 打 /auth/refresh。
 *   - getMe：BackendUser → User 映射（display_name 空回退 username、roles[0] 推 role、
 *     source==='builtIn' 派生 isSuperAdmin）。
 *
 * 用 vi.mock 替换 http 客户端与 passwordCipher（避免真实 WebCrypto / 公钥拉取）。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';

const { getMock, postMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
}));
vi.mock('../../http', () => ({
  default: { get: getMock, post: postMock, patch: vi.fn(), delete: vi.fn() },
}));

const { prepareMock, invalidateMock } = vi.hoisted(() => ({
  prepareMock: vi.fn(),
  invalidateMock: vi.fn(),
}));
vi.mock('../../crypto/passwordCipher', () => ({
  preparePasswordPayload: prepareMock,
  invalidatePublicKeyCache: invalidateMock,
}));

import { authApi } from '../authApi';

beforeEach(() => {
  getMock.mockReset();
  postMock.mockReset();
  prepareMock.mockReset();
  invalidateMock.mockReset();
  prepareMock.mockResolvedValue({ encryptedPassword: 'ENC', keyId: 'kid-1' });
});

describe('authApi.login', () => {
  it('发 encrypted_password + key_id（密码不明文），透传 token 对', async () => {
    postMock.mockResolvedValue({
      data: { access_token: 'a', refresh_token: 'r', expires_at: '2026-06-10T00:00:00Z' },
    });
    const out = await authApi.login('admin', 'secret');
    const [url, body] = postMock.mock.calls[0];
    expect(url).toBe('/auth/login');
    expect(body.username).toBe('admin');
    expect(body.encrypted_password).toBe('ENC');
    expect(body.key_id).toBe('kid-1');
    // 明文密码绝不出现在请求体
    expect(JSON.stringify(body)).not.toContain('secret');
    expect(out.access_token).toBe('a');
  });

  it('401 时主动失效公钥缓存并继续抛错（密钥轮换路径）', async () => {
    postMock.mockRejectedValue({ response: { status: 401 } });
    await expect(authApi.login('admin', 'bad')).rejects.toEqual({ response: { status: 401 } });
    expect(invalidateMock).toHaveBeenCalledOnce();
  });

  it('500 等非 401 错误：不动公钥缓存，原样抛错', async () => {
    postMock.mockRejectedValue({ response: { status: 500 } });
    await expect(authApi.login('admin', 'x')).rejects.toEqual({ response: { status: 500 } });
    expect(invalidateMock).not.toHaveBeenCalled();
  });

  it('网络错误（无 response）：不失效缓存，原样抛错', async () => {
    const netErr = new Error('Network Error');
    postMock.mockRejectedValue(netErr);
    await expect(authApi.login('admin', 'x')).rejects.toThrow('Network Error');
    expect(invalidateMock).not.toHaveBeenCalled();
  });
});

describe('authApi.refresh', () => {
  it('带 refresh_token 打 /auth/refresh', async () => {
    postMock.mockResolvedValue({
      data: { access_token: 'a2', refresh_token: 'r2', expires_at: '2026-06-11T00:00:00Z' },
    });
    const out = await authApi.refresh('old-refresh');
    const [url, body] = postMock.mock.calls[0];
    expect(url).toBe('/auth/refresh');
    expect(body.refresh_token).toBe('old-refresh');
    expect(out.access_token).toBe('a2');
  });

  it('401 错误原样抛（refresh 不触发公钥失效）', async () => {
    postMock.mockRejectedValue({ response: { status: 401 } });
    await expect(authApi.refresh('expired')).rejects.toEqual({ response: { status: 401 } });
    expect(invalidateMock).not.toHaveBeenCalled();
  });
});

describe('authApi.getMe — BackendUser → User 映射', () => {
  it('完整字段：roles[0] 推 role、source==builtIn 派生 isSuperAdmin', async () => {
    getMock.mockResolvedValue({
      data: {
        id: 'u1',
        username: 'admin',
        display_name: '管理员',
        email: 'admin@omc.io',
        source: 'builtIn',
        status: 'active',
        roles: [{ id: 'r1', name: 'super_admin', description: '' }],
        last_login_at: '2026-06-09T08:00:00Z',
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-06-09T08:00:00Z',
      },
    });
    const u = await authApi.getMe();
    expect(getMock.mock.calls[0][0]).toBe('/auth/me');
    expect(u.displayName).toBe('管理员');
    expect(u.role).toBe('super_admin');
    expect(u.isSuperAdmin).toBe(true);
    expect(u.lastLoginTime).toBe('2026-06-09T08:00:00Z');
  });

  it('字段缺失兜底：display_name/email/roles 缺 → 回退 username/空串/viewer，非 builtIn 非超管', async () => {
    getMock.mockResolvedValue({
      data: {
        id: 'u2',
        username: 'op',
        display_name: '',
        email: '',
        source: 'admin',
        status: '',
        created_at: '2026-02-01T00:00:00Z',
        updated_at: '2026-02-01T00:00:00Z',
      },
    });
    const u = await authApi.getMe();
    expect(u.displayName).toBe('op'); // display_name 空回退 username
    expect(u.email).toBe('');
    expect(u.role).toBe('viewer'); // roles 缺省回退 viewer
    expect(u.isSuperAdmin).toBe(false);
    expect(u.status).toBe('active'); // status 空回退 active
    expect(u.lastLoginTime).toBe(''); // last_login_at 缺省 → 空串
  });

  it('getMe 500 错误原样抛（不吞错）', async () => {
    getMock.mockRejectedValue({ response: { status: 500 } });
    await expect(authApi.getMe()).rejects.toEqual({ response: { status: 500 } });
  });
});
