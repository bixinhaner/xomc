/**
 * 设备组树形筛选组件
 * @module components/GISMap/GroupTree
 */

import React, { useMemo, useState } from 'react';
import { Input, Checkbox, Typography, Spin, Empty } from 'antd';
import { SearchOutlined, MinusOutlined, PlusOutlined } from '@ant-design/icons';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useT } from '@/hooks/useT';
import type { DeviceGroupNode } from '@core/types/map';
import styles from './styles.module.css';

const { Text } = Typography;

interface GroupTreeProps {
  /** 设备组树数据 */
  data?: DeviceGroupNode[];
  /** 选中的设备组ID列表（支持多选） */
  selectedGroupIds?: string[];
  /** 选中变化回调 */
  onSelect: (groupIds: string[]) => void;
  /** 是否显示搜索 */
  showSearch?: boolean;
  /** 加载状态 */
  loading?: boolean;
}

/**
 * 设备组树形筛选
 * 根据 UI 设计图:
 * - 多选支持
 * - +/- 展开/收起图标
 * - 叶子节点无图标
 */
const GroupTree: React.FC<GroupTreeProps> = ({
  data = [],
  selectedGroupIds = [],
  onSelect,
  showSearch = true,
  loading = false,
}) => {
  const t = useT();
  const token = useThemeToken();
  const [searchValue, setSearchValue] = useState('');
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>([]);

  // 过滤树数据
  const filteredData = useMemo(() => {
    if (!searchValue.trim()) return data;

    const filterNodes = (nodes: DeviceGroupNode[]): DeviceGroupNode[] => {
      return nodes.reduce<DeviceGroupNode[]>((acc, node) => {
        const matchesSearch = node.name.toLowerCase().includes(searchValue.toLowerCase());
        const filteredChildren = node.children ? filterNodes(node.children) : [];

        if (matchesSearch || filteredChildren.length > 0) {
          acc.push({
            ...node,
            children: filteredChildren.length > 0 ? filteredChildren : node.children,
          });
        }
        return acc;
      }, []);
    };

    return filterNodes(data);
  }, [data, searchValue]);

  // 收集所有子节点ID
  const getAllDescendantIds = (node: DeviceGroupNode): string[] => {
    const ids = [node.id];
    if (node.children) {
      node.children.forEach(child => {
        ids.push(...getAllDescendantIds(child));
      });
    }
    return ids;
  };

  // 处理复选框变化
  const _handleCheck = (checked: React.Key[], _info: unknown) => {
    onSelect(checked as string[]);
  };

  // 切换展开/收起
  const toggleExpand = (key: string) => {
    setExpandedKeys(prev =>
      prev.includes(key)
        ? prev.filter(k => k !== key)
        : [...prev, key]
    );
  };

  // 渲染树节点
  const renderTreeNodes = (nodes: DeviceGroupNode[]): React.ReactNode[] => {
    return nodes.map(node => {
      const isLeaf = !node.children || node.children.length === 0;
      const isExpanded = expandedKeys.includes(node.id);
      const isSelected = selectedGroupIds.includes(node.id);

      // 递归渲染子节点
      const children = !isLeaf && isExpanded ? renderTreeNodes(node.children!) : null;

      // 缩进级别
      const level = node.level || 1;
      const paddingLeft = (level - 1) * 24 + 12;

      return (
        <div key={node.id}>
          {/* 节点行 */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              padding: '6px 12px',
              paddingLeft,
              cursor: 'pointer',
              background: isSelected
                ? token.colorPrimaryBg
                : 'transparent',
              borderRadius: 6,
              margin: '2px 8px',
              transition: 'background 0.2s',
            }}
            onMouseEnter={(e) => {
              if (!isSelected) {
                e.currentTarget.style.background = token.colorBgTextHover;
              }
            }}
            onMouseLeave={(e) => {
              if (!isSelected) {
                e.currentTarget.style.background = 'transparent';
              }
            }}
          >
            {/* 复选框 */}
            <Checkbox
              checked={selectedGroupIds.includes(node.id)}
              onChange={(e) => {
                e.stopPropagation();
                const ids = getAllDescendantIds(node);
                if (e.target.checked) {
                  onSelect([...new Set([...selectedGroupIds, ...ids])]);
                } else {
                  onSelect(selectedGroupIds.filter(id => !ids.includes(id)));
                }
              }}
              style={{ marginRight: 8 }}
            />

            {/* 展开/收起图标 */}
            {!isLeaf && (
              <div
                onClick={(e) => {
                  e.stopPropagation();
                  toggleExpand(node.id);
                }}
                style={{
                  width: 16,
                  height: 16,
                  borderRadius: 2,
                  border: `1.5px solid ${isExpanded ? token.colorPrimary : token.colorBorder}`,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  marginRight: 8,
                  cursor: 'pointer',
                  flexShrink: 0,
                }}
              >
                {isExpanded ? (
                  <MinusOutlined style={{ fontSize: 10, color: token.colorPrimary }} />
                ) : (
                  <PlusOutlined style={{ fontSize: 10, color: token.colorTextSecondary }} />
                )}
              </div>
            )}

            {/* 节点名称 */}
            <Text
              style={{
                fontSize: level === 1 ? 14 : 13,
                color: isSelected ? token.colorPrimary : token.colorText,
                fontWeight: isSelected ? 500 : 400,
              }}
            >
              {node.name}
            </Text>
          </div>

          {/* 子节点 */}
          {children}
        </div>
      );
    });
  };

  const containerStyle: React.CSSProperties = {
    display: 'flex',
    flexDirection: 'column',
    height: '100%',
  };

  const searchBoxStyle: React.CSSProperties = {
    padding: '0 16px 12px',
  };

  const treeContainerStyle: React.CSSProperties = {
    flex: 1,
    overflow: 'auto',
    padding: '4px 0',
  };

  const summaryStyle: React.CSSProperties = {
    margin: '8px 16px',
    padding: '8px 12px',
    background: token.colorPrimaryBg,
    borderRadius: 8,
  };

  return (
    <div style={containerStyle} className={styles.groupTree}>
      {/* 搜索框 */}
      {showSearch && (
        <div style={searchBoxStyle}>
          <Input
            size="small"
            placeholder={t('device.searchGroup')}
            prefix={<SearchOutlined style={{ color: token.colorTextDisabled }} />}
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            allowClear
            style={{ borderRadius: 8 }}
          />
        </div>
      )}

      {/* 树区域 */}
      <div style={treeContainerStyle}>
        {loading ? (
          <div style={{ textAlign: 'center', padding: 24 }}>
            <Spin size="small" />
          </div>
        ) : filteredData.length === 0 ? (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description={searchValue ? t('common.noSearchResults') : t('device.noGroups')}
            style={{ margin: '24px 0' }}
          />
        ) : (
          renderTreeNodes(filteredData)
        )}
      </div>

      {/* 已选汇总 */}
      {selectedGroupIds.length > 0 && (
        <div style={summaryStyle}>
          <Text style={{ fontSize: 13, color: token.colorPrimary }}>
            {t('device.selectedGroups', { count: selectedGroupIds.length })}
          </Text>
        </div>
      )}
    </div>
  );
};

export default GroupTree;
