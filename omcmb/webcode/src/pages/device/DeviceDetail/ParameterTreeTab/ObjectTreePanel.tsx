import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Typography, Empty, Tooltip, Button, theme } from 'antd';
import { LoadingSpinner } from '@/components/LoadingSpinner';
import {
  PlusOutlined,
  DeleteOutlined,
  ApartmentOutlined,
  CaretDownOutlined,
  CaretRightOutlined,
} from '@ant-design/icons';
import { Virtuoso } from 'react-virtuoso';
import type { ParameterTreeNode } from '@core/types/deviceParameter';
import './ObjectTreePanel.css';

const { Text } = Typography;

// Flattened tree node for virtual list rendering
interface FlatTreeNode {
  key: string;
  name: string;
  fullPath: string;
  depth: number;
  isLeaf: boolean;
  isExpanded: boolean;
  isVisible: boolean;
  isInstance: boolean;
  isMultiInstanceContainer: boolean;
  canAdd: boolean;
  instanceCount?: number;
  maxInstances?: number;
  originalNode: ParameterTreeNode;
}

interface ObjectTreePanelProps {
  treeData: ParameterTreeNode[] | undefined;
  loading: boolean;
  selectedPath: string;
  searchKeyword: string;
  onSelect: (path: string) => void;
  onAddObject?: (objectPath: string) => void;
  onDeleteObject?: (objectPath: string) => void;
}

function HighlightText({ text, keyword }: { text: string; keyword: string }) {
  const { token } = theme.useToken();
  if (!keyword) return <>{text}</>;
  const idx = text.toLowerCase().indexOf(keyword.toLowerCase());
  if (idx === -1) return <>{text}</>;
  return (
    <>
      {text.slice(0, idx)}
      <mark
        style={{
          background: token.colorWarningBg,
          color: token.colorWarningText,
          padding: 0,
          borderRadius: 2,
        }}
      >
        {text.slice(idx, idx + keyword.length)}
      </mark>
      {text.slice(idx + keyword.length)}
    </>
  );
}

function isInstanceNumber(name: string): boolean {
  return /^\d+$/.test(name);
}

function countObjects(nodes: ParameterTreeNode[]): number {
  let count = 0;
  for (const node of nodes) {
    if (node.isObject) {
      count++;
      if (node.children) count += countObjects(node.children);
    }
  }
  return count;
}

// Flatten tree structure while preserving hierarchy info
function flattenTree(
  nodes: ParameterTreeNode[],
  expandedKeys: Set<string>,
  searchKeyword: string,
  parentDepth: number = 0,
  parentMultiInstance: boolean = false,
  parentVisible: boolean = true
): FlatTreeNode[] {
  const result: FlatTreeNode[] = [];

  for (const node of nodes) {
    if (!node.isObject) continue;

    const isInstance = parentMultiInstance && isInstanceNumber(node.name);
    const isMultiInstanceContainer = Boolean(node.multiInstance || node.canAdd);
    const childObjects = node.children?.filter((c) => c.isObject) ?? [];
    const isLeafNode = childObjects.length === 0;
    const isExpanded = expandedKeys.has(node.fullPath);

    // Search filtering
    const nameMatch = searchKeyword
      ? node.name.toLowerCase().includes(searchKeyword.toLowerCase())
      : true;
    const pathMatch = searchKeyword
      ? node.fullPath.toLowerCase().includes(searchKeyword.toLowerCase())
      : true;

    // Check if any descendant matches search
    let descendantMatches = false;
    if (searchKeyword && node.children) {
      const checkDescendants = (children: ParameterTreeNode[]): boolean => {
        for (const child of children) {
          if (!child.isObject) continue;
          if (
            child.name.toLowerCase().includes(searchKeyword.toLowerCase()) ||
            child.fullPath.toLowerCase().includes(searchKeyword.toLowerCase())
          ) {
            return true;
          }
          if (child.children && checkDescendants(child.children)) {
            return true;
          }
        }
        return false;
      };
      descendantMatches = checkDescendants(node.children);
    }

    const isVisible = parentVisible && (!searchKeyword || nameMatch || pathMatch || descendantMatches);

    const flatNode: FlatTreeNode = {
      key: node.fullPath,
      name: node.name,
      fullPath: node.fullPath,
      depth: parentDepth,
      isLeaf: isLeafNode,
      isExpanded,
      isVisible,
      isInstance,
      isMultiInstanceContainer,
      canAdd: Boolean(node.canAdd),
      instanceCount: node.instanceCount,
      maxInstances: node.maxInstances,
      originalNode: node,
    };

    if (isVisible) {
      result.push(flatNode);
    }

    // Recursively process children - only if expanded
    // Search auto-expansion is handled by useEffect which adds matching keys to expandedKeys
    // So we only need to check isExpanded here to respect user's manual collapse action
    if (node.children && isExpanded) {
      const childResults = flattenTree(
        node.children,
        expandedKeys,
        searchKeyword,
        parentDepth + 1,
        isMultiInstanceContainer,
        isVisible
      );
      result.push(...childResults);
    }
  }

  return result;
}

