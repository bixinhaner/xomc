/**
 * 系统时区显示统一工具（issue #459，子单 D / 母单 #455）。
 *
 * 背景 / 设计要点：
 *   - 子单 B（#457）后，后端「成功响应出口」已统一把时间转换为**系统时区**的钟面 + 偏移：
 *       · 标准 time.Time 序列化为 RFC3339 带偏移，例如系统设 Asia/Tokyo 时返回
 *         "2026-06-16T10:00:00+09:00"（偏移已是系统时区）；
 *       · 自定义 model.Time 序列化为 "2006/1/2 15:04:05" 形态（如 "2026/6/16 10:00:00"），
 *         **不带偏移后缀**，但其钟面已是系统时区的本地钟面（B 转换层改了 Location）。
 *   - 因此前端**必须保留后端给的钟面**原样显示，**绝不能**再用
 *     `new Date(v).toLocaleString()` —— 那会按**浏览器 OS 时区**二次本地化，抵消 B 的转换。
 *
 * 解析规则（保留偏移 / 保留钟面，不做浏览器本地转换）：
 *   1. 带偏移的 ISO（含 'Z' 或 '+hh:mm' / '-hh:mm'）→ dayjs.parseZone 保留原偏移，
 *      直接取其钟面格式化（不调 .local()、不调 .tz()）。
 *   2. 无偏移的 "YYYY/M/D H:m:s" 或 "YYYY-MM-DD HH:mm:ss" 钟面串 → 按**字面钟面**解析
 *      （customParseFormat，不附加任何时区），原样格式化输出。
 *   3. epoch 毫秒 / 秒（number）→ 这类是绝对时刻、后端未做钟面转换，按提供的
 *      `systemTimezone`（IANA 名）转换到系统时区钟面显示；无系统时区时回落 UTC。
 *   4. Date 对象同 3。
 *
 * 之所以 1/2 不依赖 systemTimezone：后端已把钟面算好，前端只负责「原样呈现」。
 * 只有当输入是绝对时刻（epoch / Date）时才需要 systemTimezone 把它落到系统钟面。
 */
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import timezone from 'dayjs/plugin/timezone';
import customParseFormat from 'dayjs/plugin/customParseFormat';

dayjs.extend(utc);
dayjs.extend(timezone);
dayjs.extend(customParseFormat);

/** 默认显示格式（与全站旧 toLocaleString('zh-CN') 视觉近似的 24h 钟面）。 */
export const DEFAULT_TIME_FORMAT = 'YYYY-MM-DD HH:mm:ss';
/** 仅日期。 */
export const DATE_FORMAT = 'YYYY-MM-DD';
/** 仅时间（时:分:秒）。 */
export const TIME_FORMAT = 'HH:mm:ss';

/** 空值占位（与各页面历史习惯一致，调用方可覆盖）。 */
const PLACEHOLDER = '-';

/** 后端 model.Time 的无偏移钟面候选格式（按序尝试）。 */
const NAKED_WALLCLOCK_FORMATS = [
  'YYYY/M/D H:m:s',
  'YYYY/MM/DD HH:mm:ss',
  'YYYY-MM-DD HH:mm:ss',
  'YYYY-MM-DDTHH:mm:ss',
  'YYYY/M/D H:m',
  'YYYY-MM-DD HH:mm',
];

/** 判断字符串是否带时区偏移（Z 或 ±hh:mm / ±hhmm）。 */
function hasOffset(s: string): boolean {
  // 末尾 Z，或 ISO 时间部分后的 +/-hh(:?mm)
  return /Z$/i.test(s) || /[+-]\d{2}:?\d{2}$/.test(s);
}

/**
 * 从带偏移的 ISO 串解析偏移分钟数（东为正）。
 * "Z" → 0；"+09:00" → 540；"-05:30" → -330；无法解析返回 null。
 */
function parseIsoOffsetMinutes(s: string): number | null {
  if (/Z$/i.test(s)) return 0;
  const m = s.match(/([+-])(\d{2}):?(\d{2})$/);
  if (!m) return null;
  const sign = m[1] === '-' ? -1 : 1;
  const hh = Number(m[2]);
  const mm = Number(m[3]);
  if (Number.isNaN(hh) || Number.isNaN(mm)) return null;
  return sign * (hh * 60 + mm);
}

/**
 * 把任意后端时间值转成「保留系统时区钟面」的 dayjs 对象。
 * 返回 null 表示无法解析（调用方显示占位）。
 *
 * @param value 后端时间值：带偏移 ISO / 无偏移钟面串 / epoch number / Date
 * @param systemTimezone 系统时区 IANA 名（仅 number/Date 输入时用于落钟面），缺省时回落 UTC
 */
