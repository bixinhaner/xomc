/**
 * 三皮肤部署的 base 前缀工具。
 *
 * 前端是「单底层 frontend-core + 三套皮肤」：v1 部署在 `/`、v2 在 `/v2/`、v3 在
 * `/v3/`，各皮肤的 vite `base` 在构建期固化进 `import.meta.env.BASE_URL`
 * （vite 会把 `import.meta.env.BASE_URL` 静态替换成字面量）。任何**硬跳转**
 * （`window.location.href`，用于登录失效 / 闲置登出等绕过 React Router 的场景）
 * 都必须带上当前皮肤的 base 前缀，否则 v2/v3 会被甩到 v1 的 `/login`，丢失皮肤
 * 上下文（用户在 v2 里超时，却落到 v1 登录页）。
 *
 * React Router 内部导航走 basename，不需要本工具；本工具只服务于硬跳转。
 *
 * 纯函数版（`basePathOf` / `loginUrlFor`）便于单测；读环境版（`appBasePath` /
 * `loginUrl`）在运行期取 vite 注入的 base。
 */

/** 把 vite base（`'/'` | `'/v2/'` | `'/v3/'` | 空）规整成无尾斜杠前缀：`''` | `'/v2'`。 */
export function basePathOf(baseUrl: string | undefined | null): string {
  return (baseUrl || '/').replace(/\/+$/, '');
}

/** 给定 base 算登录页绝对路径：`/login` | `/v2/login` | `/v3/login`。 */
export function loginUrlFor(baseUrl: string | undefined | null): string {
  return `${basePathOf(baseUrl)}/login`;
}

/** 当前皮肤的 base 前缀（去尾斜杠）。 */
export function appBasePath(): string {
  return basePathOf(import.meta.env.BASE_URL);
}

/** 当前皮肤登录页的绝对路径（带 base 前缀）。 */
export function loginUrl(): string {
  return loginUrlFor(import.meta.env.BASE_URL);
}
