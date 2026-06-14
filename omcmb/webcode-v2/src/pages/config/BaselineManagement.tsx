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

import { useBaselineConfigs } from '@core/hooks/api/useConfig'

// ============================================================
// 配置基线管理 — 对照 v1 config/BaselineManagement。
// 真实数据：useBaselineConfigs 列表 + 设备类型/状态筛选。
// 行 → 链接到基线详情 /config/baseline/:id（参数对比清单）。
// ============================================================

const BASELINE_STATUS: Record<
  string,
  { variant: 'success' | 'warning' | 'muted'; label: string }
> = {
  active: { variant: 'success', label: '生效' },
  draft: { variant: 'warning', label: '草稿' },
  deprecated: { variant: 'muted', label: '废弃' },
}

export default function BaselineManagement() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [deviceType, setDeviceType] = useState('')
  const [status, setStatus] = useState('')

  const query = useBaselineConfigs(
    useMemo(
      () => ({
        page,
        pageSize,
        ...(deviceType ? { deviceType } : {}),
        ...(status ? { status } : {}),
      }),
      [page, deviceType, status]
    )
  )
  const { data, isLoading, isError, error, isFetching, refetch } = query
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['基线名称', '设备类型', '版本', '参数数量', '状态', '创建人', '更新时间']

  return (
    <PageShell
      title="配置基线管理"
      description="按设备类型管理配置基线 — 点击进入参数对比详情"
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Select
            value={deviceType || 'all'}
            onValueChange={(v) => {
              setDeviceType(v === 'all' ? '' : v)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-40">
              <SelectValue placeholder="设备类型" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部类型</SelectItem>
              <SelectItem value="eNB">eNB (4G)</SelectItem>
              <SelectItem value="gNB">gNB (5G)</SelectItem>
              <SelectItem value="CPE">CPE</SelectItem>
            </SelectContent>
          </Select>
          <Select
            value={status || 'all'}
            onValueChange={(v) => {
              setStatus(v === 'all' ? '' : v)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder="状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              <SelectItem value="active">生效</SelectItem>
              <SelectItem value="draft">草稿</SelectItem>
              <SelectItem value="deprecated">废弃</SelectItem>
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
              <EmptyRow colSpan={cols.length}>暂无配置基线</EmptyRow>
            ) : (
              rows.map((b) => {
                const s = BASELINE_STATUS[b.status] ?? BASELINE_STATUS.draft
                return (
                  <TableRow key={b.id}>
                    <TableCell>
                      <button
                        type="button"
                        className="text-left font-medium text-primary hover:underline"
                        onClick={() => navigate(`/config/baseline/${b.id}`)}
                      >
                        {b.baselineName}
                      </button>
                      <div className="line-clamp-1 text-xs text-muted-foreground">
                        {b.description || '—'}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{b.deviceType || '—'}</Badge>
                    </TableCell>
                    <TableCell className="font-mono text-xs">{b.version || '—'}</TableCell>
                    <TableCell className="tabular-nums">{b.params?.length ?? 0}</TableCell>
                    <TableCell>
                      <Badge variant={s.variant}>{s.label}</Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">{b.creator || '—'}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(b.updateTime)}
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
