import React, { useCallback, useMemo, useState } from 'react';
import {
  Button,
  Divider,
  Drawer,
  Dropdown,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Switch,
  Tree,
  Typography,
  message,
  Radio,
} from 'antd';
import type { MenuProps } from 'antd';
import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  SyncOutlined,
  CloseCircleOutlined,
  MenuOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';

const { Text } = Typography;

// 名称过滤条件
interface NameFilterItem {
  id: string;
  condition: 'contain' | 'notContain' | 'startWith' | 'endWith';
  value: string;
  andOr?: 'and' | 'or';
}

// 设备归属规则
interface DeviceGroupRule extends Record<string, unknown> {
  id: string;
  order: number;
  moveToGroupId: string;
  moveToGroupName: string;
  enable: '0' | '1';
  matchingMode: 'deviceName' | 'lac' | 'tac';
  nameRuleList: NameFilterItem[];
  tacRag: string;
  operators: string;
  createTime: string;
}

// 设备组选项
interface DeviceGroup {
  id: string;
  groupName: string;
}

// 设备组树节点
interface DeviceGroupTreeNode {
  id: string;
  groupName: string;
  children?: DeviceGroupTreeNode[];
}

// Mock 设备组数据
const MOCK_DEVICE_GROUPS: DeviceGroup[] = [
  { id: '1', groupName: 'Default Group' },
  { id: '2', groupName: 'Beijing Region' },
  { id: '3', groupName: 'Shanghai Region' },
  { id: '4', groupName: 'Guangzhou Region' },
  { id: '5', groupName: 'Test Group' },
];

// Mock 设备组树
const MOCK_GROUP_TREE: DeviceGroupTreeNode[] = [
  {
    id: 'root',
    groupName: 'All Devices',
    children: [
      { id: '1', groupName: 'Default Group' },
      { id: '2', groupName: 'Beijing Region' },
      { id: '3', groupName: 'Shanghai Region' },
      { id: '4', groupName: 'Guangzhou Region' },
      { id: '5', groupName: 'Test Group' },
    ],
  },
];

