import { useMemo } from 'react';
import type { CSSProperties } from 'react';
import { Checkbox, Input, Tag, Tooltip, Empty } from 'antd';
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
  const toggleSubField = useMmlConsoleStore((s) => s.toggleSubField);

  // MOD：path 改为选填（用户决策 2026-05-23）—— 每行加 Checkbox 与 LST 一致，
  // 用户勾选的 path 才纳入下发；未勾选行的 Input 灰显但保留用户已输入值（再次
  // 勾选时无需重新填）。ADD 暂保留 "全字段编辑" 语义（创建新对象需要所有
  // is_required 字段，未来再分流）。
  const isMod = statement.operationType === 'MOD';
  const selectedSet = useMemo(
    () => new Set(statement.selectedSubFieldIds),
    [statement.selectedSubFieldIds],
  );

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
        const isSelected = !isMod || selectedSet.has(sf.id);
        return (
          <div key={sf.id} style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                opacity: isSelected ? 1 : 0.55,
              }}
            >
              {isMod && (
                <Checkbox
                  checked={selectedSet.has(sf.id)}
                  onChange={() => toggleSubField(statement.uid, sf.id)}
                  aria-label={`select-${sf.mmlCode}`}
                />
              )}
              <span style={{ flex: '0 0 200px', display: 'flex', flexDirection: 'column', minWidth: 0 }}>
                <span style={{ fontWeight: 500 }}>
                  {/* MOD 模式 path 改为选填，不再用 is_required 标红 *；ADD 保留 */}
                  {sf.isRequired && !isMod && (
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
                // 用户决策 2026-05-23：value 输入框宽度减半（原 flex:1 占满剩余空间，
                // 现取固定 240px 上限 + 0 1 缩放，留出右侧空间给 path Tag / OnReboot 等）
                style={{ flex: '0 1 240px', maxWidth: 240 }}
                status={!isMod && sf.isRequired && !currentValue ? 'warning' : undefined}
                // MOD 未勾选行禁用输入（仍保留已有 value，避免误清）
                disabled={isMod && !isSelected}
              />
              {/* 填充剩余空间，把 path Tag 等推到右侧对齐 */}
              <div style={{ flex: 1 }} />
              {multiInstanceHint(sf.tr069Path, t('mml.console.subField.multiInstanceTip'))}
              <AccessTypeTag
                accessType={sf.accessType}
                valueType={sf.valueType}
                constraintText={sf.constraintText}
                tr069Path={sf.tr069Path}
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
