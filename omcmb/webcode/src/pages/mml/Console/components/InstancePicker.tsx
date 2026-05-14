import { InputNumber } from 'antd';
import { useMmlConsoleStore } from '@core/store/mmlConsoleStore';
import type { Statement } from '@core/types/mmlConsole';
import { useT } from '@/hooks/useT';

export interface InstancePickerProps {
  statement: Statement;
}

export default function InstancePicker({ statement }: InstancePickerProps) {
  const t = useT();
  const setRmvIndex = useMmlConsoleStore((s) => s.setRmvIndex);

  const handleChange = (value: number | null) => {
    if (value === null || value === undefined) {
      setRmvIndex(statement.uid, undefined);
      return;
    }
    if (value < 0) return;
    setRmvIndex(statement.uid, value);
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <span style={{ flex: '0 0 200px', fontWeight: 500 }}>
          <span style={{ color: '#ff4d4f', marginRight: 4 }} aria-label="required">
            *
          </span>
          {t('mml.console.picker.indexLabel')}
        </span>
        <InputNumber
          value={statement.rmvInstanceIndex ?? null}
          onChange={handleChange}
          min={0}
          step={1}
          precision={0}
          style={{ width: 160 }}
        />
      </div>
      <span style={{ fontSize: 12, color: '#999', paddingLeft: 208 }}>
        {t('mml.console.picker.indexHelp')}
      </span>
    </div>
  );
}
