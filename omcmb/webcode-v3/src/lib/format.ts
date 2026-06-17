import { formatSystemTime } from '@core/utils/systemTime'

export function formatBytes(bytes?: number | null) {
  if (bytes == null) return '—'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = bytes
  while (v > 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(1)} ${units[i]}`
}

/**
 * 格式化后端时间（#459 子单 D）：保留后端返回的系统时区钟面，**不**再用 new Date()
 * 按浏览器本地二次转换（那会抵消子单 B 的出口转换）。统一走共享 formatSystemTime。
 * 空值占位沿用 v3 习惯的 '—'。
 */
export function formatTime(iso?: string | null) {
  return formatSystemTime(iso, { placeholder: '—' })
}

export function formatHHMMSS(date: Date = new Date()) {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

export function formatDateLong(date: Date = new Date()) {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}.${pad(date.getMonth() + 1)}.${pad(date.getDate())}`
}

/** 把数值百分位映射为 "ok / warn / crit" */
export function levelOf(v: number, warn = 0.7, crit = 0.9): 'ok' | 'warn' | 'crit' {
  if (v >= crit) return 'crit'
  if (v >= warn) return 'warn'
  return 'ok'
}
