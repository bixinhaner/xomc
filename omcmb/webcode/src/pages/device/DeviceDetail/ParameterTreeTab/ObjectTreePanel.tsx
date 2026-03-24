import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Tree, Typography, Spin, Empty, Tooltip, Button } from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  ApartmentOutlined,
  CaretDownOutlined,
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
      <mark style={{ background: '#fff1b8', padding: 0, borderRadius: 2 }}>
        {text.slice(idx, idx + keyword.length)}
      </mark>
      {text.slice(idx + keyword.length)}
    </>
  );
}

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
      const childObjects = node.children?.filter((c) => c.isObject) ?? [];
      const isLeafNode = childObjects.length === 0;

      return {
        key: node.fullPath,
        title: (
          <span className="tree-node-title-row">
            <span className="tree-node-label">
              {isInstance ? (
                <span className="tree-node-instance-name">
                  #{node.name}
                </span>
              ) : (
                <span className="tree-node-name">
                  {searchKeyword ? highlightText(node.name, searchKeyword) : node.name}
                </span>
              )}
              {isMultiInstanceContainer && (
                <span className="tree-node-mi-badge">
                  {node.instanceCount ?? 0}
                  {node.maxInstances ? <span className="tree-node-mi-max">/{node.maxInstances}</span> : ''}
                </span>
              )}
            </span>
            <span className="tree-node-actions">
              {isMultiInstanceContainer && node.canAdd && onAdd && (
                <Tooltip title="添加实例" mouseEnterDelay={0.5}>
                  <Button
                    type="text"
                    size="small"
                    className="tree-action-btn tree-action-add"
                    icon={<PlusOutlined />}
                    onClick={(e) => {
                      e.stopPropagation();
                      onAdd(node.fullPath + '.');
                    }}
                  />
                </Tooltip>
              )}
              {isInstance && onDelete && (
                <Tooltip title={`删除实例 #${node.name}`} mouseEnterDelay={0.5}>
                  <Button
                    type="text"
                    size="small"
                    className="tree-action-btn tree-action-delete"
                    icon={<DeleteOutlined />}
                    onClick={(e) => {
                      e.stopPropagation();
                      onDelete(node.fullPath + '.');
                    }}
                  />
                </Tooltip>
              )}
            </span>
          </span>
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
        isLeaf: isLeafNode,
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

  const totalObjects = useMemo(() => {
    if (!treeData) return 0;
    return countObjects(treeData);
  }, [treeData]);

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
      <div className="tree-panel-empty">
        <Spin tip="加载参数树..." />
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

      {/* Tree Body */}
      <div className="tree-panel-body">
        <Tree
          treeData={antdTreeData}
          expandedKeys={activeExpandedKeys}
          onExpand={handleExpand}
          selectedKeys={
            selectedPath ? [selectedPath.replace(/\.$/, '')] : []
          }
          onSelect={handleSelect}
          autoExpandParent={Boolean(searchKeyword)}
          blockNode
          switcherIcon={({ expanded }: { expanded?: boolean }) => (
            <CaretDownOutlined
              style={{
                fontSize: 10,
                color: '#8c8c8c',
                transition: 'transform 0.2s',
                transform: expanded ? 'rotate(0deg)' : 'rotate(-90deg)',
              }}
            />
          )}
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

      <style>{treeStyles}</style>
    </div>
  );
}

