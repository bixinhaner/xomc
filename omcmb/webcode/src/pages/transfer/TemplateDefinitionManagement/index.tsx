import { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Col,
  Descriptions,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  Popconfirm,
  Row,
  Select,
  Space,
  Switch,
  Tabs,
  Tag,
  Typography,
  message,
} from 'antd';
import {
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  PlusOutlined,
} from '@ant-design/icons';

import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import {
  useCreateUnifiedFileTransferTaskType,
  useDeleteUnifiedFileTransferTaskType,
  useUnifiedFileTransferTaskTypes,
  useUpdateUnifiedFileTransferTaskType,
} from '@core/hooks/api/useUnifiedFileTransfer';
import { useProductList } from '@core/hooks/api/useProducts';
import type {
  CreateUnifiedFileTransferTypeInput,
  UnifiedFileTransferTaskType,
  UpdateUnifiedFileTransferTaskTypeInput,
} from '@core/types/unifiedFileTransfer';
import {
  buildCategoryPayload,
  buildCategoryTabs,
  filterTaskTypesForCategory,
  getSoftwareLibraryFileTypeLabel,
  getSoftwareLibraryFileTypeOptions,
  getStepLabels,
  localizeBuiltinCategoryLabel,
  localizeBuiltinDescription,
  localizeBuiltinTypeName,
  TYPE_DRAWER_DEFAULT_STEPS,
} from '../shared';
import type { TaskTypeFormValues } from '../shared';
import { TransferTemplateCard } from '../TransferTemplateCard';

const { Paragraph, Text, Title } = Typography;

