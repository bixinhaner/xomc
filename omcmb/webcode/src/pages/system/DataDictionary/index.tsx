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
  theme,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  PlusSquareOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import { useI18nText } from '@/hooks/useI18nText';
import { adminApi } from '@core/services/api/adminApi';
import type {
  Dictionary,
  DictionaryDetail,
  CreateDictionaryPayload,
  UpdateDictionaryPayload,
  CreateDictionaryDetailPayload,
  UpdateDictionaryDetailPayload,
} from '@core/services/api/adminApi';
import SourcePicker, { type SourcePickerValue } from './SourcePicker';

// ---- Dictionary List (Left Panel) ----
interface DictListPanelProps {
  selectedId: number | null;
  onSelect: (dict: Dictionary) => void;
}

function DictListPanel({ selectedId, onSelect }: DictListPanelProps) {
  const t = useT();
  const { token } = theme.useToken();
  const { fromRecord } = useI18nText();
  const { message, modal } = App.useApp();
  const queryClient = useQueryClient();

  const [searchText, setSearchText] = useState('');
  const [dictModalOpen, setDictModalOpen] = useState(false);
  const [editingDict, setEditingDict] = useState<Dictionary | null>(null);
  const [dictForm] = Form.useForm<CreateDictionaryPayload>();
  // T-0182 数据源绑定:与表单解耦的受控状态(避免 antd Form.Item 嵌套 SourcePicker 的复杂性)
  const [sourcePickerValue, setSourcePickerValue] = useState<SourcePickerValue>({});
  // T-0182 手动刷新中的字典 ID(loading + 禁用其它操作)
  const [refreshingId, setRefreshingId] = useState<number | null>(null);

  const { data: dictData, isLoading } = useQuery({
    queryKey: ['dictionaries'],
    queryFn: () => adminApi.getDictionaryList(),
  });

  const createDictMutation = useMutation({
    mutationFn: (payload: CreateDictionaryPayload) => adminApi.createDictionary(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['dictionaries'] });
      void queryClient.invalidateQueries({ queryKey: ['dictionary'] });
      void message.success(t('common.save'));
      setDictModalOpen(false);
      dictForm.resetFields();
    },
    onError: () => void message.error(t('common.saveFailed')),
  });

  const updateDictMutation = useMutation({
    mutationFn: (payload: UpdateDictionaryPayload) => adminApi.updateDictionary(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['dictionaries'] });
      void queryClient.invalidateQueries({ queryKey: ['dictionary'] });
      void message.success(t('common.save'));
      setDictModalOpen(false);
      dictForm.resetFields();
    },
    onError: () => void message.error(t('common.saveFailed')),
  });

  // T-0182 手动刷新单字典的数据源同步。
  const refreshSourceMutation = useMutation({
    mutationFn: (id: number) => adminApi.refreshDictionarySource(id),
    onMutate: (id) => setRefreshingId(id),
    onSettled: () => setRefreshingId(null),
    onSuccess: (res) => {
      void queryClient.invalidateQueries({ queryKey: ['dictionaries'] });
      void queryClient.invalidateQueries({ queryKey: ['dictionary'] });
      // 字典项列表也要刷新(同步可能改了 auto 项)
      void queryClient.invalidateQueries({ queryKey: ['dictionary-details'] });
      void message.success(
        t('dictionary.source.refreshSuccess', {
          inserted: res.inserted,
          deleted: res.deleted,
          total: res.total,
        }),
      );
    },
    onError: (err) => {
      const msg = err instanceof Error ? err.message : String(err);
      void message.error(t('dictionary.source.refreshFailed', { error: msg }));
    },
  });

  const deleteDictMutation = useMutation({
    mutationFn: (id: number) => adminApi.deleteDictionary(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['dictionaries'] });
      void queryClient.invalidateQueries({ queryKey: ['dictionary'] });
      void message.success(t('common.deleteSuccess'));
    },
    onError: () => void message.error(t('common.deleteFailed')),
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
    setSourcePickerValue({});
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
    // 编辑现有字典:回填已有源绑定;手工字典三字段全 null
    setSourcePickerValue({
      sourceTable: dict.sourceTable ?? null,
      sourceLabelField: dict.sourceLabelField ?? null,
      sourceValueField: dict.sourceValueField ?? null,
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

  // T-0182 手动刷新按钮:仅托管字典显示。
  const handleRefreshSource = (dict: Dictionary) => {
    refreshSourceMutation.mutate(dict.id);
  };

  // 组合 form fields + source picker state 提交。
  // 校验三字段同时空或同时填(部分填写阻拦在前端,避免后端 400 噪声)。
  const collectSourceFields = (): {
    valid: boolean;
    fields: {
      sourceTable?: string | null;
      sourceLabelField?: string | null;
      sourceValueField?: string | null;
    };
  } => {
    const t1 = sourcePickerValue.sourceTable;
    const l = sourcePickerValue.sourceLabelField;
    const v = sourcePickerValue.sourceValueField;
    const all = [t1, l, v];
    const filled = all.filter((x) => x && x !== '').length;
    if (filled === 0) {
      // 未绑定:create 不传字段,update 传 null 表示解绑(若原来是托管字典)
      return { valid: true, fields: {} };
    }
    if (filled === 3) {
      return {
        valid: true,
        fields: {
          sourceTable: t1 || undefined,
          sourceLabelField: l || undefined,
          sourceValueField: v || undefined,
        },
      };
    }
    return { valid: false, fields: {} };
  };

  const handleSave = () => {
    void dictForm.validateFields().then((vals) => {
      const sc = collectSourceFields();
      if (!sc.valid) {
        void message.error(t('dictionary.source.partialFieldsError'));
        return;
      }

      // 编辑时:旧绑定 → 新绑定切表 → 弹确认(auto 项会被清空重建)
      const isSwitchingTable =
        !!editingDict &&
        !!editingDict.sourceTable &&
        !!sc.fields.sourceTable &&
        editingDict.sourceTable !== sc.fields.sourceTable;
      // 编辑时:旧绑定 → 解绑 → 弹确认
      const isUnbinding =
        !!editingDict &&
        !!editingDict.sourceTable &&
        Object.keys(sc.fields).length === 0;

      const doSave = () => {
        if (editingDict) {
          // update 路径:解绑时显式发空字符串触发后端 sourceCleared 分支
          const updateBody: UpdateDictionaryPayload = { id: editingDict.id, ...vals };
          if (Object.keys(sc.fields).length === 0 && editingDict.sourceTable) {
            updateBody.sourceTable = '';
            updateBody.sourceLabelField = '';
            updateBody.sourceValueField = '';
          } else if (sc.fields.sourceTable) {
            updateBody.sourceTable = sc.fields.sourceTable;
            updateBody.sourceLabelField = sc.fields.sourceLabelField;
            updateBody.sourceValueField = sc.fields.sourceValueField;
          }
          updateDictMutation.mutate(updateBody);
        } else {
          // create 路径
          const createBody: CreateDictionaryPayload = vals as CreateDictionaryPayload;
          if (sc.fields.sourceTable) {
            createBody.sourceTable = sc.fields.sourceTable;
            createBody.sourceLabelField = sc.fields.sourceLabelField ?? undefined;
            createBody.sourceValueField = sc.fields.sourceValueField ?? undefined;
          }
          createDictMutation.mutate(createBody);
        }
      };

      if (isSwitchingTable) {
        modal.confirm({
          title: t('dictionary.source.switchTableConfirm'),
          content: t('dictionary.source.switchTableContent'),
          okType: 'danger',
          onOk: doSave,
        });
      } else if (isUnbinding) {
        modal.confirm({
          title: t('dictionary.source.unbindConfirm'),
          content: t('dictionary.source.unbindContent'),
          okType: 'danger',
          onOk: doSave,
        });
      } else {
        doSave();
      }
    });
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', borderRight: `1px solid ${token.colorBorderSecondary}` }}>
      {/* Header */}
      <div style={{ padding: '12px 12px 8px', borderBottom: `1px solid ${token.colorBorderSecondary}`, background: token.colorFillAlter }}>
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
                  background: selectedId === dict.id ? token.controlItemBgActive : undefined,
                  borderLeft: selectedId === dict.id ? `3px solid ${token.colorPrimary}` : '3px solid transparent',
                  transition: 'background 0.2s',
                }}
              >
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <span
                      style={{
                        fontWeight: 500,
                        color: selectedId === dict.id ? token.colorPrimary : undefined,
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                        maxWidth: 130,
                      }}
                      title={fromRecord(dict as unknown as Record<string, unknown>, 'name') || dict.name}
                    >
                      {fromRecord(dict as unknown as Record<string, unknown>, 'name') || dict.name}
                    </span>
                    <Space size={2}>
                      {/* T-0182 手动刷新:仅托管字典(sourceTable != null)显示;按钮放在编辑前 */}
                      {dict.sourceTable && (
                        <Tooltip title={t('dictionary.source.refreshBtn')}>
                          <Button
                            type="text"
                            size="small"
                            icon={<SyncOutlined spin={refreshingId === dict.id} />}
                            disabled={refreshingId !== null}
                            onClick={(e) => {
                              e.stopPropagation();
                              handleRefreshSource(dict);
                            }}
                          />
                        </Tooltip>
                      )}
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
                  <div style={{ fontSize: 11, color: token.colorTextSecondary, marginTop: 2 }}>
                    <Tag color="blue" style={{ fontSize: 11, lineHeight: '18px' }}>{dict.type}</Tag>
                    {!dict.status && <Tag color="default" style={{ fontSize: 11, lineHeight: '18px' }}>{t('status.disabled')}</Tag>}
                    {/* T-0182 托管字典标记 + 上次同步时间 */}
                    {dict.sourceTable && (
                      <Tooltip
                        title={
                          dict.lastRefreshAt
                            ? t('dictionary.source.lastRefreshAt', { time: dict.lastRefreshAt })
                            : t('dictionary.source.notSyncedYet')
                        }
                      >
                        <Tag color="cyan" style={{ fontSize: 11, lineHeight: '18px' }}>
                          {t('dictionary.source.managedTag')}
                        </Tag>
                      </Tooltip>
                    )}
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
          {/* T-0182 数据源(可选):三字段同时空 = 手工字典;同时填 = 托管字典 */}
          <SourcePicker
            value={sourcePickerValue}
            onChange={setSourcePickerValue}
          />
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
  const { token } = theme.useToken();
  const { fromRecord } = useI18nText();
  const { message, modal } = App.useApp();
  const queryClient = useQueryClient();

  const [searchLabel, setSearchLabel] = useState('');
  const [detailModalOpen, setDetailModalOpen] = useState(false);
  const [editingDetail, setEditingDetail] = useState<DictionaryDetail | null>(null);
  const [detailForm] = Form.useForm<CreateDictionaryDetailPayload & { labelEn?: string }>();
  // T-0182:托管字典(source_table != null)隐藏"+添加详情" / "+添加子项";
  // auto 行编辑/删除禁用(只读),manual 行仍允许(方案 A 兼容历史数据)。
  const isManaged = !!selectedDict?.sourceTable;

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
      void queryClient.invalidateQueries({ queryKey: ['dictionary'] });
      void message.success(t('common.save'));
      setDetailModalOpen(false);
      detailForm.resetFields();
    },
    onError: () => void message.error(t('common.saveFailed')),
  });

  const updateDetailMutation = useMutation({
    mutationFn: (payload: UpdateDictionaryDetailPayload) => adminApi.updateDictionaryDetail(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['dictionary-details', selectedDict?.id] });
      void queryClient.invalidateQueries({ queryKey: ['dictionary'] });
      void message.success(t('common.save'));
      setDetailModalOpen(false);
      detailForm.resetFields();
    },
    onError: () => void message.error(t('common.saveFailed')),
  });

  const deleteDetailMutation = useMutation({
    mutationFn: (id: number) => adminApi.deleteDictionaryDetail(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['dictionary-details', selectedDict?.id] });
      void queryClient.invalidateQueries({ queryKey: ['dictionary'] });
      void message.success(t('common.deleteSuccess'));
    },
    onError: () => void message.error(t('common.deleteFailed')),
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
      label: detail.labelI18n?.['zh-CN'] ?? detail.label,
      labelEn: detail.labelI18n?.['en-US'] ?? '',
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
      // 展示值双语:label(中文) + labelEn(英语) → label_i18n;label 保留=中文(向后兼容)。
      const { labelEn, ...rest } = vals as typeof vals & { labelEn?: string };
      const labelI18n = { 'zh-CN': rest.label ?? '', 'en-US': (labelEn ?? '').trim() };
      // antd Select allowClear 会把空值返回为 undefined；显式归一为 null（=切顶层）。
      const normalized = {
        ...rest,
        labelI18n,
        parentId: (rest as { parentId?: number | null }).parentId ?? null,
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
        render: (_val, record) =>
          fromRecord(record as unknown as Record<string, unknown>, 'label') || record.label,
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
        title: t('dictionary.level'),
        dataIndex: 'level',
        width: 70,
        render: (val) => Number(val ?? 0),
      },
      {
        // T-0182:来源列(auto=自动同步 / manual=手工录入)
        key: 'origin',
        title: t('dictionary.detail.originColumn'),
        dataIndex: 'origin',
        width: 100,
        render: (val) => {
          const o = (val as string) ?? 'manual';
          return o === 'auto' ? (
            <Tag color="cyan">{t('dictionary.detail.originAuto')}</Tag>
          ) : (
            <Tag color="default">{t('dictionary.detail.originManual')}</Tag>
          );
        },
      },
      {
        key: 'status',
        title: t('dictionary.status'),
        dataIndex: 'status',
        width: 90,
        render: (val, record) => {
          const detail = record as DictionaryDetail;
          // T-0182:auto 行的 status 也不允许手工切换 — 同步任务才能改
          const isAuto = detail.origin === 'auto';
          return (
            <Tooltip title={isAuto ? t('dictionary.detail.autoReadOnly') : ''}>
              <Switch
                size="small"
                checked={Boolean(val)}
                loading={updateDetailMutation.isPending}
                disabled={isAuto}
                onChange={(checked) => handleStatusChange(detail, checked)}
              />
            </Tooltip>
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
          // T-0182:托管字典里 auto 行禁编辑/删除/+子项;manual 行(历史补录)仍可改
          const isAuto = detail.origin === 'auto';
          const autoTip = t('dictionary.detail.autoReadOnly');
          return (
            <Space size={4}>
              {/* 托管字典下"+ 添加子项"按钮整体隐藏(托管字典不支持层级,参 PRD §3.2.5) */}
              {!isManaged && (
                <Tooltip title={canAddChild ? '' : t('dictionary.maxDepthReached')}>
                  <Button
                    type="link"
                    size="small"
                    icon={<PlusSquareOutlined />}
                    onClick={() => openAdd(detail)}
                    disabled={!canAddChild}
                  >
                    {t('dictionary.addChild')}
                  </Button>
                </Tooltip>
              )}
              <Tooltip title={isAuto ? autoTip : ''}>
                <Button
                  type="link"
                  size="small"
                  icon={<EditOutlined />}
                  disabled={isAuto}
                  onClick={() => openEdit(detail)}
                >
                  {t('common.edit')}
                </Button>
              </Tooltip>
              <Tooltip title={isAuto ? autoTip : ''}>
                <Button
                  type="link"
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                  disabled={isAuto}
                  onClick={() => handleDelete(detail)}
                >
                  {t('common.delete')}
                </Button>
              </Tooltip>
            </Space>
          );
        },
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [t, updateDetailMutation.isPending, isManaged],
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
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          background: token.colorFillAlter,
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
          {/* T-0182:托管字典隐藏顶部"+ 添加详情"按钮(整托管 = 不能手工增 auto 项) */}
          {!isManaged && (
            <Button
              type="primary"
              size="small"
              icon={<PlusOutlined />}
              disabled={!selectedDict}
              onClick={() => openAdd()}
            >
              {t('dictionary.addDetail')}
            </Button>
          )}
          {isManaged && (
            <Tooltip title={t('dictionary.source.managedTip')}>
              <Tag color="cyan">{t('dictionary.source.managedTag')}</Tag>
            </Tooltip>
          )}
        </Space>
      </div>

      {/* Table */}
      <div style={{ flex: 1, overflow: 'auto' }}>
        {!selectedDict ? (
          <Empty description={t('common.selectHint')} style={{ marginTop: 80 }} />
        ) : (
          <Card
            size="small"
            variant="outlined"
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
              // T-0182 P4 工具栏精简:关掉实时刷新/列设置/密度 — 字典数据量小、
              // 字段固定,这三件套对管理员无价值,反而是视觉噪声。刷新按钮保留。
              hideRealtime
              hideColumnSettings
              hideDensity
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
          <Form.Item name="parentId" label={t('dictionary.detail.parentLabel')}>
            <Select
              allowClear
              placeholder={t('dictionary.detail.parentPlaceholder')}
              options={parentOptions}
              showSearch
              optionFilterProp="label"
            />
          </Form.Item>
          <Form.Item name="label" label={t('dictionary.detail.labelZh')} rules={[{ required: true }]}>
            <Input placeholder={t('dictionary.detail.labelZh')} />
          </Form.Item>
          <Form.Item name="labelEn" label={t('dictionary.detail.labelEn')}>
            <Input placeholder={t('dictionary.detail.labelEn')} />
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
  const { token } = theme.useToken();
  const [selectedDict, setSelectedDict] = useState<Dictionary | null>(null);

  return (
    <App style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <ListPageLayout>
        <div style={{ display: 'flex', flex: 1, minHeight: 0, overflow: 'hidden', background: token.colorBgContainer }}>
          {/* Left panel - dict list */}
          <div style={{ width: 300, flexShrink: 0, minHeight: 0, overflow: 'hidden' }}>
            <DictListPanel
              selectedId={selectedDict?.id ?? null}
              onSelect={(dict) => setSelectedDict(dict)}
            />
          </div>

          {/* Right panel - dict details */}
          <div style={{ flex: 1, minWidth: 0, minHeight: 0, overflow: 'hidden' }}>
            <DictDetailPanel selectedDict={selectedDict} />
          </div>
        </div>
      </ListPageLayout>
    </App>
  );
}
