/**
 * passwordCipher.ts — 登录类接口的密码加密传输（前端侧）。
 *
 * 流程：
 *   1. 调 GET /auth/public-key 拉取后端公钥（PEM SubjectPublicKeyInfo）+ keyId
 *   2. 用 Web Crypto API 把 RSA-OAEP/SHA-256 公钥导入为 CryptoKey
 *   3. 加密 JSON 载荷 {password, ts, nonce} → base64
 *   4. 上层 API 把 {encrypted_password, key_id} 提交给后端
 *
 * 设计选择：
 *   - 使用浏览器原生 Web Crypto API，避免引入 jsencrypt / node-forge 等额外依赖
 *   - 公钥结果在内存缓存 5 分钟，减少对 /auth/public-key 的请求；登录失败/401 时主动失效
 *   - nonce 用 crypto.getRandomValues 生成 16 字节随机数，hex 编码
 *   - ts 取客户端 Date.now()，后端校验 ±5 min 容差
 */
import http from '../http';

interface PublicKeyResponse {
  key_id: string;
  public_key: string;
  algorithm: string;
  hash: string;
}

interface CachedKey {
  keyId: string;
  cryptoKey: CryptoKey;
  fetchedAt: number;
}

const PUBLIC_KEY_TTL_MS = 5 * 60 * 1000; // 5 min；与后端 ReplayWindow 同量级

let cached: CachedKey | null = null;

/**
 * 把 PEM 编码的 SubjectPublicKeyInfo 转成 ArrayBuffer，供 Web Crypto 导入。
 */
function pemToArrayBuffer(pem: string): ArrayBuffer {
  const cleaned = pem
    .replace(/-----BEGIN [^-]+-----/g, '')
    .replace(/-----END [^-]+-----/g, '')
    .replace(/\s+/g, '');
  const binary = atob(cleaned);
  const buf = new ArrayBuffer(binary.length);
  const view = new Uint8Array(buf);
  for (let i = 0; i < binary.length; i++) view[i] = binary.charCodeAt(i);
  return buf;
}

function arrayBufferToBase64(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf);
  let binary = '';
  for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i]);
  return btoa(binary);
}

function generateNonce(): string {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  let out = '';
  for (let i = 0; i < bytes.length; i++) {
    out += bytes[i].toString(16).padStart(2, '0');
  }
  return out;
}

/**
 * 拉取并缓存后端公钥；缓存内则直接返回。
 *
 * 为减少集成复杂度，本函数对 /auth/public-key 不带 Authorization 头依赖（公钥本就是公开端点）。
 */
async function getPublicKey(): Promise<CachedKey> {
  const now = Date.now();
  if (cached && now - cached.fetchedAt < PUBLIC_KEY_TTL_MS) {
    return cached;
  }
  const { data } = await http.get<PublicKeyResponse>('/auth/public-key');
  const der = pemToArrayBuffer(data.public_key);
  const cryptoKey = await crypto.subtle.importKey(
    'spki',
    der,
    { name: 'RSA-OAEP', hash: 'SHA-256' },
    false,
    ['encrypt']
  );
  cached = { keyId: data.key_id, cryptoKey, fetchedAt: now };
  return cached;
}

/**
 * 主动失效公钥缓存。登录返回 401 时建议调用，避免下次仍用过期公钥重试。
 */
export function invalidatePublicKeyCache(): void {
  cached = null;
}

export interface EncryptedPasswordPayload {
  encryptedPassword: string;
  keyId: string;
}

/**
 * T-0120 plaintext fallback：非 secure context（如 http://内网IP）部署下
 * crypto.subtle 不可用，无法 RSA-OAEP 加密。preparePasswordPayload 返回明文
 * marker；调用方据此切到后端 plaintext path（要求后端 LoginCrypto.AllowPlaintext=true）。
 */
export interface PlaintextPasswordPayload {
  plaintextPassword: string;
}

export type PasswordPayload = EncryptedPasswordPayload | PlaintextPasswordPayload;

/** Type guard：是否走 plaintext fallback 路径。 */
export function isPlaintextPayload(p: PasswordPayload): p is PlaintextPasswordPayload {
  return 'plaintextPassword' in p;
}

/**
 * 加密单个密码字段。返回 {encryptedPassword, keyId}，由调用方按各端点字段名映射。
 *
 * 单次登录/改密里多次调用本函数会复用同一公钥（缓存内），但每次生成不同的 ts/nonce，
 * 因此密文也不同；后端按 nonce 去重防重放。
 *
 * T-0117 fail-fast：Web Crypto API 仅在 secure context（HTTPS / localhost / 127.0.0.1）
 * 可用。访问 `http://<内网 IP>:8081` 之类非 secure context 时 `crypto.subtle === undefined`，
 * 历史上会在 `crypto.subtle.importKey(...)` 抛 TypeError，调用方看到的现象是
 * "/auth/public-key 已成功，但 /auth/login 永不发出"。本检测让错误前置 + 信息明确。
 */
export async function encryptPassword(plain: string): Promise<EncryptedPasswordPayload> {
  if (!plain) throw new Error('encryptPassword: plain password is empty');

  // T-0117 fail-fast 保留：encryptPassword 是 secure context 专用 API
  // T-0120 调用方应改走 preparePasswordPayload（自动 fallback 明文），但本函数
  // 作为底层加密 primitive，保留显式抛错语义供测试 / 调试时直接调用。
  if (typeof window !== 'undefined' && window.isSecureContext === false) {
    throw new Error(
      '当前访问非安全上下文（非 HTTPS / 非 localhost），Web Crypto API 不可用。' +
      '请通过 https://<host> 或 http://localhost 访问；如部署在内网 IP，请联系运维启用 TLS。'
    );
  }
  if (typeof crypto === 'undefined' || !crypto.subtle) {
    throw new Error(
      '浏览器或当前页面环境不支持 Web Crypto API (crypto.subtle 不可用)。' +
      '请使用现代浏览器并通过 HTTPS 或 localhost 访问。'
    );
  }

  const { keyId, cryptoKey } = await getPublicKey();
  const payload = {
    password: plain,
    ts: Math.floor(Date.now() / 1000),
    nonce: generateNonce(),
  };
  const data = new TextEncoder().encode(JSON.stringify(payload));
  const ciphertext = await crypto.subtle.encrypt({ name: 'RSA-OAEP' }, cryptoKey, data);
  return {
    encryptedPassword: arrayBufferToBase64(ciphertext),
    keyId,
  };
}

/**
 * preparePasswordPayload — 登录/改密的统一密码准备入口。**永远走加密路径**。
 *
 * 历史：T-0120 一度引入 plaintext fallback（非 secure context 自动降级明文），
 * 2026-05-15 移除——明文传输违反"密码不落明文 / 传输全加密"的商用品质门，
 * fallback 形同虚设。
 *
 * 当前行为：
 *   - secure context（HTTPS / localhost / 127.0.0.1）→ RSA-OAEP 加密返回 {encryptedPassword, keyId}
 *   - 非 secure context → 直接 throw，提示用户切到 HTTPS 或 localhost
 *
 * 调用方仍可用 isPlaintextPayload 兼容老接口（永远 false）。
 */
export async function preparePasswordPayload(plain: string): Promise<PasswordPayload> {
  if (!plain) throw new Error('preparePasswordPayload: plain password is empty');
  // encryptPassword 内部已 fail-fast 检查 isSecureContext + crypto.subtle，
  // 这里不重复，让单一真相源在 encryptPassword。
  return await encryptPassword(plain);
}
