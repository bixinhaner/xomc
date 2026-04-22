import { useMemo, useState } from 'react'
import { RefreshCcw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  EmptyRow,
  ErrorRow,
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
  formatTime,
} from '@/components/layout/PageShell'

import { useLicenses } from '@core/hooks/api/useLicense'

export function LicensePage() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const params = useMemo(() => ({ page, pageSize }), [page, pageSize])

  const { data, isLoading, isError, error, isFetching, refetch } = useLicenses(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['许可证', '产品', '类型', '状态', '使用量', '到期时间']

  return (
    <PageShell
      title="许可证"
      description="软件许可证池 · 容量、过期、特性"
      isFetching={isFetching}
      toolbar={
        <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
          <RefreshCcw /> 刷新
        </Button>
      }
    >
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无许可证</EmptyRow>
            ) : (
              rows.map((l) => {
                const pct = l.maxDevices > 0 ? (l.usedDevices / l.maxDevices) * 100 : 0
                const daysLeft = l.expiryDate
                  ? Math.floor((new Date(l.expiryDate).getTime() - Date.now()) / 86_400_000)
                  : null
                return (
                  <TableRow key={l.id}>
                    <TableCell>
                      <div className="font-medium">{l.licenseName}</div>
                      <div className="font-mono text-xs text-muted-foreground">
                        {l.licenseCode}
                      </div>
                    </TableCell>
                    <TableCell>{l.productName}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{l.licenseType}</Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant={l.status === 'active' ? 'success' : 'muted'}>
                        {l.status}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2 text-xs">
                        <span className="tabular-nums">
                          {l.usedDevices}/{l.maxDevices}
                        </span>
                        <div className="h-1.5 w-20 overflow-hidden rounded bg-muted">
                          <div
                            className={
                              pct >= 90
                                ? 'h-full bg-destructive'
                                : pct >= 70
                                  ? 'h-full bg-amber-500'
                                  : 'h-full bg-emerald-500'
                            }
                            style={{ width: `${Math.min(100, pct)}%` }}
                          />
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {l.expiryDate ? formatTime(l.expiryDate) : '永久'}
                      {daysLeft !== null && daysLeft < 30 && daysLeft >= 0 ? (
                        <span className="ml-2 text-amber-600">还剩 {daysLeft} 天</span>
                      ) : null}
                      {daysLeft !== null && daysLeft < 0 ? (
                        <span className="ml-2 text-destructive">已过期</span>
                      ) : null}
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={pageSize} onChange={setPage} />
    </PageShell>
  )
}
