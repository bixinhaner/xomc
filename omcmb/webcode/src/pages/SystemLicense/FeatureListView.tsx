/**
 * FeatureListView — 渲染 system_license.feature_list 三级嵌套结构（PRD §4）。
 *
 * 顶层 key = 一级模块；值可以是：
 *   - string "All"        — 整个模块全部解锁
 *   - string[] / array    — 该模块部分功能
 *   - object              — 嵌套：子模块 → 功能项数组
 *
 * 视觉布局参考老 OMC 页面（§2.3）：每个一级模块一行，左侧 label 右侧详情。
 */
import { useMemo } from 'react';
import { Empty, Tag, Typography } from 'antd';

import { useT } from '@/hooks/useT';
import { useAppStore } from '@core/store/appStore';
import type { FeatureList, SystemLicenseFeature } from '@core/services/api/systemLicenseApi';

const { Text } = Typography;

interface FeatureListViewProps {
  featureList: FeatureList;
}

export function FeatureListView({ featureList }: FeatureListViewProps) {
  const t = useT();
  const normalizedFeatures = Array.isArray(featureList?.features) ? featureList.features : null;
  const entries = useMemo(
    () => Object.entries(featureList ?? {}).sort(([a], [b]) => a.localeCompare(b)),
    [featureList],
  );

  if (normalizedFeatures) {
    if (normalizedFeatures.length === 0) {
      return <Empty description={t('systemLicense.featureList.empty')} />;
    }
    return (
      <div style={featureViewportStyle}>
        <NormalizedFeatureGroups features={normalizedFeatures} />
      </div>
    );
  }

  if (entries.length === 0) {
    return <Empty description={t('systemLicense.featureList.empty')} />;
  }

  return (
    <div style={featureViewportStyle}>
      {entries.map(([module, value]) => (
        <FeatureRow key={module} module={module} value={value} />
      ))}
    </div>
  );
}

const featureViewportStyle: React.CSSProperties = {
  maxHeight: 'min(62vh, 680px)',
  overflowY: 'auto',
  paddingRight: 8,
};
interface NormalizedFeatureGroupsProps {
  features: SystemLicenseFeature[];
}

function NormalizedFeatureGroups({ features }: NormalizedFeatureGroupsProps) {
  const t = useT();
  const locale = useAppStore((state) => state.locale);
  const groups = useMemo(() => {
    const result = new Map<string, Map<string, SystemLicenseFeature[]>>();
    for (const feature of features) {
      const path = locale === 'en-US' ? (feature.pathEn ?? feature.path) : feature.path;
      const parts = (path ?? '').split(' / ').filter(Boolean);
      const root = featureGroupRoot(feature, parts[0] || t('systemLicense.featureList.other'));
      const section = parts[1] || t('systemLicense.featureList.other');
      const sections = result.get(root) ?? new Map<string, SystemLicenseFeature[]>();
      const items = sections.get(section) ?? [];
      items.push(feature);
      sections.set(section, items);
      result.set(root, sections);
    }
    return [...result.entries()].sort(([left], [right]) => compareFeatureNames(left, right, rootOrder));
  }, [features, t, locale]);

  return (
    <div>
      {groups.map(([root, sections]) => (
        <div key={root} style={rowStyle}>
          <div style={labelStyle}>{localizeRoot(technicalRootLabel(root), locale)}</div>
          <div style={sectionGridStyle}>
            {[...sections.entries()]
              .sort(([left], [right]) => compareFeatureNames(left, right, sectionOrder))
              .map(([section, items]) => {
                  const visibleItems = items.filter((feature) =>
                    feature.recognized || feature.nameZh || feature.nameEn,
                  );
                  if (visibleItems.length === 0) return null;
                const direct = (visibleItems.length === 1 && section === featureLabel(visibleItems[0], locale))
                  || (sections.size === 1 && (
                    section === root || section === t('systemLicense.featureList.other')
                  ));
                  const sortedItems = visibleItems
                  .slice()
                  .sort((left, right) => featureLabel(left, locale).localeCompare(featureLabel(right, locale)));
                return (
                  <div key={section} style={direct ? directSectionStyle : sectionStyle}>
                    {direct ? null : (
                      <div style={sectionTitleStyle}>
                        <Text strong>{section}</Text>
                        <Text type="secondary" style={{ marginLeft: 6 }}>({visibleItems.length})</Text>
                      </div>
                    )}
                    <div style={itemsStyle}>
                      {sortedItems.map((feature, index) => (
                        <Tag key={`${feature.featureCode ?? feature.featureId ?? 'feature'}-${index}`}>
                          {featureTagLabel(feature, root, t, locale)}
                        </Tag>
                      ))}
                    </div>
                  </div>
                );
                })}
          </div>
        </div>
      ))}
    </div>
  );
}

const rootOrder = ['Dashboard', '地图', 'eNB', 'gNB', 'CPE', 'EGW', 'UPS', 'EPC', '告警', '性能', '高级', '系统'];
const sectionOrder = ['监控', 'Monitor', '维护', 'Maintenance', '升级&回退', 'Upgrade&Rollback', '升级', 'Upgrade', '设备', 'Inventory'];

function compareFeatureNames(left: string, right: string, preferred: string[]): number {
  const leftIndex = preferred.indexOf(left);
  const rightIndex = preferred.indexOf(right);
  if (leftIndex >= 0 || rightIndex >= 0) {
    return (leftIndex < 0 ? Number.MAX_SAFE_INTEGER : leftIndex)
      - (rightIndex < 0 ? Number.MAX_SAFE_INTEGER : rightIndex);
  }
  return left.localeCompare(right);
}

