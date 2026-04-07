import { useCallback, useMemo, useState } from 'react';
import {
  App,
  Button,
  Drawer,
  Dropdown,
  Form,
  Input,
  Modal,
  Progress,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Tree,
  Typography,
  Radio,
  Divider,
} from 'antd';
import type { MenuProps } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  DeleteOutlined,
  EditOutlined,
  ExportOutlined,
  PlusOutlined,
  SyncOutlined,
  MenuOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import {
  useRules,
  useCreateRule,
  useUpdateRule,
  useDeleteRule,
  useToggleRule,
  useApplyRule,
  useBatchSortRules,
  useRuleTasks,
} from '@/hooks/api/useDeviceRules';
import type {
  DeviceRule,
  NameRule,
  RuleTask,
  CreateRuleRequest,
  UpdateRuleRequest,
} from '@/services/api/deviceRulesApi';
import { deviceRulesApi } from '@/services/api/deviceRulesApi';
import { useDomains } from '@/hooks/api/useTopology';

const { Text } = Typography;

// 内部使用的过滤条件结构
interface NameFilterItem {
  id: string;
  condition: 'contain' | 'notContain' | 'startWith' | 'endWith';
  value: string;
  andOr?: 'and' | 'or';
}

// 设备组选项（扁平化）
interface DeviceGroupOption {
  id: string;
  name: string;
  level: number;
}

// 过滤条件选项
const getFilterConditionOptions = (t: (key: string) => string) => [
  { label: t('filter.contain'), value: 'contain' },
  { label: t('filter.notContain'), value: 'notContain' },
  { label: t('filter.startWith'), value: 'startWith' },
  { label: t('filter.endWith'), value: 'endWith' },
];

// And/Or 选项
const getAndOrOptions = (t: (key: string) => string) => [
  { label: t('filter.and'), value: 'and' },
  { label: t('filter.or'), value: 'or' },
];

// 生成唯一ID
const generateId = () => Math.random().toString(36).substring(2, 9);

// 生成操作描述
function generateOperators(
  matchingMode: string,
  nameRuleList: NameRule[] | null,
  lacList: number[] | null,
  tacList: number[] | null,
  t: (key: string) => string
): string {
  if (matchingMode === 'and' && nameRuleList?.length) {
    // 将 NameRule 转换为 NameFilterItem 格式进行处理
    const filters: NameFilterItem[] = nameRuleList.map((rule, index) => ({
      id: String(index),
      condition: rule.type as NameFilterItem['condition'],
      value: rule.value,
      andOr: index > 0 ? 'and' : undefined,
    }));

    const orGroups: NameFilterItem[][] = [[]];

    filters.forEach((filter, index) => {
      if (filter.value && filter.value.trim() !== '') {
        if (index > 0 && filter.andOr === 'or') {
          orGroups.push([]);
        }
        orGroups[orGroups.length - 1].push(filter);
      }
    });

    const filteredGroups = orGroups.filter((g) => g.length > 0);
    if (filteredGroups.length === 0) return '';

    const verbMap: Record<string, string> = {
      contain: t('filter.contain'),
      notContain: t('filter.notContain'),
      startWith: t('filter.startWith'),
      endWith: t('filter.endWith'),
    };

    const groupParts = filteredGroups.map((group) => {
      const conditionParts = group.map((item) => `${verbMap[item.condition]} "${item.value}"`);
      const groupText = conditionParts.join(` ${t('filter.and')} `);
      return filteredGroups.length > 1 || group.length > 1 ? `(${groupText})` : groupText;
    });

    return groupParts.join(` ${t('filter.or')} `);
  } else if (matchingMode === 'or' && nameRuleList?.length) {
    // OR 模式
    const parts = nameRuleList.map((rule) => `"${rule.value}"`);
    return `${t('filter.or')}: ${parts.join(', ')}`;
  } else if (tacList?.length) {
    return `TAC: ${tacList.join(', ')}`;
  } else if (lacList?.length) {
    return `LAC: ${lacList.join(', ')}`;
  }
  return '';
}

// 迁移任务状态
type MigrationStatus = 'pending' | 'running' | 'completed' | 'failed';

