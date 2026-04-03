import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { App, Form } from 'antd';
import {
  DeleteOutlined,
  FolderOutlined as MoveToGroupIcon,
  RestOutlined,
} from '@ant-design/icons';
import type { UploadFile } from 'antd';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import type { BatchAction } from '@/components/DataTable';
import { useDeviceGroups, useDeviceList, useCreateGroup, useUpdateGroup, useDeleteGroup, useMoveDevices, useAddDevicesToGroup, useDeleteDevices, useBatchRebootDevices, useUpdateDevice } from '@/hooks/api/useDevices';
import { useT } from '@/hooks/useT';
import type { Device, EngStatus } from '@/types/device';
import type { NameFilterItem } from './types';
import { generateId, generateOperators } from './types';
import GroupTreePanel from './GroupTreePanel';
import DeviceListPanel from './DeviceListPanel';
import GroupDialogs from './GroupDialogs';
import DeviceDialogs from './DeviceDialogs';

export default function DeviceGrouping() {
  const t = useT();
  const { modal, message } = App.useApp();
  const { data: groupsData, refetch: refetchGroups } = useDeviceGroups();
  const groups = groupsData?.groups ?? [];
  const totalDevicesFromStats = groupsData?.stats?.totalDevices ?? 0;

  // --- Selection & pagination state ---
  const [selectedGroupId, setSelectedGroupId] = useState<string | null>(null);
  const [groupSearchText, setGroupSearchText] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selectedDeviceIds, setSelectedDeviceIds] = useState<React.Key[]>([]);

  const createGroupMutation = useCreateGroup();
  const updateGroupMutation = useUpdateGroup();
  const deleteGroupMutation = useDeleteGroup();
  const moveDevicesMutation = useMoveDevices();
  const addDevicesToGroupMutation = useAddDevicesToGroup();
  const deleteDevicesMutation = useDeleteDevices();
  const batchRebootMutation = useBatchRebootDevices();
  const updateDeviceMutation = useUpdateDevice();

  // --- Group dialog state ---
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editingGroupId, setEditingGroupId] = useState<string | null>(null);
  const [addForm] = Form.useForm<{ name: string; description: string }>();
  const [editForm] = Form.useForm<{ name: string; description: string }>();

  // --- Add child group state ---
  const [addChildDrawerOpen, setAddChildDrawerOpen] = useState(false);
  const [parentGroupId, setParentGroupId] = useState<string | null>(null);
  const [addChildForm] = Form.useForm<{
    name: string;
    matchingMode: 'deviceName' | 'lac' | 'tac';
    tacRag: string;
  }>();
  const [nameFilters, setNameFilters] = useState<NameFilterItem[]>([
    { id: generateId(), condition: 'contain', value: '' },
  ]);
  const matchingMode = Form.useWatch('matchingMode', addChildForm);

  // --- Edit level-2 group state ---
  const [editLevel2DrawerOpen, setEditLevel2DrawerOpen] = useState(false);
  const [editLevel2GroupId, setEditLevel2GroupId] = useState<string | null>(null);
  const [editLevel2Form] = Form.useForm<{
    name: string;
    matchingMode: 'deviceName' | 'lac' | 'tac';
    tacRag: string;
  }>();
  const [editLevel2NameFilters, setEditLevel2NameFilters] = useState<NameFilterItem[]>([
    { id: generateId(), condition: 'contain', value: '' },
  ]);
  const editLevel2MatchingMode = Form.useWatch('matchingMode', editLevel2Form);

  // --- Device dialog state ---
  const [moveToGroupModalOpen, setMoveToGroupModalOpen] = useState(false);
  const [editDeviceModalOpen, setEditDeviceModalOpen] = useState(false);
  const [editingDevice, setEditingDevice] = useState<Device | null>(null);
  const [targetGroupId, setTargetGroupId] = useState<string | null>(null);
  const [editDeviceForm] = Form.useForm<{
    engStatus: EngStatus;
    longitude: number;
    latitude: number;
    gpsHeight: number;
    remark: string;
  }>();

  // --- Add device to group state ---
  const [addDeviceDrawerOpen, setAddDeviceDrawerOpen] = useState(false);
  const [addDeviceGroupId, setAddDeviceGroupId] = useState<string | null>(null);
  const [addDeviceForm] = Form.useForm<{
    addMethod: 'manual' | 'import';
    deviceSnList: string;
  }>();
  const addMethod = Form.useWatch('addMethod', addDeviceForm);

  // --- Data fetching ---
  const queryParams = useMemo(
    () => {
      console.log('[DeviceGrouping] queryParams changed:', {
        page: currentPage,
        pageSize,
        groupId: selectedGroupId ?? undefined,
      });
      return { page: currentPage, pageSize, groupId: selectedGroupId ?? undefined } as Parameters<typeof useDeviceList>[0];
    },
    [currentPage, pageSize, selectedGroupId]
  );
  const { data: deviceData, isLoading, refetch } = useDeviceList(queryParams);
  const devices: Device[] = deviceData?.items ?? [];
  const total = deviceData?.total ?? 0;

  const selectedGroup = useMemo(
    () => groups.find((g) => g.id === selectedGroupId),
    [groups, selectedGroupId]
  );

  // Default select first level-2 node under the first root group
  useEffect(() => {
    if (groups.length > 0 && !selectedGroupId) {
      // 兼容 null 和 undefined（后端 omitempty 导致根分组没有 parent_id 字段）
      const firstRoot = groups.find((g) => !g.parentId);
      if (firstRoot) {
        const firstChild = groups.find((g) => g.parentId === firstRoot.id);
        if (firstChild) {
          setSelectedGroupId(firstChild.id);
        }
      }
    }
  }, [groups, selectedGroupId]);

  // --- Filtered groups for tree search ---
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

  // --- Target group options for move-to-group modal ---
  const getParentName = useCallback((parentId: string | null): string => {
    if (!parentId) return '';
    const parent = groups.find((g) => g.id === parentId);
    return parent?.name ?? '';
  }, [groups]);

  const targetGroupOptions = useMemo(() => {
    return groups
      .filter((g) => g.parentId !== null)
      .map((g) => {
        const parentName = getParentName(g.parentId);
        return {
          label: parentName ? `${parentName} / ${g.name}` : g.name,
          value: g.id,
        };
      });
  }, [groups, getParentName]);

  // --- Tree context menu handler ---
  const handleContextMenu = useCallback(
    (action: string) => {
      const [cmd, groupId] = action.split(':');
      if (cmd === 'add-child') {
        setParentGroupId(groupId);
        addChildForm.resetFields();
        addChildForm.setFieldsValue({ matchingMode: 'deviceName', tacRag: '' });
        setNameFilters([{ id: generateId(), condition: 'contain', value: '' }]);
        setAddChildDrawerOpen(true);
      } else if (cmd === 'add-device') {
        setAddDeviceGroupId(groupId);
        addDeviceForm.resetFields();
        addDeviceForm.setFieldsValue({ addMethod: 'manual', deviceSnList: '' });
        setAddDeviceDrawerOpen(true);
      } else if (cmd === 'edit-level1') {
        const grp = groups.find((g) => g.id === groupId);
        if (grp) {
          setEditingGroupId(groupId);
          editForm.setFieldsValue({ name: grp.name, description: grp.description });
          setEditModalOpen(true);
        }
      } else if (cmd === 'edit-level2') {
        const grp = groups.find((g) => g.id === groupId);
        if (grp) {
          setEditLevel2GroupId(groupId);
          editLevel2Form.resetFields();
          editLevel2Form.setFieldsValue({ name: grp.name, matchingMode: 'deviceName', tacRag: '' });
          setEditLevel2NameFilters([{ id: generateId(), condition: 'contain', value: '' }]);
          setEditLevel2DrawerOpen(true);
        }
      } else if (cmd === 'delete-level1' || cmd === 'delete-level2') {
        const isLevel1 = cmd === 'delete-level1';
        modal.confirm({
          title: t('common.confirmDelete'),
          width: 480,
          content: (
            <div>
              <div>{t('common.deleteConfirmMsg')}</div>
              <div style={{ marginTop: 8, color: 'var(--color-text-secondary)', fontSize: 13, whiteSpace: 'nowrap' }}>
                {isLevel1 ? t('device.deleteLevel1Desc') : t('device.deleteLevel2Desc')}
              </div>
            </div>
          ),
          okText: t('common.confirmDelete'),
          okType: 'danger',
          onOk: async () => {
            try {
              await deleteGroupMutation.mutateAsync(groupId);
              if (selectedGroupId === groupId) {
                setSelectedGroupId(null);
              }
              void message.success(t('common.deleteSuccess'));
            } catch {
              void message.error(t('common.operationFailed'));
            }
          },
        });
      }
    },
    [groups, addChildForm, addDeviceForm, editForm, editLevel2Form, deleteGroupMutation, selectedGroupId, t, modal, message]
  );

  // --- Group handlers ---
  const handleAddGroup = useCallback(async () => {
    try {
      const values = await addForm.validateFields();
      // 支持选择父级分组
      await createGroupMutation.mutateAsync({
        name: values.name,
        parent_id: values.parentId || undefined,
        remark: values.description,
      });
      void message.success(t('common.success'));
      setAddModalOpen(false);
    } catch {
      // validation or API error
    }
  }, [addForm, createGroupMutation, message, t]);

  const handleEditGroup = useCallback(async () => {
    try {
      if (!editingGroupId) return;
      const values = await editForm.validateFields();
      await updateGroupMutation.mutateAsync({ id: editingGroupId, data: { name: values.name, remark: values.description } });
      void message.success(t('common.success'));
      setEditModalOpen(false);
    } catch {
      // validation or API error
    }
  }, [editForm, editingGroupId, updateGroupMutation, message, t]);

  // --- Add child group filter handlers ---
  const handleAddFilter = useCallback(() => {
    if (nameFilters.length >= 10) {
      void message.warning(t('device.rules.maxConditions', { max: 10 }));
      return;
    }
    const hasOr = nameFilters.some((f, index) => index > 0 && f.andOr === 'or');
    setNameFilters((prev) => [
      ...prev,
      { id: generateId(), condition: 'contain', value: '', andOr: hasOr ? 'or' : 'and' },
    ]);
  }, [nameFilters, message, t]);

  const handleRemoveFilter = useCallback((id: string) => {
    setNameFilters((prev) => {
      if (prev.length <= 1) return prev;
      const newFilters = prev.filter((f) => f.id !== id);
      if (newFilters.length > 0 && newFilters[0].andOr !== undefined) {
        const { andOr: _, ...rest } = newFilters[0];
        newFilters[0] = rest as NameFilterItem;
      }
      return newFilters;
    });
  }, []);

  const handleUpdateFilter = useCallback((id: string, field: keyof NameFilterItem, value: string) => {
    setNameFilters((prev) => prev.map((f) => (f.id === id ? { ...f, [field]: value } : f)));
  }, []);

  const handleMatchingModeChange = useCallback(() => {
    setNameFilters([{ id: generateId(), condition: 'contain', value: '' }]);
    addChildForm.setFieldsValue({ tacRag: '' });
  }, [addChildForm]);

  const handleSaveChildGroup = useCallback(async () => {
    try {
      const values = await addChildForm.validateFields();
      await createGroupMutation.mutateAsync({
        name: values.name,
        parent_id: parentGroupId ?? undefined,
        remark: '',
      });
      void message.success(t('common.success'));
      setAddChildDrawerOpen(false);
    } catch {
      // validation or API error
    }
  }, [addChildForm, parentGroupId, createGroupMutation, message, t]);

  // --- Edit level-2 handler ---
  const handleSaveEditLevel2 = useCallback(async () => {
    try {
      const values = await editLevel2Form.validateFields();
      await updateGroupMutation.mutateAsync({
        id: editLevel2GroupId!,
        data: { name: values.name },
      });
      void message.success(t('common.success'));
      setEditLevel2DrawerOpen(false);
    } catch {
      // validation or API error
    }
  }, [editLevel2Form, editLevel2GroupId, updateGroupMutation, message, t]);

  // --- Device handlers ---
  const handleEditDevice = useCallback((device: Device) => {
    setEditingDevice(device);
    editDeviceForm.setFieldsValue({
      engStatus: device.engStatus,
      longitude: device.longitude,
      latitude: device.latitude,
      gpsHeight: device.gpsHeight,
      remark: device.remark || '',
    });
    setEditDeviceModalOpen(true);
  }, [editDeviceForm]);

  const handleSaveDevice = useCallback(async () => {
    try {
      const values = await editDeviceForm.validateFields();
      if (editingDevice) {
        await updateDeviceMutation.mutateAsync({
          id: editingDevice.id,
          data: {
            engStatus: values.engStatus,
            longitude: values.longitude,
            latitude: values.latitude,
            gpsHeight: values.gpsHeight,
            remark: values.remark,
          },
        });
      }
      void message.success(t('common.operationSuccess'));
      setEditDeviceModalOpen(false);
      await refetch();
    } catch {
      // validation or API error
    }
  }, [editDeviceForm, editingDevice, updateDeviceMutation, message, t, refetch]);

  const handleMoveToGroup = useCallback(async () => {
    if (!targetGroupId) {
      message.warning(t('device.batch.selectGroup'));
      return;
    }
    try {
      await moveDevicesMutation.mutateAsync({
        device_ids: selectedDeviceIds as string[],
        target_group_id: targetGroupId,
      });
      void message.success(t('common.success'));
      setMoveToGroupModalOpen(false);
      setSelectedDeviceIds([]);
      await refetch();
    } catch {
      void message.error(t('common.operationFailed'));
    }
  }, [targetGroupId, selectedDeviceIds, moveDevicesMutation, message, t, refetch]);

  const handleSaveDevices = useCallback(async () => {
    try {
      const values = await addDeviceForm.validateFields();
      const snList = values.deviceSnList?.trim();
      if (!snList) {
        void message.error(t('device.snListRequired'));
        return;
      }
      const snArray = snList.split(/[\n,;]+/).map((sn) => sn.trim()).filter((sn) => sn.length > 0);
      if (snArray.length === 0) {
        void message.error(t('device.snListRequired'));
        return;
      }
      if (!addDeviceGroupId) return;
      // TODO: SNs should be resolved to device IDs via backend lookup before sending.
      // For now, pass SNs as device_ids — backend may accept SN resolution in the future.
      // Frontend should ideally call a device search API to resolve SNs → UUIDs first.
      await addDevicesToGroupMutation.mutateAsync({
        groupId: addDeviceGroupId,
        deviceIds: snArray,
      });
      void message.success(t('device.addDeviceSuccess', { count: snArray.length }));
      setAddDeviceDrawerOpen(false);
      await refetch();
    } catch {
      // validation or API error
    }
  }, [addDeviceForm, addDeviceGroupId, addDevicesToGroupMutation, refetch, message, t]);

  const handleExport = useCallback(() => {
    // TODO: 调用 CSV 导出 API
    message.success(t('common.exportInProgress'));
  }, [message, t]);

  const handleImport = useCallback(async (fileList: UploadFile[]) => {
    if (fileList.length === 0) {
      void message.error(t('device.fileRequired'));
      return;
    }
    void message.success(t('device.importSuccess'));
    await refetch();
  }, [message, t, refetch]);

  const handleDownloadTemplate = useCallback(() => {
    // 生成导入模板 CSV
    const csvContent = 'SN,名称,经度,纬度,高度,备注\n';
    const blob = new Blob(['\ufeff' + csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'device_import_template.csv';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    void message.success(t('common.download'));
  }, [message, t]);

  // --- Batch actions ---
  const batchActions = useMemo((): BatchAction[] => [
    {
      key: 'moveToGroup',
      label: t('device.batch.moveToGroup'),
      icon: <MoveToGroupIcon />,
      onClick: (selectedKeys: React.Key[]) => {
        setSelectedDeviceIds(selectedKeys);
        setTargetGroupId(null);
        setMoveToGroupModalOpen(true);
      },
    },
    {
      key: 'recycle',
      label: t('device.batch.recycle'),
      icon: <RestOutlined />,
      onClick: (selectedKeys: React.Key[]) => {
        modal.confirm({
          title: t('device.batch.recycleConfirm'),
          width: 480,
          content: (
            <div>
              <div>{t('device.batch.recycleMsg', { count: selectedKeys.length })}</div>
              <div style={{ marginTop: 8, color: 'var(--color-text-secondary)', fontSize: 13, whiteSpace: 'nowrap' }}>
                {t('device.batch.deleteWarning')}
              </div>
            </div>
          ),
          okText: t('common.confirm'),
          okType: 'danger',
          onOk: async () => {
            try {
              await batchRebootMutation.mutateAsync(selectedKeys as string[]);
              void message.success(t('common.success'));
            } catch {
              void message.error(t('common.operationFailed'));
            }
            setSelectedDeviceIds([]);
            await refetch();
          },
        });
      },
    },
    {
      key: 'delete',
      label: t('common.delete'),
      icon: <DeleteOutlined />,
      danger: true,
      onClick: (selectedKeys: React.Key[]) => {
        modal.confirm({
          title: t('common.confirmDelete'),
          width: 480,
          content: (
            <div>
              <div>{t('common.deleteConfirmMsg', { count: selectedKeys.length })}</div>
              <div style={{ marginTop: 8, color: 'var(--color-text-secondary)', fontSize: 13, whiteSpace: 'nowrap' }}>
                {t('device.batch.deleteWarning')}
              </div>
            </div>
          ),
          okText: t('common.confirmDelete'),
          okType: 'danger',
          onOk: async () => {
            try {
              await deleteDevicesMutation.mutateAsync(selectedKeys as string[]);
              void message.success(t('common.deleteSuccess'));
            } catch {
              void message.error(t('common.operationFailed'));
            }
            setSelectedDeviceIds([]);
            await refetch();
          },
        });
      },
    },
  ], [t, modal, message, refetch, deleteDevicesMutation, batchRebootMutation]);

  // --- Tree panel ---
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
      onAddGroup={() => {
        addForm.resetFields();
        setAddModalOpen(true);
      }}
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
          onExport={handleExport}
          onImport={handleImport}
          onDownloadTemplate={handleDownloadTemplate}
          onEditDevice={handleEditDevice}
          t={t as (id: string, values?: Record<string, unknown>) => string}
        />
      </TreeListPageLayout>

      <GroupDialogs
        addModalOpen={addModalOpen}
        addForm={addForm}
        groups={groups}
        groups={groups}
        onAddModalOk={() => void handleAddGroup()}
        onAddModalCancel={() => setAddModalOpen(false)}
        editModalOpen={editModalOpen}
        editForm={editForm}
        onEditModalOk={() => void handleEditGroup()}
        onEditModalCancel={() => setEditModalOpen(false)}
        addChildDrawerOpen={addChildDrawerOpen}
        addChildForm={addChildForm}
        matchingMode={matchingMode}
        nameFilters={nameFilters}
        onAddChildDrawerClose={() => setAddChildDrawerOpen(false)}
        onSaveChildGroup={() => void handleSaveChildGroup()}
        onMatchingModeChange={handleMatchingModeChange}
        onAddFilter={handleAddFilter}
        onRemoveFilter={handleRemoveFilter}
        onUpdateFilter={handleUpdateFilter}
        editLevel2DrawerOpen={editLevel2DrawerOpen}
        editLevel2Form={editLevel2Form}
        editLevel2MatchingMode={editLevel2MatchingMode}
        editLevel2NameFilters={editLevel2NameFilters}
        onEditLevel2DrawerClose={() => setEditLevel2DrawerOpen(false)}
        onSaveEditLevel2={() => void handleSaveEditLevel2()}
        onEditLevel2NameFiltersChange={setEditLevel2NameFilters}
        t={t as (id: string, values?: Record<string, unknown>) => string}
      />

      <DeviceDialogs
        moveToGroupModalOpen={moveToGroupModalOpen}
        selectedDeviceIds={selectedDeviceIds}
        targetGroupId={targetGroupId}
        targetGroupOptions={targetGroupOptions}
        onTargetGroupChange={setTargetGroupId}
        onMoveToGroupOk={() => void handleMoveToGroup()}
        onMoveToGroupCancel={() => setMoveToGroupModalOpen(false)}
        editDeviceModalOpen={editDeviceModalOpen}
        editingDevice={editingDevice}
        editDeviceForm={editDeviceForm}
        onEditDeviceOk={() => void handleSaveDevice()}
        onEditDeviceCancel={() => setEditDeviceModalOpen(false)}
        addDeviceDrawerOpen={addDeviceDrawerOpen}
        addDeviceForm={addDeviceForm}
        addMethod={addMethod}
        onAddDeviceDrawerClose={() => setAddDeviceDrawerOpen(false)}
        onSaveDevices={() => void handleSaveDevices()}
        onDownloadTemplate={handleDownloadTemplate}
        t={t as (id: string, values?: Record<string, unknown>) => string}
      />
    </>
  );
}
