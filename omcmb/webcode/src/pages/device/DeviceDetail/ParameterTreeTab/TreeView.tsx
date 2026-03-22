import React, { useMemo, useState } from 'react';
import { Tree, Tag, Space, Typography, Spin, Empty } from 'antd';
import { EditOutlined, FolderOutlined, FileOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import type { ParameterTreeNode, ParameterType } from '@/types/deviceParameter';
import ParameterEditModal from './ParameterEditModal';

const { Text } = Typography;

interface TreeViewProps {
  deviceId: string;
  treeData: ParameterTreeNode[] | undefined;
  loading: boolean;
  searchKeyword: string;
}

const TYPE_COLOR: Record<string, string> = {
  string: 'blue',
  int: 'green',
  unsignedInt: 'green',
  boolean: 'orange',
  dateTime: 'purple',
  base64: 'cyan',
  hexBinary: 'cyan',
  object: 'default',
};

interface EditTarget {
  parameterPath: string;
  currentValue: string;
  parameterType: ParameterType;
}

function renderLeafTitle(
  node: ParameterTreeNode,
  onEdit: (target: EditTarget) => void,
  highlightKeyword: string
) {
  const nameContent = highlightKeyword
    ? highlightText(node.name, highlightKeyword)
    : node.name;

  return (
    <Space size={8} style={{ lineHeight: '28px' }}>
      <Text strong style={{ fontFamily: 'monospace', fontSize: 13 }}>
        {nameContent}
      </Text>
      <Text type="secondary">=</Text>
      <Text style={{ fontFamily: 'monospace', fontSize: 13, maxWidth: 300 }} ellipsis>
        {node.parameterValue ?? ''}
      </Text>
      {node.parameterType && (
        <Tag color={TYPE_COLOR[node.parameterType] ?? 'default'} style={{ fontSize: 11 }}>
          {node.parameterType}
        </Tag>
      )}
      {node.writable && (
        <Tag color="success" style={{ fontSize: 11 }}>
          可写
        </Tag>
      )}
      {node.writable && (
        <EditOutlined
          style={{ color: '#1677ff', cursor: 'pointer', fontSize: 14 }}
          onClick={(e) => {
            e.stopPropagation();
            onEdit({
              parameterPath: node.fullPath,
              currentValue: node.parameterValue ?? '',
              parameterType: node.parameterType ?? 'string',
            });
          }}
        />
      )}
    </Space>
  );
}

function highlightText(text: string, keyword: string): React.ReactNode {
  if (!keyword) return text;
  const lowerText = text.toLowerCase();
  const lowerKeyword = keyword.toLowerCase();
  const index = lowerText.indexOf(lowerKeyword);
  if (index === -1) return text;

  return (
    <>
      {text.slice(0, index)}
      <span style={{ backgroundColor: '#ffd666', padding: '0 1px' }}>
        {text.slice(index, index + keyword.length)}
      </span>
      {text.slice(index + keyword.length)}
    </>
  );
}

function convertToAntdTree(
  nodes: ParameterTreeNode[],
  onEdit: (target: EditTarget) => void,
  searchKeyword: string
): DataNode[] {
  return nodes.map((node) => {
    const isLeaf = !node.isObject;
    const nameContent = searchKeyword
      ? highlightText(node.name, searchKeyword)
      : node.name;

    const treeNode: DataNode = {
      key: node.fullPath,
      icon: isLeaf ? <FileOutlined /> : <FolderOutlined />,
      title: isLeaf
        ? renderLeafTitle(node, onEdit, searchKeyword)
        : (
            <Text strong style={{ fontSize: 13 }}>
              {nameContent}
            </Text>
          ),
      children: node.children
        ? convertToAntdTree(node.children, onEdit, searchKeyword)
        : undefined,
      isLeaf,
    };
    return treeNode;
  });
}

function filterTree(
  nodes: ParameterTreeNode[],
  keyword: string
): ParameterTreeNode[] {
  if (!keyword) return nodes;
  const lowerKeyword = keyword.toLowerCase();

  return nodes.reduce<ParameterTreeNode[]>((acc, node) => {
    const nameMatches = node.name.toLowerCase().includes(lowerKeyword);
    const pathMatches = node.fullPath.toLowerCase().includes(lowerKeyword);
    const valueMatches = node.parameterValue?.toLowerCase().includes(lowerKeyword) ?? false;

    if (node.children) {
      const filteredChildren = filterTree(node.children, keyword);
      if (filteredChildren.length > 0 || nameMatches || pathMatches) {
        acc.push({
          ...node,
          children: filteredChildren.length > 0 ? filteredChildren : node.children,
        });
      }
    } else if (nameMatches || pathMatches || valueMatches) {
      acc.push(node);
    }
    return acc;
  }, []);
}

function collectAllKeys(nodes: ParameterTreeNode[]): string[] {
  const keys: string[] = [];
  for (const node of nodes) {
    if (node.isObject) {
      keys.push(node.fullPath);
      if (node.children) {
        keys.push(...collectAllKeys(node.children));
      }
    }
  }
  return keys;
}

export default function TreeView({
  deviceId,
  treeData,
  loading,
  searchKeyword,
}: TreeViewProps) {
  const [editTarget, setEditTarget] = useState<EditTarget | null>(null);

  const filteredData = useMemo(() => {
    if (!treeData) return [];
    return filterTree(treeData, searchKeyword);
  }, [treeData, searchKeyword]);

  const expandedKeys = useMemo(() => {
    if (!searchKeyword || !filteredData.length) return undefined;
    return collectAllKeys(filteredData);
  }, [filteredData, searchKeyword]);

  const antdTreeData = useMemo(
    () => convertToAntdTree(filteredData, setEditTarget, searchKeyword),
    [filteredData, searchKeyword]
  );

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 48 }}>
        <Spin tip="加载参数树..." />
      </div>
    );
  }

  if (!treeData || treeData.length === 0) {
    return <Empty description="暂无参数数据，请先执行参数发现" />;
  }

  if (filteredData.length === 0) {
    return <Empty description="未找到匹配的参数" />;
  }

  return (
    <>
      <Tree
        showIcon
        showLine={{ showLeafIcon: false }}
        treeData={antdTreeData}
        defaultExpandedKeys={expandedKeys}
        expandedKeys={searchKeyword ? expandedKeys : undefined}
        autoExpandParent={Boolean(searchKeyword)}
        blockNode
        style={{ padding: '8px 0' }}
      />

      {editTarget && (
        <ParameterEditModal
          open={Boolean(editTarget)}
          deviceId={deviceId}
          parameterPath={editTarget.parameterPath}
          currentValue={editTarget.currentValue}
          parameterType={editTarget.parameterType}
          onClose={() => setEditTarget(null)}
        />
      )}
    </>
  );
}
