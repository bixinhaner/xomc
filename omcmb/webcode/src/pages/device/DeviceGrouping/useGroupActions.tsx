import React, { useCallback } from 'react';
import { App, Form } from 'antd';
import type { GroupItem, NameFilterItem } from './types';
import { parseRangeString } from './types';
import type { UseNameFiltersReturn } from './useNameFilters';

export interface AddGroupFormValues {
  name: string;
  parentId?: string;
  description: string;
}

export interface AddChildFormValues {
  name: string;
  matchingMode: 'deviceName' | 'lac' | 'tac';
  tacRag: string;
}

interface CreateGroupArgs {
  name: string;
  parent_id?: string;
  remark: string;
  matching_mode?: string;
  name_rule_list?: NameFilterItem[];
  lac_list?: number[];
  tac_list?: number[];
}

interface UpdateGroupArgs {
  id: string;
  data: { name?: string; parent_id?: string; remark?: string };
}

export interface MutationLike<TArgs> {
  mutateAsync: (args: TArgs) => Promise<unknown>;
}

/**
 * 分组相关：状态 + handlers 一站式封装。
 * 拆分自 DeviceGrouping/index.tsx（W2.C.2 / T-0054）。
 *
 * 每个对话框（add / edit / addChild / editLevel2）的开关、目标 ID、表单
 * 都在这里 useState/useForm，并暴露对应的 open / save / cancel handler。
 */
