import React from 'react';
import type { FormInstance } from 'antd';
import type { NameFilterItem, GroupItem } from './types';
import type { AddGroupFormValues } from './useGroupActions';
import { getRecordI18n } from '@core/utils/i18nText';
import { useAppStore } from '@core/store/appStore';
import { UNASSIGNED_GROUP_ID } from '@core/utils/deviceGroupTargets';
import AddGroupDrawer from './AddGroupDrawer';
import EditGroupModal from './EditGroupModal';
import AddChildGroupDrawer from './AddChildGroupDrawer';
import EditLevel2GroupDrawer from './EditLevel2GroupDrawer';

const DEFAULT_ROOT_GROUP_ID = '00000000-0000-0000-0000-000000000001';

export interface GroupDialogsProps {
  // Add Level-1 Group Drawer
  addModalOpen: boolean;
  addForm: FormInstance<AddGroupFormValues>;
  groups: GroupItem[];
  onAddModalOk: () => void;
  onAddModalCancel: () => void;

  // Edit Level-1 Group Modal
  editModalOpen: boolean;
  editForm: FormInstance<AddGroupFormValues>;
  editingGroupId?: string;
  onEditModalOk: () => void;
  onEditModalCancel: () => void;

  // Add Child Group (Level-2) Drawer
  addChildDrawerOpen: boolean;
  addChildForm: FormInstance<{ name_i18n?: Record<string, string>; autoAssignEnabled?: boolean; matchingMode: 'deviceName' | 'lac' | 'tac' | 'serialNumber'; tacRag: string; sourceGroupId?: string; serialNumbers?: string }>;
  addChildParentName?: string;
  autoAssignEnabled?: boolean;
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
  editLevel2GroupId?: string;
  editLevel2Form: FormInstance<{ name_i18n?: Record<string, string>; autoAssignEnabled?: boolean; matchingMode: 'deviceName' | 'lac' | 'tac' | 'serialNumber'; tacRag: string; sourceGroupId?: string; serialNumbers?: string }>;
  editLevel2ParentName?: string;
  editLevel2AutoAssignEnabled?: boolean;
  editLevel2MatchingMode: string | undefined;
  editLevel2NameFilters: NameFilterItem[];
  onEditLevel2DrawerClose: () => void;
  onSaveEditLevel2: () => void;
  onEditLevel2NameFiltersChange: React.Dispatch<React.SetStateAction<NameFilterItem[]>>;

  t: (id: string, values?: Record<string, string | number>) => string;
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
  addChildParentName,
  autoAssignEnabled,
  matchingMode,
  nameFilters,
  onAddChildDrawerClose,
  onSaveChildGroup,
  onMatchingModeChange,
  onAddFilter,
  onRemoveFilter,
  onUpdateFilter,
  editLevel2DrawerOpen,
  editLevel2GroupId,
  editLevel2Form,
  editLevel2ParentName,
  editLevel2AutoAssignEnabled,
  editLevel2MatchingMode,
  editLevel2NameFilters,
  onEditLevel2DrawerClose,
  onSaveEditLevel2,
  onEditLevel2NameFiltersChange,
  t,
}: GroupDialogsProps) {
  const locale = useAppStore((s) => s.locale);
  const getGroupName = (group: GroupItem): string => getBuiltInGroupName(group, t) ||
    getRecordI18n(group as unknown as Record<string, unknown>, 'name', locale) ||
    group.name;
  const getSourceGroupOptionLabel = (group: GroupItem): string => {
    const parent = groups.find((item) => item.id === group.parentId);
    const parentName = parent ? getGroupName(parent) : '';
    const groupName = getGroupName(group);
    return parentName ? `${parentName} / ${groupName}` : groupName;
  };

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
        parentGroupName={addChildParentName}
        autoAssignEnabled={Boolean(autoAssignEnabled)}
        matchingMode={matchingMode}
        nameFilters={nameFilters}
        sourceGroupOptions={groups
          .filter((group) => group.parentId != null)
          .map((group) => ({
            label: getSourceGroupOptionLabel(group),
            value: group.id,
          }))}
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
        parentGroupName={editLevel2ParentName}
        autoAssignEnabled={Boolean(editLevel2AutoAssignEnabled)}
        matchingMode={editLevel2MatchingMode}
        nameFilters={editLevel2NameFilters}
        sourceGroupOptions={groups
          .filter((group) => group.parentId != null && group.id !== editLevel2GroupId)
          .map((group) => ({
            label: getSourceGroupOptionLabel(group),
            value: group.id,
          }))}
        onClose={onEditLevel2DrawerClose}
        onSave={onSaveEditLevel2}
        onNameFiltersChange={onEditLevel2NameFiltersChange}
        t={t}
      />
    </>
  );
}

function getBuiltInGroupName(group: GroupItem, t: (id: string) => string): string {
  const isDefaultGroup = group.id === DEFAULT_ROOT_GROUP_ID || group.id === UNASSIGNED_GROUP_ID;
  if (!isDefaultGroup) return '';
  return t('device.defaultGroupName');
}
