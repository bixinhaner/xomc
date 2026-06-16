/**
 * 系统时区下拉选项生成（issue #456 子单 A，三皮肤共享）。
 *
 * 用浏览器内置 `Intl.supportedValuesOf('timeZone')` 生成完整 IANA 时区列表，
 * 每项带当前 GMT 偏移说明（如 "GMT+08:00"），按偏移升序排序。
 * 字段仍是 timezoneCode（IANA 名，如 "Asia/Shanghai"），保存链路不变。
 *
 * 兼容回退：若运行环境不支持 `Intl.supportedValuesOf`（旧浏览器），
 * 回落到一组内置常用时区，保证下拉始终可用、不报错。
 */

export interface TimezoneOption {
  /** 选项展示文本，如 "(GMT+08:00) Asia/Shanghai" */
  label: string;
  /** 选项值 = IANA 时区名，如 "Asia/Shanghai" */
  value: string;
}

/** 不支持 Intl.supportedValuesOf 时的内置回退列表（IANA 名）。 */
const FALLBACK_ZONES = [
  'UTC',
  'Asia/Shanghai',
  'Asia/Tokyo',
  'Asia/Kolkata',
  'Europe/London',
  'Europe/Paris',
  'America/New_York',
  'America/Los_Angeles',
  'Australia/Sydney',
];

/**
 * 取指定 IANA 时区当前的 GMT 偏移分钟数（东为正）。
 * 解析失败返回 null（该项视为偏移 0 排末尾，label 不带偏移）。
 */
export function timezoneOffsetMinutes(timeZone: string, at: Date = new Date()): number | null {
  try {
    const dtf = new Intl.DateTimeFormat('en-US', {
      timeZone,
      timeZoneName: 'shortOffset',
      hour: '2-digit',
    });
    const parts = dtf.formatToParts(at);
    const tzName = parts.find((p) => p.type === 'timeZoneName')?.value ?? '';
    // tzName 形如 "GMT+8" / "GMT+05:30" / "GMT" / "UTC"
    const m = /GMT([+-])(\d{1,2})(?::(\d{2}))?/.exec(tzName);
    if (!m) {
      // "GMT" / "UTC" 无符号 => 0 偏移
      if (/^(GMT|UTC)$/.test(tzName)) return 0;
      return null;
    }
    const sign = m[1] === '-' ? -1 : 1;
    const hh = Number(m[2]);
    const mm = m[3] ? Number(m[3]) : 0;
    return sign * (hh * 60 + mm);
  } catch {
    return null;
  }
}

/** 把偏移分钟数格式化成 "GMT+08:00" / "GMT-05:30" / "GMT+00:00"。 */
function formatOffset(minutes: number): string {
  const sign = minutes < 0 ? '-' : '+';
  const abs = Math.abs(minutes);
  const hh = String(Math.floor(abs / 60)).padStart(2, '0');
  const mm = String(abs % 60).padStart(2, '0');
  return `GMT${sign}${hh}:${mm}`;
}

/** 取全量 IANA 时区名列表，不支持时回落内置列表。 */
function supportedTimeZones(): string[] {
  const intlAny = Intl as unknown as {
    supportedValuesOf?: (key: string) => string[];
  };
  if (typeof intlAny.supportedValuesOf === 'function') {
    try {
      const zones = intlAny.supportedValuesOf('timeZone');
      if (Array.isArray(zones) && zones.length > 0) {
        return ensureUTC(zones);
      }
    } catch {
      // 落到回退
    }
  }
  return ensureUTC(FALLBACK_ZONES);
}

// 确保列表含 "UTC"：部分运行环境 supportedValuesOf 只给 "Etc/UTC" 不给 "UTC"，
// 而系统时区种子默认值正是 "UTC"，下拉必须能选中该值以正确回显 / 保存。
function ensureUTC(zones: string[]): string[] {
  return zones.includes('UTC') ? zones : ['UTC', ...zones];
}

/**
 * 生成完整 IANA 时区下拉选项，带 GMT 偏移、按偏移升序排序。
 * @param at 计算偏移的参考时刻（默认当前），便于测试稳定取值。
 */
export function buildTimezoneOptions(at: Date = new Date()): TimezoneOption[] {
  const zones = supportedTimeZones();
  const withOffset = zones.map((zone) => {
    const off = timezoneOffsetMinutes(zone, at);
    const label = off === null ? zone : `(${formatOffset(off)}) ${zone}`;
    return { value: zone, offset: off ?? 0, label };
  });
  // 按偏移升序，偏移相同按 IANA 名字典序，保证稳定排序。
  withOffset.sort((a, b) => {
    if (a.offset !== b.offset) return a.offset - b.offset;
    return a.value.localeCompare(b.value);
  });
  return withOffset.map(({ label, value }) => ({ label, value }));
}
