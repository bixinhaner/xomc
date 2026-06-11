import { AutoComplete, Button, Input, Select, Space, Typography } from 'antd';
import { MinusOutlined, PlusOutlined } from '@ant-design/icons';
import type { MMLOperationType } from '@core/types/mml';
import type { RawPathPayload, RawPathRow } from '../types';
import { newRawPathRow as newRow } from '../rawPathRow';
import { rawPathPlaceholder, validateRawPath } from '../rawPathValidate';

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
 * onChange 上抛，由 ConfigParamsModal/index 在执行时驱动结果表格。
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
    <Space orientation="vertical" size={14} style={{ width: '100%' }}>
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
        {/* 提示文案紧跟「参数路径」之后（§需求 2）；ADD/RMV 给出 TR-069 对象路径形态提示（plan A）。 */}
        <Text type="secondary" style={{ display: 'block', fontSize: 12, marginTop: 4 }}>
          ⓘ 裸路径直发，支持 TR-069 参数树路径格式（须填具体实例号，不支持 {'{i}'} 占位符）。
          {operationType === 'ADD' && ' ADD：路径填到对象表层级、以 . 结尾，末级不带实例号（设备自动分配）。'}
          {(operationType === 'RMV') && ' RMV：路径须以 .<实例号>. 结尾，指定要删除的实例。'}
          {showValue && ' MOD 需为每条路径填写参数值。'}
        </Text>
        <Space orientation="vertical" size={8} style={{ width: '100%', marginTop: 8 }}>
          {rows.map((row, index) => {
            const pathError = validateRawPath(operationType, row.path);
            return (
              <div key={row.id}>
                <div style={{ display: 'flex', alignItems: 'flex-start', gap: 8 }}>
                  <span style={{ width: 18, lineHeight: '32px', color: 'rgba(0,0,0,0.45)' }}>
                    {index + 1}.
                  </span>
                  <AutoComplete
                    value={row.path}
                    options={suggestions}
                    onChange={(v) => updatePath(row.id, v)}
                    placeholder={rawPathPlaceholder(operationType)}
                    status={pathError ? 'error' : undefined}
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
                {pathError && (
                  <Text type="danger" style={{ display: 'block', fontSize: 12, marginLeft: 26 }}>
                    {pathError}
                  </Text>
                )}
              </div>
            );
          })}
        </Space>
      </div>
    </Space>
  );
}
