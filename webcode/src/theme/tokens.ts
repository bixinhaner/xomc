// Design System Tokens — OMC 统一网管系统
// Source: design/01-design-system/00-design-tokens.md

// ============ PRIMARY COLORS ============
export const COLOR_PRIMARY_50 = '#EBF5FF';
export const COLOR_PRIMARY_100 = '#D6EBFF';
export const COLOR_PRIMARY_200 = '#ADD6FF';
export const COLOR_PRIMARY_300 = '#85C1FF';
export const COLOR_PRIMARY_400 = '#5CACFF';
export const COLOR_PRIMARY_500 = '#3396FF';
export const COLOR_PRIMARY_600 = '#1677FF'; // Main action color
export const COLOR_PRIMARY_700 = '#0958D9';
export const COLOR_PRIMARY_800 = '#003EB3';
export const COLOR_PRIMARY_900 = '#002C8C';

// ============ NEUTRAL COLORS ============
export const COLOR_NEUTRAL_50 = '#FAFAFA';  // Page background
export const COLOR_NEUTRAL_100 = '#F5F5F5'; // Card bg, zebra stripe
export const COLOR_NEUTRAL_200 = '#F0F0F0'; // Dividers
export const COLOR_NEUTRAL_300 = '#D9D9D9'; // Disabled borders
export const COLOR_NEUTRAL_400 = '#BFBFBF'; // Disabled text
export const COLOR_NEUTRAL_500 = '#8C8C8C'; // Secondary text
export const COLOR_NEUTRAL_600 = '#595959'; // Default body text
export const COLOR_NEUTRAL_700 = '#434343'; // Emphasized body
export const COLOR_NEUTRAL_800 = '#262626'; // Main headings
export const COLOR_NEUTRAL_900 = '#1F1F1F'; // Sidebar bg
export const COLOR_NEUTRAL_950 = '#141414'; // Sidebar deep

// ============ ALARM SEVERITY (YD/T Standard) ============
export const SEVERITY_CRITICAL = '#F5222D';
export const SEVERITY_CRITICAL_BG = '#FFF1F0';
export const SEVERITY_CRITICAL_BORDER = '#FFA39E';
export const SEVERITY_CRITICAL_DARK = '#A8071A';

export const SEVERITY_MAJOR = '#FA8C16';
export const SEVERITY_MAJOR_BG = '#FFF7E6';
export const SEVERITY_MAJOR_BORDER = '#FFD591';
export const SEVERITY_MAJOR_DARK = '#AD4E00';

export const SEVERITY_MINOR = '#FAAD14';
export const SEVERITY_MINOR_BG = '#FFFBE6';
export const SEVERITY_MINOR_BORDER = '#FFE58F';
export const SEVERITY_MINOR_DARK = '#AD6800';

export const SEVERITY_WARNING = '#1890FF';
export const SEVERITY_WARNING_BG = '#E6F7FF';
export const SEVERITY_WARNING_BORDER = '#91D5FF';
export const SEVERITY_WARNING_DARK = '#0050B3';

// ============ STATUS COLORS ============
export const STATUS_SUCCESS = '#52C41A';
export const STATUS_SUCCESS_BG = '#F6FFED';
export const STATUS_ERROR = '#F5222D';
export const STATUS_ERROR_BG = '#FFF1F0';
export const STATUS_PROCESSING = '#1677FF';
export const STATUS_PROCESSING_BG = '#E6F7FF';
export const STATUS_WARNING = '#FA8C16';
export const STATUS_WARNING_BG = '#FFF7E6';
export const STATUS_INACTIVE = '#8C8C8C';
export const STATUS_INACTIVE_BG = '#FAFAFA';
export const STATUS_LOCKED = '#722ED1';
export const STATUS_LOCKED_BG = '#F9F0FF';

// ============ SIDEBAR (Always dark) ============
export const SIDEBAR_BG = '#001529';
export const SIDEBAR_BG_HOVER = 'rgba(255,255,255,0.08)';
export const SIDEBAR_BG_ACTIVE = '#1677FF';
export const SIDEBAR_TEXT = 'rgba(255,255,255,0.65)';
export const SIDEBAR_TEXT_ACTIVE = '#FFFFFF';
export const SIDEBAR_TEXT_HOVER = 'rgba(255,255,255,0.85)';
export const SIDEBAR_DIVIDER = 'rgba(255,255,255,0.06)';
export const SIDEBAR_SUBMENU_BG = '#000C17';

