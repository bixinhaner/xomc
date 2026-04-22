import type { PageResponse } from '../types/pagination';

export function delay(min: number, max: number): Promise<void> {
  const ms = Math.floor(Math.random() * (max - min + 1)) + min;
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export function paginate<T>(items: T[], page: number, pageSize: number): PageResponse<T> {
  const start = (page - 1) * pageSize;
  const end = start + pageSize;
  return {
    items: items.slice(start, end),
    total: items.length,
    page,
    pageSize,
  };
}

export function sortBy<T>(items: T[], key: keyof T, order: 'ascend' | 'descend'): T[] {
  return [...items].sort((a, b) => {
    const av = a[key];
    const bv = b[key];
    if (av === null || av === undefined) return 1;
    if (bv === null || bv === undefined) return -1;
    let cmp = 0;
    if (typeof av === 'string' && typeof bv === 'string') {
      cmp = av.localeCompare(bv, 'zh-CN');
    } else if (typeof av === 'number' && typeof bv === 'number') {
      cmp = av - bv;
    } else {
      cmp = String(av).localeCompare(String(bv));
    }
    return order === 'ascend' ? cmp : -cmp;
  });
}

export function filterByText<T>(items: T[], key: keyof T, text: string): T[] {
  if (!text) return items;
  const lower = text.toLowerCase();
  return items.filter((item) => {
    const val = item[key];
    return val != null && String(val).toLowerCase().includes(lower);
  });
}

export function randomItem<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)];
}

export function randomBetween(min: number, max: number): number {
  return Math.random() * (max - min) + min;
}

export function generateId(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
}

export function generateTimeSeries(
  days: number,
  intervalMinutes: number,
  baseValue: number,
  variance: number
): [string, number][] {
  const result: [string, number][] = [];
  const now = Date.now();
  const totalPoints = Math.floor((days * 24 * 60) / intervalMinutes);
  const intervalMs = intervalMinutes * 60 * 1000;
  for (let i = totalPoints; i >= 0; i--) {
    const ts = new Date(now - i * intervalMs).toISOString();
    const value = Math.max(0, baseValue + (Math.random() - 0.5) * 2 * variance);
    result.push([ts, parseFloat(value.toFixed(2))]);
  }
  return result;
}

export function generateIP(): string {
  const octets = [
    randomBetween(10, 192) | 0,
    randomBetween(0, 255) | 0,
    randomBetween(0, 255) | 0,
    randomBetween(1, 254) | 0,
  ];
  return octets.join('.');
}

export function generateSN(type: string): string {
  const num = String(Math.floor(Math.random() * 99999) + 1).padStart(5, '0');
  switch (type) {
    case 'eNB':
      return `ENB${num}`;
    case 'gNB':
      return `GNB${num}`;
    case 'CPE':
      return `CPE${num}`;
    case 'eGW':
      return `EGW${num}`;
    default:
      return `DEV${num}`;
  }
}
