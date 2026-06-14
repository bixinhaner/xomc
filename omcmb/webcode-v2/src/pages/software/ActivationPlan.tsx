import { useMemo, useState } from 'react'
import { CalendarClock, CheckCircle2, ListChecks, Play, RefreshCcw, XCircle } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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

import { useUpgradePlans, useCancelUpgradePlan } from '@core/hooks/api/useSoftware'
import type { UpgradePlan, UpgradePlanStatus } from '@core/mock/data/software'

import { PLAN_STATUS, progressPct } from './_shared'
import { IconBtn, ProgressBar, Stat } from './_components'

// ===========================================================================
// 激活计划 — 对照 v1 webcode/src/pages/software/ActivationPlan
// 版本激活 / 升级计划（含排期），数据走真实 useUpgradePlans（旧版计划表）。
// 排期/执行中计划可取消（useCancelUpgradePlan）。
// ===========================================================================

export default function ActivationPlan() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [status, setStatus] = useState<'' | UpgradePlanStatus>('')
  const [search, setSearch] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(status ? { status } : {}),
    }),
    [page, pageSize, status]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useUpgradePlans(params)
  const cancel = useCancelUpgradePlan()

  const allRows = data?.items ?? []
  const kw = search.trim().toLowerCase()
  const rows = kw
    ? allRows.filter(
        (p) =>
          p.planName.toLowerCase().includes(kw) ||
          p.targetVersionCode.toLowerCase().includes(kw)
      )
    : allRows
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const scheduled = allRows.filter((p) => p.status === 'scheduled').length
  const running = allRows.filter((p) => p.status === 'running').length
  const done = allRows.filter((p) => p.status === 'success' || p.status === 'partial').length

  const cols = ['计划名称', '目标版本', '设备数', '状态', '进度', '排期/开始', '创建人', '操作']

  return (
    <PageShell title="激活计划" description="版本激活 / 升级计划（含排期），可取消未执行完成的计划">
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="计划总数" value={total} icon={ListChecks} />
        <Stat label="已排期" value={scheduled} icon={CalendarClock} />
        <Stat label="执行中" value={running} icon={Play} tone="amber" />
        <Stat label="已完成" value={done} icon={CheckCircle2} tone="emerald" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <Input
          className="w-72"
          placeholder="搜索计划名称 / 目标版本"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <Select
          value={status || 'all'}
          onValueChange={(v) => {
            setStatus(v === 'all' ? '' : (v as UpgradePlanStatus))
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="pending">等待中</SelectItem>
            <SelectItem value="scheduled">已排期</SelectItem>
            <SelectItem value="running">执行中</SelectItem>
            <SelectItem value="success">成功</SelectItem>
            <SelectItem value="partial">部分成功</SelectItem>
            <SelectItem value="failed">失败</SelectItem>
            <SelectItem value="cancelled">已取消</SelectItem>
          </SelectContent>
        </Select>
        <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
          <RefreshCcw className={isFetching ? 'animate-spin' : ''} /> 刷新
        </Button>
      </div>

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
              <EmptyRow colSpan={cols.length}>暂无激活计划</EmptyRow>
            ) : (
              rows.map((p: UpgradePlan) => {
                const st = PLAN_STATUS[p.status]
                const cancellable =
                  p.status === 'pending' || p.status === 'scheduled' || p.status === 'running'
                return (
                  <TableRow key={p.id}>
                    <TableCell className="max-w-[220px] truncate font-medium" title={p.planName}>
                      {p.planName}
                    </TableCell>
                    <TableCell className="font-mono text-xs">{p.targetVersionCode || '—'}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {p.totalCount || p.deviceSns.length} 台
                    </TableCell>
                    <TableCell>
                      <Badge variant={st.variant}>{st.label}</Badge>
                    </TableCell>
                    <TableCell className="w-40">
                      <ProgressBar
                        pct={
                          p.progress ||
                          progressPct({
                            totalCount: p.totalCount,
                            successCount: p.successCount,
                            failCount: p.failCount,
                          })
                        }
                      />
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(p.startTime || p.scheduledTime)}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">{p.creator || '—'}</TableCell>
                    <TableCell>
                      {cancellable ? (
                        <IconBtn
                          title="取消计划"
                          tone="destructive"
                          disabled={cancel.isPending}
                          onClick={() => cancel.mutate(p.id)}
                        >
                          <XCircle className="size-3.5" />
                        </IconBtn>
                      ) : (
                        <span className="text-xs text-muted-foreground">—</span>
                      )}
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
