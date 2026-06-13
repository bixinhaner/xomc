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
 * 模式约束（用户决策 2026-05-22）：
 *   - 全部 op_type（LST / MOD / ADD / RMV）一律**必填**，不再保留 LST partial path 例外；
 *     占位符给出 spec 范围说明（如 `1~24` / `≥ 0` / `N 由 NumberOfEntries 决定` 等），
 *     由 `instance_range_meta` 派生。
 */

import { Input, Space, Tooltip } from 'antd';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';
import type { MMLOperationType } from '@core/types/mml';
import type { InstanceRange } from '@core/types/mmlConsole';
import {
  validateInstanceLayer,
  findRangeByLayer,
  buildRangeHint,
} from './instanceRangeValidation';
import { selectorKeysForArity } from './instanceArity';

/**
 * 由 InstanceRange 派生输入框 placeholder 文案。
 *   · 静态范围 → "1~24"
 *   · 动态上限 → "≥ 1（上限：NumberOfEntries）"
 *   · 无 metadata → "请输入实例号"
 */
function buildPlaceholder(
  range: InstanceRange | undefined,
  fallback: string,
  t: (id: string, values?: Record<string, string | number>) => string,
): string {
  if (!range) return fallback;
  const min = range.rangeMin;
  const max = range.rangeMax;
  if (typeof min === 'number' && typeof max === 'number') return `${min}~${max}`;
  if (typeof min === 'number' && range.dynamic && range.nSource) {
    return t('mml.console.instanceArity.placeholderDynamic', { min, source: range.nSource });
  }
  if (typeof min === 'number') return `≥ ${min}`;
  if (typeof max === 'number') return `≤ ${max}`;
  return fallback;
}

// selectorKeysForArity / deriveArityFromSubFields 已拆到同级 ./instanceArity.ts
// （react-refresh/only-export-components：组件文件只导出组件）。

export interface InstanceArityInputProps {
  /** 路径中 `.{i}.` 的层数。0 时组件不渲染。 */
  arity: number;
  /** 当前 selectors（key=Greek selector key，例 iα/iβ） */
  values: Record<string, string>;
  /** 单层值变化回调（key 为 selector key，可直接合并入 store.setInstanceSelectors） */
  onChange: (selectorKey: string, value: string) => void;
  /** 命令的 op_type，决定"是否必填"提示 */
  operationType: MMLOperationType;
  /**
   * R-4.1.1: 每层 {i} 占位符的取值范围 metadata（来自 cmd.instanceRangeMeta）。
   * undefined / 空数组 → 仅做"必填+整数格式"基本校验；
   * 单个 layer 在数组中找不到 entry → 该层跳过范围校验。
   */
  instanceRangeMeta?: InstanceRange[];
  /** 是否禁用（如设备未选 / 字典未加载） */
  disabled?: boolean;
}

export function InstanceArityInput({
  arity,
  values,
  onChange,
  operationType,
  instanceRangeMeta,
  disabled,
}: InstanceArityInputProps): JSX.Element | null {
  const t = useT();
  const token = useThemeToken();

  if (arity <= 0) {
    return null;
  }
  const keys = selectorKeysForArity(arity);
  // 用户决策 2026-05-22：全部 op_type 一律必填（含 LST），不再保留 partial path 例外。
  // 仍保留参数 operationType 以兼容调用方 / 日志，仅作 reference。
  void operationType;
  const isRequired = true;

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
          const layer = idx + 1; // 1-based，对齐后端 InstanceRange.layer
          const range = findRangeByLayer(instanceRangeMeta, layer);
          const validation = validateInstanceLayer(value, range, isRequired);
          const rangeHint = buildRangeHint(range, t);
          // Tooltip 文案：错误信息（如有）在第一行，spec 范围/描述 hint 在后面
          // 先合成数组再 join，避免错误展示和 hint 互相覆盖
          const tooltipLines: string[] = [];
          if (!validation.ok && validation.errorKey) {
            tooltipLines.push(t(validation.errorKey, validation.errorValues));
          }
          if (rangeHint) tooltipLines.push(rangeHint);
          const tooltipTitle = tooltipLines.length > 0 ? tooltipLines.join('\n') : undefined;

          return (
            <Tooltip
              key={key}
              title={tooltipTitle}
              placement="top"
              styles={{ root: { whiteSpace: 'pre-line', maxWidth: 320 } }}
            >
              <span
                style={{ display: 'inline-flex', alignItems: 'center', gap: 4, fontSize: 12 }}
              >
                <span style={{ color: token.colorTextTertiary }}>
                  {t('mml.console.instanceArity.layer', { n: layer })}
                </span>
                <span style={{ color: token.colorTextQuaternary }}>{key} =</span>
                <Input
                  size="small"
                  status={!validation.ok ? 'error' : undefined}
                  style={{ width: 120 }}
                  placeholder={buildPlaceholder(range, t('mml.console.instanceArity.placeholder'), t)}
                  value={value}
                  disabled={disabled}
                  onChange={(e) => onChange(key, e.target.value)}
                  data-testid={`instance-arity-input-${key}`}
                  aria-invalid={!validation.ok || undefined}
                />
              </span>
            </Tooltip>
          );
        })}
      </Space>
    </div>
  );
}
