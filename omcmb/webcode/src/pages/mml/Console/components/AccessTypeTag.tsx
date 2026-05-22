/**
 * AccessTypeTag — sub_field 行的访问/类型/范围标签
 *
 * 显示规则（spec / 用户决策 2026-05-22）：
 *   - READ_ONLY  (GetParameterValues 可查，SetParameterValues 不可改) → 默认 Tag "只读 <TYPE>"
 *   - READ_WRITE (GetParameterValues 可查 + SetParameterValues 可改)   → 蓝 Tag "读写 <TYPE> <constraint>"
 *
 * 数据来源：
 *   · accessType / valueType — sub_field 自带（来自 standard_params.access / data_type）
 *   · constraintText         — sub_field i18n 解析后的取值范围文案
 *                              （如 "STRING(64)" / "[0, 65535]" / "true / false"）
 *
 * 使用方：SubFieldChecklist（LST）/ SubFieldInputList（MOD / ADD）。
 */

import { Tag, Tooltip } from 'antd';
import { useT } from '@/hooks/useT';

export interface AccessTypeTagProps {
  accessType: 'READ_ONLY' | 'READ_WRITE' | string;
  valueType?: string;
  constraintText?: string;
}

export default function AccessTypeTag({
  accessType,
  valueType,
  constraintText,
}: AccessTypeTagProps): JSX.Element {
  const t = useT();
  const isReadOnly = accessType === 'READ_ONLY';

  const typeText = (valueType ?? '').trim().toUpperCase() || '—';
  const range = (constraintText ?? '').trim();

  const accessLabel = isReadOnly
    ? t('mml.console.subField.accessTag.readOnly')
    : t('mml.console.subField.accessTag.readWrite');

  // 读写默认带范围；只读不带（用户不会编辑，范围信息冗余）
  const showRange = !isReadOnly && range !== '';
  const tagText = `${accessLabel} ${typeText}${showRange ? ` ${range}` : ''}`;

  const tooltipKey = isReadOnly
    ? 'mml.console.subField.readOnlyTip'
    : 'mml.console.subField.readWriteTip';

  return (
    <Tooltip title={t(tooltipKey)}>
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
