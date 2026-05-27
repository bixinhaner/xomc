import { useState, useMemo, useEffect } from 'react';
import type { Key } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  App,
  Button,
  Card,
  Input,
  Modal,
  Form,
  InputNumber,
  Select,
  Switch,
  Space,
  Tag,
  List,
  Spin,
  Empty,
  Tooltip,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  PlusSquareOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import { adminApi } from '@core/services/api/adminApi';
import type {
  Dictionary,
  DictionaryDetail,
  CreateDictionaryPayload,
  UpdateDictionaryPayload,
  CreateDictionaryDetailPayload,
  UpdateDictionaryDetailPayload,
} from '@core/services/api/adminApi';

// ---- Dictionary List (Left Panel) ----
interface DictListPanelProps {
  selectedId: number | null;
  onSelect: (dict: Dictionary) => void;
}

function DictListPanel({ selectedId, onSelect }: DictListPanelProps) {
  const t = useT();
  const { message, modal } = App.useApp();
  const queryClient = useQueryClient();

  const [searchText, setSearchText] = useState('');
  const [dictModalOpen, setDictModalOpen] = useState(false);
  const [editingDict, setEditingDict] = useState<Dictionary | null>(null);
  const [dictForm] = Form.useForm<CreateDictionaryPayload>();

  const { data: dictData, isLoading } = useQuery({
    queryKey: ['dictionaries'],
    queryFn: () => adminApi.getDictionaryList(),
  });

  const createDictMutation = useMutation({
    mutationFn: (payload: CreateDictionaryPayload) => adminApi.createDictionary(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['dictionaries'] });
      void message.success(t('common.save'));
      setDictModalOpen(false);
      dictForm.resetFields();
    },
    onError: () => void message.error(t('common.saveFailed') || '保存失败'),
  });

  const updateDictMutation = useMutation({
    mutationFn: (payload: UpdateDictionaryPayload) => adminApi.updateDictionary(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['dictionaries'] });
      void message.success(t('common.save'));
      setDictModalOpen(false);
      dictForm.resetFields();
    },
    onError: () => void message.error(t('common.saveFailed') || '保存失败'),
  });

  const deleteDictMutation = useMutation({
    mutationFn: (id: number) => adminApi.deleteDictionary(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['dictionaries'] });
      void message.success(t('common.deleteSuccess'));
    },
    onError: () => void message.error(t('common.deleteFailed') || '删除失败'),
  });

  const filteredList = useMemo(() => {
    const list = dictData?.list ?? [];
    if (!searchText) return list;
    const lower = searchText.toLowerCase();
    return list.filter(
      (d) =>
        d.name.toLowerCase().includes(lower) ||
        d.type.toLowerCase().includes(lower),
    );
  }, [dictData, searchText]);

  const openAdd = () => {
    setEditingDict(null);
    dictForm.resetFields();
    dictForm.setFieldsValue({ status: true });
    setDictModalOpen(true);
  };

  const openEdit = (dict: Dictionary) => {
    setEditingDict(dict);
    dictForm.setFieldsValue({
      name: dict.name,
      type: dict.type,
      status: dict.status,
      desc: dict.desc,
    });
    setDictModalOpen(true);
  };

  const handleDelete = (dict: Dictionary) => {
    modal.confirm({
      title: t('common.delete'),
      content: t('dictionary.deleteDictConfirm'),
      okType: 'danger',
      onOk: () => deleteDictMutation.mutate(dict.id),
    });
  };

  const handleSave = () => {
    void dictForm.validateFields().then((vals) => {
      if (editingDict) {
        updateDictMutation.mutate({ id: editingDict.id, ...vals });
      } else {
        createDictMutation.mutate(vals as CreateDictionaryPayload);
      }
    });
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', borderRight: '1px solid #f0f0f0' }}>
      {/* Header */}
      <div style={{ padding: '12px 12px 8px', borderBottom: '1px solid #f0f0f0', background: '#fafafa' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
          <span style={{ fontWeight: 600, fontSize: 14 }}>{t('dictionary.listTitle')}</span>
          <Button
            type="primary"
            size="small"
            icon={<PlusOutlined />}
            onClick={openAdd}
          >
            {t('dictionary.addDict')}
          </Button>
        </div>
        <Input
          placeholder={t('common.search')}
          prefix={<SearchOutlined />}
          value={searchText}
          onChange={(e) => setSearchText(e.target.value)}
          size="small"
          allowClear
        />
      </div>

      {/* List */}
      <div style={{ flex: 1, overflow: 'auto' }}>
        {isLoading ? (
          <div style={{ display: 'flex', justifyContent: 'center', padding: 24 }}>
            <Spin />
          </div>
        ) : filteredList.length === 0 ? (
          <Empty style={{ marginTop: 40 }} />
        ) : (
          <List
            dataSource={filteredList}
            renderItem={(dict) => (
              <List.Item
                key={dict.id}
                onClick={() => onSelect(dict)}
                style={{
                  cursor: 'pointer',
                  padding: '8px 12px',
                  background: selectedId === dict.id ? '#e6f4ff' : undefined,
                  borderLeft: selectedId === dict.id ? '3px solid #1677ff' : '3px solid transparent',
                  transition: 'background 0.2s',
                }}
              >
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <span
                      style={{
                        fontWeight: 500,
                        color: selectedId === dict.id ? '#1677ff' : undefined,
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                        maxWidth: 130,
                      }}
                      title={dict.name}
                    >
                      {dict.name}
                    </span>
                    <Space size={2}>
                      <Tooltip title={t('common.edit')}>
                        <Button
                          type="text"
                          size="small"
                          icon={<EditOutlined />}
                          onClick={(e) => { e.stopPropagation(); openEdit(dict); }}
                        />
                      </Tooltip>
                      <Tooltip title={t('common.delete')}>
                        <Button
                          type="text"
                          size="small"
                          danger
                          icon={<DeleteOutlined />}
                          onClick={(e) => { e.stopPropagation(); handleDelete(dict); }}
                        />
                      </Tooltip>
                    </Space>
                  </div>
                  <div style={{ fontSize: 11, color: '#8c8c8c', marginTop: 2 }}>
                    <Tag color="blue" style={{ fontSize: 11, lineHeight: '18px' }}>{dict.type}</Tag>
                    {!dict.status && <Tag color="default" style={{ fontSize: 11, lineHeight: '18px' }}>{t('status.disabled')}</Tag>}
                  </div>
                </div>
              </List.Item>
            )}
          />
        )}
      </div>

      {/* Add/Edit Dictionary Modal */}
      <Modal
        title={editingDict ? t('dictionary.editDict') : t('dictionary.addDict')}
        open={dictModalOpen}
        onOk={handleSave}
        onCancel={() => { setDictModalOpen(false); dictForm.resetFields(); setEditingDict(null); }}
        confirmLoading={createDictMutation.isPending || updateDictMutation.isPending}
        width={480}
        destroyOnHidden
      >
        <Form form={dictForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="name" label={t('dictionary.name')} rules={[{ required: true }]}>
            <Input placeholder={t('dictionary.name')} />
          </Form.Item>
          <Form.Item name="type" label={t('dictionary.type')} rules={[{ required: true }]}>
            <Input placeholder={t('dictionary.type')} disabled={!!editingDict} />
          </Form.Item>
          <Form.Item name="status" label={t('dictionary.status')} valuePropName="checked" initialValue={true}>
            <Switch checkedChildren={t('status.enabled')} unCheckedChildren={t('status.disabled')} />
          </Form.Item>
          <Form.Item name="desc" label={t('dictionary.desc')}>
            <Input.TextArea rows={2} placeholder={t('dictionary.desc')} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

// ---- Dictionary Detail Table (Right Panel) ----
interface DictDetailPanelProps {
  selectedDict: Dictionary | null;
}

function DictDetailPanel({ selectedDict }: DictDetailPanelProps) {
  const t = useT();
  const { message, modal } = App.useApp();
  const queryClient = useQueryClient();

  const [searchLabel, setSearchLabel] = useState('');
  const [detailModalOpen, setDetailModalOpen] = useState(false);
  const [editingDetail, setEditingDetail] = useState<DictionaryDetail | null>(null);
  const [detailForm] = Form.useForm<CreateDictionaryDetailPayload>();

  const { data: detailData, isLoading } = useQuery({
    queryKey: ['dictionary-details', selectedDict?.id, searchLabel],
    queryFn: () =>
      adminApi.getDictionaryDetailList({
        sysDictionaryId: selectedDict!.id,
        label: searchLabel || undefined,
      }),
    enabled: !!selectedDict,
  });

  const createDetailMutation = useMutation({
    mutationFn: (payload: CreateDictionaryDetailPayload) => adminApi.createDictionaryDetail(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['dictionary-details', selectedDict?.id] });
      void message.success(t('common.save'));
      setDetailModalOpen(false);
      detailForm.resetFields();
    },
    onError: () => void message.error(t('common.saveFailed') || '保存失败'),
  });

  const updateDetailMutation = useMutation({
    mutationFn: (payload: UpdateDictionaryDetailPayload) => adminApi.updateDictionaryDetail(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['dictionary-details', selectedDict?.id] });
      void message.success(t('common.save'));
      setDetailModalOpen(false);
      detailForm.resetFields();
    },
    onError: () => void message.error(t('common.saveFailed') || '保存失败'),
  });

  const deleteDetailMutation = useMutation({
    mutationFn: (id: number) => adminApi.deleteDictionaryDetail(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['dictionary-details', selectedDict?.id] });
      void message.success(t('common.deleteSuccess'));
    },
    onError: () => void message.error(t('common.deleteFailed') || '删除失败'),
  });

  // PRD §10：「+ 添加子项」打开抽屉时把 parent_id 预填为该行 id；
  // 顶部「+ 添加详情」打开时 parent_id 为 null。
  const openAdd = (parentDetail?: DictionaryDetail) => {
    if (!selectedDict) return;
    setEditingDetail(null);
    detailForm.resetFields();
    detailForm.setFieldsValue({
      status: true,
      sort: 0,
      sysDictionaryId: selectedDict.id,
      parentId: parentDetail ? parentDetail.id : null,
    });
    setDetailModalOpen(true);
  };

  const openEdit = (detail: DictionaryDetail) => {
    setEditingDetail(detail);
    detailForm.setFieldsValue({
      label: detail.label,
      value: detail.value,
      extend: detail.extend,
      status: detail.status,
      sort: detail.sort,
      sysDictionaryId: detail.sysDictionaryId,
      parentId: detail.parentId ?? null,
    });
    setDetailModalOpen(true);
  };

  const handleDelete = (detail: DictionaryDetail) => {
    modal.confirm({
      title: t('common.delete'),
      content: t('dictionary.deleteDetailConfirm'),
      okType: 'danger',
      onOk: () => deleteDetailMutation.mutate(detail.id),
    });
  };

  const handleSave = () => {
    void detailForm.validateFields().then((vals) => {
      // antd Select allowClear 会把空值返回为 undefined；显式归一为 null（=切顶层）。
      const normalized = {
        ...vals,
        parentId: (vals as { parentId?: number | null }).parentId ?? null,
      };
      if (editingDetail) {
        updateDetailMutation.mutate({ id: editingDetail.id, ...normalized });
      } else {
        createDetailMutation.mutate({
          ...normalized,
          sysDictionaryId: selectedDict!.id,
        } as CreateDictionaryDetailPayload);
      }
    });
  };

  const handleStatusChange = (detail: DictionaryDetail, checked: boolean) => {
    updateDetailMutation.mutate({ id: detail.id, status: checked });
  };

  // PRD §10 Q9：父级下拉选项 = 当前字典内所有可选父项。
  //   - 编辑模式：剔除自己 + 自己的所有后代（防环 / 与后端 cycle 检验对齐）
  //   - 仅显示 level < 2 的项（其下加一层就到 MaxDepth-1=2，仍合法）
  //   - 创建模式：显示所有 level < 2 的项
  const parentOptions = useMemo(() => {
    const list = detailData?.list ?? [];
    if (!editingDetail) {
      return list
        .filter((d) => (d.level ?? 0) < 2)
        .map((d) => ({ label: `${d.label} (level ${d.level ?? 0})`, value: d.id }));
    }
    // 计算 editingDetail 的所有后代 id 集合（含自身）
    const excluded = new Set<number>([editingDetail.id]);
    let changed = true;
    while (changed) {
      changed = false;
      for (const d of list) {
        if (d.parentId != null && excluded.has(d.parentId) && !excluded.has(d.id)) {
          excluded.add(d.id);
          changed = true;
        }
      }
    }
    return list
      .filter((d) => !excluded.has(d.id) && (d.level ?? 0) < 2)
      .map((d) => ({ label: `${d.label} (level ${d.level ?? 0})`, value: d.id }));
  }, [detailData?.list, editingDetail]);

  const columns: DataTableColumn<DictionaryDetail & Record<string, unknown>>[] = useMemo(
    () => [
      {
        key: 'label',
        title: t('dictionary.label'),
        dataIndex: 'label',
        width: 140,
      },
      {
        key: 'value',
        title: t('dictionary.value'),
        dataIndex: 'value',
        width: 120,
        render: (val) => (
          <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>
        ),
      },
      {
        key: 'extend',
        title: t('dictionary.extend'),
        dataIndex: 'extend',
        width: 120,
        render: (val) => (val ? String(val) : '—'),
      },
      {
        // PRD §10 v0.2：层级列（0=顶层 / 1=一级子 / 2=二级子）
        key: 'level',
        title: '层级',
        dataIndex: 'level',
        width: 70,
        render: (val) => Number(val ?? 0),
      },
      {
        key: 'status',
        title: t('dictionary.status'),
        dataIndex: 'status',
        width: 90,
        render: (val, record) => {
          const detail = record as DictionaryDetail;
          return (
            <Switch
              size="small"
              checked={Boolean(val)}
              loading={updateDetailMutation.isPending}
              onChange={(checked) => handleStatusChange(detail, checked)}
            />
          );
        },
      },
      {
        key: 'sort',
        title: t('dictionary.sort'),
        dataIndex: 'sort',
        width: 70,
      },
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 220,
        fixed: 'right',
        render: (_, record) => {
          const detail = record as DictionaryDetail;
          // PRD §10：仅当 detail.level < MaxDepth-1 (=2) 时允许加子项。
          const canAddChild = (detail.level ?? 0) < 2;
          return (
            <Space size={4}>
              <Tooltip title={canAddChild ? '' : '已达最大深度'}>
                <Button
                  type="link"
                  size="small"
                  icon={<PlusSquareOutlined />}
                  onClick={() => openAdd(detail)}
                  disabled={!canAddChild}
                >
                  添加子项
                </Button>
              </Tooltip>
              <Button
                type="link"
                size="small"
                icon={<EditOutlined />}
                onClick={() => openEdit(detail)}
              >
                {t('common.edit')}
              </Button>
              <Button
                type="link"
                size="small"
                danger
                icon={<DeleteOutlined />}
                onClick={() => handleDelete(detail)}
              >
                {t('common.delete')}
              </Button>
            </Space>
          );
        },
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [t, updateDetailMutation.isPending],
  );

  // PRD §10 v0.2：把扁平的字典项列表构建成 parentId → children 的树形结构，
  //   交给 Antd Table 的内置 tree 模式渲染（自动添加缩进与展开/收起箭头）。
  type DetailWithChildren = DictionaryDetail & { children?: DetailWithChildren[] };
  const treeData = useMemo<DetailWithChildren[]>(() => {
    const flat = detailData?.list ?? [];
    const map = new Map<number, DetailWithChildren>();
    for (const item of flat) {
      map.set(item.id, { ...item });
    }
    const roots: DetailWithChildren[] = [];
    for (const node of map.values()) {
      if (node.parentId != null && map.has(node.parentId)) {
        const parent = map.get(node.parentId)!;
        if (!parent.children) parent.children = [];
        parent.children.push(node);
      } else {
        roots.push(node);
      }
    }
    const bySort = (a: DictionaryDetail, b: DictionaryDetail) => (a.sort ?? 0) - (b.sort ?? 0);
    roots.sort(bySort);
    for (const node of map.values()) {
      if (node.children) node.children.sort(bySort);
    }
    return roots;
  }, [detailData?.list]);

  // 默认展开所有有子项的父节点；用户可在表格内自行收起，数据刷新后重新展开。
  const [expandedRowKeys, setExpandedRowKeys] = useState<Key[]>([]);
  useEffect(() => {
    const flat = detailData?.list ?? [];
    const parentIds: Key[] = [];
    const seen = new Set<number>();
    for (const d of flat) {
      if (d.parentId != null && !seen.has(d.parentId)) {
        seen.add(d.parentId);
        parentIds.push(d.parentId);
      }
    }
    setExpandedRowKeys(parentIds);
  }, [detailData?.list]);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Header */}
      <div
        style={{
          padding: '8px 16px',
          borderBottom: '1px solid #f0f0f0',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          background: '#fafafa',
          gap: 8,
        }}
      >
        <span style={{ fontWeight: 600, fontSize: 14, flexShrink: 0 }}>
          {selectedDict
            ? `${t('dictionary.detailTitle')} — ${selectedDict.name}`
            : t('dictionary.detailTitle')}
        </span>
        <Space>
          <Input
            placeholder={t('dictionary.searchLabel')}
            prefix={<SearchOutlined />}
            value={searchLabel}
            onChange={(e) => setSearchLabel(e.target.value)}
            style={{ width: 200 }}
            size="small"
            allowClear
            disabled={!selectedDict}
          />
          <Button
            type="primary"
            size="small"
            icon={<PlusOutlined />}
            disabled={!selectedDict}
            onClick={() => openAdd()}
          >
            {t('dictionary.addDetail')}
          </Button>
        </Space>
      </div>

      {/* Table */}
      <div style={{ flex: 1, overflow: 'auto' }}>
        {!selectedDict ? (
          <Empty description={t('common.selectHint') || '请先选择左侧字典'} style={{ marginTop: 80 }} />
        ) : (
          <Card
            size="small"
            bordered
            style={{ height: '100%', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
            styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
          >
            <DataTable
              tableId="dict-detail-table"
              columns={columns}
              dataSource={treeData as (DictionaryDetail & Record<string, unknown>)[]}
              loading={isLoading}
              rowKey="id"
              total={detailData?.total ?? 0}
              pageSize={20}
              currentPage={1}
              expandable={{
                expandedRowKeys,
                onExpandedRowsChange: (keys) => setExpandedRowKeys([...keys]),
                indentSize: 24,
              }}
            />
          </Card>
        )}
      </div>

      {/* Add/Edit Detail Modal */}
      <Modal
        title={editingDetail ? t('dictionary.editDetail') : t('dictionary.addDetail')}
        open={detailModalOpen}
        onOk={handleSave}
        onCancel={() => { setDetailModalOpen(false); detailForm.resetFields(); setEditingDetail(null); }}
        confirmLoading={createDetailMutation.isPending || updateDetailMutation.isPending}
        width={480}
        destroyOnHidden
      >
        <Form form={detailForm} layout="vertical" style={{ marginTop: 16 }}>
          {/* PRD §10：父级字典项 — allowClear=切顶层；过滤掉自身+后代防环 */}
          <Form.Item name="parentId" label="父级字典项">
            <Select
              allowClear
              placeholder="（顶层项）"
              options={parentOptions}
              showSearch
              optionFilterProp="label"
            />
          </Form.Item>
          <Form.Item name="label" label={t('dictionary.label')} rules={[{ required: true }]}>
            <Input placeholder={t('dictionary.label')} />
          </Form.Item>
          <Form.Item name="value" label={t('dictionary.value')} rules={[{ required: true }]}>
            <Input placeholder={t('dictionary.value')} />
          </Form.Item>
          <Form.Item name="extend" label={t('dictionary.extend')}>
            <Input placeholder={t('dictionary.extend')} />
          </Form.Item>
          <Form.Item name="status" label={t('dictionary.status')} valuePropName="checked" initialValue={true}>
            <Switch checkedChildren={t('status.enabled')} unCheckedChildren={t('status.disabled')} />
          </Form.Item>
          <Form.Item name="sort" label={t('dictionary.sort')} initialValue={0}>
            <InputNumber min={0} max={9999} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="sysDictionaryId" hidden>
            <InputNumber />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

// ---- Main Page ----
export default function DataDictionary() {
  const [selectedDict, setSelectedDict] = useState<Dictionary | null>(null);

  return (
    <App>
      <div style={{ display: 'flex', height: '100%', background: '#fff' }}>
        {/* Left panel - dict list */}
        <div style={{ width: 300, flexShrink: 0, height: '100%', overflow: 'hidden' }}>
          <DictListPanel
            selectedId={selectedDict?.id ?? null}
            onSelect={(dict) => setSelectedDict(dict)}
          />
        </div>

        {/* Right panel - dict details */}
        <div style={{ flex: 1, height: '100%', overflow: 'hidden' }}>
          <DictDetailPanel selectedDict={selectedDict} />
        </div>
      </div>
    </App>
  );
}
