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

// issue #311: 特性列表按现网 OMC 菜单结构展示，不再复刻旧 OMC 的 eNB/gNB/CPE 一级分组。
// root = mapping path 第一段（现网菜单目录：设备管理/软件管理/告警管理…）；
// section = path 第二段（现网菜单：设备详情/版本升级…）；
// 同名功能跨设备类型合并为一个条目，后缀注明包含的设备类型（eNB、gNB、CPE…）。
function NormalizedFeatureGroups({ features }: NormalizedFeatureGroupsProps) {
  const t = useT();
  const locale = useAppStore((state) => state.locale);
  const groups = useMemo(() => {
    const result = new Map<string, Map<string, Map<string, Set<string>>>>();
    for (const feature of features) {
      if (feature.hidden) continue;
      if (!(feature.recognized || feature.nameZh || feature.nameEn)) continue;
      const path = locale === 'en-US' ? (feature.pathEn ?? feature.path) : feature.path;
      const parts = (path ?? '').split(' / ').filter(Boolean);
      const root = parts[0] || t('systemLicense.featureList.other');
      const section = parts[1] || t('systemLicense.featureList.other');
      const label = moduleLevelCode(feature.featureCode)
        ? t('systemLicense.featureList.all')
        : featureLabel(feature, locale);
      const sections = result.get(root) ?? new Map<string, Map<string, Set<string>>>();
      const merged = sections.get(section) ?? new Map<string, Set<string>>();
      const types = merged.get(label) ?? new Set<string>();
      const deviceType = deviceTypeOf(feature.featureCode);
      if (deviceType) types.add(deviceType);
      merged.set(label, types);
      sections.set(section, merged);
      result.set(root, sections);
    }
    return [...result.entries()].sort(([left], [right]) => compareFeatureNames(left, right, rootOrder));
  }, [features, t, locale]);

  return (
    <div>
      {groups.map(([root, sections]) => (
        <div key={root} style={rowStyle}>
          <div style={labelStyle}>{root}</div>
          <div style={sectionGridStyle}>
            {[...sections.entries()]
              .sort(([left], [right]) => compareFeatureNames(left, right, sectionOrder))
              .map(([section, merged]) => {
                const tags = [...merged.entries()]
                  .map(([name, types]) => mergedTagLabel(name, types, locale))
                  .sort((left, right) => left.localeCompare(right));
                const direct = (merged.size === 1 && section === [...merged.keys()][0])
                  || (sections.size === 1 && (
                    section === root || section === t('systemLicense.featureList.other')
                  ));
                return (
                  <div key={section} style={direct ? directSectionStyle : sectionStyle}>
                    {direct ? null : (
                      <div style={sectionTitleStyle}>
                        <Text strong>{section}</Text>
                        <Text type="secondary" style={{ marginLeft: 6 }}>({tags.length})</Text>
                      </div>
                    )}
                    <div style={itemsStyle}>
                      {tags.map((label) => (
                        <Tag key={label}>{label}</Tag>
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

// root/section 与 mapping path 对齐后的现网菜单目录顺序（未列出的按字母序）
const rootOrder = [
  '首页', 'Dashboard',
  '设备管理', '告警管理', '性能管理', 'MML管理', '拓扑管理', '软件管理', '运维管理', '产品中心', '系统管理', '高级', '帮助', '其他',
  'Device Mgmt', 'Alarm Mgmt', 'Performance', 'MML', 'Topology', 'Software', 'O&M', 'Product', 'System', 'Advanced', 'Help', 'Other',
];
const sectionOrder = [
  '设备列表', 'Device List', '设备注册', 'Registration', '即插即用', 'Plug-and-Play', '批量配置', 'Batch Config',
  '设备详情', 'Device Detail', '核心网', 'Core Network',
  '当前告警', 'Current Alarms', '历史告警', 'Historical Alarms', '告警规则', 'Rules', '告警统计', 'Statistics', '告警库', 'Library',
  '指标查询', 'KPI Query', '指标库', 'KPI Library', '测量任务管理', 'Measurement Tasks',
  '拓扑图', 'Canvas',
  '版本升级', 'Upgrade', '升级文件', 'Firmware Files', '版本回退', 'Rollback',
  'MML控制台', 'MML Console',
  '设备日志', 'Device Logs', '运维命令', 'Commands', '网络诊断', 'Diagnostics', 'TR069报文跟踪', 'Message Trace',
  '备份恢复', 'Backup & Restore',
  '系统配置', 'Configuration', '用户管理', 'Users', '角色管理', 'Roles', '日志', 'Logs', 'License',
  '参数模型库', 'Param Model',
];

function compareFeatureNames(left: string, right: string, preferred: string[]): number {
  const leftIndex = preferred.indexOf(left);
  const rightIndex = preferred.indexOf(right);
  if (leftIndex >= 0 || rightIndex >= 0) {
    return (leftIndex < 0 ? Number.MAX_SAFE_INTEGER : leftIndex)
      - (rightIndex < 0 ? Number.MAX_SAFE_INTEGER : rightIndex);
  }
  return left.localeCompare(right);
}

// 模块级整体授权（整个模块一个开关）：首页 / 地图
function moduleLevelCode(code?: string): boolean {
  return code === 'CODE_DASHBOARD' || code === 'CODE_TOPO';
}

// 旧 license code 前缀 → 设备类型；无前缀的裸 code 显式指认
const explicitDeviceTypes: Record<string, string> = {
  CODE_EGW: 'EGW',
  CODE_UPS: 'UPS',
  CODE_EPC: 'EPC',
  CODE_IPSEC_CERT: 'eNB',
};
const deviceTypeOrder = ['eNB', 'gNB', 'CPE', 'EGW', 'UPS', 'EPC'];

function deviceTypeOf(code?: string): string | null {
  if (!code) return null;
  if (explicitDeviceTypes[code]) return explicitDeviceTypes[code];
  if (code.startsWith('CODE_ENB_')) return 'eNB';
  if (code.startsWith('CODE_GNB_') || code === 'CODE_GNB') return 'gNB';
  if (code.startsWith('CODE_CPE_') || code === 'CODE_CPE') return 'CPE';
  return null;
}

function mergedTagLabel(name: string, types: Set<string>, locale: string): string {
  if (types.size === 0) return name;
  const ordered = deviceTypeOrder.filter((type) => types.has(type));
  return locale === 'en-US'
    ? `${name} (${ordered.join(', ')})`
    : `${name}（${ordered.join('、')}）`;
}

function featureLabel(feature: SystemLicenseFeature, locale: string): string {
  if (locale === 'en-US') {
    return feature.nameEn || feature.nameZh || feature.featureCode || feature.featureId || '-';
  }
  return feature.nameZh || feature.nameEn || feature.featureCode || feature.featureId || '-';
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
