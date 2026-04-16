import { useEffect } from 'react';
import { Modal, Form, Input, Select, message } from 'antd';
import { useCreateMMLTemplate } from '@/hooks/api/useMML';
import { useAllMMLCommands } from '@/hooks/api/useMML';
import { useDictionary } from '@/hooks/api/useSystem';
import type { MMLTemplate } from '@/types/mml';

interface AddTemplateModalProps {
  open: boolean;
  scope: 'public' | 'private';
  onClose: () => void;
  onSuccess: () => void;
}

const OPERATION_TYPE_OPTIONS = [
  { label: 'LST - 查询', value: 'LST' },
  { label: 'MOD - 修改', value: 'MOD' },
  { label: 'ADD - 增加', value: 'ADD' },
  { label: 'RMV - 删除', value: 'RMV' },
  { label: 'DSP - 显示', value: 'DSP' },
  { label: 'ACT - 激活', value: 'ACT' },
  { label: 'DEA - 去激活', value: 'DEA' },
  { label: 'RST - 重启', value: 'RST' },
  { label: 'CLR - 清除', value: 'CLR' },
];

export default function AddTemplateModal({ open, scope, onClose, onSuccess }: AddTemplateModalProps) {
  const [form] = Form.useForm();
  const createMutation = useCreateMMLTemplate();
  const { data: commandsResponse } = useAllMMLCommands();
  const { data: categoryDict } = useDictionary('mml_command_category');

  const commands = commandsResponse ?? [];

  const categoryOptions = (() => {
    const dictDetails = categoryDict?.sysDictionaryDetails;
    if (dictDetails && dictDetails.length > 0) {
      return dictDetails.map((d) => ({ label: d.label, value: d.value }));
    }
    return [...new Set(commands.map((c) => c.category))].map((v) => ({ label: v, value: v }));
  })();

  // Derive command code options from existing commands
  const commandCodeOptions = commands.map((c) => ({
    label: `${c.commandName} (${c.commandCode})`,
    value: c.commandCode,
  }));

  useEffect(() => {
    if (open) {
      form.resetFields();
    }
  }, [form, open]);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();

      const template: Omit<MMLTemplate, 'id' | 'creator' | 'createdAt' | 'updatedAt'> = {
        templateName: values.templateName,
        commandCode: values.commandCode,
        operationType: values.operationType,
        templateScope: scope,
        categoryGroup: values.categoryGroup ?? '',
        parameters: {},
        paramPaths: [],
        description: values.description ?? '',
        productTypes: values.productTypes ?? [],
      };

      await createMutation.mutateAsync(template);
      message.success(scope === 'public' ? '公有命令创建成功' : '私有命令创建成功');
      onSuccess();
      onClose();
    } catch {
      // validation errors are shown inline
    }
  };

  return (
    <Modal
      title={scope === 'public' ? '新增公有命令' : '新增私有命令'}
      open={open}
      onOk={handleSubmit}
      onCancel={onClose}
      confirmLoading={createMutation.isPending}
      width={520}
      destroyOnClose
    >
      <Form form={form} layout="vertical" size="small">
        <Form.Item
          name="templateName"
          label="命令名称"
          rules={[{ required: true, message: '请输入命令名称' }]}
        >
          <Input placeholder="请输入命令名称" maxLength={200} />
        </Form.Item>

        <Form.Item
          name="commandCode"
          label="命令编码"
          rules={[{ required: true, message: '请选择或输入命令编码' }]}
        >
          <Select
            placeholder="选择已有命令或自定义"
            showSearch
            allowClear
            options={commandCodeOptions}
            filterOption={(input, option) =>
              (option?.label as string)?.toLowerCase().includes(input.toLowerCase()) ??
              (option?.value as string)?.toLowerCase().includes(input.toLowerCase())
            }
          />
        </Form.Item>

        <Form.Item
          name="operationType"
          label="操作类型"
          rules={[{ required: true, message: '请选择操作类型' }]}
        >
          <Select placeholder="请选择操作类型" options={OPERATION_TYPE_OPTIONS} />
        </Form.Item>

        <Form.Item name="categoryGroup" label="所属分类">
          <Select placeholder="请选择分类" allowClear options={categoryOptions} />
        </Form.Item>

        <Form.Item name="description" label="描述">
          <Input.TextArea rows={2} placeholder="命令描述（可选）" maxLength={500} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