// Collect all keys for expansion
function _collectAllKeys(nodes: ParameterTreeNode[]): string[] {
  const keys: string[] = [];
  for (const node of nodes) {
    if (node.isObject) {
      keys.push(node.fullPath);
      if (node.children) keys.push(..._collectAllKeys(node.children));
    }
  }
  return keys;
}
void _collectAllKeys;

// Collect keys that match search or have matching descendants
function collectSearchMatchKeys(
  nodes: ParameterTreeNode[],
  keyword: string
): string[] {
  const keys: string[] = [];
  const lk = keyword.toLowerCase();

  const collect = (nodeList: ParameterTreeNode[]): boolean => {
    let hasMatch = false;
    for (const node of nodeList) {
      if (!node.isObject) continue;

      const nameMatch = node.name.toLowerCase().includes(lk);
      const pathMatch = node.fullPath.toLowerCase().includes(lk);
      const childHasMatch = node.children
        ? collect(node.children)
        : false;

      if (nameMatch || pathMatch || childHasMatch) {
        keys.push(node.fullPath);
        hasMatch = true;
      }
    }
    return hasMatch;
  };

  collect(nodes);
  return keys;
}

// Tree Node Component
interface TreeNodeItemProps {
  node: FlatTreeNode;
  searchKeyword: string;
  isSelected: boolean;
  onSelect: (path: string) => void;
  onToggleExpand: (key: string) => void;
  onAdd?: (path: string) => void;
  onDelete?: (path: string) => void;
}

const TreeNodeItem = React.memo<TreeNodeItemProps>(
  ({
    node,
    searchKeyword,
    isSelected,
    onSelect,
    onToggleExpand,
    onAdd,
    onDelete,
  }) => {
    const handleClick = useCallback(() => {
      const path = node.fullPath.endsWith('.') ? node.fullPath : node.fullPath + '.';
      onSelect(path);
    }, [node.fullPath, onSelect]);

    const handleToggle = useCallback(
      (e: React.MouseEvent) => {
        e.stopPropagation();
        onToggleExpand(node.key);
      },
      [node.key, onToggleExpand]
    );

    const handleAdd = useCallback(
      (e: React.MouseEvent) => {
        e.stopPropagation();
        onAdd?.(node.fullPath + '.');
      },
      [node.fullPath, onAdd]
    );

    const handleDelete = useCallback(
      (e: React.MouseEvent) => {
        e.stopPropagation();
        onDelete?.(node.fullPath + '.');
      },
      [node.fullPath, onDelete]
    );

    const paddingLeft = 12 + node.depth * 16;

    return (
      <div
        className={`virtuoso-tree-node ${isSelected ? 'virtuoso-tree-node-selected' : ''}`}
        style={{ paddingLeft }}
        onClick={handleClick}
      >
        {/* Switcher */}
        <span
          className="virtuoso-tree-switcher"
          onClick={node.isLeaf ? undefined : handleToggle}
          style={{ cursor: node.isLeaf ? 'default' : 'pointer' }}
        >
          {node.isLeaf ? (
            <span style={{ width: 14, display: 'inline-block' }} />
          ) : node.isExpanded ? (
            <CaretDownOutlined style={{ fontSize: 10 }} />
          ) : (
            <CaretRightOutlined style={{ fontSize: 10 }} />
          )}
        </span>

        {/* Content */}
        <span className="tree-node-title-row">
          <span className="tree-node-label">
            {node.isInstance ? (
              <span className="tree-node-instance-name">#{node.name}</span>
            ) : (
              <span className="tree-node-name">
                {searchKeyword ? (
                  <HighlightText text={node.name} keyword={searchKeyword} />
                ) : (
                  node.name
                )}
              </span>
            )}
            {node.isMultiInstanceContainer && (
              <span className="tree-node-mi-badge">
                {node.instanceCount ?? 0}
                {node.maxInstances ? (
                  <span className="tree-node-mi-max">/{node.maxInstances}</span>
                ) : (
                  ''
                )}
              </span>
            )}
          </span>
          <span className="tree-node-actions">
            {node.isMultiInstanceContainer && node.canAdd && onAdd && (
              <Tooltip title="添加实例" mouseEnterDelay={0.5}>
                <Button
                  type="text"
                  size="small"
                  className="tree-action-btn tree-action-add"
                  icon={<PlusOutlined />}
                  onClick={handleAdd}
                />
              </Tooltip>
            )}
            {node.isInstance && onDelete && (
              <Tooltip title={`删除实例 #${node.name}`} mouseEnterDelay={0.5}>
                <Button
                  type="text"
                  size="small"
                  className="tree-action-btn tree-action-delete"
                  icon={<DeleteOutlined />}
                  onClick={handleDelete}
                />
              </Tooltip>
            )}
          </span>
        </span>
      </div>
    );
  }
);

