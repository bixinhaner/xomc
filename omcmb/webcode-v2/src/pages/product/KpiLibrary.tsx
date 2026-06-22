import { useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Layers, RefreshCcw, Search, X } from 'lucide-react'

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
  TableCard,
} from '@/components/layout/PageShell'

import { useIndicatorSummary } from '@core/hooks/api/useIndicatorsLibrary'
import type { DeviceType } from '@core/types/indicatorLibrary'
import type { IndicatorPlatformSummary, TechLower } from '@core/types/indicatorLibrary'

import { useT } from '@/hooks/useT'

import { KpiGroupsDialog } from './KpiGroupsDialog'
import { KpiIndicatorsDetail } from './KpiIndicatorsDetail'

// ============================================================
// KPI 指标库 — 对照 v1 webcode/src/pages/product/kpi-library SummaryTab。
// 一级列表以 (制式, 平台) 为唯一行;点平台进该平台指标详情。
// ============================================================

const TECH_LABEL: Record<TechLower, string> = {
  enb: 'ENB (LTE)',
  gsm: 'GSM',
  gnb: 'GNB (5G NR)',
}

// TechLower(平台行制式) → DeviceType(分组维护用)。
const TECH_TO_DEVICE: Record<TechLower, DeviceType> = {
  enb: 'ENB',
  gsm: 'GSM',
  gnb: 'GNB',
}

function basename(path: string): string {
  if (!path) return '—'
  return path.split('/').pop() || path
}

export function KpiLibraryPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const t = useT()
  const [keyword, setKeyword] = useState('')
  const [groupsOpen, setGroupsOpen] = useState(false)

  const { data, isLoading, isError, error, isFetching, refetch } = useIndicatorSummary()

  // drill-down：URL ?tech=&platform= 即进二级指标详情视图（可分享/可后退）。
  const techParam = searchParams.get('tech') as TechLower | null
  const platformParam = searchParams.get('platform')
  const detailTech: TechLower | null =
    techParam && techParam in TECH_TO_DEVICE ? techParam : null

  const openDetail = (tech: TechLower, platform: string) => {
    const next = new URLSearchParams(searchParams)
    next.set('tech', tech)
    next.set('platform', platform)
    setSearchParams(next)
  }

  const backToSummary = () => {
    const next = new URLSearchParams(searchParams)
    next.delete('tech')
    next.delete('platform')
    setSearchParams(next)
  }

  // 一级列表无选中行，按首行制式推断分组维护的默认制式（弹窗内仍可切换）。
  const defaultDeviceType: DeviceType = TECH_TO_DEVICE[data?.items?.[0]?.tech ?? 'enb'] ?? 'ENB'

  const items = useMemo(() => {
    const raw = data?.items ?? []
    const k = keyword.trim().toLowerCase()
    if (!k) return raw
    return raw.filter((row) =>
      `${row.platform} ${TECH_LABEL[row.tech] ?? row.tech}`.toLowerCase().includes(k)
    )
  }, [data, keyword])

  const colCount = 5

  // drill-down 早返回必须放在所有 hook 之后(否则详情态少调 useMemo → "Rendered fewer hooks")。
  if (detailTech && platformParam) {
    return (
      <KpiIndicatorsDetail
        deviceType={TECH_TO_DEVICE[detailTech]}
        platform={platformParam}
        onBack={backToSummary}
      />
    )
  }

  return (
    <PageShell
      title="KPI 指标库"
      description={`共 ${items.length} 个平台`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-80 pl-9"
              placeholder="搜索平台 / 制式"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          {keyword.trim() && (
            <Button variant="ghost" size="sm" onClick={() => setKeyword('')}>
              <X className="size-4" /> 重置
            </Button>
          )}
          <div className="ml-auto flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => setGroupsOpen(true)}>
              <Layers className="size-4" /> {t('product.kpi.group.manage')}
            </Button>
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
              <TableHead className="w-32">制式</TableHead>
              <TableHead>平台</TableHead>
              <TableHead className="w-28 text-right">指标数</TableHead>
              <TableHead>XML 文件</TableHead>
              <TableHead className="w-24">来源</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={colCount} />
            ) : isError ? (
              <ErrorRow colSpan={colCount} error={error} />
            ) : items.length === 0 ? (
              <EmptyRow colSpan={colCount}>
                {keyword.trim() ? '没有匹配的平台' : '暂无平台'}
              </EmptyRow>
            ) : (
              items.map((row) => (
                <SummaryRow
                  key={`${row.tech}__${row.platform}`}
                  row={row}
                  onOpen={openDetail}
                />
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <KpiGroupsDialog
        open={groupsOpen}
        onClose={() => setGroupsOpen(false)}
        initialDeviceType={defaultDeviceType}
      />
    </PageShell>
  )
}

function SummaryRow({
  row,
  onOpen,
}: {
  row: IndicatorPlatformSummary
  onOpen: (tech: TechLower, platform: string) => void
}) {
  return (
    <TableRow>
      <TableCell>
        <Badge variant="outline">{TECH_LABEL[row.tech] ?? row.tech.toUpperCase()}</Badge>
      </TableCell>
      <TableCell>
        <button
          type="button"
          className="font-medium text-primary hover:underline"
          onClick={() => onOpen(row.tech, row.platform)}
        >
          {row.platform}
        </button>
      </TableCell>
      <TableCell className="text-right tabular-nums">{row.indicators}</TableCell>
      <TableCell className="font-mono text-xs text-muted-foreground" title={row.loadedFrom}>
        {basename(row.loadedFrom)}
      </TableCell>
      <TableCell>
        <Badge variant={row.source === 'custom' ? 'default' : 'muted'}>
          {row.source === 'custom' ? '自定义' : row.source === 'builtin' ? '内置' : '未知'}
        </Badge>
      </TableCell>
    </TableRow>
  )
}

export default KpiLibraryPage
