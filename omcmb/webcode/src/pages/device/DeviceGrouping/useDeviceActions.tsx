import React, { useCallback, useState } from 'react';
import { Form } from 'antd';
import type { App as AppNS } from 'antd';
import type { Device, EngStatus } from '@core/types/device';
import type { MutationLike } from './useGroupActions';

export interface EditDeviceFormValues {
  engStatus: EngStatus;
  longitude: number | null;
  latitude: number | null;
  gpsHeight: number;
  remark: string;
}

export interface AddDeviceFormValues {
  addMethod: 'manual' | 'import';
  deviceSnList: string;
}

/**
 * 设备相关：state + handlers 一站式封装。
 * 拆分自 DeviceGrouping/index.tsx（W2.C.2 / T-0054）。
 */
export function useDeviceActions(deps: {
  message: ReturnType<typeof AppNS.useApp>['message'];
  t: (id: string, values?: Record<string, string | number>) => string;
  refetch: () => Promise<unknown>;
  selectedDeviceIds: React.Key[];
  setSelectedDeviceIds: React.Dispatch<React.SetStateAction<React.Key[]>>;
  updateDeviceMutation: MutationLike<{ id: string; data: Partial<EditDeviceFormValues> }>;
  moveDevicesMutation: MutationLike<{ device_ids: string[]; target_group_id: string }>;
  addDevicesToGroupMutation: MutationLike<{ groupId: string; deviceIds: string[] }>;
}) {
  const {
    message, t, refetch,
    selectedDeviceIds, setSelectedDeviceIds,
    updateDeviceMutation, moveDevicesMutation, addDevicesToGroupMutation,
  } = deps;

  // ── State ──
  const [moveToGroupModalOpen, setMoveToGroupModalOpen] = useState(false);
  const [editDeviceModalOpen, setEditDeviceModalOpen] = useState(false);
  const [editingDevice, setEditingDevice] = useState<Device | null>(null);
  const [targetGroupId, setTargetGroupId] = useState<string | null>(null);
  const [addDeviceDrawerOpen, setAddDeviceDrawerOpen] = useState(false);
  const [addDeviceGroupId, setAddDeviceGroupId] = useState<string | null>(null);

  // ── Forms ──
  const [editDeviceForm] = Form.useForm<EditDeviceFormValues>();
  const [addDeviceForm] = Form.useForm<AddDeviceFormValues>();
  const addMethod = Form.useWatch('addMethod', addDeviceForm);

  // ── Open handlers ──
  const openEdit = useCallback(
    (device: Device) => {
      setEditingDevice(device);
      editDeviceForm.setFieldsValue({
        engStatus: device.engStatus,
        longitude: device.longitude,
        latitude: device.latitude,
        gpsHeight: device.gpsHeight ?? undefined,
        remark: device.remark || '',
      });
      setEditDeviceModalOpen(true);
    },
    [editDeviceForm]
  );

  const openMove = useCallback((selectedKeys: React.Key[]) => {
    setSelectedDeviceIds(selectedKeys);
    setTargetGroupId(null);
    setMoveToGroupModalOpen(true);
  }, [setSelectedDeviceIds]);

  const openAddToGroup = useCallback(
    (groupId: string) => {
      setAddDeviceGroupId(groupId);
      addDeviceForm.resetFields();
      addDeviceForm.setFieldsValue({ addMethod: 'manual', deviceSnList: '' });
      setAddDeviceDrawerOpen(true);
    },
    [addDeviceForm]
  );

  // ── Save handlers ──
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
  }, [targetGroupId, selectedDeviceIds, moveDevicesMutation, message, t, refetch, setSelectedDeviceIds]);

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

  return {
    forms: { editDeviceForm, addDeviceForm },
    state: {
      moveToGroupModalOpen,
      editDeviceModalOpen,
      editingDevice,
      targetGroupId,
      addDeviceDrawerOpen,
      addMethod,
    },
    setTargetGroupId,
    open: {
      edit: openEdit,
      move: openMove,
      addToGroup: openAddToGroup,
    },
    close: {
      move: () => setMoveToGroupModalOpen(false),
      edit: () => setEditDeviceModalOpen(false),
      addToGroup: () => setAddDeviceDrawerOpen(false),
    },
    save: {
      device: handleSaveDevice,
      moveToGroup: handleMoveToGroup,
      devices: handleSaveDevices,
    },
  };
}
