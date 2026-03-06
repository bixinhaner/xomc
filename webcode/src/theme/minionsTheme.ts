import type { ThemeConfig } from 'antd';

/**
 * Minions Theme - 萌宠版/小黄人风格
 * 可爱活泼的卡通主题，超大圆角泡泡风格
 */
export const antdMinionsTheme: ThemeConfig = {
  token: {
    colorPrimary: '#FFD93D',
    colorBgLayout: '#FFFEF5',
    colorBgContainer: '#FFFFFF',
    colorBgElevated: '#FFFFFF',
    colorBorder: '#FFE066',
    colorBorderSecondary: '#FFF3B8',
    colorText: '#2D2D2D',
    colorTextSecondary: '#666666',
    colorTextHeading: '#1A1A1A',
    colorTextDisabled: 'rgba(45, 45, 45, 0.3)',
    borderRadius: 20, // 超大圆角泡泡风格
    wireframe: false,
    fontFamily: "'Nunito', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
  },
  components: {
    Layout: {
      headerBg: '#FFD93D',
      headerHeight: 56,
      siderBg: '#4169E1',
      bodyBg: '#FFFEF5',
    },
    Menu: {
      // Light menu for blue sidebar
      itemBg: 'transparent',
      subMenuItemBg: 'rgba(255, 255, 255, 0.1)',
      itemSelectedBg: 'rgba(255, 217, 61, 0.3)',
      itemHoverBg: 'rgba(255, 217, 61, 0.15)',
      itemSelectedColor: '#FFD93D',
      itemColor: 'rgba(255, 255, 255, 0.85)',
      itemHoverColor: '#FFFFFF',
      itemHeight: 48,
      itemBorderRadius: 12,
    },
    Table: {
      headerBg: '#FFF8DC',
      headerColor: '#4169E1',
      rowHoverBg: 'rgba(255, 217, 61, 0.08)',
      borderColor: '#FFE066',
      headerSplitColor: '#FFE066',
      borderRadiusLG: 16,
    },
    Card: {
      paddingLG: 20,
      borderRadiusLG: 24,
      colorBorderSecondary: '#FFE066',
    },
    Button: {
      controlHeight: 40,
      controlHeightSM: 32,
      controlHeightLG: 48,
      borderRadius: 20,
      primaryShadow: '0 4px 12px rgba(255, 217, 61, 0.4)',
    },
    Input: {
      controlHeight: 40,
      controlHeightSM: 32,
      controlHeightLG: 48,
      borderRadius: 20,
      activeShadow: '0 0 0 3px rgba(255, 217, 61, 0.2)',
    },
    Select: {
      controlHeight: 40,
      controlHeightSM: 32,
      borderRadius: 20,
    },
    Tabs: {
      horizontalMargin: '0',
      itemSelectedColor: '#FFD93D',
      itemHoverColor: '#FFD93D',
      inkBarColor: '#FFD93D',
      borderRadius: 16,
    },
    Modal: {
      contentBg: '#FFFFFF',
      headerBg: '#FFFFFF',
      footerBg: '#FFFFFF',
      titleColor: '#4169E1',
      borderRadiusLG: 24,
    },
    Tag: {
      borderRadiusSM: 12,
    },
    Progress: {
      remainingColor: '#FFF8DC',
    },
    Switch: {
      colorPrimary: '#FFD93D',
      colorPrimaryHover: '#FFE570',
    },
    Badge: {
      colorError: '#FF6B35',
    },
  },
};
