import React, { useMemo, useState } from 'react';
import { Tree, Tag, Space, Typography, Spin, Empty, Tooltip, Popconfirm, message } from 'antd';
import {
  EditOutlined,
  FolderOutlined,
  FileOutlined,
  PlusCircleOutlined,
  DeleteOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import type { ParameterTreeNode, ParameterType } from '@/types/deviceParameter';
import { useAddObject, useDeleteObject } from '@/hooks/api/useDeviceParameters';
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
  constraints?: ParameterTreeNode['constraints'];
  description?: string;
  changeApplies?: string;
  defaultValue?: string;
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
      {node.changeApplies === 'RebootRequired' && (
        <Tag color="warning" style={{ fontSize: 11 }}>
          需重启
        </Tag>
      )}
      {node.description && (
        <Tooltip title={node.description}>
          <InfoCircleOutlined style={{ color: '#8c8c8c', fontSize: 12 }} />
        </Tooltip>
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
              constraints: node.constraints,
              description: node.description,
              changeApplies: node.changeApplies,
              defaultValue: node.defaultValue,
            });
          }}
        />
      )}
    </Space>
  );
}

function renderObjectTitle(
  node: ParameterTreeNode,
  highlightKeyword: string,
  onAdd: (objectPath: string) => void,
  onDelete: (objectPath: string) => void,
) {
  const nameContent = highlightKeyword
    ? highlightText(node.name, highlightKeyword)
    : node.name;

  return (
    <Space size={8} style={{ lineHeight: '28px' }}>
      <Text strong style={{ fontSize: 13 }}>
        {nameContent}
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
      {node.canAdd && (
        <Tooltip title={`添加实例（当前 ${node.instanceCount ?? 0}，最大 ${node.maxInstances ?? '无限制'}）`}>
          <PlusCircleOutlined
            style={{ color: '#52c41a', cursor: 'pointer', fontSize: 14 }}
            onClick={(e) => {
              e.stopPropagation();
              onAdd(node.fullPath.endsWith('.') ? node.fullPath : node.fullPath + '.');
            }}
          />
        </Tooltip>
      )}
      {node.canDelete && (
        <Popconfirm
          title="确认删除此实例？"
          description="删除操作将异步下发到设备。"
          onConfirm={(e) => {
            e?.stopPropagation();
            onDelete(node.fullPath.endsWith('.') ? node.fullPath : node.fullPath + '.');
          }}
          onCancel={(e) => e?.stopPropagation()}
          okText="删除"
          cancelText="取消"
        >
          <DeleteOutlined
            style={{ color: '#ff4d4f', cursor: 'pointer', fontSize: 14 }}
            onClick={(e) => e.stopPropagation()}
          />
        </Popconfirm>
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
  onAdd: (objectPath: string) => void,
  onDelete: (objectPath: string) => void,
  searchKeyword: string
): DataNode[] {
  return nodes.map((node) => {
    const isLeaf = !node.isObject;

    const treeNode: DataNode = {
      key: node.fullPath,
      icon: isLeaf ? <FileOutlined /> : <FolderOutlined />,
      title: isLeaf
        ? renderLeafTitle(node, onEdit, searchKeyword)
        : renderObjectTitle(node, searchKeyword, onAdd, onDelete),
      children: node.children
        ? convertToAntdTree(node.children, onEdit, onAdd, onDelete, searchKeyword)
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
  const addObjectMutation = useAddObject();
  const deleteObjectMutation = useDeleteObject();

  const handleAddObject = (objectPath: string) => {
    addObjectMutation.mutate(
      { deviceId, objectPath },
      {
        onSuccess: () => {
          message.success('添加实例命令已下发');
        },
        onError: () => {
          message.error('添加实例失败');
        },
      }
    );
  };

  const handleDeleteObject = (objectPath: string) => {
    deleteObjectMutation.mutate(
      { deviceId, objectPath },
      {
        onSuccess: () => {
          message.success('删除实例命令已下发');
        },
        onError: () => {
          message.error('删除实例失败');
        },
      }
    );
  };

  const filteredData = useMemo(() => {
    if (!treeData) return [];
    return filterTree(treeData, searchKeyword);
  }, [treeData, searchKeyword]);

  const expandedKeys = useMemo(() => {
    if (!searchKeyword || !filteredData.length) return undefined;
    return collectAllKeys(filteredData);
  }, [filteredData, searchKeyword]);

  const antdTreeData = useMemo(
    () => convertToAntdTree(filteredData, setEditTarget, handleAddObject, handleDeleteObject, searchKeyword),
    // eslint-disable-next-line react-hooks/exhaustive-deps
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
          constraints={editTarget.constraints}
          description={editTarget.description}
          changeApplies={editTarget.changeApplies}
          defaultValue={editTarget.defaultValue}
          onClose={() => setEditTarget(null)}
        />
      )}
    </>
  );
}
