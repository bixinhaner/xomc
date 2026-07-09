import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { App, Modal, Radio, Space, Typography } from 'antd';
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
import { useAppStore } from '@core/store/appStore';
import { expandSelectedGroupIds } from '@core/utils/deviceGroupFilter';
import { withDeviceGroupDisplayName } from '@core/utils/deviceGroupDisplay';
import { useT } from '@/hooks/useT';
import { useI18nText } from '@/hooks/useI18nText';
import type { Device } from '@core/types/device';
import GroupTreePanel from './GroupTreePanel';
import DeviceListPanel from './DeviceListPanel';
import GroupDialogs from './GroupDialogs';
import DeviceDialogs from './DeviceDialogs';
import { useNameFilters } from './useNameFilters';
import { useGroupActions } from './useGroupActions';
import { useDeviceActions } from './useDeviceActions';
import { useBatchActions } from './useBatchActions';
import { useImportExportHandlers, type DeviceGroupExportMode } from './useImportExportHandlers';
import { buildGroupTargetOptions } from '@core/utils/deviceGroupTargets';
import { buildDeviceGroupSubtreeCountMap } from '@core/utils/deviceGroupCounts';

type DeviceQueryParams = Parameters<typeof useDeviceList>[0];
type DeviceExportParams = Partial<Omit<DeviceQueryParams, 'page' | 'pageSize'>>;

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
  const { fromRecord } = useI18nText();
  const locale = useAppStore((s) => s.locale);
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
  const [selectedDeviceMap, setSelectedDeviceMap] = useState<Record<string, Device>>({});
  const [exportModalOpen, setExportModalOpen] = useState(false);
  const [exportScope, setExportScope] = useState<DeviceGroupExportMode>('groupAll');
  // SN / 设备名称 模糊搜索（多个以逗号分隔）→ 后端 ?search= → BuildSearchOR。
  const [searchText, setSearchText] = useState('');

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
  const expandedGroupIDs = useMemo(
    () => expandSelectedGroupIds(selectedGroupId ?? undefined, groups),
    [selectedGroupId, groups]
  );
  const groupExportParams = useMemo<DeviceExportParams>(() => {
    return {
      ...(expandedGroupIDs ? { groupId: expandedGroupIDs } : {}),
    };
  }, [expandedGroupIDs]);
  const trimmedSearchText = searchText.trim();
  const hasSearchFilter = trimmedSearchText.length > 0;
  const filteredExportParams = useMemo<DeviceExportParams>(() => ({
    ...groupExportParams,
    // searchText → getList 映射为后端 ?search=（覆盖 SN/设备名称等，逗号分隔多关键字）
    ...(trimmedSearchText ? { searchText: trimmedSearchText } : {}),
  }), [groupExportParams, trimmedSearchText]);
  const queryParams = useMemo(() => {
    return {
      page: currentPage,
      pageSize,
      ...filteredExportParams,
    } as DeviceQueryParams;
  }, [currentPage, pageSize, filteredExportParams]);
  const { data: deviceData, isLoading, refetch } = useDeviceList(queryParams);
  const devices: Device[] = useMemo(
    () => withDeviceGroupDisplayName(deviceData?.items ?? [], groups, locale),
    [deviceData?.items, groups, locale]
  );
  const total = deviceData?.total ?? 0;

  const selectedGroup = useMemo(
    () => groups.find((g) => g.id === selectedGroupId),
    [groups, selectedGroupId]
  );
  const selectedGroupNameText = useMemo(
    () => selectedGroup ? (fromRecord(selectedGroup as unknown as Record<string, unknown>, 'name') || selectedGroup.name) : undefined,
    [fromRecord, selectedGroup]
  );
  const groupCountMap = useMemo(() => buildDeviceGroupSubtreeCountMap(groups), [groups]);
  const selectedGroupTotal = selectedGroupId
    ? (groupCountMap.get(selectedGroupId) ?? selectedGroup?.deviceCount ?? 0)
    : totalDevicesFromStats;
  const exportGroupName = selectedGroupNameText ?? t('common.all');

  const selectedDevices = useMemo(
    () => selectedDeviceIds
      .map((key) => selectedDeviceMap[String(key)])
      .filter((device): device is Device => Boolean(device)),
    [selectedDeviceIds, selectedDeviceMap]
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

  useEffect(() => {
    const keySet = new Set(selectedDeviceIds.map((key) => String(key)));
    setSelectedDeviceMap((prev) => {
      const next: Record<string, Device> = {};
      let changed = Object.keys(prev).length !== keySet.size;
      for (const [key, device] of Object.entries(prev)) {
        if (keySet.has(key)) {
          next[key] = device;
        } else {
          changed = true;
        }
      }
      return changed ? next : prev;
    });
  }, [selectedDeviceIds]);

  const clearDeviceSelection = useCallback(() => {
    setSelectedDeviceIds([]);
    setSelectedDeviceMap({});
  }, []);

  const handleDeviceSelectionChange = useCallback((keys: React.Key[], rows: Device[]) => {
    const keySet = new Set(keys.map((key) => String(key)));
    setSelectedDeviceIds(keys);
    setSelectedDeviceMap((prev) => {
      const next: Record<string, Device> = {};
      for (const key of keySet) {
        const existing = prev[key];
        if (existing) next[key] = existing;
      }
      for (const row of rows) {
        const key = String(row.id);
        if (keySet.has(key)) next[key] = row;
      }
      return next;
    });
  }, []);

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
      // 一级分组(root)是容器不作目标；「未分组设备」内置节点保留为"移出分组"项
      // （issue #478：选它走移出分组语义，删除归属记录让设备回到未分组态）。
      buildGroupTargetOptions(groups, getParentName, t('device.batch.removeFromGroup')),
    [groups, getParentName, t]
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
    refetchGroups,
    deleteDevicesMutation,
    setSelectedDeviceIds,
    onMoveToGroup: deviceActions.open.move,
  });

  // ── Import / export handlers ──
  const { handleExport, handleImport, handleDownloadTemplate, handlePreRegister, handleDownloadPreRegisterTemplate } = useImportExportHandlers({
    message,
    t,
    refetch,
    refetchGroups,
    selectedGroupName: selectedGroupNameText,
    groups,
    locale,
  });

  // 搜索：回车/点搜索时应用，并回到第 1 页。
  const handleSearch = useCallback((value: string) => {
    setSearchText(value);
    setCurrentPage(1);
    clearDeviceSelection();
  }, [clearDeviceSelection]);

  const handleExportClick = useCallback(() => {
    setExportScope(selectedDeviceIds.length > 0 ? 'selected' : (hasSearchFilter ? 'filtered' : 'groupAll'));
    setExportModalOpen(true);
  }, [hasSearchFilter, selectedDeviceIds.length]);

  const handleExportConfirm = useCallback(async () => {
    const normalizedScope = exportScope === 'filtered' && !hasSearchFilter ? 'groupAll' : exportScope;
    if (normalizedScope === 'selected') {
      if (selectedDevices.length === 0) {
        void message.warning(t('device.export.emptySelection'));
        return;
      }
      setExportModalOpen(false);
      await handleExport({ mode: 'selected', devices: selectedDevices });
      return;
    }

    setExportModalOpen(false);
    await handleExport({
      mode: normalizedScope,
      params: normalizedScope === 'filtered' ? filteredExportParams : groupExportParams,
    });
  }, [
    exportScope,
    hasSearchFilter,
    selectedDevices,
    message,
    t,
    handleExport,
    filteredExportParams,
    groupExportParams,
  ]);

  const exportHint = useMemo(() => {
    if (exportScope === 'selected') {
      return t('device.export.hint.selected', { count: selectedDeviceIds.length });
    }
    if (exportScope === 'filtered' && hasSearchFilter) {
      return t('device.export.hint.filtered', { count: total });
    }
    return t(
      hasSearchFilter ? 'device.export.hint.groupAllWithFilter' : 'device.export.hint.groupAll',
      { groupName: exportGroupName, count: selectedGroupTotal }
    );
  }, [exportScope, hasSearchFilter, t, selectedDeviceIds.length, total, exportGroupName, selectedGroupTotal]);
  const exportOkDisabled =
    (exportScope === 'selected' && selectedDeviceIds.length === 0) ||
    (exportScope === 'filtered' && !hasSearchFilter);

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
        clearDeviceSelection();
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
          selectedGroupId={selectedGroupId}
          selectedGroupName={selectedGroupNameText}
          batchActions={batchActions}
          onSelectionChange={handleDeviceSelectionChange}
          onPageChange={(page, size) => {
            setCurrentPage(page);
            setPageSize(size);
          }}
          onSearch={handleSearch}
          onExport={handleExportClick}
          onImport={handleImport}
          onDownloadTemplate={handleDownloadTemplate}
          onPreRegister={handlePreRegister}
          onDownloadPreRegisterTemplate={handleDownloadPreRegisterTemplate}
          t={t as (id: string, values?: Record<string, unknown>) => string}
        />
      </TreeListPageLayout>

      <Modal
        open={exportModalOpen}
        title={t('device.export.confirmAllTitle')}
        okText={t('common.export')}
        cancelText={t('common.cancel')}
        okButtonProps={{ disabled: exportOkDisabled }}
        onOk={() => void handleExportConfirm()}
        onCancel={() => setExportModalOpen(false)}
      >
        <Space orientation="vertical" size={12} style={{ width: '100%' }}>
          <Typography.Text strong>{t('device.export.scopeTitle')}</Typography.Text>
          <Radio.Group
            value={exportScope}
            onChange={(event) => setExportScope(event.target.value as DeviceGroupExportMode)}
          >
            <Space orientation="vertical" size={8}>
              {selectedDeviceIds.length > 0 && (
                <Radio value="selected">
                  {t('device.export.scope.selected', { count: selectedDeviceIds.length })}
                </Radio>
              )}
              {hasSearchFilter && (
                <Radio value="filtered">
                  {t('device.export.scope.filtered', { count: total })}
                </Radio>
              )}
              <Radio value="groupAll">
                {t('device.export.scope.groupAll', { count: selectedGroupTotal })}
              </Radio>
            </Space>
          </Radio.Group>
          <Typography.Text type="secondary">{exportHint}</Typography.Text>
        </Space>
      </Modal>

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
