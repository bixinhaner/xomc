import type { ThemeConfig } from 'antd';
import {
  COLOR_PRIMARY_600,
  COLOR_NEUTRAL_50,
  COLOR_NEUTRAL_600,
  COLOR_NEUTRAL_800,
  FONT_FAMILY,
  FONT_SIZE_BASE,
  RADIUS_SM,
  SHADOW_SM,
  SHADOW_MD,
} from './tokens';

export const antdClassicTheme: ThemeConfig = {
  token: {
    colorPrimary: COLOR_PRIMARY_600,
    colorBgLayout: COLOR_NEUTRAL_50,
    colorText: COLOR_NEUTRAL_600,
    colorTextHeading: COLOR_NEUTRAL_800,
    fontFamily: FONT_FAMILY,
    fontSize: FONT_SIZE_BASE,
    borderRadius: RADIUS_SM,
    boxShadow: SHADOW_SM,
    boxShadowSecondary: SHADOW_MD,
    controlHeight: 32,
    wireframe: false,
  },
  components: {
    Layout: {
      headerBg: '#001529',
      headerHeight: 48,
      siderBg: '#001529',
      bodyBg: COLOR_NEUTRAL_50,
    },
    Menu: {
      darkItemBg: '#001529',
      darkSubMenuItemBg: '#000C17',
      darkItemSelectedBg: COLOR_PRIMARY_600,
      darkItemHoverBg: 'rgba(255,255,255,0.08)',
      itemHeight: 40,
    },
    Table: {
      headerBg: '#FAFAFA',
      headerColor: COLOR_NEUTRAL_800,
      rowHoverBg: '#F5F5F5',
      headerSortActiveBg: '#F0F0F0',
      cellPaddingBlock: 8,
      cellPaddingInline: 16,
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
