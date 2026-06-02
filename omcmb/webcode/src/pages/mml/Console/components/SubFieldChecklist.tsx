import { useMemo } from 'react';
import type { CSSProperties } from 'react';
import { Checkbox, Tooltip, Empty } from 'antd';
type CheckboxValueType = string | number | boolean;
import { useMmlConsoleStore } from '@core/store/mmlConsoleStore';
import type { Statement } from '@core/types/mmlConsole';

export interface SubFieldChecklistProps {
  statement: Statement;
}

export default function SubFieldChecklist({ statement }: SubFieldChecklistProps) {
  const toggleSubField = useMmlConsoleStore((s) => s.toggleSubField);

  const sortedFields = useMemo(
    () => [...statement.subFields].sort((a, b) => a.sortOrder - b.sortOrder),
    [statement.subFields],
  );

  // 固定高度容器（spec：操作面板限制 path 区域 10 行可视高度，<10 行也保留空间，
  // >10 行出现滚动条；每行约 38px = 行内 padding 4+4 + content ~24 + flex gap 6）。
  const VIEWPORT_HEIGHT = 380;
  // 2026-05-28: antd Checkbox.Group 默认 display:inline-flex,内层 div 不显式
  // width:100% 时会按内容自然收缩,导致 viewport 拿不到完整宽度(实测 578 vs 813)。
  // 这里 width:100% + boxSizing:border-box 让 viewport 撑满父容器。
  const VIEWPORT_STYLE: CSSProperties = {
    width: '100%',
    boxSizing: 'border-box',
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

  // 2026-05-27 用户反馈调整:
  //   1. 滚动条永远贴右,把右侧空白全留给 path → 容器外层不加 padding,
  //      row 自己负责左 padding,右侧顶到 scrollbar 左缘
  //   2. 不再区分 READ_ONLY/READ_WRITE 视觉 → 删除 #999 灰字 + #fafafa 浅灰背景
  return (
    <Checkbox.Group value={value} onChange={handleChange} style={{ width: '100%' }}>
      <div style={{ ...VIEWPORT_STYLE, display: 'flex', flexDirection: 'column' }}>
        {sortedFields.map((sf) => (
          <div
            key={sf.id}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '6px 0 6px 8px',
              borderBottom: '1px dashed #f0f0f0',
            }}
          >
            <Checkbox value={sf.id} />
            {/* PATH名称(sf.label):hover 显示完整 PATH(tr069Path)。
                LST 取消"字段详情"图标,也不再单列展示 PATH —— PATH 改由本名称的 Tooltip 提示。 */}
            <span style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
              <Tooltip title={sf.tr069Path} placement="topLeft" mouseEnterDelay={0.3}>
                <span
                  style={{
                    fontWeight: 500,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                    cursor: 'default',
                  }}
                >
                  {sf.label}
                </span>
              </Tooltip>
              {sf.description && (
                <span style={{ color: '#bfbfbf', fontSize: 11, marginTop: 1 }}>
                  {sf.description}
                </span>
              )}
            </span>
          </div>
        ))}
      </div>
    </Checkbox.Group>
  );
}
