import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { App } from 'antd';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import {
  useDeviceGroups,
  useDeviceList,
  useCreateGroup,
  useUpdateGroup,
  useDeleteGroup,
  useMoveDevices,
  useAddDevicesToGroup,
  useDeleteDevices,
  useBatchRebootDevices,
  useUpdateDevice,
} from '@core/hooks/api/useDevices';
import { useT } from '@/hooks/useT';
import type { Device } from '@core/types/device';
import GroupTreePanel from './GroupTreePanel';
import DeviceListPanel from './DeviceListPanel';
import GroupDialogs from './GroupDialogs';
import DeviceDialogs from './DeviceDialogs';
import { useNameFilters } from './useNameFilters';
import { useGroupActions } from './useGroupActions';
import { useDeviceActions } from './useDeviceActions';
import { useBatchActions } from './useBatchActions';
import { useImportExportHandlers } from './useImportExportHandlers';

/**
 * 设备分组主页面。
 *
 * 该文件原为 650 行单体；W2.C.2 / T-0054 后将状态/handlers 拆到 5 个 hook
 * 文件，并把 4 个 dialog + import modal 拆到独立组件。本文件只保留：
 * - 数据查询（groups、devices）
 * - 跨分组的统一 state（selectedGroupId, search、page、selection）
 * - 子组件装配
 */
