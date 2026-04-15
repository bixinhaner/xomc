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
      siderBg: '#1f2937',
      bodyBg: '#f9fafb',
    },
    Menu: {
      itemBg: '#1f2937',
      subMenuItemBg: '#1f2937',
      itemSelectedBg: 'rgba(99, 102, 241, 0.08)',
      itemHoverBg: 'rgba(99, 102, 241, 0.04)',
      itemSelectedColor: '#6366F1',
      itemHeight: 40,
    },
    Table: {
      headerBg: '#f9fafb',
      headerColor: '#6b7280',
      rowHoverBg: '#f1f5f9',
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