export function useGroupActions(deps: {
  groups: GroupItem[];
  refetchGroups: () => Promise<unknown>;
  modal: ReturnType<typeof App.useApp>['modal'];
  message: ReturnType<typeof App.useApp>['message'];
  t: (id: string, values?: Record<string, string | number>) => string;
  selectedGroupId: string | null;
  setSelectedGroupId: React.Dispatch<React.SetStateAction<string | null>>;
  createGroupMutation: MutationLike<CreateGroupArgs>;
  updateGroupMutation: MutationLike<UpdateGroupArgs>;
  deleteGroupMutation: MutationLike<string>;
  childNameFilters: UseNameFiltersReturn;
  editLevel2NameFilters: UseNameFiltersReturn;
}) {
  const {
    groups, refetchGroups, modal, message, t,
    selectedGroupId, setSelectedGroupId,
    createGroupMutation, updateGroupMutation, deleteGroupMutation,
    childNameFilters, editLevel2NameFilters,
  } = deps;

  // ── 保存错误分类与提示 ──
  //
  // antd Form.validateFields() reject 时返回 { errorFields: [...] } —— 字段
  // 自身已渲染红色提示，此时不再弹 Modal。其余视为 API error：
  //   - biz_code 1107 (ErrCodeGroupNameDuplicate)：同父级下同名分组冲突，
  //     用 Modal 明确告诉用户"换一个名称"
  //   - 其他：fall back 显示后端 msg 或通用失败文案
  //
  // 之前 4 个 catch 块全部空，导致重名失败"静默无反馈"——这是用户报告的核心
  // bug。统一抽到一个 helper，4 处 catch 共享。
  const handleSaveError = useCallback(
    (err: unknown) => {
      if (err && typeof err === 'object' && 'errorFields' in err) {
        // antd 表单 validation 错误 — 字段红色提示已渲染，不再弹 Modal。
        return;
      }
      // 兼容两种 axios 错误形态：
      //   - HTTP 200 + envelope ret=0：http.ts 拦截器抛 Error 时挂 err.bizCode
      //   - HTTP 4xx：axios 抛 AxiosError，biz_code 在 err.response.data.biz_code
      const bizCode = extractBizCode(err);
      const msg = extractErrorMessage(err);
      let content: string;
      if (bizCode === 1107) {
        // ErrCodeGroupNameDuplicate — backend topology.Service.CreateGroup 抛
        content = t('device.group.nameDuplicate');
      } else {
        content = msg || t('common.operationFailed');
      }
      modal.error({
        title: t('device.group.saveFailed'),
        content,
      });
    },
    [modal, t]
  );

  // ── Modal/drawer open state ──
  const [addModalOpen, setAddModalOpen] = React.useState(false);
  const [editModalOpen, setEditModalOpen] = React.useState(false);
  const [editingGroupId, setEditingGroupId] = React.useState<string | null>(null);
  const [addChildDrawerOpen, setAddChildDrawerOpen] = React.useState(false);
  const [parentGroupId, setParentGroupId] = React.useState<string | null>(null);
  const [editLevel2DrawerOpen, setEditLevel2DrawerOpen] = React.useState(false);
  const [editLevel2GroupId, setEditLevel2GroupId] = React.useState<string | null>(null);

  // ── Forms ──
  const [addForm] = Form.useForm<AddGroupFormValues>();
  const [editForm] = Form.useForm<AddGroupFormValues>();
  const [addChildForm] = Form.useForm<AddChildFormValues>();
  const [editLevel2Form] = Form.useForm<AddChildFormValues>();

  const matchingMode = Form.useWatch('matchingMode', addChildForm);
  const editLevel2MatchingMode = Form.useWatch('matchingMode', editLevel2Form);

  // ── Open handlers ──
  const openAdd = useCallback(() => {
    addForm.resetFields();
    setAddModalOpen(true);
  }, [addForm]);

  const openAddChild = useCallback(
    (groupId: string) => {
      setParentGroupId(groupId);
      addChildForm.resetFields();
      addChildForm.setFieldsValue({ matchingMode: 'deviceName', tacRag: '' });
      childNameFilters.reset();
      setAddChildDrawerOpen(true);
    },
    [addChildForm, childNameFilters]
  );

  const openEditLevel1 = useCallback(
    (groupId: string) => {
      const grp = groups.find((g) => g.id === groupId);
      if (!grp) return;
      setEditingGroupId(groupId);
      editForm.setFieldsValue({
        name: grp.name,
        parentId: grp.parentId || undefined,
        description: grp.description,
      });
      setEditModalOpen(true);
    },
    [editForm, groups]
  );

  const openEditLevel2 = useCallback(
    (groupId: string) => {
      const grp = groups.find((g) => g.id === groupId);
      if (!grp) return;
      setEditLevel2GroupId(groupId);
      editLevel2Form.resetFields();
      editLevel2Form.setFieldsValue({ name: grp.name, matchingMode: 'deviceName', tacRag: '' });
      editLevel2NameFilters.reset();
      setEditLevel2DrawerOpen(true);
    },
    [editLevel2Form, editLevel2NameFilters, groups]
  );

  const confirmDelete = useCallback(
    (groupId: string, isLevel1: boolean) => {
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
    },
    [deleteGroupMutation, modal, message, selectedGroupId, setSelectedGroupId, t]
  );

  // ── Save handlers ──
  const handleAddGroup = useCallback(async () => {
    try {
      const values = await addForm.validateFields();
      await createGroupMutation.mutateAsync({
        name: values.name,
        parent_id: values.parentId || undefined,
        remark: values.description,
      });
      void message.success(t('common.success'));
      setAddModalOpen(false);
    } catch (err) {
      handleSaveError(err);
    }
  }, [addForm, createGroupMutation, message, t, handleSaveError]);

  const handleEditGroup = useCallback(async () => {
    try {
      if (!editingGroupId) return;
      const values = await editForm.validateFields();
      await updateGroupMutation.mutateAsync({
        id: editingGroupId,
        data: {
          name: values.name,
          parent_id: values.parentId || undefined,
          remark: values.description,
        },
      });
      void message.success(t('common.success'));
      setEditModalOpen(false);
      void refetchGroups();
    } catch (err) {
      handleSaveError(err);
    }
  }, [editForm, editingGroupId, updateGroupMutation, message, t, refetchGroups, handleSaveError]);

  const handleSaveChildGroup = useCallback(async () => {
    try {
      const values = await addChildForm.validateFields();

      let matching_mode: string | undefined;
      let name_rule_list: NameFilterItem[] | undefined;
      let lac_list: number[] | undefined;
      let tac_list: number[] | undefined;

      if (values.matchingMode === 'deviceName') {
        matching_mode = 'deviceName';
        name_rule_list = childNameFilters.filters.filter((f) => f.value && f.value.trim() !== '');
      } else if (values.matchingMode === 'lac') {
        matching_mode = 'lac';
        lac_list = parseRangeString(values.tacRag || '');
      } else if (values.matchingMode === 'tac') {
        matching_mode = 'tac';
        tac_list = parseRangeString(values.tacRag || '');
      }

      await createGroupMutation.mutateAsync({
        name: values.name,
        parent_id: parentGroupId ?? undefined,
        remark: '',
        matching_mode,
        name_rule_list,
        lac_list,
        tac_list,
      });
      void message.success(t('common.success'));
      setAddChildDrawerOpen(false);
    } catch (err) {
      handleSaveError(err);
    }
  }, [addChildForm, parentGroupId, createGroupMutation, message, t, childNameFilters.filters, handleSaveError]);

  const handleSaveEditLevel2 = useCallback(async () => {
    try {
      const values = await editLevel2Form.validateFields();
      if (!editLevel2GroupId) return;
      await updateGroupMutation.mutateAsync({
        id: editLevel2GroupId,
        data: { name: values.name },
      });
      void message.success(t('common.success'));
      setEditLevel2DrawerOpen(false);
    } catch (err) {
      handleSaveError(err);
    }
  }, [editLevel2Form, editLevel2GroupId, updateGroupMutation, message, t, handleSaveError]);

  const handleMatchingModeChange = useCallback(() => {
    childNameFilters.reset();
    addChildForm.setFieldsValue({ tacRag: '' });
  }, [addChildForm, childNameFilters]);

  return {
    forms: { addForm, editForm, addChildForm, editLevel2Form },
    state: {
      addModalOpen,
      editModalOpen,
      editingGroupId,
      addChildDrawerOpen,
      parentGroupId,
      editLevel2DrawerOpen,
      editLevel2GroupId,
      matchingMode,
      editLevel2MatchingMode,
    },
    open: {
      add: openAdd,
      addChild: openAddChild,
      editLevel1: openEditLevel1,
      editLevel2: openEditLevel2,
      confirmDelete,
    },
    close: {
      add: () => setAddModalOpen(false),
      edit: () => setEditModalOpen(false),
      addChild: () => setAddChildDrawerOpen(false),
      editLevel2: () => setEditLevel2DrawerOpen(false),
    },
    save: {
      addGroup: handleAddGroup,
      editGroup: handleEditGroup,
      saveChildGroup: handleSaveChildGroup,
      saveEditLevel2: handleSaveEditLevel2,
      onMatchingModeChange: handleMatchingModeChange,
    },
  };
}

