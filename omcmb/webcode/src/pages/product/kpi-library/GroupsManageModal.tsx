/**
 * GroupsManageModal — KPI 指标分组(功能集)维护 UI (Issue #525, v1 webcode/Antd5)。
 *
 * 一个 Modal 列出某制式下的全部指标分组,支持新建/编辑/删除:
 *   · 新建分组 → 内嵌 Form Modal,id 由前端 crypto.randomUUID() 生成,name 必填。
 *   · 编辑/删除 → 内置组(isBuildIn,后端 is_build_in==='1')禁用 + Tooltip 提示。
 *   · 删除走 Popconfirm 二次确认;成功后 hook 自带 invalidate,列表自动刷新。
 */
import { useState } from 'react';
import {
  Modal,
  Table,
  Button,
  Space,
  Tag,
  Tooltip,
  Popconfirm,
  Form,
  Input,
  message,
} from 'antd';
import type { AxiosError } from 'axios';
import {
  useIndicatorGroups,
  useCreateGroup,
  useUpdateGroup,
  useDeleteGroup,
} from '@core/hooks/api/useIndicatorsLibrary';
import type { DeviceType, IndicatorGroup } from '@core/types/indicatorLibrary';
import GroupTreeSelect from './GroupTreeSelect';
import { useT } from '@/hooks/useT';

interface Props {
  open: boolean;
  onClose: () => void;
  deviceType: DeviceType;
  operatorCode?: string;
}

type FormState =
  | { mode: 'create' }
  | { mode: 'edit'; group: IndicatorGroup }
  | null;

interface GroupFormValues {
  name: string;
  parentId?: string;
  description?: string;
}


function errMsg(e: unknown): string {
  const ax = e as AxiosError<{ msg?: string; message?: string }>;
  return (
    ax.response?.data?.msg ??
    ax.response?.data?.message ??
    (e instanceof Error ? e.message : String(e))
  );
}

