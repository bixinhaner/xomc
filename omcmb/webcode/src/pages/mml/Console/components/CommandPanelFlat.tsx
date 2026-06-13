/**
 * Task #9: 扁平化命令右侧面板
 *
 * 输入：CommandTreeFlat 选中的 FlatCommand
 * 渲染策略（按 name 前缀派生 op_type）：
 *   - LST: object_path 是 string[]，渲染只读参数路径表格
 *   - MOD: object_path 是 ModParamPath[]，渲染可编辑表单（每个 path 显示 type 和约束）
 *   - ADD/RMV: object_path 是 string，渲染目标对象路径展示框（含 op 标签）
 *
 * 状态：MOD 表单值由本组件本地维护（Record<path, string>）；切换命令时重置。
 *      执行/提交动作不在 Task #9 范围，本组件只负责形态渲染与编辑态收集。
 *      上层若需提交可通过 onValuesChange 订阅 MOD 表单值。
 */
import { useEffect, useMemo, useState } from 'react';
import { Empty, Tag, Table, Form, Input, InputNumber, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type {
  FlatCommand,
  FlatCommandOp,
  ModParamPath,
} from '@core/types/mmlConsole';
import {
  parseOpFromCommandName,
  stripCommandNameOpPrefix,
  isLstObjectPath,
  isModObjectPath,
  isStringObjectPath,
} from '@core/types/mmlConsole';
import { useT } from '@/hooks/useT';

const { Text, Paragraph } = Typography;

const OP_TAG_COLOR: Record<FlatCommandOp, string> = {
  LST: 'blue',
  MOD: 'orange',
  ADD: 'green',
  RMV: 'red',
};

const OP_DESC_KEY: Record<FlatCommandOp, string> = {
  LST: 'mml.console.panelFlat.opDesc.LST',
  MOD: 'mml.console.panelFlat.opDesc.MOD',
  ADD: 'mml.console.panelFlat.opDesc.ADD',
  RMV: 'mml.console.panelFlat.opDesc.RMV',
};

export interface CommandPanelFlatProps {
  command: FlatCommand | null;
  /** MOD 表单值变更回调（key=path, value=用户输入） */
  onValuesChange?: (values: Record<string, string>) => void;
}

/* ----------------------------- LST 渲染 ----------------------------- */

interface LstRow {
  key: string;
  path: string;
}

function LstView({ paths }: { paths: string[] }) {
  const t = useT();
  const columns = useMemo<ColumnsType<LstRow>>(
    () => [
      { title: '#', dataIndex: 'key', width: 56 },
      {
        title: t('mml.console.panelFlat.paramPath'),
        dataIndex: 'path',
        render: (p: string) => (
          <Text code copyable={{ text: p }} style={{ fontSize: 12 }}>
            {p}
          </Text>
        ),
      },
    ],
    [t],
  );
  const rows = useMemo<LstRow[]>(
    () => paths.map((p, i) => ({ key: String(i + 1), path: p })),
    [paths],
  );
  if (rows.length === 0) {
    return <Empty description={t('mml.console.empty.noPaths')} />;
  }
  return (
    <Table
      size="small"
      bordered
      pagination={false}
      columns={columns}
      dataSource={rows}
      scroll={{ y: 'calc(100vh - 460px)' }}
    />
  );
}

/* ----------------------------- MOD 渲染 ----------------------------- */

/** 把 MOD 单条参数的约束格式化为人类可读 hint。 */
function formatModConstraint(p: ModParamPath): string {
  const parts: string[] = [];
  if (typeof p.min === 'number') parts.push(`min=${p.min}`);
  if (typeof p.max === 'number') parts.push(`max=${p.max}`);
  if (typeof p.max_length === 'number') parts.push(`maxLen=${p.max_length}`);
  return parts.join(', ');
}

/**
 * 数值类型（unsignedInt/int/uint/long...）→ InputNumber，其他 → Input。
 * 后端 spec 类型大小写不敏感地匹配，最终走 InputNumber 的均为数值类。
 */
function isNumericType(type: string): boolean {
  return /^(unsigned)?(int|long|short|byte)$/i.test(type);
}

function ModView({
  params,
  values,
  onChange,
}: {
  params: ModParamPath[];
  values: Record<string, string>;
  onChange: (path: string, value: string) => void;
}) {
  const t = useT();
  if (params.length === 0) {
    return <Empty description={t('mml.console.empty.noWritable')} />;
  }
  return (
    <Form layout="vertical" size="small">
      <div style={{ maxHeight: 'calc(100vh - 460px)', overflowY: 'auto', paddingRight: 4 }}>
        {params.map((p) => {
          const constraint = formatModConstraint(p);
          const numeric = isNumericType(p.type);
          return (
            <Form.Item
              key={p.path}
              label={
                <span style={{ display: 'inline-flex', gap: 8, alignItems: 'center' }}>
                  <Text code style={{ fontSize: 12 }}>{p.path}</Text>
                  <Tag color="geekblue" style={{ marginRight: 0 }}>{p.type}</Tag>
                  {constraint && (
                    <Tooltip title={constraint}>
                      <Tag color="default" style={{ marginRight: 0 }}>{constraint}</Tag>
                    </Tooltip>
                  )}
                </span>
              }
            >
              {numeric ? (
                <InputNumber
                  style={{ width: '100%' }}
                  value={values[p.path] === undefined || values[p.path] === '' ? null : Number(values[p.path])}
                  min={p.min}
                  max={p.max}
                  onChange={(v) => onChange(p.path, v === null || v === undefined ? '' : String(v))}
                />
              ) : (
                <Input
                  value={values[p.path] ?? ''}
                  maxLength={p.max_length}
                  onChange={(e) => onChange(p.path, e.target.value)}
                  placeholder={t('mml.console.panelFlat.inputValuePlaceholder', { type: p.type })}
                />
              )}
            </Form.Item>
          );
        })}
      </div>
    </Form>
  );
}

/* ----------------------------- ADD / RMV 渲染 ----------------------------- */

function ObjectPathView({ op, path, t }: { op: 'ADD' | 'RMV'; path: string; t: (id: string) => string }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <Paragraph type="secondary" style={{ marginBottom: 0 }}>
        {op === 'ADD'
          ? t('mml.console.panelFlat.addHint')
          : t('mml.console.panelFlat.rmvHint')}
      </Paragraph>
      <div
        style={{
          padding: '12px 14px',
          border: '1px solid #d9d9d9',
          borderRadius: 6,
          background: '#fafafa',
        }}
      >
        <Text code copyable={{ text: path }} style={{ fontSize: 13 }}>
          {path}
        </Text>
      </div>
    </div>
  );
}

