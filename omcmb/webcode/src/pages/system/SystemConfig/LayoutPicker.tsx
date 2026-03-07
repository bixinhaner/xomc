import { Radio, Button, Divider, Switch } from 'antd';
import {
  SkinOutlined,
  ExperimentOutlined,
  CoffeeOutlined,
  ThunderboltOutlined,
  SmileOutlined,
  StarOutlined,
  DollarOutlined,
  CheckCircleFilled,
} from '@ant-design/icons';
import { useAppStore } from '@/store/appStore';
import { useT } from '@/hooks/useT';
import type { SidebarPosition, TabBarPosition, Theme } from '@/types/common';
import styles from './LayoutPicker.module.css';

const STYLE_OPTIONS: {
  key: Theme;
  icon: React.ReactNode;
  labelKey: string;
  descKey: string;
  colors: { header: string; sidebar: string; primary: string; bg: string };
}[] = [
  {
    key: 'classic',
    icon: <SkinOutlined />,
    labelKey: 'style.classic',
    descKey: 'style.classicDesc',
    colors: { header: '#001529', sidebar: '#001529', primary: '#1677FF', bg: '#F5F5F5' },
  },
  {
    key: 'tech',
    icon: <ExperimentOutlined />,
    labelKey: 'style.tech',
    descKey: 'style.techDesc',
    colors: { header: '#0D1117', sidebar: '#0D1117', primary: '#00D4FF', bg: '#0D1117' },
  },
  {
    key: 'fresh',
    icon: <CoffeeOutlined />,
    labelKey: 'style.fresh',
    descKey: 'style.freshDesc',
    colors: { header: '#FFFFFF', sidebar: '#FFFFFF', primary: '#6366F1', bg: '#FAFAF9' },
  },
  {
    key: 'cyberpunk',
    icon: <ThunderboltOutlined />,
    labelKey: 'style.cyberpunk',
    descKey: 'style.cyberpunkDesc',
    colors: { header: '#0A0A1A', sidebar: '#0A0A1A', primary: '#00F5FF', bg: '#0A0A1A' },
  },
  {
    key: 'minions',
    icon: <SmileOutlined />,
    labelKey: 'style.minions',
    descKey: 'style.minionsDesc',
    colors: { header: '#FFD93D', sidebar: '#4169E1', primary: '#FFD93D', bg: '#FFFEF5' },
  },
  {
    key: 'tiffany',
    icon: <StarOutlined />,
    labelKey: 'style.tiffany',
    descKey: 'style.tiffanyDesc',
    colors: { header: '#FFFFFF', sidebar: '#81D8D0', primary: '#81D8D0', bg: '#FEFEFE' },
  },
  {
    key: 'rmb',
    icon: <DollarOutlined />,
    labelKey: 'style.rmb',
    descKey: 'style.rmbDesc',
    colors: { header: '#E60012', sidebar: '#8B0000', primary: '#E60012', bg: '#FFFEF5' },
  },
];

