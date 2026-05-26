import { useMemo } from 'react';
import type { CSSProperties } from 'react';
import { Checkbox, Tag, Tooltip, Empty } from 'antd';
import { NumberOutlined } from '@ant-design/icons';
type CheckboxValueType = string | number | boolean;
import { useMmlConsoleStore } from '@core/store/mmlConsoleStore';
import type { Statement, SubFieldDef } from '@core/types/mmlConsole';
import { useT } from '@/hooks/useT';

export interface SubFieldChecklistProps {
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

export default function SubFieldChecklist({ statement }: SubFieldChecklistProps) {
  const t = useT();
  const toggleSubField = useMmlConsoleStore((s) => s.toggleSubField);

  const sortedFields = useMemo(
    () => [...statement.subFields].sort((a, b) => a.sortOrder - b.sortOrder),
    [statement.subFields],
  );

  // 固定高度容器（spec：操作面板限制 path 区域 10 行可视高度，<10 行也保留空间，
  // >10 行出现滚动条；每行约 38px = 行内 padding 4+4 + content ~24 + flex gap 6）。
  const VIEWPORT_HEIGHT = 380;
  const VIEWPORT_STYLE: CSSProperties = {
    height: VIEWPORT_HEIGHT,
    overflowY: 'auto',
    border: '1px solid #f0f0f0',
    borderRadius: 4,
  };

  if (sortedFields.length === 0) {
    return (
      <div style={{ ...VIEWPORT_STYLE, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Empty description={false} />
      </div>
    );
  }

  const value: CheckboxValueType[] = statement.selectedSubFieldIds;

  const handleChange = (next: CheckboxValueType[]) => {
    const nextSet = new Set(next.map(String));
    const prevSet = new Set(statement.selectedSubFieldIds);
    // 计算 diff 调 toggleSubField（store 内已用 Set 去重，多调一次幂等）
    sortedFields.forEach((sf) => {
      const inNext = nextSet.has(sf.id);
      const inPrev = prevSet.has(sf.id);
      if (inNext !== inPrev) {
        toggleSubField(statement.uid, sf.id);
      }
    });
  };

  return (
    <Checkbox.Group value={value} onChange={handleChange} style={{ width: '100%' }}>
      <div style={{ ...VIEWPORT_STYLE, display: 'flex', flexDirection: 'column', gap: 6, padding: 4 }}>
        {sortedFields.map((sf) => {
          const isReadOnly = sf.accessType === 'READ_ONLY';
          return (
            <div
              key={sf.id}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '4px 8px',
                borderBottom: '1px dashed #f0f0f0',
                background: isReadOnly ? '#fafafa' : undefined,
              }}
            >
              <Checkbox value={sf.id} />
              <Tooltip title={sf.label} placement="top" mouseEnterDelay={0.3}>
                <span
                  style={{
                    flex: '0 0 200px',
                    fontWeight: 500,
                    color: isReadOnly ? '#999' : undefined,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {sf.label}
                </span>
              </Tooltip>
              <span style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
                <span style={{ color: '#888', fontFamily: 'monospace', fontSize: 12, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {sf.tr069Path}
                </span>
                {sf.description && (
                  <span style={{ color: '#bfbfbf', fontSize: 11, marginTop: 1 }}>
                    {sf.description}
                  </span>
                )}
              </span>
              {multiInstanceHint(sf.tr069Path, t('mml.console.subField.multiInstanceTip'))}
              {/* 用户决策 2026-05-23：LST 只读查询，访问类型 Tag（"读写 / 只读 + TYPE"）
                  对用户无操作意义（用户只是查值，不存在编辑动作），移除以减少视觉噪音。
                  MOD / ADD 行的 Tag 仍保留（用户决定要不要改的 path，access 类型有意义）。 */}
              {sf.changeApplies === 'OnReboot' && (
                <Tag color="warning">{t('mml.console.subField.onReboot')}</Tag>
              )}
            </div>
          );
        })}
      </div>
    </Checkbox.Group>
  );
}