function toDisplayDayjs(
  value: string | number | Date | null | undefined,
  systemTimezone?: string | null
): dayjs.Dayjs | null {
  if (value === null || value === undefined || value === '') return null;

  // number / Date：绝对时刻，按系统时区落钟面
  if (typeof value === 'number' || value instanceof Date) {
    const base = dayjs(value);
    if (!base.isValid()) return null;
    if (systemTimezone) {
      const z = base.tz(systemTimezone);
      return z.isValid() ? z : base.utc();
    }
    return base.utc();
  }

  const s = String(value).trim();
  if (s === '') return null;

  // 1. 带偏移 ISO（如 "2026-06-16T10:00:00+09:00" / "...Z"）：后端已把钟面算成系统时区，
  //    直接取偏移标记**之前**的钟面字段原样显示，绝不按浏览器本地二次转换。
  //    用 utcOffset(offset, keepLocalTime=true) 把钟面固定在该偏移上，format 即得原钟面。
  if (hasOffset(s)) {
    const offsetMin = parseIsoOffsetMinutes(s);
    // 先按字面解析钟面字段（不带时区），再钉到原偏移，保证 format 输出 = 原钟面
    const wallPart = s.replace(/(Z|[+-]\d{2}:?\d{2})$/i, '');
    const wall = dayjs(wallPart);
    if (wall.isValid() && offsetMin !== null) {
      return wall.utcOffset(offsetMin, true);
    }
    return wall.isValid() ? wall : null;
  }

  // 2. 无偏移钟面串：按字面解析，不附加时区
  const wall = dayjs(s, NAKED_WALLCLOCK_FORMATS, true);
  if (wall.isValid()) return wall;

  // 兜底：宽松解析（仍不做本地转换，直接取钟面）
  const loose = dayjs(s);
  return loose.isValid() ? loose : null;
}

/**
 * 统一时间格式化：保留后端系统时区钟面，不做浏览器本地二次转换。
 *
 * @param value 后端时间值
 * @param options 可选：format 显示格式（默认 'YYYY-MM-DD HH:mm:ss'）、
 *                placeholder 空值占位（默认 '-'）、systemTimezone（仅 epoch/Date 输入时生效）
 * @returns 格式化后的字符串；无法解析返回 placeholder
 */
export function formatSystemTime(
  value: string | number | Date | null | undefined,
  options?: { format?: string; placeholder?: string; systemTimezone?: string | null }
): string {
  const d = toDisplayDayjs(value, options?.systemTimezone);
  if (!d) return options?.placeholder ?? PLACEHOLDER;
  return d.format(options?.format ?? DEFAULT_TIME_FORMAT);
}

/** 仅日期（YYYY-MM-DD）。 */
export function formatSystemDate(
  value: string | number | Date | null | undefined,
  options?: { placeholder?: string; systemTimezone?: string | null }
): string {
  return formatSystemTime(value, { ...options, format: DATE_FORMAT });
}

/** 仅时间（HH:mm:ss）。 */
export function formatSystemTimeOnly(
  value: string | number | Date | null | undefined,
  options?: { placeholder?: string; systemTimezone?: string | null }
): string {
  return formatSystemTime(value, { ...options, format: TIME_FORMAT });
}

/**
 * 取「当前时刻」在系统时区下的钟面（用于顶部实时时钟）。
 * systemTimezone 缺省时回落 UTC。
 */
export function nowInSystemTimezone(systemTimezone?: string | null): dayjs.Dayjs {
  const now = dayjs();
  if (systemTimezone) {
    const z = now.tz(systemTimezone);
    if (z.isValid()) return z;
  }
  return now.utc();
}

/**
 * 当前系统时区的 GMT 偏移文案（如 "GMT+09:00" / "UTC"）。
 * 用于顶部只读时钟旁标注时区。
 */
export function systemTimezoneLabel(systemTimezone?: string | null): string {
  const tz = systemTimezone?.trim();
  if (!tz || tz.toUpperCase() === 'UTC') return 'UTC';
  return tz;
}

/**
 * 把「用户在系统时区下选的钟面时刻」转为发给后端的 RFC3339（带系统时区偏移）。
 *
 * 用于时间范围筛选输入：用户在 UI 里选的是系统时区的钟面（如 Asia/Tokyo 10:00），
 * 必须附加系统时区偏移后再发后端（后端按 RFC3339 解析为 UTC）。
 * 若直接发本地钟面或浏览器本地偏移，会与系统时区错位。
 *
 * @param wallClock 用户选的钟面（Date / dayjs / 'YYYY-MM-DD HH:mm:ss' 串），其字段值
 *                  视为**系统时区下的钟面**
 * @param systemTimezone 系统时区 IANA 名；缺省回落 UTC
 * @returns RFC3339 字符串（带系统时区偏移），无效输入返回 null
 */
export function toSystemTimezoneRFC3339(
  wallClock: string | number | Date | dayjs.Dayjs | null | undefined,
  systemTimezone?: string | null
): string | null {
  if (wallClock === null || wallClock === undefined || wallClock === '') return null;

  // 取出钟面的年月日时分秒（字面字段，不带任何时区语义）
  let parts: dayjs.Dayjs;
  if (dayjs.isDayjs(wallClock)) {
    parts = wallClock;
  } else if (wallClock instanceof Date) {
    parts = dayjs(wallClock);
  } else if (typeof wallClock === 'number') {
    parts = dayjs(wallClock);
  } else {
    parts = dayjs(String(wallClock), NAKED_WALLCLOCK_FORMATS, true);
    if (!parts.isValid()) parts = dayjs(String(wallClock));
  }
  if (!parts.isValid()) return null;

  const wallStr = parts.format('YYYY-MM-DD HH:mm:ss');
  const tz = systemTimezone?.trim() || 'UTC';
  // 把这串钟面「解释为系统时区下的本地时刻」，再输出带偏移的 RFC3339
  const zoned = dayjs.tz(wallStr, 'YYYY-MM-DD HH:mm:ss', tz);
  if (!zoned.isValid()) {
    // tz 非法时回落 UTC
    return dayjs.utc(wallStr, 'YYYY-MM-DD HH:mm:ss').format();
  }
  return zoned.format(); // dayjs.tz 的 format() 默认输出带该时区偏移的 ISO
}
