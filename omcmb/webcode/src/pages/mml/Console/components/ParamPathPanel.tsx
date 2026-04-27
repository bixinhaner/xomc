import { useEffect, useMemo, useRef, useState } from 'react';
import { AutoComplete, Button, Input, Select, Space, Typography } from 'antd';
import { MinusOutlined, PlusOutlined } from '@ant-design/icons';
import type { MMLCommand, MMLOperationType } from '@core/types/mml';
import { resolveOperationType } from '../utils/resolveOperationType';
import { useT } from '@/hooks/useT';

export interface ParamPathChangePayload {
  operationType: string;
  paramPaths: string[];
  // 与 paramPaths 等长，仅 MOD 时使用；其他操作类型回传与路径数等长的空串数组
  // 以便上层无须做长度对齐。
  paramValues: string[];
}

interface ParamPathPanelProps {
  command: MMLCommand | null;
  onChange?: (payload: ParamPathChangePayload) => void;
}

// 裸路径模式仅暴露 MML 控制台需求里明确的四种操作。其他 OP（DSP/ACT/DEA/RST/CLR/UPG）
// 在没有命令绑定的情况下下发语义不清，且后端 service.go 也只接受 LST/DSP/MOD/ADD/RMV，
// 这里直接收口避免出现"选了但执行失败"的死路径。
const RAW_DEFAULT_OPERATIONS: MMLOperationType[] = ['LST', 'MOD', 'ADD', 'RMV'];

const OP_LABELS: Record<string, string> = {
  LST: 'GetParameterValues',
  DSP: 'GetParameterValues',
  MOD: 'SetParameterValues',
  ADD: 'AddObject',
  RMV: 'DeleteObject',
  DEL: 'DeleteObject',
  ACT: 'SetParameterValues',
  DEA: 'SetParameterValues',
  RST: 'Reboot',
  CLR: 'SetParameterValues',
  UPG: 'Download',
};

interface PathRow {
  id: number;
  path: string;
  value: string;
}

let rowIdSeq = 1;
const newRow = (path = '', value = ''): PathRow => ({ id: rowIdSeq++, path, value });

export default function ParamPathPanel({ command, onChange }: ParamPathPanelProps) {
  const t = useT();
  const defaultOperation = useMemo(() => resolveOperationType(command), [command]);

  const operationOptions = useMemo(() => {
    const declared = command?.supportedOperations?.map((o) => o.toUpperCase()) ?? [];
    if (declared.length > 0) return declared;
    return RAW_DEFAULT_OPERATIONS.slice();
  }, [command]);

  const suggestedPaths = useMemo(() => {
    if (!command?.paramPaths) return [];
    return command.paramPaths.map((item) => ({
      value: item.path,
      label: item.label ? `${item.label} (${item.path})` : item.path,
    }));
  }, [command]);

  const [operationType, setOperationType] = useState<string>(defaultOperation);
  const [rows, setRows] = useState<PathRow[]>([newRow()]);

  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  // MOD 需要每行带值，其他操作类型 value 字段不参与 SOAP body 但仍保留在 row 上，
  // 以便用户切换 op 时不丢已经填的值。
  const showValueColumn = operationType === 'MOD';

  useEffect(() => {
    const nextOperation = (operationOptions[0] || defaultOperation).toUpperCase();
    const firstSuggested = command?.paramPaths?.[0]?.path;
    const nextRows = firstSuggested ? [newRow(firstSuggested)] : [newRow()];
    setOperationType(nextOperation);
    setRows(nextRows);
    onChangeRef.current?.({
      operationType: nextOperation,
      paramPaths: nextRows.map((r) => r.path.trim()).filter(Boolean),
      paramValues: nextRows.filter((r) => r.path.trim()).map((r) => r.value),
    });
  }, [command, defaultOperation, operationOptions]);

  const emitChange = (nextOperation: string, nextRows: PathRow[]) => {
    // 同时过滤 path 和 value，让两个数组下标严格对齐——后端按下标平行绑定。
    const trimmed = nextRows.filter((r) => r.path.trim() !== '');
    onChangeRef.current?.({
      operationType: nextOperation,
      paramPaths: trimmed.map((r) => r.path.trim()),
      paramValues: trimmed.map((r) => r.value),
    });
  };

  const handleOperationChange = (nextOperation: string) => {
    setOperationType(nextOperation);
    emitChange(nextOperation, rows);
  };

  const updatePath = (id: number, value: string) => {
    setRows((prev) => {
      const next = prev.map((r) => (r.id === id ? { ...r, path: value } : r));
      emitChange(operationType, next);
      return next;
    });
  };

  const updateValue = (id: number, value: string) => {
    setRows((prev) => {
      const next = prev.map((r) => (r.id === id ? { ...r, value } : r));
      emitChange(operationType, next);
      return next;
    });
  };

  const addPath = (id: number) => {
    setRows((prev) => {
      const idx = prev.findIndex((r) => r.id === id);
      if (idx < 0) return prev;
      const next = [...prev.slice(0, idx + 1), newRow(), ...prev.slice(idx + 1)];
      emitChange(operationType, next);
      return next;
    });
  };

  const removePath = (id: number) => {
    setRows((prev) => {
      if (prev.length <= 1) return prev;
      const next = prev.filter((r) => r.id !== id);
      emitChange(operationType, next);
      return next;
    });
  };

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {!command && (
        <Typography.Text type="secondary">
          {t('mml.console.rawParamPathHint')}
        </Typography.Text>
      )}
      <div>
        <Typography.Text style={{ display: 'block', marginBottom: 8 }}>
          {t('mml.console.operationType')}
        </Typography.Text>
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
        <Typography.Text style={{ display: 'block', marginBottom: 8 }}>
          {t('mml.console.parameterPath')}
          {showValueColumn && (
            <span style={{ marginLeft: 12, color: 'rgba(0, 0, 0, 0.45)' }}>
              · {t('mml.console.parameterValue')}
            </span>
          )}
        </Typography.Text>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8, width: '100%' }}>
          {rows.map((row, index) => (
            <div
              key={row.id}
              style={{
                display: 'flex',
                alignItems: 'flex-start',
                gap: 8,
                width: '100%',
                minWidth: 0,
              }}
            >
              <span style={{ width: 20, lineHeight: '32px', color: 'rgba(0, 0, 0, 0.45)' }}>
                {index + 1}.
              </span>
              <AutoComplete
                value={row.path}
                options={suggestedPaths}
                onChange={(value) => updatePath(row.id, value)}
                placeholder="Device.Services.FAPService.{i}..."
                style={{ flex: showValueColumn ? 2 : 1, minWidth: 0 }}
                filterOption={(inputValue, option) =>
                  String(option?.value ?? '').toLowerCase().includes(inputValue.toLowerCase())
                }
              />
              {showValueColumn && (
                <Input
                  value={row.value}
                  onChange={(event) => updateValue(row.id, event.target.value)}
                  placeholder={t('mml.console.parameterValuePlaceholder')}
                  style={{ flex: 1, minWidth: 0 }}
                />
              )}
              <Button icon={<PlusOutlined />} onClick={() => addPath(row.id)} size="small" />
              <Button
                icon={<MinusOutlined />}
                onClick={() => removePath(row.id)}
                disabled={rows.length <= 1}
                size="small"
              />
            </div>
          ))}
        </div>
      </div>

      <Typography.Text type="secondary">
        {t('mml.console.paramPathFormatHint')}
      </Typography.Text>
    </Space>
  );
}
