import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Search } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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

import { useIndicatorList, useEnabledIndicators } from '@core/hooks/api/useIndicatorsLibrary'
import {
  techToDeviceType,
  type DeviceType,
  type IndicatorInfo,
  type TechLower,
} from '@core/types/indicatorLibrary'

// ============================================================
// KPI 指标详情 — 对照 v1 kpi-library IndicatorsByTech。
// 路由 /product/kpi-library/:tech/:platform,服务端分页拉取该平台指标。
// 启用状态走 default 桶(与 v1 一致)。
// ============================================================

const TECH_LABEL: Record<TechLower, string> = {
  enb: 'ENB (LTE)',
  gsm: 'GSM',
  gnb: 'GNB (5G NR)',
}
const VALID_TECHS: TechLower[] = ['enb', 'gsm', 'gnb']
const OPERATOR_CODE = 'default'
const PAGE_SIZE = 50

function parseTech(raw?: string): TechLower | undefined {
  if (!raw) return undefined
  const lower = raw.toLowerCase()
  return VALID_TECHS.includes(lower as TechLower) ? (lower as TechLower) : undefined
}

export function KpiIndicatorsPage() {
  const navigate = useNavigate()
  const params = useParams<{ tech: string; platform: string }>()
  const tech = parseTech(params.tech)
  const platform = params.platform ? decodeURIComponent(params.platform) : undefined
  const deviceType: DeviceType = tech ? techToDeviceType(tech) : 'ENB'

  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)

  const { data, isLoading, isError, error, isFetching } = useIndicatorList(deviceType, {
    platformName: platform,
    keyword: keyword.trim() || undefined,
    page,
    pageSize: PAGE_SIZE,
  })
  const enabledQuery = useEnabledIndicators(deviceType, OPERATOR_CODE)
  const enabledSet = useMemo(
    () => new Set(enabledQuery.data?.items ?? []),
    [enabledQuery.data]
  )

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const isGnb = deviceType === 'GNB'
  const colCount = isGnb ? 5 : 6

  return (
    <PageShell
      title={`KPI 指标 · ${platform ?? ''}`}
      description={`${tech ? TECH_LABEL[tech] : '—'} · 共 ${total} 个指标`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => navigate('/product/kpi-library')}>
            <ArrowLeft className="size-4" /> 返回
          </Button>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-80 pl-9"
              placeholder="搜索 ID / 名称"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
        </div>
      }
    >
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-40">ID</TableHead>
              <TableHead>中文名</TableHead>
              <TableHead>英文名</TableHead>
              <TableHead className="w-40">分组</TableHead>
              {!isGnb && <TableHead className="w-24">级别</TableHead>}
              <TableHead className="w-24">启用</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={colCount} />
            ) : isError ? (
              <ErrorRow colSpan={colCount} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={colCount}>
                {keyword.trim() ? '没有匹配的指标' : '该平台暂无指标'}
              </EmptyRow>
            ) : (
              rows.map((row) => (
                <IndicatorRow
                  key={row.id}
                  row={row}
                  isGnb={isGnb}
                  enabled={enabledSet.has(row.id)}
                />
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination page={page} totalPages={totalPages} pageSize={PAGE_SIZE} onChange={setPage} />
    </PageShell>
  )
}

function IndicatorRow({
  row,
  isGnb,
  enabled,
}: {
  row: IndicatorInfo
  isGnb: boolean
  enabled: boolean
}) {
  return (
    <TableRow>
      <TableCell className="font-mono text-xs">{row.id}</TableCell>
      <TableCell className="max-w-[220px] truncate text-sm" title={row.cnName}>
        {row.cnName || '—'}
      </TableCell>
      <TableCell className="max-w-[260px] truncate text-sm text-muted-foreground" title={row.enName}>
        {row.enName || '—'}
      </TableCell>
      <TableCell>
        {row.groupName || row.groupId ? (
          <Badge variant="outline">{row.groupName || row.groupId}</Badge>
        ) : (
          <span className="text-muted-foreground">—</span>
        )}
      </TableCell>
      {!isGnb && (
        <TableCell className="text-xs text-muted-foreground">{row.indicatorLevel || '—'}</TableCell>
      )}
      <TableCell>
        {enabled ? (
          <Badge variant="success">已启用</Badge>
        ) : (
          <Badge variant="muted">未启用</Badge>
        )}
      </TableCell>
    </TableRow>
  )
}

export default KpiIndicatorsPage
