import { useMemo } from 'react';
import { Form, Input, Select } from 'antd';
import type { FormInstance } from 'antd';
import { useT } from '@/hooks/useT';

// T-0090 子项 ①：操作类型差异化 — 选 MOD 时显示「修改值入口」TextArea。
// 抽出为公共组件供 AddTemplateModal 和子任务 d 私有命令页共用，
// 规避 R-NEW-3（组件抽象不足导致复制粘贴）。
//
// 调用方负责在外层 <Form form={form}> 内渲染本组件，并通过 form prop 让
// 组件读到当前 operationType 用于条件渲染修改值入口。

export interface OperationTypeWithModifyProps {
  form: FormInstance;
  operationTypeName?: string;
  modifyValuesName?: string;
  options?: Array<{ label: string; value: string }>;
}

export default function OperationTypeWithModify({
  form,
  operationTypeName = 'operationType',
  modifyValuesName = 'modifyValues',
  options,
}: OperationTypeWithModifyProps) {
  const t = useT();
  const selectedType = Form.useWatch(operationTypeName, form);

  const defaultOptions = useMemo(
    () => [
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
    ],
    [t],
  );

  return (
    <>
      <Form.Item
        name={operationTypeName}
        label={t('mml.console.operationType')}
        rules={[{ required: true, message: t('mml.console.selectOperationType') }]}
      >
        <Select
          placeholder={t('mml.console.selectOperationType')}
          options={options ?? defaultOptions}
        />
      </Form.Item>

      {selectedType === 'MOD' && (
        <Form.Item
          name={modifyValuesName}
          label={t('mml.console.modifyValuesLabel')}
          rules={[{ required: true, message: t('mml.console.modifyValuesRequired') }]}
          extra={t('mml.console.modifyValuesTip')}
        >
          <Input.TextArea
            rows={4}
            maxLength={1000}
            placeholder={t('mml.console.modifyValuesPlaceholder')}
            style={{ fontFamily: "'SFMono-Regular', Consolas, monospace", fontSize: 12 }}
          />
        </Form.Item>
      )}
    </>
  );
}
