import { useEffect, useMemo, useState } from 'react';
import { AutoComplete, Button, Empty, Select, Space, Typography } from 'antd';
import { MinusOutlined, PlusOutlined } from '@ant-design/icons';
import type { MMLCommand } from '@/types/mml';
import { resolveOperationType } from '../utils/resolveOperationType';
import { useT } from '@/hooks/useT';

export interface ParamPathChangePayload {
  operationType: string;
  paramPaths: string[];
}

interface ParamPathPanelProps {
  command: MMLCommand | null;
  onChange?: (payload: ParamPathChangePayload) => void;
}

const OP_LABELS: Record<string, string> = {
  LST: 'GetParameterValues',
  DSP: 'GetParameterValues',
  MOD: 'SetParameterValues',
  ADD: 'AddObject',
  RMV: 'DeleteObject',
  DEL: 'DeleteObject',
  ACT: 'SetParameterValues',
  DEA: 'SetParameterValues',
  RST: 'SetParameterValues',
  CLR: 'SetParameterValues',
};

export default function ParamPathPanel({ command, onChange }: ParamPathPanelProps) {
  const t = useT();
  const defaultOperation = useMemo(() => resolveOperationType(command), [command]);
  const operationOptions = useMemo(() => {
    const operations = command?.supportedOperations?.length
      ? command.supportedOperations
      : [defaultOperation];

    return operations.map((item) => item.toUpperCase());
  }, [command, defaultOperation]);

  const suggestedPaths = useMemo(() => {
    if (!command?.paramPaths) {
      return [];
    }

    return command.paramPaths.map((item) => ({
      value: item.path,
      label: item.label ? `${item.label} (${item.path})` : item.path,
    }));
  }, [command]);

  const [operationType, setOperationType] = useState<string>(defaultOperation);
  const [paths, setPaths] = useState<string[]>(['']);

  useEffect(() => {
    const nextOperation = operationOptions[0] || defaultOperation;
    const nextPaths = command?.paramPaths?.length
      ? [command.paramPaths[0].path]
      : [''];

    setOperationType(nextOperation);
    setPaths(nextPaths);
    onChange?.({ operationType: nextOperation, paramPaths: nextPaths.filter((item) => item.trim()) });
  }, [command, defaultOperation, onChange, operationOptions]);

  const emitChange = (nextOperation: string, nextPaths: string[]) => {
    onChange?.({
      operationType: nextOperation,
      paramPaths: nextPaths.map((item) => item.trim()).filter(Boolean),
    });
  };

  const handleOperationChange = (nextOperation: string) => {
    setOperationType(nextOperation);
    emitChange(nextOperation, paths);
  };

  const updatePath = (index: number, value: string) => {
    const nextPaths = [...paths];
    nextPaths[index] = value;
    setPaths(nextPaths);
    emitChange(operationType, nextPaths);
  };

  const addPath = (index: number) => {
    const nextPaths = [...paths];
    nextPaths.splice(index + 1, 0, '');
    setPaths(nextPaths);
    emitChange(operationType, nextPaths);
  };

  const removePath = (index: number) => {
    if (paths.length <= 1) {
      return;
    }

    const nextPaths = paths.filter((_, currentIndex) => currentIndex !== index);
    setPaths(nextPaths);
    emitChange(operationType, nextPaths);
  };

  if (!command) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('mml.console.selectCommandFirst')} />;
  }

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <div>
        <Typography.Text style={{ display: 'block', marginBottom: 8 }}>Operation Type</Typography.Text>
        <Select
          value={operationType}
          onChange={handleOperationChange}
          style={{ width: '100%' }}
          options={operationOptions.map((item) => ({
            label: `${item} - ${OP_LABELS[item] || item}`,
            value: item,
          }))}
        />
      </div>

      <div>
        <Typography.Text style={{ display: 'block', marginBottom: 8 }}>Parameter Path</Typography.Text>
        <Space direction="vertical" size={8} style={{ width: '100%' }}>
          {paths.map((path, index) => (
            <Space key={`${index}-${path}`} style={{ width: '100%' }} align="start">
              <span style={{ width: 20, lineHeight: '32px', color: 'rgba(0, 0, 0, 0.45)' }}>{index + 1}.</span>
              <AutoComplete
                value={path}
                options={suggestedPaths}
                onChange={(value) => updatePath(index, value)}
                placeholder="Device.Services.FAPService.{i}..."
                style={{ flex: 1 }}
                filterOption={(inputValue, option) =>
                  String(option?.value ?? '').toLowerCase().includes(inputValue.toLowerCase())
                }
              />
              <Button
                icon={<PlusOutlined />}
                onClick={() => addPath(index)}
                size="small"
              />
              <Button
                icon={<MinusOutlined />}
                onClick={() => removePath(index)}
                disabled={paths.length <= 1}
                size="small"
              />
            </Space>
          ))}
        </Space>
      </div>

      <Typography.Text type="secondary">
        {t('mml.console.paramPathFormatHint')}
      </Typography.Text>
    </Space>
  );
}
