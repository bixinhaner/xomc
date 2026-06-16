import { useEffect, useMemo, useState } from 'react'
import { Loader2, Plus, Save, Trash2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { PageShell } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useKPILayout,
  useKPIDefinitions,
  useSaveKPILayout,
} from '@core/hooks/api/useDashboard'
import type { KPILayoutPanel } from '@core/types/dashboard'
import {
  validateKpiPanels,
  formatKpiPanelViolations,
} from '@core/utils/kpiPanelValidation'

// ============================================================
// 系统管理 / 首页 KPI 配置（管理员）— 对齐 v1 webcode/src/pages/system/KpiConfig
// 真实数据 useKPILayout（读制式布局）+ useKPIDefinitions（指标库）+
//   useSaveKPILayout（整套写回，最后写入生效）。
// v1 用 react-grid-layout 拖拽画布；v2 无该依赖，改为顺序卡片编辑器（增/删图、
// 改标题、勾指标），保留每张图原有 x/y/w/h 坐标透传，新图给默认坐标。详见返回 notes。
// ============================================================

const TECHS: { key: string; label: string }[] = [
  { key: 'lte', label: 'LTE (4G)' },
  { key: 'nr', label: 'NR (5G)' },
  { key: 'gsm', label: 'GSM (2G)' },
]

let panelSeq = 0
type WorkingPanel = KPILayoutPanel & { _id: string }

function defaultPanel(index: number): WorkingPanel {
  return {
    _id: `new-${panelSeq++}`,
    title: '新图表',
    metrics: [],
    x: (index % 2) * 6,
    y: Math.floor(index / 2) * 8,
    w: 6,
    h: 8,
    chartType: 'line',
  }
}