function featureGroupRoot(feature: SystemLicenseFeature, fallback: string): string {
  const code = feature.featureCode ?? '';
  if (code.startsWith('CODE_ENB_')) return 'eNB';
  if (code.startsWith('CODE_GNB_') || code === 'CODE_GNB') return 'gNB';
  if (code.startsWith('CODE_CPE_')) return 'CPE';
  if (code === 'CODE_EGW') return 'EGW';
  if (code === 'CODE_UPS') return 'UPS';
  if (code === 'CODE_EPC') return 'EPC';
  if (code.startsWith('CODE_ALARM_')) return '告警';
  if (code.startsWith('CODE_PERFORMANCE_')) return '性能';
  if (code.startsWith('CODE_SYSTEM_')) return '系统';
  if (code.startsWith('CODE_ADVANCE_')) return '高级';
  if (code === 'CODE_TOPO') return '地图';
  if (code === 'CODE_DASHBOARD') return 'Dashboard';
  return fallback;
}

function featureLabel(feature: SystemLicenseFeature, locale: string): string {
  if (locale === 'en-US') {
    return feature.nameEn || feature.nameZh || feature.featureCode || feature.featureId || '-';
  }
  return feature.nameZh || feature.nameEn || feature.featureCode || feature.featureId || '-';
}

function featureTagLabel(feature: SystemLicenseFeature, root: string, t: ReturnType<typeof useT>, locale: string): string {
  if (root === 'Dashboard' || root === '地图') {
    return t('systemLicense.featureList.all');
  }
  return featureLabel(feature, locale);
}

function technicalRootLabel(root: string): string {
  switch (root) {
    case '4G基站':
      return 'eNB';
    case '5G基站':
      return 'gNB';
    default:
      return root;
  }
}

function localizeRoot(root: string, locale: string): string {
  if (locale !== 'en-US') return root;
  const labels: Record<string, string> = {
    地图: 'MAP',
    告警: 'Alarm',
    性能: 'Performance',
    高级: 'Advanced',
    系统: 'System',
    其他: 'Other',
  };
  return labels[root] ?? root;
}

const sectionStyle: React.CSSProperties = {
  minWidth: 0,
};

const directSectionStyle: React.CSSProperties = {
  minWidth: 0,
  alignSelf: 'center',
};

const sectionGridStyle: React.CSSProperties = {
  flex: 1,
  display: 'grid',
  gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))',
  columnGap: 24,
  rowGap: 12,
};

const sectionTitleStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  paddingBottom: 6,
  borderBottom: '1px solid #f0f0f0',
};

const itemsStyle: React.CSSProperties = {
  display: 'grid',
  gridTemplateColumns: 'repeat(auto-fill, minmax(130px, 1fr))',
  gap: 8,
  paddingTop: 8,
};

interface FeatureRowProps {
  module: string;
  value: unknown;
}

function FeatureRow({ module, value }: FeatureRowProps) {
  const t = useT();

  // 一级 "All" — 整个模块全开
  if (typeof value === 'string') {
    return (
      <div style={rowStyle}>
        <div style={labelStyle}>{module}</div>
        <div style={contentStyle}>
          <Tag color="green">{value === 'All' ? t('systemLicense.featureList.all') : value}</Tag>
        </div>
      </div>
    );
  }

  // 一级数组 — 多个功能名
  if (Array.isArray(value)) {
    return (
      <div style={rowStyle}>
        <div style={labelStyle}>{module}</div>
        <div style={contentStyle}>
          {value.length === 0 ? (
            <Text type="secondary">-</Text>
          ) : (
            value.map((item, idx) => (
              <Tag key={`${item}-${idx}`} style={{ marginBottom: 4 }}>
                {String(item)}
              </Tag>
            ))
          )}
        </div>
      </div>
    );
  }

  // 二级嵌套：object { subModule: string[] }
  if (value && typeof value === 'object') {
    const subEntries = Object.entries(value as Record<string, unknown>).sort(([a], [b]) =>
      a.localeCompare(b),
    );
    return (
      <div style={rowStyle}>
        <div style={labelStyle}>{module}</div>
        <div style={contentStyle}>
          {subEntries.map(([sub, items]) => (
            <div key={sub} style={{ marginBottom: 8 }}>
              <Text strong>{sub}</Text>
              {Array.isArray(items) ? (
                <span style={{ marginLeft: 4 }}>
                  ({items.length}):{' '}
                  {items.length === 0 ? (
                    <Text type="secondary">-</Text>
                  ) : (
                    items.map((item, idx) => (
                      <Tag key={`${item}-${idx}`} style={{ marginRight: 4, marginBottom: 4 }}>
                        {String(item)}
                      </Tag>
                    ))
                  )}
                </span>
              ) : typeof items === 'string' ? (
                <Tag color="green" style={{ marginLeft: 8 }}>
                  {items === 'All' ? t('systemLicense.featureList.all') : items}
                </Tag>
              ) : (
                <Text type="secondary" style={{ marginLeft: 8 }}>
                  {JSON.stringify(items)}
                </Text>
              )}
            </div>
          ))}
        </div>
      </div>
    );
  }

  // 兜底
  return (
    <div style={rowStyle}>
      <div style={labelStyle}>{module}</div>
      <div style={contentStyle}>
        <Text type="secondary">{JSON.stringify(value)}</Text>
      </div>
    </div>
  );
}

const rowStyle: React.CSSProperties = {
  display: 'flex',
  padding: '12px 0',
  borderBottom: '1px solid #f0f0f0',
};

const labelStyle: React.CSSProperties = {
  width: 96,
  flexShrink: 0,
  fontWeight: 600,
  paddingRight: 16,
};

const contentStyle: React.CSSProperties = {
  flex: 1,
};
