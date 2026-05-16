import { useMemo } from 'react';
import { Checkbox, Tag, Tooltip, Empty } from 'antd';
import { EyeOutlined, EditOutlined } from '@ant-design/icons';
type CheckboxValueType = string | number | boolean;
import { useMmlConsoleStore } from '@core/store/mmlConsoleStore';
import type { Statement, SubFieldDef } from '@core/types/mmlConsole';
import { useT } from '@/hooks/useT';

export interface SubFieldChecklistProps {
  statement: Statement;
}

function accessIcon(sf: SubFieldDef, readOnlyLabel: string) {
  if (sf.accessType === 'READ_ONLY') {
    return (
      <Tooltip title={readOnlyLabel}>
        <EyeOutlined style={{ color: '#999' }} />
      </Tooltip>
    );
  }
  return <EditOutlined style={{ color: '#1677ff' }} />;
}

export default function SubFieldChecklist({ statement }: SubFieldChecklistProps) {
  const t = useT();
  const toggleSubField = useMmlConsoleStore((s) => s.toggleSubField);

  const sortedFields = useMemo(
    () => [...statement.subFields].sort((a, b) => a.sortOrder - b.sortOrder),
    [statement.subFields],
  );

  if (sortedFields.length === 0) {
    return <Empty description={false} style={{ padding: 16 }} />;
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
      <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
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
              <span
                style={{
                  flex: '0 0 200px',
                  fontWeight: 500,
                  color: isReadOnly ? '#999' : undefined,
                }}
              >
                {sf.label}
              </span>
              <span style={{ flex: 1, color: '#888', fontFamily: 'monospace', fontSize: 12 }}>
                {sf.tr069Path}
              </span>
              {accessIcon(sf, t('mml.console.subField.readOnly'))}
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
