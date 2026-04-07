import React, { useMemo, useState } from 'react';
import { Button, Dropdown, Input, Tree, Typography } from 'antd';
import {
  DeleteOutlined,
  EditOutlined,
  FolderAddOutlined,
  FolderOutlined,
  MoreOutlined,
  PlusOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import type { DataNode } from 'antd/es/tree';
import type { GroupItem } from './types';

const { Title, Text } = Typography;

export interface GroupTreePanelProps {
  groups: GroupItem[];
  filteredGroups: GroupItem[];
  selectedGroupId: string | null;
  groupSearchText: string;
  total: number;
  onSelect: (groupId: string) => void;
  onSearchChange: (text: string) => void;
  onContextMenu: (action: string) => void;
  onAddGroup: () => void;
  t: (id: string, values?: Record<string, unknown>) => string;
}

function buildTreeData(
  groups: GroupItem[],
  selectedId: string | null,
  onContextMenu: (groupId: string) => void,
  t: (id: string, values?: Record<string, unknown>) => string
): DataNode[] {
  // 兼容 null 和 undefined（后端 omitempty 导致根分组没有 parent_id 字段）
  const rootGroups = groups.filter((g) => !g.parentId);

  function buildNode(group: GroupItem, isRootLevel: boolean): DataNode {
    const children = groups.filter((g) => g.parentId === group.id);
    // 兼容 null 和 undefined
    const isLevel1 = !group.parentId;
    const isDefaultGroup = group.builtIn === 1;

    let menuItems: MenuProps['items'];
    if (isRootLevel && isDefaultGroup) {
      menuItems = [
        {
          key: 'add-child',
          label: t('common.add'),
          icon: <FolderAddOutlined />,
          onClick: (info) => {
            info.domEvent.stopPropagation();
            onContextMenu(`add-child:${group.id}`);
          },
        },
      ];
    } else if (isRootLevel) {
      menuItems = [
        {
          key: 'add-child',
          label: t('common.add'),
          icon: <FolderAddOutlined />,
          onClick: (info) => {
            info.domEvent.stopPropagation();
            onContextMenu(`add-child:${group.id}`);
          },
        },
        {
          key: 'edit-level1',
          label: t('common.edit'),
          icon: <EditOutlined />,
          onClick: (info) => {
            info.domEvent.stopPropagation();
            onContextMenu(`edit-level1:${group.id}`);
          },
        },
        { type: 'divider' },
        {
          key: 'delete-level1',
          label: t('common.delete'),
          icon: <DeleteOutlined />,
          danger: true,
          onClick: (info) => {
            info.domEvent.stopPropagation();
            onContextMenu(`delete-level1:${group.id}`);
          },
        },
      ];
    } else {
      menuItems = [
        {
          key: 'add-device',
          label: t('common.add'),
          icon: <FolderAddOutlined />,
          onClick: (info) => {
            info.domEvent.stopPropagation();
            onContextMenu(`add-device:${group.id}`);
          },
        },
        {
          key: 'edit-level2',
          label: t('common.edit'),
          icon: <EditOutlined />,
          onClick: (info) => {
            info.domEvent.stopPropagation();
            onContextMenu(`edit-level2:${group.id}`);
          },
        },
        { type: 'divider' },
        {
          key: 'delete-level2',
          label: t('common.delete'),
          icon: <DeleteOutlined />,
          danger: true,
          onClick: (info) => {
            info.domEvent.stopPropagation();
            onContextMenu(`delete-level2:${group.id}`);
          },
        },
      ];
    }

    return {
      key: group.id,
      // 所有分组都可以点击，即使没有设备或子分组
      selectable: true,
      title: (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            width: '100%',
            padding: '2px 0',
          }}
        >
          <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            <FolderOutlined style={{ marginRight: 6, color: '#FA8C16' }} />
            {group.name}
            <Text type="secondary" style={{ fontSize: 11, marginLeft: 6 }}>
              ({group.deviceCount})
            </Text>
          </span>
          <Dropdown
            menu={{ items: menuItems }}
            trigger={['click']}
          >
            <Button
              type="text"
              size="small"
              icon={<MoreOutlined />}
              onClick={(e) => e.stopPropagation()}
              style={{ flexShrink: 0 }}
            />
          </Dropdown>
        </div>
      ),
      icon: null,
      children: children.length > 0 ? children.map((child) => buildNode(child, false)) : undefined,
    };
  }

  return rootGroups.map((group) => buildNode(group, true));
}