export default function TemplateDefinitionManagement() {
  const t = useT();
  const { data: taskTypes = [], isLoading: taskTypesLoading } = useUnifiedFileTransferTaskTypes({
    refetchOnMount: 'always',
  });
  const { data: productsData } = useProductList();
  const products = useMemo(() => productsData?.items ?? [], [productsData]);
  const categories = useMemo(() => buildCategoryTabs(taskTypes), [taskTypes]);
  const [selectedCategory, setSelectedCategory] = useState('');
  const [selectedTypeCode, setSelectedTypeCode] = useState('');
  const [detailType, setDetailType] = useState<UnifiedFileTransferTaskType | null>(null);
  const [detailDrawerOpen, setDetailDrawerOpen] = useState(false);
  const [typeDrawerOpen, setTypeDrawerOpen] = useState(false);
  const [editingType, setEditingType] = useState<UnifiedFileTransferTaskType | null>(null);
  const [typeForm] = Form.useForm<TaskTypeFormValues>();
  const formRpcType = Form.useWatch('rpcType', typeForm);

  const createTaskTypeMutation = useCreateUnifiedFileTransferTaskType();
  const updateTaskTypeMutation = useUpdateUnifiedFileTransferTaskType();
  const deleteTaskTypeMutation = useDeleteUnifiedFileTransferTaskType();

  const filteredTaskTypes = useMemo(
    // #483：buildCategoryTabs 已把 4G/5G/2G 折叠成虚拟分类 device_upgrade，selectedCategory
    // 可能就是它。后端没有 category==='device_upgrade' 的模板（真实是 enb/gnb/gsm_upgrade），
    // 故必须经 filterTaskTypesForCategory 展开成员，不能精确等值——否则虚拟分类下模板恒为空。
    () => filterTaskTypesForCategory(taskTypes, selectedCategory),
    [selectedCategory, taskTypes],
  );
  const builtInTypes = useMemo(
    () => filteredTaskTypes.filter((item) => item.builtIn),
    [filteredTaskTypes],
  );
  const customTypes = useMemo(
    () => filteredTaskTypes.filter((item) => !item.builtIn),
    [filteredTaskTypes],
  );

  const categoryOptions = useMemo(
    () => categories.map((item) => ({
      label: localizeBuiltinCategoryLabel(item.category, item.categoryLabel, t),
      value: item.category,
    })),
    [categories, t],
  );

  // STEP_LABELS / SOFTWARE_LIBRARY_FILE_TYPE_OPTIONS 常量已下线 —— 改用 hook 形式
  const stepLabels = useMemo(() => getStepLabels(t), [t]);
  const softwareLibraryFileTypeOptions = useMemo(() => getSoftwareLibraryFileTypeOptions(t), [t]);

  // #492：模板「适用范围」改为按产品英文名选择（来自产品中心-产品管理目录），
  // 取代旧的 productClass 自由文本。label 带制式 tag 便于辨识。
  const productOptions = useMemo(
    () => products
      .map((p) => ({ label: `${p.name} (${p.tech})`, value: p.name }))
      .sort((left, right) => left.label.localeCompare(right.label, 'zh-CN')),
    [products],
  );
  const productTechByName = useMemo(() => {
    const map = new Map<string, string>();
    products.forEach((p) => map.set(p.name, p.tech));
    return map;
  }, [products]);
  // #492：「适用产品」空 = 适用全部产品。展示/编辑层统一把"空"呈现为"全选所有产品名"，
  // 保存时若仍是全选则回存空（保持"不限、未来新产品自动纳入"语义）。
  const allProductNames = useMemo(() => products.map((p) => p.name), [products]);
  // 制式按所选产品自动派生（只读展示），不再让用户手填 techHint。
  const formProducts = Form.useWatch('products', typeForm);
  const derivedTech = useMemo(() => {
    const set = new Set<string>();
    (formProducts ?? []).forEach((name) => {
      const tech = productTechByName.get(name);
      if (tech) set.add(tech);
    });
    return Array.from(set);
  }, [formProducts, productTechByName]);

  useEffect(() => {
    if (categories.length === 0) {
      setSelectedCategory('');
      return;
    }
    if (!categories.some((item) => item.category === selectedCategory)) {
      setSelectedCategory(categories[0].category);
    }
  }, [categories, selectedCategory]);

  useEffect(() => {
    const preferredType = builtInTypes[0] ?? customTypes[0];
    if (!preferredType) {
      setSelectedTypeCode('');
      return;
    }
    if (!filteredTaskTypes.some((item) => item.typeCode === selectedTypeCode)) {
      setSelectedTypeCode(preferredType.typeCode);
    }
  }, [builtInTypes, customTypes, filteredTaskTypes, selectedTypeCode]);

  const selectedType = useMemo(
    () => filteredTaskTypes.find((item) => item.typeCode === selectedTypeCode),
    [filteredTaskTypes, selectedTypeCode],
  );

  const openTypeDetailDrawer = (record: UnifiedFileTransferTaskType) => {
    setSelectedTypeCode(record.typeCode);
    setDetailType(record);
    setDetailDrawerOpen(true);
  };

  const openTypeDrawer = (record?: UnifiedFileTransferTaskType) => {
    setEditingType(record ?? null);
    if (record) {
      typeForm.setFieldsValue({
        categorySelection: record.category,
        categoryCustomLabel: undefined,
        displayName: record.displayName,
        description: record.description,
        rpcType: record.rpcType,
        stepChain: record.stepChain,
        postTcEventCode: record.postTcEventCode,
        enabled: record.enabled,
        // 空（=适用全部）→ 编辑器里呈现为全选所有产品名，与升级模板格式统一。
        products: (record.products && record.products.length > 0) ? record.products : allProductNames,
        fileType: record.fileType,
        fileTypeEditable: record.fileTypeEditable,
        firmwareFileType: record.firmwareFileType,
        delaySeconds: record.delaySeconds,
      });
    } else {
      typeForm.setFieldsValue({
        categorySelection: selectedCategory,
        categoryCustomLabel: undefined,
        rpcType: 'DOWNLOAD',
        enabled: true,
        // 新建模板默认适用全部产品（全选展示；保存时全选回存空=不限）。
        products: allProductNames,
        fileTypeEditable: true,
        firmwareFileType: undefined,
        delaySeconds: 0,
        stepChain: TYPE_DRAWER_DEFAULT_STEPS,
      });
    }
    setTypeDrawerOpen(true);
  };

  const handleSaveType = async () => {
    // 防止双击重复提交：mutation 进行中直接忽略。
    // validateFields 是异步的，仅靠按钮 loading 不能阻止第二次点击在 isPending=false 窗口期内进入。
    if (createTaskTypeMutation.isPending || updateTaskTypeMutation.isPending) {
      return;
    }
    const values = await typeForm.validateFields();
    let categoryPayload;
    try {
      categoryPayload = buildCategoryPayload(values, categories, editingType);
    } catch (e) {
      // buildCategoryPayload 通过 throw i18n key 表达 "缺业务 Tab" 校验失败；前端在此 catch 翻译展示。
      const key = e instanceof Error ? e.message : 'ufte.template.pickCategory';
      void message.warning(t(key));
      return;
    }
    // #492：全选所有产品 = 适用全部 → 回存空（不限、未来新产品自动纳入）；子集则存具体列表。
    const selectedProducts = values.products ?? [];
    const isAllProducts = allProductNames.length > 0 && allProductNames.every((n) => selectedProducts.includes(n));
    const productsToSave = isAllProducts ? [] : selectedProducts;
    const payload: CreateUnifiedFileTransferTypeInput = {
      category: categoryPayload.category,
      categoryLabel: categoryPayload.categoryLabel,
      displayName: values.displayName,
      description: values.description,
      rpcType: values.rpcType,
      stepChain: values.stepChain,
      postTcEventCode: values.postTcEventCode,
      enabled: values.enabled,
      // #492：模板编辑改用 products（适用产品名）。platformScope 不再在表单里编辑，
      // 透传 editingType 原值保留（作为 product_scope 为空时的后端回退口径）。
      platformScope: editingType?.platformScope ?? [],
      products: productsToSave,
      fileType: values.fileType,
      fileTypeLabel: values.fileType,
      fileTypeEditable: values.fileTypeEditable,
      firmwareFileType: values.firmwareFileType,
      urlTemplate: editingType?.urlTemplate,
      targetFileNameTemplate: editingType?.targetFileNameTemplate,
      fileNameTemplate: editingType?.fileNameTemplate,
      fileSizeField: editingType?.fileSizeField,
      checksumField: editingType?.checksumField,
      rawMode: editingType?.rawMode,
      delaySeconds: values.delaySeconds,
      transportPath: editingType?.transportPath,
    };

    let savedType: UnifiedFileTransferTaskType;
    if (editingType) {
      savedType = await updateTaskTypeMutation.mutateAsync({
        typeCode: editingType.typeCode,
        ...payload,
      } as UpdateUnifiedFileTransferTaskTypeInput);
      void message.success(t('ufte.template.msg.updated'));
    } else {
      savedType = await createTaskTypeMutation.mutateAsync(payload);
      void message.success(t('ufte.template.msg.created'));
    }

    setSelectedCategory(savedType.category);
    setSelectedTypeCode(savedType.typeCode);
    setTypeDrawerOpen(false);
    setEditingType(null);
    typeForm.resetFields();
  };

  const handleDeleteType = async (taskType: UnifiedFileTransferTaskType) => {
    await deleteTaskTypeMutation.mutateAsync(taskType.typeCode);
    if (detailType?.typeCode === taskType.typeCode) {
      setDetailDrawerOpen(false);
      setDetailType(null);
    }
    if (editingType?.typeCode === taskType.typeCode) {
      setTypeDrawerOpen(false);
      setEditingType(null);
      typeForm.resetFields();
    }
    void message.success(t('ufte.template.msg.deleted'));
  };

  const renderTemplateSection = (
    title: string,
    items: UnifiedFileTransferTaskType[],
    emptyDescription: string,
  ) => (
    <Card title={title} loading={taskTypesLoading}>
      {items.length === 0 ? (
        <Empty description={emptyDescription} image={Empty.PRESENTED_IMAGE_SIMPLE} />
      ) : (
        <div style={{ maxHeight: 460, overflowY: 'auto', overflowX: 'hidden', paddingRight: 4 }}>
          <Space orientation="vertical" size={12} style={{ width: '100%' }}>
            {items.map((taskType) => {
              const active = selectedType?.typeCode === taskType.typeCode;
              return (
                <Space key={taskType.typeCode} orientation="vertical" size={8} style={{ width: '100%' }}>
                  <div onClick={() => setSelectedTypeCode(taskType.typeCode)} role="presentation" style={{ cursor: 'pointer' }}>
                    <TransferTemplateCard taskType={taskType} active={active} />
                  </div>
                  <Space wrap>
                    <Button type="link" icon={<EyeOutlined />} onClick={() => openTypeDetailDrawer(taskType)}>
                      {t('ufte.action.viewDetail')}
                    </Button>
                    <Button type="link" icon={<EditOutlined />} onClick={() => openTypeDrawer(taskType)}>
                      {taskType.builtIn ? t('ufte.action.adjustTemplate') : t('ufte.action.editTemplate')}
                    </Button>
                    {!taskType.builtIn ? (
                      <Popconfirm
                        title={t('ufte.confirm.deleteTemplate')}
                        description={t('ufte.confirm.deleteTemplateDesc')}
                        onConfirm={() => void handleDeleteType(taskType)}
                      >
                        <Button type="link" icon={<DeleteOutlined />} danger loading={deleteTaskTypeMutation.isPending}>
                          {t('ufte.action.deleteTemplate')}
                        </Button>
                      </Popconfirm>
                    ) : null}
                  </Space>
                </Space>
              );
            })}
          </Space>
        </div>
      )}
    </Card>
  );

  return (
    <ListPageLayout
      title={t('ufte.page.templateManagement')}
      extra={(
        <Button type="primary" icon={<PlusOutlined />} onClick={() => openTypeDrawer()}>
          {t('ufte.action.newCustomTemplate')}
        </Button>
      )}
    >
      <Space orientation="vertical" size={16} style={{ width: '100%' }}>
        <Card>
          <Space orientation="vertical" size={12} style={{ width: '100%' }}>
            <Title level={4} style={{ margin: 0 }}>{t('ufte.page.templateConfig')}</Title>
            <Tabs
              activeKey={selectedCategory}
              items={categories.map((item) => ({
                key: item.category,
                label: localizeBuiltinCategoryLabel(item.category, item.categoryLabel, t),
              }))}
              onChange={(key) => setSelectedCategory(key)}
            />
          </Space>
        </Card>

        <Row gutter={[16, 16]}>
          <Col xs={24} xl={14}>
            {renderTemplateSection(
              t('ufte.template.builtInGroup'),
              builtInTypes,
              t('ufte.template.emptyBuiltIn'),
            )}
          </Col>
          <Col xs={24} xl={10}>
            {renderTemplateSection(
              t('ufte.template.customGroup'),
              customTypes,
              t('ufte.template.emptyCustom'),
            )}
          </Col>
        </Row>
      </Space>

      <Drawer
        title={detailType ? t('ufte.drawer.templateDetailWithName', { name: localizeBuiltinTypeName(detailType.typeCode, detailType.displayName, t) }) : t('ufte.drawer.templateDetail')}
        size={560}
        open={detailDrawerOpen}
        onClose={() => setDetailDrawerOpen(false)}
        destroyOnHidden
        extra={detailType ? (
          <Space>
            {!detailType.builtIn ? (
              <Popconfirm
                title={t('ufte.confirm.deleteTemplate')}
                description={t('ufte.confirm.deleteTemplateDesc')}
                onConfirm={() => void handleDeleteType(detailType)}
              >
                <Button icon={<DeleteOutlined />} danger loading={deleteTaskTypeMutation.isPending}>
                  {t('ufte.action.deleteTemplate')}
                </Button>
              </Popconfirm>
            ) : null}
            <Button type="primary" onClick={() => {
              setDetailDrawerOpen(false);
              openTypeDrawer(detailType);
            }}>
              {detailType.builtIn ? t('ufte.action.adjustTemplate') : t('ufte.action.editTemplate')}
            </Button>
          </Space>
        ) : null}
      >
        {detailType ? (
          <Space orientation="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions column={2} size="small" bordered>
              <Descriptions.Item label={t('ufte.template.businessView')}>{localizeBuiltinCategoryLabel(detailType.category, detailType.categoryLabel, t)}</Descriptions.Item>
              <Descriptions.Item label={t('ufte.template.typeCode')}>{detailType.typeCode}</Descriptions.Item>
              <Descriptions.Item label={t('ufte.template.rpcType')}>{detailType.rpcType}</Descriptions.Item>
              <Descriptions.Item label={t('ufte.template.fileType')}>{detailType.fileType}</Descriptions.Item>
              <Descriptions.Item label={t('ufte.template.softLib')}>{getSoftwareLibraryFileTypeLabel(detailType.firmwareFileType, t)}</Descriptions.Item>
              <Descriptions.Item label={t('ufte.template.permCode')}>{detailType.permissionCode}</Descriptions.Item>
              <Descriptions.Item label={t('ufte.template.products')}>
                {/* #492：空 = 适用全部产品 → 展示「全部产品」+ 全部产品名，与升级模板格式统一。 */}
                {(detailType.products ?? []).length > 0
                  ? (detailType.products ?? []).join(' / ')
                  : (allProductNames.length > 0
                      ? `${t('ufte.template.allProducts')}（${allProductNames.length}）：${allProductNames.join(' / ')}`
                      : t('ufte.template.allProducts'))}
              </Descriptions.Item>
              <Descriptions.Item label={t('ufte.template.lastEditor')}>{detailType.lastEditor}</Descriptions.Item>
              <Descriptions.Item label={t('ufte.template.postEvent')}>{detailType.postTcEventCode || '-'}</Descriptions.Item>
            </Descriptions>
            <Paragraph style={{ marginBottom: 0 }}>{localizeBuiltinDescription(detailType.typeCode, detailType.description, t)}</Paragraph>
            <Descriptions column={2} size="small" bordered>
              <Descriptions.Item label={t('ufte.template.fileTypeEditable')}>
                <Tag color={detailType.fileTypeEditable ? 'green' : 'default'}>
                  {detailType.fileTypeEditable ? t('ufte.template.fileTypeEditable.yes') : t('ufte.template.fileTypeEditable.no')}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label={t('ufte.template.delaySeconds')}>{detailType.delaySeconds ?? 0}</Descriptions.Item>
            </Descriptions>
            <div>
              <Text strong>{t('ufte.template.stepChain')}</Text>
              <div style={{ marginTop: 8 }}>
                <Space wrap size={[6, 8]}>
                  {detailType.stepChain.map((stepId, index) => (
                    <Tag key={stepId} color={index < 2 ? 'blue' : index === detailType.stepChain.length - 1 ? 'purple' : 'default'}>
                      {index + 1}. {stepLabels[stepId]}
                    </Tag>
                  ))}
                </Space>
              </div>
            </div>
          </Space>
        ) : null}
      </Drawer>

      <Drawer
        title={editingType ? t('ufte.drawer.editTemplateWithName', { name: localizeBuiltinTypeName(editingType.typeCode, editingType.displayName, t) }) : t('ufte.drawer.newCustomTemplate')}
        size={520}
        open={typeDrawerOpen}
        onClose={() => {
          setTypeDrawerOpen(false);
          setEditingType(null);
        }}
        destroyOnHidden
        extra={(
          <Space>
            <Button onClick={() => {
              setTypeDrawerOpen(false);
              setEditingType(null);
            }}>
              {t('common.cancel')}
            </Button>
            <Button
              type="primary"
              loading={createTaskTypeMutation.isPending || updateTaskTypeMutation.isPending || deleteTaskTypeMutation.isPending}
              onClick={() => void handleSaveType()}
            >
              {editingType ? t('common.save') : t('ufte.action.create')}
            </Button>
          </Space>
        )}
      >
        <Form form={typeForm} layout="vertical">
          <Form.Item label={t('ufte.template.existingCategoryTab')} name="categorySelection">
            <Select allowClear options={categoryOptions} placeholder={t('ufte.template.existingCategoryTab.placeholder')} />
          </Form.Item>
          <Form.Item label={t('ufte.template.newCategoryLabel')} name="categoryCustomLabel">
            <Input placeholder={t('ufte.template.newCategoryLabel.placeholder')} />
          </Form.Item>
          <Form.Item label={t('ufte.template.displayName')} name="displayName" rules={[{ required: true, message: t('ufte.template.displayName.required') }]}>
            <Input placeholder={t('ufte.template.displayName.placeholder')} />
          </Form.Item>
          <Form.Item label={t('ufte.template.description')} name="description" rules={[{ required: true, message: t('ufte.template.description.required') }]}>
            <Input.TextArea rows={3} placeholder={t('ufte.template.description.placeholder')} />
          </Form.Item>
          <Form.Item label={t('ufte.template.rpcType')} name="rpcType" rules={[{ required: true, message: t('ufte.template.rpcType.required') }]}>
            <Select
              options={[
                { label: 'DOWNLOAD', value: 'DOWNLOAD' },
                { label: 'UPLOAD', value: 'UPLOAD' },
                { label: 'SET_PARAM_VALUES', value: 'SET_PARAM_VALUES' },
              ]}
            />
          </Form.Item>
          <Form.Item label={t('ufte.template.stepChain')} name="stepChain" rules={[{ required: true, message: t('ufte.template.stepChain.required') }]}>
            <Select
              mode="multiple"
              options={Object.entries(stepLabels).map(([value, label]) => ({ value, label }))}
              placeholder={t('ufte.template.stepChain.placeholder')}
            />
          </Form.Item>
          {/* #492：模板「适用范围」改为按产品英文名（产品中心-产品管理）多选；制式按所选产品自动派生（只读）。 */}
          <Form.Item
            label={t('ufte.template.products')}
            name="products"
            extra={derivedTech.length > 0
              ? `${t('ufte.template.derivedTech')}: ${derivedTech.join(' / ').toUpperCase()}`
              : undefined}
          >
            <Select
              mode="multiple"
              showSearch
              optionFilterProp="label"
              options={productOptions}
              placeholder={t('ufte.template.products.placeholder')}
            />
          </Form.Item>
          <Form.Item label={t('ufte.template.fileType')} name="fileType" rules={[{ required: true, message: t('ufte.template.fileType.required') }]}>
                <Input placeholder={t('ufte.template.fileType.placeholder')} />
          </Form.Item>
          <Form.Item label={t('ufte.template.softLib')} name="firmwareFileType" extra={formRpcType === 'DOWNLOAD' ? t('ufte.template.softLib.extraDownload') : t('ufte.template.softLib.extraOther')}>
            <Select
              allowClear
              placeholder={formRpcType === 'DOWNLOAD' ? t('ufte.template.softLib.placeholder') : t('ufte.template.softLib.placeholderDisabled')}
              options={softwareLibraryFileTypeOptions}
              disabled={formRpcType !== 'DOWNLOAD'}
            />
          </Form.Item>
          <Form.Item label={t('ufte.template.delaySeconds')} name="delaySeconds">
            <InputNumber min={0} max={86400} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label={t('ufte.template.fileTypeEditableLabel')} name="fileTypeEditable" valuePropName="checked">
            <Switch checkedChildren={t('ufte.template.switchEditable')} unCheckedChildren={t('ufte.template.switchFixed')} />
          </Form.Item>
          <Form.Item label={t('ufte.template.enabledStatus')} name="enabled" valuePropName="checked">
            <Switch checkedChildren={t('ufte.template.switchEnabled')} unCheckedChildren={t('ufte.template.switchDisabled')} />
          </Form.Item>
          <Form.Item label={t('ufte.template.postTcEventCode')} name="postTcEventCode">
            <Input placeholder={t('ufte.template.postTcEventCode.placeholder')} />
          </Form.Item>
        </Form>
      </Drawer>
    </ListPageLayout>
  );
}