export default function LayoutPicker() {
  const t = useT();
  const sidebarPosition = useAppStore((s) => s.sidebarPosition);
  const tabBarPosition = useAppStore((s) => s.tabBarPosition);
  const setSidebarPosition = useAppStore((s) => s.setSidebarPosition);
  const setTabBarPosition = useAppStore((s) => s.setTabBarPosition);
  const theme = useAppStore((s) => s.theme);
  const setTheme = useAppStore((s) => s.setTheme);
  const effects3D = useAppStore((s) => s.effects3DEnabled);
  const setEffects3D = useAppStore((s) => s.setEffects3DEnabled);

  const handleReset = () => {
    setSidebarPosition('left');
    setTabBarPosition('top');
  };

  const isDefault = sidebarPosition === 'left' && tabBarPosition === 'top';

  return (
    <div className={styles.container}>
      {/* Style Picker section */}
      <div className={styles.section}>
        <div className={styles.sectionTitle}>{t('style.title')}</div>
        <div className={styles.sectionSubtitle}>{t('style.subtitle')}</div>
        <div className={styles.styleCards}>
          {STYLE_OPTIONS.map((opt) => (
            <button
              key={opt.key}
              type="button"
              className={`${styles.styleCard} ${theme === opt.key ? styles.styleCardActive : ''}`}
              onClick={() => setTheme(opt.key)}
            >
              {/* Mini preview */}
              <div className={styles.stylePreview}>
                <div className={styles.stylePreviewHeader} style={{ background: opt.colors.header }} />
                <div className={styles.stylePreviewBody}>
                  <div className={styles.stylePreviewSidebar} style={{ background: opt.colors.sidebar }} />
                  <div className={styles.stylePreviewContent} style={{ background: opt.colors.bg }}>
                    <div className={styles.stylePreviewAccent} style={{ background: opt.colors.primary }} />
                  </div>
                </div>
              </div>
              <div className={styles.styleInfo}>
                <span className={styles.styleIcon} style={{ color: opt.colors.primary }}>{opt.icon}</span>
                <span className={styles.styleName}>{t(opt.labelKey)}</span>
                {theme === opt.key && <CheckCircleFilled className={styles.styleCheck} />}
              </div>
              <div className={styles.styleDesc}>{t(opt.descKey)}</div>
            </button>
          ))}
        </div>
        <div className={styles.controlGroup} style={{ marginTop: 16 }}>
          <label className={styles.label} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Switch size="small" checked={effects3D} onChange={setEffects3D} />
            <span>{t('effects3d.toggle')}</span>
          </label>
          <div className={styles.sectionSubtitle} style={{ marginTop: 4 }}>{t('effects3d.description')}</div>
        </div>
      </div>

      <Divider />

      {/* Layout Picker section */}
      <div className={styles.section}>
        <div className={styles.sectionTitle}>{t('layout.title')}</div>
        <div className={styles.sectionSubtitle}>{t('layout.subtitle')}</div>
        <div className={styles.wrapper}>
          <div className={styles.controls}>
            <div className={styles.controlGroup}>
              <label className={styles.label}>{t('layout.sidebarPosition')}</label>
              <Radio.Group
                value={sidebarPosition}
                onChange={(e) => setSidebarPosition(e.target.value as SidebarPosition)}
                optionType="button"
                buttonStyle="solid"
                size="small"
              >
                <Radio.Button value="left">{t('layout.sidebarLeft')}</Radio.Button>
                <Radio.Button value="right">{t('layout.sidebarRight')}</Radio.Button>
                <Radio.Button value="top">{t('layout.sidebarTop')}</Radio.Button>
              </Radio.Group>
            </div>

            <div className={styles.controlGroup}>
              <label className={styles.label}>{t('layout.tabBarPosition')}</label>
              <Radio.Group
                value={tabBarPosition}
                onChange={(e) => setTabBarPosition(e.target.value as TabBarPosition)}
                optionType="button"
                buttonStyle="solid"
                size="small"
              >
                <Radio.Button value="top">{t('layout.tabBarTop')}</Radio.Button>
                <Radio.Button value="bottom">{t('layout.tabBarBottom')}</Radio.Button>
                <Radio.Button value="left">{t('layout.tabBarLeft')}</Radio.Button>
              </Radio.Group>
            </div>

            <Button size="small" onClick={handleReset} disabled={isDefault}>
              {t('layout.resetDefault')}
            </Button>
          </div>

          <Divider type="vertical" className={styles.vertDivider} />

          {/* Miniature layout preview */}
          <div className={styles.preview}>
            <div className={styles.previewLabel}>{t('layout.preview')}</div>
            <div
              className={styles.miniLayout}
              data-sidebar={sidebarPosition}
              data-tabbar={tabBarPosition}
            >
              <div className={styles.miniHeader}>Header</div>
              <div className={styles.miniSidebar}>Menu</div>
              <div className={styles.miniTabBar}>Tabs</div>
              <div className={styles.miniContent}>Content</div>
              <div className={styles.miniTaskPanel} />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
