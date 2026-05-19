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
import {
  useCreateUnifiedFileTransferTaskType,
  useDeleteUnifiedFileTransferTaskType,
  useUnifiedFileTransferTaskTypes,
  useUpdateUnifiedFileTransferTaskType,
} from '@core/hooks/api/useUnifiedFileTransfer';
import { useProductClasses } from '@core/hooks/api/useDevices';
import type {
  CreateUnifiedFileTransferTypeInput,
  UnifiedFileTransferTaskType,
  UpdateUnifiedFileTransferTaskTypeInput,
} from '@core/types/unifiedFileTransfer';
import {
  buildCategoryPayload,
  buildCategoryTabs,
  getSoftwareLibraryFileTypeLabel,
  SOFTWARE_LIBRARY_FILE_TYPE_OPTIONS,
  STEP_LABELS,
  TransferTemplateCard,
  TYPE_DRAWER_DEFAULT_STEPS,
} from '../shared';
import type { TaskTypeFormValues } from '../shared';

const { Paragraph, Text, Title } = Typography;

export default function TemplateDefinitionManagement() {
  const { data: taskTypes = [], isLoading: taskTypesLoading } = useUnifiedFileTransferTaskTypes({
    refetchOnMount: 'always',
  });
  const { data: productClasses = [] } = useProductClasses();
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
    () => taskTypes.filter((item) => item.category === selectedCategory),
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
    () => categories.map((item) => ({ label: item.categoryLabel, value: item.category })),
    [categories],
  );

  const platformScopeOptions = useMemo(() => {
    const values = new Set<string>(productClasses);
    editingType?.platformScope.forEach((entry) => values.add(entry));
    return Array.from(values)
      .filter(Boolean)
      .sort((left, right) => left.localeCompare(right, 'zh-CN'))
      .map((item) => ({ label: item, value: item }));
  }, [editingType?.platformScope, productClasses]);

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
        platformScope: record.platformScope,
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
        platformScope: [],
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
    const categoryPayload = buildCategoryPayload(values, categories, editingType);
    const payload: CreateUnifiedFileTransferTypeInput = {
      category: categoryPayload.category,
      categoryLabel: categoryPayload.categoryLabel,
      displayName: values.displayName,
      description: values.description,
      rpcType: values.rpcType,
      stepChain: values.stepChain,
      postTcEventCode: values.postTcEventCode,
      enabled: values.enabled,
      platformScope: values.platformScope,
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
      void message.success('模板已更新。');
    } else {
      savedType = await createTaskTypeMutation.mutateAsync(payload);
      void message.success('自定义模板已创建。');
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
    void message.success('自定义模板已删除。');
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
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            {items.map((taskType) => {
              const active = selectedType?.typeCode === taskType.typeCode;
              return (
                <Space key={taskType.typeCode} direction="vertical" size={8} style={{ width: '100%' }}>
                  <div onClick={() => setSelectedTypeCode(taskType.typeCode)} role="presentation" style={{ cursor: 'pointer' }}>
                    <TransferTemplateCard taskType={taskType} active={active} />
                  </div>
                  <Space wrap>
                    <Button type="link" icon={<EyeOutlined />} onClick={() => openTypeDetailDrawer(taskType)}>
                      查看详情
                    </Button>
                    <Button type="link" icon={<EditOutlined />} onClick={() => openTypeDrawer(taskType)}>
                      {taskType.builtIn ? '调整模板' : '编辑模板'}
                    </Button>
                    {!taskType.builtIn ? (
                      <Popconfirm
                        title="确认删除这个自定义模板吗？"
                        description="删除后将无法恢复。"
                        onConfirm={() => void handleDeleteType(taskType)}
                      >
                        <Button type="link" icon={<DeleteOutlined />} danger loading={deleteTaskTypeMutation.isPending}>
                          删除模板
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
      title="传输模板管理"
      extra={(
        <Button type="primary" icon={<PlusOutlined />} onClick={() => openTypeDrawer()}>
          新增自定义模板
        </Button>
      )}
    >
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <Card>
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <Title level={4} style={{ margin: 0 }}>模板配置</Title>
            <Tabs
              activeKey={selectedCategory}
              items={categories.map((item) => ({ key: item.category, label: item.categoryLabel }))}
              onChange={(key) => setSelectedCategory(key)}
            />
          </Space>
        </Card>

        <Row gutter={[16, 16]}>
          <Col xs={24} xl={14}>
            {renderTemplateSection(
              '内置模板',
              builtInTypes,
              '当前业务暂无内置模板',
            )}
          </Col>
          <Col xs={24} xl={10}>
            {renderTemplateSection(
              '自定义模板',
              customTypes,
              '当前业务暂无自定义模板',
            )}
          </Col>
        </Row>
      </Space>

      <Drawer
        title={detailType ? `模板详情 · ${detailType.displayName}` : '模板详情'}
        width={560}
        open={detailDrawerOpen}
        onClose={() => setDetailDrawerOpen(false)}
        destroyOnClose
        extra={detailType ? (
          <Space>
            {!detailType.builtIn ? (
              <Popconfirm
                title="确认删除这个自定义模板吗？"
                description="删除后将无法恢复。"
                onConfirm={() => void handleDeleteType(detailType)}
              >
                <Button icon={<DeleteOutlined />} danger loading={deleteTaskTypeMutation.isPending}>
                  删除模板
                </Button>
              </Popconfirm>
            ) : null}
            <Button type="primary" onClick={() => {
              setDetailDrawerOpen(false);
              openTypeDrawer(detailType);
            }}>
              {detailType.builtIn ? '调整模板' : '编辑模板'}
            </Button>
          </Space>
        ) : null}
      >
        {detailType ? (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions column={2} size="small" bordered>
              <Descriptions.Item label="业务视图">{detailType.categoryLabel}</Descriptions.Item>
              <Descriptions.Item label="类型编码">{detailType.typeCode}</Descriptions.Item>
              <Descriptions.Item label="RPC 类型">{detailType.rpcType}</Descriptions.Item>
              <Descriptions.Item label="FileType">{detailType.fileType}</Descriptions.Item>
              <Descriptions.Item label="软件库分类">{getSoftwareLibraryFileTypeLabel(detailType.firmwareFileType)}</Descriptions.Item>
              <Descriptions.Item label="权限编码">{detailType.permissionCode}</Descriptions.Item>
              <Descriptions.Item label="平台范围">{detailType.platformScope.join(' / ')}</Descriptions.Item>
              <Descriptions.Item label="最近编辑人">{detailType.lastEditor}</Descriptions.Item>
              <Descriptions.Item label="30天任务量">{detailType.taskCount30d}</Descriptions.Item>
              <Descriptions.Item label="30天成功率">{detailType.successRate30d}%</Descriptions.Item>
              <Descriptions.Item label="后置事件">{detailType.postTcEventCode || '-'}</Descriptions.Item>
            </Descriptions>
            <Paragraph style={{ marginBottom: 0 }}>{detailType.description}</Paragraph>
            <Descriptions column={2} size="small" bordered>
              <Descriptions.Item label="FileType 可编辑">
                <Tag color={detailType.fileTypeEditable ? 'green' : 'default'}>
                  {detailType.fileTypeEditable ? '支持' : '固定'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="DelaySeconds">{detailType.delaySeconds ?? 0}</Descriptions.Item>
            </Descriptions>
            <div>
              <Text strong>步骤链</Text>
              <div style={{ marginTop: 8 }}>
                <Space wrap size={[6, 8]}>
                  {detailType.stepChain.map((stepId, index) => (
                    <Tag key={stepId} color={index < 2 ? 'blue' : index === detailType.stepChain.length - 1 ? 'purple' : 'default'}>
                      {index + 1}. {STEP_LABELS[stepId]}
                    </Tag>
                  ))}
                </Space>
              </div>
            </div>
          </Space>
        ) : null}
      </Drawer>

      <Drawer
        title={editingType ? `编辑模板 · ${editingType.displayName}` : '新增自定义模板'}
        width={520}
        open={typeDrawerOpen}
        onClose={() => {
          setTypeDrawerOpen(false);
          setEditingType(null);
        }}
        destroyOnClose
        extra={(
          <Space>
            <Button onClick={() => {
              setTypeDrawerOpen(false);
              setEditingType(null);
            }}>
              取消
            </Button>
            <Button
              type="primary"
              loading={createTaskTypeMutation.isPending || updateTaskTypeMutation.isPending || deleteTaskTypeMutation.isPending}
              onClick={() => void handleSaveType()}
            >
              {editingType ? '保存' : '创建'}
            </Button>
          </Space>
        )}
      >
        <Form form={typeForm} layout="vertical">
          <Form.Item label="已有业务 Tab" name="categorySelection">
            <Select allowClear options={categoryOptions} placeholder="已有的业务 Tab 可直接复用" />
          </Form.Item>
          <Form.Item label="新增 Tab 名称" name="categoryCustomLabel">
            <Input placeholder="如果没有合适的业务 Tab，可直接输入新名称，例如：诊断包分发" />
          </Form.Item>
          <Form.Item label="显示名称" name="displayName" rules={[{ required: true, message: '请输入类型名称' }]}> 
            <Input placeholder="例如：诊断包采集" />
          </Form.Item>
          <Form.Item label="功能描述" name="description" rules={[{ required: true, message: '请输入功能描述' }]}> 
            <Input.TextArea rows={3} placeholder="说明这个类型想统一哪类文件交互，以及为什么需要独立类型。" />
          </Form.Item>
          <Form.Item label="RPC 类型" name="rpcType" rules={[{ required: true, message: '请选择 RPC 类型' }]}> 
            <Select
              options={[
                { label: 'DOWNLOAD', value: 'DOWNLOAD' },
                { label: 'UPLOAD', value: 'UPLOAD' },
                { label: 'SET_PARAM_VALUES', value: 'SET_PARAM_VALUES' },
              ]}
            />
          </Form.Item>
          <Form.Item label="步骤链" name="stepChain" rules={[{ required: true, message: '请选择至少一个步骤' }]}> 
            <Select
              mode="multiple"
              options={Object.entries(STEP_LABELS).map(([value, label]) => ({ value, label }))}
              placeholder="按顺序选择步骤"
            />
          </Form.Item>
          <Form.Item label="平台范围" name="platformScope" rules={[{ required: true, message: '请至少输入一个平台范围' }]}> 
            <Select
              mode="tags"
              showSearch
              optionFilterProp="label"
              options={platformScopeOptions}
              placeholder="输入产品类型关键字，可从设备库联想选择，例如：FAP/BU1810、QAFA、BBU-XSS"
            />
          </Form.Item>
          <Form.Item label="FileType" name="fileType" rules={[{ required: true, message: '请输入 FileType' }]}> 
                <Input placeholder="例如：1 Firmware Upgrade Image / 3 Vendor Configuration File / Firmware Upgrade Fpga" />
          </Form.Item>
          <Form.Item label="软件库分类" name="firmwareFileType" extra={formRpcType === 'DOWNLOAD' ? '用于关联软件管理中的镜像/PATCH/FPGA 文件分类，避免按文件名猜测。' : '仅 DOWNLOAD 模板需要配置；其他 RPC 类型会忽略该值。'}>
            <Select
              allowClear
              placeholder={formRpcType === 'DOWNLOAD' ? '请选择软件库分类' : '当前 RPC 类型无需配置'}
              options={SOFTWARE_LIBRARY_FILE_TYPE_OPTIONS}
              disabled={formRpcType !== 'DOWNLOAD'}
            />
          </Form.Item>
          <Form.Item label="DelaySeconds" name="delaySeconds">
            <InputNumber min={0} max={86400} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="允许页面直接调整 FileType" name="fileTypeEditable" valuePropName="checked">
            <Switch checkedChildren="可调" unCheckedChildren="固定" />
          </Form.Item>
          <Form.Item label="启用状态" name="enabled" valuePropName="checked">
            <Switch checkedChildren="启用" unCheckedChildren="停用" />
          </Form.Item>
          <Form.Item label="TransferComplete 后事件码" name="postTcEventCode">
            <Input placeholder="例如：102 UPGRADE FINISH；无则留空" />
          </Form.Item>
        </Form>
      </Drawer>
    </ListPageLayout>
  );
}