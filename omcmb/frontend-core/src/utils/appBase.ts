/**
 * 前端部署 base 前缀工具。
 *
 * V1 通常部署在 `/`，也允许通过 Vite `base` 部署到自定义子路径；构建期会把
 * `import.meta.env.BASE_URL`
 * （vite 会把 `import.meta.env.BASE_URL` 静态替换成字面量）。任何**硬跳转**
 * （`window.location.href`，用于登录失效 / 闲置登出等绕过 React Router 的场景）
 * 都必须带上应用 base 前缀，避免自定义子路径部署时跳错登录页。
 *
 * React Router 内部导航走 basename，不需要本工具；本工具只服务于硬跳转。
 *
 * 纯函数版（`basePathOf` / `loginUrlFor`）便于单测；读环境版（`appBasePath` /
 * `loginUrl`）在运行期取 vite 注入的 base。
 */

/** 把 vite base（如 `'/'`、`'/omc/'` 或空）规整成无尾斜杠前缀。 */
export function basePathOf(baseUrl: string | undefined | null): string {
  return (baseUrl || '/').replace(/\/+$/, '');
}

/** 给定 base 算登录页绝对路径。 */
export function loginUrlFor(baseUrl: string | undefined | null): string {
  return `${basePathOf(baseUrl)}/login`;
}

/** 当前应用的 base 前缀（去尾斜杠）。 */
export function appBasePath(): string {
  return basePathOf(import.meta.env.BASE_URL);
}

/** 当前应用登录页的绝对路径（带 base 前缀）。 */
export function loginUrl(): string {
  return loginUrlFor(import.meta.env.BASE_URL);
}
