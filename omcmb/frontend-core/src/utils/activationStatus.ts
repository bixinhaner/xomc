/**
 * 设备激活状态语义三态 —— 三皮肤(webcode / webcode-v2 / webcode-v3)的"激活状态"
 * 渲染层(列表 Tag 列、详情头部 Tag、详情 KV 文字、HUD 仪表 KV)必须复用此函数,
 * 否则不同入口("设备列表激活状态列" vs "设备详情头部状态")会出现不一致 ——
 * 历史上 PR #435 / commit c69ba889 试图收敛但口径仍有 drift,见该提交评注。
 *
 * 后端 op_state 来源：handler 层 `DeriveOpStateActivated(first_online_time)`：
 *   - '1'         → 设备已首次上线 → 激活 (active)
 *   - '0'         → 设备从未上线   → 未激活 (inactive)
 *   - 其它字符串  → 含 deviceApi.mapBackendDevice 把空值兜底成的 'unknown'，
 *                  以及任何理论上不该出现的 raw 值
 *
 * 判定规则（统一）:
 *   - 空 / null / undefined / 字符串 'unknown'  → null (UI 应显示占位符 '-' 或 '—')
 *   - '1' / 'true'（忽略大小写）                 → 'active'
 *   - 其它一切                                   → 'inactive'
 *     (按"未激活"展示比按原值回显更安全 —— 避免详情页打出 raw "unknown"/"active" 等
 *      内部字符串与列表 Tag 表现脱节)
 */
import { getI18nText, type Locale } from './i18nText';
import { containsHan } from './deviceDisplay';

export type ActivationStatus = 'active' | 'inactive' | null;

export function activationStatusOf(opState: string | undefined | null): ActivationStatus {
  if (opState == null) return null;
  const normalized = opState.trim().toLowerCase();
  if (normalized === '' || normalized === 'unknown') return null;
  return normalized === '1' || normalized === 'true' ? 'active' : 'inactive';
}

interface ActivationStatusDetailLike {
  label?: string;
  labelI18n?: Record<string, string>;
  value?: string;
  status?: boolean;
}

interface ActivationStatusFallbackLabels {
  active: string;
  inactive: string;
}

export function activationStatusLabelOf(
  opState: string | undefined | null,
  details: ReadonlyArray<ActivationStatusDetailLike> | null | undefined,
  fallback: ActivationStatusFallbackLabels,
  locale: Locale = 'zh-CN',
): string | null {
  const status = activationStatusOf(opState);
  if (status == null) return null;

  const findLabel = (value: string) => {
    const detail = details?.find((item) => item?.status !== false && item?.value === value);
    if (!detail) return null;
    const label = getI18nText(detail.labelI18n, locale, detail.label).trim();
    if (locale === 'en-US' && containsHan(label) && !detail.labelI18n?.['en-US']) return null;
    return label || null;
  };

  if (opState != null && opState !== '') {
    const exact = findLabel(opState);
    if (exact) return exact;
  }

  if (status === 'active') {
    return findLabel('1') || fallback.active;
  }

  return findLabel('0') || fallback.inactive;
}
