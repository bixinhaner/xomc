// 应用版本号（展示在左侧菜单底部）。
//
// 构建期由 Vite 环境变量 VITE_APP_VERSION 注入,链路为:
//   build-release.sh -v <X.Y.Z>
//     → VERSION=<X.Y.Z>-<YYYYMMDD-HHMM>（交付包和镜像 tag 使用完整版本）
//     → release 渠道的 APP_VERSION=<X.Y.Z>；test 渠道的 APP_VERSION 使用完整版本
//     → docker build --build-arg APP_VERSION=$APP_VERSION
//     → Dockerfile.web: ENV VITE_APP_VERSION=$APP_VERSION
//     → vite build 把它内联进产物
//
// 本地 dev / 未注入时回退 'dev',明确区分"非发布构建"。
const raw = (import.meta.env.VITE_APP_VERSION as string | undefined)?.trim();

export const APP_VERSION = raw ? `v${raw}` : 'dev';
