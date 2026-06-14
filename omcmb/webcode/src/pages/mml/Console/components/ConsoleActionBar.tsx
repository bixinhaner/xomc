/**
 * ConsoleActionBar — Control Panel 底部操作行（R-9.1 + R-10）
 *
 * 方案：docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md §7.2 / §7.7
 *
 * 取代 MmlEditor 中"生成 MML + 执行"的混合区域；执行按钮显式带 op_type
 * （例 `执行 LST ▸` / `执行 MOD ▸`），数据全部来自 SubFieldChecklist /
 * SubFieldInputList / InstancePicker 的勾选与填值状态。
 *
 * 设计目的：让操作面板下方有一个稳定可见的执行入口,避免用户在 LST/MOD/ADD/RMV
 * 切换时找不到执行按钮（之前 MmlEditor 内联执行按钮,切换时位置漂移）。
 *
 * 2026-05-28 用户决策(单PATH执行):
 *   - LST / MOD 新增"执行模式" Radio: 整体执行 / 单PATH执行
 *   - ADD / RMV 单实例语义,不显示开关
 *   - 整体放一行: [Tag] 摘要 · 设备数  |  执行模式 ◉/○  |  [全部取消] [执行 ▸]
 */

import type React from 'react';
import { Button, Radio, Space, Tooltip } from 'antd';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';
import type { MMLOperationType } from '@core/types/mml';

/** 单PATH 执行模式适用 op 集合。ADD/RMV 单实例,开关无意义。 */
const MULTI_PATH_OPS: ReadonlyArray<MMLOperationType> = ['LST', 'MOD'];

export type ExecutionMode = 'batch' | 'per-path';

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
  /**
   * 切换全选/全部取消（v2.4 D36 — 合并 onSelectAll + onClearAll 为单个 toggle）。
   * undefined 时按钮不渲染（如 MOD/ADD/RMV 模式下不需要全选）。
   */
  onToggleAll?: () => void;
  /** 当前是否已经全部选中（决定按钮文案：未全选 → "全选"；已全选 → "全部取消"） */
  allSelected?: boolean;
  /** 执行回调 */
  onExecute: () => void;
  /**
   * 执行模式:'batch'(默认) | 'per-path'(单PATH独立任务)。
   * 仅 LST/MOD 显示 Radio,ADD/RMV 即使传入也不渲染。
   * undefined 时按 batch 处理,Radio 也不渲染(向后兼容老调用方)。
   */
  executionMode?: ExecutionMode;
  onExecutionModeChange?: (mode: ExecutionMode) => void;
}

export function ConsoleActionBar({
  operationType,
  summaryText,
  deviceCount,
  disabled,
  loading,
  onValidate,
  onToggleAll,
  allSelected,
  onExecute,
  executionMode,
  onExecutionModeChange,
}: ConsoleActionBarProps): React.JSX.Element {
  const t = useT();
  const token = useThemeToken();

  const showExecModeRadio =
    executionMode !== undefined &&
    onExecutionModeChange !== undefined &&
    MULTI_PATH_OPS.includes(operationType);

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 16,
        padding: '8px 12px',
        borderTop: `1px solid ${token.colorBorderSecondary}`,
        background: token.colorBgContainer,
        position: 'sticky',
        bottom: 0,
        zIndex: 1,
        flexWrap: 'wrap', // < 700 px 容器自动 wrap,小屏兜底
      }}
    >
      {/* 左:摘要信息(已选 N/M path · K 台设备) */}
      <div style={{ fontSize: 12, color: token.colorTextSecondary, whiteSpace: 'nowrap' }}>
        {summaryText && <span>{summaryText}</span>}
        {summaryText && deviceCount != null && <span> · </span>}
        {deviceCount != null && (
          <span>
            {deviceCount} {t('mml.console.actionBar.devices')}
          </span>
        )}
      </div>

      {/* 中:执行模式 Radio(仅 LST/MOD) */}
      {showExecModeRadio && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, whiteSpace: 'nowrap' }}>
          <span style={{ fontSize: 12, color: token.colorTextSecondary }}>
            {t('mml.console.actionBar.execMode')}
          </span>
          <Radio.Group
            value={executionMode}
            onChange={(e) => onExecutionModeChange?.(e.target.value as ExecutionMode)}
            disabled={loading}
            size="small"
          >
            <Tooltip title={t('mml.console.actionBar.execMode.batchTip')}>
              <Radio value="batch">{t('mml.console.actionBar.execMode.batch')}</Radio>
            </Tooltip>
            <Tooltip title={t('mml.console.actionBar.execMode.perPathTip')}>
              <Radio value="per-path">{t('mml.console.actionBar.execMode.perPath')}</Radio>
            </Tooltip>
          </Radio.Group>
        </div>
      )}

      {/* 右:按钮组(校验 / 全选/全部取消 / 执行) */}
      <Space size="small" style={{ marginLeft: 'auto' }}>
        {onValidate && (
          <Button size="small" onClick={onValidate} disabled={loading}>
            {t('mml.console.actionBar.validate')}
          </Button>
        )}
        {onToggleAll && (
          <Button size="small" onClick={onToggleAll} disabled={loading}>
            {allSelected
              ? t('mml.console.actionBar.clearAll')
              : t('mml.console.actionBar.selectAll')}
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