// ============ CHART PALETTE ============
export const CHART_COLORS = [
  '#1677FF', '#52C41A', '#FA8C16', '#F5222D',
  '#722ED1', '#13C2C2', '#EB2F96', '#FAAD14',
];

// ============ FONT ============
export const FONT_FAMILY = "-apple-system, BlinkMacSystemFont, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'Helvetica Neue', Helvetica, Arial, sans-serif";
export const FONT_FAMILY_MONO = "'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, Courier, monospace";

export const FONT_SIZE_XS = 12;
export const FONT_SIZE_SM = 13;
export const FONT_SIZE_BASE = 14;
export const FONT_SIZE_MD = 16;
export const FONT_SIZE_LG = 18;
export const FONT_SIZE_XL = 20;
export const FONT_SIZE_2XL = 24;
export const FONT_SIZE_3XL = 32;
export const FONT_SIZE_4XL = 40;

// ============ SPACING ============
export const SPACE_05 = 2;
export const SPACE_1 = 4;
export const SPACE_2 = 8;
export const SPACE_3 = 12;
export const SPACE_4 = 16;
export const SPACE_5 = 20;
export const SPACE_6 = 24;
export const SPACE_8 = 32;
export const SPACE_10 = 40;
export const SPACE_12 = 48;
export const SPACE_16 = 64;

// ============ LAYOUT DIMENSIONS ============
export const HEADER_HEIGHT = 48;
export const SIDEBAR_WIDTH = 240;
export const SIDEBAR_COLLAPSED_WIDTH = 64;
export const TAB_BAR_HEIGHT = 36;
export const TASK_PANEL_COLLAPSED = 40;
export const TASK_PANEL_EXPANDED = 240;
export const CONTENT_PADDING_H = 24;
export const CONTENT_PADDING_V = 16;
export const TABLE_ROW_HEIGHT = 40;
export const TABLE_ROW_COMPACT = 32;
export const TABLE_ROW_COMFORTABLE = 48;
export const FILTER_INPUT_HEIGHT = 32;

// ============ SHADOWS ============
export const SHADOW_SM = '0 1px 2px 0 rgba(0,0,0,0.03), 0 1px 6px -1px rgba(0,0,0,0.02), 0 2px 4px 0 rgba(0,0,0,0.02)';
export const SHADOW_MD = '0 3px 6px -4px rgba(0,0,0,0.12), 0 6px 16px 0 rgba(0,0,0,0.08), 0 9px 28px 8px rgba(0,0,0,0.05)';
export const SHADOW_LG = '0 6px 16px -8px rgba(0,0,0,0.08), 0 9px 28px 0 rgba(0,0,0,0.05), 0 12px 48px 16px rgba(0,0,0,0.03)';

// ============ BORDER RADIUS ============
export const RADIUS_NONE = 0;
export const RADIUS_XS = 2;
export const RADIUS_SM = 4;
export const RADIUS_MD = 6;
export const RADIUS_LG = 8;
export const RADIUS_XL = 12;
export const RADIUS_FULL = 9999;

// ============ MOTION ============
export const DURATION_FAST = 100;
export const DURATION_NORMAL = 200;
export const DURATION_SLOW = 300;
export const DURATION_SLOWER = 400;

// ============ Z-INDEX ============
export const Z_CONTENT = 0;
export const Z_FLOATING = 10;
export const Z_SIDEBAR = 100;
export const Z_HEADER = 100;
export const Z_TASK_PANEL = 200;
export const Z_DROPDOWN = 1000;
export const Z_TOOLTIP = 1050;
export const Z_TOAST = 2000;

// ============ BREAKPOINTS ============
export const SCREEN_SM = 576;
export const SCREEN_MD = 768;
export const SCREEN_LG = 992;
export const SCREEN_XL = 1200;
export const SCREEN_2XL = 1600;
