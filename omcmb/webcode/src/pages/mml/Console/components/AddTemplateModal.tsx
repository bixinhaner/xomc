import { useEffect, useMemo, useState } from 'react';
import { Modal, Form, Input, Select, Button, message, Space, InputNumber, Switch } from 'antd';
import { useCreateMMLTemplate } from '@core/hooks/api/useMML';
import { useAllMMLCommands } from '@core/hooks/api/useMML';
import { useDictionary } from '@core/hooks/api/useSystem';
import type { MMLCustomCommand, MMLCommand, MMLParam } from '@core/types/mml';
import { useT } from '@/hooks/useT';

interface AddTemplateModalProps {
  open: boolean;
  scope: 'public' | 'private';
  onClose: () => void;
  onSuccess: () => void;
  onSaveAndExecute?: (template: Omit<MMLCustomCommand, 'id' | 'creator' | 'createdAt' | 'updatedAt'>) => void;
}

export default function AddTemplateModal({ open, scope, onClose, onSuccess, onSaveAndExecute }: AddTemplateModalProps) {
  const t = useT();

  const OPERATION_TYPE_OPTIONS = [
    { label: t('mml.console.opTypeLST'), value: 'LST' },
    { label: t('mml.console.opTypeMOD'), value: 'MOD' },
    { label: t('mml.console.opTypeADD'), value: 'ADD' },
    { label: t('mml.console.opTypeRMV'), value: 'RMV' },
    { label: t('mml.console.opTypeDSP'), value: 'DSP' },
    { label: t('mml.console.opTypeACT'), value: 'ACT' },
    { label: t('mml.console.opTypeDEA'), value: 'DEA' },
    { label: t('mml.console.opTypeRST'), value: 'RST' },
    { label: t('mml.console.opTypeCLR'), value: 'CLR' },
    { label: t('mml.console.opTypeUPG'), value: 'UPG' },
  ];

  const [form] = Form.useForm();
  const createMutation = useCreateMMLTemplate();
  const { data: commandsResponse } = useAllMMLCommands();
  const { data: categoryDict } = useDictionary('mml_command_category');
  const { data: productTypeDict } = useDictionary('product_type');
  const [paramValues, setParamValues] = useState<Record<string, string | number | boolean>>({});

  const commands = commandsResponse ?? [];

  const categoryOptions = useMemo(() => {
    const dictDetails = categoryDict?.sysDictionaryDetails;
    if (dictDetails && dictDetails.length > 0) {
      return dictDetails.map((d) => ({ label: d.label, value: d.value }));
    }
    return [...new Set(commands.map((c) => c.category))].map((v) => ({ label: v, value: v }));
  }, [categoryDict, commands]);

  const productTypeOptions = useMemo(() => {
    const details = productTypeDict?.sysDictionaryDetails;
    if (details && details.length > 0) {
      return details.map((d) => ({ label: d.label, value: d.value }));
    }
    return [];
  }, [productTypeDict]);

  // Derive command code options from existing commands
  const commandCodeOptions = commands.map((c) => ({
    label: `${c.commandName} (${c.commandCode})`,
    value: c.commandCode,
  }));

  // Find the matching command definition for param editing
  const selectedCommandCode = Form.useWatch('commandCode', form);
  const matchedCommand = useMemo<MMLCommand | null>(() => {
    if (!selectedCommandCode) return null;
    return commands.find((c) => c.commandCode === selectedCommandCode) ?? null;
  }, [commands, selectedCommandCode]);

  // Params from matched command for the parameter editing form
  const editableParams = useMemo<MMLParam[]>(() => {
    if (!matchedCommand?.params?.length) return [];
    return matchedCommand.params;
  }, [matchedCommand]);

  useEffect(() => {
    if (open) {
      form.resetFields();
      setParamValues({});
    }
  }, [form, open]);

  // Auto-fill operationType when commandCode changes
  useEffect(() => {
    if (matchedCommand?.operationType) {
      form.setFieldValue('operationType', matchedCommand.operationType);
    }
  }, [matchedCommand, form]);

  const handleParamChange = (paramName: string, value: string | number | boolean | undefined) => {
    setParamValues((prev) => {
      const next = { ...prev };
      if (value === undefined || value === '') {
        delete next[paramName];
      } else {
        next[paramName] = value;
      }
      return next;
    });
  };

  const handleSubmit = async (andExecute: boolean) => {
    try {
      const values = await form.validateFields();

      const template: Omit<MMLCustomCommand, 'id' | 'creator' | 'createdAt' | 'updatedAt'> = {
        commandName: values.templateName,
        commandCode: values.commandCode,
        operationType: values.operationType,
        commandScope: scope,
        categoryGroup: values.categoryGroup ?? '',
        parameters: paramValues,
        paramPaths: matchedCommand?.paramPaths?.map((p) => p.path) ?? [],
        description: values.description ?? '',
        productTypes: values.productTypes ?? [],
      };

      await createMutation.mutateAsync(template);
      message.success(scope === 'public' ? t('mml.console.publicTemplateCreated') : t('mml.console.privateTemplateCreated'));
      onSuccess();

      if (andExecute && onSaveAndExecute) {
        onSaveAndExecute(template);
      }

      onClose();
    } catch {
      // validation errors are shown inline
    }
  };

  const renderParamControl = (param: MMLParam) => {
    const currentValue = paramValues[param.name];
    switch (param.type) {
      case 'number':
      case 'unsignedInt':
        return (
          <InputNumber
            style={{ width: '100%' }}
            min={param.minValue ?? (param.type === 'unsignedInt' ? 0 : undefined)}
            max={param.maxValue}
            value={currentValue as number | undefined}
            placeholder={param.description || param.name}
            onChange={(v) => handleParamChange(param.name, v ?? undefined)}
          />
        );
      case 'boolean':
        return (
          <Switch
            checked={Boolean(currentValue)}
            onChange={(checked) => handleParamChange(param.name, checked)}
          />
        );
      case 'enum': {
        const enumOptions = param.options?.length
          ? param.options
          : (param.enumValues ?? []).map((item) => ({ label: item, value: item }));
        return (
          <Select
            allowClear
            value={currentValue as string | number | undefined}
            options={enumOptions}
            placeholder={param.description || t('common.pleaseSelect')}
            onChange={(v) => handleParamChange(param.name, v)}
            onClear={() => handleParamChange(param.name, undefined)}
          />
        );
      }
      default:
        return (
          <Input
            value={currentValue as string | undefined}
            placeholder={param.description || param.name}
            onChange={(e) => handleParamChange(param.name, e.target.value)}
          />
        );
    }
  };

  return (
    <Modal
      title={scope === 'public' ? t('mml.console.addPublicTemplate') : t('mml.console.addPrivateTemplate')}
      open={open}
      onCancel={onClose}
      width={600}
      destroyOnClose
      footer={
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button onClick={() => void handleSubmit(false)} loading={createMutation.isPending}>
            {t('mml.console.saveOnly')}
          </Button>
          {onSaveAndExecute && (
            <Button type="primary" onClick={() => void handleSubmit(true)} loading={createMutation.isPending}>
              {t('mml.console.saveAndExecute')}
            </Button>
          )}
        </div>
      }
    >
      <Form form={form} layout="vertical" size="small">
        <Form.Item
          name="templateName"
          label={t('mml.console.commandName')}
          rules={[{ required: true, message: t('mml.console.inputCommandName') }]}
        >
          <Input placeholder={t('mml.console.inputCommandName')} maxLength={200} />
        </Form.Item>

        <Form.Item
          name="commandCode"
          label={t('mml.console.commandCode')}
          rules={[{ required: true, message: t('mml.console.selectOrInputCommandCode') }]}
        >
          <Select
            placeholder={t('mml.console.selectExistingOrCustom')}
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
          label={t('mml.console.operationType')}
          rules={[{ required: true, message: t('mml.console.selectOperationType') }]}
        >
          <Select placeholder={t('mml.console.selectOperationType')} options={OPERATION_TYPE_OPTIONS} />
        </Form.Item>

        <Form.Item name="categoryGroup" label={t('mml.console.categoryGroup')}>
          <Select placeholder={t('mml.console.selectCategory')} allowClear options={categoryOptions} />
        </Form.Item>

        <Form.Item name="productTypes" label={t('mml.console.productTypes')}>
          <Select
            mode="multiple"
            placeholder={t('mml.console.selectProductTypes')}
            allowClear
            options={productTypeOptions}
          />
        </Form.Item>

        {/* Parameter editing section when matched command has params */}
        {editableParams.length > 0 && (
          <div style={{ marginBottom: 16 }}>
            <div style={{ marginBottom: 8, fontWeight: 500, color: '#333' }}>{t('mml.console.paramConfig')}</div>
            <Space direction="vertical" size={8} style={{ width: '100%' }}>
              {editableParams.map((param) => (
                <div key={param.name} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <span style={{ width: 120, flexShrink: 0, fontSize: 13, textAlign: 'right' }}>
                    {param.name}
                    {param.required && <span style={{ color: '#ff4d4f' }}> *</span>}
                  </span>
                  <div style={{ flex: 1 }}>{renderParamControl(param)}</div>
                </div>
              ))}
            </Space>
          </div>
        )}

        <Form.Item name="description" label={t('common.description')}>
          <Input.TextArea rows={2} placeholder={t('mml.console.commandDescriptionOptional')} maxLength={500} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
