import type { ThemeConfig } from 'antd';
import { theme } from 'antd';

export const antdTechTheme: ThemeConfig = {
  algorithm: theme.darkAlgorithm,
  token: {
    colorPrimary: '#00D4FF',
    colorBgLayout: '#0D1117',
    colorBgContainer: '#161B22',
    colorBgElevated: '#1C2128',
    colorBorderSecondary: '#21262D',
    colorBorder: '#30363D',
    colorText: '#C9D1D9',
    colorTextSecondary: '#8B949E',
    colorTextHeading: '#E6EDF3',
    borderRadius: 4,
    wireframe: false,
  },
  components: {
    Layout: {
      headerBg: '#0D1117',
      siderBg: '#0D1117',
      bodyBg: '#0D1117',
    },
    Menu: {
      darkItemBg: '#0D1117',
      darkSubMenuItemBg: '#0A0E14',
      darkItemSelectedBg: 'rgba(0, 212, 255, 0.15)',
      darkItemHoverBg: 'rgba(0, 212, 255, 0.08)',
      darkItemSelectedColor: '#00D4FF',
    },
    Table: {
      headerBg: '#161B22',
      rowHoverBg: '#1C2128',
    },
    Card: {
      paddingLG: 16,
    },
    Button: {
      controlHeight: 32,
      controlHeightSM: 24,
      controlHeightLG: 40,
    },
    Input: {
      controlHeight: 32,
      controlHeightSM: 24,
      controlHeightLG: 40,
    },
    Select: {
      controlHeight: 32,
      controlHeightSM: 24,
    },
    Tabs: {
      horizontalMargin: '0',
    },
  },
};
