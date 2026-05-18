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
import type { FeatureList } from '@core/services/api/systemLicenseApi';

const { Text } = Typography;

interface FeatureListViewProps {
  featureList: FeatureList;
}

export function FeatureListView({ featureList }: FeatureListViewProps) {
  const t = useT();
  const entries = useMemo(
    () => Object.entries(featureList ?? {}).sort(([a], [b]) => a.localeCompare(b)),
    [featureList],
  );

  if (entries.length === 0) {
    return <Empty description={t('systemLicense.featureList.empty')} />;
  }

  return (
    <div>
      {entries.map(([module, value]) => (
        <FeatureRow key={module} module={module} value={value} />
      ))}
    </div>
  );
}

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
  width: 180,
  flexShrink: 0,
  fontWeight: 600,
  paddingRight: 16,
};

const contentStyle: React.CSSProperties = {
  flex: 1,
};
