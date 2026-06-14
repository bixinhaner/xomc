import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Loader2, MapPin, RefreshCcw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { PageShell } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useSiteById, useDomains } from '@core/hooks/api/useTopology'
import type { Site, SiteStatus } from '@core/types/topology'

// ============================================================
// 站点详情（v2 皮肤）— 带 :id 参数，对照 v1 SiteManagement 的「查看/定位」。
//  v1 无独立详情路由（弹 message 提示），v2 抽出真实详情页：
//   - useSiteById(id)：按 id 拉单站点（真实数据，非恒定未找到）
//   - useDomains    ：解析域名展示
//  从站点列表 / 域管理站点行点进，可返回列表。
// ============================================================

const STATUS_META: Record<SiteStatus, { label: string; variant: 'success' | 'warning' | 'muted' }> = {
  active: { label: '正常', variant: 'success' },
  maintenance: { label: '维护', variant: 'warning' },
  inactive: { label: '停用', variant: 'muted' },
}

type FieldVal = string | number | null | undefined

function InfoRow({ label, value, mono }: { label: string; value: FieldVal; mono?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-4 border-b py-2.5 text-sm">
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span className={cn('truncate text-right', mono && 'font-mono text-xs')}>
        {value === '' || value == null ? '—' : value}
      </span>
    </div>
  )
}

export default function SiteDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data: site, isLoading, isError, error, isFetching, refetch } = useSiteById(id)
  const { data: domainsData } = useDomains()
  const domainName =
    site && domainsData
      ? domainsData.find((d) => d.id === site.domainId)?.name ?? site.domainId
      : undefined

  const fields = (s: Site): { label: string; value: FieldVal; mono?: boolean }[] => [
    { label: '站点名称', value: s.name },
    { label: '站点 ID', value: s.id, mono: true },
    { label: '所属域', value: domainName ?? s.domainId },
    { label: '详细地址', value: s.address },
    { label: '设备数', value: s.deviceCount },
    { label: '经度', value: s.longitude != null ? s.longitude.toFixed(6) : '', mono: true },
    { label: '纬度', value: s.latitude != null ? s.latitude.toFixed(6) : '', mono: true },
  ]

  const meta = site ? STATUS_META[site.status] ?? STATUS_META.inactive : null

  return (
    <PageShell
      title={site ? site.name : '站点详情'}
      description={id ? `站点 ID ${id}` : undefined}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => navigate('/topology/site')}>
            <ArrowLeft className="size-4" /> 返回
          </Button>
          {meta && <Badge variant={meta.variant}>{meta.label}</Badge>}
          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCcw className="size-4" /> 刷新
            </Button>
          </div>
        </div>
      }
    >
      {isLoading ? (
        <div className="flex h-64 items-center justify-center">
          <Loader2 className="size-6 animate-spin text-muted-foreground" />
        </div>
      ) : isError ? (
        <Card className="flex h-48 items-center justify-center p-6 text-sm text-destructive">
          加载失败：{error instanceof Error ? error.message : '未知错误'}
        </Card>
      ) : !site ? (
        <Card className="flex h-48 flex-col items-center justify-center gap-2 p-6 text-muted-foreground">
          <span className="text-sm">未找到站点 {id}</span>
          <Button variant="outline" size="sm" onClick={() => navigate('/topology/site')}>
            返回站点列表
          </Button>
        </Card>
      ) : (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader className="p-4 pb-2">
              <CardTitle className="text-base font-medium">基本信息</CardTitle>
            </CardHeader>
            <CardContent className="p-4 pt-2">
              <div className="grid grid-cols-1 gap-x-8 sm:grid-cols-2">
                {fields(site).map((f) => (
                  <InfoRow key={f.label} label={f.label} value={f.value} mono={f.mono} />
                ))}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="p-4 pb-2">
              <CardTitle className="text-base font-medium">位置</CardTitle>
            </CardHeader>
            <CardContent className="p-4 pt-2">
              <div className="flex flex-col items-center justify-center gap-2 rounded-lg border bg-slate-50 py-10 text-center dark:bg-slate-900">
                <MapPin className="size-6 text-primary" />
                <div className="font-mono text-sm">
                  {site.latitude != null && site.longitude != null
                    ? `${site.latitude.toFixed(4)}, ${site.longitude.toFixed(4)}`
                    : '无坐标信息'}
                </div>
                <div className="text-xs text-muted-foreground">{site.address || '—'}</div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </PageShell>
  )
}
