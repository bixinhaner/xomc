import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
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
import { cn } from '@/lib/utils'

import { useProvisioningTasks } from '@core/hooks/api/useProvisioning'
import type { ProvisioningTask } from '@core/services/api/provisionApi'

// ============================================================
// 自动开站 — 对照 v1 config/AutoProvisioning（F09）。
// 真实数据：useProvisioningTasks（10s 轮询）；状态筛选 + 进度。
// 行 → 链接到任务详情 /config/auto-provision/:id。
// ============================================================

const STATUS_VARIANT: Record<
  string,
  'success' | 'warning' | 'destructive' | 'secondary' | 'muted'
> = {
  completed: 'success',
  failed: 'destructive',
  discovered: 'secondary',
  identifying: 'warning',
  matching: 'warning',
  configuring: 'warning',
  verifying: 'warning',
  discovering: 'warning',
  syncing: 'warning',
}

const STATUS_OPTIONS = [
  'discovered',
  'identifying',
  'matching',
  'configuring',
  'verifying',
  'completed',
  'failed',
]

export default function AutoProvisioning() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [status, setStatus] = useState('')

  const queryParams = useMemo(
    () => ({ page, pageSize, ...(status ? { status } : {}) }),
    [page, status]
  )
  const { data, isLoading, isError, error, isFetching, refetch } =
    useProvisioningTasks(queryParams)
  const rows: ProvisioningTask[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['任务 ID', '设备 ID', '状态', '进度', '重试', '创建时间', '完成时间']

  return (
    <PageShell
      title="自动开站"
      description="设备自动发现 / 模板匹配 / 配置下发任务（F09）"
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Select
            value={status || 'all'}
            onValueChange={(v) => {
              setStatus(v === 'all' ? '' : v)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-40">
              <SelectValue placeholder="状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              {STATUS_OPTIONS.map((s) => (
                <SelectItem key={s} value={s}>
                  {s}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => void refetch()}>
              <RefreshCcw className="size-4" /> 刷新
            </Button>
          </div>
        </div>
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
              <EmptyRow colSpan={cols.length}>暂无开站任务</EmptyRow>
            ) : (
              rows.map((t) => {
                const pct =
                  t.totalSteps > 0 ? Math.round((t.currentStep / t.totalSteps) * 100) : 0
                return (
                  <TableRow key={t.id}>
                    <TableCell>
                      <button
                        type="button"
                        className="font-mono text-xs text-primary hover:underline"
                        onClick={() => navigate(`/config/auto-provision`)}
                      >
                        {t.id}
                      </button>
                    </TableCell>
                    <TableCell className="font-mono text-xs">{t.deviceId}</TableCell>
                    <TableCell>
                      <Badge variant={STATUS_VARIANT[t.status] ?? 'secondary'}>{t.status}</Badge>
                    </TableCell>
                    <TableCell className="w-40">
                      <div className="flex items-center gap-2">
                        <div className="h-1.5 w-20 overflow-hidden rounded-full bg-muted">
                          <div
                            className={cn(
                              'h-full rounded-full',
                              t.status === 'failed' ? 'bg-destructive' : 'bg-primary'
                            )}
                            style={{ width: `${pct}%` }}
                          />
                        </div>
                        <span className="text-xs tabular-nums text-muted-foreground">
                          {t.currentStep}/{t.totalSteps}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell className="text-xs tabular-nums">
                      {t.retryCount}/{t.maxRetries}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(t.createdAt)}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(t.completedAt)}
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