export default function GroupTreePanel({
  groups,
  filteredGroups,
  selectedGroupId,
  groupSearchText,
  total,
  onSelect,
  onSearchChange,
  onContextMenu,
  onAddGroup,
  t,
}: GroupTreePanelProps) {
  // 默认展开 __all__ 和所有 L1 root groups
  // 兼容 null 和 undefined（后端 omitempty 导致根分组没有 parent_id 字段）
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>(() => ['__all__', ...groups.filter((g) => !g.parentId).map((g) => g.id)]);

  const filteredTreeData = useMemo(
    () => buildTreeData(filteredGroups, selectedGroupId, onContextMenu, t),
    [filteredGroups, selectedGroupId, onContextMenu, t]
  );

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div
        style={{
          padding: '12px 12px 8px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderBottom: '1px solid #f0f0f0',
        }}
      >
        <Title level={5} style={{ margin: 0, fontSize: 14 }}>
          {t('nav.device.group')}
        </Title>
        <Button
          type="text"
          size="small"
          icon={<PlusOutlined />}
          onClick={onAddGroup}
        >
          {t('common.add')}
        </Button>
      </div>
      <div style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
        <Input
          placeholder={t('device.searchGroup')}
          prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
          value={groupSearchText}
          onChange={(e) => onSearchChange(e.target.value)}
          allowClear
          size="small"
        />
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: '8px 4px' }}>
        <style>{`
          .group-tree .ant-tree-node-content-wrapper.ant-tree-node-selected {
            background-color: var(--color-primary-100, #e6f4ff) !important;
            font-weight: 500;
          }
          .group-tree .ant-tree-node-content-wrapper.ant-tree-node-selected .ant-tree-title {
            color: var(--color-primary-700, #1d4ed8);
          }
          .group-tree .ant-tree-treenode-selected > .ant-tree-node-content-wrapper {
            background-color: var(--color-primary-100, #e6f4ff) !important;
          }
        `}</style>
        <Tree
          className="group-tree"
          treeData={[
            {
              key: '__all__',
              title: (
                <span>
                  <FolderOutlined style={{ marginRight: 6, color: 'var(--color-primary-600)' }} />
                  {t('common.all')}
                  <Text type="secondary" style={{ fontSize: 11, marginLeft: 6 }}>
                    ({total})
                  </Text>
                </span>
              ),
              children: filteredTreeData,
              selectable: false,
            },
          ]}
          expandedKeys={expandedKeys}
          onExpand={(keys) => setExpandedKeys(keys)}
          selectedKeys={selectedGroupId ? [selectedGroupId] : []}
          onSelect={(keys) => {
            const key = keys[0] as string | undefined;
            if (!key) return;
            const clickedGroup = groups.find((g) => g.id === key);
            // L2 分组（有 parentId）或没有子分组的 L1 分组可以被选择
            if (clickedGroup) {
              onSelect(key);
            }
          }}
          onDoubleClick={(_, node) => {
            // 双击 L1 分组（根分组）时，切换展开/折叠状态
            const nodeKey = node.key as string;
            if (nodeKey && nodeKey !== '__all__') {
              const clickedGroup = groups.find((g) => g.id === nodeKey);
              // 只有 L1 分组（没有 parentId）才处理双击展开
              if (clickedGroup && !clickedGroup.parentId) {
                setExpandedKeys((prev) => {
                  if (prev.includes(nodeKey)) {
                    // 如果已展开，则折叠
                    return prev.filter((k) => k !== nodeKey);
                  } else {
                    // 如果未展开，则展开
                    return [...prev, nodeKey];
                  }
                });
              }
            }
          }}
          blockNode
          style={{ fontSize: 13 }}
        />
      </div>
    </div>
  );
}
