import { useMemo } from 'react';
import { Form, Select } from 'antd';
import { useT } from '@/hooks/useT';

// 操作类型选择器（供 AddTemplateModal 等「新增/编辑命令定义」表单复用）。
//
// 历史：曾在选 MOD 时附带一个「修改值入口」TextArea（把 K=V 烘焙进 parameters）。
// 现按需求移除——命令定义阶段只声明参数 PATH，具体修改值在 console 执行时再录入，
// 故任何操作类型（含 MOD）都不再展示修改值入口。

export interface OperationTypeWithModifyProps {
  operationTypeName?: string;
  options?: Array<{ label: string; value: string }>;
}

export default function OperationTypeWithModify({
  operationTypeName = 'operationType',
  options,
}: OperationTypeWithModifyProps) {
  const t = useT();

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
  );
}