export default function GroupsManageModal({
  open,
  onClose,
  deviceType,
  operatorCode,
}: Props) {
  const t = useT();
  // 分组管理列出该制式全部分组(不按 platform 过滤),否则新建的空分组不在平台关联里会看不到。
  const { data, isLoading } = useIndicatorGroups(deviceType, operatorCode);
  const createMut = useCreateGroup();
  const updateMut = useUpdateGroup();
  const deleteMut = useDeleteGroup();

  const [formState, setFormState] = useState<FormState>(null);
  const [form] = Form.useForm<GroupFormValues>();

  // 父分组下拉用 GroupTreeSelect(真·树形,可展开收起);编辑时通过 excludeId
  // 排除自身及子树,避免成环。数据源由组件内部 useIndicatorGroups 获取。
  const editingId = formState?.mode === 'edit' ? formState.group.id : undefined;

  const openCreate = () => {
    form.resetFields();
    setFormState({ mode: 'create' });
  };

  const openEdit = (group: IndicatorGroup) => {
    form.setFieldsValue({
      name: group.name,
      parentId: group.parentId,
      description: group.description,
    });
    setFormState({ mode: 'edit', group });
  };

  const closeForm = () => {
    setFormState(null);
    form.resetFields();
  };

  const submitForm = async () => {
    const values = await form.validateFields();
    if (!formState) return;
    try {
      if (formState.mode === 'create') {
        await createMut.mutateAsync({
          deviceType,
          input: {
            id: crypto.randomUUID().replace(/-/g, ''),
            name: values.name.trim(),
            // 未选父分组 → '0'(后端 buildTree 约定的顶层 root),即新建平级/顶层节点。
            parentId: values.parentId || '0',
            description: values.description,
            operatorCode,
          },
        });
        message.success(t('product.kpi.group.createSuccess'));
      } else {
        await updateMut.mutateAsync({
          deviceType,
          id: formState.group.id,
          input: {
            name: values.name.trim(),
            parentId: values.parentId,
            description: values.description,
          },
        });
        message.success(t('product.kpi.group.updateSuccess'));
      }
      closeForm();
    } catch (e) {
      message.error(errMsg(e));
    }
  };

  const onDelete = async (group: IndicatorGroup) => {
    try {
      await deleteMut.mutateAsync({ deviceType, id: group.id });
      message.success(t('product.kpi.group.deleteSuccess'));
    } catch (e) {
      message.error(errMsg(e));
    }
  };

  const columns = [
    {
      title: t('product.kpi.group.nameLabel'),
      dataIndex: 'name',
      render: (v: string, row: IndicatorGroup) => v || row.id,
    },
    {
      title: t('common.builtin'),
      dataIndex: 'isBuildIn',
      width: 100,
      render: (v: boolean | undefined) =>
        v ? <Tag color="blue">{t('common.builtin')}</Tag> : null,
    },
    {
      title: t('common.operation'),
      key: 'actions',
      width: 160,
      render: (_: unknown, row: IndicatorGroup) => {
        const builtin = Boolean(row.isBuildIn);
        return (
          <Space>
            <Tooltip title={builtin ? t('product.kpi.group.builtinNoEdit') : undefined}>
              <Button
                size="small"
                type="link"
                disabled={builtin}
                onClick={() => openEdit(row)}
              >
                {t('common.edit')}
              </Button>
            </Tooltip>
            <Tooltip title={builtin ? t('product.kpi.group.builtinNoDelete') : undefined}>
              {/* Tooltip 需要可聚焦子节点;disabled 时用 span 包裹按钮保证 hover 生效 */}
              {builtin ? (
                <span>
                  <Button size="small" type="link" danger disabled>
                    {t('common.delete')}
                  </Button>
                </span>
              ) : (
                <Popconfirm
                  title={t('product.kpi.group.confirmDelete')}
                  okText={t('common.yes')}
                  cancelText={t('common.cancel')}
                  onConfirm={() => onDelete(row)}
                >
                  <Button size="small" type="link" danger loading={deleteMut.isPending}>
                    {t('common.delete')}
                  </Button>
                </Popconfirm>
              )}
            </Tooltip>
          </Space>
        );
      },
    },
  ];

  return (
    <Modal
      title={t('product.kpi.group.manage')}
      open={open}
      onCancel={onClose}
      footer={null}
      width={720}
      destroyOnHidden
    >
      <Space style={{ marginBottom: 12 }}>
        <Button type="primary" onClick={openCreate}>
          {t('product.kpi.group.createTitle')}
        </Button>
      </Space>
      {/* 分组是 parent_id 树:直接喂嵌套 data.items,Antd Table 按 children 渲染树形(默认全展开)。 */}
      <Table<IndicatorGroup>
        rowKey="id"
        size="small"
        loading={isLoading}
        dataSource={data?.items ?? []}
        columns={columns}
        pagination={false}
        expandable={{ defaultExpandAllRows: true }}
        locale={{ emptyText: t('product.kpi.group.empty') }}
        scroll={{ y: 360 }}
      />

      <Modal
        title={
          formState?.mode === 'edit'
            ? t('product.kpi.group.editTitle')
            : t('product.kpi.group.createTitle')
        }
        open={Boolean(formState)}
        onCancel={closeForm}
        onOk={submitForm}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
        confirmLoading={createMut.isPending || updateMut.isPending}
        destroyOnHidden
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label={t('product.kpi.group.nameLabel')}
            rules={[{ required: true, message: t('product.kpi.group.nameRequired') }]}
          >
            <Input maxLength={128} />
          </Form.Item>
          <Form.Item name="parentId" label={t('product.kpi.group.parentLabel')}>
            <GroupTreeSelect
              deviceType={deviceType}
              operatorCode={operatorCode}
              excludeId={editingId}
              allowClear
              placeholder={t('product.kpi.group.parentLabel')}
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item name="description" label={t('product.kpi.group.descriptionLabel')}>
            <Input.TextArea rows={3} maxLength={512} />
          </Form.Item>
        </Form>
      </Modal>
    </Modal>
  );
}
