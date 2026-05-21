import { useMemo } from 'react';
import { Input, Tag, Tooltip, Empty } from 'antd';
import { EditOutlined, NumberOutlined } from '@ant-design/icons';
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

  if (sortedFields.length === 0) {
    return <Empty description={false} style={{ padding: 16 }} />;
  }

  // D35：笔图标 ✏️ 仅在 MOD 上下文渲染（装饰性，告知"此字段可改"）；
  // ADD 上下文虽也可输入，但语义是"创建新对象"，统一不渲染。
  const showEditIcon = statement.operationType === 'MOD';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      {sortedFields.map((sf) => {
        const currentValue = statement.values[sf.mmlCode] ?? '';
        return (
          <div key={sf.id} style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ flex: '0 0 200px', fontWeight: 500 }}>
                {sf.isRequired && (
                  <span style={{ color: '#ff4d4f', marginRight: 4 }} aria-label="required">
                    *
                  </span>
                )}
                {sf.label}
              </span>
              <Input
                value={currentValue}
                onChange={(e) => setValue(statement.uid, sf.mmlCode, e.target.value)}
                placeholder={sf.defaultValue ?? ''}
                style={{ flex: 1 }}
                status={sf.isRequired && !currentValue ? 'warning' : undefined}
              />
              {multiInstanceHint(sf.tr069Path, t('mml.console.subField.multiInstanceTip'))}
              {showEditIcon && (
                <Tooltip title={t('mml.console.subField.readWriteTip')}>
                  <EditOutlined
                    style={{ color: '#1677ff' }}
                    aria-label="editable"
                  />
                </Tooltip>
              )}
              {sf.changeApplies === 'OnReboot' && (
                <Tag color="warning">{t('mml.console.subField.onReboot')}</Tag>
              )}
            </div>
            {sf.constraintText && (
              <span
                style={{
                  fontSize: 12,
                  color: '#999',
                  paddingLeft: 208,
                }}
              >
                {t('mml.console.input.constraint')}: {sf.constraintText}
              </span>
            )}
          </div>
        );
      })}
    </div>
  );
}
