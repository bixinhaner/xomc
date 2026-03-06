import type { ThemeConfig } from 'antd';

/**
 * Tiffany Theme - 纯美/蒂芙尼风格
 * 优雅奢华的珠宝品牌风格，精致细腻
 */
export const antdTiffanyTheme: ThemeConfig = {
  token: {
    colorPrimary: '#81D8D0',
    colorBgLayout: '#FEFEFE',
    colorBgContainer: '#FFFFFF',
    colorBgElevated: '#FFFFFF',
    colorBorder: '#E8E8E8',
    colorBorderSecondary: '#F0F0F0',
    colorText: '#2C2C2C',
    colorTextSecondary: '#5A5A5A',
    colorTextHeading: '#1A1A1A',
    colorTextDisabled: 'rgba(44, 44, 44, 0.3)',
    borderRadius: 4, // 精致小圆角
    wireframe: false,
    fontFamily: "'Lato', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
  },
  components: {
    Layout: {
      headerBg: '#FFFFFF',
      headerHeight: 52,
      siderBg: '#81D8D0',
      bodyBg: '#FEFEFE',
    },
    Menu: {
      // Light menu for Tiffany blue sidebar
      itemBg: 'transparent',
      subMenuItemBg: 'rgba(255, 255, 255, 0.2)',
      itemSelectedBg: 'rgba(255, 255, 255, 0.35)',
      itemHoverBg: 'rgba(255, 255, 255, 0.2)',
      itemSelectedColor: '#FFFFFF',
      itemColor: 'rgba(255, 255, 255, 0.9)',
      itemHoverColor: '#FFFFFF',
      itemHeight: 44,
    },
    Table: {
      headerBg: '#F8F8F8',
      headerColor: '#1A1A1A',
      rowHoverBg: 'rgba(129, 216, 208, 0.05)',
      borderColor: '#E8E8E8',
      headerSplitColor: '#E8E8E8',
    },
    Card: {
      paddingLG: 24,
      borderRadiusLG: 8,
      colorBorderSecondary: '#E8E8E8',
      boxShadowTertiary: '0 2px 8px rgba(0, 0, 0, 0.04)',
    },
    Button: {
      controlHeight: 38,
      controlHeightSM: 30,
      controlHeightLG: 46,
      borderRadius: 4,
      primaryShadow: '0 2px 8px rgba(129, 216, 208, 0.3)',
    },
    Input: {
      controlHeight: 38,
      controlHeightSM: 30,
      controlHeightLG: 46,
      borderRadius: 4,
      activeShadow: '0 0 0 2px rgba(129, 216, 208, 0.1)',
    },
    Select: {
      controlHeight: 38,
      controlHeightSM: 30,
      borderRadius: 4,
    },
    Tabs: {
      horizontalMargin: '0',
      itemSelectedColor: '#0ABAB5',
      itemHoverColor: '#81D8D0',
      inkBarColor: '#81D8D0',
    },
    Modal: {
      contentBg: '#FFFFFF',
      headerBg: '#FFFFFF',
      footerBg: '#FFFFFF',
      titleColor: '#1A1A1A',
      borderRadiusLG: 8,
    },
    Tag: {
      borderRadiusSM: 4,
    },
    Progress: {
      remainingColor: '#F0F0F0',
    },
    Switch: {
      colorPrimary: '#81D8D0',
      colorPrimaryHover: '#5DCDC3',
    },
  },
};
