import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Tree, Tag, Space, Typography, Spin, Empty, Tooltip, Button } from 'antd';
import {
  FolderOutlined,
  InfoCircleOutlined,
  PlusOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import type { ParameterTreeNode } from '@/types/deviceParameter';

const { Text } = Typography;

interface ObjectTreePanelProps {
  treeData: ParameterTreeNode[] | undefined;
  loading: boolean;
  selectedPath: string;
  searchKeyword: string;
  onSelect: (path: string) => void;
  onAddObject?: (objectPath: string) => void;
  onDeleteObject?: (objectPath: string) => void;
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

/** Check if a node name is a numeric instance identifier */
function isInstanceNumber(name: string): boolean {
  return /^\d+$/.test(name);
}

function convertToAntdTree(
  nodes: ParameterTreeNode[],
  searchKeyword: string,
  parentMultiInstance: boolean,
  onAdd?: (path: string) => void,
  onDelete?: (path: string) => void
): DataNode[] {
  return nodes
    .filter((n) => n.isObject)
    .map((node) => {
      const isInstance = parentMultiInstance && isInstanceNumber(node.name);
      const isMultiInstanceContainer = Boolean(node.multiInstance || node.canAdd);

      return {
        key: node.fullPath,
        icon: <FolderOutlined />,
        title: (
          <Space size={4} style={{ width: '100%', justifyContent: 'space-between' }}>
            <Space size={4}>
              <Text strong style={{ fontSize: 13 }}>
                {searchKeyword
                  ? highlightText(node.name, searchKeyword)
                  : node.name}
              </Text>
              {isMultiInstanceContainer && (
                <Tag color="geekblue" style={{ fontSize: 10, lineHeight: '16px', padding: '0 4px', margin: 0 }}>
                  {node.instanceCount ?? 0}
                  {node.maxInstances ? `/${node.maxInstances}` : ''}
                </Tag>
              )}
              {node.description && (
                <Tooltip title={node.description}>
                  <InfoCircleOutlined
                    style={{ color: '#8c8c8c', fontSize: 12 }}
                  />
                </Tooltip>
              )}
            </Space>
            <Space size={2}>
              {isMultiInstanceContainer && node.canAdd && onAdd && (
                <Tooltip title="添加实例">
                  <Button
                    type="text"
                    size="small"
                    icon={<PlusOutlined style={{ fontSize: 12 }} />}
                    style={{ width: 20, height: 20, minWidth: 20, padding: 0, color: '#52c41a' }}
                    onClick={(e) => {
                      e.stopPropagation();
                      onAdd(node.fullPath + '.');
                    }}
                  />
                </Tooltip>
              )}
              {isInstance && onDelete && (
                <Tooltip title={`删除实例 ${node.name}`}>
                  <Button
                    type="text"
                    size="small"
                    icon={<DeleteOutlined style={{ fontSize: 12 }} />}
                    style={{ width: 20, height: 20, minWidth: 20, padding: 0, color: '#ff4d4f' }}
                    onClick={(e) => {
                      e.stopPropagation();
                      onDelete(node.fullPath + '.');
                    }}
                  />
                </Tooltip>
              )}
            </Space>
          </Space>
        ),
        children: node.children
          ? convertToAntdTree(
              node.children,
              searchKeyword,
              isMultiInstanceContainer,
              onAdd,
              onDelete
            )
          : undefined,
        isLeaf:
          !node.children ||
          node.children.filter((c) => c.isObject).length === 0,
      };
    });
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
  onAddObject,
  onDeleteObject,
}: ObjectTreePanelProps) {
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>([]);

  const filteredData = useMemo(() => {
    if (!treeData) return [];
    return filterTree(treeData, searchKeyword);
  }, [treeData, searchKeyword]);

  useEffect(() => {
    if (treeData && treeData.length > 0 && expandedKeys.length === 0) {
      setExpandedKeys(
        treeData.filter((n) => n.isObject).map((n) => n.fullPath)
      );
    }
  }, [treeData]); // eslint-disable-line react-hooks/exhaustive-deps

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
    () => convertToAntdTree(filteredData, searchKeyword, false, onAddObject, onDeleteObject),
    [filteredData, searchKeyword, onAddObject, onDeleteObject]
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
