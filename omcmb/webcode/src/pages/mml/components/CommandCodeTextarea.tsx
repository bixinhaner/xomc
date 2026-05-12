import { Input } from 'antd';
import { useT } from '@/hooks/useT';

// T-0090 子项 ③④：命令编码 input → textarea，且不允许从下拉选择既有命令。
// 抽出为公共组件供 AddTemplateModal（公有/私有）和子任务 d 私有命令页共用，
// 规避 R-NEW-3（组件抽象不足导致复制粘贴）。
//
// Form.Item 通过 cloneElement 注入 value / onChange，本组件接收后转发到 TextArea。

export interface CommandCodeTextareaProps {
  value?: string;
  onChange?: (value: string) => void;
  rows?: number;
  maxLength?: number;
  disabled?: boolean;
}

export default function CommandCodeTextarea({
  value,
  onChange,
  rows = 3,
  maxLength = 200,
  disabled,
}: CommandCodeTextareaProps) {
  const t = useT();
  return (
    <Input.TextArea
      value={value}
      rows={rows}
      maxLength={maxLength}
      disabled={disabled}
      placeholder={t('mml.console.commandCodeTextareaPlaceholder')}
      onChange={(e) => onChange?.(e.target.value)}
      style={{ fontFamily: "'SFMono-Regular', Consolas, monospace", fontSize: 12 }}
    />
  );
}
