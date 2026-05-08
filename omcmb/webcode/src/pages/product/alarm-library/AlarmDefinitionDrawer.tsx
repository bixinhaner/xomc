import { useEffect } from 'react';
import { Drawer, Form, Input, Select, Switch, Space, Button, message, Tag } from 'antd';
import {
  useCreateAlarmDefinition,
  useUpdateAlarmDefinition,
  useAlarmSeverityLevels,
} from '@core/hooks/api/useAlarmDefinitions';
import type {
  AlarmDefinition,
  CreateAlarmDefinitionInput,
  UpdateAlarmDefinitionInput,
} from '@core/types/alarmDefinition';

interface Props {
  open: boolean;
  definition: AlarmDefinition | null;
  onClose: () => void;
}

interface FormValues {
  identifier: string;
  neType: string;
  cnName: string;
  enName: string;
  severityCode: number;
  eventType: string;
  cnProbableCause?: string;
  enProbableCause?: string;
  cnSuggestion?: string;
  enSuggestion?: string;
  isShow: boolean;
}

export default function AlarmDefinitionDrawer({ open, definition, onClose }: Props) {
  const isEdit = Boolean(definition);
  const [form] = Form.useForm<FormValues>();
  const { data: sevData } = useAlarmSeverityLevels();
  const createMut = useCreateAlarmDefinition();
  const updateMut = useUpdateAlarmDefinition();

  useEffect(() => {
    if (!open) return;
    if (definition) {
      form.setFieldsValue({
        identifier: definition.identifier,
        neType: definition.neType,
        cnName: definition.cnName,
        enName: definition.enName,
        severityCode: definition.severityCode,
        eventType: definition.eventType || '',
        cnProbableCause: definition.cnProbableCause,
        enProbableCause: definition.enProbableCause,
        cnSuggestion: definition.cnSuggestion,
        enSuggestion: definition.enSuggestion,
        isShow: definition.isShow,
      });
    } else {
      form.resetFields();
      form.setFieldsValue({ isShow: true, severityCode: 4 });
    }
  }, [open, definition, form]);

  const handleSave = async () => {
    try {
      const v = await form.validateFields();
      if (isEdit && definition) {
        const input: UpdateAlarmDefinitionInput = {
          neType: v.neType,
          cnName: v.cnName,
          enName: v.enName,
          severityCode: v.severityCode,
          eventType: v.eventType,
          cnProbableCause: v.cnProbableCause,
          enProbableCause: v.enProbableCause,
          cnSuggestion: v.cnSuggestion,
          enSuggestion: v.enSuggestion,
          isShow: v.isShow,
        };
        await updateMut.mutateAsync({ identifier: definition.identifier, input });
        message.success('已更新');
      } else {
        const input: CreateAlarmDefinitionInput = {
          identifier: v.identifier,
          neType: v.neType,
          cnName: v.cnName,
          enName: v.enName,
          severityCode: v.severityCode,
          eventType: v.eventType,
          cnProbableCause: v.cnProbableCause,
          enProbableCause: v.enProbableCause,
          cnSuggestion: v.cnSuggestion,
          enSuggestion: v.enSuggestion,
          isShow: v.isShow,
        };
        await createMut.mutateAsync(input);
        message.success('已创建');
      }
      onClose();
    } catch (e) {
      const msg = (e as Error).message;
      if (msg) message.error(msg);
    }
  };

  return (
    <Drawer
      title={
        isEdit ? (
          <Space>
            告警定义详情：<Tag color="blue">{definition?.identifier}</Tag>
            {definition?.isUnknown && <Tag color="warning">未识别 fallback</Tag>}
          </Space>
        ) : (
          '新增告警定义'
        )
      }
      placement="right"
      width={680}
      open={open}
      onClose={onClose}
      destroyOnClose
      footer={
        <Space style={{ float: 'right' }}>
          <Button onClick={onClose}>取消</Button>
          <Button
            type="primary"
            loading={createMut.isPending || updateMut.isPending}
            onClick={() => void handleSave()}
          >
            保存
          </Button>
        </Space>
      }
    >
      <Form<FormValues> form={form} layout="vertical">
        <Form.Item name="identifier" label="标识符 (identifier)" rules={[{ required: true }]}>
          <Input disabled={isEdit} placeholder="如 101001 (主键，不可改)" />
        </Form.Item>
        <Form.Item name="neType" label="网元类型 (ne_type)" rules={[{ required: true }]}>
          <Input placeholder="eNodeB / gNodeB / BTS / ..." />
        </Form.Item>
        <Form.Item name="cnName" label="中文名" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item name="enName" label="英文名" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item name="severityCode" label="严重级别" rules={[{ required: true }]}>
          <Select
            options={(sevData?.items || []).map((s) => ({
              label: `${s.code} - ${s.cnName} / ${s.enName}`,
              value: s.code,
            }))}
          />
        </Form.Item>
        <Form.Item name="eventType" label="事件类型 (event_type)">
          <Input placeholder="communication / qualityOfService / processingError / ..." />
        </Form.Item>
        <Form.Item name="cnProbableCause" label="可能原因（中）">
          <Input.TextArea rows={2} />
        </Form.Item>
        <Form.Item name="enProbableCause" label="Probable Cause (EN)">
          <Input.TextArea rows={2} />
        </Form.Item>
        <Form.Item name="cnSuggestion" label="处置建议（中）">
          <Input.TextArea rows={2} />
        </Form.Item>
        <Form.Item name="enSuggestion" label="Suggestion (EN)">
          <Input.TextArea rows={2} />
        </Form.Item>
        <Form.Item name="isShow" label="UI 可见" valuePropName="checked">
          <Switch />
        </Form.Item>
      </Form>
    </Drawer>
  );
}
