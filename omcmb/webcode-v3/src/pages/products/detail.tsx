import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  ArrowLeft,
  Loader2,
  Boxes,
  ShieldCheck,
  Regex,
  Layers,
  Cpu,
  AlertTriangle,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useProductDetail } from '@core/hooks/api/useProducts'
import { useParamMappings } from '@core/hooks/api/useParamModels'
import type { ProductPattern } from '@core/types/product'

/**
 * 产品装配件详情（带 :id 参数路由）。
 * 真实数据：useProductDetail(id) → { product, patterns }；并联动加载该产品绑定的
 * 参数模型 mapping 计数（useParamMappings(paramModelName)）。
 */
export default function ProductDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data, isLoading, isError, error, isFetching } = useProductDetail(id)
  const product = data?.product
  const patterns = useMemo<ProductPattern[]>(() => data?.patterns ?? [], [data])

  // 绑定的参数模型映射条目（详情深度，真实数据）。
  const { data: mappingData, isFetching: mappingFetching } = useParamMappings(
    product?.paramModelName || undefined,
  )
  const mappingCount = mappingData?.total ?? mappingData?.items?.length ?? 0

  const sortedPatterns = useMemo(
    () => [...patterns].sort((a, b) => a.sortOrder - b.sortOrder),
    [patterns],
  )

  const back = (
    <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/product/products')}>
      BACK
    </NeonButton>
  )

  if (isLoading) {
    return (
      <PageShell code="F02" title="PRODUCT · 装配件详情" subtitle="LOADING" toolbar={back}>
        <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
          <Loader2 className="size-4 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
        </div>
      </PageShell>
    )
  }

  if (isError || !product) {
    return (
      <PageShell code="F02" title="PRODUCT · 装配件详情" subtitle="ERROR" toolbar={back}>
        <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
          {isError ? `FAILURE · ${error instanceof Error ? error.message : '未知错误'}` : 'NOT FOUND · 装配件不存在'}
        </div>
      </PageShell>
    )
  }

  const overrideKeys = Object.entries(product.deviceAttrsOverride ?? {})
    .filter(([, v]) => v === true)
    .map(([k]) => k)

  return (
    <PageShell
      code="F02"
      title={`PRODUCT · ${product.name}`}
      subtitle={`${(product.tech || '—').toUpperCase()} · ${product.vendor || 'UNKNOWN VENDOR'}`}
      isFetching={isFetching || mappingFetching}
      toolbar={back}
    >
      {/* KPI 摘要带 */}
      <div className="mb-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Stat label="DEVICES" color="#5b9eff" value={product.deviceCount ?? 0} icon={<Cpu className="size-3.5" />} />
        <Stat label="PATTERNS" color="#00f0ff" value={patterns.length} icon={<Regex className="size-3.5" />} />
        <Stat label="MODEL MAPPINGS" color="#00ff88" value={mappingCount} icon={<Layers className="size-3.5" />} />
        <Stat
          label="SOURCE"
          color={product.isBuiltin ? '#a855f7' : '#00ff88'}
          textValue={product.isBuiltin ? 'BUILTIN' : 'CUSTOM'}
          icon={product.isBuiltin ? <ShieldCheck className="size-3.5" /> : <Boxes className="size-3.5" />}
        />
      </div>

      <div className="grid grid-cols-12 gap-4">
        {/* 基本属性 */}
        <GlassPanel title="ASSEMBLY SPEC · 装配规格" meta="REGISTRY" className="col-span-12 lg:col-span-5">
          <dl className="divide-y divide-cyan-500/8">
            <Field label="产品名称" value={product.name} />
            <Field label="厂商 VENDOR" value={product.vendor} />
            <Field label="制式 TECH" value={(product.tech || '—').toUpperCase()} />
            <Field label="工作模式" value={product.radioModes} />
            <Field label="参数模型库" value={product.paramModelName} />
            <Field label="指标平台" value={product.indicatorPlatform} />
            <Field label="告警网元类型" value={product.alarmNeType} />
            <Field label="指标设备类型" value={product.indicatorDeviceType} />
            <Field
              label="未知告警"
              valueNode={
                <StatusBadge
                  status={product.enableUnknownAlarm ? 'warning' : 'offline'}
                  label={product.enableUnknownAlarm ? '接受' : '丢弃'}
                />
              }
            />
            <Field
              label="Filetype11"
              valueNode={
                <StatusBadge
                  status={product.enableFiletype11 ? 'ok' : 'offline'}
                  label={product.enableFiletype11 ? '启用' : '停用'}
                />
              }
            />
            {product.description && <Field label="描述" value={product.description} />}
          </dl>
        </GlassPanel>

        {/* 匹配正则 */}
        <GlassPanel
          title="MATCH PATTERNS · 匹配正则"
          meta={`${patterns.length} RULES`}
          className="col-span-12 lg:col-span-7"
        >
          {sortedPatterns.length === 0 ? (
            <div className="px-4 py-8 text-center font-mono text-[11px] uppercase tracking-[0.2em] text-cyan-300/45">
              NO PATTERNS · 无匹配正则
            </div>
          ) : (
            <div className="divide-y divide-cyan-500/8">
              {sortedPatterns.map((p) => (
                <div key={p.id} className="flex items-center gap-3 px-3.5 py-2.5">
                  <span className="w-8 shrink-0 text-center font-mono text-[11px] text-cyan-300/50">
                    #{p.sortOrder}
                  </span>
                  <code className="min-w-0 flex-1 truncate font-mono text-xs text-cyan-100/90">
                    {p.productClass}
                  </code>
                  <StatusBadge
                    status={p.source === 'builtin' ? 'active' : 'ok'}
                    label={p.source === 'builtin' ? 'BUILTIN' : 'CUSTOM'}
                    className="scale-90"
                  />
                  <StatusBadge
                    status={p.isActive ? 'online' : 'offline'}
                    label={p.isActive ? '生效' : '停用'}
                    className="scale-90"
                  />
                </div>
              ))}
            </div>
          )}
        </GlassPanel>

        {/* 设备属性覆盖 */}
        <GlassPanel
          title="ATTR OVERRIDES · 设备属性覆盖"
          meta={`${overrideKeys.length} FIELDS`}
          className="col-span-12"
        >
          {overrideKeys.length === 0 ? (
            <div className="flex items-center gap-2 px-4 py-6 font-mono text-[11px] uppercase tracking-[0.18em] text-cyan-300/45">
              <AlertTriangle className="size-3.5" />
              NO OVERRIDES · 全部沿用标准参数树
            </div>
          ) : (
            <div className="flex flex-wrap gap-2 p-3.5">
              {overrideKeys.map((k) => (
                <span key={k} className="chip text-cyan-200">
                  {k}
                </span>
              ))}
            </div>
          )}
        </GlassPanel>
      </div>
    </PageShell>
  )
}

function Stat({
  label,
  color,
  value,
  textValue,
  icon,
}: {
  label: string
  color: string
  value?: number
  textValue?: string
  icon?: React.ReactNode
}) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2.5"
      style={{ borderLeftColor: color }}
    >
      <div className="flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/65">
        {icon}
        {label}
      </div>
      <div
        className="font-display text-2xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {textValue ?? value ?? 0}
      </div>
    </div>
  )
}

function Field({
  label,
  value,
  valueNode,
}: {
  label: string
  value?: string
  valueNode?: React.ReactNode
}) {
  return (
    <div className="flex items-center justify-between gap-4 px-3.5 py-2">
      <dt className="shrink-0 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/55">
        {label}
      </dt>
      <dd className="min-w-0 truncate text-right text-xs text-cyan-100/90">
        {valueNode ?? (value && value.trim() ? value : <span className="text-cyan-300/40">—</span>)}
      </dd>
    </div>
  )
}
