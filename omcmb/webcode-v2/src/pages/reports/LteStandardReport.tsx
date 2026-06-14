import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, Plus, Download, FileBarChart2, FolderTree } from 'lucide-react'

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
  formatBytes,
  formatTime,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useReportDefinitions,
  useReportRecords,
  useGenerateReport,
  useDownloadReport,
} from '@core/hooks/api/useReports'
import type { ReportRecord } from '@core/mock/data/reports'

// ============================================================
// LTE 标准报表 (report/lte-standard) — v2
// 业务对齐 v1 webcode/src/pages/report/LTEStandardReport：
//   左侧报表分类树（KPI / 可用性 / 容量 / 质量 / 移动性）+ 右侧记录列表。
// 数据：真实 @core hooks（useReportDefinitions / useReportRecords /
//   useGenerateReport / useDownloadReport）。分类树作为本地视图过滤维度，
//   将记录按 reportType + period 归桶到对应分类（无服务端分类端点，按记录字段映射）。
//   行点击进入 /reports/record/:id 详情。
// ============================================================

interface CategoryNode {
  key: string
  label: string
  children?: CategoryNode[]
  /** 命中本分类的判定（基于记录 reportType + period） */
  match?: (rec: ReportRecord) => boolean
}

function periodKind(period: string): 'daily' | 'weekly' | 'monthly' | 'other' {
  const p = period.toLowerCase()
  if (p.includes('w')) return 'weekly'
  if (/^\d{4}-\d{2}$/.test(period) || /q[1-4]/i.test(period)) return 'monthly'
  if (/^\d{4}-\d{2}-\d{2}$/.test(period)) return 'daily'
  return 'other'
}

const CATEGORY_TREE: CategoryNode[] = [
  {
    key: 'kpi',
    label: 'KPI 报表',
    children: [
      { key: 'kpi-daily', label: 'KPI 日报', match: (r) => periodKind(r.period) === 'daily' },
      { key: 'kpi-weekly', label: 'KPI 周报', match: (r) => periodKind(r.period) === 'weekly' },
      { key: 'kpi-monthly', label: 'KPI 月报', match: (r) => periodKind(r.period) === 'monthly' },
    ],
  },
  {
    key: 'availability',
    label: '可用性报表',
    children: [
      { key: 'avail-station', label: '基站可用性', match: (r) => r.reportName.includes('可用') },
      { key: 'avail-cell', label: '小区可用性', match: (r) => r.reportName.includes('小区') },
    ],
  },
  {
    key: 'capacity',
    label: '容量报表',
    children: [
      { key: 'cap-prb', label: 'PRB 利用率', match: (r) => r.reportName.includes('PRB') || r.reportName.includes('容量') },
      { key: 'cap-user', label: '用户容量', match: (r) => r.reportName.includes('用户') },
    ],
  },
  {
    key: 'quality',
    label: '质量报表',
    children: [
      { key: 'qual-voice', label: '语音质量', match: (r) => r.reportName.includes('VoLTE') || r.reportName.includes('语音') },
      { key: 'qual-data', label: '数据质量', match: (r) => r.reportName.includes('数据') },
    ],
  },
]

const RECORD_STATUS_LABEL: Record<ReportRecord['status'], string> = {
  generating: '生成中',
  ready: '就绪',
  failed: '失败',
}

const RECORD_STATUS_VARIANT: Record<
  ReportRecord['status'],
  'success' | 'warning' | 'destructive'
> = {
  generating: 'warning',
  ready: 'success',
  failed: 'destructive',
}

const PAGE_SIZE = 50

