// 日志模块内部格式化工具（仅本模块使用）

// 秒数 → 人类可读时长（重启前运行时长），对齐 v1 AbnormalReboot 的 formatRuntime 语义。
export function formatRuntime(seconds: number): string {
  if (!seconds || seconds <= 0) return '—'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const parts: string[] = []
  if (days > 0) parts.push(`${days} 天`)
  if (hours > 0) parts.push(`${hours} 小时`)
  if (minutes > 0 && days === 0) parts.push(`${minutes} 分`)
  return parts.length > 0 ? parts.join(' ') : '< 1 分'
}
