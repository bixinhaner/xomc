import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Eye, RefreshCcw, Search, X } from 'lucide-react'

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
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useNEList } from '@core/hooks/api/useDevices'
import type { NE } from '@core/types/device'
import type { AlarmSeverity } from '@core/types/common'

// ============================================================
// 网元管理 — NE 列表 + 关键字筛选 + 类型/连接状态过滤
// 对照 v1 webcode/src/pages/device/NEManagement 的业务深度
// ============================================================

type NeTypeFilter = 'all' | 'eNB' | 'gNB' | 'CPE' | 'eGW'
type ConnFilter = 'all' | 'online' | 'offline'

const ALARM_VARIANT: Record<
  AlarmSeverity | 'none',
  'destructive' | 'warning' | 'default' | 'muted'
> = {
  critical: 'destructive',
  major: 'destructive',
  minor: 'warning',
  warning: 'warning',
  none: 'muted',
}

const ALARM_LABEL: Record<AlarmSeverity | 'none', string> = {
  critical: '紧急',
  major: '重要',
  minor: '次要',
  warning: '警告',
  none: '无',
}

function ConnStatusBadge({ online }: { online: boolean }) {
  return (
    <Badge variant={online ? 'success' : 'muted'}>
      <span
        className={cn(
          'mr-1 inline-block size-1.5 rounded-full',
          online ? 'bg-emerald-500' : 'bg-muted-foreground/40'
        )}
      />
      {online ? '在线' : '离线'}
    </Badge>
  )
}

export default function NEManagement() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const [neType, setNeType] = useState<NeTypeFilter>('all')
  const [conn, setConn] = useState<ConnFilter>('all')

  const queryParams = useMemo(
    () => ({
      page,
      pageSize,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
    }),
    [page, pageSize, keyword]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useNEList(queryParams)

  // 后端 NE 列表暂不支持 neType / connStatus 服务端过滤，这两项在当前页客户端兜底。
  const allRows = data?.items ?? []
  const rows = useMemo(
    () =>
      allRows.filter((ne) => {
        const matchType = neType === 'all' || ne.neType === neType
        const matchConn = conn === 'all' || ne.connStatus === conn
        return matchType && matchConn
      }),
    [allRows, neType, conn]
  )
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const hasActiveFilter =
    Boolean(keyword.trim()) || neType !== 'all' || conn !== 'all'

  function resetFilters() {
    setKeyword('')
    setNeType('all')
    setConn('all')
    setPage(1)
  }

  const colCount = 8

  return (
    <PageShell
      title="网元管理"
      description={`共 ${total} 个网元`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-64 pl-9"
              placeholder="搜索 网元名称 / SN"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>

          <Select
            value={neType}
            onValueChange={(v) => {
              setNeType(v as NeTypeFilter)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="网元类型" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部类型</SelectItem>
              <SelectItem value="eNB">eNB</SelectItem>
              <SelectItem value="gNB">gNB</SelectItem>
              <SelectItem value="CPE">CPE</SelectItem>
              <SelectItem value="eGW">eGW</SelectItem>
            </SelectContent>
          </Select>

          <Select
            value={conn}
            onValueChange={(v) => {
              setConn(v as ConnFilter)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="连接状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              <SelectItem value="online">在线</SelectItem>
              <SelectItem value="offline">离线</SelectItem>
            </SelectContent>
          </Select>

          {hasActiveFilter && (
            <Button variant="ghost" size="sm" onClick={resetFilters}>
              <X className="size-4" /> 重置
            </Button>
          )}

          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
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
              <TableHead>网元名称</TableHead>
              <TableHead>SN</TableHead>
              <TableHead>类型</TableHead>
              <TableHead>厂商</TableHead>
              <TableHead>区域</TableHead>
              <TableHead>连接状态</TableHead>
              <TableHead>告警</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={colCount} />
            ) : isError ? (
              <ErrorRow colSpan={colCount} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={colCount}>
                {hasActiveFilter ? '没有匹配的网元' : '暂无网元'}
              </EmptyRow>
            ) : (
              rows.map((ne: NE) => (
                <TableRow key={ne.id}>
                  <TableCell className="text-sm">{ne.neName || '—'}</TableCell>
                  <TableCell>
                    <span className="font-mono text-xs">{ne.sn || '—'}</span>
                  </TableCell>
                  <TableCell>
                    <Badge variant="secondary">{ne.neType || '—'}</Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {ne.vendor || '—'}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {ne.region || '—'}
                  </TableCell>
                  <TableCell>
                    <ConnStatusBadge online={ne.connStatus === 'online'} />
                  </TableCell>
                  <TableCell>
                    <Badge variant={ALARM_VARIANT[ne.alarmLevel]}>
                      {ALARM_LABEL[ne.alarmLevel]}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={!ne.sn}
                      onClick={() => navigate(`/device/detail/${ne.sn}`)}
                    >
                      <Eye className="size-4" /> 详情
                    </Button>
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
        pageSize={pageSize}
        onChange={setPage}
      />
    </PageShell>
  )
}