function TechEditor({ tech }: { tech: string }) {
  const layoutQuery = useKPILayout(tech)
  const defsQuery = useKPIDefinitions()
  const saveMutation = useSaveKPILayout()

  const [panels, setPanels] = useState<WorkingPanel[]>([])
  const [initializedTech, setInitializedTech] = useState<string | null>(null)
  // 保存前校验不通过时的提示文案（拦截 PUT，内联告警展示）。
  const [validationError, setValidationError] = useState<string | null>(null)

  // 读到布局后一次性初始化本地工作态（按 tech 重置）。
  useEffect(() => {
    if (layoutQuery.isLoading || initializedTech === tech) return
    const loaded = layoutQuery.data?.panels ?? []
    setPanels(loaded.map((p) => ({ ...p, _id: `srv-${panelSeq++}` })))
    setInitializedTech(tech)
  }, [tech, layoutQuery.data, layoutQuery.isLoading, initializedTech])

  // 当前制式可选指标（available=false 置灰不可选）。
  const metricOptions = useMemo(() => {
    const techDef = defsQuery.data?.technologies.find((t) => t.tech === tech)
    return techDef?.items ?? []
  }, [defsQuery.data, tech])

  function updateTitle(id: string, title: string) {
    setPanels((prev) => prev.map((p) => (p._id === id ? { ...p, title } : p)))
  }

  function toggleMetric(id: string, key: string) {
    setPanels((prev) =>
      prev.map((p) => {
        if (p._id !== id) return p
        const has = p.metrics.includes(key)
        return {
          ...p,
          metrics: has
            ? p.metrics.filter((m) => m !== key)
            : [...p.metrics, key],
        }
      })
    )
  }

  function addPanel() {
    setPanels((prev) => [...prev, defaultPanel(prev.length)])
  }

  function removePanel(id: string) {
    setPanels((prev) => prev.filter((p) => p._id !== id))
  }

  function handleSave() {
    const payload: KPILayoutPanel[] = panels.map(({ _id, ...rest }) => {
      void _id
      return rest
    })
    // 保存前校验：每张图必须「标题非空 + 至少 1 指标」，否则拦截不发 PUT。
    const result = validateKpiPanels(
      payload.map((p) => ({ title: p.title, metrics: p.metrics })),
    )
    if (!result.valid) {
      setValidationError(formatKpiPanelViolations(result.violations))
      return
    }
    setValidationError(null)
    saveMutation.mutate({ tech, panels: payload })
  }

  if (layoutQuery.isLoading) {
    return (
      <div className="flex h-40 items-center justify-center text-muted-foreground">
        <Loader2 className="size-5 animate-spin" />
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        <Button variant="outline" size="sm" onClick={addPanel}>
          <Plus className="size-4" /> 新增图表
        </Button>
        <Button
          size="sm"
          disabled={saveMutation.isPending}
          onClick={handleSave}
        >
          {saveMutation.isPending ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            <Save className="size-4" />
          )}
          保存布局
        </Button>
        <span className="text-xs text-muted-foreground">
          保存后对所有用户生效（最后写入生效）
        </span>
      </div>

      {validationError ? (
        <div className="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-2 text-sm text-destructive">
          {validationError}
        </div>
      ) : null}

      {saveMutation.isSuccess ? (
        <div className="rounded-md border border-emerald-500/30 bg-emerald-500/5 px-4 py-2 text-sm text-emerald-600 dark:text-emerald-400">
          布局已保存
        </div>
      ) : null}
      {saveMutation.isError ? (
        <div className="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-2 text-sm text-destructive">
          保存失败：
          {saveMutation.error instanceof Error
            ? saveMutation.error.message
            : '未知错误（需管理员权限）'}
        </div>
      ) : null}

      {panels.length === 0 ? (
        <div className="rounded-lg border bg-card px-4 py-8 text-center text-sm text-muted-foreground">
          暂无图表，点击「新增图表」开始配置
        </div>
      ) : (
        <div className="grid gap-3 md:grid-cols-2">
          {panels.map((panel) => (
            <div
              key={panel._id}
              className="space-y-3 rounded-lg border bg-card p-4"
            >
              <div className="flex items-center gap-2">
                <div className="flex-1 space-y-1">
                  <Label className="text-xs">图表标题</Label>
                  <Input
                    value={panel.title}
                    onChange={(e) => updateTitle(panel._id, e.target.value)}
                  />
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  className="mt-5 text-destructive"
                  onClick={() => removePanel(panel._id)}
                >
                  <Trash2 className="size-4" />
                </Button>
              </div>

              <div>
                <Label className="text-xs">
                  指标{' '}
                  <span className="text-muted-foreground">
                    已选 {panel.metrics.length}
                  </span>
                </Label>
                <div className="mt-1.5 max-h-44 space-y-1 overflow-auto rounded-md border p-2">
                  {metricOptions.length === 0 ? (
                    <div className="py-2 text-center text-xs text-muted-foreground">
                      {defsQuery.isLoading ? '加载指标库…' : '该制式暂无指标定义'}
                    </div>
                  ) : (
                    metricOptions.map((m) => {
                      const checked = panel.metrics.includes(m.key)
                      const disabled = !m.available && !checked
                      return (
                        <label
                          key={m.key}
                          className={cn(
                            'flex cursor-pointer items-center gap-2 rounded px-1.5 py-1 text-xs hover:bg-muted',
                            disabled && 'cursor-not-allowed opacity-50'
                          )}
                        >
                          <input
                            type="checkbox"
                            className="size-3.5 accent-primary"
                            checked={checked}
                            disabled={disabled}
                            onChange={() => toggleMetric(panel._id, m.key)}
                          />
                          <span className="flex-1 truncate">
                            {m.cn_name || m.key}
                          </span>
                          {m.unit ? (
                            <Badge variant="muted">{m.unit}</Badge>
                          ) : null}
                          {!m.available ? (
                            <Badge variant="warning">缺指标</Badge>
                          ) : null}
                        </label>
                      )
                    })
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export default function KpiConfig() {
  const [tech, setTech] = useState<string>('lte')

  return (
    <PageShell
      title="首页 KPI 配置"
      description="按制式配置首页 KPI 图表与指标（管理员）"
      toolbar={
        <div className="flex flex-wrap items-center gap-1.5">
          {TECHS.map((t) => (
            <Button
              key={t.key}
              size="sm"
              variant={tech === t.key ? 'default' : 'outline'}
              onClick={() => setTech(t.key)}
            >
              {t.label}
            </Button>
          ))}
        </div>
      }
    >
      <TechEditor key={tech} tech={tech} />
    </PageShell>
  )
}
