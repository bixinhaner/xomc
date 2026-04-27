import { useEffect, useMemo, useRef, useState } from 'react';
import { AutoComplete, Button, Select, Space, Typography } from 'antd';
import { MinusOutlined, PlusOutlined } from '@ant-design/icons';
import type { MMLCommand, MMLOperationType } from '@core/types/mml';
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

// 全量 MML OperationType → TR-069 RPC 方法映射。命令未声明 supportedOperations
// 时 fallback 到这份全集，保证下拉始终可选其他 RPC 类型（to-do-list #1）。
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

// 默认展示顺序：从高频到低频，便于用户快速扫读。
const DEFAULT_OPERATIONS: MMLOperationType[] = [
  'LST', 'DSP', 'MOD', 'ADD', 'RMV', 'ACT', 'DEA', 'RST', 'CLR', 'UPG',
];

// 用"稳定 id + path"模型描述每一行；避免像以前用 `${index}-${path}` 做 key
// 导致输入 path 时键变化 → 组件重挂载 → 光标丢失 → 看起来像"不能输入"
// （to-do-list #2）。
interface PathRow {
  id: number;
  path: string;
}

let rowIdSeq = 1;
const newRow = (path = ''): PathRow => ({ id: rowIdSeq++, path });

export default function ParamPathPanel({ command, onChange }: ParamPathPanelProps) {
  const t = useT();
  const defaultOperation = useMemo(() => resolveOperationType(command), [command]);

  const operationOptions = useMemo(() => {
    const declared = command?.supportedOperations?.map((o) => o.toUpperCase()) ?? [];
    if (declared.length > 0) return declared;
    return DEFAULT_OPERATIONS.slice();
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

  // onChange 通过 ref 读取，避免 useEffect 依赖它导致父组件每次 render
  // 都重置行状态。
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  // 命令切换时重置：操作类型回默认、路径清空或预填第一个建议路径。
  useEffect(() => {
    const nextOperation = (operationOptions[0] || defaultOperation).toUpperCase();
    const firstSuggested = command?.paramPaths?.[0]?.path;
    const nextRows = firstSuggested ? [newRow(firstSuggested)] : [newRow()];
    setOperationType(nextOperation);
    setRows(nextRows);
    onChangeRef.current?.({
      operationType: nextOperation,
      paramPaths: nextRows.map((r) => r.path.trim()).filter(Boolean),
    });
  }, [command, defaultOperation, operationOptions]);

  const emitChange = (nextOperation: string, nextRows: PathRow[]) => {
    onChangeRef.current?.({
      operationType: nextOperation,
      paramPaths: nextRows.map((r) => r.path.trim()).filter(Boolean),
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

  // 不再要求"先选命令"。无命令时面板进入"裸路径直接执行"模式：
  //   - 操作类型下拉 fallback 到 DEFAULT_OPERATIONS
  //   - 路径输入无 AutoComplete 建议（命令树没绑定 → suggestedPaths 为空）
  //   - 后端仅接受 LST/DSP（其它操作类型在 service 层会拒）
  // 设计依据：MML 控制台需求 #1。

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
        </Typography.Text>
        {/* 使用原生 flex 容器而非 <Space>：antd Space 会把每个子项再包一层
            .ant-space-item（无 flex-grow），导致设在 AutoComplete 上的
            `flex:1` 不生效——AutoComplete 会按内部 rc-select 的 search-input
            长度自收缩，表现为"输入时输入框变小、输入无法正常赋值"。
            见 to-do-list 本轮 MML#1。 */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8, width: '100%' }}>
          {rows.map((row, index) => (
            <div
              key={row.id}
              style={{
                display: 'flex',
                alignItems: 'flex-start',
                gap: 8,
                width: '100%',
                minWidth: 0, // 让 flex:1 在溢出时能压缩而非撑开父容器
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
                style={{ flex: 1, minWidth: 0 }}
                filterOption={(inputValue, option) =>
                  String(option?.value ?? '').toLowerCase().includes(inputValue.toLowerCase())
                }
              />
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
