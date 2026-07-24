import React, { useCallback } from 'react';
import { App, Form } from 'antd';
import type { GroupItem, NameFilterItem } from './types';
import { parseRangeString } from './types';
import type { UseNameFiltersReturn } from './useNameFilters';
import { getRecordI18n, type Locale } from '@core/utils/i18nText';
import { UNASSIGNED_GROUP_ID } from '@core/utils/deviceGroupTargets';
import styles from './DeviceGrouping.module.css';

const DEFAULT_ROOT_GROUP_ID = '00000000-0000-0000-0000-000000000001';

export interface AddGroupFormValues {
  /** 单值名称 — 表单只有一个 antd Input（去多语言，方案 A）。 */
  name?: string;
  description?: string;
  parentId?: string;
}

export interface AddChildFormValues {
  /** 单值名称 — 与一级分组一致，单个 antd Input。 */
  name?: string;
  autoAssignEnabled?: boolean;
  matchingMode: 'deviceName' | 'lac' | 'tac' | 'serialNumber';
  tacRag: string;
  sourceGroupId?: string;
  serialNumbers?: string;
}

interface CreateGroupArgs {
  name: string;
  name_i18n?: Record<string, string>;
  description_i18n?: Record<string, string>;
  remark_i18n?: Record<string, string>;
  parent_id?: string;
  remark: string;
  matching_mode?: string;
  source_group_id?: string;
  name_rule_list?: NameFilterItem[];
  lac_list?: number[];
  tac_list?: number[];
  serial_number_list?: string[];
}

interface UpdateGroupArgs {
  id: string;
  data: {
    name?: string;
    name_i18n?: Record<string, string>;
    description_i18n?: Record<string, string>;
    remark_i18n?: Record<string, string>;
    parent_id?: string;
    remark?: string;
    /**
     * 匹配规则字段（device-list-and-group-improvements-20260520.md R1.1）：
     * 之前类型只允许 name/parent_id/remark，导致 L2 编辑改匹配规则时被 TS 静默
     * 截断 → 后端收不到 → fireGroupMatch 跑旧规则 → 设备不重新入组。
     */
    matching_mode?: 'deviceName' | 'lac' | 'tac' | 'serialNumber' | '';
    source_group_id?: string;
    name_rule_list?: NameFilterItem[];
    lac_list?: number[];
    tac_list?: number[];
    serial_number_list?: string[];
  };
}

export interface MutationLike<TArgs> {
  mutateAsync: (args: TArgs) => Promise<unknown>;
}

