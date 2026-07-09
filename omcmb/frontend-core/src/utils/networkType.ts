/**
 * 基站制式（network_type）显示工具。
 *
 * 背景（issue #223）：设备列表与回收站两处「基站制式」列原本各算各的——
 * 列表直接显示 device.networkType（eNB/gNB），回收站由 product_class 字符串
 * 启发式推导（gnb→gNB / enb→eNB / 否则 CPE），导致同一设备「列表 eNB、
 * 回收站 CPE」不一致。
 *
 * 治法：两处统一走 device.networkType 字段 + 同一份 network_type 字典
 * value→label 映射（与筛选下拉同源），文案天然一致。
 *
 * 字段对齐细节：
 * - device.networkType 取值为 'eNB' / 'gNB' / 'GSM'（deviceApi.toRadioMode 由
 *   后端 technology 'lte'/'nr'/'gsm' 翻译而来）。
 * - network_type 字典的 value 为 'lte' / 'nr'（= 后端 technology 原值），
 *   label 为 'eNB (LTE)' / 'gNB (NR)'。
 * - 所以先把 networkType 反向归一回字典 value，再查字典 label；查不到时回退
 *   原始 networkType 值，避免显示空白。
 */

/** network_type 字典项的最小形状（只关心 value→label）。 */
import { getI18nText, type Locale } from './i18nText';

export interface NetworkTypeDictDetail {
  label: string;
  labelI18n?: Record<string, string>;
  value: string;
}

/** device.networkType（eNB/gNB/GSM）→ 字典 value（technology：lte/nr/gsm）。 */
const RADIO_MODE_TO_DICT_VALUE: Record<string, string> = {
  eNB: 'lte',
  gNB: 'nr',
  GSM: 'gsm',
};

/** 制式码（technology：lte/nr/gsm）→ device.networkType（eNB/gNB/GSM）。 */
const DICT_VALUE_TO_RADIO_MODE: Record<string, string> = {
  lte: 'eNB',
  nr: 'gNB',
  gsm: 'GSM',
};

/**
 * 把设备列表过滤入参的「制式」归一到 device.networkType 字段实际存储的值
 * （eNB/gNB/GSM）。
 *
 * 背景（#443）：设备列表过滤入参的 networkType 可能是制式码（'lte'/'nr'/'gsm'，
 * 与后端 technology 同源）或老链路的基站类型码（'eNB'/'gNB'/'GSM'）。而
 * device.networkType 字段（deviceApi.toRadioMode 由后端 technology 翻译而来）
 * 恒为 'eNB'/'gNB'/'GSM'。mock 过滤层若直接 `d.networkType === params.networkType`
 * 比对，传入 'lte' 永不命中 'eNB'。此函数统一把两种入参都归一到基站类型码，
 * 与真实 deviceApi（接受 lte/nr/gsm 并兼容 eNB/gNB）的过滤语义对齐。
 *
 * @param networkType 过滤入参（'lte'/'nr'/'gsm' 或 'eNB'/'gNB'/'GSM' 或其它）
 * @returns           归一后的基站类型码；无法识别时回退原值
 */
export function normalizeNetworkTypeFilter(networkType: string): string {
  return DICT_VALUE_TO_RADIO_MODE[networkType] ?? networkType;
}

/**
 * 把设备的 networkType 字段值映射为 network_type 字典 label（与筛选下拉同源）。
 *
 * @param networkType 设备的 networkType 字段（'eNB' / 'gNB' / 'GSM' / 其它）
 * @param details     network_type 字典明细（来自 useDictionaryBatch 的
 *                    sysDictionaryDetails），缺省时安全回退
 * @returns           字典 label（如 'eNB (LTE)'）；无字典 / 无匹配时回退原值；
 *                    networkType 为空时返回 '-'
 */
export function resolveNetworkTypeLabel(
  networkType: string | undefined | null,
  details: NetworkTypeDictDetail[] | undefined,
  locale: Locale = 'zh-CN',
): string {
  if (!networkType) return '-';

  const dictValue = RADIO_MODE_TO_DICT_VALUE[networkType] ?? networkType;
  const matched = (details ?? []).find(
    (d) => d.value === dictValue || d.value === networkType,
  );
  return matched ? getI18nText(matched.labelI18n, locale, matched.label) : networkType;
}
