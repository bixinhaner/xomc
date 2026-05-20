/**
 * ConsoleActionBar — Control Panel 底部操作行（R-9.1 + R-10）
 *
 * 方案：docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md §7.2 / §7.7
 *
 * 取代 MmlEditor 中"生成 MML + 执行"的混合区域；执行按钮显式带 op_type
 * （例 `执行 LST ▸` / `执行 MOD ▸`），数据全部来自 SubFieldChecklist /
 * SubFieldInputList / InstancePicker 的勾选与填值状态。
 *
 * 设计目的：让操作面板下方有一个稳定可见的执行入口，避免用户在 LST/MOD/ADD/RMV
 * 切换时找不到执行按钮（之前 MmlEditor 内联执行按钮，切换时位置漂移）。
 */

import { Button, Space } from 'antd';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';
import type { MMLOperationType } from '@core/types/mml';

export interface ConsoleActionBarProps {
  /** 当前命令的 op_type；按钮文字显示该 op（如 `执行 LST ▸`） */
  operationType: MMLOperationType;
  /** 已选 path / 已填值 / 已选实例数（用于左侧信息显示） */
  summaryText?: string;
  /** 设备数（"· N 设备"） */
  deviceCount?: number;
  /** 是否禁用执行按钮（如校验失败 / 未选设备 / 未配置 path） */
  disabled?: boolean;
  /** 执行中 loading 状态 */
  loading?: boolean;
  /** 校验所有值（MOD/ADD 模式可选） */
  onValidate?: () => void;
  /** 全选 path / 字段 */
  onSelectAll?: () => void;
  /** 全不选 path / 字段 */
  onClearAll?: () => void;
  /** 执行回调 */
  onExecute: () => void;
}

export function ConsoleActionBar({
  operationType,
  summaryText,
  deviceCount,
  disabled,
  loading,
  onValidate,
  onSelectAll,
  onClearAll,
  onExecute,
}: ConsoleActionBarProps): JSX.Element {
  const t = useT();
  const token = useThemeToken();

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: 12,
        padding: '8px 12px',
        borderTop: `1px solid ${token.colorBorderSecondary}`,
        background: token.colorBgContainer,
        position: 'sticky',
        bottom: 0,
        zIndex: 1,
      }}
    >
      {/* 左：摘要信息 */}
      <div style={{ fontSize: 12, color: token.colorTextSecondary }}>
        {summaryText && <span>{summaryText}</span>}
        {summaryText && deviceCount != null && <span> · </span>}
        {deviceCount != null && (
          <span>
            {deviceCount} {t('mml.console.actionBar.devices')}
          </span>
        )}
      </div>

      {/* 右：按钮组 */}
      <Space size="small">
        {onValidate && (
          <Button size="small" onClick={onValidate} disabled={loading}>
            {t('mml.console.actionBar.validate')}
          </Button>
        )}
        {onSelectAll && (
          <Button size="small" onClick={onSelectAll} disabled={loading}>
            {t('mml.console.actionBar.selectAll')}
          </Button>
        )}
        {onClearAll && (
          <Button size="small" onClick={onClearAll} disabled={loading}>
            {t('mml.console.actionBar.clearAll')}
          </Button>
        )}
        <Button
          type="primary"
          size="small"
          onClick={onExecute}
          disabled={disabled}
          loading={loading}
        >
          {t('mml.console.actionBar.execute', { op: operationType })}
        </Button>
      </Space>
    </div>
  );
}
