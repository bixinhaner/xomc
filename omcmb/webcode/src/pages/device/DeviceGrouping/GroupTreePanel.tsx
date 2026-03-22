import React, { useMemo } from 'react';
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
  const rootGroups = groups.filter((g) => g.parentId === null);

  function buildNode(group: GroupItem, isRootLevel: boolean): DataNode {
    const children = groups.filter((g) => g.parentId === group.id);
    const isLevel1 = group.parentId === null;
    const isDefaultGroup = group.id === 'grp-default';

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
      selectable: !isLevel1,
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
        <Tree
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
          defaultExpandAll
          selectedKeys={selectedGroupId ? [selectedGroupId] : []}
          onSelect={(keys) => {
            const key = keys[0] as string | undefined;
            if (!key) return;
            const clickedGroup = groups.find((g) => g.id === key);
            if (clickedGroup && clickedGroup.parentId !== null) {
              onSelect(key);
            }
          }}
          blockNode
          style={{ fontSize: 13 }}
        />
      </div>
    </div>
  );
}
