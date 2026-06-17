import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, Search, Trash2, Eye } from 'lucide-react'

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
import { cn } from '@/lib/utils'

import { useOpsTemplates, useDeleteOpsTemplates } from '@core/hooks/api/useOpsTools'
import type { OpsTemplate } from '@core/mock/data/opsTools'

// ============================================================
// 运维模板库 — 对齐 v1 webcode/src/pages/ops/Templates
// 列表 + 筛选；点模板名进 /ops/templates/:id 详情看执行步骤
// ============================================================

const TEMPLATE_CATEGORIES = [
  '巡检运维',
  '故障处置',
  '性能优化',
  '软件管理',
  '网络配置',
  '维护操作',
] as const

const CATEGORY_VARIANT: Record<
  string,
  'default' | 'warning' | 'success' | 'destructive' | 'secondary'
> = {
  巡检运维: 'default',
  故障处置: 'destructive',
  性能优化: 'success',
  软件管理: 'secondary',
  网络配置: 'default',
  维护操作: 'warning',
}

const DEVICE_TYPES = ['eNB', 'gNB', 'CPE', 'eGW'] as const

function formatDuration(seconds: number): string {
  if (!seconds) return '—'
  if (seconds >= 60) return `${Math.floor(seconds / 60)} 分`
  return `${seconds} 秒`
}

export default function Templates() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const [category, setCategory] = useState('')
  const [targetDeviceType, setTargetDeviceType] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(category ? { category } : {}),
      ...(targetDeviceType ? { targetDeviceType } : {}),
    }),
    [page, pageSize, keyword, category, targetDeviceType]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useOpsTemplates(params)
  const deleteTemplates = useDeleteOpsTemplates()

  const rows: OpsTemplate[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = [
    '模板',
    '分类',
    '适用设备',
    '步骤数',
    '预计耗时',
    '使用次数',
    '创建人',
    '更新时间',
    '操作',
  ]

  return (
    <PageShell title="运维模板库" description="标准化运维操作步骤模板，供自动化任务编排引用">
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-64 pl-9"
            placeholder="模板名称"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <Select
          value={category || 'all'}
          onValueChange={(v) => {
            setCategory(v === 'all' ? '' : v)
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="分类" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部分类</SelectItem>
            {TEMPLATE_CATEGORIES.map((c) => (
              <SelectItem key={c} value={c}>
                {c}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={targetDeviceType || 'all'}
          onValueChange={(v) => {
            setTargetDeviceType(v === 'all' ? '' : v)
            setPage(1)
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="适用设备" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部设备类型</SelectItem>
            {DEVICE_TYPES.map((d) => (
              <SelectItem key={d} value={d}>
                {d}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
          <RefreshCcw className={cn(isFetching && 'animate-spin')} /> 刷新
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
              <EmptyRow colSpan={cols.length}>暂无模板</EmptyRow>
            ) : (
              rows.map((t) => (
                <TableRow key={t.id}>
                  <TableCell>
                    <button
                      type="button"
                      className="text-left font-medium hover:underline"
                      onClick={() => navigate(`/ops/templates`)}
                    >
                      {t.templateName}
                    </button>
                    <div className="text-xs text-muted-foreground line-clamp-1">
                      {t.description}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={CATEGORY_VARIANT[t.category] ?? 'outline'}>
                      {t.category || '—'}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs">
                    <div className="flex flex-wrap gap-1">
                      {(t.targetDeviceTypes || []).length > 0
                        ? t.targetDeviceTypes.map((d) => (
                            <Badge key={d} variant="muted">
                              {d}
                            </Badge>
                          ))
                        : '—'}
                    </div>
                  </TableCell>
                  <TableCell className="text-xs tabular-nums">{t.steps?.length ?? 0}</TableCell>
                  <TableCell className="text-xs tabular-nums">
                    {formatDuration(t.estimatedDuration)}
                  </TableCell>
                  <TableCell className="text-xs tabular-nums">{t.useCount}</TableCell>
                  <TableCell className="text-xs">{t.creator}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(t.updateTime)}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => navigate(`/ops/templates`)}
                        title="详情"
                      >
                        <Eye className="size-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={deleteTemplates.isPending}
                        className="text-destructive hover:text-destructive"
                        onClick={() => {
                          if (window.confirm(`确认删除模板「${t.templateName}」？`)) {
                            deleteTemplates.mutate([t.id])
                          }
                        }}
                        title="删除"
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={pageSize} onChange={setPage} />
    </PageShell>
  )
}