export default function DeviceGrouping() {
  const t = useT();
  const { modal, message } = App.useApp();
  const { data: groupsData, refetch: refetchGroups } = useDeviceGroups();
  const groups = groupsData?.groups ?? [];
  const totalDevicesFromStats = groupsData?.stats?.totalDevices ?? 0;

  // ── Selection & pagination ──
  const [selectedGroupId, setSelectedGroupId] = useState<string | null>(null);
  const [groupSearchText, setGroupSearchText] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selectedDeviceIds, setSelectedDeviceIds] = useState<React.Key[]>([]);
  // 「开启实时刷新」按钮 → useDeviceList.refetchInterval（5s 一次轮询）。
  const [autoRefresh, setAutoRefresh] = useState(false);

  // ── Mutations (passed into action hooks) ──
  const createGroupMutation = useCreateGroup();
  const updateGroupMutation = useUpdateGroup();
  const deleteGroupMutation = useDeleteGroup();
  const moveDevicesMutation = useMoveDevices();
  const addDevicesToGroupMutation = useAddDevicesToGroup();
  const deleteDevicesMutation = useDeleteDevices();
  // batchRebootMutation 暂未使用，保留 hook 触发以便后续启用而不破坏依赖图
  const _batchRebootMutation = useBatchRebootDevices();
  const updateDeviceMutation = useUpdateDevice();
  void _batchRebootMutation;

  // ── Name filter state (for AddChild and EditLevel2 drawers) ──
  const childNameFilters = useNameFilters({
    onMaxReached: () => void message.warning(t('device.rules.maxConditions', { max: 10 })),
  });
  const editLevel2NameFilters = useNameFilters();

  // ── Data fetching ──
  const queryParams = useMemo(
    () => ({
      page: currentPage,
      pageSize,
      groupId: selectedGroupId ?? undefined,
    } as Parameters<typeof useDeviceList>[0]),
    [currentPage, pageSize, selectedGroupId]
  );
  const { data: deviceData, isLoading, refetch } = useDeviceList(queryParams, {
    refetchInterval: autoRefresh ? 5000 : undefined,
  });
  const devices: Device[] = deviceData?.items ?? [];
  const total = deviceData?.total ?? 0;

  const selectedGroup = useMemo(
    () => groups.find((g) => g.id === selectedGroupId),
    [groups, selectedGroupId]
  );

  // Default select first level-2 node under the first root group
  useEffect(() => {
    if (groups.length > 0 && !selectedGroupId) {
      const firstRoot = groups.find((g) => !g.parentId);
      if (firstRoot) {
        const firstChild = groups.find((g) => g.parentId === firstRoot.id);
        if (firstChild) {
          setSelectedGroupId(firstChild.id);
        }
      }
    }
  }, [groups, selectedGroupId]);

  // ── Filtered groups for tree search ──
  const filteredGroups = useMemo(() => {
    if (!groupSearchText.trim()) return groups;
    const searchLower = groupSearchText.toLowerCase();
    const matchedIds = new Set<string>();
    groups.forEach((g) => {
      if (g.name.toLowerCase().includes(searchLower)) {
        matchedIds.add(g.id);
        if (g.parentId) {
          matchedIds.add(g.parentId);
        }
      }
    });
    return groups.filter((g) => matchedIds.has(g.id));
  }, [groups, groupSearchText]);

  // ── Target group options for move-to-group modal ──
  const getParentName = useCallback(
    (parentId: string | null): string => {
      if (!parentId) return '';
      const parent = groups.find((g) => g.id === parentId);
      return parent?.name ?? '';
    },
    [groups]
  );

  const targetGroupOptions = useMemo(
    () =>
      groups.map((g) => {
        const parentName = getParentName(g.parentId ?? null);
        return {
          label: parentName ? `${parentName} / ${g.name}` : g.name,
          value: g.id,
        };
      }),
    [groups, getParentName]
  );

  // ── Group / Device action hooks ──
  const groupActions = useGroupActions({
    groups,
    refetchGroups,
    modal,
    message,
    t,
    selectedGroupId,
    setSelectedGroupId,
    createGroupMutation: createGroupMutation as unknown as Parameters<typeof useGroupActions>[0]['createGroupMutation'],
    updateGroupMutation: updateGroupMutation as unknown as Parameters<typeof useGroupActions>[0]['updateGroupMutation'],
    deleteGroupMutation: deleteGroupMutation as unknown as Parameters<typeof useGroupActions>[0]['deleteGroupMutation'],
    childNameFilters,
    editLevel2NameFilters,
  });

  const deviceActions = useDeviceActions({
    message,
    t,
    refetch,
    selectedDeviceIds,
    setSelectedDeviceIds,
    updateDeviceMutation,
    moveDevicesMutation,
    addDevicesToGroupMutation,
  });

  // ── Tree context menu handler (cmd:groupId 字符串协议) ──
  const handleContextMenu = useCallback(
    (action: string) => {
      const [cmd, groupId] = action.split(':');
      if (cmd === 'add-child') {
        groupActions.open.addChild(groupId);
      } else if (cmd === 'add-device') {
        deviceActions.open.addToGroup(groupId);
      } else if (cmd === 'edit-level1') {
        groupActions.open.editLevel1(groupId);
      } else if (cmd === 'edit-level2') {
        groupActions.open.editLevel2(groupId);
      } else if (cmd === 'delete-level1') {
        groupActions.open.confirmDelete(groupId, true);
      } else if (cmd === 'delete-level2') {
        groupActions.open.confirmDelete(groupId, false);
      }
    },
    [groupActions.open, deviceActions.open]
  );

  // ── Batch actions ──
  const batchActions = useBatchActions({
    modal,
    message,
    t,
    refetch,
    deleteDevicesMutation,
    setSelectedDeviceIds,
    onMoveToGroup: deviceActions.open.move,
  });

  // ── Import / export handlers ──
  const { handleExport, handleImport, handleDownloadTemplate } = useImportExportHandlers({
    message,
    t,
    refetch,
    selectedGroupId,
    selectedGroupName: selectedGroup?.name,
  });

  // ── Tree panel ──
  const treePanel = (
    <GroupTreePanel
      groups={groups}
      filteredGroups={filteredGroups}
      selectedGroupId={selectedGroupId}
      groupSearchText={groupSearchText}
      total={totalDevicesFromStats}
      onSelect={(key) => {
        setSelectedGroupId(key);
        setCurrentPage(1);
      }}
      onSearchChange={setGroupSearchText}
      onContextMenu={handleContextMenu}
      onAddGroup={groupActions.open.add}
      t={t as (id: string, values?: Record<string, unknown>) => string}
    />
  );

  return (
    <>
      <TreeListPageLayout tree={treePanel} defaultTreeWidth={260}>
        <DeviceListPanel
          devices={devices}
          total={total}
          loading={isLoading}
          selectedDeviceIds={selectedDeviceIds}
          currentPage={currentPage}
          pageSize={pageSize}
          selectedGroupName={selectedGroup?.name}
          batchActions={batchActions}
          onSelectionChange={setSelectedDeviceIds}
          onPageChange={(page, size) => {
            setCurrentPage(page);
            setPageSize(size);
          }}
          onRefresh={() => void refetch()}
          onRealtimeRefreshChange={setAutoRefresh}
          onExport={handleExport}
          onImport={handleImport}
          onDownloadTemplate={handleDownloadTemplate}
          onEditDevice={deviceActions.open.edit}
          t={t as (id: string, values?: Record<string, unknown>) => string}
        />
      </TreeListPageLayout>

      <GroupDialogs
        addModalOpen={groupActions.state.addModalOpen}
        addForm={groupActions.forms.addForm}
        groups={groups}
        onAddModalOk={() => void groupActions.save.addGroup()}
        onAddModalCancel={groupActions.close.add}
        editModalOpen={groupActions.state.editModalOpen}
        editForm={groupActions.forms.editForm}
        editingGroupId={groupActions.state.editingGroupId ?? undefined}
        onEditModalOk={() => void groupActions.save.editGroup()}
        onEditModalCancel={groupActions.close.edit}
        addChildDrawerOpen={groupActions.state.addChildDrawerOpen}
        addChildForm={groupActions.forms.addChildForm}
        addChildParentName={groupActions.state.addChildParentName}
        matchingMode={groupActions.state.matchingMode}
        nameFilters={childNameFilters.filters}
        onAddChildDrawerClose={groupActions.close.addChild}
        onSaveChildGroup={() => void groupActions.save.saveChildGroup()}
        onMatchingModeChange={groupActions.save.onMatchingModeChange}
        onAddFilter={childNameFilters.add}
        onRemoveFilter={childNameFilters.remove}
        onUpdateFilter={childNameFilters.update}
        editLevel2DrawerOpen={groupActions.state.editLevel2DrawerOpen}
        editLevel2Form={groupActions.forms.editLevel2Form}
        editLevel2ParentName={groupActions.state.editLevel2ParentName}
        editLevel2MatchingMode={groupActions.state.editLevel2MatchingMode}
        editLevel2NameFilters={editLevel2NameFilters.filters}
        onEditLevel2DrawerClose={groupActions.close.editLevel2}
        onSaveEditLevel2={() => void groupActions.save.saveEditLevel2()}
        onEditLevel2NameFiltersChange={editLevel2NameFilters.setFilters}
        t={t as (id: string, values?: Record<string, unknown>) => string}
      />

      <DeviceDialogs
        moveToGroupModalOpen={deviceActions.state.moveToGroupModalOpen}
        selectedDeviceIds={selectedDeviceIds}
        targetGroupId={deviceActions.state.targetGroupId}
        targetGroupOptions={targetGroupOptions}
        onTargetGroupChange={deviceActions.setTargetGroupId}
        onMoveToGroupOk={() => void deviceActions.save.moveToGroup()}
        onMoveToGroupCancel={deviceActions.close.move}
        editDeviceModalOpen={deviceActions.state.editDeviceModalOpen}
        editingDevice={deviceActions.state.editingDevice}
        editDeviceForm={deviceActions.forms.editDeviceForm}
        onEditDeviceOk={() => void deviceActions.save.device()}
        onEditDeviceCancel={deviceActions.close.edit}
        addDeviceDrawerOpen={deviceActions.state.addDeviceDrawerOpen}
        addDeviceForm={deviceActions.forms.addDeviceForm}
        addMethod={deviceActions.state.addMethod}
        onAddDeviceDrawerClose={deviceActions.close.addToGroup}
        onSaveDevices={() => void deviceActions.save.devices()}
        onDownloadTemplate={handleDownloadTemplate}
        t={t as (id: string, values?: Record<string, unknown>) => string}
      />
    </>
  );
}