/* ----------------------------- 主组件 ----------------------------- */

export default function CommandPanelFlat({ command, onValuesChange }: CommandPanelFlatProps) {
  const t = useT();
  // MOD 表单本地值；切换命令时重置
  const [modValues, setModValues] = useState<Record<string, string>>({});

  useEffect(() => {
    setModValues({});
  }, [command?.id]);

  if (!command) {
    return <Empty description={t('mml.console.empty.selectFromTree')} style={{ padding: 24 }} />;
  }

  const op = parseOpFromCommandName(command.name);
  const displayName = stripCommandNameOpPrefix(command.name);

  const handleModChange = (path: string, value: string) => {
    setModValues((prev) => {
      const next = { ...prev, [path]: value };
      onValuesChange?.(next);
      return next;
    });
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {/* 头部：op 标签 + 命令中文名 + 描述 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        {op && (
          <Tag color={OP_TAG_COLOR[op]} style={{ fontWeight: 600 }}>
            {op}
          </Tag>
        )}
        <Text strong style={{ fontSize: 15 }}>{displayName}</Text>
      </div>
      {op && (
        <Text type="secondary" style={{ fontSize: 12 }}>
          {t(OP_DESC_KEY[op])}
        </Text>
      )}

      {/* 主体：按 op 派发渲染。type guards 收紧 object_path 的 union 类型，
          guard 与前缀不匹配时（如后端契约异常）显示原始 JSON 兜底。 */}
      {op === 'LST' && isLstObjectPath(command.object_path) && (
        <LstView paths={command.object_path} />
      )}
      {op === 'MOD' && isModObjectPath(command.object_path) && (
        <ModView
          params={command.object_path}
          values={modValues}
          onChange={handleModChange}
        />
      )}
      {(op === 'ADD' || op === 'RMV') && isStringObjectPath(command.object_path) && (
        <ObjectPathView op={op} path={command.object_path} t={t} />
      )}

      {/* 兜底：op 缺失或 object_path 与前缀不匹配（数据异常） */}
      {(!op ||
        (op === 'LST' && !isLstObjectPath(command.object_path)) ||
        (op === 'MOD' && !isModObjectPath(command.object_path)) ||
        ((op === 'ADD' || op === 'RMV') &&
          !isStringObjectPath(command.object_path))) && (
        <Empty
          description={t('mml.console.panelFlat.shapeMismatch', { op: op ?? 'unknown' })}
        />
      )}
    </div>
  );
}
