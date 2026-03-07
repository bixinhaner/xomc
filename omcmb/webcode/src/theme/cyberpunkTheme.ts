import type { ThemeConfig } from 'antd';
import { theme } from 'antd';

/**
 * Cyberpunk Theme - 赛博朋克/黑客帝国风格
 * 极致炫酷的深色科技主题，配合霓虹光效
 */
export const antdCyberpunkTheme: ThemeConfig = {
  algorithm: theme.darkAlgorithm,
  token: {
    // 多色渐变主色（电光蓝）
    colorPrimary: '#00F5FF',
    colorBgLayout: '#0A0A1A',
    colorBgContainer: '#12122A',
    colorBgElevated: '#1A1A3A',
    colorBorder: 'rgba(0, 245, 255, 0.2)',
    colorBorderSecondary: 'rgba(191, 0, 255, 0.15)',
    colorText: '#E0E0FF',
    colorTextSecondary: '#A0A0CC',
    colorTextHeading: '#FFFFFF',
    colorTextDisabled: 'rgba(224, 224, 255, 0.3)',
    borderRadius: 2, // 棱角分明
    wireframe: false,
    fontFamily: "'Rajdhani', -apple-system, 'Segoe UI', sans-serif",
  },
  components: {
    Layout: {
      headerBg: 'rgba(10, 10, 26, 0.95)',
      headerHeight: 48,
      siderBg: 'rgba(10, 10, 26, 0.95)',
      bodyBg: '#0A0A1A',
    },
    Menu: {
      darkItemBg: 'transparent',
      darkSubMenuItemBg: 'rgba(0, 245, 255, 0.03)',
      darkItemSelectedBg: 'rgba(0, 245, 255, 0.15)',
      darkItemHoverBg: 'rgba(0, 245, 255, 0.08)',
      darkItemSelectedColor: '#00F5FF',
      darkItemColor: 'rgba(224, 224, 255, 0.7)',
      itemHeight: 40,
    },
    Table: {
      headerBg: '#12122A',
      headerColor: '#00F5FF',
      rowHoverBg: 'rgba(0, 245, 255, 0.05)',
      borderColor: 'rgba(0, 245, 255, 0.1)',
      headerSplitColor: 'rgba(0, 245, 255, 0.1)',
    },
    Card: {
      paddingLG: 16,
      borderRadiusLG: 4,
      colorBorderSecondary: 'rgba(0, 245, 255, 0.1)',
    },
    Button: {
      controlHeight: 36,
      controlHeightSM: 28,
      controlHeightLG: 44,
      borderRadius: 2,
      primaryShadow: '0 0 20px rgba(0, 245, 255, 0.4)',
    },
    Input: {
      controlHeight: 36,
      controlHeightSM: 28,
      controlHeightLG: 44,
      borderRadius: 2,
      activeShadow: '0 0 0 2px rgba(0, 245, 255, 0.3), 0 0 20px rgba(0, 245, 255, 0.2)',
    },
    Select: {
      controlHeight: 36,
      controlHeightSM: 28,
      borderRadius: 2,
    },
    Tabs: {
      horizontalMargin: '0',
      itemSelectedColor: '#00F5FF',
      itemHoverColor: '#00F5FF',
      inkBarColor: '#00F5FF',
    },
    Modal: {
      contentBg: '#12122A',
      headerBg: 'transparent',
      footerBg: 'transparent',
      titleColor: '#00F5FF',
    },
    Tag: {
      borderRadiusSM: 2,
    },
    Progress: {
      remainingColor: 'rgba(0, 245, 255, 0.1)',
    },
    Switch: {
      colorPrimary: '#00F5FF',
      colorPrimaryHover: '#33F7FF',
    },
  },
};