TreeNodeItem.displayName = 'TreeNodeItem';

export default function ObjectTreePanel({
  treeData,
  loading,
  selectedPath,
  searchKeyword,
  onSelect,
  onAddObject,
  onDeleteObject,
}: ObjectTreePanelProps) {
  const [expandedKeys, setExpandedKeys] = useState<Set<string>>(new Set());

  // Initialize expanded keys on first load
  useEffect(() => {
    if (treeData && treeData.length > 0 && expandedKeys.size === 0) {
      const initialKeys = treeData
        .filter((n) => n.isObject)
        .map((n) => n.fullPath);
      setExpandedKeys(new Set(initialKeys));
    }
  }, [treeData]); // eslint-disable-line react-hooks/exhaustive-deps

  // Auto-expand on search
  useEffect(() => {
    if (searchKeyword && treeData) {
      const matchKeys = collectSearchMatchKeys(treeData, searchKeyword);
      setExpandedKeys((prev) => new Set([...prev, ...matchKeys]));
    }
  }, [searchKeyword, treeData]);

  const totalObjects = useMemo(() => {
    if (!treeData) return 0;
    return countObjects(treeData);
  }, [treeData]);

  // Flatten tree for virtual rendering
  const flatNodes = useMemo(() => {
    if (!treeData) return [];
    return flattenTree(treeData, expandedKeys, searchKeyword);
  }, [treeData, expandedKeys, searchKeyword]);

  const handleToggleExpand = useCallback((key: string) => {
    setExpandedKeys((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  }, []);

  const handleSelect = useCallback(
    (path: string) => {
      onSelect(path);
    },
    [onSelect]
  );

  const normalizedSelectedPath = selectedPath?.replace(/\.$/, '');

  if (loading) {
    return (
      <div className="tree-panel-empty">
        <LoadingSpinner tip="加载参数树..." />
      </div>
    );
  }

  if (!treeData || treeData.length === 0) {
    return (
      <div className="tree-panel-empty">
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description="暂无参数数据"
        />
      </div>
    );
  }

  return (
    <div className="object-tree-panel">
      {/* Panel Header */}
      <div className="tree-panel-header">
        <div className="tree-panel-title">
          <ApartmentOutlined />
          <span>对象树</span>
        </div>
        <Text type="secondary" style={{ fontSize: 11 }}>
          {totalObjects} 个对象
        </Text>
      </div>

      {/* Tree Body — virtual scroll with Virtuoso */}
      <div className="tree-panel-body">
        <Virtuoso
          data={flatNodes}
          itemContent={(_index, node) => (
            <TreeNodeItem
              key={node.key}
              node={node}
              searchKeyword={searchKeyword}
              isSelected={node.key === normalizedSelectedPath}
              onSelect={handleSelect}
              onToggleExpand={handleToggleExpand}
              onAdd={onAddObject}
              onDelete={onDeleteObject}
            />
          )}
          overscan={20}
          style={{ height: '100%' }}
        />
      </div>

      {/* Panel Footer */}
      {selectedPath && (
        <div className="tree-panel-footer">
          <Text type="secondary" ellipsis style={{ fontSize: 11 }}>
            {selectedPath}
          </Text>
        </div>
      )}
    </div>
  );
}