const treeStyles = `
  .object-tree-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: #fff;
  }

  .tree-panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 14px;
    border-bottom: 1px solid #f0f0f0;
    flex-shrink: 0;
  }

  .tree-panel-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 13px;
    font-weight: 600;
    color: #262626;
  }

  .tree-panel-title .anticon {
    color: #1677ff;
    font-size: 14px;
  }

  .tree-panel-body {
    flex: 1;
    overflow: auto;
    padding: 6px 0;
  }

  .tree-panel-footer {
    padding: 6px 12px;
    border-top: 1px solid #f5f5f5;
    background: #fafafa;
    flex-shrink: 0;
  }

  .tree-panel-empty {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    min-height: 200px;
  }

  /* Tree node layout */
  .tree-node-title-row {
    display: inline-flex;
    align-items: center;
    justify-content: space-between;
    width: calc(100% - 4px);
    min-height: 26px;
    padding-right: 4px;
  }

  .tree-node-label {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
    overflow: hidden;
  }

  .tree-node-name {
    font-size: 13px;
    color: #262626;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    font-weight: 500;
    letter-spacing: 0.01em;
  }

  .tree-node-instance-name {
    font-size: 12px;
    color: #8c8c8c;
    font-family: 'SF Mono', 'Monaco', 'Menlo', 'Consolas', monospace;
    font-weight: 500;
  }

  .tree-node-mi-badge {
    display: inline-flex;
    align-items: center;
    font-size: 10px;
    font-weight: 600;
    color: #1677ff;
    background: #e6f4ff;
    border: 1px solid #bae0ff;
    border-radius: 10px;
    padding: 0 6px;
    height: 16px;
    line-height: 16px;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .tree-node-mi-max {
    color: #8c8c8c;
    font-weight: 400;
  }

  /* Action buttons - hidden by default, show on hover */
  .tree-node-actions {
    display: inline-flex;
    align-items: center;
    gap: 0;
    opacity: 0;
    transition: opacity 0.15s;
    flex-shrink: 0;
  }

  .ant-tree-treenode:hover .tree-node-actions {
    opacity: 1;
  }

  .tree-action-btn {
    width: 22px !important;
    height: 22px !important;
    min-width: 22px !important;
    padding: 0 !important;
    border-radius: 4px !important;
    display: inline-flex !important;
    align-items: center;
    justify-content: center;
  }

  .tree-action-btn .anticon {
    font-size: 11px;
  }

  .tree-action-add {
    color: #52c41a !important;
  }
  .tree-action-add:hover {
    background: #f6ffed !important;
  }

  .tree-action-delete {
    color: #ff4d4f !important;
  }
  .tree-action-delete:hover {
    background: #fff2f0 !important;
  }

  /* Override Ant Design Tree styles for cleaner look */
  .object-tree-panel .ant-tree {
    background: transparent;
    font-size: 13px;
  }

  .object-tree-panel .ant-tree .ant-tree-treenode {
    padding: 0 4px;
    border-radius: 6px;
    margin: 0 6px;
    transition: background 0.15s;
  }

  .object-tree-panel .ant-tree .ant-tree-treenode:hover {
    background: #f5f5f5;
  }

  .object-tree-panel .ant-tree .ant-tree-treenode-selected,
  .object-tree-panel .ant-tree .ant-tree-treenode-selected:hover {
    background: #e6f4ff;
  }

  .object-tree-panel .ant-tree .ant-tree-node-content-wrapper {
    padding: 2px 4px;
    border-radius: 4px;
    min-height: 28px;
    line-height: 28px;
  }

  .object-tree-panel .ant-tree .ant-tree-node-content-wrapper:hover {
    background: transparent;
  }

  .object-tree-panel .ant-tree .ant-tree-node-content-wrapper.ant-tree-node-selected {
    background: transparent;
    color: #1677ff;
  }

  .object-tree-panel .ant-tree .ant-tree-node-content-wrapper.ant-tree-node-selected .tree-node-name {
    color: #1677ff;
  }

  /* Switcher (expand/collapse arrow) */
  .object-tree-panel .ant-tree .ant-tree-switcher {
    width: 20px;
    height: 28px;
    line-height: 28px;
    color: #bfbfbf;
  }

  .object-tree-panel .ant-tree .ant-tree-switcher:hover {
    color: #1677ff;
  }

  /* Indent guide */
  .object-tree-panel .ant-tree .ant-tree-indent-unit {
    width: 16px;
  }

  /* Scrollbar styling */
  .tree-panel-body::-webkit-scrollbar {
    width: 4px;
  }
  .tree-panel-body::-webkit-scrollbar-thumb {
    background: #d9d9d9;
    border-radius: 4px;
  }
  .tree-panel-body::-webkit-scrollbar-thumb:hover {
    background: #bfbfbf;
  }
  .tree-panel-body::-webkit-scrollbar-track {
    background: transparent;
  }
`;
