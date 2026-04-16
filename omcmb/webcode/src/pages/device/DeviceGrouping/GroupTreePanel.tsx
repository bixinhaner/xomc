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

const { Text } = Typography;

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
  const rootGroups = groups.filter((g) => !g.parentId);

  function buildNode(group: GroupItem, isRootLevel: boolean): DataNode {
    const children = groups.filter((g) => g.parentId === group.id);
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

    const isL1 = !group.parentId;

    return {
      key: group.id,
      selectable: true,
      title: (
        <div className="group-tree-node">
          <Tooltip title={group.name} mouseEnterDelay={0.8} placement="right">
            <span className="group-tree-node-name">
              {isL1 ? (
                <FolderOutlined style={{ marginRight: 6, fontSize: 12, color: '#FA8C16' }} />
              ) : (
                <span className="group-tree-dot" />
              )}
              <span className="group-tree-node-text">{group.name}</span>
            </span>
          </Tooltip>
          <span className="group-tree-node-count">{group.deviceCount}</span>
          <span className="group-tree-node-actions">
            <Dropdown
              menu={{ items: menuItems }}
              trigger={['click']}
            >
              <Button
                type="text"
                size="small"
                icon={<MoreOutlined />}
                onClick={(e) => e.stopPropagation()}
                className="group-tree-more-btn"
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
    <div className="group-tree-panel">
      {/* 头部：标题 + 搜索/添加图标 */}
      <div className="group-tree-header">
        <span className="group-tree-header-title">{t('nav.device.group')}</span>
        <span className="group-tree-header-actions">
          <Button
            type="text"
            size="small"
            icon={<SearchOutlined />}
            onClick={() => { setSearchVisible(!searchVisible); if (searchVisible) onSearchChange(''); }}
            title={t('device.searchGroup')}
            className="group-tree-icon-btn"
          />
          <Button
            type="text"
            size="small"
            icon={<PlusOutlined />}
            onClick={onAddGroup}
            title={t('common.add')}
            className="group-tree-icon-btn"
          />
        </span>
      </div>

      {/* 搜索框：展开式 */}
      {searchVisible && (
        <div className="group-tree-search">
          <Input
            placeholder={t('device.searchGroup')}
            prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
            value={groupSearchText}
            onChange={(e) => onSearchChange(e.target.value)}
            allowClear
            size="small"
            autoFocus
            onBlur={() => { if (!groupSearchText) setSearchVisible(false); }}
          />
        </div>
      )}

      {/* 树区域 */}
      <div className="group-tree-body">
        <Tree
          className="group-tree"
          treeData={[
            {
              key: '__all__',
              title: (
                <div className="group-tree-root-node">
                  <span className="group-tree-node-name">
                    <AppstoreOutlined style={{ marginRight: 6, fontSize: 12, color: 'var(--color-primary-600)' }} />
                    <span className="group-tree-node-text" style={{ fontWeight: 500 }}>{t('common.all')}</span>
                  </span>
                  <span className="group-tree-node-count">{total}</span>
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

      {/* 样式 */}
      <style>{`
        /* Panel layout */
        .group-tree-panel {
          display: flex;
          flex-direction: column;
          height: 100%;
        }

        /* Header */
        .group-tree-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 0 14px;
          height: 40px;
          flex-shrink: 0;
        }
        .group-tree-header-title {
          font-size: 14px;
          font-weight: 600;
          color: var(--color-text, #262626);
        }
        .group-tree-header-actions {
          display: flex;
          gap: 4px;
        }
        .group-tree-icon-btn {
          color: var(--color-text-tertiary, #8c8c8c) !important;
          border: none !important;
          box-shadow: none !important;
        }
        .group-tree-icon-btn:hover {
          color: var(--color-primary-600, #1677ff) !important;
          background: var(--color-primary-1, #e6f4ff) !important;
        }

        /* Search */
        .group-tree-search {
          padding: 0 14px 8px;
          flex-shrink: 0;
        }

        /* Tree body */
        .group-tree-body {
          flex: 1;
          overflow: auto;
          padding: 4px 6px 12px 10px;
        }

        /* Root "全部设备" node */
        .group-tree-root-node {
          display: flex;
          align-items: center;
          justify-content: space-between;
          width: 100%;
          padding: 4px 0;
        }

        /* Tree node row */
        .group-tree-node {
          display: flex;
          align-items: center;
          justify-content: space-between;
          width: 100%;
          padding: 2px 0;
          position: relative;
        }
        .group-tree-node-name {
          display: flex;
          align-items: center;
          flex: 1;
          min-width: 0;
          overflow: hidden;
        }
        .group-tree-node-text {
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }
        .group-tree-node-count {
          font-size: 11px;
          color: var(--color-text-quaternary, #bfbfbf);
          margin-left: 8px;
          flex-shrink: 0;
          min-width: 20px;
          text-align: right;
        }
        .group-tree-node-actions {
          flex-shrink: 0;
          margin-left: 2px;
          transition: opacity 150ms ease;
        }
        .group-tree-node-actions .group-tree-more-btn {
          color: var(--color-text-quaternary, #d9d9d9) !important;
          border: none !important;
          box-shadow: none !important;
          font-size: 12px !important;
          width: 20px !important;
          height: 20px !important;
          min-width: 20px !important;
          padding: 0 !important;
        }

        /* Hover / selected: deepen more button */
        .group-tree .ant-tree-treenode:hover .group-tree-more-btn,
        .group-tree .ant-tree-treenode-selected .group-tree-more-btn {
          color: var(--color-text-tertiary, #8c8c8c) !important;
        }

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

        /* Selection state: left bar + text color instead of full background */
        .group-tree .ant-tree-node-content-wrapper.ant-tree-node-selected {
          background: rgba(var(--color-primary-rgb, 22,119,255), 0.04) !important;
          border-radius: 4px;
        }
        .group-tree .ant-tree-node-content-wrapper.ant-tree-node-selected .group-tree-node-text {
          color: var(--color-primary-600, #1677ff);
          font-weight: 500;
        }
        .group-tree .ant-tree-node-content-wrapper.ant-tree-node-selected .group-tree-dot {
          background: var(--color-primary-600, #1677ff);
        }
        .group-tree .ant-tree-node-content-wrapper.ant-tree-node-selected .group-tree-node-count {
          color: var(--color-primary-400, #69b1ff);
        }

        /* Hover state */
        .group-tree .ant-tree-node-content-wrapper:hover:not(.ant-tree-node-selected) {
          background: var(--color-fill-quaternary, rgba(0,0,0,0.02)) !important;
          border-radius: 4px;
        }

        /* Tree item spacing */
        .group-tree .ant-tree-treenode {
          padding: 1px 0 !important;
        }
        .group-tree .ant-tree-node-content-wrapper {
          border-radius: 4px;
          transition: background 150ms ease;
        }

        /* Root node divider */
        .group-tree > .ant-tree-treenode:first-child {
          border-bottom: 1px solid var(--color-border-secondary, #f0f0f0);
          margin-bottom: 4px;
          padding-bottom: 4px !important;
        }

        /* L1 vertical guide line */
        .group-tree .ant-tree-treenode-selected::before,
        .group-tree .ant-tree-treenode:hover::before {
          content: none;
        }

        /* Reduce indent */
        .group-tree .ant-tree-indent-unit {
          width: 16px !important;
        }

        /* Scrollbar */
        .group-tree-body::-webkit-scrollbar {
          width: 4px;
        }
        .group-tree-body::-webkit-scrollbar-track {
          background: transparent;
        }
        .group-tree-body::-webkit-scrollbar-thumb {
          background: var(--color-border-secondary, #f0f0f0);
          border-radius: 2px;
        }
      `}</style>
    </div>
  );
}
