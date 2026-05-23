/**
 * AccessTypeTag — sub_field 行的访问/类型/范围标签
 *
 * 显示规则（spec / 用户决策 2026-05-22 + 2026-05-23）：
 *   - READ_ONLY  (GetParameterValues 可查，SetParameterValues 不可改) → 默认 Tag "只读 <TYPE>"
 *   - READ_WRITE (GetParameterValues 可查 + SetParameterValues 可改)   → 蓝 Tag "读写 <TYPE> <constraint>"
 *   - Tooltip → 路径（若提供）+ 该 path 的取值范围（standard_params.min_value /
 *               max_value 派生）；范围未维护时回落 "无明确取值范围"。
 *
 * 数据来源：
 *   · accessType / valueType — sub_field 自带（来自 standard_params.access / data_type）
 *   · constraintText         — 后端从 standard_params.min_value / max_value 派生
 *                              （如 "[0, 65535]" / "≥ 0" / "≤ 100"）；为空表示
 *                              standard_params 该行 min/max 都未配置
 *   · tr069Path              — MOD / ADD 行内不显示 path（被 Input 占位），Tooltip
 *                              里补齐让用户能 hover Tag 看到完整 path；LST 行内已
 *                              显式渲染 path，传入仍无副作用（重复但不冗余）
 *
 * 使用方：SubFieldChecklist（LST）/ SubFieldInputList（MOD / ADD）。
 */

import { Tag, Tooltip } from 'antd';
import { useT } from '@/hooks/useT';

export interface AccessTypeTagProps {
  accessType: 'READ_ONLY' | 'READ_WRITE' | string;
  valueType?: string;
  constraintText?: string;
  /** TR-069 完整 path（如 'Device.DeviceInfo.SoftwareVersion'）。Tooltip 顶部展示。 */
  tr069Path?: string;
}

export default function AccessTypeTag({
  accessType,
  valueType,
  constraintText,
  tr069Path,
}: AccessTypeTagProps): JSX.Element {
  const t = useT();
  const isReadOnly = accessType === 'READ_ONLY';

  const typeText = (valueType ?? '').trim().toUpperCase() || '—';
  const range = (constraintText ?? '').trim();
  const path = (tr069Path ?? '').trim();

  const accessLabel = isReadOnly
    ? t('mml.console.subField.accessTag.readOnly')
    : t('mml.console.subField.accessTag.readWrite');

  // 读写默认带范围；只读不带（用户不会编辑，范围信息冗余）
  const showRange = !isReadOnly && range !== '';
  const tagText = `${accessLabel} ${typeText}${showRange ? ` ${range}` : ''}`;

  // Tooltip 两行：path（如传入）+ 取值范围
  //   - path 用 monospace 凸显；超长不截断（让 antd Tooltip 自适应宽度）
  //   - range 用同一份 noRange 兜底文案
  const rangeText = range !== '' ? range : t('mml.console.subField.noRange');
  const tooltipNode =
    path !== '' ? (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        <div style={{ fontFamily: 'monospace', fontSize: 12 }}>{path}</div>
        <div style={{ fontSize: 12, opacity: 0.85 }}>{rangeText}</div>
      </div>
    ) : (
      rangeText
    );

  return (
    <Tooltip title={tooltipNode} overlayStyle={{ maxWidth: 480 }}>
      <Tag
        color={isReadOnly ? 'default' : 'blue'}
        style={{ fontSize: 11, marginRight: 0 }}
        aria-label={isReadOnly ? 'read-only' : 'read-write'}
      >
        {tagText}
      </Tag>
    </Tooltip>
  );
}
