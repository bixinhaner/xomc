import type { ThemeConfig } from 'antd';

/**
 * RMB Theme - 人民币/国潮风格
 * 中国红色+金色配色，庄重大气
 */
export const antdRmbTheme: ThemeConfig = {
  token: {
    colorPrimary: '#E60012',
    colorBgLayout: '#FFFEF5',
    colorBgContainer: '#FFFFFF',
    colorBgElevated: '#FFFFFF',
    colorBorder: '#E8DCD0',
    colorBorderSecondary: '#F0E8DC',
    colorText: '#2C2C2C',
    colorTextSecondary: '#5A5A5A',
    colorTextHeading: '#1A1A1A',
    colorTextDisabled: 'rgba(44, 44, 44, 0.3)',
    borderRadius: 4,
    wireframe: false,
    fontFamily: "'Noto Serif SC', 'Source Han Serif SC', 'Source Han Serif CN', serif",
  },
  components: {
    Layout: {
      headerBg: '#E60012',
      headerHeight: 52,
      siderBg: '#8B0000',
      bodyBg: '#FFFEF5',
    },
    Menu: {
      // Dark menu for red sidebar
      itemBg: 'transparent',
      subMenuItemBg: 'rgba(255, 255, 255, 0.08)',
      itemSelectedBg: 'rgba(212, 175, 55, 0.2)',
      itemHoverBg: 'rgba(255, 255, 255, 0.1)',
      itemSelectedColor: '#FFD700',
      itemColor: 'rgba(255, 255, 255, 0.85)',
      itemHoverColor: '#FFFFFF',
      itemHeight: 44,
    },
    Table: {
      headerBg: '#FFF8F0',
      headerColor: '#8B0000',
      rowHoverBg: 'rgba(230, 0, 18, 0.04)',
      borderColor: '#E8DCD0',
      headerSplitColor: '#E8DCD0',
    },
    Card: {
      paddingLG: 24,
      borderRadiusLG: 6,
      colorBorderSecondary: '#E8DCD0',
      boxShadowTertiary: '0 2px 8px rgba(139, 0, 0, 0.06)',
    },
    Button: {
      controlHeight: 38,
      controlHeightSM: 30,
      controlHeightLG: 46,
      borderRadius: 4,
      primaryShadow: '0 2px 8px rgba(230, 0, 18, 0.25)',
    },
    Input: {
      controlHeight: 38,
      controlHeightSM: 30,
      controlHeightLG: 46,
      borderRadius: 4,
      activeShadow: '0 0 0 2px rgba(230, 0, 18, 0.1)',
    },
    Select: {
      controlHeight: 38,
      controlHeightSM: 30,
      borderRadius: 4,
    },
    Tabs: {
      horizontalMargin: '0',
      itemSelectedColor: '#E60012',
      itemHoverColor: '#B8000F',
      inkBarColor: '#E60012',
    },
    Modal: {
      contentBg: '#FFFFFF',
      headerBg: '#FFFFFF',
      footerBg: '#FFFFFF',
      titleColor: '#8B0000',
      borderRadiusLG: 6,
    },
    Tag: {
      borderRadiusSM: 4,
    },
    Progress: {
      remainingColor: '#F0E8DC',
    },
    Switch: {
      colorPrimary: '#E60012',
      colorPrimaryHover: '#B8000F',
    },
  },
};
