import { useMemo } from 'react';
import type { CSSProperties } from 'react';
import { Checkbox, Tooltip, Empty } from 'antd';
type CheckboxValueType = string | number | boolean;
import { useMmlConsoleStore } from '@core/store/mmlConsoleStore';
import type { Statement } from '@core/types/mmlConsole';
import SubFieldInfoPopover from './SubFieldInfoPopover';

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
                {/* path 列宽度有限（操作面板 ~600px - label 200 - checkbox/icons 占用），
                    超长 path 必然被 ellipsis 截断。统一加 Tooltip：hover/focus 任意 path
                    都能看到完整 monospace 路径；点击文本可复制（用户决策 2026-05-26）。
                    placement=topLeft 让 Tooltip 不遮挡当前行下方的 description / 下一行。 */}
                <Tooltip title={sf.tr069Path} placement="topLeft" mouseEnterDelay={0.3}>
                  <span
                    style={{
                      color: '#888',
                      fontFamily: 'monospace',
                      fontSize: 12,
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                      cursor: 'default',
                    }}
                  >
                    {sf.tr069Path}
                  </span>
                </Tooltip>
                {sf.description && (
                  <span style={{ color: '#bfbfbf', fontSize: 11, marginTop: 1 }}>
                    {sf.description}
                  </span>
                )}
              </span>
              {/* 用户决策 2026-05-26：行尾散落的 # 多实例图标 + OnReboot 警告 Tag +
                  access Tooltip 收敛为单个 InfoCircle 入口。每行都有同一个图标，
                  hover/点击展开包含 path / 类型 / 访问 / 取值范围 / 多实例 / 生效方式 /
                  描述的完整元数据，消除"为什么这行有 # 那行没有"的视觉割裂感。 */}
              <SubFieldInfoPopover sf={sf} />
            </div>
          );
        })}
      </div>
    </Checkbox.Group>
  );
}
