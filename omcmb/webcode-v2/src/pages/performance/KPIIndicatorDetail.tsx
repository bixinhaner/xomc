import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, RefreshCcw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { PageShell, formatTime } from '@/components/layout/PageShell'

import { useIndicatorInfo } from '@core/hooks/api/useIndicator'

// ============================================================
// KPI 指标详情 — 对齐 v1 /performance/kpi-standard/detail/:deviceType/:indicatorId
//   带参路由，useParams 取 deviceType + indicatorId，useIndicatorInfo 真实加载。
// ============================================================

type DeviceTypeTab = 'ENB' | 'GSM' | 'GNB'

function normalizeDeviceType(raw: string | undefined): DeviceTypeTab {
  if (raw === 'GSM' || raw === 'GNB' || raw === 'ENB') return raw
  return 'ENB'
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1 border-b py-2 last:border-b-0 sm:flex-row sm:items-center sm:gap-4">
      <div className="w-40 shrink-0 text-xs uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      <div className="text-sm">{children}</div>
    </div>
  )
}

export function KPIIndicatorDetailPage() {
  const navigate = useNavigate()
  const { deviceType: rawDeviceType, indicatorId } = useParams<{
    deviceType: string
    indicatorId: string
  }>()

  const deviceType = useMemo(() => normalizeDeviceType(rawDeviceType), [rawDeviceType])
  const id = indicatorId ?? ''
  const isGNB = deviceType === 'GNB'

  const { data: indicator, isLoading, isError, error, isFetching, refetch } = useIndicatorInfo(
    id,
    deviceType
  )

  const indicatorName = useMemo(() => {
    if (!indicator) return '—'
    return indicator.kpiNameZh || indicator.kpiName || indicator.kpiNameEn || '—'
  }, [indicator])

  const definition = useMemo(() => {
    if (!indicator) return '—'
    return indicator.definitionZh || indicator.definition || indicator.definitionEn || '—'
  }, [indicator])

  const back = () => navigate('/performance/kpi-standard')

  return (
    <PageShell
      title={isLoading ? '指标详情' : indicatorName}
      description={`设备制式 ${deviceType} · 指标编号 ${id || '—'}`}
      isFetching={isFetching}
      toolbar={
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={back}>
            <ArrowLeft className="size-4" /> 返回
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={isFetching}
            onClick={() => void refetch()}
          >
            <RefreshCcw className="size-4" /> 刷新
          </Button>
        </div>
      }
    >
      {isLoading ? (
        <Card className="p-6 text-sm text-muted-foreground">加载指标详情…</Card>
      ) : isError ? (
        <Card className="p-6 text-sm text-destructive">
          加载失败：{error instanceof Error ? error.message : '未知错误'}
        </Card>
      ) : !indicator ? (
        <Card className="p-6 text-sm text-muted-foreground">
          未找到指标（ID: {id || '—'}）
        </Card>
      ) : (
        <Card className="p-6">
          <Field label="指标编号">
            <span className="font-mono">{indicator.kpiId}</span>
          </Field>
          <Field label="指标名称">{indicatorName}</Field>
          <Field label="自定义名称">{indicator.custName || '—'}</Field>
          <Field label="功能集">{indicator.catagoryName || indicator.catagoryId || '—'}</Field>
          {!isGNB && <Field label="产品类">{indicator.productClass || '—'}</Field>}
          {!isGNB && (
            <Field label="级别">
              {indicator.indicatorLevel === 'device'
                ? 'Device'
                : indicator.indicatorLevel === 'plmn'
                  ? 'PLMN'
                  : indicator.indicatorLevel || '—'}
            </Field>
          )}
          <Field label="单位">{indicator.unit || '—'}</Field>
          <Field label="统计类型">{indicator.statisType || '—'}</Field>
          <Field label="指标类型">
            <div className="flex items-center gap-1.5">
              <Badge
                variant={
                  indicator.isCounter || indicator.indicatorType === 'counter'
                    ? 'outline'
                    : 'default'
                }
              >
                {indicator.isCounter || indicator.indicatorType === 'counter' ? 'Counter' : 'KPI'}
              </Badge>
              <Badge variant={indicator.isCustomize ? 'warning' : 'muted'}>
                {indicator.isCustomize ? '自定义' : '系统内置'}
              </Badge>
            </div>
          </Field>
          <Field label="是否计量">
            <Badge variant={indicator.isEnable ? 'success' : 'muted'}>
              {indicator.isEnable ? '是' : '否'}
            </Badge>
          </Field>
          <Field label="算法 / 公式">
            <span className="font-mono text-xs">{indicator.arithmetic || '—'}</span>
          </Field>
          <Field label="定义说明">
            <span className="text-muted-foreground">{definition}</span>
          </Field>
          <Field label="更新人">{indicator.updater || '—'}</Field>
          <Field label="更新时间">{formatTime(indicator.updateTime)}</Field>
        </Card>
      )}
    </PageShell>
  )
}

export default KPIIndicatorDetailPage