function getDeleteImpact(groups: GroupItem[], groupId: string, isLevel1: boolean) {
  const group = groups.find((g) => g.id === groupId);
  const children = isLevel1 ? groups.filter((g) => g.parentId === groupId) : [];
  const deviceCount = [group, ...children]
    .filter((g): g is GroupItem => Boolean(g))
    .reduce((sum, g) => sum + (g.deviceCount ?? 0), 0);

  return {
    group,
    childCount: children.length,
    deviceCount,
  };
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
  locale: Locale;
  selectedGroupId: string | null;
  setSelectedGroupId: React.Dispatch<React.SetStateAction<string | null>>;
  createGroupMutation: MutationLike<CreateGroupArgs>;
  updateGroupMutation: MutationLike<UpdateGroupArgs>;
  deleteGroupMutation: MutationLike<string>;
  childNameFilters: UseNameFiltersReturn;
  editLevel2NameFilters: UseNameFiltersReturn;
}) {
  const {
    groups, refetchGroups, modal, message, t, locale,
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
  const autoAssignEnabled = Form.useWatch('autoAssignEnabled', addChildForm);
  const editLevel2MatchingMode = Form.useWatch('matchingMode', editLevel2Form);
  const editLevel2AutoAssignEnabled = Form.useWatch('autoAssignEnabled', editLevel2Form);
  const getGroupName = useCallback(
    (group: GroupItem): string => getBuiltInGroupName(group, t) ||
      getRecordI18n(group as unknown as Record<string, unknown>, 'name', locale) ||
      group.name,
    [locale, t]
  );

  // ── Open handlers ──
  const openAdd = useCallback(() => {
    addForm.resetFields();
    setAddModalOpen(true);
  }, [addForm]);

  const openAddChild = useCallback(
    (groupId: string) => {
      setParentGroupId(groupId);
      addChildForm.resetFields();
      addChildForm.setFieldsValue({
        autoAssignEnabled: false,
        matchingMode: 'deviceName',
        tacRag: '',
        serialNumbers: '',
        sourceGroupId: undefined,
      });
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
      // 单值回填:按当前语言展示，缺失时由 i18n 工具回退到中文/legacy。
      editForm.setFieldsValue({
        name: getGroupName(grp),
        description: grp.description || grp.descriptionI18n?.['zh-CN'] || '',
        parentId: grp.parentId || undefined,
      });
      setEditModalOpen(true);
    },
    [editForm, getGroupName, groups]
  );

  const openEditLevel2 = useCallback(
    (groupId: string) => {
      const grp = groups.find((g) => g.id === groupId);
      if (!grp) return;
      setEditLevel2GroupId(groupId);
      editLevel2Form.resetFields();

      // R1.2: 回填当前分组的匹配规则到表单。之前这里硬编码 'deviceName' + 空
      // tacRag，无视 grp 的真实规则，导致用户打开编辑抽屉看到的就是空白。
      const mode: AddChildFormValues['matchingMode'] =
        grp.matchingMode === 'lac' || grp.matchingMode === 'tac' || grp.matchingMode === 'serialNumber'
          ? grp.matchingMode
          : 'deviceName';
      const autoAssign = Boolean(grp.matchingMode && grp.sourceGroupId);

      let tacRag = '';
      if (mode === 'lac' && grp.lacList && grp.lacList.length > 0) {
        tacRag = grp.lacList.join(',');
      } else if (mode === 'tac' && grp.tacList && grp.tacList.length > 0) {
        tacRag = grp.tacList.join(',');
      }

      // 单值名称回填：按当前语言展示，缺失时由 i18n 工具回退到中文/legacy。
      editLevel2Form.setFieldsValue({
        name: getGroupName(grp),
        autoAssignEnabled: autoAssign,
        matchingMode: mode,
        tacRag,
        sourceGroupId: grp.sourceGroupId,
        serialNumbers: grp.serialNumberList?.join(',') ?? '',
      });

      // 回填 name_rule_list 到 NameFilters hook 的内部状态
      if (autoAssign && mode === 'deviceName' && Array.isArray(grp.nameRuleList) && grp.nameRuleList.length > 0) {
        // 注入既有规则；保持原 ID 让 React key 稳定（避免不必要重渲染）
        editLevel2NameFilters.setFilters(grp.nameRuleList.map((r, i) => ({
          // 后端返回的 NameFilterItem 可能缺 id（仅 condition/value/andOr）；缺失时补一个
          id: r.id || `r-${i}-${Date.now()}`,
          condition: r.condition,
          value: r.value,
          andOr: r.andOr,
        })));
      } else {
        editLevel2NameFilters.reset();
      }

      setEditLevel2DrawerOpen(true);
    },
    [editLevel2Form, editLevel2NameFilters, getGroupName, groups]
  );

  const confirmDelete = useCallback(
    (groupId: string, isLevel1: boolean) => {
      const { group, childCount, deviceCount } = getDeleteImpact(groups, groupId, isLevel1);
      if (!group) {
        void message.error(t('common.operationFailed'));
        return;
      }

      modal.confirm({
        title: t('common.confirmDelete'),
        width: 520,
        content: (
          <div className={styles.groupDeleteConfirm}>
            <div className={styles.groupDeleteConfirmTitle}>
              {t('device.group.deleteConfirmMsg', { name: getGroupName(group) })}
            </div>
            <div className={styles.groupDeleteImpactList}>
              {isLevel1 ? (
                <div className={styles.groupDeleteImpactItem}>
                  <span>{t('device.group.deleteChildGroupsLabel')}</span>
                  <strong>{t('device.group.deleteChildGroupsValue', { count: childCount })}</strong>
                </div>
              ) : null}
              <div className={styles.groupDeleteImpactItem}>
                <span>{t('device.group.deleteDevicesLabel')}</span>
                <strong>{t('device.group.deleteDevicesValue', { count: deviceCount })}</strong>
              </div>
              <div className={styles.groupDeleteImpactItem}>
                <span>{t('device.group.deleteResultLabel')}</span>
                <strong>{t('device.group.deleteResultUngrouped')}</strong>
              </div>
            </div>
            <div className={styles.groupDeleteConfirmHint}>
              {t('device.group.deleteDevicePreserveHint')}
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
    [deleteGroupMutation, getGroupName, groups, modal, message, selectedGroupId, setSelectedGroupId, t]
  );

  // ── Save handlers ──
  const handleAddGroup = useCallback(async () => {
    try {
      const values = await addForm.validateFields();
      // 单值表单 → 同时发顶层单值与 i18n 单键({'zh-CN': 值}),后端读哪个都拿到同一值。
      const name = values.name || '';
      const desc = values.description || '';
      await createGroupMutation.mutateAsync({
        name,
        name_i18n: { [locale]: name },
        description_i18n: { 'zh-CN': desc },
        parent_id: values.parentId || undefined,
        remark: desc,
        remark_i18n: { 'zh-CN': desc },
      });
      void message.success(t('common.success'));
      setAddModalOpen(false);
    } catch (err) {
      handleSaveError(err);
    }
  }, [addForm, createGroupMutation, locale, message, t, handleSaveError]);

  const handleEditGroup = useCallback(async () => {
    try {
      if (!editingGroupId) return;
      const values = await editForm.validateFields();
      // 单值表单 → 同时发顶层单值与 i18n 单键({'zh-CN': 值})。
      const name = values.name || '';
      const desc = values.description || '';
      const currentGroup = groups.find((g) => g.id === editingGroupId);
      await updateGroupMutation.mutateAsync({
        id: editingGroupId,
        data: {
          name,
          name_i18n: { ...(currentGroup?.nameI18n ?? {}), [locale]: name },
          description_i18n: { 'zh-CN': desc },
          parent_id: values.parentId || undefined,
          remark: desc,
          remark_i18n: { 'zh-CN': desc },
        },
      });
      void message.success(t('common.success'));
      setEditModalOpen(false);
      void refetchGroups();
    } catch (err) {
      handleSaveError(err);
    }
  }, [editForm, editingGroupId, groups, locale, updateGroupMutation, message, t, refetchGroups, handleSaveError]);

  const handleSaveChildGroup = useCallback(async () => {
    try {
      const values = await addChildForm.validateFields();
      // 单值名称 → 同时发顶层单值与 i18n 单键({'zh-CN': 值})（同一级）。
      const name = values.name || '';

      let matching_mode: string | undefined;
      let name_rule_list: NameFilterItem[] | undefined;
      let lac_list: number[] | undefined;
      let tac_list: number[] | undefined;
      let serial_number_list: string[] | undefined;

      if (!values.autoAssignEnabled) {
        matching_mode = undefined;
      } else if (values.matchingMode === 'deviceName') {
        matching_mode = 'deviceName';
        name_rule_list = childNameFilters.filters.filter((f) => f.value && f.value.trim() !== '');
      } else if (values.matchingMode === 'lac') {
        matching_mode = 'lac';
        lac_list = parseRangeString(values.tacRag || '');
      } else if (values.matchingMode === 'tac') {
        matching_mode = 'tac';
        tac_list = parseRangeString(values.tacRag || '');
	  } else if (values.matchingMode === 'serialNumber') {
		matching_mode = 'serialNumber';
		serial_number_list = parseSerialNumbers(values.serialNumbers);
      }

      await createGroupMutation.mutateAsync({
        name,
        name_i18n: { [locale]: name },
        parent_id: parentGroupId ?? undefined,
        remark: '',
        matching_mode,
        source_group_id: values.autoAssignEnabled ? values.sourceGroupId : undefined,
        name_rule_list,
        lac_list,
        tac_list,
        serial_number_list,
      });
      void message.success(t('common.success'));
      setAddChildDrawerOpen(false);
    } catch (err) {
      handleSaveError(err);
    }
  }, [addChildForm, parentGroupId, createGroupMutation, locale, message, t, childNameFilters.filters, handleSaveError]);

  const handleSaveEditLevel2 = useCallback(async () => {
    try {
      const values = await editLevel2Form.validateFields();
      if (!editLevel2GroupId) return;
      // 单值名称 → 同时发顶层单值与 i18n 单键({'zh-CN': 值})（同一级）。
      const name = values.name || '';
      const currentGroup = groups.find((g) => g.id === editLevel2GroupId);

      // R1.3: 全量替换语义 — 用户在表单上看到的就是最终落库的，避免增量合并歧义。
      // 切换 matchingMode 时显式清空非当前模式的列表字段，让后端覆盖为空数组。
      let matching_mode: 'deviceName' | 'lac' | 'tac' | 'serialNumber' | '' = '';
      let name_rule_list: NameFilterItem[] = [];
      let lac_list: number[] = [];
      let tac_list: number[] = [];
      let serial_number_list: string[] = [];

      if (!values.autoAssignEnabled) {
        matching_mode = '';
      } else if (values.matchingMode === 'deviceName') {
        matching_mode = 'deviceName';
        name_rule_list = editLevel2NameFilters.filters.filter(
          (f) => f.value && f.value.trim() !== ''
        );
      } else if (values.matchingMode === 'lac') {
        matching_mode = 'lac';
        lac_list = parseRangeString(values.tacRag || '');
      } else if (values.matchingMode === 'tac') {
        matching_mode = 'tac';
        tac_list = parseRangeString(values.tacRag || '');
	  } else if (values.matchingMode === 'serialNumber') {
		matching_mode = 'serialNumber';
		serial_number_list = parseSerialNumbers(values.serialNumbers);
      }

      await updateGroupMutation.mutateAsync({
        id: editLevel2GroupId,
        data: {
          name,
          name_i18n: { ...(currentGroup?.nameI18n ?? {}), [locale]: name },
          matching_mode,
          source_group_id: values.autoAssignEnabled ? values.sourceGroupId : '',
          name_rule_list,
          lac_list,
          tac_list,
          serial_number_list,
        },
      });
      // 后端异步 fireGroupMatch 可能命中 0 台，提示只确认规则保存和匹配触发。
      void message.success(t('device.group.editMatchingSuccess'));
      setEditLevel2DrawerOpen(false);
      void refetchGroups();
    } catch (err) {
      handleSaveError(err);
    }
  }, [
    editLevel2Form,
    editLevel2GroupId,
    editLevel2NameFilters.filters,
    groups,
    locale,
    updateGroupMutation,
    message,
    t,
    refetchGroups,
    handleSaveError,
  ]);

  const handleMatchingModeChange = useCallback(() => {
    childNameFilters.reset();
    addChildForm.setFieldsValue({ tacRag: '', serialNumbers: '' });
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
      autoAssignEnabled,
      editLevel2MatchingMode,
      editLevel2AutoAssignEnabled,
      // 上级（一级）分组名称，供子分组新增/编辑抽屉只读展示。
      addChildParentName: parentGroupId
        ? (() => {
            const group = groups.find((g) => g.id === parentGroupId);
            return group ? getGroupName(group) : undefined;
          })()
        : undefined,
      editLevel2ParentName: (() => {
        const parent = groups.find(
        (g) => g.id === groups.find((x) => x.id === editLevel2GroupId)?.parentId
        );
        return parent ? getGroupName(parent) : undefined;
      })(),
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

function parseSerialNumbers(value?: string): string[] {
  return [...new Set((value ?? '').split(/[\s,;]+/).map((item) => item.trim()).filter(Boolean))];
}

function getBuiltInGroupName(group: GroupItem, t: (id: string) => string): string {
  const isDefaultGroup = group.id === DEFAULT_ROOT_GROUP_ID || group.id === UNASSIGNED_GROUP_ID;
  if (!isDefaultGroup) return '';
  return t('device.defaultGroupName');
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
