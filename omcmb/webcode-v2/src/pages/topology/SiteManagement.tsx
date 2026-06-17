import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { MapPin, RefreshCcw, Search } from 'lucide-react'

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

import { useSites, useDomains } from '@core/hooks/api/useTopology'
import type { SiteStatus } from '@core/types/topology'

// ============================================================
// 站点管理（v2 皮肤）— 对照 v1 webcode/pages/topology/SiteManagement
//  v1 是站点列表 + 名称/区域/状态筛选（后端仅 GET /sites，无 PUT/DELETE）。
//  v2 用真实 useSites（服务端 keyword/domainId/status 筛选）+ useDomains 解析
//  域名；站点名/行可点进站点详情（/topology/site/:id）。客户端分页。
// ============================================================

const STATUS_META: Record<SiteStatus, { label: string; variant: 'success' | 'warning' | 'muted' }> = {
  active: { label: '正常', variant: 'success' },
  maintenance: { label: '维护', variant: 'warning' },
  inactive: { label: '停用', variant: 'muted' },
}

const PAGE_SIZE = 20

type StatusFilter = '' | SiteStatus

export default function SiteManagement() {
  const navigate = useNavigate()

  const [keyword, setKeyword] = useState('')
  const [domainId, setDomainId] = useState('')
  const [status, setStatus] = useState<StatusFilter>('')

  const { data: domainsData } = useDomains()
  const domains = domainsData ?? []
  const domainNameMap = useMemo(() => {
    const m: Record<string, string> = {}
    for (const d of domains) m[d.id] = d.name
    return m
  }, [domains])

  const {
    data: sitesData,
    isLoading,
    isError,
    error,
    isFetching,
    refetch,
  } = useSites({
    ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
    ...(domainId ? { domainId } : {}),
    ...(status ? { status } : {}),
  })

  const sites = sitesData?.items ?? []

  // 客户端分页（筛选键变化归 1）
  const listKey = `${keyword}-${domainId}-${status}`
  const [paging, setPaging] = useState({ key: listKey, page: 1 })
  const page = paging.key === listKey ? paging.page : 1
  if (paging.key !== listKey) setPaging({ key: listKey, page: 1 })

  const totalPages = Math.max(1, Math.ceil(sites.length / PAGE_SIZE))
  const pagedSites = useMemo(
    () => sites.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE),
    [sites, page],
  )

  const cols = ['站点名称', '所属域', '地址', '设备数', '状态', '坐标']

  return (
    <PageShell
      title="站点管理"
      description={`共 ${sitesData?.total ?? sites.length} 个站点`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-64 pl-9"
              placeholder="搜索站点名称 / 地址"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>

          <Select
            value={domainId || 'all'}
            onValueChange={(v) => setDomainId(v === 'all' ? '' : v)}
          >
            <SelectTrigger className="w-44">
              <SelectValue placeholder="所属域" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部域</SelectItem>
              {domains.map((d) => (
                <SelectItem key={d.id} value={d.id}>
                  {d.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select
            value={status || 'all'}
            onValueChange={(v) => setStatus(v === 'all' ? '' : (v as SiteStatus))}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder="站点状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              <SelectItem value="active">正常</SelectItem>
              <SelectItem value="maintenance">维护</SelectItem>
              <SelectItem value="inactive">停用</SelectItem>
            </SelectContent>
          </Select>

          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCcw className="mr-1 h-4 w-4" />
              刷新
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
            ) : pagedSites.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无站点</EmptyRow>
            ) : (
              pagedSites.map((s) => {
                const meta = STATUS_META[s.status] ?? STATUS_META.inactive
                return (
                  <TableRow
                    key={s.id}
                    className="cursor-pointer"
                    onClick={() => navigate(`/topology/site`)}
                  >
                    <TableCell className="font-medium text-primary hover:underline">
                      {s.name}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {domainNameMap[s.domainId] ?? s.domainId ?? '—'}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">{s.address}</TableCell>
                    <TableCell className="tabular-nums">{s.deviceCount}</TableCell>
                    <TableCell>
                      <Badge variant={meta.variant}>{meta.label}</Badge>
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      <span className="inline-flex items-center gap-1">
                        <MapPin className="h-3 w-3" />
                        {s.latitude != null && s.longitude != null
                          ? `${s.latitude.toFixed(3)}, ${s.longitude.toFixed(3)}`
                          : '—'}
                      </span>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      {totalPages > 1 && (
        <Pagination
          page={page}
          totalPages={totalPages}
          pageSize={PAGE_SIZE}
          onChange={(p) => setPaging({ key: listKey, page: p })}
        />
      )}
    </PageShell>
  )
}
