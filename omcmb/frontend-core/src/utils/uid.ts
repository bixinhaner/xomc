/**
 * 生成客户端 UID（用于 React key / store 内部 ID，非密码学场景）。
 *
 * 优先 `crypto.randomUUID()`，但浏览器只在 secure context（HTTPS / localhost）
 * 暴露它；HTTP 部署（如内网 http://172.19.1.73:8081）下 `crypto.randomUUID`
 * 是 undefined，直接调用会抛 `TypeError: crypto.randomUUID is not a function`。
 *
 * 回退用 `Date.now() + Math.random()` 拼接，碰撞概率对前端短期使用足够低。
 */
export function generateUid(prefix = 'uid'): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}
