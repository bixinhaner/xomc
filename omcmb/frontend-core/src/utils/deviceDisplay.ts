import type { Locale } from './i18nText';

const HAN_RE = /[\u4e00-\u9fff]/;

export function containsHan(text: string | null | undefined): boolean {
  return HAN_RE.test(text ?? '');
}

export function localizeDeviceProductName(name: string | null | undefined, locale: Locale): string {
  const raw = (name ?? '').trim();
  if (!raw) return '-';
  if (locale === 'zh-CN') return raw;

  return raw
    .replace(/\s*产品$/u, ' Product')
    .replace(/\s*设备$/u, ' Device')
    .replace(/\s*系列$/u, ' Series');
}
