import { Loader2 } from 'lucide-react'
import type { ReactNode } from 'react'

import { cn } from '@/lib/utils'
import { formatSystemTime } from '@core/utils/systemTime'

export function PageShell({
  title,
  description,
  toolbar,
  isFetching,
  children,
  className,
}: {
  title: string
  description?: string
  toolbar?: ReactNode
  isFetching?: boolean
  children: ReactNode
  className?: string
}) {
  return (
    <div className={cn('mx-auto max-w-[1400px] px-6 py-8', className)}>
      <div className="mb-6 flex items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
          {description && (
            <p className="mt-1 text-sm text-muted-foreground">{description}</p>
          )}
        </div>
        {isFetching ? (
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <Loader2 className="size-3.5 animate-spin" />
            加载中
          </div>
        ) : null}
      </div>
      {toolbar ? <div className="mb-3 flex flex-wrap items-center gap-2">{toolbar}</div> : null}
      {children}
    </div>
  )
}

export function TableCard({ children }: { children: ReactNode }) {
  return <div className="overflow-hidden rounded-lg border bg-card">{children}</div>
}

export function EmptyRow({ colSpan, children }: { colSpan: number; children: ReactNode }) {
  return (
    <tr>
      <td colSpan={colSpan} className="h-32 p-3 text-center text-muted-foreground">
        {children}
      </td>
    </tr>
  )
}

export function LoadingRow({ colSpan }: { colSpan: number }) {
  return (
    <tr>
      <td colSpan={colSpan} className="h-32 p-3 text-center">
        <Loader2 className="mx-auto size-5 animate-spin text-muted-foreground" />
      </td>
    </tr>
  )
}

export function ErrorRow({
  colSpan,
  error,
}: {
  colSpan: number
  error: unknown
}) {
  const msg = error instanceof Error ? error.message : '未知错误'
  return (
    <tr>
      <td colSpan={colSpan} className="h-32 p-3 text-center text-destructive">
        加载失败：{msg}
      </td>
    </tr>
  )
}

import { Button } from '@/components/ui/button'

export function Pagination({
  page,
  totalPages,
  pageSize,
  onChange,
}: {
  page: number
  totalPages: number
  pageSize: number
  onChange: (p: number) => void
}) {
  return (
    <div className="mt-4 flex items-center justify-between text-sm">
      <span className="text-muted-foreground">
        第 {page} / {totalPages} 页 · 每页 {pageSize} 条
      </span>
      <div className="flex items-center gap-2">
        <Button
          size="sm"
          variant="outline"
          disabled={page <= 1}
          onClick={() => onChange(Math.max(1, page - 1))}
        >
          上一页
        </Button>
        <Button
          size="sm"
          variant="outline"
          disabled={page >= totalPages}
          onClick={() => onChange(Math.min(totalPages, page + 1))}
        >
          下一页
        </Button>
      </div>
    </div>
  )
}

export function formatBytes(bytes?: number | null) {
  if (!bytes) return '—'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = bytes
  while (v > 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(1)} ${units[i]}`
}

export function formatTime(iso?: string | null) {
  // #459 子单 D：保留后端系统时区钟面，不按浏览器本地二次转换。沿用 v2 的 YYYY-MM-DD HH:mm 视觉。
  return formatSystemTime(iso, { format: 'YYYY-MM-DD HH:mm', placeholder: '—' })
}
