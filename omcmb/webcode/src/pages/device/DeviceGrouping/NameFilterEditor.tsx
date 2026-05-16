import { Button, Input, Select, Typography } from 'antd';
import { CloseCircleOutlined, PlusOutlined } from '@ant-design/icons';
import type { NameFilterItem } from './types';
import { getAndOrOptions, getFilterConditionOptions } from './types';

const { Text } = Typography;

export interface NameFilterEditorProps {
  filters: NameFilterItem[];
  /** Update a single field on a single filter row. */
  onUpdate: (id: string, field: keyof NameFilterItem, value: string) => void;
  /** Append a new empty filter row. */
  onAdd: () => void;
  /** Remove a filter row. */
  onRemove: (id: string) => void;
  t: (id: string, values?: Record<string, string | number>) => string;
  maxConditions?: number;
}

/**
 * 设备名称匹配条件编辑器：行级 Select + Input + and/or + 删除按钮。
 * 拆分自 GroupDialogs.tsx，供 AddChildGroupDrawer / EditLevel2GroupDrawer 共用。
 */
export default function NameFilterEditor({
  filters,
  onUpdate,
  onAdd,
  onRemove,
  t,
  maxConditions = 10,
}: NameFilterEditorProps) {
  return (
    <>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {filters.map((filter, index) => (
          <div key={filter.id} style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            {index === 0 ? (
              <>
                <Select
                  value={filter.condition}
                  style={{ width: 120 }}
                  options={getFilterConditionOptions(t)}
                  onChange={(v) => onUpdate(filter.id, 'condition', v)}
                />
                <Input
                  value={filter.value}
                  style={{ flex: 1 }}
                  maxLength={64}
                  placeholder={t('common.placeholder')}
                  onChange={(e) => onUpdate(filter.id, 'value', e.target.value)}
                />
              </>
            ) : (
              <>
                <Select
                  value={filter.andOr || 'and'}
                  style={{ width: 70 }}
                  options={getAndOrOptions(t)}
                  onChange={(v) => onUpdate(filter.id, 'andOr', v)}
                />
                <Select
                  value={filter.condition}
                  style={{ width: 120 }}
                  options={getFilterConditionOptions(t)}
                  onChange={(v) => onUpdate(filter.id, 'condition', v)}
                />
                <Input
                  value={filter.value}
                  style={{ flex: 1 }}
                  maxLength={64}
                  placeholder={t('common.placeholder')}
                  onChange={(e) => onUpdate(filter.id, 'value', e.target.value)}
                />
                <Button
                  type="text"
                  size="small"
                  icon={<CloseCircleOutlined />}
                  onClick={() => onRemove(filter.id)}
                  style={{ color: 'var(--color-text-quaternary)' }}
                />
              </>
            )}
          </div>
        ))}
      </div>
      {filters.length < maxConditions && (
        <Button type="dashed" icon={<PlusOutlined />} onClick={onAdd} style={{ marginTop: 8 }}>
          {t('device.rules.addCondition')}
        </Button>
      )}
    </>
  );
}

/** Render the small grey "filterCondition (max N)" label used inside Form.Item. */
export function FilterConditionLabel({ t, max = 10 }: { t: NameFilterEditorProps['t']; max?: number }) {
  return (
    <span>
      {t('device.rules.filterCondition')}
      <Text type="secondary" style={{ fontSize: 12, marginLeft: 4 }}>
        {t('device.rules.conditionLimit', { max })}
      </Text>
    </span>
  );
}