// ── 错误抽取辅助（文件内私有） ──

/**
 * 从 axios / 业务 error 上抽 biz_code。
 *
 * 兼容三种路径：
 *   1. http.ts 拦截器手工抛的 Error & { bizCode: number }（HTTP 200 + ret=0）
 *   2. AxiosError.response.data.biz_code（HTTP 4xx 响应体仍是 envelope）
 *   3. AxiosError.response.data.bizCode（camelCase 防御性兜底）
 *
 * 未识别返 0（caller 走通用失败文案）。
 */
function extractBizCode(err: unknown): number {
  if (!err || typeof err !== 'object') return 0;
  const e = err as Record<string, unknown>;
  if (typeof e.bizCode === 'number') return e.bizCode;
  const resp = e.response as Record<string, unknown> | undefined;
  if (resp && typeof resp === 'object') {
    const data = resp.data as Record<string, unknown> | undefined;
    if (data && typeof data === 'object') {
      if (typeof data.biz_code === 'number') return data.biz_code;
      if (typeof data.bizCode === 'number') return data.bizCode;
    }
  }
  return 0;
}

/**
 * 从 error 取后端 msg（envelope.msg 字段），fallback 到 Error.message。
 *
 * 优先级：response.data.msg > Error.message > 空字符串
 */
function extractErrorMessage(err: unknown): string {
  if (!err || typeof err !== 'object') return '';
  const e = err as Record<string, unknown>;
  const resp = e.response as Record<string, unknown> | undefined;
  if (resp && typeof resp === 'object') {
    const data = resp.data as Record<string, unknown> | undefined;
    if (data && typeof data === 'object' && typeof data.msg === 'string') {
      return data.msg;
    }
  }
  if (err instanceof Error) return err.message;
  return '';
}
