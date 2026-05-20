/**
 * InstanceArityInput — 多层 {i} 实例索引输入区（R-4）
 *
 * 方案：docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md §7.7.7
 *
 * 当命令路径含多层 `.{i}.` 占位符时（arity > 0），按层数渲染输入框。
 *
 * Selector key 命名约定（与后端 substituteInstanceSelectors 对接）：
 *   - key 按 Greek 字母字典序 iα/iβ/iγ/iδ/iε... 命名
 *   - 字典序与 path 中 `.{i}.` 左到右出现顺序对齐
 *   - 后端按 key 字典序左到右 replace path 中 `.{i}.`
 *
 * 模式约束：
 *   - LST 模式：允许任意层留空 = partial path（CWMP 标准行为）— 注：当前实现要求填齐
 *   - MOD/ADD 模式：必须填齐所有层
 *   - RMV 模式：父路径用本组件；最后一层"实例号"由 InstancePicker 提供
 */

import { Input, Space } from 'antd';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';
import type { MMLOperationType } from '@core/types/mml';

/**
 * Greek 字母对应每层的 selector key。
 * arity 通常 ≤ 3（v2.3 catalog 最深 MU→Slot→EU），余量到 ε 应对未来扩展。
 */
const SELECTOR_KEYS = ['iα', 'iβ', 'iγ', 'iδ', 'iε'] as const;

/** 按 arity 生成 selector key 数组。 */
export function selectorKeysForArity(arity: number): string[] {
  return SELECTOR_KEYS.slice(0, Math.min(arity, SELECTOR_KEYS.length));
}

/**
 * deriveArityFromSubFields — 由 sub_field.tr069Path 推断 instance arity。
 * 后端 BuildTree 尚未暴露 mml_param_groups.instance_arity，前端 derive。
 *
 * 同 command 内所有 sub_field 应同 arity（catalog 由组级 instance_arity 决定）；
 * 异常时取最大值，由后端校验数量 mismatch 兜底。
 */
export function deriveArityFromSubFields(
  subFields: ReadonlyArray<{ tr069Path?: string }>,
): number {
  let max = 0;
  for (const sf of subFields) {
    const path = sf.tr069Path ?? '';
    let count = 0;
    let idx = path.indexOf('.{i}.');
    while (idx !== -1) {
      count++;
      idx = path.indexOf('.{i}.', idx + 5);
    }
    if (count > max) max = count;
  }
  return max;
}

export interface InstanceArityInputProps {
  /** 路径中 `.{i}.` 的层数。0 时组件不渲染。 */
  arity: number;
  /** 当前 selectors（key=Greek selector key，例 iα/iβ） */
  values: Record<string, string>;
  /** 单层值变化回调（key 为 selector key，可直接合并入 store.setInstanceSelectors） */
  onChange: (selectorKey: string, value: string) => void;
  /** 命令的 op_type，决定"是否必填"提示 */
  operationType: MMLOperationType;
  /** 是否禁用（如设备未选 / 字典未加载） */
  disabled?: boolean;
}

export function InstanceArityInput({
  arity,
  values,
  onChange,
  operationType,
  disabled,
}: InstanceArityInputProps): JSX.Element | null {
  const t = useT();
  const token = useThemeToken();

  if (arity <= 0) {
    return null;
  }
  const keys = selectorKeysForArity(arity);
  // LST 允许任意层留空（partial path）；MOD/ADD/RMV 强制必填，否则后端 400 mismatch
  const isRequired = operationType !== 'LST';

  return (
    <div
      style={{
        padding: '10px 12px',
        border: `1px solid ${token.colorBorderSecondary}`,
        borderRadius: 4,
        background: token.colorFillTertiary,
      }}
      data-testid="instance-arity-input"
    >
      <div style={{ fontSize: 12, color: token.colorTextSecondary, marginBottom: 6 }}>
        {t('mml.console.instanceArity.label')}
        {isRequired && (
          <span style={{ color: token.colorError, marginLeft: 4 }}>
            * {t('mml.console.instanceArity.requiredHint')}
          </span>
        )}
      </div>
      <Space size={[6, 6]} wrap>
        {keys.map((key, idx) => {
          const value = values[key] ?? '';
          const isEmpty = value.trim() === '';
          return (
            <span
              key={key}
              style={{ display: 'inline-flex', alignItems: 'center', gap: 4, fontSize: 12 }}
            >
              <span style={{ color: token.colorTextTertiary }}>
                {t('mml.console.instanceArity.layer', { n: idx + 1 })}
              </span>
              <span style={{ color: token.colorTextQuaternary }}>{key} =</span>
              <Input
                size="small"
                style={{
                  width: 80,
                  borderColor: isEmpty && isRequired ? token.colorError : undefined,
                }}
                placeholder={isRequired ? '1' : t('mml.console.instanceArity.placeholder')}
                value={value}
                disabled={disabled}
                onChange={(e) => onChange(key, e.target.value)}
              />
            </span>
          );
        })}
      </Space>
    </div>
  );
}
