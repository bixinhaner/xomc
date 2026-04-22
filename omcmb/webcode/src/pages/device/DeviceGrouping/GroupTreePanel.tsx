import React, { useMemo, useState } from 'react';
import { Button, Dropdown, Input, Tooltip, Tree, Typography } from 'antd';
import {
  AppstoreOutlined,
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
import styles from './DeviceGrouping.module.css';

const { _Text } = Typography;

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
  _selectedId: string | null,
  onContextMenu: (groupId: string) => void,
  t: (id: string, values?: Record<string, unknown>) => string
): DataNode[] {
  const rootGroups = groups.filter((g) => !g.parentId);

  function buildNode(group: GroupItem, isRootLevel: boolean): DataNode {
    const children = groups.filter((g) => g.parentId === group.id);
    const _isLevel1 = !group.parentId;
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

    const isL1 = !group.parentId;

    return {
      key: group.id,
      selectable: true,
      title: (
        <div className={`${styles.groupNode} ${isL1 ? styles.groupNodeLevel1 : styles.groupNodeLevel2}`}>
          <Tooltip title={group.name} mouseEnterDelay={0.8} placement="right">
            <span className={styles.treeNodeContent}>
              {isL1 ? (
                <FolderOutlined style={{ marginRight: 6, fontSize: 12, color: '#FA8C16' }} />
              ) : (
                <span className="group-tree-dot" />
              )}
              <span className={styles.treeNodeText}>{group.name}</span>
            </span>
          </Tooltip>
          <span className={styles.groupCountBadge}>{group.deviceCount}</span>
          <span className={styles.treeNodeActions}>
            <Dropdown
              menu={{ items: menuItems }}
              trigger={['click']}
            >
              <Button
                type="text"
                size="small"
                icon={<MoreOutlined />}
                onClick={(e) => e.stopPropagation()}
                className={styles.treeNodeActionButton}
              />
            </Dropdown>
          </span>
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
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>(() => ['__all__', ...groups.filter((g) => !g.parentId).map((g) => g.id)]);
  const [searchVisible, setSearchVisible] = useState(false);

  const filteredTreeData = useMemo(
    () => buildTreeData(filteredGroups, selectedGroupId, onContextMenu, t),
    [filteredGroups, selectedGroupId, onContextMenu, t]
  );

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* 头部：标题 + 搜索/添加图标 */}
      <div className={styles.treePanelHeader}>
        <span className={styles.treePanelTitle}>{t('nav.device.group')}</span>
        <span style={{ display: 'flex', gap: 4 }}>
          <Button
            type="text"
            size="small"
            icon={<SearchOutlined />}
            onClick={() => { setSearchVisible(!searchVisible); if (searchVisible) onSearchChange(''); }}
            title={t('device.searchGroup')}
          />
          <Button
            type="text"
            size="small"
            icon={<PlusOutlined />}
            onClick={onAddGroup}
            title={t('common.add')}
          />
        </span>
      </div>

      {/* 搜索框：展开式 */}
      {searchVisible && (
        <div className={styles.treeSearchContainer}>
          <Input
            placeholder={t('device.searchGroup')}
            prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
            value={groupSearchText}
            onChange={(e) => onSearchChange(e.target.value)}
            allowClear
            size="small"
            autoFocus
            onBlur={() => { if (!groupSearchText) setSearchVisible(false); }}
            className={styles.treeSearchInput}
          />
        </div>
      )}

      {/* 树区域 */}
      <div className={styles.treeNodesContainer}>
        <Tree
          className="group-tree"
          treeData={[
            {
              key: '__all__',
              title: (
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%', padding: '4px 0' }}>
                  <span className={styles.treeNodeContent}>
                    <AppstoreOutlined style={{ marginRight: 6, fontSize: 12, color: 'var(--color-primary-600)' }} />
                    <span className={styles.treeNodeText} style={{ fontWeight: 500 }}>{t('common.all')}</span>
                  </span>
                  <span className={styles.groupCountBadge}>{total}</span>
                </div>
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
            if (!clickedGroup) return;

            if (!clickedGroup.parentId) {
              setExpandedKeys((prev) => {
                if (prev.includes(key)) {
                  return prev.filter((k) => k !== key);
                } else {
                  return [...prev, key];
                }
              });
              return;
            }

            onSelect(key);
          }}
          blockNode
        />
      </div>

      {/* 保留部分原有的样式，用于特定细节 */}
      <style>{`
        /* L2 dot indicator */
        .group-tree-dot {
          display: inline-block;
          width: 4px;
          height: 4px;
          border-radius: 50%;
          background: var(--color-text-quaternary, #d9d9d9);
          margin-right: 8px;
          margin-left: 2px;
          flex-shrink: 0;
        }

        /* Root node divider */
        .group-tree > .ant-tree-treenode:first-child {
          border-bottom: 1px solid var(--color-border-secondary, #f0f0f0);
          margin-bottom: 4px;
          padding-bottom: 4px !important;
        }

        /* Reduce indent */
        .group-tree .ant-tree-indent-unit {
          width: 16px !important;
        }
      `}</style>
    </div>
  );
}
