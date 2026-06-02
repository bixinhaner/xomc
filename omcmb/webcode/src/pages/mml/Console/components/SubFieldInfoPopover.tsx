/**
 * SubFieldInfoPopover — LST/MOD/ADD sub_field 行尾"详情"图标。
 *
 * 用户决策（2026-05-26）：用单个 InfoCircle 收敛过去散落在行尾的多个图标 / Tag
 * （# 多实例占位、OnReboot 重启 Tag、access Tag 等），让每一行都有一个**统一入口**
 * 看完整 metadata。原条件式提示（仅含 {i} 时才出 # 图标）在 popover 内通过显式
 * "多实例：是 / 否"行表达，避免"为什么这行有 # 那行没有"的视觉割裂感。
 *
 * 内容（按重要性 + LST 视角排序）：
 *   - 路径（tr069Path，monospace，超长自动折行 — Popover 而非 Tooltip 撑得起）
 *   - 描述（standard_params.description，可空）
 *   - 类型（valueType）
 *   - 访问（READ_ONLY / READ_WRITE，文案 i18n）
 *   - 取值范围（constraintText，空 → 显式"无明确取值范围"）
 *   - 多实例（path 是否含 .{i}.，含 → 显式提示要在外层"实例索引"填编号）
 *   - 生效方式（changeApplies：立即 / 重启 / 会话内）
 *
 * 触发方式：hover 或 click，便于鼠标 + 键盘双访问；mouseEnterDelay 0.3 防误触。
 */
import { Popover } from 'antd';
import { InfoCircleOutlined } from '@ant-design/icons';
import type { SubFieldDef } from '@core/types/mmlConsole';
import { useT } from '@/hooks/useT';

export interface SubFieldInfoPopoverProps {
  sf: SubFieldDef;
}

/** 把 changeApplies 翻译为可读文案；未知值兜底原值。 */
function formatChangeApplies(value: string, t: (key: string) => string): string {
  switch (value) {
    case 'OnReboot':
      return t('mml.console.subField.onReboot');
    case 'Immediate':
      return t('mml.console.subField.info.immediate');
    case 'OnSession':
      return t('mml.console.subField.info.onSession');
    default:
      return value || '-';
  }
}

interface InfoRow {
  label: string;
  value: React.ReactNode;
}

function renderRows(rows: InfoRow[]): React.ReactNode {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 6, maxWidth: 360 }}>
      {rows.map((r) => (
        <div key={r.label} style={{ display: 'flex', gap: 8, alignItems: 'flex-start' }}>
          <span style={{ flex: '0 0 64px', color: '#8c8c8c', fontSize: 12 }}>{r.label}</span>
          <span style={{ flex: 1, fontSize: 12, wordBreak: 'break-all' }}>{r.value}</span>
        </div>
      ))}
    </div>
  );
}

export default function SubFieldInfoPopover({ sf }: SubFieldInfoPopoverProps) {
  const t = useT();

  const accessLabel =
    sf.accessType === 'READ_ONLY'
      ? t('mml.console.subField.accessTag.readOnly')
      : sf.accessType === 'READ_WRITE'
        ? t('mml.console.subField.accessTag.readWrite')
        : sf.accessType || '-';

  const rangeLabel =
    sf.constraintText && sf.constraintText.trim() !== ''
      ? sf.constraintText
      : t('mml.console.subField.noRange');

  const isMultiInstance = sf.tr069Path.includes('.{i}.');
  const multiInstanceLabel = isMultiInstance
    ? t('mml.console.subField.multiInstanceTip')
    : '-';

  const changeAppliesLabel = formatChangeApplies(sf.changeApplies, t);

  // 字段详情不再展示 PATH（PATH 改由行内"PATH名称"的 Tooltip 提示）。
  // description 仅在非空时出行，减少空 popover 噪音。
  const rows: InfoRow[] = [];
  if (sf.description && sf.description.trim() !== '') {
    rows.push({
      label: t('mml.console.subField.info.description'),
      value: sf.description,
    });
  }
  rows.push(
    { label: t('mml.console.subField.info.type'), value: sf.valueType || '-' },
    { label: t('mml.console.subField.info.access'), value: accessLabel },
    { label: t('mml.console.subField.info.range'), value: rangeLabel },
    { label: t('mml.console.subField.info.multiInstance'), value: multiInstanceLabel },
    { label: t('mml.console.subField.info.changeApplies'), value: changeAppliesLabel },
  );

  return (
    <Popover
      title={t('mml.console.subField.info.title')}
      content={renderRows(rows)}
      placement="leftTop"
      mouseEnterDelay={0.3}
      trigger={['hover', 'click']}
    >
      <InfoCircleOutlined
        style={{ color: '#8c8c8c', fontSize: 14, cursor: 'pointer' }}
        aria-label="field-info"
      />
    </Popover>
  );
}
