/**
 * GIS 地图视图样式常量
 * 统一的设计 Token 和样式定义
 */

// ========== 设计 Token ==========

/** 间距系统 (4px 基准) */
export const SPACING = {
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 20,
  xxl: 24,
  xxxl: 32,
} as const;

/** 圆角系统 */
export const RADIUS = {
  sm: 6,
  md: 8,
  lg: 12,
  xl: 16,
  full: '50%',
} as const;

/** 阴影系统 - 分层阴影 */
export const SHADOWS = {
  /** 轻微阴影 - 用于浮层 */
  light: '0 1px 2px 0 rgba(0, 0, 0, 0.03), 0 1px 6px -1px rgba(0, 0, 0, 0.02)',
  /** 标准阴影 - 用于卡片 */
  base: '0 1px 3px 0 rgba(0, 0, 0, 0.04), 0 1px 2px -1px rgba(0, 0, 0, 0.04)',
  /** 中等阴影 - 用于下拉面板 */
  medium: '0 4px 6px -1px rgba(0, 0, 0, 0.05), 0 2px 4px -2px rgba(0, 0, 0, 0.05)',
  /** 深度阴影 - 用于模态框 */
  large: '0 10px 15px -3px rgba(0, 0, 0, 0.05), 0 4px 6px -4px rgba(0, 0, 0, 0.05)',
  /** 悬浮阴影 - 用于 hover 状态 */
  hover: '0 10px 15px -3px rgba(0, 0, 0, 0.08), 0 4px 6px -4px rgba(0, 0, 0, 0.08)',
} as const;

/** 颜色系统 - 状态色 */
export const COLORS = {
  /** 在线激活 - 绿色渐变 */
  onlineActive: {
    primary: '#52C41A',
    light: '#73D13D',
    gradient: 'linear-gradient(135deg, #73D13D 0%, #52C41A 100%)',
    bg: 'rgba(82, 196, 26, 0.1)',
    glow: 'rgba(82, 196, 26, 0.3)',
  },
  /** 在线未激活 - 黄色渐变 */
  onlineInactive: {
    primary: '#FAAD14',
    light: '#FFC53D',
    gradient: 'linear-gradient(135deg, #FFC53D 0%, #FAAD14 100%)',
    bg: 'rgba(250, 173, 20, 0.1)',
    glow: 'rgba(250, 173, 20, 0.3)',
  },
  /** 离线 - 红色 */
  offline: {
    primary: '#CF1322',
    light: '#F5222D',
    gradient: 'linear-gradient(135deg, #F5222D 0%, #CF1322 100%)',
    bg: 'rgba(207, 19, 34, 0.1)',
    glow: 'rgba(207, 19, 34, 0.3)',
  },
  /** 主色 */
  primary: {
    DEFAULT: '#1890FF',
    light: '#40A9FF',
    dark: '#096DD9',
    gradient: 'linear-gradient(135deg, #40A9FF 0%, #1890FF 100%)',
  },
  /** 中性色 */
  neutral: {
    50: '#FAFAFA',
    100: '#F5F5F5',
    200: '#E8E8E8',
    300: '#D9D9D9',
    400: '#BFBFBF',
    500: '#8C8C8C',
    600: '#595959',
    700: '#262626',
    800: '#1F1F1F',
    900: '#141414',
  },
} as const;

/** 动画时长 */
export const DURATION = {
  fast: 150,
  normal: 200,
  slow: 300,
} as const;

/** 缓动函数 */
export const EASING = {
  default: 'cubic-bezier(0.4, 0, 0.2, 1)',
  in: 'cubic-bezier(0.4, 0, 1, 1)',
  out: 'cubic-bezier(0, 0, 0.2, 1)',
  bounce: 'cubic-bezier(0.68, -0.55, 0.265, 1.55)',
} as const;

/** 过渡字符串生成器 */
export const transitionString = (
  properties: string[],
  duration: keyof typeof DURATION = 'normal',
  easing: keyof typeof EASING = 'default'
) => {
  return properties.map(p => `${p} ${DURATION[duration]}ms ${EASING[easing]}`).join(', ');
};

// ========== 组件样式生成器 ==========

/** 状态指示器样式 */
export const statusIndicatorStyle = (status: 'onlineActive' | 'onlineInactive' | 'offline', size: number = 14) => {
  const color = COLORS[status];
  return {
    width: size,
    height: size,
    borderRadius: '50%',
    background: color.gradient,
    boxShadow: `0 0 0 2px rgba(255,255,255,1), 0 0 0 3px ${color.glow}`,
    animation: 'pulse-glow 2s ease-in-out infinite',
  } as const;
};

/** 可点击区域样式 */
export const clickableStyle = (hover = true) => ({
  cursor: 'pointer',
  userSelect: 'none' as const,
  transition: transitionString(['transform', 'box-shadow'], 'fast'),
  ...(hover && {
    ':hover': {
      transform: 'translateY(-1px)',
      boxShadow: SHADOWS.hover,
    },
  }),
});
