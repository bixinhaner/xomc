import React from 'react';
import type { FormInstance } from 'antd';
import type { NameFilterItem, GroupItem } from './types';
import AddGroupDrawer from './AddGroupDrawer';
import EditGroupModal from './EditGroupModal';
import AddChildGroupDrawer from './AddChildGroupDrawer';
import EditLevel2GroupDrawer from './EditLevel2GroupDrawer';

export interface GroupDialogsProps {
  // Add Level-1 Group Drawer
  addModalOpen: boolean;
  addForm: FormInstance<{ name: string; parentId?: string; description: string }>;
  groups: GroupItem[];
  onAddModalOk: () => void;
  onAddModalCancel: () => void;

  // Edit Level-1 Group Modal
  editModalOpen: boolean;
  editForm: FormInstance<{ name: string; parentId?: string; description: string }>;
  editingGroupId?: string;
  onEditModalOk: () => void;
  onEditModalCancel: () => void;

  // Add Child Group (Level-2) Drawer
  addChildDrawerOpen: boolean;
  addChildForm: FormInstance<{ name: string; matchingMode: 'deviceName' | 'lac' | 'tac'; tacRag: string }>;
  matchingMode: string | undefined;
  nameFilters: NameFilterItem[];
  onAddChildDrawerClose: () => void;
  onSaveChildGroup: () => void;
  onMatchingModeChange: () => void;
  onAddFilter: () => void;
  onRemoveFilter: (id: string) => void;
  onUpdateFilter: (id: string, field: keyof NameFilterItem, value: string) => void;

  // Edit Level-2 Group Drawer
  editLevel2DrawerOpen: boolean;
  editLevel2Form: FormInstance<{ name: string; matchingMode: 'deviceName' | 'lac' | 'tac'; tacRag: string }>;
  editLevel2MatchingMode: string | undefined;
  editLevel2NameFilters: NameFilterItem[];
  onEditLevel2DrawerClose: () => void;
  onSaveEditLevel2: () => void;
  onEditLevel2NameFiltersChange: React.Dispatch<React.SetStateAction<NameFilterItem[]>>;

  t: (id: string, values?: Record<string, unknown>) => string;
}

/**
 * 分组相关对话框组合容器：4 个 dialog 拆分到独立文件，本组件仅负责装配。
 * 拆分自旧版 GroupDialogs.tsx（W2.C.2 / T-0054，单文件 ≤ 400 行）。
 */
export default function GroupDialogs({
  addModalOpen,
  addForm,
  groups,
  onAddModalOk,
  onAddModalCancel,
  editModalOpen,
  editForm,
  editingGroupId,
  onEditModalOk,
  onEditModalCancel,
  addChildDrawerOpen,
  addChildForm,
  matchingMode,
  nameFilters,
  onAddChildDrawerClose,
  onSaveChildGroup,
  onMatchingModeChange,
  onAddFilter,
  onRemoveFilter,
  onUpdateFilter,
  editLevel2DrawerOpen,
  editLevel2Form,
  editLevel2MatchingMode,
  editLevel2NameFilters,
  onEditLevel2DrawerClose,
  onSaveEditLevel2,
  onEditLevel2NameFiltersChange,
  t,
}: GroupDialogsProps) {
  return (
    <>
      <AddGroupDrawer
        open={addModalOpen}
        form={addForm}
        groups={groups}
        onOk={onAddModalOk}
        onCancel={onAddModalCancel}
        t={t}
      />

      <EditGroupModal
        open={editModalOpen}
        form={editForm}
        groups={groups}
        editingGroupId={editingGroupId}
        onOk={onEditModalOk}
        onCancel={onEditModalCancel}
        t={t}
      />

      <AddChildGroupDrawer
        open={addChildDrawerOpen}
        form={addChildForm}
        matchingMode={matchingMode}
        nameFilters={nameFilters}
        onClose={onAddChildDrawerClose}
        onSave={onSaveChildGroup}
        onMatchingModeChange={onMatchingModeChange}
        onAddFilter={onAddFilter}
        onRemoveFilter={onRemoveFilter}
        onUpdateFilter={onUpdateFilter}
        t={t}
      />

      <EditLevel2GroupDrawer
        open={editLevel2DrawerOpen}
        form={editLevel2Form}
        matchingMode={editLevel2MatchingMode}
        nameFilters={editLevel2NameFilters}
        onClose={onEditLevel2DrawerClose}
        onSave={onSaveEditLevel2}
        onNameFiltersChange={onEditLevel2NameFiltersChange}
        t={t}
      />
    </>
  );
}
