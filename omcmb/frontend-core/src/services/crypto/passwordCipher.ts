/**
 * passwordCipher.ts — 登录类接口的密码加密传输（前端侧）。
 *
 * 流程：
 *   1. 调 GET /auth/public-key 拉取后端公钥（PEM SubjectPublicKeyInfo）+ keyId
 *   2. 用 node-forge 把 RSA 公钥从 PEM 解析为 forge PublicKey
 *   3. 用 RSA-OAEP/SHA-256 加密 JSON 载荷 {password, ts, nonce} → base64
 *   4. 上层 API 把 {encrypted_password, key_id} 提交给后端
 *
 * 设计选择：
 *   - 用 node-forge（纯 JS RSA 实现）而非浏览器原生 Web Crypto API：
 *     Web Crypto 的 crypto.subtle 仅在 secure context（HTTPS / localhost）可用，
 *     `http://<内网 IP>:8081` 部署会直接 throw。node-forge 在任何上下文都能跑。
 *   - 仍用浏览器的 `crypto.getRandomValues` 生成 nonce —— 该 API 在非 secure
 *     context 也可用（只有 `crypto.subtle` 才受限）。
 *   - 公钥缓存 5 分钟，减少 /auth/public-key 请求；401 时主动失效。
 */
import forge from 'node-forge';
import http from '../http';

interface PublicKeyResponse {
  key_id: string;
  public_key: string;
  algorithm: string;
  hash: string;
}

interface CachedKey {
  keyId: string;
  publicKey: forge.pki.rsa.PublicKey;
  fetchedAt: number;
}

const PUBLIC_KEY_TTL_MS = 5 * 60 * 1000; // 5 min；与后端 ReplayWindow 同量级

let cached: CachedKey | null = null;

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
 * /auth/public-key 是公开端点，不需要 Authorization。
 */
async function getPublicKey(): Promise<CachedKey> {
  const now = Date.now();
  if (cached && now - cached.fetchedAt < PUBLIC_KEY_TTL_MS) {
    return cached;
  }
  const { data } = await http.get<PublicKeyResponse>('/auth/public-key');
  const publicKey = forge.pki.publicKeyFromPem(data.public_key) as forge.pki.rsa.PublicKey;
  cached = { keyId: data.key_id, publicKey, fetchedAt: now };
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
 * preparePasswordPayload — 登录 / 改密的密码准备入口，永远走加密路径。
 *
 * 输出：{encryptedPassword, keyId}。调用方按各端点字段名映射到请求体
 * （登录：encrypted_password；改密：encrypted_old_password / encrypted_new_password）。
 *
 * 同一次登录里多次调用复用同一公钥（缓存内），每次生成不同的 ts/nonce，
 * 因此密文也不同；后端按 nonce 去重防重放。
 */
export async function preparePasswordPayload(plain: string): Promise<EncryptedPasswordPayload> {
  if (!plain) throw new Error('preparePasswordPayload: plain password is empty');

  const { keyId, publicKey } = await getPublicKey();
  const payload = {
    password: plain,
    ts: Math.floor(Date.now() / 1000),
    nonce: generateNonce(),
  };
  const json = JSON.stringify(payload);
  // node-forge 接受字节串（latin1 字符串）作为 encrypt 输入；JSON 含非 ASCII 时
  // 必须先 UTF-8 编码再喂给 forge，否则 byte 长度与解密侧不一致。
  const utf8 = forge.util.encodeUtf8(json);
  const ciphertext = publicKey.encrypt(utf8, 'RSA-OAEP', {
    md: forge.md.sha256.create(),
    mgf1: { md: forge.md.sha256.create() },
  });
  return {
    encryptedPassword: forge.util.encode64(ciphertext),
    keyId,
  };
}