// 过滤条件选项 - 使用函数以便获取 t
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
function generateOperators(rule: Partial<DeviceGroupRule>, t: (key: string) => string): string {
  if (rule.matchingMode === 'deviceName' && rule.nameRuleList?.length) {
    const orGroups: NameFilterItem[][] = [[]];

    rule.nameRuleList.forEach((filter, index) => {
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

    let result = groupParts.join(` ${t('filter.or')} `);
    return result;
  } else if (rule.matchingMode === 'tac') {
    return `TAC: ${rule.tacRag || ''}`;
  } else if (rule.matchingMode === 'lac') {
    return `LAC: ${rule.tacRag || ''}`;
  }
  return '';
}

// Mock 规则数据
const MOCK_RULES: DeviceGroupRule[] = [
  {
    id: '1',
    order: 1,
    moveToGroupId: '2',
    moveToGroupName: 'Beijing Region',
    enable: '1',
    matchingMode: 'deviceName',
    nameRuleList: [{ id: '1', condition: 'contain', value: 'BJ-' }],
    tacRag: '',
    operators: '包含 "BJ-"',
    createTime: '2024-01-10 09:00:00',
  },
  {
    id: '2',
    order: 2,
    moveToGroupId: '3',
    moveToGroupName: 'Shanghai Region',
    enable: '1',
    matchingMode: 'deviceName',
    nameRuleList: [
      { id: '2', condition: 'contain', value: 'SH-' },
      { id: '3', condition: 'contain', value: '-5G-', andOr: 'and' },
    ],
    tacRag: '',
    operators: '必须 (包含 "SH-" 且 包含 "-5G-")',
    createTime: '2024-01-12 10:00:00',
  },
  {
    id: '3',
    order: 3,
    moveToGroupId: '4',
    moveToGroupName: 'Guangzhou Region',
    enable: '0',
    matchingMode: 'tac',
    nameRuleList: [],
    tacRag: '1-100,200-300',
    operators: 'TAC: 1-100,200-300',
    createTime: '2024-01-15 11:00:00',
  },
  {
    id: '4',
    order: 4,
    moveToGroupId: '5',
    moveToGroupName: 'Test Group',
    enable: '1',
    matchingMode: 'deviceName',
    nameRuleList: [{ id: '4', condition: 'startWith', value: 'TEST' }],
    tacRag: '',
    operators: '以...开始 "TEST"',
    createTime: '2024-01-20 08:00:00',
  },
];

// 设备类型
const DEVICE_TYPE = 'ENB';
const SUPPORT_GSM = true;

export default function DeviceRules() {
  const t = useT();
  const [rules, setRules] = useState<DeviceGroupRule[]>(MOCK_RULES);
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [activeModalOpen, setActiveModalOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<DeviceGroupRule | null>(null);
  const [form] = Form.useForm();
  const [nameFilters, setNameFilters] = useState<NameFilterItem[]>([
    { id: generateId(), condition: 'contain', value: '' },
  ]);
  const [selectedGroupIds, setSelectedGroupIds] = useState<string[]>([]);

  // 匹配模式
  const matchingMode = Form.useWatch('matchingMode', form);

  // 过滤字段配置
  const FILTER_FIELDS: FilterField[] = useMemo(
    () => [
      { name: 'operators', label: t('device.rules.operators'), type: 'input' },
      {
        name: 'enable',
        label: t('table.status'),
        type: 'select',
        options: [
          { label: t('status.enabled'), value: '1' },
          { label: t('status.disabled'), value: '0' },
        ],
      },
      {
        name: 'moveToGroupId',
        label: t('device.rules.targetGroup'),
        type: 'select',
        options: MOCK_DEVICE_GROUPS.map((g) => ({ label: g.groupName, value: g.id })),
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
      if (filterParams.enable && r.enable !== filterParams.enable) {
        return false;
      }
      if (filterParams.moveToGroupId && r.moveToGroupId !== filterParams.moveToGroupId) {
        return false;
      }
      return true;
    });
  }, [rules, filterParams]);

  // 切换启用状态
  const handleToggle = useCallback(
    (id: string, checked: boolean) => {
      setRules((prev) => prev.map((r) => (r.id === id ? { ...r, enable: checked ? '1' : '0' } : r)));
      void message.success(checked ? t('status.enabled') : t('status.disabled'));
    },
    [t]
  );

  // 删除规则
  const handleDelete = useCallback(
    (id: string) => {
      Modal.confirm({
        title: t('common.confirmDelete'),
        content: t('device.rules.deleteConfirm'),
        okText: t('common.confirm'),
        cancelText: t('common.cancel'),
        okType: 'danger',
        onOk: () => {
          setRules((prev) => prev.filter((r) => r.id !== id));
          void message.success(t('common.deleteSuccess'));
        },
      });
    },
    [t]
  );

  // 打开编辑弹窗
  const handleEdit = useCallback(
    (rule: DeviceGroupRule) => {
      setEditingRule(rule);
      form.setFieldsValue({
        moveToGroupId: rule.moveToGroupId,
        enable: rule.enable,
        matchingMode: rule.matchingMode,
        tacRag: rule.tacRag,
      });
      if (rule.nameRuleList?.length > 0) {
        setNameFilters(rule.nameRuleList.map((item) => ({ ...item, id: item.id || generateId() })));
      } else {
        setNameFilters([{ id: generateId(), condition: 'contain', value: '' }]);
      }
      setEditModalOpen(true);
    },
    [form]
  );

  // 打开新增弹窗
  const handleCreate = useCallback(() => {
    setEditingRule(null);
    form.resetFields();
    form.setFieldsValue({
      moveToGroupId: MOCK_DEVICE_GROUPS[0]?.id,
      enable: '0',
      matchingMode: 'deviceName',
      tacRag: '',
    });
    setNameFilters([{ id: generateId(), condition: 'contain', value: '' }]);
    setEditModalOpen(true);
  }, [form]);

  // 保存规则
  const handleSave = useCallback(async () => {
    try {
      const values = (await form.validateFields()) as {
        moveToGroupId: string;
        enable: '0' | '1';
        matchingMode: 'deviceName' | 'lac' | 'tac';
        tacRag: string;
      };

      // 验证名称过滤器
      if (values.matchingMode === 'deviceName') {
        const validFilters = nameFilters.filter((f) => f.value?.trim());
        if (values.enable === '1' && validFilters.length === 0) {
          void message.error(t('device.rules.atLeastOneFilter'));
          return;
        }
      }

      // 验证 TAC/LAC
      if ((values.matchingMode === 'tac' || values.matchingMode === 'lac') && !values.tacRag?.trim()) {
        void message.error(t('device.rules.inputRange', { type: values.matchingMode === 'tac' ? 'TAC' : 'LAC' }));
        return;
      }

      const groupName = MOCK_DEVICE_GROUPS.find((g) => g.id === values.moveToGroupId)?.groupName || '';
      const ruleData: Partial<DeviceGroupRule> = {
        moveToGroupId: values.moveToGroupId,
        moveToGroupName: groupName,
        enable: values.enable,
        matchingMode: values.matchingMode,
        nameRuleList: values.matchingMode === 'deviceName' ? nameFilters : [],
        tacRag: values.matchingMode !== 'deviceName' ? values.tacRag : '',
      };
      ruleData.operators = generateOperators(ruleData);

      if (editingRule) {
        setRules((prev) => prev.map((r) => (r.id === editingRule.id ? { ...r, ...ruleData } : r)));
        void message.success(t('status.success'));
      } else {
        const newRule: DeviceGroupRule = {
          id: generateId(),
          order: rules.length + 1,
          ...ruleData,
          createTime: new Date().toLocaleString('zh-CN'),
        } as DeviceGroupRule;
        setRules((prev) => [...prev, newRule]);
        void message.success(t('status.success'));
      }
      setEditModalOpen(false);
    } catch {
      // validation error
    }
  }, [editingRule, form, nameFilters, rules.length, t]);

  // 打开应用规则弹窗
  const handleActive = useCallback((rule: DeviceGroupRule) => {
    setActiveModalOpen(true);
  }, []);

  // 应用规则
  const handleExecute = useCallback(() => {
    if (selectedGroupIds.length === 0) {
      void message.warning(t('device.rules.selectAtLeastOne'));
      return;
    }
    void message.success(t('device.rules.appliedTo', { count: selectedGroupIds.length }));
    setActiveModalOpen(false);
  }, [selectedGroupIds, t]);

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
  }, [nameFilters]);

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
    form.setFieldsValue({ tacRag: '' });
  }, [form]);

  // 上移
  const handleMoveUp = useCallback((index: number) => {
    if (index === 0) return;
    setRules((prev) => {
      const newRules = [...prev];
      [newRules[index - 1], newRules[index]] = [newRules[index], newRules[index - 1]];
      return newRules.map((r, i) => ({ ...r, order: i + 1 }));
    });
  }, []);

  // 下移
  const handleMoveDown = useCallback((index: number) => {
    setRules((prev) => {
      if (index >= prev.length - 1) return prev;
      const newRules = [...prev];
      [newRules[index], newRules[index + 1]] = [newRules[index + 1], newRules[index]];
      return newRules.map((r, i) => ({ ...r, order: i + 1 }));
    });
  }, []);

  // 表格列定义
  const columns: DataTableColumn<DeviceGroupRule>[] = useMemo(
    () => [
      {
        key: 'sort',
        title: '',
        dataIndex: 'id',
        width: 50,
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
            {record.enable === '1' && (
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
        key: 'enable',
        title: 'Enable',
        dataIndex: 'enable',
        width: 80,
        render: (_val, record) => (
          <Switch
            checked={record.enable === '1'}
            size="small"
            onChange={(checked) => handleToggle(record.id, checked)}
          />
        ),
      },
      {
        key: 'operators',
        title: t('device.rules.operation'),
        dataIndex: 'operators',
        ellipsis: true,
        render: (val) => (
          <Text style={{ color: 'var(--color-text-secondary)' }}>{String(val)}</Text>
        ),
      },
      {
        key: 'moveToGroupName',
        title: t('device.rules.targetGroup'),
        dataIndex: 'moveToGroupName',
        width: 150,
      },
      {
        key: 'createTime',
        title: t('table.createTime'),
        dataIndex: 'createTime',
        width: 160,
      },
    ],
    [handleToggle, handleEdit, handleDelete, handleActive, handleMoveUp, handleMoveDown, filteredRules.length, t]
  );

  // 预览条件描述
  const previewText = useMemo(() => {
    if (matchingMode === 'deviceName') {
      return generateOperators({ matchingMode: 'deviceName', nameRuleList: nameFilters });
    }
    return '';
  }, [matchingMode, nameFilters]);

  return (
    <>
      <ListPageLayout
        title="设备归属设备组规则"
        subtitle={`${t('table.total')} ${rules.length}`}
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
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

        <DataTable<DeviceGroupRule>
          tableId="device-rules-table"
          columns={columns}
          dataSource={filteredRules}
          loading={false}
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
            <Button type="primary" onClick={() => void handleSave()}>
              {t('common.confirm')}
            </Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          {/* 基本设置 */}
          <div style={{ marginBottom: 8, fontWeight: 500, color: 'var(--color-text)' }}>{t('device.rules.basicSettings')}</div>
          <Form.Item name="moveToGroupId" label={t('device.rules.targetGroup')} rules={[{ required: true, message: t('device.rules.selectTargetGroup') }]}>
            <Select
              placeholder={t('common.pleaseSelect')}
              options={MOCK_DEVICE_GROUPS.map((g) => ({ label: g.groupName, value: g.id }))}
            />
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
          <div style={{ marginBottom: 8, fontWeight: 500, color: 'var(--color-text)' }}>{t('device.rules.matchingRule')}</div>
          <Form.Item name="matchingMode" label={t('device.rules.matchingMode')} rules={[{ required: true }]}>
            <Radio.Group onChange={handleMatchingModeChange}>
              <Radio value="deviceName">{t('device.rules.deviceName')}</Radio>
              {SUPPORT_GSM && DEVICE_TYPE === 'ENB' && <Radio value="lac">LAC</Radio>}
              {DEVICE_TYPE !== 'CPE' && <Radio value="tac">TAC</Radio>}
            </Radio.Group>
          </Form.Item>

          {/* 设备名称过滤条件 */}
          {matchingMode === 'deviceName' && (
            <>
              <Form.Item label={<span>{t('device.rules.filterCondition')} <Text type="secondary" style={{ fontSize: 12 }}>{t('device.rules.conditionLimit', { max: 10 })}</Text></span>}>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                  {nameFilters.map((filter, index) => (
                    <div key={filter.id} style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                      {index === 0 ? (
                        <>
                          <Select
                            value={filter.condition}
                            style={{ width: 120 }}
                            options={FILTER_CONDITION_OPTIONS}
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
                            options={AND_OR_OPTIONS}
                            onChange={(v) => handleUpdateFilter(filter.id, 'andOr', v)}
                          />
                          <Select
                            value={filter.condition}
                            style={{ width: 120 }}
                            options={FILTER_CONDITION_OPTIONS}
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
                  <Button
                    type="dashed"
                    icon={<PlusOutlined />}
                    onClick={handleAddFilter}
                    style={{ marginTop: 8 }}
                  >
                    添加条件
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

          {/* TAC/LAC 输入 */}
          {(matchingMode === 'tac' || matchingMode === 'lac') && (
            <Form.Item
              name="tacRag"
              label={matchingMode === 'tac' ? 'TAC' : 'LAC'}
              rules={[{ required: true, message: `请输入${matchingMode === 'tac' ? 'TAC' : 'LAC'}范围` }]}
              extra={
                <Text type="secondary" style={{ fontSize: 12 }}>
                  格式: 1,2,3 或 1-10,20-30 (范围: {DEVICE_TYPE === 'ENB' ? '0-65535' : '0-16777215'})
                </Text>
              }
            >
              <Input placeholder="eg: 1,2,3,1-3" maxLength={50} />
            </Form.Item>
          )}
        </Form>
      </Drawer>

      {/* 应用规则弹窗 */}
      <Modal
        title="应用规则"
        open={activeModalOpen}
        onOk={handleExecute}
        onCancel={() => setActiveModalOpen(false)}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        width={480}
      >
        <div style={{ marginBottom: 12, color: 'var(--color-text-secondary)' }}>选择要应用规则的设备组</div>
        <div
          style={{
            border: '1px solid var(--color-border)',
            borderRadius: 6,
            padding: 8,
            maxHeight: 400,
            overflow: 'auto',
          }}
        >
          <Tree
            checkable
            defaultExpandedKeys={['root']}
            checkedKeys={selectedGroupIds}
            onCheck={(keys) => setSelectedGroupIds(keys as string[])}
            treeData={MOCK_GROUP_TREE.map((node) => ({
              key: node.id,
              title: node.groupName,
              children: node.children?.map((child) => ({
                key: child.id,
                title: child.groupName,
              })),
            }))}
          />
        </div>
      </Modal>
    </>
  );
}
