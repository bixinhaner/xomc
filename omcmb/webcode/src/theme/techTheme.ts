import type { ThemeConfig } from 'antd';
import { theme } from 'antd';

export const antdTechTheme: ThemeConfig = {
  algorithm: theme.darkAlgorithm,
  token: {
    colorPrimary: '#0071e3',
    colorBgLayout: '#000000',
    colorBgContainer: '#1d1d1f',
    colorBgElevated: '#272729',
    colorBorderSecondary: '#333336',
    colorBorder: '#424245',
    colorText: '#f5f5f7',
    colorTextSecondary: '#86868b',
    colorTextHeading: '#ffffff',
    borderRadius: 8,
    wireframe: false,
  },
  components: {
    Layout: {
      headerBg: '#0D1117',
      siderBg: '#0D1117',
      bodyBg: '#000000',
    },
    Menu: {
      darkItemBg: '#0D1117',
      darkSubMenuItemBg: '#0A0E14',
      darkItemSelectedBg: 'rgba(0, 212, 255, 0.15)',
      darkItemHoverBg: 'rgba(0, 212, 255, 0.08)',
      darkItemSelectedColor: '#00D4FF',
    },
    Table: {
      headerBg: '#1d1d1f',
      rowHoverBg: '#272729',
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
