import { AutoComplete, Button, Input, Select, Space, Typography } from 'antd';
import { MinusOutlined, PlusOutlined } from '@ant-design/icons';
import type { MMLOperationType } from '@core/types/mml';
import type { RawPathPayload, RawPathRow } from '../types';
import { newRawPathRow as newRow } from '../rawPathRow';

const { Text } = Typography;

// 裸路径模式仅暴露 LST/MOD/ADD/RMV 四种操作（与 V1 ParameterPathCommand 一致）：
// 其它 OP 在无命令绑定时下发语义不清，后端 service.go 也只接受这四种。
const RAW_OPERATIONS: MMLOperationType[] = ['LST', 'MOD', 'ADD', 'RMV'];

// 操作类型 → TR-069 RPC，下拉里并列展示帮助用户理解裸路径会触发哪种 RPC。
const RPC_LABELS: Record<string, string> = {
  LST: 'GetParameterValues',
  MOD: 'SetParameterValues',
  ADD: 'AddObject',
  RMV: 'DeleteObject',
};

interface RawPathPanelProps {
  value: RawPathPayload;
  onChange: (next: RawPathPayload) => void;
  /** 已选命令的参数路径，作为 AutoComplete 建议（可选）。 */
  suggestions?: Array<{ value: string; label: string }>;
}

/**
 * 参数路径指定（裸路径专家模式，设计 §3.9）—— 操作类型下拉 + 可增删的 TR-069 路径行；
 * MOD 每行追加值列；ADD/RMV 协议单次仅作用一个对象，锁单行。当前为 mock：编辑态通过
 * onChange 上抛，由 OperationPanel/index 在执行时驱动结果表格。
 */
export default function RawPathPanel({ value, onChange, suggestions }: RawPathPanelProps) {
  const { operationType, rows } = value;
  const showValue = operationType === 'MOD';
  const singleRowOnly = operationType === 'ADD' || operationType === 'RMV';

  const emitRows = (nextRows: RawPathRow[]) => onChange({ operationType, rows: nextRows });

  const handleOperationChange = (next: MMLOperationType) => {
    const truncate = next === 'ADD' || next === 'RMV';
    const nextRows = truncate && rows.length > 1 ? rows.slice(0, 1) : rows;
    onChange({ operationType: next, rows: nextRows });
  };

  const updatePath = (id: number, path: string) =>
    emitRows(rows.map((r) => (r.id === id ? { ...r, path } : r)));

  const updateValue = (id: number, v: string) =>
    emitRows(rows.map((r) => (r.id === id ? { ...r, value: v } : r)));

  const addRow = (id: number) => {
    const idx = rows.findIndex((r) => r.id === id);
    if (idx < 0) return;
    emitRows([...rows.slice(0, idx + 1), newRow(), ...rows.slice(idx + 1)]);
  };

  const removeRow = (id: number) => {
    if (rows.length <= 1) return;
    emitRows(rows.filter((r) => r.id !== id));
  };

  return (
    <Space direction="vertical" size={14} style={{ width: '100%' }}>
      <div>
        <Text type="secondary">操作类型</Text>
        <Select<MMLOperationType>
          value={operationType}
          onChange={handleOperationChange}
          style={{ width: '100%', marginTop: 6 }}
          options={RAW_OPERATIONS.map((op) => ({
            label: `${op} - ${RPC_LABELS[op]}`,
            value: op,
          }))}
        />
      </div>

      <div>
        <Text type="secondary">参数路径</Text>
        {showValue && <Text type="secondary" style={{ marginLeft: 12 }}>· 参数值</Text>}
        <Space direction="vertical" size={8} style={{ width: '100%', marginTop: 8 }}>
          {rows.map((row, index) => (
            <div key={row.id} style={{ display: 'flex', alignItems: 'flex-start', gap: 8 }}>
              <span style={{ width: 18, lineHeight: '32px', color: 'rgba(0,0,0,0.45)' }}>
                {index + 1}.
              </span>
              <AutoComplete
                value={row.path}
                options={suggestions}
                onChange={(v) => updatePath(row.id, v)}
                placeholder="Device.Services.FAPService.{i}..."
                style={{ flex: showValue ? 2 : 1, minWidth: 0 }}
                filterOption={(input, option) =>
                  String(option?.value ?? '').toLowerCase().includes(input.toLowerCase())
                }
              />
              {showValue && (
                <Input
                  value={row.value}
                  onChange={(e) => updateValue(row.id, e.target.value)}
                  placeholder="请输入参数值"
                  style={{ flex: 1, minWidth: 0 }}
                />
              )}
              <Button
                size="small"
                icon={<PlusOutlined />}
                onClick={() => addRow(row.id)}
                disabled={singleRowOnly}
                title={singleRowOnly ? 'ADD / RMV 单次仅作用一个对象，已锁定单行' : undefined}
              />
              <Button
                size="small"
                icon={<MinusOutlined />}
                onClick={() => removeRow(row.id)}
                disabled={rows.length <= 1}
              />
            </div>
          ))}
        </Space>
      </div>

      <Text type="secondary" style={{ fontSize: 12 }}>
        ⓘ 裸路径直发，不经 sub_field 字典校验，支持 TR-069 参数树路径格式。
        {showValue && ' MOD 需为每条路径填写参数值。'}
      </Text>
      {singleRowOnly && (
        <Text type="warning" style={{ fontSize: 12 }}>
          ADD / RMV 协议规定单次仅作用于一个对象路径，已锁定为单行。
        </Text>
      )}
    </Space>
  );
}
