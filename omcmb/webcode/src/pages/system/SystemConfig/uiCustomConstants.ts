// 与后端 sys_configs (category='ui_custom') 的 5 个 key 一一对应，
// 与 omcgo/migrations/seed/000065_seed_ui_custom.sql 默认值保持同步。
// 详见 docs/prd/system/ui-customization.md §3。

export const UI_CUSTOM_KEYS = [
  'ui_omc_name',
  'ui_login_background',
  'ui_menu_logo_up',
  'ui_menu_logo_down',
] as const;

export const UI_CUSTOM_DEFAULTS: Record<string, string> = {
  ui_omc_name: 'BaiOMC',
  ui_login_background: './images/login/login_bg.png',
  ui_menu_logo_up: './images/login/nav_logo_collapse.png',
  ui_menu_logo_down: './images/login/logo_big.png',
};

// 上传体积上限（字节）— 与后端 internal/admin/ui_asset_handler.go 的 uiAssetMaxLoginBg / uiAssetMaxLogo 对齐。
export const SIZE_LOGIN_BG = 1 * 1024 * 1024; // 1 MiB
export const SIZE_LOGO = 400 * 1024; // 400 KiB

export const ACCEPT_IMAGE_TYPES = ['image/png', 'image/jpeg', 'image/jpg'] as const;

export type UIAssetKind = 'login_bg' | 'logo_small' | 'logo_large';