export default function LteStandardReport() {
  const navigate = useNavigate()
  const [selectedKey, setSelectedKey] = useState('kpi-daily')
  const [page, setPage] = useState(1)

  // 拉全量记录（PAGE_SIZE 足够覆盖本视图），在前端按分类树过滤。
  const recordsQuery = useReportRecords({ page: 1, pageSize: PAGE_SIZE })
  // 探测定义端点，用于「生成报表」操作（取首个可用定义）。
  const definitionsQuery = useReportDefinitions({ page: 1, pageSize: PAGE_SIZE })
  const generate = useGenerateReport()
  const download = useDownloadReport()

  const allRecords = recordsQuery.data?.items ?? []
  const defs = definitionsQuery.data?.items ?? []

  // 找到当前选中的叶子分类匹配器。
  const matcher = useMemo(() => {
    for (const top of CATEGORY_TREE) {
      for (const child of top.children ?? []) {
        if (child.key === selectedKey) return child.match
      }
      if (top.key === selectedKey) {
        // 选中父节点：合并所有子匹配器。
        const childMatchers = (top.children ?? []).map((c) => c.match).filter(Boolean) as ((r: ReportRecord) => boolean)[]
        return (r: ReportRecord) => childMatchers.some((m) => m(r))
      }
    }
    return undefined
  }, [selectedKey])

  const filtered = matcher ? allRecords.filter(matcher) : allRecords
  const total = filtered.length
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const start = (page - 1) * PAGE_SIZE
  const paginated = filtered.slice(start, start + PAGE_SIZE)

  const handleGenerate = () => {
    const firstDef = defs[0]
    if (!firstDef || generate.isPending) return
    generate.mutate({ definitionId: firstDef.id })
  }

  const handleDownload = (rec: ReportRecord, e: React.MouseEvent) => {
    e.stopPropagation()
    if (download.isPending || rec.status !== 'ready') return
    download.mutate(rec.id, {
      onSuccess: (res) => {
        if (res?.url) window.open(res.url, '_blank', 'noopener,noreferrer')
      },
    })
  }

  const isFetching =
    recordsQuery.isFetching ||
    definitionsQuery.isFetching ||
    generate.isPending ||
    download.isPending

  return (
    <PageShell
      title="LTE 标准报表"
      description={`按分类浏览生成记录 · 当前分类 ${total} 份`}
      isFetching={isFetching}
      toolbar={
        <>
          <Button
            size="sm"
            disabled={defs.length === 0 || generate.isPending}
            onClick={handleGenerate}
          >
            <Plus /> 生成报表
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="ml-auto"
            onClick={() => {
              void recordsQuery.refetch()
              void definitionsQuery.refetch()
            }}
          >
            <RefreshCcw /> 刷新
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4 md:grid-cols-[220px_1fr]">
        {/* 左侧分类树 */}
        <aside className="rounded-lg border bg-card p-2">
          <div className="mb-2 flex items-center gap-1.5 px-2 py-1 text-xs font-medium uppercase tracking-wider text-muted-foreground">
            <FolderTree className="size-3.5" /> 报表分类
          </div>
          <nav className="space-y-1">
            {CATEGORY_TREE.map((top) => (
              <div key={top.key}>
                <div className="px-2 py-1 text-xs font-medium text-muted-foreground">
                  {top.label}
                </div>
                <div className="ml-2 space-y-0.5">
                  {(top.children ?? []).map((child) => (
                    <button
                      key={child.key}
                      type="button"
                      onClick={() => {
                        setSelectedKey(child.key)
                        setPage(1)
                      }}
                      className={cn(
                        'flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-left text-sm transition-colors',
                        selectedKey === child.key
                          ? 'bg-primary/10 text-primary'
                          : 'text-foreground hover:bg-muted',
                      )}
                    >
                      <FileBarChart2 className="size-3.5 shrink-0 opacity-70" />
                      {child.label}
                    </button>
                  ))}
                </div>
              </div>
            ))}
          </nav>
        </aside>

        {/* 右侧记录列表 */}
        <div>
          <TableCard>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>报表名称</TableHead>
                  <TableHead>周期</TableHead>
                  <TableHead>格式</TableHead>
                  <TableHead>大小</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>生成时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {recordsQuery.isLoading ? (
                  <LoadingRow colSpan={7} />
                ) : recordsQuery.isError ? (
                  <ErrorRow colSpan={7} error={recordsQuery.error} />
                ) : paginated.length === 0 ? (
                  <EmptyRow colSpan={7}>该分类暂无报表记录</EmptyRow>
                ) : (
                  paginated.map((r) => (
                    <TableRow
                      key={r.id}
                      className="cursor-pointer"
                      onClick={() => navigate(`/reports/record/${r.id}`)}
                    >
                      <TableCell className="font-medium text-primary hover:underline">
                        {r.reportName}
                      </TableCell>
                      <TableCell className="text-xs">{r.period || '—'}</TableCell>
                      <TableCell>
                        <Badge variant="muted" className="uppercase">
                          {r.format}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-xs tabular-nums text-muted-foreground">
                        {formatBytes(r.fileSize)}
                      </TableCell>
                      <TableCell>
                        <Badge variant={RECORD_STATUS_VARIANT[r.status]}>
                          {RECORD_STATUS_LABEL[r.status]}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {formatTime(r.generateTime)}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center justify-end">
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={r.status !== 'ready' || download.isPending}
                            onClick={(e) => handleDownload(r, e)}
                          >
                            <Download className="size-3.5" /> 下载
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableCard>

          <Pagination
            page={page}
            totalPages={totalPages}
            pageSize={PAGE_SIZE}
            onChange={setPage}
          />
        </div>
      </div>
    </PageShell>
  )
}
