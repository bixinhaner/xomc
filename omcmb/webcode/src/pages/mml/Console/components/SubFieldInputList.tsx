import { useMemo } from 'react';
import type { CSSProperties } from 'react';
import { Input, Tag, Tooltip, Empty } from 'antd';
import { NumberOutlined } from '@ant-design/icons';
import AccessTypeTag from './AccessTypeTag';
import { useMmlConsoleStore } from '@core/store/mmlConsoleStore';
import type { Statement } from '@core/types/mmlConsole';
import { useT } from '@/hooks/useT';

export interface SubFieldInputListProps {
  statement: Statement;
}

/** path 含 `.{i}.` 多实例占位符时渲染的提示图标（D35）。 */
function multiInstanceHint(tr069Path: string, tip: string) {
  if (!tr069Path.includes('.{i}.')) return null;
  return (
    <Tooltip title={tip}>
      <NumberOutlined style={{ color: '#fa8c16' }} aria-label="multi-instance" />
    </Tooltip>
  );
}

export default function SubFieldInputList({ statement }: SubFieldInputListProps) {
  const t = useT();
  const setValue = useMmlConsoleStore((s) => s.setValue);

  const sortedFields = useMemo(() => {
    const list = [...statement.subFields].sort((a, b) => a.sortOrder - b.sortOrder);
    // MOD 过滤 READ_ONLY（PRD §7.3）；ADD 保留全部字段
    if (statement.operationType === 'MOD') {
      return list.filter((sf) => sf.accessType !== 'READ_ONLY');
    }
    return list;
  }, [statement.subFields, statement.operationType]);

  // 固定高度容器（spec：操作面板限制 path 区域 10 行可视高度，<10 行也保留空间，
  // >10 行出现滚动条；MOD/ADD 行更高 — input 32px + flex gap 10 + 可选 constraint 行 ≈ 42-62px，
  // 取 420px = 10 行不带 constraint 的基准，constraint 多时自然出现滚动）。
  const VIEWPORT_HEIGHT = 420;
  const VIEWPORT_STYLE: CSSProperties = {
    height: VIEWPORT_HEIGHT,
    overflowY: 'auto',
    border: '1px solid #f0f0f0',
    borderRadius: 4,
    padding: 8,
  };

  if (sortedFields.length === 0) {
    return (
      <div style={{ ...VIEWPORT_STYLE, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Empty description={false} />
      </div>
    );
  }

  return (
    <div style={{ ...VIEWPORT_STYLE, display: 'flex', flexDirection: 'column', gap: 10 }}>
      {sortedFields.map((sf) => {
        const currentValue = statement.values[sf.mmlCode] ?? '';
        return (
          <div key={sf.id} style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ flex: '0 0 200px', display: 'flex', flexDirection: 'column', minWidth: 0 }}>
                <span style={{ fontWeight: 500 }}>
                  {sf.isRequired && (
                    <span style={{ color: '#ff4d4f', marginRight: 4 }} aria-label="required">
                      *
                    </span>
                  )}
                  {sf.label}
                </span>
                {sf.description && (
                  <span style={{ color: '#bfbfbf', fontSize: 11, marginTop: 1 }}>
                    {sf.description}
                  </span>
                )}
              </span>
              <Input
                value={currentValue}
                onChange={(e) => setValue(statement.uid, sf.mmlCode, e.target.value)}
                placeholder={sf.defaultValue ?? ''}
                style={{ flex: 1 }}
                status={sf.isRequired && !currentValue ? 'warning' : undefined}
              />
              {multiInstanceHint(sf.tr069Path, t('mml.console.subField.multiInstanceTip'))}
              <AccessTypeTag
                accessType={sf.accessType}
                valueType={sf.valueType}
                constraintText={sf.constraintText}
              />
              {sf.changeApplies === 'OnReboot' && (
                <Tag color="warning">{t('mml.console.subField.onReboot')}</Tag>
              )}
            </div>
            {/* 约束 / 取值范围已合并到 AccessTypeTag（用户决策 2026-05-22），
                输入框下方不再重复渲染 hint 行，避免视觉冗余。 */}
          </div>
        );
      })}
    </div>
  );
}