// 迁移任务（用于UI显示）
interface MigrationTask {
  id: string;
  sn: string;
  deviceName: string;
  sourceGroup: string;
  targetGroup: string;
  status: MigrationStatus;
  progress: number;
  message?: string;
}

export default function DeviceRules() {
  const t = useT();
  const { modal, message } = App.useApp();
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [activeModalOpen, setActiveModalOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<DeviceRule | null>(null);
  const [form] = Form.useForm();
  const [nameFilters, setNameFilters] = useState<NameFilterItem[]>([
    { id: generateId(), condition: 'contain', value: '' },
  ]);
  const [selectedGroupIds, setSelectedGroupIds] = useState<string[]>([]);
  const [currentEditingRule, setCurrentEditingRule] = useState<DeviceRule | null>(null);
  const [migrationDrawerOpen, setMigrationDrawerOpen] = useState(false);
  const [migrationTasks, setMigrationTasks] = useState<MigrationTask[]>([]);

  // 匹配模式
  const matchingMode = Form.useWatch('matchingMode', form);

  // 获取规则列表
  const listParams = useMemo(() => ({
    page: 1,
    pageSize: 100,
    enabled: filterParams.enable === '1' ? true : filterParams.enable === '0' ? false : undefined,
    name: filterParams.operators as string | undefined,
  }), [filterParams]);

  const { data: rulesData, isLoading: rulesLoading, refetch: refetchRules } = useRules(listParams);
  const rules = rulesData?.items || [];

  // 获取设备分组
  const { data: domains } = useDomains();

  // 构建设备分组选项（显示层级结构，L1不可选）
  const deviceGroupOptions: { id: string; name: string; level: number; fullName: string; disabled?: boolean }[] = useMemo(() => {
    const options: { id: string; name: string; level: number; fullName: string; disabled?: boolean }[] = [];

    if (!domains) return options;

    // 遍历分组树，构建层级选项
    const buildOptions = (items: { id: string; name: string; level: number; children?: { id: string; name: string; level: number; children?: { id: string; name: string; level: number }[] }[] }[], parentPath: string = '') => {
      items.forEach((item) => {
        const fullName = parentPath ? `${parentPath}/${item.name}` : item.name;

        if (item.level === 1) {
          // L1 分组：添加但标记为不可选择
          options.push({
            id: item.id,
            name: item.name,
            level: item.level,
            fullName: item.name,
            disabled: true,
          });
          // 继续处理子分组
          if (item.children && item.children.length > 0) {
            buildOptions(item.children as { id: string; name: string; level: number; children?: { id: string; name: string; level: number }[] }[], item.name);
          }
        } else if (item.level === 2) {
          // L2 分组：可选择，显示完整路径
          options.push({
            id: item.id,
            name: item.name,
            level: item.level,
            fullName: fullName,
            disabled: false,
          });
        }
      });
    };

    buildOptions(domains as { id: string; name: string; level: number; children?: { id: string; name: string; level: number; children?: { id: string; name: string; level: number }[] }[] }[]);
    return options;
  }, [domains]);

  // Mutations
  const createMutation = useCreateRule();
  const updateMutation = useUpdateRule();
  const deleteMutation = useDeleteRule();
  const toggleMutation = useToggleRule();
  const applyMutation = useApplyRule();
  const batchSortMutation = useBatchSortRules();

  // 过滤字段配置
  const FILTER_FIELDS: FilterField[] = useMemo(
    () => [
      { name: 'operators', label: t('common.search'), type: 'input', placeholder: t('device.rules.searchPlaceholder') },
      {
        name: 'enable',
        label: t('table.status'),
        type: 'select',
        options: [
          { label: t('status.enabled'), value: '1' },
          { label: t('status.disabled'), value: '0' },
        ],
      },
    ],
    [t]
  );

  // 过滤规则列表
  const filteredRules = useMemo(() => {
    return rules.filter((r) => {
      if (filterParams.operators && !r.operators.toLowerCase().includes(String(filterParams.operators).toLowerCase())) {
        return false;
      }
      if (filterParams.enable === '1' && !r.enabled) {
        return false;
      }
      if (filterParams.enable === '0' && r.enabled) {
        return false;
      }
      return true;
    });
  }, [rules, filterParams]);

  // 切换启用状态
  const handleToggle = useCallback(
    (id: string, checked: boolean) => {
      toggleMutation.mutate({ id, enabled: checked }, {
        onSuccess: () => {
          void message.success(checked ? t('status.enabled') : t('status.disabled'));
        },
        onError: () => {
          void message.error(t('common.operationFailed'));
        },
      });
    },
    [toggleMutation, message, t]
  );

  // 删除规则
  const handleDelete = useCallback(
    (id: string) => {
      modal.confirm({
        title: t('common.confirmDelete'),
        content: t('device.rules.deleteConfirm'),
        okText: t('common.confirm'),
        cancelText: t('common.cancel'),
        okType: 'danger',
        onOk: () => {
          deleteMutation.mutate(id, {
            onSuccess: () => {
              void message.success(t('common.deleteSuccess'));
            },
            onError: () => {
              void message.error(t('common.deleteFailed'));
            },
          });
        },
      });
    },
    [deleteMutation, modal, message, t]
  );

  // 将 NameFilterItem 转换为 NameRule
  const convertToNameRules = (filters: NameFilterItem[]): NameRule[] => {
    return filters
      .filter((f) => f.value?.trim())
      .map((f) => ({
        type: f.condition as NameRule['type'],
        operator: f.andOr || 'and',
        value: f.value,
      }));
  };

  // 打开编辑弹窗
  const handleEdit = useCallback(
    (rule: DeviceRule) => {
      setEditingRule(rule);
      form.setFieldsValue({
        name: rule.name,
        targetGroupId: rule.targetGroupId,
        enable: rule.enabled ? '1' : '0',
        matchingMode: rule.matchingMode,
        lacList: rule.lacList?.join(', '),
        tacList: rule.tacList?.join(', '),
      });

      // 将 NameRule 转换为 NameFilterItem
      if (rule.nameRuleList?.length) {
        setNameFilters(
          rule.nameRuleList.map((nr, index) => ({
            id: String(index),
            condition: nr.type as NameFilterItem['condition'],
            value: nr.value,
            andOr: index > 0 ? (nr.operator as 'and' | 'or') : undefined,
          }))
        );
      } else {
        setNameFilters([{ id: generateId(), condition: 'contain', value: '' }]);
      }
      setEditModalOpen(true);
    },
    [form]
  );

  // 打开新增弹窗
  const handleCreate = useCallback(async () => {
    setEditingRule(null);
    form.resetFields();

    // 获取下一个优先级
    try {
      const nextPriority = await deviceRulesApi.getNextPriority();
      form.setFieldsValue({
        name: '',
        priority: nextPriority,
        targetGroupId: deviceGroupOptions[0]?.id,
        enable: '0',
        matchingMode: 'and',
        lacList: '',
        tacList: '',
      });
    } catch {
      form.setFieldsValue({
        name: '',
        priority: 1,
        targetGroupId: deviceGroupOptions[0]?.id,
        enable: '0',
        matchingMode: 'and',
        lacList: '',
        tacList: '',
      });
    }

    setNameFilters([{ id: generateId(), condition: 'contain', value: '' }]);
    setEditModalOpen(true);
  }, [form, deviceGroupOptions]);

  // 保存规则
  const handleSave = useCallback(async () => {
    try {
      const values = await form.validateFields();

      // 验证名称过滤器
      if (matchingMode === 'and' || matchingMode === 'or') {
        const validFilters = nameFilters.filter((f) => f.value?.trim());
        if (values.enable === '1' && validFilters.length === 0) {
          void message.error(t('device.rules.atLeastOneFilter'));
          return;
        }
      }

      const nameRuleList = matchingMode === 'and' || matchingMode === 'or'
        ? convertToNameRules(nameFilters)
        : undefined;

      const lacList = values.lacList
        ? values.lacList.split(',').map((s: string) => parseInt(s.trim())).filter((n: number) => !isNaN(n))
        : undefined;

      const tacList = values.tacList
        ? values.tacList.split(',').map((s: string) => parseInt(s.trim())).filter((n: number) => !isNaN(n))
        : undefined;

      if (editingRule) {
        // 更新
        const req: UpdateRuleRequest = {
          name: values.name,
          targetGroupId: values.targetGroupId,
          enabled: values.enable === '1',
          matchingMode: matchingMode,
          nameRuleList,
          lacList,
          tacList,
        };
        updateMutation.mutate({ id: editingRule.id, req }, {
          onSuccess: () => {
            void message.success(t('status.success'));
            setEditModalOpen(false);
          },
          onError: () => {
            void message.error(t('common.operationFailed'));
          },
        });
      } else {
        // 创建
        const req: CreateRuleRequest = {
          name: values.name,
          priority: values.priority,
          targetGroupId: values.targetGroupId,
          enabled: values.enable === '1',
          matchingMode: matchingMode,
          nameRuleList,
          lacList,
          tacList,
        };
        createMutation.mutate(req, {
          onSuccess: () => {
            void message.success(t('status.success'));
            setEditModalOpen(false);
          },
          onError: () => {
            void message.error(t('common.operationFailed'));
          },
        });
      }
    } catch {
      // validation error
    }
  }, [editingRule, form, nameFilters, matchingMode, createMutation, updateMutation, message, t]);

  // 打开应用规则弹窗
  const handleActive = useCallback((rule: DeviceRule) => {
    setCurrentEditingRule(rule);
    setSelectedGroupIds([]);
    setActiveModalOpen(true);
  }, []);

  // 应用规则
  const handleExecute = useCallback(() => {
    if (!currentEditingRule) return;

    if (!currentEditingRule.enabled) {
      void message.warning(t('device.rules.mustEnableFirst'));
      return;
    }

    applyMutation.mutate({ id: currentEditingRule.id }, {
      onSuccess: (task) => {
        void message.success(t('common.commandSent'));
        setActiveModalOpen(false);

        // 创建模拟迁移任务用于显示
        const mockTask: MigrationTask = {
          id: task.id,
          sn: '',
          deviceName: t('device.rules.taskInProgress'),
          sourceGroup: '',
          targetGroup: currentEditingRule.targetGroupName || '',
          status: task.status as MigrationStatus,
          progress: 0,
        };
        setMigrationTasks([mockTask]);
        setMigrationDrawerOpen(true);

        // 轮询任务状态
        const pollInterval = setInterval(async () => {
          try {
            const updatedTask = await deviceRulesApi.getTask(currentEditingRule.id, task.id);
            setMigrationTasks((prev) =>
              prev.map((t) =>
                t.id === task.id
                  ? {
                      ...t,
                      status: updatedTask.status as MigrationStatus,
                      progress:
                        updatedTask.totalDevices > 0
                          ? Math.round(
                              ((updatedTask.matchedCount + updatedTask.failedCount) /
                                updatedTask.totalDevices) *
                                100
                            )
                          : 0,
                      message:
                        updatedTask.status === 'completed'
                          ? t('device.rules.taskCompleted', {
                              matched: updatedTask.matchedCount,
                              failed: updatedTask.failedCount,
                            })
                          : updatedTask.errorMessage || '',
                    }
                  : t
              )
            );

            if (updatedTask.status === 'completed' || updatedTask.status === 'failed') {
              clearInterval(pollInterval);
            }
          } catch {
            clearInterval(pollInterval);
          }
        }, 2000);
      },
      onError: () => {
        void message.error(t('common.operationFailed'));
      },
    });
  }, [currentEditingRule, applyMutation, message, t]);

  // 添加过滤条件
  const handleAddFilter = useCallback(() => {
    if (nameFilters.length >= 10) {
      void message.warning(t('device.rules.maxConditions', { max: 10 }));
      return;
    }
    const hasOr = nameFilters.some((f, index) => index > 0 && f.andOr === 'or');
    setNameFilters((prev) => [
      ...prev,
      {
        id: generateId(),
        condition: 'contain',
        value: '',
        andOr: hasOr ? 'or' : 'and',
      },
    ]);
  }, [nameFilters, message, t]);

  // 删除过滤条件
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

  // 更新过滤条件
  const handleUpdateFilter = useCallback((id: string, field: keyof NameFilterItem, value: string) => {
    setNameFilters((prev) => prev.map((f) => (f.id === id ? { ...f, [field]: value } : f)));
  }, []);

  // 切换匹配模式时重置
  const handleMatchingModeChange = useCallback(() => {
    setNameFilters([{ id: generateId(), condition: 'contain', value: '' }]);
    form.setFieldsValue({ lacList: '', tacList: '' });
  }, [form]);

  // 上移
  const handleMoveUp = useCallback(
    (index: number) => {
      if (index === 0) return;
      const items = [
        { id: filteredRules[index - 1].id, priority: filteredRules[index].priority },
        { id: filteredRules[index].id, priority: filteredRules[index - 1].priority },
      ];
      batchSortMutation.mutate(items);
    },
    [filteredRules, batchSortMutation]
  );

  // 下移
  const handleMoveDown = useCallback(
    (index: number) => {
      if (index >= filteredRules.length - 1) return;
      const items = [
        { id: filteredRules[index].id, priority: filteredRules[index + 1].priority },
        { id: filteredRules[index + 1].id, priority: filteredRules[index].priority },
      ];
      batchSortMutation.mutate(items);
    },
    [filteredRules, batchSortMutation]
  );

  // 表格列定义
  const columns: DataTableColumn<DeviceRule>[] = useMemo(
    () => [
      {
        key: 'sort',
        title: t('device.rules.priority'),
        dataIndex: 'priority',
        width: 70,
        render: (_val, _record, index) => {
          const items: MenuProps['items'] = [
            {
              key: 'up',
              label: t('device.rules.moveUp'),
              icon: <ArrowUpOutlined />,
              disabled: index === 0,
              onClick: () => handleMoveUp(index),
            },
            {
              key: 'down',
              label: t('device.rules.moveDown'),
              icon: <ArrowDownOutlined />,
              disabled: index === filteredRules.length - 1,
              onClick: () => handleMoveDown(index),
            },
          ];
          return (
            <Dropdown menu={{ items }} trigger={['click']}>
              <Button
                type="text"
                size="small"
                icon={<MenuOutlined style={{ color: 'var(--color-text-quaternary)', cursor: 'grab' }} />}
                style={{ padding: '0 4px' }}
              />
            </Dropdown>
          );
        },
      },
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 200,
        fixed: 'left',
        render: (_val, record) => (
          <Space size={4}>
            {record.enabled && (
              <Button
                type="link"
                size="small"
                icon={<SyncOutlined />}
                onClick={() => handleActive(record)}
              >
                {t('device.rules.apply')}
              </Button>
            )}
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
            >
              {t('common.edit')}
            </Button>
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => handleDelete(record.id)}
            >
              {t('common.delete')}
            </Button>
          </Space>
        ),
      },
      {
        key: 'enabled',
        title: t('common.enable'),
        dataIndex: 'enabled',
        width: 80,
        render: (_val, record) => (
          <Switch
            checked={record.enabled}
            size="small"
            onChange={(checked) => handleToggle(record.id, checked)}
          />
        ),
      },
      {
        key: 'name',
        title: t('device.rules.ruleName'),
        dataIndex: 'name',
        width: 150,
        ellipsis: true,
      },
      {
        key: 'operators',
        title: t('device.rules.rule'),
        dataIndex: 'operators',
        ellipsis: true,
        render: (val) => (
          <Text style={{ color: 'var(--color-text-secondary)' }}>{String(val)}</Text>
        ),
      },
      {
        key: 'targetGroupName',
        title: t('device.rules.targetGroup'),
        dataIndex: 'targetGroupName',
        width: 150,
      },
      {
        key: 'createdAt',
        title: t('table.createTime'),
        dataIndex: 'createdAt',
        width: 160,
        render: (val) => (val ? new Date(val).toLocaleString('zh-CN') : ''),
      },
    ],
    [handleToggle, handleEdit, handleDelete, handleActive, handleMoveUp, handleMoveDown, filteredRules.length, t]
  );

  // 预览条件描述
  const previewText = useMemo(() => {
    if (matchingMode === 'and' || matchingMode === 'or') {
      return generateOperators(matchingMode, convertToNameRules(nameFilters), null, null, t);
    }
    return '';
  }, [matchingMode, nameFilters, t]);

  // 迁移任务表格列定义
  const migrationColumns: ColumnsType<MigrationTask> = useMemo(
    () => [
      {
        title: 'SN',
        dataIndex: 'sn',
        key: 'sn',
        width: 120,
        ellipsis: true,
        render: (sn: string) => (
          <span style={{ fontFamily: 'monospace', fontSize: 11 }}>{sn}</span>
        ),
      },
      {
        title: t('alarm.deviceName'),
        dataIndex: 'deviceName',
        key: 'deviceName',
        ellipsis: true,
        width: 120,
      },
      {
        title: t('device.rules.sourceGroup'),
        dataIndex: 'sourceGroup',
        key: 'sourceGroup',
        width: 100,
        ellipsis: true,
      },
      {
        title: t('device.rules.targetGroup'),
        dataIndex: 'targetGroup',
        key: 'targetGroup',
        width: 100,
        ellipsis: true,
      },
      {
        title: t('table.status'),
        dataIndex: 'status',
        key: 'status',
        width: 80,
        render: (status: MigrationStatus) => {
          const statusConfig: Record<MigrationStatus, { color: string; text: string }> = {
            pending: { color: 'default', text: t('status.pending') },
            running: { color: 'processing', text: t('task.status.running') },
            completed: { color: 'success', text: t('task.status.completed') },
            failed: { color: 'error', text: t('task.status.failed') },
          };
          const cfg = statusConfig[status];
          return (
            <Tag color={cfg.color} style={{ fontSize: 11, padding: '0 4px', margin: 0 }}>
              {cfg.text}
            </Tag>
          );
        },
      },
      {
        title: t('task.progress'),
        dataIndex: 'progress',
        key: 'progress',
        width: 120,
        render: (progress: number, record) => (
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <Progress
              percent={Math.round(progress)}
              size="small"
              status={record.status === 'failed' ? 'exception' : record.status === 'completed' ? 'success' : 'active'}
              showInfo={false}
              style={{ flex: 1, minWidth: 60 }}
            />
            <span style={{ fontSize: 11, color: 'var(--color-neutral-600)', whiteSpace: 'nowrap' }}>
              {Math.round(progress)}%
            </span>
          </div>
        ),
      },
    ],
    [t]
  );

  // 导出迁移结果
  const handleExportMigration = useCallback(() => {
    if (migrationTasks.length === 0) return;

    const headers = [
      'SN',
      t('alarm.deviceName'),
      t('device.rules.sourceGroup'),
      t('device.rules.targetGroup'),
      t('table.status'),
      t('task.progress'),
      t('task.message'),
    ];
    const rows = migrationTasks.map((task) => [
      task.sn,
      task.deviceName,
      task.sourceGroup,
      task.targetGroup,
      task.status === 'completed'
        ? t('task.status.completed')
        : task.status === 'failed'
          ? t('task.status.failed')
          : task.status === 'running'
            ? t('task.status.running')
            : t('status.pending'),
      `${task.progress}%`,
      task.message || '',
    ]);

    const csvContent = [headers, ...rows].map((row) => row.join(',')).join('\n');
    const blob = new Blob(['\ufeff' + csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `migration_result_${new Date().toISOString().slice(0, 10)}.csv`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);

    void message.success(t('common.exportSuccess'));
  }, [migrationTasks, message, t]);

  return (
    <>
      <ListPageLayout
        title={t('device.rules.title')}
        subtitle={`${t('table.total')} ${rules.length}`}
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => void handleCreate()}>
            {t('common.add')}
          </Button>
        }
      >
        <FilterBar
          filterId="device-rules"
          fields={FILTER_FIELDS}
          onSearch={(v) => setFilterParams(v)}
          onReset={() => setFilterParams({})}
          collapsedRows={1}
        />

        <DataTable<DeviceRule>
          tableId="device-rules-table"
          columns={columns}
          dataSource={filteredRules}
          loading={rulesLoading}
          rowKey="id"
          total={filteredRules.length}
          pagination={false}
          showPagination={false}
          defaultDensity="compact"
        />
      </ListPageLayout>

      {/* 添加/编辑规则抽屉 */}
      <Drawer
        title={editingRule ? t('common.edit') : t('common.add')}
        open={editModalOpen}
        onClose={() => setEditModalOpen(false)}
        width={520}
        destroyOnClose
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button onClick={() => setEditModalOpen(false)}>{t('common.cancel')}</Button>
            <Button
              type="primary"
              loading={createMutation.isPending || updateMutation.isPending}
              onClick={() => void handleSave()}
            >
              {t('common.confirm')}
            </Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          {/* 基本设置 */}
          <div style={{ marginBottom: 8, fontWeight: 500, color: 'var(--color-text)' }}>
            {t('device.rules.basicSettings')}
          </div>

          <Form.Item
            name="name"
            label={t('device.rules.ruleName')}
            rules={[{ required: true, message: t('device.rules.inputRuleName') }]}
          >
            <Input placeholder={t('common.placeholder')} maxLength={100} />
          </Form.Item>

          <Form.Item
            name="targetGroupId"
            label={t('device.rules.targetGroup')}
            rules={[{ required: true, message: t('device.rules.selectTargetGroup') }]}
          >
            <Select
              placeholder={t('common.pleaseSelect')}
              showSearch
              optionFilterProp="label"
            >
              {deviceGroupOptions.map((g) => (
                <Select.Option key={g.id} value={g.id} disabled={g.disabled}>
                  <span style={{ color: g.disabled ? 'var(--color-text-disabled)' : 'inherit' }}>
                    {g.fullName}
                  </span>
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="enable"
            label={t('device.rules.enableStatus')}
            valuePropName="checked"
            getValueProps={(v) => ({ checked: v === '1' })}
            getValueFromEvent={(checked: boolean) => (checked ? '1' : '0')}
          >
            <Switch checkedChildren={t('common.enable')} unCheckedChildren={t('common.disable')} />
          </Form.Item>

          <Divider style={{ margin: '16px 0' }} />

          {/* 匹配规则 */}
          <div style={{ marginBottom: 8, fontWeight: 500, color: 'var(--color-text)' }}>
            {t('device.rules.matchingRule')}
          </div>

          <Form.Item name="matchingMode" label={t('device.rules.matchingMode')} rules={[{ required: true }]}>
            <Radio.Group onChange={handleMatchingModeChange}>
              <Radio value="and">{t('device.rules.nameMatch')}</Radio>
              <Radio value="lac">LAC</Radio>
              <Radio value="tac">TAC</Radio>
            </Radio.Group>
          </Form.Item>

          {/* 设备名称过滤条件 */}
          {(matchingMode === 'and' || matchingMode === 'or') && (
            <>
              <Form.Item
                label={
                  <span>
                    {t('device.rules.filterCondition')}{' '}
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {t('device.rules.conditionLimit', { max: 10 })}
                    </Text>
                  </span>
                }
              >
                <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                  {nameFilters.map((filter, index) => (
                    <div key={filter.id} style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                      {index === 0 ? (
                        <>
                          <Select
                            value={filter.condition}
                            style={{ width: 120 }}
                            options={getFilterConditionOptions(t)}
                            onChange={(v) => handleUpdateFilter(filter.id, 'condition', v)}
                          />
                          <Input
                            value={filter.value}
                            style={{ flex: 1 }}
                            maxLength={64}
                            placeholder={t('common.placeholder')}
                            onChange={(e) => handleUpdateFilter(filter.id, 'value', e.target.value)}
                          />
                        </>
                      ) : (
                        <>
                          <Select
                            value={filter.andOr || 'and'}
                            style={{ width: 70 }}
                            options={getAndOrOptions(t)}
                            onChange={(v) => handleUpdateFilter(filter.id, 'andOr', v)}
                          />
                          <Select
                            value={filter.condition}
                            style={{ width: 120 }}
                            options={getFilterConditionOptions(t)}
                            onChange={(v) => handleUpdateFilter(filter.id, 'condition', v)}
                          />
                          <Input
                            value={filter.value}
                            style={{ flex: 1 }}
                            maxLength={64}
                            placeholder={t('common.placeholder')}
                            onChange={(e) => handleUpdateFilter(filter.id, 'value', e.target.value)}
                          />
                          <Button
                            type="text"
                            size="small"
                            icon={<CloseCircleOutlined />}
                            onClick={() => handleRemoveFilter(filter.id)}
                            style={{ color: 'var(--color-text-quaternary)' }}
                          />
                        </>
                      )}
                    </div>
                  ))}
                </div>
                {nameFilters.length < 10 && (
                  <Button type="dashed" icon={<PlusOutlined />} onClick={handleAddFilter} style={{ marginTop: 8 }}>
                    {t('device.rules.addCondition')}
                  </Button>
                )}
              </Form.Item>

              {/* 预览条件描述 */}
              {previewText && (
                <div
                  style={{
                    color: 'var(--color-text-tertiary)',
                    fontSize: 12,
                    marginBottom: 16,
                    padding: '8px 12px',
                    background: 'var(--color-fill-quaternary)',
                    borderRadius: 4,
                    wordBreak: 'break-all',
                  }}
                >
                  {previewText}
                </div>
              )}
            </>
          )}

          {/* TAC 输入 */}
          {matchingMode === 'tac' && (
            <Form.Item
              name="tacList"
              label="TAC"
              rules={[{ required: true, message: t('device.rules.inputRange', { type: 'TAC' }) }]}
              extra={
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {t('device.rules.formatRange', { range: '0-16777215' })}
                </Text>
              }
            >
              <Input placeholder="eg: 1,2,3" maxLength={200} />
            </Form.Item>
          )}

          {/* LAC 输入 */}
          {matchingMode === 'lac' && (
            <Form.Item
              name="lacList"
              label="LAC"
              rules={[{ required: true, message: t('device.rules.inputRange', { type: 'LAC' }) }]}
              extra={
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {t('device.rules.formatRange', { range: '0-65535' })}
                </Text>
              }
            >
              <Input placeholder="eg: 1,2,3" maxLength={200} />
            </Form.Item>
          )}
        </Form>
      </Drawer>

      {/* 应用规则弹窗 */}
      <Modal
        title={t('device.rules.applyRule')}
        open={activeModalOpen}
        onOk={handleExecute}
        onCancel={() => setActiveModalOpen(false)}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        width={480}
        confirmLoading={applyMutation.isPending}
      >
        <div style={{ marginBottom: 12, color: 'var(--color-text-secondary)' }}>
          {t('device.rules.applyConfirm', { name: currentEditingRule?.name })}
        </div>
        <div
          style={{
            padding: '12px 16px',
            background: 'var(--color-fill-quaternary)',
            borderRadius: 6,
          }}
        >
          <div style={{ marginBottom: 8, fontWeight: 500 }}>{t('device.rules.matchingRule')}:</div>
          <div style={{ color: 'var(--color-text-secondary)' }}>{currentEditingRule?.operators}</div>
        </div>
      </Modal>

      {/* 迁移结果抽屉 */}
      <Drawer
        title={t('device.rules.migrationResult')}
        placement="right"
        width={640}
        open={migrationDrawerOpen}
        onClose={() => setMigrationDrawerOpen(false)}
        styles={{
          body: { padding: 0, display: 'flex', flexDirection: 'column', height: '100%' },
        }}
      >
        {/* 任务统计 */}
        <div
          style={{
            padding: '12px 16px',
            borderBottom: '1px solid var(--color-border)',
            flexShrink: 0,
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <Space size={16}>
            <span>
              {t('task.total')}: {migrationTasks.length}
            </span>
            <span style={{ color: 'var(--color-primary-600)' }}>
              {t('task.status.running')}: {migrationTasks.filter((item) => item.status === 'running').length}
            </span>
            <span style={{ color: '#52c41a' }}>
              {t('task.status.completed')}: {migrationTasks.filter((item) => item.status === 'completed').length}
            </span>
            <span style={{ color: '#ff4d4f' }}>
              {t('task.status.failed')}: {migrationTasks.filter((item) => item.status === 'failed').length}
            </span>
          </Space>
          <Button
            size="small"
            icon={<ExportOutlined />}
            onClick={handleExportMigration}
            disabled={migrationTasks.length === 0}
          >
            {t('common.export')}
          </Button>
        </div>

        {/* 任务列表 */}
        <div style={{ flex: 1, overflow: 'hidden', padding: 8 }}>
          <Table<MigrationTask>
            dataSource={migrationTasks}
            columns={migrationColumns}
            rowKey="id"
            size="small"
            pagination={false}
            scroll={{ y: 'calc(100vh - 180px)' }}
            locale={{ emptyText: t('common.noData') }}
            style={{ fontSize: 12 }}
          />
        </div>
      </Drawer>
    </>
  );
}
