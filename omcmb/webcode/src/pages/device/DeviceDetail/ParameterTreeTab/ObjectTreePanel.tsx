import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Tree, Tag, Space, Typography, Spin, Empty, Tooltip } from 'antd';
import { FolderOutlined, InfoCircleOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import type { ParameterTreeNode } from '@/types/deviceParameter';

const { Text } = Typography;

interface ObjectTreePanelProps {
  treeData: ParameterTreeNode[] | undefined;
  loading: boolean;
  selectedPath: string;
  searchKeyword: string;
  onSelect: (path: string) => void;
}

function highlightText(text: string, keyword: string): React.ReactNode {
  if (!keyword) return text;
  const idx = text.toLowerCase().indexOf(keyword.toLowerCase());
  if (idx === -1) return text;
  return (
    <>
      {text.slice(0, idx)}
      <span style={{ backgroundColor: '#ffd666', padding: '0 1px' }}>
        {text.slice(idx, idx + keyword.length)}
      </span>
      {text.slice(idx + keyword.length)}
    </>
  );
}

function convertToAntdTree(
  nodes: ParameterTreeNode[],
  searchKeyword: string
): DataNode[] {
  return nodes
    .filter((n) => n.isObject)
    .map((node) => ({
      key: node.fullPath,
      icon: <FolderOutlined />,
      title: (
        <Space size={4}>
          <Text strong style={{ fontSize: 13 }}>
            {searchKeyword ? highlightText(node.name, searchKeyword) : node.name}
          </Text>
          {node.multiInstance && (
            <Tag color="geekblue" style={{ fontSize: 11 }}>
              {node.instanceCount ?? 0}/{node.maxInstances ?? '?'}
            </Tag>
          )}
          {node.description && (
            <Tooltip title={node.description}>
              <InfoCircleOutlined style={{ color: '#8c8c8c', fontSize: 12 }} />
            </Tooltip>
          )}
        </Space>
      ),
      children: node.children
        ? convertToAntdTree(node.children, searchKeyword)
        : undefined,
      isLeaf:
        !node.children ||
        node.children.filter((c) => c.isObject).length === 0,
    }));
}

function filterTree(
  nodes: ParameterTreeNode[],
  keyword: string
): ParameterTreeNode[] {
  if (!keyword) return nodes;
  const lk = keyword.toLowerCase();
  return nodes.reduce<ParameterTreeNode[]>((acc, node) => {
    if (!node.isObject) return acc;
    const nameMatch = node.name.toLowerCase().includes(lk);
    const pathMatch = node.fullPath.toLowerCase().includes(lk);
    const filteredChildren = node.children
      ? filterTree(node.children, keyword)
      : [];
    if (filteredChildren.length > 0 || nameMatch || pathMatch) {
      acc.push({
        ...node,
        children:
          filteredChildren.length > 0 ? filteredChildren : node.children,
      });
    }
    return acc;
  }, []);
}

function collectAllKeys(nodes: ParameterTreeNode[]): string[] {
  const keys: string[] = [];
  for (const node of nodes) {
    if (node.isObject) {
      keys.push(node.fullPath);
      if (node.children) keys.push(...collectAllKeys(node.children));
    }
  }
  return keys;
}

export default function ObjectTreePanel({
  treeData,
  loading,
  selectedPath,
  searchKeyword,
  onSelect,
}: ObjectTreePanelProps) {
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>([]);

  const filteredData = useMemo(() => {
    if (!treeData) return [];
    return filterTree(treeData, searchKeyword);
  }, [treeData, searchKeyword]);

  // Initialize: expand first-level nodes when tree data first arrives
  useEffect(() => {
    if (treeData && treeData.length > 0 && expandedKeys.length === 0) {
      setExpandedKeys(
        treeData.filter((n) => n.isObject).map((n) => n.fullPath)
      );
    }
  }, [treeData]); // eslint-disable-line react-hooks/exhaustive-deps

  // When searching, expand all matching nodes; otherwise use user-controlled state
  const activeExpandedKeys = useMemo(() => {
    if (searchKeyword && filteredData.length) {
      return collectAllKeys(filteredData);
    }
    return expandedKeys;
  }, [searchKeyword, filteredData, expandedKeys]);

  const handleExpand = useCallback(
    (keys: React.Key[]) => {
      if (!searchKeyword) setExpandedKeys(keys);
    },
    [searchKeyword]
  );

  const antdTreeData = useMemo(
    () => convertToAntdTree(filteredData, searchKeyword),
    [filteredData, searchKeyword]
  );

  const handleSelect = useCallback(
    (keys: React.Key[]) => {
      if (keys.length > 0) {
        const path = keys[0] as string;
        onSelect(path.endsWith('.') ? path : path + '.');
      }
    },
    [onSelect]
  );

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 48 }}>
        <Spin tip="加载参数树..." />
      </div>
    );
  }

  if (!treeData || treeData.length === 0) {
    return (
      <Empty
        description="暂无参数数据，请先执行参数发现"
        style={{ padding: 48 }}
      />
    );
  }

  return (
    <Tree
      showIcon
      showLine={{ showLeafIcon: false }}
      treeData={antdTreeData}
      expandedKeys={activeExpandedKeys}
      onExpand={handleExpand}
      selectedKeys={
        selectedPath ? [selectedPath.replace(/\.$/, '')] : []
      }
      onSelect={handleSelect}
      autoExpandParent={Boolean(searchKeyword)}
      blockNode
      style={{ padding: '8px 4px' }}
    />
  );
}
