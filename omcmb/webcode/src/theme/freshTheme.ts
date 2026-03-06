import type { ThemeConfig } from 'antd';

export const antdFreshTheme: ThemeConfig = {
  token: {
    colorPrimary: '#6366F1',
    colorBgLayout: '#F5F5F4',
    colorBgContainer: '#FFFFFF',
    colorBgElevated: '#FFFFFF',
    colorBorderSecondary: '#E7E5E4',
    colorBorder: '#D6D3D1',
    colorText: '#57534E',
    colorTextSecondary: '#78716C',
    colorTextHeading: '#292524',
    borderRadius: 8,
    wireframe: false,
  },
  components: {
    Layout: {
      headerBg: '#FFFFFF',
      siderBg: '#FFFFFF',
      bodyBg: '#F5F5F4',
    },
    Menu: {
      itemBg: '#FFFFFF',
      subMenuItemBg: '#FAFAF9',
      itemSelectedBg: 'rgba(99, 102, 241, 0.08)',
      itemHoverBg: 'rgba(99, 102, 241, 0.04)',
      itemSelectedColor: '#6366F1',
      itemHeight: 40,
    },
    Table: {
      headerBg: '#FAFAF9',
      headerColor: '#292524',
      rowHoverBg: '#FAFAF9',
    },
    Card: {
      paddingLG: 16,
      borderRadiusLG: 12,
    },
    Button: {
      controlHeight: 32,
      controlHeightSM: 24,
      controlHeightLG: 40,
      borderRadius: 8,
    },
    Input: {
      controlHeight: 32,
      controlHeightSM: 24,
      controlHeightLG: 40,
      borderRadius: 8,
    },
    Select: {
      controlHeight: 32,
      controlHeightSM: 24,
      borderRadius: 8,
    },
    Tabs: {
      horizontalMargin: '0',
    },
  },
};
